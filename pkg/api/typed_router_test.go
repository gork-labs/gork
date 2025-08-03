package api

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

// Test types for TypedRouter testing
type TestRouterRequest struct {
	Path struct {
		ID string `gork:"id"`
	}
	Body struct {
		Name string `gork:"name" validate:"required"`
	}
}

type TestRouterGetRequest struct {
	Path struct {
		ID string `gork:"id"`
	}
	Query struct {
		Filter string `gork:"filter"`
	}
}

type TestRouterResponse struct {
	Body struct {
		Message string `gork:"message"`
		Success bool   `gork:"success"`
	}
}

func testRouterHandler(ctx context.Context, req TestRouterRequest) (*TestRouterResponse, error) {
	return &TestRouterResponse{
		Body: struct {
			Message string `gork:"message"`
			Success bool   `gork:"success"`
		}{
			Message: "Hello " + req.Body.Name,
			Success: true,
		},
	}, nil
}

func testRouterGetHandler(ctx context.Context, req TestRouterGetRequest) (*TestRouterResponse, error) {
	return &TestRouterResponse{
		Body: struct {
			Message string `gork:"message"`
			Success bool   `gork:"success"`
		}{
			Message: "Hello " + req.Path.ID,
			Success: true,
		},
	}, nil
}

// Mock parameter adapter for testing
type mockTypedRouterAdapter struct {
	pathParams  map[string]string
	queryParams map[string]string
}

func (m *mockTypedRouterAdapter) Path(r *http.Request, key string) (string, bool) {
	if m.pathParams == nil {
		return "", false
	}
	val, ok := m.pathParams[key]
	return val, ok
}

func (m *mockTypedRouterAdapter) Query(r *http.Request, key string) (string, bool) {
	if m.queryParams == nil {
		return "", false
	}
	val, ok := m.queryParams[key]
	return val, ok
}

func (m *mockTypedRouterAdapter) Header(r *http.Request, key string) (string, bool) {
	return r.Header.Get(key), true
}

func (m *mockTypedRouterAdapter) Cookie(r *http.Request, key string) (string, bool) {
	cookie, err := r.Cookie(key)
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}

// Mock register function to capture registration calls
type registrationCall struct {
	method string
	path   string
	info   *RouteInfo
}

type mockRegisterFn struct {
	calls []registrationCall
}

func (m *mockRegisterFn) register(method, path string, handler http.HandlerFunc, info *RouteInfo) {
	m.calls = append(m.calls, registrationCall{
		method: method,
		path:   path,
		info:   info,
	})
}

func TestNewTypedRouter(t *testing.T) {
	registry := NewRouteRegistry()
	adapter := &mockTypedRouterAdapter{}
	middleware := []Option{WithTags("test")}
	registerFn := func(method, path string, handler http.HandlerFunc, info *RouteInfo) {}

	router := NewTypedRouter("test-underlying", registry, "/api", middleware, adapter, registerFn)

	if router.underlying != "test-underlying" {
		t.Errorf("underlying = %v, want test-underlying", router.underlying)
	}

	if router.registry != registry {
		t.Error("registry not set correctly")
	}

	if router.prefix != "/api" {
		t.Errorf("prefix = %s, want /api", router.prefix)
	}

	if len(router.middleware) != 1 {
		t.Errorf("middleware length = %d, want 1", len(router.middleware))
	}

	if router.adapter != adapter {
		t.Error("adapter not set correctly")
	}
}

func TestTypedRouter_GetRegistry(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*string](
		nil,
		registry,
		"",
		nil,
		&mockTypedRouterAdapter{},
		func(method, path string, handler http.HandlerFunc, info *RouteInfo) {},
	)

	if router.GetRegistry() != registry {
		t.Error("GetRegistry() returned different registry")
	}
}

func TestTypedRouter_Unwrap(t *testing.T) {
	underlying := "test-value"
	router := NewTypedRouter(
		underlying,
		NewRouteRegistry(),
		"",
		nil,
		&mockTypedRouterAdapter{},
		func(method, path string, handler http.HandlerFunc, info *RouteInfo) {},
	)

	if router.Unwrap() != underlying {
		t.Errorf("Unwrap() = %v, want %v", router.Unwrap(), underlying)
	}
}

