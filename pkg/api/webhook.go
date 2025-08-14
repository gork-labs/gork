package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
)

// WebhookRequest defines the conventional request structure for webhooks.
// This is a marker interface that webhook request types must implement.
type WebhookRequest interface {
	// WebhookRequest marker method to identify webhook request types
	WebhookRequest()
}

// WebhookEvent represents a verified and parsed webhook event.
// ProviderObject is a concrete provider payload (e.g., *stripe.PaymentIntent).
// UserMetaJSON optionally contains provider-extracted metadata for user validation.
type WebhookEvent struct {
	Type           string
	ProviderObject interface{}
	UserMetaJSON   json.RawMessage
}

// WebhookProviderInfo describes the webhook provider metadata for documentation.
type WebhookProviderInfo struct {
	Name    string
	Website string
	DocsURL string
}

// WebhookHandler defines the interface for webhook providers.
// T must be a type that implements WebhookRequest.
// All providers MUST implement ProviderInfo() to expose basic metadata.
type WebhookHandler[T WebhookRequest] interface {
	// ParseRequest verifies the webhook signature and extracts event data using conventional struct
	ParseRequest(req T) (WebhookEvent, error)

	// SuccessResponse returns the appropriate success response for the provider
	SuccessResponse() interface{}

	// ErrorResponse returns the appropriate error response for the provider
	ErrorResponse(err error) interface{}

	// GetValidEventTypes returns a list of valid event types for validation and OpenAPI.
	GetValidEventTypes() []string

	// ProviderInfo exposes provider name and documentation metadata.
	ProviderInfo() WebhookProviderInfo
}

// EventHandlerFunc is a generic interface for event handlers.
// Actual signature is validated at runtime and must be:
// func(ctx context.Context, webhookPayload *ProviderPayloadType, userPayload *UserDefinedType) error
// where:
// - ProviderPayloadType is the webhook provider's payload (e.g. *stripe.PaymentIntent)
// - UserDefinedType is user-defined and validated using go-playground/validator.
type EventHandlerFunc interface{}

// WebhookHandlerOption extends handler configuration with webhook-specific fields.
type WebhookHandlerOption struct {
	EventHandlers map[string]EventHandlerFunc
	// When true: user metadata validation/unmarshal errors produce 400 with ErrorResponse.
	// When false (default): handlers receive a nil user metadata pointer on validation errors.
	StrictUserValidation bool
}

// WebhookOption is a function that configures WebhookHandlerOption.
type WebhookOption func(*WebhookHandlerOption)

// WithEventHandler registers a type-safe handler for a specific event type.
// P is the provider payload type (e.g., *stripe.PaymentIntent), U is the user metadata type.
// Handlers are error-only; successful processing results in provider SuccessResponse.
func WithEventHandler[P any, U any](eventType string, handler func(context.Context, *P, *U) error) WebhookOption {
	return func(h *WebhookHandlerOption) {
		if h.EventHandlers == nil {
			h.EventHandlers = make(map[string]EventHandlerFunc)
		}
		h.EventHandlers[eventType] = handler
	}
}

// webhookHandlerRegistry stores original webhook handlers by their address.
// This allows the OpenAPI generator to access the handler's response methods via reflection.
var webhookHandlerRegistry = make(map[uintptr]interface{})

// GetOriginalWebhookHandler extracts the original webhook handler from a handler function.
func GetOriginalWebhookHandler(handler interface{}) interface{} {
	if handlerFunc, ok := handler.(http.HandlerFunc); ok {
		// Get the function pointer address
		handlerPtr := reflect.ValueOf(handlerFunc).Pointer()
		if entry, exists := webhookHandlerRegistry[handlerPtr]; exists {
			switch v := entry.(type) {
			case webhookRegistryEntry:
				return v.original
			default:
				return v
			}
		}
	}
	return nil
}

// registerWebhookHandler stores the original webhook handler in the registry.
func registerWebhookHandler(handlerFunc http.HandlerFunc, entry webhookRegistryEntry) {
	handlerPtr := reflect.ValueOf(handlerFunc).Pointer()
	webhookHandlerRegistry[handlerPtr] = entry
}

type webhookRegistryEntry struct {
	original      interface{}
	providerInfo  *WebhookProviderInfo
	handledEvents []string
	handlersMeta  []RegisteredEventHandler
}

// GetWebhookRouteMetadata returns provider info and handled events for a registered webhook handler.
func GetWebhookRouteMetadata(handler http.HandlerFunc) (*WebhookProviderInfo, []string) {
	handlerPtr := reflect.ValueOf(handler).Pointer()
	if entry, ok := webhookHandlerRegistry[handlerPtr]; ok {
		if e, ok2 := entry.(webhookRegistryEntry); ok2 {
			return e.providerInfo, e.handledEvents
		}
	}
	return nil, nil
}

