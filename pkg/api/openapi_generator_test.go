package api

import (
	"context"
	"reflect"
	"testing"
)

// Test types for OpenAPI generation
type GeneratorTestRequest struct {
	ID    string `json:"id" openapi:"name=id,in=path"`
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" openapi:"name=email,in=query"`
	Count int    `json:"count"`
}

type GeneratorTestResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func generatorTestHandler(ctx context.Context, req GeneratorTestRequest) (GeneratorTestResponse, error) {
	return GeneratorTestResponse{Message: "test"}, nil
}

func TestDefaultRouteFilter(t *testing.T) {
	tests := []struct {
		name     string
		route    *RouteInfo
		expected bool
	}{
		{
			name:     "nil route",
			route:    nil,
			expected: true,
		},
		{
			name: "route with nil response type",
			route: &RouteInfo{
				Method:       "GET",
				Path:         "/test",
				ResponseType: nil,
			},
			expected: true,
		},
		{
			name: "normal route",
			route: &RouteInfo{
				Method:       "GET",
				Path:         "/test",
				ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
			},
			expected: true,
		},
		{
			name: "OpenAPISpec pointer route (should be filtered)",
			route: &RouteInfo{
				Method:       "GET",
				Path:         "/openapi.json",
				ResponseType: reflect.TypeOf(&OpenAPISpec{}),
			},
			expected: false,
		},
		{
			name: "OpenAPISpec struct route (should pass - not filtered)",
			route: &RouteInfo{
				Method:       "GET",
				Path:         "/openapi.json",
				ResponseType: reflect.TypeOf(OpenAPISpec{}),
			},
			expected: true, // Non-pointer struct type should pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultRouteFilter(tt.route)
			if result != tt.expected {
				t.Errorf("defaultRouteFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateOpenAPI_Basic(t *testing.T) {
	registry := NewRouteRegistry()

	// Register a test route
	route := &RouteInfo{
		Method:       "GET",
		Path:         "/users/{id}",
		Handler:      generatorTestHandler,
		HandlerName:  "generatorTestHandler",
		RequestType:  reflect.TypeOf(GeneratorTestRequest{}),
		ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
		Options:      &HandlerOption{},
	}
	registry.Register(route)

	spec := GenerateOpenAPI(registry)

	// Test basic spec structure
	if spec.OpenAPI != "3.1.0" {
		t.Errorf("OpenAPI version = %q, want '3.1.0'", spec.OpenAPI)
	}

	if spec.Info.Title != "Generated API" {
		t.Errorf("Info.Title = %q, want 'Generated API'", spec.Info.Title)
	}

	if spec.Info.Version != "0.1.0" {
		t.Errorf("Info.Version = %q, want '0.1.0'", spec.Info.Version)
	}

	// Test paths
	if len(spec.Paths) != 1 {
		t.Errorf("Number of paths = %d, want 1", len(spec.Paths))
	}

	pathItem := spec.Paths["/users/{id}"]
	if pathItem == nil {
		t.Fatal("Path '/users/{id}' not found")
	}

	if pathItem.Get == nil {
		t.Fatal("GET operation not found")
	}

	if pathItem.Get.OperationID != "generatorTestHandler" {
		t.Errorf("OperationID = %q, want 'generatorTestHandler'", pathItem.Get.OperationID)
	}
}

func TestGenerateOpenAPI_WithOptions(t *testing.T) {
	registry := NewRouteRegistry()

	route := &RouteInfo{
		Method:       "POST",
		Path:         "/users",
		Handler:      generatorTestHandler,
		HandlerName:  "generatorTestHandler",
		RequestType:  reflect.TypeOf(GeneratorTestRequest{}),
		ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
		Options: &HandlerOption{
			Tags: []string{"users", "api"},
		},
	}
	registry.Register(route)

	// Test with custom options
	customTitle := "My Custom API"
	customVersion := "2.0.0"

	spec := GenerateOpenAPI(registry,
		WithTitle(customTitle),
		WithVersion(customVersion),
	)

	if spec.Info.Title != customTitle {
		t.Errorf("Info.Title = %q, want %q", spec.Info.Title, customTitle)
	}

	if spec.Info.Version != customVersion {
		t.Errorf("Info.Version = %q, want %q", spec.Info.Version, customVersion)
	}

	// Test that tags are included
	operation := spec.Paths["/users"].Post
	if operation == nil {
		t.Fatal("POST operation not found")
	}

	if len(operation.Tags) != 2 {
		t.Errorf("Number of tags = %d, want 2", len(operation.Tags))
	}

	expectedTags := []string{"users", "api"}
	for i, expectedTag := range expectedTags {
		if operation.Tags[i] != expectedTag {
			t.Errorf("Tag[%d] = %q, want %q", i, operation.Tags[i], expectedTag)
		}
	}
}

func TestGenerateOpenAPI_WithSecurity(t *testing.T) {
	registry := NewRouteRegistry()

	route := &RouteInfo{
		Method:       "GET",
		Path:         "/protected",
		Handler:      generatorTestHandler,
		HandlerName:  "generatorTestHandler",
		RequestType:  reflect.TypeOf(GeneratorTestRequest{}),
		ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
		Options: &HandlerOption{
			Security: []SecurityRequirement{
				{Type: "bearer"},
				{Type: "basic"},
				{Type: "apiKey"},
			},
		},
	}
	registry.Register(route)

	spec := GenerateOpenAPI(registry)

	// Check security schemes were created
	if spec.Components.SecuritySchemes == nil {
		t.Fatal("SecuritySchemes not created")
	}

	expectedSchemes := []string{"BearerAuth", "BasicAuth", "ApiKeyAuth"}
	for _, schemeName := range expectedSchemes {
		if spec.Components.SecuritySchemes[schemeName] == nil {
			t.Errorf("Security scheme %q not found", schemeName)
		}
	}

	// Check security applied to operation
	operation := spec.Paths["/protected"].Get
	if operation == nil {
		t.Fatal("GET operation not found")
	}

	if len(operation.Security) == 0 {
		t.Error("No security requirements found on operation")
	}
}

func TestGenerateOpenAPI_CustomRouteFilter(t *testing.T) {
	registry := NewRouteRegistry()

	// Register multiple routes
	route1 := &RouteInfo{
		Method:       "GET",
		Path:         "/users",
		Handler:      generatorTestHandler,
		HandlerName:  "getUsersHandler",
		RequestType:  reflect.TypeOf(GeneratorTestRequest{}),
		ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
		Options:      &HandlerOption{Tags: []string{"public"}},
	}

	route2 := &RouteInfo{
		Method:       "GET",
		Path:         "/admin",
		Handler:      generatorTestHandler,
		HandlerName:  "getAdminHandler",
		RequestType:  reflect.TypeOf(GeneratorTestRequest{}),
		ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
		Options:      &HandlerOption{Tags: []string{"admin"}},
	}

	registry.Register(route1)
	registry.Register(route2)

	// Custom filter that excludes admin routes
	customFilter := func(info *RouteInfo) bool {
		if info.Options != nil && len(info.Options.Tags) > 0 {
			for _, tag := range info.Options.Tags {
				if tag == "admin" {
					return false
				}
			}
		}
		return true
	}

	spec := GenerateOpenAPI(registry, WithRouteFilter(customFilter))

	// Should only have the public route
	if len(spec.Paths) != 1 {
		t.Errorf("Number of paths = %d, want 1", len(spec.Paths))
	}

	if spec.Paths["/users"] == nil {
		t.Error("Public route not found")
	}

	if spec.Paths["/admin"] != nil {
		t.Error("Admin route should be filtered out")
	}
}

func TestGenerateOpenAPI_HTTPMethods(t *testing.T) {
	registry := NewRouteRegistry()

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		route := &RouteInfo{
			Method:       method,
			Path:         "/test",
			Handler:      generatorTestHandler,
			HandlerName:  "handler" + method,
			RequestType:  reflect.TypeOf(GeneratorTestRequest{}),
			ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
			Options:      &HandlerOption{},
		}
		registry.Register(route)
	}

	spec := GenerateOpenAPI(registry)

	pathItem := spec.Paths["/test"]
	if pathItem == nil {
		t.Fatal("Path '/test' not found")
	}

	// Check that all operations were created
	operations := map[string]*Operation{
		"GET":    pathItem.Get,
		"POST":   pathItem.Post,
		"PUT":    pathItem.Put,
		"PATCH":  pathItem.Patch,
		"DELETE": pathItem.Delete,
	}

	for method, operation := range operations {
		if operation == nil {
			t.Errorf("%s operation not found", method)
		} else if operation.OperationID != "handler"+method {
			t.Errorf("%s operation ID = %q, want %q", method, operation.OperationID, "handler"+method)
		}
	}
}

func TestGenerateOpenAPI_EmptyRegistry(t *testing.T) {
	registry := NewRouteRegistry()

	spec := GenerateOpenAPI(registry)

	// Should still create a valid spec
	if spec.OpenAPI != "3.1.0" {
		t.Errorf("OpenAPI version = %q, want '3.1.0'", spec.OpenAPI)
	}

	if len(spec.Paths) != 0 {
		t.Errorf("Number of paths = %d, want 0", len(spec.Paths))
	}

	if spec.Components == nil {
		t.Error("Components should not be nil")
	}
}

func TestGenerateOpenAPI_SchemaGeneration(t *testing.T) {
	registry := NewRouteRegistry()

	route := &RouteInfo{
		Method:       "POST",
		Path:         "/test",
		Handler:      generatorTestHandler,
		HandlerName:  "testHandler",
		RequestType:  reflect.TypeOf(GeneratorTestRequest{}),
		ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
		Options:      &HandlerOption{},
	}
	registry.Register(route)

	spec := GenerateOpenAPI(registry)

	// Check that schemas were generated
	if spec.Components.Schemas == nil {
		t.Fatal("Schemas not generated")
	}

	// Should have request and response schemas
	requestSchema := spec.Components.Schemas["GeneratorTestRequest"]
	if requestSchema == nil {
		t.Error("Request schema not found")
	}

	responseSchema := spec.Components.Schemas["GeneratorTestResponse"]
	if responseSchema == nil {
		t.Error("Response schema not found")
	}

	// Check basic schema properties
	if requestSchema != nil {
		if requestSchema.Type != "object" {
			t.Errorf("Request schema type = %q, want 'object'", requestSchema.Type)
		}

		if len(requestSchema.Properties) == 0 {
			t.Error("Request schema should have properties")
		}
	}

	if responseSchema != nil {
		if responseSchema.Type != "object" {
			t.Errorf("Response schema type = %q, want 'object'", responseSchema.Type)
		}

		if len(responseSchema.Properties) == 0 {
			t.Error("Response schema should have properties")
		}
	}
}

// Test OpenAPIOption functions
func TestOpenAPIOptions(t *testing.T) {
	t.Run("WithTitle", func(t *testing.T) {
		spec := &OpenAPISpec{Info: Info{}}
		opt := WithTitle("Test Title")
		opt(spec)

		if spec.Info.Title != "Test Title" {
			t.Errorf("Title = %q, want 'Test Title'", spec.Info.Title)
		}
	})

	t.Run("WithVersion", func(t *testing.T) {
		spec := &OpenAPISpec{Info: Info{}}
		opt := WithVersion("1.2.3")
		opt(spec)

		if spec.Info.Version != "1.2.3" {
			t.Errorf("Version = %q, want '1.2.3'", spec.Info.Version)
		}
	})

	t.Run("WithRouteFilter", func(t *testing.T) {
		spec := &OpenAPISpec{}
		filter := func(*RouteInfo) bool { return false }
		opt := WithRouteFilter(filter)
		opt(spec)

		if spec.routeFilter == nil {
			t.Error("RouteFilter was not set")
		}
	})
}

// Test with complex nested types
type ComplexRequest struct {
	User     UserInfo          `json:"user"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
	Settings *SettingsInfo     `json:"settings,omitempty"`
}

type UserInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SettingsInfo struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
}

func complexHandler(ctx context.Context, req ComplexRequest) (GeneratorTestResponse, error) {
	return GeneratorTestResponse{}, nil
}

func TestGenerateOpenAPI_ComplexTypes(t *testing.T) {
	registry := NewRouteRegistry()

	route := &RouteInfo{
		Method:       "POST",
		Path:         "/complex",
		Handler:      complexHandler,
		HandlerName:  "complexHandler",
		RequestType:  reflect.TypeOf(ComplexRequest{}),
		ResponseType: reflect.TypeOf(GeneratorTestResponse{}),
		Options:      &HandlerOption{},
	}
	registry.Register(route)

	spec := GenerateOpenAPI(registry)

	// Check that nested schemas were generated
	schemas := []string{"ComplexRequest", "UserInfo", "SettingsInfo", "GeneratorTestResponse"}
	for _, schemaName := range schemas {
		if spec.Components.Schemas[schemaName] == nil {
			t.Errorf("Schema %q not found", schemaName)
		}
	}

	// Check that the main schema references nested schemas properly
	complexSchema := spec.Components.Schemas["ComplexRequest"]
	if complexSchema == nil {
		t.Fatal("ComplexRequest schema not found")
	}

	// Verify nested object references
	userProp := complexSchema.Properties["user"]
	if userProp == nil {
		t.Error("User property not found in ComplexRequest schema")
	}
}
