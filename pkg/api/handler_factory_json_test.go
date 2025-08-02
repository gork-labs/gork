package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// Mock failing JSON encoder for testing
type failingJSONEncoder struct{}

func (f failingJSONEncoder) Encode(v interface{}) error {
	return errors.New("mock JSON encoding error")
}

// Mock failing JSON encoder factory
type failingJSONEncoderFactory struct{}

func (f failingJSONEncoderFactory) NewEncoder(w io.Writer) JSONEncoder {
	return failingJSONEncoder{}
}

// TestProcessHandlerResponseWithFactory_JSONEncodingError tests the JSON encoding error path
func TestProcessHandlerResponseWithFactory_JSONEncodingError(t *testing.T) {
	// Create a test handler that returns a successful response
	handler := func(ctx context.Context, req struct{}) (*struct{ Message string }, error) {
		return &struct{ Message string }{Message: "success"}, nil
	}
	
	// Create HTTP test recorder
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	
	// Get handler value and create request pointer
	handlerValue := reflect.ValueOf(handler)
	reqType := reflect.TypeOf(struct{}{})
	reqPtr := reflect.New(reqType)
	
	// Use failing encoder factory
	failingFactory := failingJSONEncoderFactory{}
	
	// Call the function with failing factory
	processHandlerResponseWithFactory(w, r, handlerValue, reqPtr, failingFactory)
	
	// Should return 500 error due to JSON encoding failure
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got: %d", w.Code)
	}
	
	// Check error message (should be generic for security, not the specific message)
	body := w.Body.String()
	expected := `{"error":"Internal Server Error"}` + "\n"
	if body != expected {
		t.Errorf("Expected %q, got %q", expected, body)
	}
}

// TestProcessHandlerResponse_NormalPath tests the normal wrapper function
func TestProcessHandlerResponse_NormalPath(t *testing.T) {
	// Create a test handler that returns a successful response
	handler := func(ctx context.Context, req struct{}) (*struct{ Message string }, error) {
		return &struct{ Message string }{Message: "success"}, nil
	}
	
	// Create HTTP test recorder
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	
	// Get handler value and create request pointer
	handlerValue := reflect.ValueOf(handler)
	reqType := reflect.TypeOf(struct{}{})
	reqPtr := reflect.New(reqType)
	
	// Call the normal function (uses default factory)
	processHandlerResponse(w, r, handlerValue, reqPtr)
	
	// Should succeed
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
	}
	
	// Check response body
	body := w.Body.String()
	expected := `{"Message":"success"}` + "\n"
	if body != expected {
		t.Errorf("Expected %q, got: %q", expected, body)
	}
}