func TestTypedRouter_HTTPMethods(t *testing.T) {
	registry := NewRouteRegistry()
	mockRegister := &mockRegisterFn{}
	router := NewTypedRouter[*string](
		nil,
		registry,
		"",
		nil,
		&mockTypedRouterAdapter{},
		mockRegister.register,
	)

	tests := []struct {
		name     string
		method   string
		register func(string, interface{}, ...Option)
		handler  interface{}
	}{
		{"GET", "GET", router.Get, testRouterGetHandler},
		{"POST", "POST", router.Post, testRouterHandler},
		{"PUT", "PUT", router.Put, testRouterHandler},
		{"DELETE", "DELETE", router.Delete, testRouterHandler},
		{"PATCH", "PATCH", router.Patch, testRouterHandler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialCalls := len(mockRegister.calls)
			tt.register("/test", tt.handler)

			if len(mockRegister.calls) != initialCalls+1 {
				t.Errorf("Expected %d registration calls, got %d", initialCalls+1, len(mockRegister.calls))
			}

			lastCall := mockRegister.calls[len(mockRegister.calls)-1]
			if lastCall.method != tt.method {
				t.Errorf("Method = %s, want %s", lastCall.method, tt.method)
			}

			if lastCall.path != "/test" {
				t.Errorf("Path = %s, want /test", lastCall.path)
			}

			if lastCall.info == nil {
				t.Error("RouteInfo should not be nil")
			}
		})
	}
}

func TestTypedRouter_HTTPMethodsWithOptions(t *testing.T) {
	registry := NewRouteRegistry()
	mockRegister := &mockRegisterFn{}
	router := NewTypedRouter[*string](
		nil,
		registry,
		"",
		nil,
		&mockTypedRouterAdapter{},
		mockRegister.register,
	)

	// Test with options
	router.Get("/test", testRouterGetHandler, WithTags("api", "test"), WithBasicAuth())

	if len(mockRegister.calls) != 1 {
		t.Fatalf("Expected 1 registration call, got %d", len(mockRegister.calls))
	}

	call := mockRegister.calls[0]
	if call.info.Options == nil {
		t.Fatal("RouteInfo.Options should not be nil")
	}

	if len(call.info.Options.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(call.info.Options.Tags))
	}

	expectedTags := []string{"api", "test"}
	for i, tag := range expectedTags {
		if call.info.Options.Tags[i] != tag {
			t.Errorf("Tag[%d] = %s, want %s", i, call.info.Options.Tags[i], tag)
		}
	}

	if len(call.info.Options.Security) != 1 {
		t.Errorf("Expected 1 security requirement, got %d", len(call.info.Options.Security))
	}

	if call.info.Options.Security[0].Type != "basic" {
		t.Errorf("Security type = %s, want basic", call.info.Options.Security[0].Type)
	}
}

func TestTypedRouter_CopyMiddleware(t *testing.T) {
	middleware := []Option{WithTags("test"), WithBasicAuth()}
	router := NewTypedRouter[*string](
		nil,
		NewRouteRegistry(),
		"",
		middleware,
		&mockTypedRouterAdapter{},
		func(method, path string, handler http.HandlerFunc, info *RouteInfo) {},
	)

	copied := router.CopyMiddleware()

	// Test that length is the same
	if len(copied) != len(middleware) {
		t.Errorf("Copied middleware length = %d, want %d", len(copied), len(middleware))
	}

	// Test that it's a copy, not the same slice
	if &copied == &middleware {
		t.Error("CopyMiddleware should return a different slice reference")
	}

	// Test that modifying the copy doesn't affect the original
	copied = append(copied, WithTags("additional"))
	if len(router.middleware) != len(middleware) {
		t.Error("Modifying copied middleware affected the original")
	}
}

func TestTypedRouter_RegistrationWithPrefix(t *testing.T) {
	registry := NewRouteRegistry()
	mockRegister := &mockRegisterFn{}
	router := NewTypedRouter[*string](
		nil,
		registry,
		"/api/v1",
		nil,
		&mockTypedRouterAdapter{},
		mockRegister.register,
	)

	router.Get("/users", testRouterGetHandler)

	if len(mockRegister.calls) != 1 {
		t.Fatalf("Expected 1 registration call, got %d", len(mockRegister.calls))
	}

	call := mockRegister.calls[0]
	if call.path != "/users" {
		t.Errorf("Registered path = %s, want /users", call.path)
	}

	// Verify that the RouteInfo has the correct prefixed path
	if call.info.Path != "/api/v1/users" {
		t.Errorf("RouteInfo path = %s, want /api/v1/users", call.info.Path)
	}
}

