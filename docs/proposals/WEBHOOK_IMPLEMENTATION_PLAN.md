# Stripe Webhook Support Implementation Plan

## Overview

This document outlines the implementation plan for adding Stripe webhook support to the OpenAPI generator. The design follows a builder pattern that allows type-safe registration of event-specific handlers while maintaining the existing architecture's elegance.

## Goals

1. **Type Safety**: Each webhook event handler receives strongly-typed event data
2. **Provider Agnostic**: Design supports multiple webhook providers (Stripe, GitHub, SendGrid, etc.)
3. **OpenAPI Generation**: Automatically generate accurate OpenAPI schemas for webhook endpoints
4. **Developer Experience**: Simple, intuitive API for registering webhook handlers
5. **Backward Compatibility**: No breaking changes to existing functionality

## Proposed API

### Handler Registration Pattern

```go
// User-defined event payload structures
type CheckoutMetadata struct {
    ProjectID    string `json:"project_id" validate:"required"`
    UserID       string `json:"user_id" validate:"required"`
    PlanType     string `json:"plan_type" validate:"oneof=basic premium enterprise"`
    CustomField  string `json:"custom_field,omitempty"`
}

type SubscriptionMetadata struct {
    TenantID     string `json:"tenant_id" validate:"required"`
    Environment  string `json:"environment" validate:"oneof=dev staging production"`
}

// Individual typed event handlers with user-defined generic payload
func paymentIntentSucceeded(ctx context.Context, pi *stripe.PaymentIntent, metadata *CheckoutMetadata) (*stripe.Response, error) {
    // Standard Stripe payment intent data is always available
    log.Printf("Payment %s succeeded for amount %d", pi.ID, pi.Amount)
    
    // If validation fails, metadata will be nil
    if metadata == nil {
        log.Printf("No valid metadata provided for payment %s", pi.ID)
        // Can still process the payment without metadata
        return &stripe.Response{Received: true}, nil
    }
    
    // Business logic with validated user-defined payload
    log.Printf("Processing payment for project %s, user %s, plan %s", 
        metadata.ProjectID, metadata.UserID, metadata.PlanType)
    
    // Update project billing, send confirmation email, etc.
    return &stripe.Response{Received: true}, nil
}

func customerSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription, metadata *SubscriptionMetadata) (*stripe.Response, error) {
    // Standard Stripe subscription data is always available
    log.Printf("Subscription %s cancelled for customer %s", sub.ID, sub.Customer)
    
    // If validation fails, metadata will be nil
    if metadata == nil {
        log.Printf("No valid metadata provided for subscription %s", sub.ID)
        return &stripe.Response{Received: true}, nil
    }
    
    // Business logic with validated user-defined payload
    log.Printf("Cancelling subscription for tenant %s in %s environment", 
        metadata.TenantID, metadata.Environment)
    
    return &stripe.Response{Received: true}, nil
}

// Route registration with automatic event type validation
mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
    stripe.NewHandler("whsec_test_secret"),
    api.WithEventHandler("payment_intent.succeeded", paymentIntentSucceeded),
    api.WithEventHandler("customer.subscription.deleted", customerSubscriptionDeleted),
    // This would panic at registration time due to validation in WebhookHandlerFunc:
    // api.WithEventHandler("invalid.event.type", someHandler),
    api.WithTags("webhooks", "stripe"),
))
```

## Architecture

### Core Components

#### 1. Generic Webhook Interface (`pkg/api/webhook.go`)

