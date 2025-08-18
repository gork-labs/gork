package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test structures for raw body parsing
type StripeWebhookTestRequest struct {
	Headers struct {
		StripeSignature string `gork:"Stripe-Signature" validate:"required"`
		ContentType     string `gork:"Content-Type"`
	}
	Body []byte // Raw body field - Gork should handle this automatically
}

type MixedRequest struct {
	Query struct {
		ID string `gork:"id"`
	}
	Headers struct {
		Authorization string `gork:"Authorization"`
	}
	Body []byte // Raw body
}

func TestParseRequest_RawBodyField(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		body        string
		headers     map[string]string
		expected    []byte
		expectError bool
	}{
		{
			name:     "POST request with raw body",
			method:   http.MethodPost,
			body:     `{"event": "payment.succeeded", "data": {"id": "pi_123"}}`,
			expected: []byte(`{"event": "payment.succeeded", "data": {"id": "pi_123"}}`),
		},
		{
			name:     "PUT request with raw body",
			method:   http.MethodPut,
			body:     `webhook payload content`,
			expected: []byte(`webhook payload content`),
		},
		{
			name:     "GET request with raw body (should work for webhooks)",
			method:   http.MethodGet,
			body:     `some content`,
			expected: []byte(`some content`),
		},
		{
			name:     "empty body",
			method:   http.MethodPost,
			body:     "",
			expected: []byte(``),
		},
		{
			name:     "nil body handled gracefully",
			method:   http.MethodPost,
			body:     "",
			expected: []byte(``),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/webhook", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, "/webhook", nil)
			}

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Parse request
			var webhookReq StripeWebhookTestRequest
			err := ParseRequest(req, &webhookReq)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				if string(webhookReq.Body) != string(tt.expected) {
					t.Errorf("expected body %q, got %q", string(tt.expected), string(webhookReq.Body))
				}
			}
		})
	}
}

func TestParseRequest_MixedWithRawBody(t *testing.T) {
	// Test mixing raw body with other sections
	req := httptest.NewRequest(http.MethodPost, "/test?id=123", strings.NewReader(`raw webhook data`))
	req.Header.Set("Authorization", "Bearer token123")

	var mixedReq MixedRequest
	err := ParseRequest(req, &mixedReq)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Check query parsing
	if mixedReq.Query.ID != "123" {
		t.Errorf("expected Query.ID = '123', got %q", mixedReq.Query.ID)
	}

	// Check header parsing
	if mixedReq.Headers.Authorization != "Bearer token123" {
		t.Errorf("expected Headers.Authorization = 'Bearer token123', got %q", mixedReq.Headers.Authorization)
	}

	// Check raw body parsing
	expectedBody := `raw webhook data`
	if string(mixedReq.Body) != expectedBody {
		t.Errorf("expected Body = %q, got %q", expectedBody, string(mixedReq.Body))
	}
}

func TestParseRequest_WebhookWithValidation(t *testing.T) {
	// Test with validation tags on headers
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "data"}`))
	req.Header.Set("Stripe-Signature", "t=123,v1=signature")
	req.Header.Set("Content-Type", "application/json")

	var webhookReq StripeWebhookTestRequest
	err := ParseRequest(req, &webhookReq)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Check header parsing
	if webhookReq.Headers.StripeSignature != "t=123,v1=signature" {
		t.Errorf("expected StripeSignature header, got %q", webhookReq.Headers.StripeSignature)
	}

	if webhookReq.Headers.ContentType != "application/json" {
		t.Errorf("expected ContentType header, got %q", webhookReq.Headers.ContentType)
	}

	// Check raw body
	expectedBody := `{"test": "data"}`
	if string(webhookReq.Body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(webhookReq.Body))
	}
}

func TestParseRequest_ErrorCases(t *testing.T) {
	t.Run("non-pointer request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		var webhookReq StripeWebhookTestRequest

		err := ParseRequest(req, webhookReq) // Not a pointer

		if err == nil {
			t.Error("expected error for non-pointer request")
		}

		if !strings.Contains(err.Error(), "must be a pointer") {
			t.Errorf("expected pointer error, got %v", err)
		}
	})
}

// Test backward compatibility with existing Body struct parsing
type TraditionalRequest struct {
	Body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
}

func TestParseRequest_BackwardCompatibility(t *testing.T) {
	// Ensure existing struct Body parsing still works
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name": "John", "email": "john@example.com"}`))
	req.Header.Set("Content-Type", "application/json")

	var traditionalReq TraditionalRequest
	err := ParseRequest(req, &traditionalReq)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Check struct body parsing still works
	if traditionalReq.Body.Name != "John" {
		t.Errorf("expected Name = 'John', got %q", traditionalReq.Body.Name)
	}

	if traditionalReq.Body.Email != "john@example.com" {
		t.Errorf("expected Email = 'john@example.com', got %q", traditionalReq.Body.Email)
	}
}

// Test edge cases for raw body parsing
func TestConventionParser_ParseRawBodyField_EdgeCases(t *testing.T) {
	t.Run("nil request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)

		var webhookReq StripeWebhookTestRequest
		err := ParseRequest(req, &webhookReq)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		// Should have empty slice, not nil
		if webhookReq.Body == nil {
			t.Error("expected empty slice, got nil")
		}

		if len(webhookReq.Body) != 0 {
			t.Errorf("expected empty slice, got length %d", len(webhookReq.Body))
		}
	})

	t.Run("large body content", func(t *testing.T) {
		// Test with larger realistic webhook content
		largeWebhookPayload := `{"event": "payment.succeeded", "data": {"id": "pi_` + strings.Repeat("1234567890", 100) + `", "amount": 2000, "currency": "usd"}}`
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(largeWebhookPayload))

		var webhookReq StripeWebhookTestRequest
		err := ParseRequest(req, &webhookReq)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if string(webhookReq.Body) != largeWebhookPayload {
			t.Error("large body content not preserved correctly")
		}
	})
}
