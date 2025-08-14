# Design Document

## Overview

This design enhances the existing webhook system with simple, focused improvements based on the current implementation. The webhook system already works well - we're just adding some missing pieces.

## Current Implementation

The webhook system has:
- `WebhookHandler[T]` interface with `ParseRequest`, `SuccessResponse`, `ErrorResponse`, `GetValidEventTypes`, `ProviderInfo`
- `WebhookHandlerFunc` that creates HTTP handlers
- Stripe provider in `pkg/webhooks/stripe/`
- Event handler registration with `WithEventHandler`
- OpenAPI integration

## What We're Adding

### 1. Better Error Logging

Add some log statements to the existing `processWebhookRequest` function:

```go
// In processWebhookRequest, add logging:
log.Printf("Webhook received from %s", handler.ProviderInfo().Name)

// When errors happen:
log.Printf("Webhook error from %s: %v", handler.ProviderInfo().Name, err)
```

### 2. Testing Utilities

Create a mock provider that implements `WebhookHandler[T]`:

```go
type MockProvider[T WebhookRequest] struct {
    events map[string]WebhookEvent
    shouldFail bool
}

func (m *MockProvider[T]) ParseRequest(req T) (WebhookEvent, error) {
    // Return mock events or errors for testing
}
// ... implement other WebhookHandler methods
```

Add signature generation helpers for tests:

```go
func GenerateTestSignature(secret string, payload []byte) string {
    // Generate valid signatures for testing
}
```

### 3. Configuration Validation

Add a function to validate webhook setup:

```go
func ValidateWebhookSetup[T WebhookRequest](handler WebhookHandler[T], options *WebhookHandlerOption) error {
    // Check that event handlers match provider's valid event types
    // Return error if configuration is invalid
}
```

### 4. Additional Providers

Create more webhook providers following the Stripe pattern:
- Each provider implements `WebhookHandler[T]`
- Each has its own request type that implements `WebhookRequest`
- Each handles its own signature verification

## Implementation Details

### 1. Enhanced Logging

Add simple logging to existing functions in `pkg/api/webhook.go`:

```go
// Add to processWebhookRequest function
func processWebhookRequest[T WebhookRequest](...) error {
    log.Printf("Processing webhook from %s", handler.ProviderInfo().Name)
    
    // existing code...
    
    if err := ParseRequest(r, &req); err != nil {
        log.Printf("Webhook parsing failed for %s: %v", handler.ProviderInfo().Name, err)
        // existing error handling...
    }
}
```

### 2. Mock Provider for Testing

Create `pkg/api/webhook_test_utils.go`:

```go
type MockWebhookProvider[T WebhookRequest] struct {
    name        string
    events      map[string]WebhookEvent
    shouldFail  bool
}

func NewMockProvider[T WebhookRequest](name string) *MockWebhookProvider[T]
func (m *MockWebhookProvider[T]) AddEvent(eventType string, event WebhookEvent)
func (m *MockWebhookProvider[T]) SetShouldFail(fail bool)

// Implement WebhookHandler interface
func (m *MockWebhookProvider[T]) ParseRequest(req T) (WebhookEvent, error)
func (m *MockWebhookProvider[T]) SuccessResponse() interface{}
func (m *MockWebhookProvider[T]) ErrorResponse(err error) interface{}
func (m *MockWebhookProvider[T]) GetValidEventTypes() []string
func (m *MockWebhookProvider[T]) ProviderInfo() WebhookProviderInfo
```

### 3. Test Signature Utilities

Add to `pkg/webhooks/stripe/test_utils.go`:

```go
func GenerateTestStripeSignature(secret string, payload []byte, timestamp time.Time) string
func CreateTestStripeEvent(eventType string, payload interface{}) []byte
```

### 4. Enhanced Event Validation

Modify the existing `validateEventTypes` function in `pkg/api/webhook.go` to provide better error messages:

```go
// Enhance existing validateEventTypes function
func validateEventTypes[T WebhookRequest](handler WebhookHandler[T], options *WebhookHandlerOption) {
    validTypes := buildValidTypesMap(handler.GetValidEventTypes())

    for eventType := range options.EventHandlers {
        if _, ok := validTypes[eventType]; !ok && len(validTypes) > 0 {
            // Enhanced panic message with provider info
            panic(fmt.Sprintf("unknown event type '%s' for webhook provider %s (%s). Valid types: %v", 
                eventType, 
                handler.ProviderInfo().Name,
                handler.ProviderInfo().Website,
                handler.GetValidEventTypes()))
        }
    }
}
```

## Files to Modify/Create

### New Files
- `pkg/api/webhook_test_utils.go` - Mock providers and test utilities
- `pkg/webhooks/stripe/test_utils.go` - Stripe-specific test helpers

### Modified Files  
- `pkg/api/webhook.go` - Add logging and validation functions

## Backward Compatibility

All changes are additive:
- Existing webhook handlers continue to work unchanged
- New logging is optional and doesn't affect functionality
- Test utilities are separate and don't impact production code
- Configuration validation is opt-in

## Backward Compatibility

All enhancements will maintain full backward compatibility with the existing webhook system:

- Existing `WebhookHandler[T]` implementations will continue to work unchanged
- Current Stripe webhook implementation will remain functional
- Existing event handler registration patterns will be preserved
- OpenAPI generation will continue to work with existing handlers

New features will be opt-in through:
- Interface extensions (optional methods)
- Configuration options with sensible defaults
- Factory patterns for enhanced providers
- Utility functions that complement existing APIs

## Migration Path

For users wanting to adopt the enhanced features:

1. **Logging Enhancement**: Add logger configuration to existing handlers
2. **Error Handling**: Opt into structured error handling through configuration
3. **Testing**: Adopt new testing utilities incrementally
4. **Type Safety**: Migrate to strongly-typed event definitions as needed
5. **Multiple Providers**: Use provider registry for multi-provider setups

The migration will be gradual and non-breaking, allowing users to adopt enhancements at their own pace.