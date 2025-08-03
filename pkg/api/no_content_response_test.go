package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Test cases for error-only handlers
func TestErrorOnlyHandler(t *testing.T) {
	handler := func(ctx context.Context, req TestRequest) error {
		return nil
	}

	// Test handler registration and execution - should generate 204
	factory := NewConventionHandlerFactory()
	httpHandler, info := factory.CreateHandler(&HTTPParameterAdapter{}, handler)

	// Verify RouteInfo
	if info.ResponseType != nil {
		t.Errorf("Expected nil ResponseType for error-only handler, got %v", info.ResponseType)
	}

	// Test HTTP execution with valid request body
	reqBody := `{"name": "test"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	httpHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

// Test cases for nil response values
func TestNilConventionalResponse(t *testing.T) {
	handler := func(ctx context.Context, req TestRequest) (*TestConventionalResponse, error) {
		return nil, nil // This should generate 204
	}

	factory := NewConventionHandlerFactory()
	httpHandler, _ := factory.CreateHandler(&HTTPParameterAdapter{}, handler)

	// Test 204 response generation for nil response
	reqBody := `{"name": "test"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	httpHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for nil response, got %d", w.Code)
	}
}

// Test error-only handler that returns an actual error
func TestErrorOnlyHandlerWithError(t *testing.T) {
	handler := func(ctx context.Context, req TestRequest) error {
		return errors.New("handler error")
	}

	factory := NewConventionHandlerFactory()
	httpHandler, info := factory.CreateHandler(&HTTPParameterAdapter{}, handler)

	// Verify RouteInfo
	if info.ResponseType != nil {
		t.Errorf("Expected nil ResponseType for error-only handler, got %v", info.ResponseType)
	}

	// Test HTTP execution with valid request body
	reqBody := `{"name": "test"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	httpHandler(w, req)

	// Should return 500 when error-only handler returns an error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	// Check that generic error message is in response body (actual error is logged, not exposed)
	body := w.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Errorf("Expected 'Internal Server Error' in response body, got: %s", body)
	}
}

// Test conventional response with headers only (absent Body)
func TestConventionalResponseWithHeadersOnly(t *testing.T) {
	type ResponseWithHeaders struct {
		Headers struct {
			CustomHeader string `gork:"X-Custom-Header"`
		}
		// Note: Body is absent, not empty
	}

	handler := func(ctx context.Context, req TestRequest) (*ResponseWithHeaders, error) {
		return &ResponseWithHeaders{
			Headers: struct {
				CustomHeader string `gork:"X-Custom-Header"`
			}{CustomHeader: "test-value"},
		}, nil
	}

	factory := NewConventionHandlerFactory()
	httpHandler, _ := factory.CreateHandler(&HTTPParameterAdapter{}, handler)

	reqBody := `{"name": "test"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	httpHandler(w, req)

	// Test that headers are sent even when Body is absent
	if w.Header().Get("X-Custom-Header") != "test-value" {
		t.Errorf("Expected custom header 'test-value', got '%s'", w.Header().Get("X-Custom-Header"))
	}

	// Should return 204 since no Body field
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for response without Body, got %d", w.Code)
	}
}

