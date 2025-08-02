package api

import (
	"context"
	"reflect"
	"testing"
)

// Test types for comprehensive OpenAPI generation testing
type ConsolidatedTestReq struct {
	ID        string   `json:"id" openapi:"id,in=path" validate:"required"`
	Name      string   `json:"name" validate:"required,min=2,max=50"`
	Email     string   `json:"email" validate:"email"`
	Age       int      `json:"age" validate:"min=0,max=120"`
	Tags      []string `json:"tags"`
	Active    bool     `json:"active"`
	Rating    float64  `json:"rating" validate:"min=0,max=5"`
	Status    string   `json:"status" validate:"oneof=active inactive pending"`
	QueryParam string  `openapi:"filter,in=query"`
	HeaderVal  string  `openapi:"x-api-key,in=header" validate:"required"`
}

type ConsolidatedTestResp struct {
	Message   string               `json:"message"`
	Success   bool                 `json:"success"`
	Data      *ConsolidatedTestReq `json:"data,omitempty"`
	Timestamp int64                `json:"timestamp"`
}

// Test handler function for OpenAPI generation
func consolidatedTestHandler(ctx context.Context, req ConsolidatedTestReq) (ConsolidatedTestResp, error) {
	return ConsolidatedTestResp{Message: "test", Success: true}, nil
}

// TestGenerateOpenAPIComprehensive consolidates OpenAPI generation tests
func TestGenerateOpenAPIComprehensive(t *testing.T) {
	t.Run("basic generation", func(t *testing.T) {
		registry := NewRouteRegistry()
		
		// Register a test route
		registry.Register(&RouteInfo{
			Method:       "GET",
			Path:         "/test/{id}",
			Handler:      consolidatedTestHandler,
			HandlerName:  "consolidatedTestHandler",
			RequestType:  reflect.TypeOf(ConsolidatedTestReq{}),
			ResponseType: reflect.TypeOf(ConsolidatedTestResp{}),
			Options:      &HandlerOption{},
		})
		
		spec := GenerateOpenAPI(registry)
		
		// Verify basic structure
		if spec == nil {
			t.Fatal("Generated spec should not be nil")
		}
		
		if spec.OpenAPI == "" {
			t.Error("OpenAPI version should be set")
		}
		
		if spec.Paths == nil {
			t.Fatal("Paths should not be nil")
		}
		
		// Verify the test path exists
		pathItem, exists := spec.Paths["/test/{id}"]
		if !exists {
			t.Error("Expected path '/test/{id}' not found in spec")
		}
		
		if pathItem == nil || pathItem.Get == nil {
			t.Error("Expected GET operation not found")
		}
	})

	t.Run("with options", func(t *testing.T) {
		registry := NewRouteRegistry()
		
		registry.Register(&RouteInfo{
			Method:      "POST",
			Path:        "/users",
			Handler:     consolidatedTestHandler,
			HandlerName: "consolidatedTestHandler",
			RequestType: reflect.TypeOf(ConsolidatedTestReq{}),
			ResponseType: reflect.TypeOf(ConsolidatedTestResp{}),
			Options: &HandlerOption{
				Tags: []string{"users", "api"},
				Security: []SecurityRequirement{
					{Type: "bearer", Scopes: []string{"read", "write"}},
				},
			},
		})
		
		spec := GenerateOpenAPI(registry, 
			WithTitle("Test API"),
			WithVersion("1.0.0"),
		)
		
		// Verify metadata
		if spec.Info.Title != "Test API" {
			t.Errorf("Title: got %q, want %q", spec.Info.Title, "Test API")
		}
		
		if spec.Info.Version != "1.0.0" {
			t.Errorf("Version: got %q, want %q", spec.Info.Version, "1.0.0")
		}
		
		// Verify operation has tags
		pathItem := spec.Paths["/users"]
		if pathItem == nil || pathItem.Post == nil {
			t.Fatal("Expected POST operation not found")
		}
		
		if len(pathItem.Post.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(pathItem.Post.Tags))
		}
	})
}

