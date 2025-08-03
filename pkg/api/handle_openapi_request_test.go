package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// failingJSONEncoder is a mock encoder that always fails
type failingJSONEncoder struct{}

func (f *failingJSONEncoder) Encode(v any) error {
	return errors.New("simulated JSON encoding failure")
}

func TestHandleOpenAPIRequest(t *testing.T) {
	t.Run("successful OpenAPI spec generation and encoding", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
		}

		// Create a static spec for testing
		staticSpec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
		}

		req, err := http.NewRequest("GET", "/openapi.json", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.handleOpenAPIRequest(rr, req, staticSpec)

		// Should return 200 OK
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Should have correct content type
		expectedContentType := "application/json"
		if ct := rr.Header().Get("Content-Type"); ct != expectedContentType {
			t.Errorf("handler returned wrong content type: got %v want %v", ct, expectedContentType)
		}

		// Should return valid JSON with OpenAPI spec
		var spec OpenAPISpec
		if err := json.Unmarshal(rr.Body.Bytes(), &spec); err != nil {
			t.Errorf("handler returned invalid JSON: %v", err)
		}

		if spec.OpenAPI != "3.1.0" {
			t.Errorf("expected OpenAPI version 3.1.0, got %v", spec.OpenAPI)
		}

		if spec.Info.Title != "Test API" {
			t.Errorf("expected title 'Test API', got %v", spec.Info.Title)
		}
	})

	t.Run("nil spec returns error", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
		}

		req, err := http.NewRequest("GET", "/openapi.json", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.handleOpenAPIRequest(rr, req, nil)

		// Should return 500 Internal Server Error
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}

		// Should return error message
		expectedBody := `{"error":"Failed to generate OpenAPI spec"}`
		if body := strings.TrimSpace(rr.Body.String()); body != expectedBody {
			t.Errorf("handler returned wrong body: got %v want %v", body, expectedBody)
		}
	})

	t.Run("JSON encoding failure returns error", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
		}

		// Create a static spec for testing
		staticSpec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
		}

		// Use the dependency injection method with a failing encoder
		rr := httptest.NewRecorder()
		failingEncoder := &failingJSONEncoder{}

		router.handleOpenAPIRequestWithEncoder(rr, staticSpec, failingEncoder)

		// Should return 500 Internal Server Error
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}

		// Should return JSON encoding error message
		expectedBody := `{"error":"Failed to encode OpenAPI spec"}`
		if body := strings.TrimSpace(rr.Body.String()); body != expectedBody {
			t.Errorf("handler returned wrong body: got %v want %v", body, expectedBody)
		}
	})

}
