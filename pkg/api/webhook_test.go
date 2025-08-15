package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Test webhook request types
type TestWebhookRequest struct {
	Headers struct {
		Signature string `gork:"X-Test-Signature" validate:"required"`
	}
	Body []byte
}

func (TestWebhookRequest) WebhookRequest() {}

// Test webhook handler
type TestWebhookHandler struct {
	secret string
}

func NewTestWebhookHandler(secret string) WebhookHandler[TestWebhookRequest] {
	return &TestWebhookHandler{secret: secret}
}

func (h *TestWebhookHandler) ParseRequest(req TestWebhookRequest) (WebhookEvent, error) {
	if req.Headers.Signature == "" {
		return WebhookEvent{}, fmt.Errorf("missing signature")
	}

	if req.Headers.Signature != h.secret {
		return WebhookEvent{}, fmt.Errorf("invalid signature")
	}

	// Simple event parsing - in real implementation this would parse the body
	if strings.Contains(string(req.Body), "payment") {
		provider := map[string]string{"id": "pi_123", "amount": "1000"}
		return WebhookEvent{Type: "payment.succeeded", ProviderObject: &provider}, nil
	}

	return WebhookEvent{Type: "unknown.event", ProviderObject: req.Body}, nil
}

func (h *TestWebhookHandler) SuccessResponse() interface{} {
	return map[string]bool{"received": true}
}

func (h *TestWebhookHandler) ErrorResponse(err error) interface{} {
	return map[string]interface{}{
		"received": false,
		"error":    err.Error(),
	}
}

func (h *TestWebhookHandler) IsValidEventType(eventType string) bool {
	validEvents := []string{"payment.succeeded", "payment.failed", "subscription.created"}
	for _, valid := range validEvents {
		if eventType == valid {
			return true
		}
	}
	return false
}

func (h *TestWebhookHandler) GetValidEventTypes() []string {
	return []string{"payment.succeeded", "payment.failed", "subscription.created"}
}

func (h *TestWebhookHandler) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "TestProvider"}
}

// Test user metadata types
type PaymentMetadata struct {
	ProjectID string `json:"project_id" validate:"required"`
	UserID    string `json:"user_id" validate:"required"`
}

func TestWebhookRequest_MarkerInterface(t *testing.T) {
	var req TestWebhookRequest

	// Test that it implements WebhookRequest interface
	var _ WebhookRequest = req

	// Test marker method exists
	req.WebhookRequest()
}

func TestWebhookHandler_Interface(t *testing.T) {
	handler := NewTestWebhookHandler("test-secret")

	// Test that it implements WebhookHandler interface
	var _ WebhookHandler[TestWebhookRequest] = handler

	// Event type validation is provided via GetValidEventTypes only in the new API
}

func TestWebhookHandler_ParseRequest(t *testing.T) {
	handler := NewTestWebhookHandler("valid-secret")

	tests := []struct {
		name          string
		req           TestWebhookRequest
		expectedEvent string
		expectError   bool
	}{
		{
			name: "valid payment event",
			req: TestWebhookRequest{
				Headers: struct {
					Signature string `gork:"X-Test-Signature" validate:"required"`
				}{Signature: "valid-secret"},
				Body: []byte(`{"event": "payment", "data": {"id": "pi_123"}}`),
			},
			expectedEvent: "payment.succeeded",
		},
		{
			name: "missing signature",
			req: TestWebhookRequest{
				Headers: struct {
					Signature string `gork:"X-Test-Signature" validate:"required"`
				}{Signature: ""},
				Body: []byte(`{"event": "payment"}`),
			},
			expectError: true,
		},
		{
			name: "invalid signature",
			req: TestWebhookRequest{
				Headers: struct {
					Signature string `gork:"X-Test-Signature" validate:"required"`
				}{Signature: "wrong-secret"},
				Body: []byte(`{"event": "payment"}`),
			},
			expectError: true,
		},
		{
			name: "unknown event",
			req: TestWebhookRequest{
				Headers: struct {
					Signature string `gork:"X-Test-Signature" validate:"required"`
				}{Signature: "valid-secret"},
				Body: []byte(`{"event": "unknown"}`),
			},
			expectedEvent: "unknown.event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := handler.ParseRequest(tt.req)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				if ev.Type != tt.expectedEvent {
					t.Errorf("expected event type %q, got %q", tt.expectedEvent, ev.Type)
				}
				if ev.ProviderObject == nil {
					t.Error("expected event data, got nil")
				}
			}
		})
	}
}

