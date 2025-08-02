package api

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

// Test types for registry export testing
type ExportTestRequest struct {
	ID   string `json:"id" openapi:"name=id,in=path"`
	Name string `json:"name" validate:"required"`
}

type ExportTestResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func exportTestHandler(ctx context.Context, req ExportTestRequest) (ExportTestResponse, error) {
	return ExportTestResponse{
		Message: "Hello " + req.Name,
		Success: true,
	}, nil
}

func TestRouteRegistry_Export(t *testing.T) {
	registry := NewRouteRegistry()

	// Register test routes
	route1 := &RouteInfo{
		Method:       "GET",
		Path:         "/users/{id}",
		Handler:      exportTestHandler,
		HandlerName:  "exportTestHandler",
		RequestType:  reflect.TypeOf(ExportTestRequest{}),
		ResponseType: reflect.TypeOf(ExportTestResponse{}),
		Options:      &HandlerOption{Tags: []string{"users"}},
	}

	route2 := &RouteInfo{
		Method:       "POST",
		Path:         "/users",
		Handler:      exportTestHandler,
		HandlerName:  "exportTestHandler",
		RequestType:  reflect.TypeOf(ExportTestRequest{}),
		ResponseType: reflect.TypeOf(ExportTestResponse{}),
		Options:      &HandlerOption{Tags: []string{"users", "create"}},
	}

	registry.Register(route1)
	registry.Register(route2)

	// Export the registry
	data, err := registry.Export()
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	// Parse the exported JSON
	var exportedRoutes []ExportableRouteInfo
	if err := json.Unmarshal(data, &exportedRoutes); err != nil {
		t.Fatalf("Failed to unmarshal exported data: %v", err)
	}

	// Verify the exported data
	if len(exportedRoutes) != 2 {
		t.Errorf("Expected 2 exported routes, got %d", len(exportedRoutes))
	}

	// Check first route
	route := exportedRoutes[0]
	if route.Method != "GET" {
		t.Errorf("Route 0 Method = %s, want GET", route.Method)
	}
	if route.Path != "/users/{id}" {
		t.Errorf("Route 0 Path = %s, want /users/{id}", route.Path)
	}
	if route.HandlerName != "exportTestHandler" {
		t.Errorf("Route 0 HandlerName = %s, want exportTestHandler", route.HandlerName)
	}
	if route.RequestType != "api.ExportTestRequest" {
		t.Errorf("Route 0 RequestType = %s, want api.ExportTestRequest", route.RequestType)
	}
	if route.ResponseType != "api.ExportTestResponse" {
		t.Errorf("Route 0 ResponseType = %s, want api.ExportTestResponse", route.ResponseType)
	}

	// Check second route
	route = exportedRoutes[1]
	if route.Method != "POST" {
		t.Errorf("Route 1 Method = %s, want POST", route.Method)
	}
	if route.Path != "/users" {
		t.Errorf("Route 1 Path = %s, want /users", route.Path)
	}
}

func TestRouteRegistry_Export_EmptyRegistry(t *testing.T) {
	registry := NewRouteRegistry()

	data, err := registry.Export()
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var exportedRoutes []ExportableRouteInfo
	if err := json.Unmarshal(data, &exportedRoutes); err != nil {
		t.Fatalf("Failed to unmarshal exported data: %v", err)
	}

	if len(exportedRoutes) != 0 {
		t.Errorf("Expected 0 exported routes from empty registry, got %d", len(exportedRoutes))
	}
}

func TestRouteRegistry_Export_WithNilTypes(t *testing.T) {
	registry := NewRouteRegistry()

	// Register route with nil types
	route := &RouteInfo{
		Method:       "GET",
		Path:         "/health",
		Handler:      nil,
		HandlerName:  "healthHandler",
		RequestType:  nil,
		ResponseType: nil,
		Options:      &HandlerOption{},
	}

	registry.Register(route)

	data, err := registry.Export()
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var exportedRoutes []ExportableRouteInfo
	if err := json.Unmarshal(data, &exportedRoutes); err != nil {
		t.Fatalf("Failed to unmarshal exported data: %v", err)
	}

	if len(exportedRoutes) != 1 {
		t.Errorf("Expected 1 exported route, got %d", len(exportedRoutes))
	}

	exportedRoute := exportedRoutes[0]
	if exportedRoute.RequestType != "" {
		t.Errorf("RequestType = %s, want empty string for nil type", exportedRoute.RequestType)
	}
	if exportedRoute.ResponseType != "" {
		t.Errorf("ResponseType = %s, want empty string for nil type", exportedRoute.ResponseType)
	}
}