```go
// WebhookHandler defines the interface for webhook providers
type WebhookHandler interface {
    // ParseRequest verifies the webhook signature and extracts event data
    ParseRequest(rawBody []byte, headers http.Header) (eventType string, eventData interface{}, error)
    
    // SuccessResponse returns the appropriate success response for the provider
    SuccessResponse() interface{}
    
    // ErrorResponse returns the appropriate error response for the provider
    ErrorResponse(err error) interface{}
}

// EventHandlerFunc is a generic interface for event handlers
// Actual signature is validated at runtime and must be:
// func(ctx context.Context, webhookPayload *ProviderPayloadType, userPayload *UserDefinedType) (response, error)
// where:
// - ProviderPayloadType is the webhook provider's payload (e.g. *stripe.PaymentIntent)
// - UserDefinedType is user-defined and validated using go-playground/validator
type EventHandlerFunc interface{}

// WebhookHandlerFunc creates an HTTP handler from a webhook handler
func WebhookHandlerFunc(handler WebhookHandler, opts ...Option) http.HandlerFunc {
    options := &HandlerOption{
        EventHandlers: make(map[string]EventHandlerFunc),
    }
    
    for _, opt := range opts {
        opt(options)
    }
    
    // Validate all registered event types at creation time
    if validator, ok := handler.(EventTypeValidator); ok {
        for eventType := range options.EventHandlers {
            if !validator.IsValidEventType(eventType) {
                panic(fmt.Sprintf("unknown event type '%s' for webhook provider %T", eventType, handler))
            }
        }
    }
    
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Read raw body for signature verification
        rawBody, err := io.ReadAll(r.Body)
        if err != nil {
            writeJSON(w, http.StatusBadRequest, handler.ErrorResponse(err))
            return
        }
        
        // 2. Parse and verify webhook
        eventType, eventData, err := handler.ParseRequest(rawBody, r.Header)
        if err != nil {
            writeJSON(w, http.StatusUnauthorized, handler.ErrorResponse(err))
            return
        }
        
        // 3. Find and invoke appropriate handler
        handlerFunc, exists := options.EventHandlers[eventType]
        if !exists {
            // Log unhandled event type but return success
            log.Printf("Unhandled webhook event type: %s", eventType)
            writeJSON(w, http.StatusOK, handler.SuccessResponse())
            return
        }
        
        // 4. Use reflection to invoke typed handler with validation
        response, err := invokeEventHandlerWithValidation(r.Context(), handlerFunc, eventData)
        if err != nil {
            writeJSON(w, http.StatusInternalServerError, handler.ErrorResponse(err))
            return
        }
        
        writeJSON(w, http.StatusOK, response)
    }
}

// WithEventHandler registers a typed handler for a specific event type with user-defined payload validation.
// The handler must have the signature:
// func(ctx context.Context, webhookPayload *ProviderPayloadType, userPayload *UserDefinedType) (response, error)
//
// Where:
// - webhookPayload is the provider's parsed payload (e.g. *stripe.PaymentIntent) - always provided
// - userPayload is user-defined and validated using go-playground/validator - nil if validation fails
func WithEventHandler[T any](eventType string, handler func(context.Context, interface{}, *T) (interface{}, error)) Option {
    return func(h *HandlerOption) {
        if h.EventHandlers == nil {
            h.EventHandlers = make(map[string]EventHandlerFunc)
        }
        h.EventHandlers[eventType] = handler
    }
}

// EventTypeValidator interface for webhook handlers that can validate event types
type EventTypeValidator interface {
    IsValidEventType(eventType string) bool
    GetValidEventTypes() []string
}
```

#### 2. Stripe Implementation (`pkg/webhooks/stripe/`)

