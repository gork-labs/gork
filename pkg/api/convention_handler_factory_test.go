package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gork-labs/gork/pkg/gorkson"
)

// MockParameterAdapter for testing
type MockParameterAdapter struct {
	pathParams  map[string]string
	queryParams map[string]string
	headers     map[string]string
	cookies     map[string]string
}

func (m *MockParameterAdapter) Path(req *http.Request, name string) (string, bool) {
	if m.pathParams == nil {
		return "", false
	}
	val, ok := m.pathParams[name]
	return val, ok
}

func (m *MockParameterAdapter) Query(req *http.Request, name string) (string, bool) {
	if m.queryParams == nil {
		return "", false
	}
	val, ok := m.queryParams[name]
	return val, ok
}

func (m *MockParameterAdapter) Header(req *http.Request, name string) (string, bool) {
	if m.headers == nil {
		return "", false
	}
	val, ok := m.headers[name]
	return val, ok
}

func (m *MockParameterAdapter) Cookie(req *http.Request, name string) (string, bool) {
	if m.cookies == nil {
		return "", false
	}
	val, ok := m.cookies[name]
	return val, ok
}

// Test request and response types for convention handler factory
type TestConventionHandlerRequest struct {
	Path struct {
		UserID string `gork:"user_id" validate:"required"`
	}
	Query struct {
		Include string `gork:"include"`
	}
	Body struct {
		Name string `gork:"name" validate:"required"`
	}
	Headers struct {
		Authorization string `gork:"Authorization"`
	}
}

type TestConventionHandlerResponse struct {
	Body struct {
		ID      string    `gork:"id"`
		Name    string    `gork:"name"`
		Created time.Time `gork:"created"`
	}
	Headers struct {
		Location string `gork:"Location"`
	}
	Cookies struct {
		SessionID string `gork:"session_id"`
	}
}

func SampleConventionHandler(ctx context.Context, req TestConventionHandlerRequest) (*TestConventionHandlerResponse, error) {
	return &TestConventionHandlerResponse{
		Body: struct {
			ID      string    `gork:"id"`
			Name    string    `gork:"name"`
			Created time.Time `gork:"created"`
		}{
			ID:      req.Path.UserID,
			Name:    req.Body.Name,
			Created: time.Now(),
		},
		Headers: struct {
			Location string `gork:"Location"`
		}{
			Location: "/users/" + req.Path.UserID,
		},
		Cookies: struct {
			SessionID string `gork:"session_id"`
		}{
			SessionID: "session-123",
		},
	}, nil
}

func TestNewConventionHandlerFactory(t *testing.T) {
	factory := NewConventionHandlerFactory()
	if factory == nil {
		t.Fatal("NewConventionHandlerFactory() returned nil")
	}
	if factory.parser == nil {
		t.Error("Factory parser is nil")
	}
	if factory.validator == nil {
		t.Error("Factory validator is nil")
	}
}

func TestConventionHandlerFactory_RegisterTypeParser(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Test registering a type parser
	err := factory.RegisterTypeParser(func(ctx context.Context, value string) (*time.Time, error) {
		t, err := time.Parse(time.RFC3339, value)
		return &t, err
	})
	if err != nil {
		t.Errorf("RegisterTypeParser() error = %v", err)
	}
}

func TestConventionHandlerFactory_CreateHandler(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a mock adapter
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			"user_id": "user-123",
		},
		queryParams: map[string]string{
			"include": "profile",
		},
		headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	}

	// Create handler
	handler, routeInfo := factory.CreateHandler(adapter, SampleConventionHandler)

	if handler == nil {
		t.Fatal("CreateHandler() returned nil handler")
	}
	if routeInfo == nil {
		t.Fatal("CreateHandler() returned nil routeInfo")
	}
	if routeInfo.HandlerName != "SampleConventionHandler" {
		t.Errorf("HandlerName = %v, want SampleConventionHandler", routeInfo.HandlerName)
	}
}

