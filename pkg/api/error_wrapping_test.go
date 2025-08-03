package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestProcessRequestParametersErrorWrapping(t *testing.T) {
	// Test that JSON parsing errors are properly wrapped
	t.Run("invalid JSON provides detailed error", func(t *testing.T) {
		type TestRequest struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		// Create a request with invalid JSON
		invalidJSON := `{"name": "test", "age": "not-a-number"}` // String instead of int
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(invalidJSON))
		req.Header.Set("Content-Type", "application/json")

		reqPtr := reflect.New(reflect.TypeOf(TestRequest{}))
		
		// Call the function
		err := processRequestParameters(reqPtr, req, nil)
		
		// Verify error is returned
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		
		// Verify error message contains both the generic message and the underlying error
		errStr := err.Error()
		if !strings.Contains(errStr, "unable to parse request body") {
			t.Errorf("error should contain 'unable to parse request body', got: %s", errStr)
		}
		
		// Verify it contains JSON unmarshal error details
		if !strings.Contains(errStr, "json") || !strings.Contains(errStr, "cannot unmarshal") {
			t.Errorf("error should contain JSON unmarshal details, got: %s", errStr)
		}
	})
	
	t.Run("malformed JSON provides syntax error details", func(t *testing.T) {
		type TestRequest struct {
			Name string `json:"name"`
		}

		// Create a request with malformed JSON
		malformedJSON := `{"name": "test"` // Missing closing brace
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(malformedJSON))
		req.Header.Set("Content-Type", "application/json")

		reqPtr := reflect.New(reflect.TypeOf(TestRequest{}))
		
		// Call the function
		err := processRequestParameters(reqPtr, req, nil)
		
		// Verify error is returned
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
		
		// Verify error message contains syntax error details
		errStr := err.Error()
		if !strings.Contains(errStr, "unable to parse request body") {
			t.Errorf("error should contain 'unable to parse request body', got: %s", errStr)
		}
		
		// Should contain details about unexpected EOF or similar
		if !strings.Contains(errStr, "EOF") && !strings.Contains(errStr, "unexpected") {
			t.Errorf("error should contain JSON syntax error details, got: %s", errStr)
		}
	})
}