```go
// handler.go
package stripe

import (
    "github.com/stripe/stripe-go/v74"
    "github.com/stripe/stripe-go/v74/webhook"
)

type Handler struct {
    endpointSecret string
}

func NewHandler(endpointSecret string) api.WebhookHandler {
    return &Handler{
        endpointSecret: endpointSecret,
    }
}

func (h *Handler) ParseRequest(rawBody []byte, headers http.Header) (string, interface{}, error) {
    // Get Stripe signature header
    signature := headers.Get("Stripe-Signature")
    if signature == "" {
        return "", nil, errors.New("missing Stripe-Signature header")
    }
    
    // Verify webhook signature
    event, err := webhook.ConstructEvent(rawBody, signature, h.endpointSecret)
    if err != nil {
        return "", nil, fmt.Errorf("webhook signature verification failed: %w", err)
    }
    
    // Parse the data.object based on event type using map lookup
    dataObject, err := h.parseEventData(event)
    if err != nil {
        return "", nil, fmt.Errorf("failed to parse event data: %w", err)
    }
    
    return event.Type, dataObject, nil
}

// Default map of event type patterns to their corresponding Go types
var defaultEventTypeMap = map[string]reflect.Type{
    "payment_intent.":           reflect.TypeOf(stripe.PaymentIntent{}),
    "customer.subscription.":    reflect.TypeOf(stripe.Subscription{}),
    "invoice.":                  reflect.TypeOf(stripe.Invoice{}),
    "charge.":                   reflect.TypeOf(stripe.Charge{}),
    "customer.":                 reflect.TypeOf(stripe.Customer{}),
    "product.":                  reflect.TypeOf(stripe.Product{}),
    "price.":                    reflect.TypeOf(stripe.Price{}),
    "coupon.":                   reflect.TypeOf(stripe.Coupon{}),
    "discount.":                 reflect.TypeOf(stripe.Discount{}),
    "transfer.":                 reflect.TypeOf(stripe.Transfer{}),
    "payout.":                   reflect.TypeOf(stripe.Payout{}),
    "balance.":                  reflect.TypeOf(stripe.Balance{}),
    "application_fee.":          reflect.TypeOf(stripe.ApplicationFee{}),
    "account.":                  reflect.TypeOf(stripe.Account{}),
    "capability.":               reflect.TypeOf(stripe.Capability{}),
    "person.":                   reflect.TypeOf(stripe.Person{}),
    "topup.":                    reflect.TypeOf(stripe.Topup{}),
    "review.":                   reflect.TypeOf(stripe.Review{}),
    "radar.early_fraud_warning.": reflect.TypeOf(stripe.RadarEarlyFraudWarning{}),
    "recipient.":                reflect.TypeOf(stripe.Recipient{}),
    "sku.":                      reflect.TypeOf(stripe.SKU{}),
    "order.":                    reflect.TypeOf(stripe.Order{}),
    "order_return.":             reflect.TypeOf(stripe.OrderReturn{}),
    "plan.":                     reflect.TypeOf(stripe.Plan{}),
    "source.":                   reflect.TypeOf(stripe.Source{}),
    "payment_method.":           reflect.TypeOf(stripe.PaymentMethod{}),
    "setup_intent.":             reflect.TypeOf(stripe.SetupIntent{}),
    "issuing_authorization.":    reflect.TypeOf(stripe.IssuingAuthorization{}),
    "issuing_card.":             reflect.TypeOf(stripe.IssuingCard{}),
    "issuing_cardholder.":       reflect.TypeOf(stripe.IssuingCardholder{}),
    "issuing_dispute.":          reflect.TypeOf(stripe.IssuingDispute{}),
    "issuing_transaction.":      reflect.TypeOf(stripe.IssuingTransaction{}),
    "terminal.reader.":          reflect.TypeOf(stripe.TerminalReader{}),
    "terminal.location.":        reflect.TypeOf(stripe.TerminalLocation{}),
    "file.":                     reflect.TypeOf(stripe.File{}),
    "reporting.report_run.":     reflect.TypeOf(stripe.ReportingReportRun{}),
    "reporting.report_type.":    reflect.TypeOf(stripe.ReportingReportType{}),
    "sigma.scheduled_query_run.": reflect.TypeOf(stripe.SigmaScheduledQueryRun{}),
    "webhook_endpoint.":         reflect.TypeOf(stripe.WebhookEndpoint{}),
}

func (h *Handler) parseEventData(event stripe.Event) (interface{}, error) {
    // Find matching type by prefix
    var targetType reflect.Type
    for prefix, typ := range defaultEventTypeMap {
        if strings.HasPrefix(event.Type, prefix) {
            targetType = typ
            break
        }
    }
    
    // If no specific type found, return raw data
    if targetType == nil {
        return event.Data.Raw, nil
    }
    
    // Create new instance of the target type
    objPtr := reflect.New(targetType)
    
    // Unmarshal into the typed object
    if err := json.Unmarshal(event.Data.Raw, objPtr.Interface()); err != nil {
        return nil, fmt.Errorf("failed to unmarshal %s: %w", event.Type, err)
    }
    
    return objPtr.Interface(), nil
}

func (h *Handler) SuccessResponse() interface{} {
    return &Response{Received: true}
}

func (h *Handler) ErrorResponse(err error) interface{} {
    return &Response{
        Received: false,
        Error:    err.Error(),
    }
}

// IsValidEventType validates if the given event type is known to Stripe
func (h *Handler) IsValidEventType(eventType string) bool {
    for _, knownEvent := range knownStripeEventTypes {
        if eventType == knownEvent {
            return true
        }
    }
    return false
}

// GetValidEventTypes returns all valid Stripe event types
func (h *Handler) GetValidEventTypes() []string {
    return knownStripeEventTypes
}

// Known Stripe event types - comprehensive list of all documented events
var knownStripeEventTypes = []string{
    // Payment Intent events
    "payment_intent.amount_capturable_updated",
    "payment_intent.canceled",
    "payment_intent.created",
    "payment_intent.partially_funded",
    "payment_intent.payment_failed",
    "payment_intent.processing",
    "payment_intent.requires_action",
    "payment_intent.succeeded",
    
    // Subscription events
    "customer.subscription.created",
    "customer.subscription.deleted",
    "customer.subscription.paused",
    "customer.subscription.pending_update_applied",
    "customer.subscription.pending_update_expired",
    "customer.subscription.resumed",
    "customer.subscription.trial_will_end",
    "customer.subscription.updated",
    
    // Invoice events
    "invoice.created",
    "invoice.deleted",
    "invoice.finalization_failed",
    "invoice.finalized",
    "invoice.marked_uncollectible",
    "invoice.paid",
    "invoice.payment_action_required",
    "invoice.payment_failed",
    "invoice.payment_succeeded",
    "invoice.sent",
    "invoice.upcoming",
    "invoice.updated",
    "invoice.voided",
    
    // Customer events
    "customer.created",
    "customer.deleted",
    "customer.updated",
    "customer.discount.created",
    "customer.discount.deleted",
    "customer.discount.updated",
    "customer.source.created",
    "customer.source.deleted",
    "customer.source.expiring",
    "customer.source.updated",
    "customer.tax_id.created",
    "customer.tax_id.deleted",
    "customer.tax_id.updated",
    
    // Charge events
    "charge.captured",
    "charge.dispute.created",
    "charge.dispute.funds_reinstated",
    "charge.dispute.funds_withdrawn",
    "charge.dispute.updated",
    "charge.expired",
    "charge.failed",
    "charge.pending",
    "charge.succeeded",
    "charge.updated",
    
    // Add more as needed...
}

// types.go
package stripe

// Response is the standard Stripe webhook response
type Response struct {
    Received bool   `json:"received"`
    Error    string `json:"error,omitempty"`
}
```