func TestWebhookHandler_ResponseMethods(t *testing.T) {
	handler := NewTestWebhookHandler("test-secret")

	t.Run("SuccessResponse", func(t *testing.T) {
		response := handler.SuccessResponse()
		responseMap, ok := response.(map[string]bool)
		if !ok {
			t.Errorf("expected map[string]bool, got %T", response)
			return
		}

		if !responseMap["received"] {
			t.Error("expected received to be true")
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		testError := fmt.Errorf("test error")
		response := handler.ErrorResponse(testError)
		responseMap, ok := response.(map[string]interface{})
		if !ok {
			t.Errorf("expected map[string]interface{}, got %T", response)
			return
		}

		if responseMap["received"] != false {
			t.Error("expected received to be false")
		}

		if responseMap["error"] != "test error" {
			t.Errorf("expected error message 'test error', got %v", responseMap["error"])
		}
	})
}

func TestEventTypeValidator(t *testing.T) {
	handler := NewTestWebhookHandler("test-secret")
	validator := handler.(*TestWebhookHandler)

	t.Run("IsValidEventType", func(t *testing.T) {
		tests := []struct {
			eventType string
			expected  bool
		}{
			{"payment.succeeded", true},
			{"payment.failed", true},
			{"subscription.created", true},
			{"invalid.event", false},
			{"", false},
		}

		for _, tt := range tests {
			result := validator.IsValidEventType(tt.eventType)
			if result != tt.expected {
				t.Errorf("IsValidEventType(%q) = %v, expected %v", tt.eventType, result, tt.expected)
			}
		}
	})

	t.Run("GetValidEventTypes", func(t *testing.T) {
		eventTypes := validator.GetValidEventTypes()
		expected := []string{"payment.succeeded", "payment.failed", "subscription.created"}

		if len(eventTypes) != len(expected) {
			t.Errorf("expected %d event types, got %d", len(expected), len(eventTypes))
			return
		}

		for i, eventType := range eventTypes {
			if eventType != expected[i] {
				t.Errorf("expected event type %q at index %d, got %q", expected[i], i, eventType)
			}
		}
	})
}

func TestWithEventHandler(t *testing.T) {
	handler := func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) error { return nil }

	option := WithEventHandler[struct{}, PaymentMetadata]("payment.succeeded", handler)

	options := &WebhookHandlerOption{}
	option(options)

	if options.EventHandlers == nil {
		t.Error("expected EventHandlers to be initialized")
		return
	}

	if len(options.EventHandlers) != 1 {
		t.Errorf("expected 1 event handler, got %d", len(options.EventHandlers))
		return
	}

	if _, exists := options.EventHandlers["payment.succeeded"]; !exists {
		t.Error("expected payment.succeeded event handler to be registered")
	}
}

func TestWebhookHandlerFunc_EventTypeValidation(t *testing.T) {
	handler := NewTestWebhookHandler("test-secret")

	t.Run("valid event type", func(t *testing.T) {
		// This should not panic
		_ = WebhookHandlerFunc[TestWebhookRequest](handler, WithEventHandler[struct{}, PaymentMetadata]("payment.succeeded", func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) error {
			return nil
		}))
	})

	t.Run("invalid event type panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid event type")
			}
		}()

		// This should panic
		_ = WebhookHandlerFunc[TestWebhookRequest](handler, WithEventHandler[struct{}, PaymentMetadata]("invalid.event", func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) error {
			return nil
		}))
	})
}