func TestTypedRouter_MiddlewarePropagation(t *testing.T) {
	registry := NewRouteRegistry()
	mockRegister := &mockRegisterFn{}
	middleware := []Option{WithTags("global")}
	router := NewTypedRouter[*string](
		nil,
		registry,
		"",
		middleware,
		&mockTypedRouterAdapter{},
		mockRegister.register,
	)

	router.Get("/test", testRouterGetHandler, WithTags("local"))

	if len(mockRegister.calls) != 1 {
		t.Fatalf("Expected 1 registration call, got %d", len(mockRegister.calls))
	}

	call := mockRegister.calls[0]
	if call.info.Options == nil {
		t.Fatal("RouteInfo.Options should not be nil")
	}

	// Should have both global and local tags
	expectedTags := []string{"global", "local"}
	if len(call.info.Options.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(call.info.Options.Tags))
	}

	for i, tag := range expectedTags {
		if call.info.Options.Tags[i] != tag {
			t.Errorf("Tag[%d] = %s, want %s", i, call.info.Options.Tags[i], tag)
		}
	}
}

func TestTypedRouter_InvalidHandler(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*string](
		nil,
		registry,
		"",
		nil,
		&mockTypedRouterAdapter{},
		func(method, path string, handler http.HandlerFunc, info *RouteInfo) {},
	)

	// Test with invalid handler signature (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid handler signature")
		}
	}()

	invalidHandler := func(req string) string { return req }
	router.Get("/test", invalidHandler)
}

func TestTypedRouter_EmptyPath(t *testing.T) {
	registry := NewRouteRegistry()
	mockRegister := &mockRegisterFn{}
	router := NewTypedRouter[*string](
		nil,
		registry,
		"/api",
		nil,
		&mockTypedRouterAdapter{},
		mockRegister.register,
	)

	router.Get("", testRouterGetHandler)

	if len(mockRegister.calls) != 1 {
		t.Fatalf("Expected 1 registration call, got %d", len(mockRegister.calls))
	}

	call := mockRegister.calls[0]
	if call.path != "" {
		t.Errorf("Registered path = %s, want empty string", call.path)
	}

	// Verify that the RouteInfo has the correct prefixed path
	if call.info.Path != "/api" {
		t.Errorf("RouteInfo path = %s, want /api", call.info.Path)
	}
}

func TestTypedRouter_RouteInfoGeneration(t *testing.T) {
	registry := NewRouteRegistry()
	mockRegister := &mockRegisterFn{}
	router := NewTypedRouter[*string](
		nil,
		registry,
		"",
		nil,
		&mockTypedRouterAdapter{},
		mockRegister.register,
	)

	router.Post("/users", testRouterHandler)

	if len(mockRegister.calls) != 1 {
		t.Fatalf("Expected 1 registration call, got %d", len(mockRegister.calls))
	}

	call := mockRegister.calls[0]
	info := call.info

	if info.HandlerName != "testRouterHandler" {
		t.Errorf("HandlerName = %s, want testRouterHandler", info.HandlerName)
	}

	if info.RequestType != reflect.TypeOf(TestRouterRequest{}) {
		t.Errorf("RequestType = %v, want %v", info.RequestType, reflect.TypeOf(TestRouterRequest{}))
	}

	if info.ResponseType != reflect.TypeOf((*TestRouterResponse)(nil)) {
		t.Errorf("ResponseType = %v, want %v", info.ResponseType, reflect.TypeOf((*TestRouterResponse)(nil)))
	}
}

// Benchmark tests for performance
func BenchmarkTypedRouter_Registration(b *testing.B) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*string](
		nil,
		registry,
		"",
		nil,
		&mockTypedRouterAdapter{},
		func(method, path string, handler http.HandlerFunc, info *RouteInfo) {},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Get("/test", testRouterGetHandler)
	}
}