#### 3. Reflection-Based Handler Invocation with Validation

```go
// invokeEventHandlerWithValidation uses reflection to call the typed handler with user payload validation.
// The handler signature must be: func(context.Context, *ProviderPayloadType, *UserDefinedType) (interface{}, error)
// The provider payload is always passed as-is, while the user-defined payload is validated and set to nil if validation fails.
func invokeEventHandlerWithValidation(ctx context.Context, handler EventHandlerFunc, eventData interface{}) (interface{}, error) {
    handlerValue := reflect.ValueOf(handler)
    handlerType := handlerValue.Type()
    
    // Validate handler signature
    if err := validateEventHandlerSignature(handlerType); err != nil {
        return nil, err
    }
    
    // Extract the provider payload and user-defined metadata from eventData
    // This assumes eventData contains both the provider payload and user metadata
    providerPayload, userMetadata := extractPayloads(eventData)
    
    // Get the expected user payload type (third parameter)
    userPayloadType := handlerType.In(2)
    isPointer := userPayloadType.Kind() == reflect.Ptr
    if isPointer {
        userPayloadType = userPayloadType.Elem()
    }
    
    // Prepare the user payload argument with validation
    var userPayloadArg reflect.Value
    if userMetadata == nil {
        // If no user metadata, pass nil
        userPayloadArg = reflect.Zero(handlerType.In(2))
    } else {
        // Try to convert and validate the user metadata
        validatedPayload, err := validateEventPayload(userMetadata, userPayloadType, isPointer)
        if err != nil {
            // Validation failed, pass nil
            userPayloadArg = reflect.Zero(handlerType.In(2))
        } else {
            userPayloadArg = validatedPayload
        }
    }
    
    // Call the handler
    results := handlerValue.Call([]reflect.Value{
        reflect.ValueOf(ctx),
        reflect.ValueOf(providerPayload),
        userPayloadArg,
    })
    
    // Extract response and error
    var response interface{}
    if !results[0].IsNil() {
        response = results[0].Interface()
    }
    
    var handlerError error
    if !results[1].IsNil() {
        handlerError = results[1].Interface().(error)
    }
    
    return response, handlerError
}

// validateEventHandlerSignature validates that the handler has the correct signature:
// func(context.Context, *ProviderPayloadType, *UserDefinedType) (interface{}, error)
func validateEventHandlerSignature(handlerType reflect.Type) error {
    if handlerType.Kind() != reflect.Func {
        return errors.New("handler must be a function")
    }
    
    if handlerType.NumIn() != 3 {
        return errors.New("handler must accept exactly 3 parameters (context.Context, *ProviderPayload, *UserPayload)")
    }
    
    if handlerType.NumOut() != 2 {
        return errors.New("handler must return exactly 2 values (response, error)")
    }
    
    // Verify first parameter is context.Context
    if !handlerType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
        return errors.New("handler first parameter must be context.Context")
    }
    
    // Second parameter is the provider payload (e.g. *stripe.PaymentIntent) - can be any type
    // Third parameter is the user-defined payload - can be any type
    
    // Verify second return value is error
    if !handlerType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
        return errors.New("handler second return value must be error")
    }
    
    return nil
}

// extractPayloads separates the provider payload from user-defined metadata
// This depends on how the webhook provider structures the incoming data
func extractPayloads(eventData interface{}) (providerPayload interface{}, userMetadata interface{}) {
    // Implementation depends on webhook provider format
    // For Stripe, this might extract from metadata fields or custom data
    // For now, assume eventData is the provider payload and metadata comes from another source
    return eventData, nil // placeholder - needs provider-specific implementation
}

// validateEventPayload attempts to convert eventData to the expected payload type and validate it.
// Returns the validated payload as a reflect.Value, or an error if validation fails.
func validateEventPayload(eventData interface{}, payloadType reflect.Type, isPointer bool) (reflect.Value, error) {
    // Create a new instance of the payload type
    var payloadPtr reflect.Value
    if isPointer {
        // Create pointer to the type
        payloadPtr = reflect.New(payloadType)
    } else {
        // For non-pointer types, we still need a pointer for JSON unmarshaling
        payloadPtr = reflect.New(payloadType)
    }
    
    // Convert eventData to JSON bytes for unmarshaling
    var jsonData []byte
    var err error
    
    switch data := eventData.(type) {
    case []byte:
        jsonData = data
    case string:
        jsonData = []byte(data)
    default:
        // Convert to JSON and back (handles map[string]interface{} and other types)
        jsonData, err = json.Marshal(data)
        if err != nil {
            return reflect.Value{}, fmt.Errorf("failed to marshal event data: %w", err)
        }
    }
    
    // Unmarshal into the payload struct
    if err := json.Unmarshal(jsonData, payloadPtr.Interface()); err != nil {
        return reflect.Value{}, fmt.Errorf("failed to unmarshal event data: %w", err)
    }
    
    // Validate the payload using go-playground/validator
    if err := validate.Struct(payloadPtr.Interface()); err != nil {
        return reflect.Value{}, fmt.Errorf("payload validation failed: %w", err)
    }
    
    // Return the appropriate value based on whether the expected type is a pointer
    if isPointer {
        return payloadPtr, nil
    }
    return payloadPtr.Elem(), nil
}
```