// RegisteredEventHandler captures metadata about a registered event handler for documentation.
type RegisteredEventHandler struct {
	EventType           string
	HandlerFunc         EventHandlerFunc
	HandlerName         string
	ProviderPayloadType reflect.Type
	UserMetadataType    reflect.Type
}

// GetWebhookHandlersMetadata returns the list of registered handlers metadata for the webhook http handler.
func GetWebhookHandlersMetadata(handler http.HandlerFunc) []RegisteredEventHandler {
	handlerPtr := reflect.ValueOf(handler).Pointer()
	if entry, ok := webhookHandlerRegistry[handlerPtr]; ok {
		if e, ok2 := entry.(webhookRegistryEntry); ok2 {
			return e.handlersMeta
		}
	}
	return nil
}

// WebhookHandlerFunc creates an HTTP handler from a webhook handler using conventional request parsing.
func WebhookHandlerFunc[T WebhookRequest](handler WebhookHandler[T], opts ...WebhookOption) http.HandlerFunc {
	options := buildWebhookOptions(opts)
	validateEventTypes(handler, options)

	httpHandlerFunc := createWebhookHTTPHandler(handler, options)
	registerWebhookMetadata(httpHandlerFunc, handler, options)

	return httpHandlerFunc
}

// buildWebhookOptions builds webhook options from the provided option functions.
func buildWebhookOptions(opts []WebhookOption) *WebhookHandlerOption {
	options := &WebhookHandlerOption{
		EventHandlers: make(map[string]EventHandlerFunc),
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}

// validateEventTypes validates all registered event types against the provider's advertised set.
func validateEventTypes[T WebhookRequest](handler WebhookHandler[T], options *WebhookHandlerOption) {
	validTypes := buildValidTypesMap(handler.GetValidEventTypes())

	for eventType := range options.EventHandlers {
		if _, ok := validTypes[eventType]; !ok && len(validTypes) > 0 {
			panic(fmt.Sprintf("unknown event type '%s' for webhook provider %T", eventType, handler))
		}
	}
}

// buildValidTypesMap creates a map of valid event types for quick lookup.
func buildValidTypesMap(validEventTypes []string) map[string]struct{} {
	validTypes := make(map[string]struct{}, len(validEventTypes))
	for _, t := range validEventTypes {
		validTypes[t] = struct{}{}
	}
	return validTypes
}

// createWebhookHTTPHandler creates the actual HTTP handler function.
func createWebhookHTTPHandler[T WebhookRequest](handler WebhookHandler[T], options *WebhookHandlerOption) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := processWebhookRequest(w, r, handler, options); err != nil {
			// Error already handled in processWebhookRequest
			return
		}
	}
}

// processWebhookRequest processes a webhook request through all stages.
func processWebhookRequest[T WebhookRequest](w http.ResponseWriter, r *http.Request, handler WebhookHandler[T], options *WebhookHandlerOption) error {
	// 1. Parse request using Gork's conventional request parsing
	var req T
	if err := ParseRequest(r, &req); err != nil {
		writeWebhookJSON(w, http.StatusBadRequest, handler.ErrorResponse(err))
		return err
	}

	// 2. Parse and verify webhook using conventional struct
	event, err := handler.ParseRequest(req)
	if err != nil {
		writeWebhookJSON(w, http.StatusUnauthorized, handler.ErrorResponse(err))
		return err
	}

	// 3. Find and invoke appropriate handler
	return handleWebhookEvent(w, r, handler, options, event)
}

// handleWebhookEvent handles the webhook event by finding and invoking the appropriate handler.
func handleWebhookEvent[T WebhookRequest](w http.ResponseWriter, r *http.Request, handler WebhookHandler[T], options *WebhookHandlerOption, event WebhookEvent) error {
	handlerFunc, exists := options.EventHandlers[event.Type]
	if !exists {
		// Log unhandled event type but return success
		log.Printf("Unhandled webhook event type: %s", event.Type)
		writeWebhookJSON(w, http.StatusOK, handler.SuccessResponse())
		return nil
	}

	// 4. Invoke typed handler with provider payload assertion and optional user metadata validation
	statusCode, err := invokeTypedEventHandler(r.Context(), handlerFunc, event, options.StrictUserValidation)
	if err != nil {
		// Map status code to 4xx/5xx as produced by invocation
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		writeWebhookJSON(w, statusCode, handler.ErrorResponse(err))
		return err
	}

	// On success, write provider's standard success response
	writeWebhookJSON(w, http.StatusOK, handler.SuccessResponse())
	return nil
}

// registerWebhookMetadata registers the webhook handler and its metadata for OpenAPI reflection.
func registerWebhookMetadata[T WebhookRequest](httpHandlerFunc http.HandlerFunc, handler WebhookHandler[T], options *WebhookHandlerOption) {
	handled, handlersMeta := buildHandlerMetadata(options)

	info := handler.ProviderInfo()
	pinfo := &info

	registerWebhookHandler(httpHandlerFunc, webhookRegistryEntry{
		original:      handler,
		providerInfo:  pinfo,
		handledEvents: handled,
		handlersMeta:  handlersMeta,
	})
}

