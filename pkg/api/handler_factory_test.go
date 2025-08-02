package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Test types for handler factory tests
type TestRequest struct {
	ID          string   `json:"id" openapi:"name=id,in=path"`
	Name        string   `json:"name" validate:"required,min=2"`
	Email       string   `json:"email" openapi:"name=email,in=query"`
	AuthToken   string   `json:"authToken" openapi:"name=Authorization,in=header"`
	SessionID   string   `json:"sessionId" openapi:"name=session_id,in=cookie"`
	Tags        []string `json:"tags"`
	OptionalAge *int     `json:"optionalAge,omitempty"`
}

type TestResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

type TestHandlerErrorResponse struct {
	Error string `json:"error"`
}

// Test handlers
func validHandler(ctx context.Context, req TestRequest) (TestResponse, error) {
	return TestResponse{
		Message: "Hello " + req.Name,
		ID:      req.ID,
		Success: true,
	}, nil
}

func errorHandler(ctx context.Context, req TestRequest) (TestResponse, error) {
	return TestResponse{}, errors.New("handler error")
}

func nilResponseHandler(ctx context.Context, req TestRequest) (*TestResponse, error) {
	return &TestResponse{Message: "Success"}, nil
}

// Mock parameter adapter for testing
type mockParameterAdapter struct {
	pathParams   map[string]string
	queryParams  map[string]string
	headerParams map[string]string
	cookieParams map[string]string
}

func (m *mockParameterAdapter) Path(r *http.Request, key string) (string, bool) {
	if m.pathParams == nil {
		return "", false
	}
	val, ok := m.pathParams[key]
	return val, ok
}

func (m *mockParameterAdapter) Query(r *http.Request, key string) (string, bool) {
	if m.queryParams == nil {
		return "", false
	}
	val, ok := m.queryParams[key]
	return val, ok
}

func (m *mockParameterAdapter) Header(r *http.Request, key string) (string, bool) {
	if m.headerParams == nil {
		return "", false
	}
	val, ok := m.headerParams[key]
	return val, ok
}

func (m *mockParameterAdapter) Cookie(r *http.Request, key string) (string, bool) {
	if m.cookieParams == nil {
		return "", false
	}
	val, ok := m.cookieParams[key]
	return val, ok
}

func TestValidateHandlerSignature(t *testing.T) {
	tests := []struct {
		name        string
		handler     interface{}
		shouldPanic bool
		panicMsg    string
	}{
		{
			name:    "valid handler",
			handler: validHandler,
		},
		{
			name:        "not a function",
			handler:     "not a function",
			shouldPanic: true,
			panicMsg:    "handler must be a function",
		},
		{
			name: "wrong number of parameters",
			handler: func() (TestResponse, error) {
				return TestResponse{}, nil
			},
			shouldPanic: true,
			panicMsg:    "handler must accept exactly 2 parameters",
		},
		{
			name: "wrong first parameter type",
			handler: func(s string, req TestRequest) (TestResponse, error) {
				return TestResponse{}, nil
			},
			shouldPanic: true,
			panicMsg:    "first handler parameter must be context.Context",
		},
		{
			name: "wrong number of return values",
			handler: func(ctx context.Context, req TestRequest) TestResponse {
				return TestResponse{}
			},
			shouldPanic: true,
			panicMsg:    "handler must return (Response, error)",
		},
		{
			name: "wrong second return type",
			handler: func(ctx context.Context, req TestRequest) (TestResponse, string) {
				return TestResponse{}, ""
			},
			shouldPanic: true,
			panicMsg:    "second handler return value must be error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					if r := recover(); r != nil {
						if !strings.Contains(r.(string), tt.panicMsg) {
							t.Errorf("Expected panic message to contain %q, got %q", tt.panicMsg, r)
						}
					} else {
						t.Error("Expected function to panic")
					}
				}()
			}

			v := reflect.ValueOf(tt.handler)
			validateHandlerSignature(v.Type())

			if tt.shouldPanic {
				t.Error("Expected function to panic, but it didn't")
			}
		})
	}
}

func TestBuildRouteInfo(t *testing.T) {
	handler := validHandler
	reqType := reflect.TypeOf(TestRequest{})
	respType := reflect.TypeOf(TestResponse{})

	// Test with no options
	info := buildRouteInfo(handler, reqType, respType, nil)

	// Can't compare function pointers directly, check other fields
	if info.HandlerName != "validHandler" {
		t.Errorf("HandlerName = %q, want validHandler", info.HandlerName)
	}
	if info.RequestType != reqType {
		t.Error("RequestType not set correctly")
	}
	if info.ResponseType != respType {
		t.Error("ResponseType not set correctly")
	}
	if info.Options == nil {
		t.Error("Options should not be nil")
	}

	// Test with options (using available fields)
	testTags := []string{"test", "api"}
	testOpt := func(opts *HandlerOption) {
		opts.Tags = testTags
	}
	info = buildRouteInfo(handler, reqType, respType, []Option{testOpt})

	if len(info.Options.Tags) != len(testTags) {
		t.Errorf("Options.Tags length = %d, want %d", len(info.Options.Tags), len(testTags))
	}
	for i, tag := range testTags {
		if info.Options.Tags[i] != tag {
			t.Errorf("Options.Tags[%d] = %q, want %q", i, info.Options.Tags[i], tag)
		}
	}
}