### OpenAPI Generation

#### 1. Route Detection Updates

The route detector needs to recognize the `WebhookHandlerFunc` pattern:

```go
// internal/generator/route_detector.go additions
func (d *RouteDetector) detectWebhookHandler(call *ast.CallExpr) *WebhookRoute {
    // Check if this is api.WebhookHandlerFunc
    if !isWebhookHandlerFunc(call) {
        return nil
    }
    
    // Extract webhook handler and options
    if len(call.Args) < 1 {
        return nil
    }
    
    webhookRoute := &WebhookRoute{
        Provider: extractProviderName(call.Args[0]),
        Events:   make([]string, 0),
    }
    
    // Parse options to find event handlers
    for _, arg := range call.Args[1:] {
        if eventType, handler := extractEventHandler(arg); eventType != "" {
            webhookRoute.Events = append(webhookRoute.Events, eventType)
            webhookRoute.Handlers[eventType] = handler
        }
    }
    
    return webhookRoute
}
```

#### 2. OpenAPI Schema Generation

```go
// internal/generator/openapi_webhook.go
func (g *Generator) generateWebhookOperation(route *WebhookRoute) *openapi.Operation {
    operation := &openapi.Operation{
        Tags:        []string{"webhooks", route.Provider},
        Summary:     fmt.Sprintf("%s webhook endpoint", strings.Title(route.Provider)),
        Description: fmt.Sprintf("Handles incoming webhooks from %s", route.Provider),
        Extensions: map[string]interface{}{
            "x-webhook-provider": route.Provider,
            "x-webhook-events":   route.Events,
        },
    }
    
    // Generate request body schema based on provider
    requestSchema := g.generateWebhookRequestSchema(route.Provider, route.Events)
    operation.RequestBody = &openapi.RequestBody{
        Required: true,
        Content: map[string]*openapi.MediaType{
            "application/json": {
                Schema: requestSchema,
            },
        },
    }
    
    // Generate responses
    operation.Responses = map[string]*openapi.Response{
        "200": {
            Description: "Webhook processed successfully",
            Content: g.generateWebhookResponseContent(route.Provider),
        },
        "400": {
            Description: "Invalid webhook payload",
        },
        "401": {
            Description: "Invalid webhook signature",
        },
    }
    
    return operation
}

func (g *Generator) generateWebhookRequestSchema(provider string, events []string) *openapi.Schema {
    switch provider {
    case "stripe":
        return g.generateStripeWebhookSchema(events)
    case "github":
        return g.generateGitHubWebhookSchema(events)
    default:
        return &openapi.Schema{
            Type:        "object",
            Description: fmt.Sprintf("%s webhook payload", provider),
        }
    }
}
```

