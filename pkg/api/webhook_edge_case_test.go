package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test edge cases that might hit missing coverage lines

func TestWebhookHandlerFunc_ParseRequestError(t *testing.T) {
	// Try to trigger a real ParseRequest error (not validation error)
	handler := NewTestWebhookHandler("test-secret")
	httpHandler := WebhookHandlerFunc(handler)

	// Create a request with a body that causes an actual parsing error
	req := httptest.NewRequest(http.MethodPost, "/webhook", &errorBodyReader{})
	req.Header.Set("X-Test-Signature", "test-secret")

	rec := httptest.NewRecorder()
	httpHandler(rec, req)

	// This should trigger a parse error, not a validation error
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for parse error, got %d", rec.Code)
	}
}

// errorBodyReader that causes parsing errors
type errorBodyReader struct{}

func (e *errorBodyReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// Removed: invokeEventHandlerWithValidation paths no longer exist after refactor

// Removed