func TestCreateHandlerFromAny(t *testing.T) {
	adapter := &mockParameterAdapter{
		pathParams:   map[string]string{"id": "123"},
		queryParams:  map[string]string{"email": "test@example.com"},
		headerParams: map[string]string{"Authorization": "Bearer token"},
		cookieParams: map[string]string{"session_id": "abc123"},
	}

	httpHandler, info := createHandlerFromAny(adapter, validHandler)

	// Test RouteInfo
	if info.HandlerName != "validHandler" {
		t.Errorf("HandlerName = %q, want validHandler", info.HandlerName)
	}

	// Test HTTP handler with valid request
	reqBody := TestRequest{
		Name: "Alice",
		Tags: []string{"tag1", "tag2"},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test/123", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	httpHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp TestResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Message != "Hello Alice" {
		t.Errorf("Response.Message = %q, want 'Hello Alice'", resp.Message)
	}
	if resp.ID != "123" {
		t.Errorf("Response.ID = %q, want '123'", resp.ID)
	}
}

func TestProcessRequestParameters(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		body     interface{}
		adapter  GenericParameterAdapter[*http.Request]
		wantErr  bool
		validate func(*testing.T, TestRequest)
	}{
		{
			name:   "POST with valid JSON body",
			method: http.MethodPost,
			body:   TestRequest{Name: "Alice", Tags: []string{"tag1"}},
			adapter: &mockParameterAdapter{
				pathParams: map[string]string{"id": "123"},
			},
			validate: func(t *testing.T, req TestRequest) {
				if req.Name != "Alice" {
					t.Errorf("Name = %q, want Alice", req.Name)
				}
				if req.ID != "123" {
					t.Errorf("ID = %q, want 123", req.ID)
				}
			},
		},
		{
			name:    "POST with invalid JSON",
			method:  http.MethodPost,
			body:    "{invalid json",
			wantErr: true,
		},
		{
			name:   "GET with query parameters",
			method: http.MethodGet,
			adapter: &mockParameterAdapter{
				queryParams: map[string]string{"email": "test@example.com"},
			},
			validate: func(t *testing.T, req TestRequest) {
				if req.Email != "test@example.com" {
					t.Errorf("Email = %q, want test@example.com", req.Email)
				}
			},
		},
		{
			name:   "GET with header parameters",
			method: http.MethodGet,
			adapter: &mockParameterAdapter{
				headerParams: map[string]string{"Authorization": "Bearer token123"},
			},
			validate: func(t *testing.T, req TestRequest) {
				if req.AuthToken != "Bearer token123" {
					t.Errorf("AuthToken = %q, want 'Bearer token123'", req.AuthToken)
				}
			},
		},
		{
			name:   "GET with cookie parameters",
			method: http.MethodGet,
			adapter: &mockParameterAdapter{
				cookieParams: map[string]string{"session_id": "cookie123"},
			},
			validate: func(t *testing.T, req TestRequest) {
				if req.SessionID != "cookie123" {
					t.Errorf("SessionID = %q, want cookie123", req.SessionID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body = []byte(str)
				} else {
					body, _ = json.Marshal(tt.body)
				}
			}

			req := httptest.NewRequest(tt.method, "/test", bytes.NewReader(body))
			reqPtr := reflect.New(reflect.TypeOf(TestRequest{}))

			err := processRequestParameters(reqPtr, req, tt.adapter)

			if (err != nil) != tt.wantErr {
				t.Errorf("processRequestParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil && err == nil {
				tt.validate(t, reqPtr.Elem().Interface().(TestRequest))
			}
		})
	}
}