#### 3. Stripe-Specific Schema Generation

```go
func (g *Generator) generateStripeWebhookSchema(events []string) *openapi.Schema {
    // Generate oneOf schemas for data.object based on event types
    dataObjectSchemas := make([]*openapi.Schema, 0)
    eventTypeEnum := make([]string, 0)
    
    for _, event := range events {
        eventTypeEnum = append(eventTypeEnum, event)
        
        // Map event type to object schema
        switch {
        case strings.HasPrefix(event, "payment_intent."):
            dataObjectSchemas = append(dataObjectSchemas, &openapi.Schema{
                Ref: "#/components/schemas/StripePaymentIntent",
            })
        case strings.HasPrefix(event, "customer.subscription."):
            dataObjectSchemas = append(dataObjectSchemas, &openapi.Schema{
                Ref: "#/components/schemas/StripeSubscription",
            })
        case strings.HasPrefix(event, "invoice."):
            dataObjectSchemas = append(dataObjectSchemas, &openapi.Schema{
                Ref: "#/components/schemas/StripeInvoice",
            })
        }
    }
    
    return &openapi.Schema{
        Type:     "object",
        Required: []string{"id", "object", "type", "data"},
        Properties: map[string]*openapi.Schema{
            "id": {
                Type:        "string",
                Description: "Unique identifier for the event",
            },
            "object": {
                Type: "string",
                Enum: []interface{}{"event"},
            },
            "type": {
                Type:        "string",
                Enum:        interfaceSlice(eventTypeEnum),
                Description: "The type of event",
            },
            "data": {
                Type:     "object",
                Required: []string{"object"},
                Properties: map[string]*openapi.Schema{
                    "object": {
                        OneOf:         dataObjectSchemas,
                        Discriminator: &openapi.Discriminator{
                            PropertyName: "object",
                            Mapping: map[string]string{
                                "payment_intent": "#/components/schemas/StripePaymentIntent",
                                "subscription":   "#/components/schemas/StripeSubscription",
                                "invoice":        "#/components/schemas/StripeInvoice",
                            },
                        },
                    },
                },
            },
            "created": {
                Type:        "integer",
                Format:      "int64",
                Description: "Time at which the event was created",
            },
            "livemode": {
                Type:        "boolean",
                Description: "Whether this is a live or test event",
            },
        },
    }
}
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- [ ] Implement `WebhookHandler` interface
- [ ] Create `WebhookHandlerFunc` with reflection-based dispatcher
- [ ] Add `WithEventHandler` option
- [ ] Implement basic error handling and logging
- [ ] Write unit tests for core functionality

### Phase 2: Stripe Implementation (Week 1-2)
- [ ] Create Stripe webhook handler
- [ ] Implement signature verification
- [ ] Add type mappings for common Stripe events
- [ ] Create response types
- [ ] Write integration tests with Stripe webhook examples

### Phase 3: OpenAPI Generation (Week 2-3)
- [ ] Update route detector to recognize webhook patterns
- [ ] Implement webhook operation generation
- [ ] Create provider-specific schema generators
- [ ] Add webhook-specific OpenAPI extensions
- [ ] Generate example webhook payloads

### Phase 4: Examples and Documentation (Week 3)
- [ ] Create comprehensive Stripe webhook example
- [ ] Add examples for other providers (GitHub, SendGrid)
- [ ] Write user documentation
- [ ] Update CLAUDE.md with webhook patterns
- [ ] Create testing guide for webhooks

### Phase 5: Advanced Features (Week 4)
- [ ] Add webhook replay protection
- [ ] Implement idempotency handling
- [ ] Create webhook event logger/debugger
- [ ] Add metrics and monitoring hooks
- [ ] Support for webhook event filtering

## Testing Strategy

### Unit Tests
- Test reflection-based handler invocation
- Test signature verification
- Test error handling scenarios
- Test OpenAPI generation for webhooks

### Integration Tests
- Test with real Stripe webhook payloads
- Test with invalid signatures
- Test with unknown event types
- Test concurrent webhook processing

### Example Tests
```go
type TestMetadata struct {
    ProjectID string `json:"project_id" validate:"required"`
    UserID    string `json:"user_id" validate:"required"`
}