func TestConventionHandlerFactory_ExecuteHandler(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a mock adapter
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			"user_id": "user-123",
		},
		queryParams: map[string]string{
			"include": "profile",
		},
		headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	}

	// Create handler
	handler, _ := factory.CreateHandler(adapter, SampleConventionHandler)

	// Create test request
	reqBody := `{"name":"John Doe"}`
	req := httptest.NewRequest(http.MethodPost, "/users/user-123?include=profile", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")

	// Execute handler
	rr := httptest.NewRecorder()
	handler(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check response body (should use gork tag names)
	if response["id"] != "user-123" {
		t.Errorf("Response id = %v, want user-123", response["id"])
	}
	if response["name"] != "John Doe" {
		t.Errorf("Response name = %v, want John Doe", response["name"])
	}

	// Check response headers
	if location := rr.Header().Get("Location"); location != "/users/user-123" {
		t.Errorf("Location header = %v, want /users/user-123", location)
	}

	// Check response cookies
	cookies := rr.Result().Cookies()
	foundSessionCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "session_id" && cookie.Value == "session-123" {
			foundSessionCookie = true
			break
		}
	}
	if !foundSessionCookie {
		t.Error("Expected session_id cookie not found")
	}
}

func TestConventionHandlerFactory_ValidationError(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a mock adapter with missing required path parameter
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			// Missing user_id - should cause validation error
		},
		queryParams: map[string]string{},
		headers:     map[string]string{},
	}

	// Create handler
	handler, _ := factory.CreateHandler(adapter, SampleConventionHandler)

	// Create test request with missing required body field
	reqBody := `{}` // Missing required "name" field
	req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute handler
	rr := httptest.NewRecorder()
	handler(rr, req)

	// Check response
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var errorResponse ValidationErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errorResponse); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorResponse.Message != "Validation failed" {
		t.Errorf("Error message = %v, want 'Validation failed'", errorResponse.Message)
	}

	if len(errorResponse.Details) == 0 {
		t.Error("Expected validation error details")
	}
}

// Test handler with convention response using Body section
func SampleSimpleConventionHandler(ctx context.Context, req TestConventionHandlerRequest) (*TestSimpleConventionResponse, error) {
	return &TestSimpleConventionResponse{
		Body: struct {
			Message string `gork:"message"`
			UserID  string `gork:"user_id"`
		}{
			Message: "success",
			UserID:  req.Path.UserID,
		},
	}, nil
}

type TestSimpleConventionResponse struct {
	Body struct {
		Message string `gork:"message"`
		UserID  string `gork:"user_id"`
	}
}

func TestConventionHandlerFactory_SimpleResponse(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a mock adapter
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			"user_id": "user-456",
		},
		headers: map[string]string{},
	}

	// Create handler
	handler, _ := factory.CreateHandler(adapter, SampleSimpleConventionHandler)

	// Create test request
	reqBody := `{"name":"Jane Doe"}`
	req := httptest.NewRequest(http.MethodPost, "/users/user-456", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute handler
	rr := httptest.NewRecorder()
	handler(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "success" {
		t.Errorf("Response message = %v, want success", response["message"])
	}
	if response["user_id"] != "user-456" {
		t.Errorf("Response user_id = %v, want user-456", response["user_id"])
	}
}

// Test error handler
func SampleErrorConventionHandler(ctx context.Context, req TestConventionHandlerRequest) (*TestConventionHandlerResponse, error) {
	return nil, &TestError{Message: "test error"}
}

type TestError struct {
	Message string
}

func (e *TestError) Error() string {
	return e.Message
}

func TestConventionHandlerFactory_HandlerError(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a mock adapter
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			"user_id": "user-789",
		},
	}

	// Create handler
	handler, _ := factory.CreateHandler(adapter, SampleErrorConventionHandler)

	// Create test request
	reqBody := `{"name":"Error Test"}`
	req := httptest.NewRequest(http.MethodPost, "/users/user-789", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute handler
	rr := httptest.NewRecorder()
	handler(rr, req)

	// Check response
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}

	var errorResponse map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errorResponse); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorResponse["error"] != "Internal Server Error" {
		t.Errorf("Error message = %v, want 'Internal Server Error'", errorResponse["error"])
	}
}