// buildHandlerMetadata builds the handled events list and detailed handler metadata.
func buildHandlerMetadata(options *WebhookHandlerOption) ([]string, []RegisteredEventHandler) {
	handled := make([]string, 0, len(options.EventHandlers))
	handlersMeta := make([]RegisteredEventHandler, 0, len(options.EventHandlers))

	// Deterministic order: sort event types before building slices
	keys := make([]string, 0, len(options.EventHandlers))
	for evt := range options.EventHandlers {
		keys = append(keys, evt)
	}
	sort.Strings(keys)

	for _, evt := range keys {
		fn := options.EventHandlers[evt]
		handled = append(handled, evt)

		fnType := reflect.TypeOf(fn)
		var providerT, userT reflect.Type
		if fnType.Kind() == reflect.Func && fnType.NumIn() == 3 {
			providerT = fnType.In(1)
			userT = fnType.In(2)
		}

		handlersMeta = append(handlersMeta, RegisteredEventHandler{
			EventType:           evt,
			HandlerFunc:         fn,
			HandlerName:         getFunctionName(fn),
			ProviderPayloadType: providerT,
			UserMetadataType:    userT,
		})
	}

	return handled, handlersMeta
}

// invokeTypedEventHandler validates the handler signature, asserts provider payload type,
// unmarshals and optionally validates user metadata, and invokes the handler.
// Returns statusCode (0 if none), error.
func invokeTypedEventHandler(ctx context.Context, handler EventHandlerFunc, event WebhookEvent, strict bool) (int, error) {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	// Validate handler signature: func(context.Context, *P, *U) error
	if err := validateEventHandlerSignature(handlerType); err != nil {
		return http.StatusInternalServerError, err
	}

	// Build provider payload argument
	providerParamType := handlerType.In(1)
	var providerArg reflect.Value
	if event.ProviderObject == nil {
		// Allow nil if handler expects interface pointer
		providerArg = reflect.Zero(providerParamType)
	} else {
		evVal := reflect.ValueOf(event.ProviderObject)
		if !evVal.Type().AssignableTo(providerParamType) {
			return http.StatusInternalServerError, fmt.Errorf("provider payload type mismatch: have %T, need %s", event.ProviderObject, providerParamType.String())
		}
		providerArg = evVal
	}

	// Build user metadata argument
	userParamType := handlerType.In(2)
	var userArg reflect.Value
	if len(event.UserMetaJSON) == 0 {
		userArg = reflect.Zero(userParamType)
	} else {
		// Create appropriate value (pointer type expected; already validated by signature check)
		elem := userParamType.Elem()
		userPtr := reflect.New(elem)
		// Unmarshal
		if err := json.Unmarshal(event.UserMetaJSON, userPtr.Interface()); err != nil {
			if strict {
				return http.StatusBadRequest, fmt.Errorf("invalid user metadata: %w", err)
			}
			userArg = reflect.Zero(userParamType)
		} else {
			userArg = userPtr
		}
	}

	// Call the handler
	results := handlerValue.Call([]reflect.Value{reflect.ValueOf(ctx), providerArg, userArg})

	var err error
	if !results[0].IsNil() {
		err = results[0].Interface().(error)
	}
	return 0, err
}

// validateEventHandlerSignature validates that the handler has the correct signature:
// func(context.Context, *ProviderPayloadType, *UserDefinedType) error.
func validateEventHandlerSignature(handlerType reflect.Type) error {
	if handlerType.Kind() != reflect.Func {
		return fmt.Errorf("handler must be a function")
	}

	if handlerType.NumIn() != 3 {
		return fmt.Errorf("handler must accept exactly 3 parameters (context.Context, *ProviderPayload, *UserPayload)")
	}

	if handlerType.NumOut() != 1 {
		return fmt.Errorf("handler must return exactly 1 value (error)")
	}

	// Verify first parameter is context.Context
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !handlerType.In(0).Implements(contextType) {
		return fmt.Errorf("handler first parameter must be context.Context")
	}

	// Second parameter is the provider payload pointer (e.g. *stripe.PaymentIntent)
	if handlerType.In(1).Kind() != reflect.Ptr {
		return fmt.Errorf("handler second parameter must be a pointer to provider payload type")
	}
	// Third parameter is the user-defined payload pointer (any type)
	if handlerType.In(2).Kind() != reflect.Ptr {
		return fmt.Errorf("handler third parameter must be a pointer to user metadata type")
	}

	// Verify return value is error
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !handlerType.Out(0).Implements(errorType) {
		return fmt.Errorf("handler return value must be error")
	}

	return nil
}

// Removed legacy extract/validate helpers in favor of provider-driven WebhookEvent and typed invocation.

// writeWebhookJSON writes a JSON response for webhooks.
func writeWebhookJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
