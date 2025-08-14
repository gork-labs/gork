package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test webhook request without EventTypeValidator interface
type SimpleWebhookRequest struct {
	Headers struct {
		Signature string `gork:"X-Signature" validate:"required"`
	}
	Body []byte
}

func (SimpleWebhookRequest) WebhookRequest() {}

// Simple webhook handler without EventTypeValidator interface
type SimpleWebhookHandler struct{}

func NewSimpleWebhookHandler() WebhookHandler[SimpleWebhookRequest] { return &SimpleWebhookHandler{} }

func (h *SimpleWebhookHandler) ParseRequest(req SimpleWebhookRequest) (WebhookEvent, error) {
	if req.Headers.Signature == "" {
		return WebhookEvent{}, fmt.Errorf("missing signature")
	}
	return WebhookEvent{Type: "test.event", ProviderObject: map[string]string{"data": "test"}}, nil
}

func (h *SimpleWebhookHandler) SuccessResponse() interface{} {
	return map[string]bool{"received": true}
}

func (h *SimpleWebhookHandler) ErrorResponse(err error) interface{} {
	return map[string]string{"error": err.Error()}
}

func (h *SimpleWebhookHandler) GetValidEventTypes() []string {
	return []string{"test.event", "payment.succeeded"}
}

func (h *SimpleWebhookHandler) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "Simple"}
}

func TestWebhookHandlerFunc_EdgeCases(t *testing.T) {
	t.Run("handler without EventTypeValidator interface", func(t *testing.T) {
		handler := NewSimpleWebhookHandler()

		httpHandler := WebhookHandlerFunc(handler)

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "data"}`))
		req.Header.Set("X-Signature", "test-sig")

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("invalid event type panics", func(t *testing.T) {
		handler := NewSimpleWebhookHandler()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid event type")
			}
		}()

		WebhookHandlerFunc[SimpleWebhookRequest](handler, WithEventHandler[struct{}, PaymentMetadata]("invalid.event", func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) error {
			return nil
		}))
	})

	t.Run("missing signature returns 401 (validated)", func(t *testing.T) {
		handler := NewTestWebhookHandler("test-secret")
		httpHandler := WebhookHandlerFunc(handler)

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "data"}`))

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 for missing signature, got %d", rec.Code)
		}
	})

	t.Run("signature verification failure returns 401", func(t *testing.T) {
		handler := NewTestWebhookHandler("correct-secret")
		httpHandler := WebhookHandlerFunc(handler)

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "data"}`))
		req.Header.Set("X-Test-Signature", "wrong-secret")

		rec := httptest.NewRecorder()
		httpHandler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 for signature verification failure, got %d", rec.Code)
		}
	})
}

// Removed duplicate validateEventHandlerSignature error-case tests; kept single table-driven version in webhook_test.go

// Test non-function type for validateEventHandlerSignature
// Removed duplicate non-function signature test; covered in table-driven tests