func TestStripeWebhookHandler(t *testing.T) {
    // Create test handler
    handler := stripe.NewHandler("whsec_test_secret")
    
    // Create test server
    mux := http.NewServeMux()
    mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
        handler,
        api.WithEventHandler("payment_intent.succeeded", func(ctx context.Context, pi *stripe.PaymentIntent, metadata *TestMetadata) (*stripe.Response, error) {
            // Provider payload is always available
            assert.Equal(t, "pi_test_123", pi.ID)
            
            // User metadata may be nil if validation failed
            if metadata != nil {
                assert.Equal(t, "proj_123", metadata.ProjectID)
                assert.Equal(t, "user_456", metadata.UserID)
            }
            
            return &stripe.Response{Received: true}, nil
        }),
    ))
    
    // Send test webhook with valid metadata
    payload := loadTestPayloadWithMetadata("payment_intent_succeeded.json", map[string]string{
        "project_id": "proj_123",
        "user_id":    "user_456",
    })
    signature := generateTestSignature(payload, "whsec_test_secret")
    
    req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
    req.Header.Set("Stripe-Signature", signature)
    
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.JSONEq(t, `{"received": true}`, rec.Body.String())
}

func TestStripeWebhookHandlerWithInvalidMetadata(t *testing.T) {
    // Create test handler
    handler := stripe.NewHandler("whsec_test_secret")
    
    // Create test server
    mux := http.NewServeMux()
    mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
        handler,
        api.WithEventHandler("payment_intent.succeeded", func(ctx context.Context, pi *stripe.PaymentIntent, metadata *TestMetadata) (*stripe.Response, error) {
            // Provider payload is always available
            assert.Equal(t, "pi_test_123", pi.ID)
            
            // Metadata should be nil due to validation failure
            if metadata == nil {
                return &stripe.Response{Received: true, Message: "Processed without metadata"}, nil
            }
            
            return &stripe.Response{Received: true}, nil
        }),
    ))
    
    // Send webhook with invalid metadata (missing required fields)
    payload := loadTestPayloadWithMetadata("payment_intent_succeeded.json", map[string]string{
        "project_id": "", // Invalid: required field is empty
        // user_id is missing
    })
    signature := generateTestSignature(payload, "whsec_test_secret")
    
    req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
    req.Header.Set("Stripe-Signature", signature)
    
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.JSONEq(t, `{"received": true, "message": "Processed without metadata"}`, rec.Body.String())
}

