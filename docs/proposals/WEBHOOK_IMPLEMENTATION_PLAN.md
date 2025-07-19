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
// Individual typed event handlers
func paymentIntentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) (*stripe.Response, error) {
    // Business logic for successful payment
    log.Printf("Payment %s succeeded for amount %d", pi.ID, pi.Amount)
    // Update order status, send confirmation email, etc.
    return &stripe.Response{Received: true}, nil
}

func customerSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) (*stripe.Response, error) {
    // Handle subscription cancellation
    log.Printf("Subscription %s cancelled for customer %s", sub.ID, sub.Customer)
    // Update user access, send cancellation email, etc.
    return &stripe.Response{Received: true}, nil
}

// Route registration
mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
    stripe.NewHandler("whsec_test_secret"),
    api.WithEventHandler("payment_intent.succeeded", paymentIntentSucceeded),
    api.WithEventHandler("customer.subscription.deleted", customerSubscriptionDeleted),
    api.WithEventHandler("invoice.payment_failed", invoicePaymentFailed),
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
// Actual signature is validated at runtime
type EventHandlerFunc interface{}

// WebhookHandlerFunc creates an HTTP handler from a webhook handler
func WebhookHandlerFunc(handler WebhookHandler, opts ...Option) http.HandlerFunc {
    options := &HandlerOption{
        EventHandlers: make(map[string]EventHandlerFunc),
    }
    
    for _, opt := range opts {
        opt(options)
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
        
        // 4. Use reflection to invoke typed handler
        response, err := invokeEventHandler(r.Context(), handlerFunc, eventData)
        if err != nil {
            writeJSON(w, http.StatusInternalServerError, handler.ErrorResponse(err))
            return
        }
        
        writeJSON(w, http.StatusOK, response)
    }
}

// WithEventHandler registers a typed handler for a specific event type
func WithEventHandler(eventType string, handler EventHandlerFunc) Option {
    return func(h *HandlerOption) {
        if h.EventHandlers == nil {
            h.EventHandlers = make(map[string]EventHandlerFunc)
        }
        h.EventHandlers[eventType] = handler
    }
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
    
    // Parse the data.object based on event type
    var dataObject interface{}
    switch event.Type {
    case "payment_intent.succeeded", "payment_intent.failed":
        var pi stripe.PaymentIntent
        if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
            return "", nil, fmt.Errorf("failed to parse payment intent: %w", err)
        }
        dataObject = &pi
        
    case "customer.subscription.created", "customer.subscription.deleted":
        var sub stripe.Subscription
        if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
            return "", nil, fmt.Errorf("failed to parse subscription: %w", err)
        }
        dataObject = &sub
        
    case "invoice.payment_failed", "invoice.payment_succeeded":
        var inv stripe.Invoice
        if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
            return "", nil, fmt.Errorf("failed to parse invoice: %w", err)
        }
        dataObject = &inv
        
    default:
        // Return raw data for unhandled types
        dataObject = event.Data.Raw
    }
    
    return event.Type, dataObject, nil
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

// types.go
package stripe

// Response is the standard Stripe webhook response
type Response struct {
    Received bool   `json:"received"`
    Error    string `json:"error,omitempty"`
}
```

#### 3. Reflection-Based Handler Invocation

```go
// invokeEventHandler uses reflection to call the typed handler
func invokeEventHandler(ctx context.Context, handler EventHandlerFunc, eventData interface{}) (interface{}, error) {
    handlerValue := reflect.ValueOf(handler)
    handlerType := handlerValue.Type()
    
    // Validate handler signature
    if handlerType.Kind() != reflect.Func {
        return nil, errors.New("handler must be a function")
    }
    
    if handlerType.NumIn() != 2 {
        return nil, errors.New("handler must accept exactly 2 parameters (context, event)")
    }
    
    if handlerType.NumOut() != 2 {
        return nil, errors.New("handler must return exactly 2 values (response, error)")
    }
    
    // Verify first parameter is context.Context
    if !handlerType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
        return nil, errors.New("handler first parameter must be context.Context")
    }
    
    // Verify second parameter matches event data type
    eventDataType := reflect.TypeOf(eventData)
    if !eventDataType.AssignableTo(handlerType.In(1)) {
        return nil, fmt.Errorf("handler expects %v but got %v", handlerType.In(1), eventDataType)
    }
    
    // Call the handler
    results := handlerValue.Call([]reflect.Value{
        reflect.ValueOf(ctx),
        reflect.ValueOf(eventData),
    })
    
    // Extract response and error
    var response interface{}
    if !results[0].IsNil() {
        response = results[0].Interface()
    }
    
    var err error
    if !results[1].IsNil() {
        err = results[1].Interface().(error)
    }
    
    return response, err
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
func TestStripeWebhookHandler(t *testing.T) {
    // Create test handler
    handler := stripe.NewHandler("whsec_test_secret")
    
    // Create test server
    mux := http.NewServeMux()
    mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
        handler,
        api.WithEventHandler("payment_intent.succeeded", func(ctx context.Context, pi *stripe.PaymentIntent) (*stripe.Response, error) {
            assert.Equal(t, "pi_test_123", pi.ID)
            return &stripe.Response{Received: true}, nil
        }),
    ))
    
    // Send test webhook
    payload := loadTestPayload("payment_intent_succeeded.json")
    signature := generateTestSignature(payload, "whsec_test_secret")
    
    req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
    req.Header.Set("Stripe-Signature", signature)
    
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.JSONEq(t, `{"received": true}`, rec.Body.String())
}
```

## Migration Guide

For users who want to add webhook support to existing projects:

1. **Install webhook dependencies**:
   ```bash
   go get github.com/stripe/stripe-go/v74
   ```

2. **Create webhook handlers**:
   ```go
   // handlers/webhooks/stripe.go
   func HandlePaymentSuccess(ctx context.Context, pi *stripe.PaymentIntent) (*stripe.Response, error) {
       // Your business logic here
       return &stripe.Response{Received: true}, nil
   }
   ```

3. **Register webhook endpoint**:
   ```go
   // routes/routes.go
   mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
       stripe.NewHandler(os.Getenv("STRIPE_WEBHOOK_SECRET")),
       api.WithEventHandler("payment_intent.succeeded", handlers.HandlePaymentSuccess),
   ))
   ```

4. **Generate OpenAPI spec**:
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