func TestRouteRegistry_Export_ConcurrentAccess(t *testing.T) {
	registry := NewRouteRegistry()

	// Register initial route
	route := &RouteInfo{
		Method:       "GET",
		Path:         "/test",
		Handler:      exportTestHandler,
		HandlerName:  "exportTestHandler",
		RequestType:  reflect.TypeOf(ExportTestRequest{}),
		ResponseType: reflect.TypeOf(ExportTestResponse{}),
		Options:      &HandlerOption{},
	}
	registry.Register(route)

	// Test concurrent export and registration
	done := make(chan bool, 2)

	// Goroutine 1: Export
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			_, err := registry.Export()
			if err != nil {
				t.Errorf("Export failed during concurrent access: %v", err)
			}
		}
	}()

	// Goroutine 2: Register more routes
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			newRoute := &RouteInfo{
				Method:       "POST",
				Path:         "/test",
				Handler:      exportTestHandler,
				HandlerName:  "exportTestHandler",
				RequestType:  reflect.TypeOf(ExportTestRequest{}),
				ResponseType: reflect.TypeOf(ExportTestResponse{}),
				Options:      &HandlerOption{},
			}
			registry.Register(newRoute)
		}
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}

func TestRouteRegistry_Export_JSONStructure(t *testing.T) {
	registry := NewRouteRegistry()

	route := &RouteInfo{
		Method:       "GET",
		Path:         "/api/users/{id}",
		Handler:      exportTestHandler,
		HandlerName:  "getUserHandler",
		RequestType:  reflect.TypeOf(ExportTestRequest{}),
		ResponseType: reflect.TypeOf(ExportTestResponse{}),
		Options:      &HandlerOption{Tags: []string{"users", "api"}},
	}
	registry.Register(route)

	data, err := registry.Export()
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	// Verify JSON structure by parsing into generic interface
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify it's an array
	routes, ok := jsonData.([]interface{})
	if !ok {
		t.Fatal("Exported data should be a JSON array")
	}

	if len(routes) != 1 {
		t.Fatalf("Expected 1 route in JSON, got %d", len(routes))
	}

	// Verify route structure
	routeObj, ok := routes[0].(map[string]interface{})
	if !ok {
		t.Fatal("Route should be a JSON object")
	}

	expectedFields := []string{"method", "path", "handlerName", "requestType", "responseType"}
	for _, field := range expectedFields {
		if _, exists := routeObj[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}
}

func TestExportableRouteInfo_JSONTags(t *testing.T) {
	// Test that ExportableRouteInfo marshals correctly
	info := ExportableRouteInfo{
		Method:       "GET",
		Path:         "/test",
		HandlerName:  "testHandler",
		RequestType:  "TestRequest",
		ResponseType: "TestResponse",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal ExportableRouteInfo: %v", err)
	}

	// Parse back to verify field names
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	expectedFields := map[string]string{
		"method":       "GET",
		"path":         "/test",
		"handlerName":  "testHandler",
		"requestType":  "TestRequest",
		"responseType": "TestResponse",
	}

	for field, expectedValue := range expectedFields {
		if value, exists := parsed[field]; !exists {
			t.Errorf("Missing field: %s", field)
		} else if value != expectedValue {
			t.Errorf("Field %s = %v, want %v", field, value, expectedValue)
		}
	}
}

func TestExportableRouteInfo_OmitEmpty(t *testing.T) {
	// Test omitempty behavior with empty RequestType and ResponseType
	info := ExportableRouteInfo{
		Method:      "GET",
		Path:        "/test",
		HandlerName: "testHandler",
		// RequestType and ResponseType are empty, should be omitted
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal ExportableRouteInfo: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// These fields should be omitted when empty
	omittedFields := []string{"requestType", "responseType"}
	for _, field := range omittedFields {
		if _, exists := parsed[field]; exists {
			t.Errorf("Field %s should be omitted when empty", field)
		}
	}

	// These fields should always be present
	requiredFields := []string{"method", "path", "handlerName"}
	for _, field := range requiredFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("Required field %s is missing", field)
		}
	}
}

// Benchmark tests for export performance
func BenchmarkRouteRegistry_Export(b *testing.B) {
	registry := NewRouteRegistry()

	// Register multiple routes for benchmarking
	for i := 0; i < 100; i++ {
		route := &RouteInfo{
			Method:       "GET",
			Path:         "/test",
			Handler:      exportTestHandler,
			HandlerName:  "exportTestHandler",
			RequestType:  reflect.TypeOf(ExportTestRequest{}),
			ResponseType: reflect.TypeOf(ExportTestResponse{}),
			Options:      &HandlerOption{},
		}
		registry.Register(route)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.Export()
		if err != nil {
			b.Fatalf("Export failed: %v", err)
		}
	}
}

func BenchmarkRouteRegistry_Export_EmptyRegistry(b *testing.B) {
	registry := NewRouteRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.Export()
		if err != nil {
			b.Fatalf("Export failed: %v", err)
		}
	}
}

func TestRouteRegistry_Register_NilRoute(t *testing.T) {
	registry := NewRouteRegistry()

	initialCount := len(registry.GetRoutes())

	// Register nil route - should be ignored
	registry.Register(nil)

	finalCount := len(registry.GetRoutes())

	if finalCount != initialCount {
		t.Errorf("Expected route count to remain %d after registering nil route, got %d", initialCount, finalCount)
	}
}