func TestEventTypeValidation(t *testing.T) {
    handler := stripe.NewHandler("whsec_test_secret")
    
    // This should work fine
    assert.True(t, handler.IsValidEventType("payment_intent.succeeded"))
    assert.True(t, handler.IsValidEventType("customer.subscription.created"))
    
    // These should fail
    assert.False(t, handler.IsValidEventType("invalid.event.type"))
    assert.False(t, handler.IsValidEventType("payment_intent.invalid_action"))
    
    // Test panic on invalid event type during WebhookHandlerFunc creation
    assert.Panics(t, func() {
        api.WebhookHandlerFunc(handler,
            api.WithEventHandler("invalid.event.type", func(ctx context.Context, data interface{}, metadata *TestMetadata) (*stripe.Response, error) {
                return &stripe.Response{Received: true}, nil
            }),
        )
    })
}
```

## Migration Guide

For users who want to add webhook support to existing projects:

1. **Install webhook dependencies**:
   ```bash
   go get github.com/stripe/stripe-go/v74
   ```

2. **Define user payload structures**:
   ```go
   // types/webhooks.go
   type PaymentMetadata struct {
       ProjectID string `json:"project_id" validate:"required"`
       UserID    string `json:"user_id" validate:"required"`
       PlanType  string `json:"plan_type" validate:"oneof=basic premium enterprise"`
   }
   ```

3. **Create webhook handlers**:
   ```go
   // handlers/webhooks/stripe.go
   func HandlePaymentSuccess(ctx context.Context, pi *stripe.PaymentIntent, metadata *PaymentMetadata) (*stripe.Response, error) {
       // Provider payload is always available
       log.Printf("Payment %s succeeded for amount %d", pi.ID, pi.Amount)
       
       // Check if user metadata validation failed
       if metadata == nil {
           log.Printf("No valid metadata provided for payment %s", pi.ID)
           // Can still process payment without metadata
           return &stripe.Response{Received: true}, nil
       }
       
       // Business logic with validated user metadata
       log.Printf("Processing payment for project %s, user %s, plan %s", 
           metadata.ProjectID, metadata.UserID, metadata.PlanType)
       
       return &stripe.Response{Received: true}, nil
   }
   ```

4. **Register webhook endpoint**:
   ```go
   // routes/routes.go
   mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
       stripe.NewHandler(os.Getenv("STRIPE_WEBHOOK_SECRET")),
       api.WithEventHandler("payment_intent.succeeded", handlers.HandlePaymentSuccess),
   ))
   ```

5. **Generate OpenAPI spec**:
   ```bash
   openapi-gen -i ./handlers -r ./routes/routes.go -o openapi.json
   ```

## Security Considerations

1. **Signature Verification**: Always verify webhook signatures
2. **Replay Protection**: Implement timestamp validation
3. **Idempotency**: Handle duplicate events gracefully
4. **Error Handling**: Don't leak sensitive information in error responses
5. **Logging**: Log webhook events for debugging but sanitize sensitive data
6. **Rate Limiting**: Implement rate limiting for webhook endpoints
7. **HTTPS Only**: Always use HTTPS in production

## Future Enhancements

1. **Webhook Testing Tools**: Built-in webhook testing and simulation
2. **Event Store**: Optional event persistence for replay and debugging
3. **Async Processing**: Queue-based webhook processing for high volume
4. **Multi-tenant Support**: Webhook routing based on tenant configuration
5. **GraphQL Webhooks**: Support for GraphQL-based webhook systems
6. **Webhook Transformers**: Transform webhook payloads to internal formats
7. **Conditional Handlers**: Route events based on payload content
8. **Batch Webhooks**: Support for providers that send batched events

## Conclusion

This implementation plan provides a robust, type-safe, and extensible foundation for webhook support in the OpenAPI generator. The design maintains backward compatibility while adding powerful new capabilities that integrate naturally with the existing architecture.