func TestWebhookHandlerFunc_HTTPHandling(t *testing.T) {
	handler := NewTestWebhookHandler("valid-secret")

	// Create event handler
	// Provider payload for payment events is map[string]string, so register handler accordingly
	eventHandler := func(ctx context.Context, payload *map[string]string, metadata *PaymentMetadata) error { return nil }
	httpHandler := WebhookHandlerFunc[TestWebhookRequest](handler, WithEventHandler[map[string]string, PaymentMetadata]("payment.succeeded", eventHandler))

	t.Run("successful webhook processing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"event": "payment", "data": {"id": "pi_123"}}`))
		req.Header.Set("X-Test-Signature", "valid-secret")

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		// Should contain provider success response
		if !strings.Contains(rec.Body.String(), "received") {
			t.Errorf("expected success response, got %s", rec.Body.String())
		}
	})

	t.Run("invalid request parsing", func(t *testing.T) {
		// Request without signature header - this gets parsed but fails signature validation
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"event": "payment"}`))

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		// Missing signature is handled during ParseRequest phase, not request parsing
		// so it returns 401 (unauthorized) rather than 400 (bad request)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"event": "payment"}`))
		req.Header.Set("X-Test-Signature", "wrong-secret")

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("unhandled event type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"event": "unknown"}`))
		req.Header.Set("X-Test-Signature", "valid-secret")

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		// Should return success response even for unhandled events
		if !strings.Contains(rec.Body.String(), "received") {
			t.Errorf("expected success response, got %s", rec.Body.String())
		}
	})

	t.Run("handler returns nil error -> 200 with provider success", func(t *testing.T) {
		handler := NewTestWebhookHandler("valid-secret")

		nilRespHandler := func(ctx context.Context, payload *map[string]string, metadata *PaymentMetadata) error { return nil }
		httpHandler := WebhookHandlerFunc[TestWebhookRequest](handler, WithEventHandler[map[string]string, PaymentMetadata]("payment.succeeded", nilRespHandler))

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"event": "payment", "data": {"id": "pi_123"}}`))
		req.Header.Set("X-Test-Signature", "valid-secret")

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("handler error returns 500", func(t *testing.T) {
		handler := NewTestWebhookHandler("valid-secret")

		errorHandler := func(ctx context.Context, payload *map[string]string, metadata *PaymentMetadata) error {
			return fmt.Errorf("processing failed")
		}
		httpHandler := WebhookHandlerFunc[TestWebhookRequest](handler, WithEventHandler[map[string]string, PaymentMetadata]("payment.succeeded", errorHandler))

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"event": "payment", "data": {"id": "pi_123"}}`))
		req.Header.Set("X-Test-Signature", "valid-secret")

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}
	})
}

func TestValidateEventHandlerSignature(t *testing.T) {
	tests := []struct {
		name      string
		handler   interface{}
		expectErr bool
	}{
		{
			name: "valid handler signature",
			handler: func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) error {
				return nil
			},
			expectErr: false,
		},
		{
			name:      "not a function",
			handler:   "not a function",
			expectErr: true,
		},
		{
			name: "wrong number of parameters",
			handler: func(ctx context.Context, payload interface{}) (interface{}, error) {
				return nil, nil
			},
			expectErr: true,
		},
		{
			name: "wrong number of return values",
			handler: func(ctx context.Context, payload interface{}, metadata *PaymentMetadata) (interface{}, error) {
				return nil, nil
			},
			expectErr: true,
		},
		{
			name: "wrong first parameter type",
			handler: func(notCtx string, payload interface{}, metadata *PaymentMetadata) (interface{}, error) {
				return nil, nil
			},
			expectErr: true,
		},
		{
			name: "wrong second return type",
			handler: func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) (interface{}, string) {
				return nil, ""
			},
			expectErr: true,
		},
		{
			name:      "third parameter not pointer",
			handler:   func(ctx context.Context, payload *struct{}, metadata PaymentMetadata) error { return nil },
			expectErr: true,
		},
		{
			name:      "no return values",
			handler:   func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) {},
			expectErr: true,
		},
		{
			name:      "single return not error",
			handler:   func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) string { return "" },
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerType := reflect.TypeOf(tt.handler)
			err := validateEventHandlerSignature(handlerType)

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
				return
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// For error cases, validate the error message if provided
			if tt.expectErr && err != nil {
				if err.Error() == "" {
					t.Error("expected non-empty error message")
				}
			}
		})
	}
}

// Removed tests for internal helpers that no longer exist