// Test conventional response with empty Body
func TestConventionalResponseWithEmptyBody(t *testing.T) {
	type ResponseWithEmptyBody struct {
		Body    struct{} // Empty body - will send {}
		Headers struct {
			CustomHeader string `gork:"X-Custom-Header"`
		}
	}

	handler := func(ctx context.Context, req TestRequest) (*ResponseWithEmptyBody, error) {
		return &ResponseWithEmptyBody{
			Headers: struct {
				CustomHeader string `gork:"X-Custom-Header"`
			}{CustomHeader: "test-value"},
		}, nil
	}

	factory := NewConventionHandlerFactory()
	httpHandler, _ := factory.CreateHandler(&HTTPParameterAdapter{}, handler)

	reqBody := `{"name": "test"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	httpHandler(w, req)

	// Test that headers are sent with empty body content ({})
	if w.Header().Get("X-Custom-Header") != "test-value" {
		t.Errorf("Expected custom header 'test-value', got '%s'", w.Header().Get("X-Custom-Header"))
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected content-type application/json, got %s", w.Header().Get("Content-Type"))
	}

	expectedBody := "{}"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body '{}', got '%s'", w.Body.String())
	}
}

// Test OpenAPI generation for error-only handlers
func TestOpenAPIGenerationForErrorOnlyHandlers(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Create a route info for error-only handler
	route := &RouteInfo{
		Method:       "DELETE",
		Path:         "/users/{id}",
		HandlerName:  "DeleteUser",
		RequestType:  reflect.TypeOf(TestRequest{}),
		ResponseType: nil, // Error-only handler
	}

	// Test OpenAPI generation
	operation := generator.buildConventionOperation(route, spec.Components)

	// Should generate 204 No Content response
	if operation.Responses["204"] == nil {
		t.Error("Expected 204 No Content response in OpenAPI spec for error-only handler")
	}

	if operation.Responses["204"].Description != "No Content" {
		t.Errorf("Expected description 'No Content', got '%s'", operation.Responses["204"].Description)
	}
}

// Test mixed scenarios in route registry
func TestMixedScenariosInRouteRegistry(t *testing.T) {
	// Create different handler types
	errorOnlyHandler := func(ctx context.Context, req TestRequest) error {
		return nil
	}

	conventionalHandler := func(ctx context.Context, req TestRequest) (*TestConventionalResponse, error) {
		return &TestConventionalResponse{}, nil
	}

	factory := NewConventionHandlerFactory()

	// Register both types
	_, errorOnlyInfo := factory.CreateHandler(&HTTPParameterAdapter{}, errorOnlyHandler)
	_, conventionalInfo := factory.CreateHandler(&HTTPParameterAdapter{}, conventionalHandler)

	// Verify route info differences
	if errorOnlyInfo.ResponseType != nil {
		t.Errorf("Expected nil ResponseType for error-only handler, got %v", errorOnlyInfo.ResponseType)
	}

	if conventionalInfo.ResponseType == nil {
		t.Error("Expected non-nil ResponseType for conventional handler")
	}

	if conventionalInfo.ResponseType.String() != "*api.TestConventionalResponse" {
		t.Errorf("Expected *api.TestConventionalResponse, got %v", conventionalInfo.ResponseType)
	}
}

// Test handler signature validation edge cases
func TestHandlerSignatureValidationEdgeCases(t *testing.T) {
	t.Run("value struct response type", func(t *testing.T) {
		// Test that value struct responses are allowed
		handler := func(ctx context.Context, req TestRequest) (TestConventionalResponse, error) {
			return TestConventionalResponse{}, nil
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Value struct response should be allowed: %v", r)
			}
		}()

		validateHandlerSignature(reflect.TypeOf(handler))
	})

	t.Run("pointer struct response type", func(t *testing.T) {
		// Test that pointer struct responses are allowed
		handler := func(ctx context.Context, req TestRequest) (*TestConventionalResponse, error) {
			return &TestConventionalResponse{}, nil
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Pointer struct response should be allowed: %v", r)
			}
		}()

		validateHandlerSignature(reflect.TypeOf(handler))
	})

	t.Run("non-struct response type", func(t *testing.T) {
		// Test that non-struct responses are rejected
		handler := func(ctx context.Context, req TestRequest) (string, error) {
			return "", nil
		}

		defer func() {
			if r := recover(); r == nil {
				t.Error("Non-struct response should be rejected")
			}
		}()

		validateHandlerSignature(reflect.TypeOf(handler))
	})
}

// Helper types for tests
type TestRequest struct {
	Body struct {
		Name string `gork:"name" validate:"required"`
	}
}

type TestConventionalResponse struct {
	Body struct {
		Message string `gork:"message"`
		Success bool   `gork:"success"`
	}
	Headers struct {
		RequestID string `gork:"X-Request-ID"`
	}
	Cookies struct {
		SessionID string `gork:"session_id"`
	}
}