// TestSchemaGenerationComprehensive consolidates schema generation tests
func TestSchemaGenerationComprehensive(t *testing.T) {
	t.Run("basic types schema", func(t *testing.T) {
		registry := make(map[string]*Schema)
		
		tests := []struct {
			name     string
			typ      reflect.Type
			expected string
		}{
			{"string", reflect.TypeOf(""), "string"},
			{"int", reflect.TypeOf(0), "integer"},
			{"bool", reflect.TypeOf(true), "boolean"},
			{"float64", reflect.TypeOf(0.0), "number"},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				schema := reflectTypeToSchema(tt.typ, registry)
				if schema.Type != tt.expected {
					t.Errorf("Type: got %q, want %q", schema.Type, tt.expected)
				}
			})
		}
	})

	t.Run("struct schema", func(t *testing.T) {
		registry := make(map[string]*Schema)
		structType := reflect.TypeOf(ConsolidatedTestReq{})
		
		schema := reflectTypeToSchema(structType, registry)
		
		// Named structs return references, not inline objects
		if schema.Ref == "" {
			t.Error("Named struct should return a reference schema")
		}
		
		// The actual schema should be registered in the registry
		schemaName := sanitizeSchemaName(structType.Name())
		actualSchema, exists := registry[schemaName]
		if !exists {
			t.Fatalf("Schema %q should be registered in registry", schemaName)
		}
		
		if actualSchema.Type != "object" {
			t.Errorf("Registered schema type: got %q, want %q", actualSchema.Type, "object")
		}
		
		if actualSchema.Properties == nil {
			t.Fatal("Registered schema should have properties")
		}
		
		// Check required fields on the actual schema
		if len(actualSchema.Required) == 0 {
			t.Error("Registered schema should have required fields")
		}
		
		// Check specific properties (note: id is not in properties because it's a path parameter)
		if nameProp, ok := actualSchema.Properties["name"]; !ok {
			t.Error("Missing 'name' property in registered schema")
		} else if nameProp.Type != "string" {
			t.Errorf("Name property type: got %q, want %q", nameProp.Type, "string")
		}
	})

	t.Run("array schema", func(t *testing.T) {
		registry := make(map[string]*Schema)
		sliceType := reflect.TypeOf([]string{})
		
		schema := reflectTypeToSchema(sliceType, registry)
		
		if schema.Type != "array" {
			t.Errorf("Array schema type: got %q, want %q", schema.Type, "array")
		}
		
		if schema.Items == nil {
			t.Fatal("Array schema should have items")
		}
		
		if schema.Items.Type != "string" {
			t.Errorf("Array items type: got %q, want %q", schema.Items.Type, "string")
		}
	})
}

// TestValidationConstraintsComprehensive consolidates validation constraint tests
func TestValidationConstraintsComprehensive(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		registry := make(map[string]*Schema)
		structType := reflect.TypeOf(ConsolidatedTestReq{})
		
		reflectTypeToSchema(structType, registry)
		
		// Get the actual registered schema
		schemaName := sanitizeSchemaName(structType.Name())
		actualSchema := registry[schemaName]
		
		// Should have required fields based on validate:"required" tags
		requiredFields := make(map[string]bool)
		for _, req := range actualSchema.Required {
			requiredFields[req] = true
		}
		
		// Note: ID field is not in required because it's a path parameter, not a body property
		if !requiredFields["name"] {
			t.Error("Name field should be required")
		}
	})

	t.Run("validation constraints", func(t *testing.T) {
		registry := make(map[string]*Schema)
		structType := reflect.TypeOf(ConsolidatedTestReq{})
		
		reflectTypeToSchema(structType, registry)
		
		// Get the actual registered schema
		schemaName := sanitizeSchemaName(structType.Name())
		actualSchema := registry[schemaName]
		
		// Check min/max constraints on age field
		if ageProp, ok := actualSchema.Properties["age"]; ok {
			if ageProp.Minimum == nil || *ageProp.Minimum != 0 {
				t.Error("Age field should have minimum constraint of 0")
			}
			if ageProp.Maximum == nil || *ageProp.Maximum != 120 {
				t.Error("Age field should have maximum constraint of 120")
			}
		} else {
			t.Error("Age property not found")
		}
		
		// Check oneof constraint on status field
		if statusProp, ok := actualSchema.Properties["status"]; ok {
			if len(statusProp.Enum) != 3 {
				t.Errorf("Status field should have 3 enum values, got %d", len(statusProp.Enum))
			}
		} else {
			t.Error("Status property not found")
		}
	})
}

