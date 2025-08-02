package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestProcessHandlerResponse_UnknownErrorCase(t *testing.T) {
	// Create a mock handler that returns a non-error type as the second return value
	// This will trigger the "unknown error" case on lines 243-244
	mockHandler := func(ctx context.Context, req struct{}) (*struct{ Message string }, string) {
		// Return a string instead of error - this will make errInterface != nil
		// but the type assertion errInterface.(error) will fail
		return nil, "this is not an error type"
	}

	// Create reflection value of the handler
	handlerValue := reflect.ValueOf(mockHandler)

	// Create a request pointer
	reqPtr := reflect.New(reflect.TypeOf(struct{}{}))

	// Create test HTTP objects
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	// Call the function directly to trigger the unknown error case
	processHandlerResponse(w, r, handlerValue, reqPtr)

	// Verify the response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Due to writeError security filtering, we expect "Internal Server Error" not "unknown error"
	expectedBody := `{"error":"Internal Server Error"}` + "\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
	}
}