func TestExecuteHandler_Success(t *testing.T) {
	adapter := &mockParameterAdapter{
		pathParams: map[string]string{"id": "123"},
	}

	reqBody := TestRequest{Name: "Alice"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test/123", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handlerValue := reflect.ValueOf(validHandler)
	reqType := reflect.TypeOf(TestRequest{})

	executeHandler(rr, req, handlerValue, reqType, adapter)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp TestResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Message != "Hello Alice" {
		t.Errorf("Response.Message = %q, want 'Hello Alice'", resp.Message)
	}
}

func TestExecuteHandler_HandlerError(t *testing.T) {
	reqBody := TestRequest{Name: "Alice"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handlerValue := reflect.ValueOf(errorHandler)
	reqType := reflect.TypeOf(TestRequest{})

	executeHandler(rr, req, handlerValue, reqType, nil)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}

	var errResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errResp["error"] != "Internal Server Error" {
		t.Errorf("Error message = %q, want 'Internal Server Error'", errResp["error"])
	}
}

func TestExecuteHandler_ValidationError(t *testing.T) {
	// Request with validation error (name too short)
	reqBody := TestRequest{Name: "A"} // Min length is 2
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handlerValue := reflect.ValueOf(validHandler)
	reqType := reflect.TypeOf(TestRequest{})

	executeHandler(rr, req, handlerValue, reqType, nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var errResp ValidationErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal validation error response: %v", err)
	}

	if errResp.Error != "Validation failed" {
		t.Errorf("Error message = %q, want 'Validation failed'", errResp.Error)
	}

	if len(errResp.Details) == 0 {
		t.Error("Expected validation details")
	}
}

func TestExecuteHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{invalid json"))
	rr := httptest.NewRecorder()

	handlerValue := reflect.ValueOf(validHandler)
	reqType := reflect.TypeOf(TestRequest{})

	executeHandler(rr, req, handlerValue, reqType, nil)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestGetParameterName(t *testing.T) {
	tests := []struct {
		name     string
		tagInfo  struct{ Name, In string }
		field    reflect.StructField
		expected string
	}{
		{
			name:    "explicit name in openapi tag",
			tagInfo: struct{ Name, In string }{Name: "user_id", In: "path"},
			field: reflect.StructField{
				Name: "ID",
				Tag:  reflect.StructTag(`json:"id"`),
			},
			expected: "user_id",
		},
		{
			name:    "fallback to json tag",
			tagInfo: struct{ Name, In string }{Name: "", In: "query"},
			field: reflect.StructField{
				Name: "UserName",
				Tag:  reflect.StructTag(`json:"username"`),
			},
			expected: "username",
		},
		{
			name:    "fallback to field name",
			tagInfo: struct{ Name, In string }{Name: "", In: "header"},
			field: reflect.StructField{
				Name: "AuthToken",
				Tag:  reflect.StructTag(`json:"-"`),
			},
			expected: "AuthToken",
		},
		{
			name:    "fallback to field name when no json tag",
			tagInfo: struct{ Name, In string }{Name: "", In: "cookie"},
			field: reflect.StructField{
				Name: "SessionID",
			},
			expected: "SessionID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getParameterName(tt.tagInfo, tt.field)
			if result != tt.expected {
				t.Errorf("getParameterName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test discriminator validation
type TestDiscriminatorRequest struct {
	Type string `json:"type" openapi:"discriminator=user"`
	Name string `json:"name" validate:"required"`
}

func discriminatorHandler(ctx context.Context, req TestDiscriminatorRequest) (TestResponse, error) {
	return TestResponse{Message: "Success"}, nil
}

func TestValidateRequest_DiscriminatorError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := TestDiscriminatorRequest{Type: "admin", Name: "Alice"} // Wrong discriminator value

	err := validateRequest(rr, req)

	if err == nil {
		t.Error("Expected discriminator validation error")
	}

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var errResp ValidationErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errResp.Error != "Validation failed" {
		t.Errorf("Error message = %q, want 'Validation failed'", errResp.Error)
	}
}

func TestValidateRequest_Success(t *testing.T) {
	rr := httptest.NewRecorder()
	req := TestRequest{Name: "Alice"} // Valid request

	err := validateRequest(rr, req)
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}

	// Check that no content was written and no headers were set
	if rr.Body.Len() != 0 {
		t.Errorf("Unexpected response body written: %s", rr.Body.String())
	}

	if len(rr.Header()) != 0 {
		t.Errorf("Unexpected headers set: %v", rr.Header())
	}
}

func TestProcessHandlerResponse_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handlerValue := reflect.ValueOf(validHandler)
	reqPtr := reflect.New(reflect.TypeOf(TestRequest{}))
	reqPtr.Elem().Set(reflect.ValueOf(TestRequest{Name: "Alice", ID: "123"}))

	processHandlerResponse(rr, req, handlerValue, reqPtr)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var resp TestResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Message != "Hello Alice" {
		t.Errorf("Response.Message = %q, want 'Hello Alice'", resp.Message)
	}
}

func TestProcessHandlerResponse_Error(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handlerValue := reflect.ValueOf(errorHandler)
	reqPtr := reflect.New(reflect.TypeOf(TestRequest{}))
	reqPtr.Elem().Set(reflect.ValueOf(TestRequest{Name: "Alice"}))

	processHandlerResponse(rr, req, handlerValue, reqPtr)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}

	var errResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errResp["error"] != "Internal Server Error" {
		t.Errorf("Error message = %q, want 'Internal Server Error'", errResp["error"])
	}
}