// TestParameterExtractionComprehensive consolidates parameter extraction tests
func TestParameterExtractionComprehensive(t *testing.T) {
	registry := make(map[string]*Schema)
	structType := reflect.TypeOf(ConsolidatedTestReq{})
	
	params := extractParameters(structType, registry)
	
	// Should extract path, query, and header parameters
	paramsByLocation := make(map[string][]*Parameter)
	for i := range params {
		param := &params[i]
		paramsByLocation[param.In] = append(paramsByLocation[param.In], param)
	}
	
	t.Run("path parameters", func(t *testing.T) {
		pathParams := paramsByLocation["path"]
		if len(pathParams) == 0 {
			t.Error("Should have at least one path parameter")
		}
		
		// Check for ID path parameter
		found := false
		for _, param := range pathParams {
			if param.Name == "id" {
				found = true
				if !param.Required {
					t.Error("Path parameter should be required")
				}
				break
			}
		}
		if !found {
			t.Error("ID path parameter not found")
		}
	})
	
	t.Run("query parameters", func(t *testing.T) {
		queryParams := paramsByLocation["query"]
		if len(queryParams) == 0 {
			t.Error("Should have at least one query parameter")
		}
		
		// Check for filter query parameter
		found := false
		for _, param := range queryParams {
			if param.Name == "filter" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Filter query parameter not found")
		}
	})
	
	t.Run("header parameters", func(t *testing.T) {
		headerParams := paramsByLocation["header"]
		if len(headerParams) == 0 {
			t.Error("Should have at least one header parameter")
		}
		
		// Check for API key header parameter
		found := false
		for _, param := range headerParams {
			if param.Name == "x-api-key" {
				found = true
				if !param.Required {
					t.Error("Header parameter should be required")
				}
				break
			}
		}
		if !found {
			t.Error("X-API-Key header parameter not found")
		}
	})
}

// TestUtilityFunctionsComprehensive consolidates utility function tests
func TestUtilityFunctionsComprehensive(t *testing.T) {
	t.Run("normalizePath", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"/users/{id}", "/users/{id}"},
			{"/users/:id", "/users/:id"}, // normalizePath currently just returns input as-is
			{"/users/*/profile", "/users/*/profile"}, // normalizePath currently just returns input as-is
			{"/api/v1/users/{id}", "/api/v1/users/{id}"},
		}
		
		for _, tt := range tests {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})
	
	t.Run("extractPathVars", func(t *testing.T) {
		tests := []struct {
			path     string
			expected []string
		}{
			{"/users/{id}", []string{"id"}},
			{"/users/{id}/posts/{postId}", []string{"id", "postId"}},
			{"/static/path", []string{}},
			{"/users/{userId}/comments/{commentId}/replies/{replyId}", []string{"userId", "commentId", "replyId"}},
		}
		
		for _, tt := range tests {
			result := extractPathVars(tt.path)
			if len(result) != len(tt.expected) {
				t.Errorf("extractPathVars(%q) length = %d, want %d", tt.path, len(result), len(tt.expected))
				continue
			}
			
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("extractPathVars(%q)[%d] = %q, want %q", tt.path, i, result[i], expected)
				}
			}
		}
	})
}