func TestConventionHandlerFactory_GetStringValue(t *testing.T) {
	factory := NewConventionHandlerFactory()

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", int(65), "65"},
		{"int8", int8(66), "66"},
		{"int16", int16(67), "67"},
		{"int32", int32(68), "68"},
		{"int64", int64(69), "69"},
		{"uint", uint(70), "70"},
		{"uint8", uint8(71), "71"},
		{"uint16", uint16(72), "72"},
		{"uint32", uint32(73), "73"},
		{"uint64", uint64(74), "74"},
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
		{"float32", float32(3.5), "3.5"},
		{"float64", float64(2.5), "2.5"},
		{"struct", struct{ Name string }{"test"}, `{"Name":"test"}`},
		{"slice", []string{"a", "b"}, `["a","b"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := factory.getStringValue(reflect.ValueOf(tt.value))
			if result != tt.expected {
				t.Errorf("getStringValue(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestConventionHandlerFactory_GetStringValue_JSONMarshalError(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a value that will fail JSON marshaling
	ch := make(chan int)
	result := factory.getStringValue(reflect.ValueOf(ch))

	// Channels can't be marshaled to JSON, so should return empty string
	if result != "" {
		t.Errorf("getStringValue(channel) = %q, want empty string", result)
	}
}

func TestConventionHandlerFactory_IsSimpleKind(t *testing.T) {
	factory := NewConventionHandlerFactory()

	simpleKinds := []reflect.Kind{
		reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Bool, reflect.Float32, reflect.Float64,
	}

	complexKinds := []reflect.Kind{
		reflect.Slice, reflect.Map, reflect.Struct, reflect.Interface, reflect.Chan,
		reflect.Func, reflect.Ptr, reflect.Array,
	}

	for _, kind := range simpleKinds {
		if !factory.isSimpleKind(kind) {
			t.Errorf("isSimpleKind(%v) = false, want true", kind)
		}
	}

	for _, kind := range complexKinds {
		if factory.isSimpleKind(kind) {
			t.Errorf("isSimpleKind(%v) = true, want false", kind)
		}
	}
}

func TestConventionHandlerFactory_GetStringValueForKind(t *testing.T) {
	factory := NewConventionHandlerFactory()

	tests := []struct {
		name     string
		kind     reflect.Kind
		value    interface{}
		expected string
	}{
		{"string", reflect.String, "hello", "hello"},
		{"string_empty", reflect.String, "", ""},
		{"int", reflect.Int, int(65), "65"},
		{"bool_true", reflect.Bool, true, "true"},
		{"bool_false", reflect.Bool, false, "false"},
		{"float32", reflect.Float32, float32(3.5), "3.5"},
		{"unsupported_kind", reflect.Slice, []string{"a"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := factory.getStringValueForKind(tt.kind, reflect.ValueOf(tt.value))
			if result != tt.expected {
				t.Errorf("getStringValueForKind(%v, %v) = %q, want %q", tt.kind, tt.value, result, tt.expected)
			}
		})
	}
}

func TestConventionHandlerFactory_HandleValidationError_NonValidationError(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a non-validation error
	testError := &TestError{Message: "server error"}

	rr := httptest.NewRecorder()
	factory.handleValidationError(rr, testError)

	// Should return 500 for non-validation errors
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestConventionHandlerFactory_ProcessConventionResponse_NilError(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a handler that returns nil error but non-nil interface
	handlerValue := reflect.ValueOf(func(ctx context.Context, req TestConventionHandlerRequest) (*TestSimpleConventionResponse, error) {
		return &TestSimpleConventionResponse{
			Body: struct {
				Message string `gork:"message"`
				UserID  string `gork:"user_id"`
			}{
				Message: "success",
				UserID:  "test",
			},
		}, nil
	})

	reqPtr := reflect.ValueOf(&TestConventionHandlerRequest{
		Body: struct {
			Name string `gork:"name" validate:"required"`
		}{Name: "test"},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)

	factory.processConventionResponse(rr, req, handlerValue, reqPtr)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestConventionHandlerFactory_ProcessConventionResponse_UnknownError(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a handler that returns a non-error interface
	handlerValue := reflect.ValueOf(func(ctx context.Context, req TestConventionHandlerRequest) (*TestSimpleConventionResponse, interface{}) {
		return nil, "unknown error type" // Return string instead of error
	})

	reqPtr := reflect.ValueOf(&TestConventionHandlerRequest{
		Body: struct {
			Name string `gork:"name" validate:"required"`
		}{Name: "test"},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)

	factory.processConventionResponse(rr, req, handlerValue, reqPtr)

	// Should return 500 for unknown error type
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestConventionHandlerFactory_ProcessResponseSections_NilPointer(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Create a nil pointer response
	var nilResponse *TestSimpleConventionResponse
	respVal := reflect.ValueOf(nilResponse)

	rr := httptest.NewRecorder()
	factory.processResponseSections(rr, respVal)

	// Should return 204 No Content for nil pointer
	if rr.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

// Test response type without sections (should fall back to encoding entire response)
type TestNonConventionResponse struct {
	Message string `gork:"message"`
	Status  string `gork:"status"`
}

func TestConventionHandlerFactory_ProcessResponseSections_NoSections(t *testing.T) {
	factory := NewConventionHandlerFactory()

	response := TestNonConventionResponse{
		Message: "success",
		Status:  "ok",
	}
	respVal := reflect.ValueOf(response)

	rr := httptest.NewRecorder()
	factory.processResponseSections(rr, respVal)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	// Should not have content type for 204 responses
	if contentType := rr.Header().Get("Content-Type"); contentType != "" {
		t.Errorf("Content-Type = %v, want empty for 204 response", contentType)
	}
}

func TestConventionHandlerFactory_SetResponseHeaders(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Test with struct containing headers with different types
	headersValue := reflect.ValueOf(struct {
		StringHeader string  `gork:"X-String-Header"`
		IntHeader    int     `gork:"X-Int-Header"`
		BoolHeader   bool    `gork:"X-Bool-Header"`
		FloatHeader  float32 `gork:"X-Float-Header"`
		NoTagHeader  string
		EmptyTag     string `gork:""`
	}{
		StringHeader: "test-value",
		IntHeader:    42,
		BoolHeader:   true,
		FloatHeader:  3.14,
		NoTagHeader:  "ignored",
		EmptyTag:     "ignored",
	})

	rr := httptest.NewRecorder()
	factory.setResponseHeaders(rr, headersValue)

	// Check that headers were set correctly
	if header := rr.Header().Get("X-String-Header"); header != "test-value" {
		t.Errorf("X-String-Header = %v, want test-value", header)
	}

	// Note: Int/Float/Bool headers will be converted to rune characters
	// This tests the type conversion branches in getStringValue
	if header := rr.Header().Get("X-Int-Header"); header == "" {
		t.Error("X-Int-Header should not be empty")
	}

	// Fields without gork tag should be ignored
	if header := rr.Header().Get("NoTagHeader"); header != "" {
		t.Error("Header without gork tag should be ignored")
	}
}

func TestConventionHandlerFactory_SetResponseHeaders_NonStruct(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Test with non-struct value (should return early)
	nonStructValue := reflect.ValueOf("not a struct")

	rr := httptest.NewRecorder()
	factory.setResponseHeaders(rr, nonStructValue)

	// Should not have set any headers
	if len(rr.Header()) > 0 {
		t.Error("Should not set headers for non-struct value")
	}
}

func TestConventionHandlerFactory_SetResponseCookies(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Test with struct containing cookies
	cookiesValue := reflect.ValueOf(struct {
		SessionID  string `gork:"session_id"`
		UserToken  string `gork:"user_token"`
		EmptyValue string `gork:"empty_cookie"` // Empty value should be ignored
		NoTag      string // No gork tag - should be ignored
	}{
		SessionID:  "session-123",
		UserToken:  "token-456",
		EmptyValue: "",
		NoTag:      "ignored",
	})

	rr := httptest.NewRecorder()
	factory.setResponseCookies(rr, cookiesValue)

	// Check that cookies were set correctly
	cookies := rr.Result().Cookies()
	cookieMap := make(map[string]string)
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
	}

	if value, exists := cookieMap["session_id"]; !exists || value != "session-123" {
		t.Errorf("session_id cookie = %v, want session-123", value)
	}

	if value, exists := cookieMap["user_token"]; !exists || value != "token-456" {
		t.Errorf("user_token cookie = %v, want token-456", value)
	}

	// Empty value cookie should not be set
	if _, exists := cookieMap["empty_cookie"]; exists {
		t.Error("Empty value cookie should not be set")
	}

	// No tag cookie should not be set
	if _, exists := cookieMap["NoTag"]; exists {
		t.Error("Cookie without gork tag should not be set")
	}
}

func TestConventionHandlerFactory_SetResponseCookies_NonStruct(t *testing.T) {
	factory := NewConventionHandlerFactory()

	// Test with non-struct value (should return early)
	nonStructValue := reflect.ValueOf("not a struct")

	rr := httptest.NewRecorder()
	factory.setResponseCookies(rr, nonStructValue)

	// Should not have set any cookies
	cookies := rr.Result().Cookies()
	if len(cookies) > 0 {
		t.Error("Should not set cookies for non-struct value")
	}
}

// Test processResponseSections edge cases for better coverage
func TestConventionHandlerFactory_ProcessResponseSections_EdgeCases(t *testing.T) {
	factory := NewConventionHandlerFactory()

	t.Run("nil pointer response", func(t *testing.T) {
		w := httptest.NewRecorder()
		var nilResponse *TestConventionHandlerResponse

		factory.processResponseSections(w, reflect.ValueOf(nilResponse))

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("non-struct response", func(t *testing.T) {
		w := httptest.NewRecorder()
		stringResponse := "plain string response"

		factory.processResponseSections(w, reflect.ValueOf(stringResponse))

		// Should return 204 No Content since validation prevents non-struct responses at registration
		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("OpenAPISpec response uses standard JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		spec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
		}

		factory.processResponseSections(w, reflect.ValueOf(spec))

		// Should use standard JSON marshaling, not gork JSON
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected content-type application/json, got %s", w.Header().Get("Content-Type"))
		}

		// Should contain the OpenAPI spec data
		body := w.Body.String()
		if !strings.Contains(body, "3.1.0") || !strings.Contains(body, "Test API") {
			t.Errorf("Expected OpenAPI spec in response body, got %s", body)
		}
	})

	t.Run("response with Headers and Cookies sections", func(t *testing.T) {
		type ResponseWithSections struct {
			Headers struct {
				CustomHeader string `gork:"X-Custom-Header"`
			}
			Cookies struct {
				SessionCookie string `gork:"session"`
			}
			Body struct {
				Message string `gork:"message"`
			}
		}

		w := httptest.NewRecorder()
		response := ResponseWithSections{
			Headers: struct {
				CustomHeader string `gork:"X-Custom-Header"`
			}{
				CustomHeader: "custom-value",
			},
			Cookies: struct {
				SessionCookie string `gork:"session"`
			}{
				SessionCookie: "session-123",
			},
			Body: struct {
				Message string `gork:"message"`
			}{
				Message: "test message",
			},
		}

		factory.processResponseSections(w, reflect.ValueOf(response))

		// Check headers
		if w.Header().Get("X-Custom-Header") != "custom-value" {
			t.Errorf("Expected X-Custom-Header to be 'custom-value', got %s", w.Header().Get("X-Custom-Header"))
		}

		// Check cookies
		cookies := w.Result().Cookies()
		found := false
		for _, cookie := range cookies {
			if cookie.Name == "session" && cookie.Value == "session-123" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected session cookie with value 'session-123' not found")
		}

		// Check body
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected content-type application/json, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("struct response without sections", func(t *testing.T) {
		type PlainStructResponse struct {
			Name  string `gork:"name"`
			Value int    `gork:"value"`
		}

		w := httptest.NewRecorder()
		response := PlainStructResponse{
			Name:  "test",
			Value: 42,
		}

		factory.processResponseSections(w, reflect.ValueOf(response))

		// Should return 204 No Content for struct without Body field
		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		// Should not have content type for 204 responses
		if contentType := w.Header().Get("Content-Type"); contentType != "" {
			t.Errorf("Expected no content type for 204 response, got %s", contentType)
		}
	})
}

// Test marshal error scenarios using dependency injection
func TestConventionHandlerFactory_MarshalErrors(t *testing.T) {
	errorMarshaler := func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}

	// Mock adapter that provides path parameter
	mockAdapter := &MockParameterAdapter{
		pathParams: map[string]string{"user_id": "123"},
	}

	// Create a request with valid body data to pass validation
	reqBody := `{"name": "Test User", "email": "test@example.com"}`

	t.Run("gork marshal error with body section", func(t *testing.T) {
		factory := &ConventionHandlerFactory{
			parser:        NewConventionParser(),
			validator:     NewConventionValidator(),
			gorkMarshaler: errorMarshaler,
			stdMarshaler:  json.Marshal,
		}

		handler := func(ctx context.Context, req TestConventionHandlerRequest) (*TestConventionHandlerResponse, error) {
			return &TestConventionHandlerResponse{
				Body: struct {
					ID      string    `gork:"id"`
					Name    string    `gork:"name"`
					Created time.Time `gork:"created"`
				}{
					ID:      "123",
					Name:    "Test User",
					Created: time.Now(),
				},
			}, nil
		}

		httpHandler, _ := factory.CreateHandler(mockAdapter, handler)

		req := httptest.NewRequest("POST", "/users/123", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		httpHandler(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Internal Server Error") {
			t.Errorf("Expected internal server error message, got: %s", body)
		}
	})

	t.Run("non-conventional struct response returns 204", func(t *testing.T) {
		factory := &ConventionHandlerFactory{
			parser:        NewConventionParser(),
			validator:     NewConventionValidator(),
			gorkMarshaler: errorMarshaler,
			stdMarshaler:  json.Marshal,
		}

		handler := func(ctx context.Context, req TestConventionHandlerRequest) (*struct {
			ID   string `gork:"id"`
			Name string `gork:"name"`
		}, error,
		) {
			return &struct {
				ID   string `gork:"id"`
				Name string `gork:"name"`
			}{
				ID:   "123",
				Name: "Test User",
			}, nil
		}

		httpHandler, _ := factory.CreateHandler(mockAdapter, handler)

		req := httptest.NewRequest("POST", "/users/123", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		httpHandler(w, req)

		// Non-conventional struct without json.Marshaler should return 204
		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("standard marshal error for OpenAPISpec", func(t *testing.T) {
		factory := &ConventionHandlerFactory{
			parser:        NewConventionParser(),
			validator:     NewConventionValidator(),
			gorkMarshaler: gorkson.Marshal,
			stdMarshaler:  errorMarshaler,
		}

		handler := func(ctx context.Context, req TestConventionHandlerRequest) (*OpenAPISpec, error) {
			return &OpenAPISpec{
				OpenAPI: "3.1.0",
				Info: Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			}, nil
		}

		httpHandler, _ := factory.CreateHandler(mockAdapter, handler)

		req := httptest.NewRequest("POST", "/users/123", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		httpHandler(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Internal Server Error") {
			t.Errorf("Expected internal server error message, got: %s", body)
		}
	})
}
