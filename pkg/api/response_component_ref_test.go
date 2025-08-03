package api

import (
	"reflect"
	"testing"
)

// Test that response types generate component references instead of inline schemas

type TestLoginResponse struct {
	Body struct {
		Token string `gork:"token"`
	}
}

type TestNamedBodyType struct {
	UserID   string `gork:"userID"`
	Username string `gork:"username"`
}

type TestResponseWithNamedBody struct {
	Body TestNamedBodyType
}

func TestResponseGeneratesComponentReference(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Generate schema for LoginResponse
	respType := reflect.TypeOf(TestLoginResponse{})
	schema := generator.generateResponseComponentSchema(respType, components)

	// Should return a reference schema
	if schema.Ref == "" {
		t.Errorf("Expected schema to have a Ref, got: %+v", schema)
	}

	expectedRef := "#/components/schemas/TestLoginResponse"
	if schema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, schema.Ref)
	}

	// Should create the component schema in components
	componentSchema, exists := components.Schemas["TestLoginResponse"]
	if !exists {
		t.Fatal("Expected TestLoginResponse component to be created")
	}

	// Component schema should have the properties from the Body field
	if componentSchema.Properties == nil {
		t.Fatal("Expected component schema to have properties")
	}

	if _, hasToken := componentSchema.Properties["token"]; !hasToken {
		t.Errorf("Expected component schema to have 'token' property. Properties: %+v", componentSchema.Properties)
	}

	// Component schema should NOT have a 'Body' property
	if _, hasBody := componentSchema.Properties["Body"]; hasBody {
		t.Errorf("Component schema should not have 'Body' property. Properties: %+v", componentSchema.Properties)
	}

	// Verify schema metadata
	if componentSchema.Type != "object" {
		t.Errorf("Expected component schema type to be 'object', got %q", componentSchema.Type)
	}

	if componentSchema.Title != "TestLoginResponse" {
		t.Errorf("Expected component schema title to be 'TestLoginResponse', got %q", componentSchema.Title)
	}
}

func TestResponseComponentReuse(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	respType := reflect.TypeOf(TestLoginResponse{})

	// Generate schema twice
	schema1 := generator.generateResponseComponentSchema(respType, components)
	schema2 := generator.generateResponseComponentSchema(respType, components)

	// Both should return the same reference
	if schema1.Ref != schema2.Ref {
		t.Errorf("Expected both schemas to reference the same component")
	}

	// Should only create one component
	if len(components.Schemas) != 1 {
		t.Errorf("Expected only one component schema, got %d", len(components.Schemas))
	}
}

func TestProcessResponseSectionsUsesComponentRef(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	operation := &Operation{
		Responses: make(map[string]*Response),
	}

	respType := reflect.TypeOf((*TestLoginResponse)(nil))
	mockRoute := &RouteInfo{
		Method:      "POST",
		Path:        "/login",
		HandlerName: "LoginHandler",
	}
	generator.processResponseSections(respType, operation, components, mockRoute)

	// Should create a 200 response
	response, exists := operation.Responses["200"]
	if !exists {
		t.Fatal("Expected 200 response to be created")
	}

	// Response should have content
	if response.Content == nil {
		t.Fatal("Expected response to have content")
	}

	jsonContent, exists := response.Content["application/json"]
	if !exists {
		t.Fatal("Expected application/json content")
	}

	// Content schema should be a reference
	if jsonContent.Schema.Ref == "" {
		t.Errorf("Expected response content to use component reference, got: %+v", jsonContent.Schema)
	}

	expectedRef := "#/components/schemas/TestLoginResponse"
	if jsonContent.Schema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, jsonContent.Schema.Ref)
	}

	// Should create the component
	if _, exists := components.Schemas["TestLoginResponse"]; !exists {
		t.Error("Expected TestLoginResponse component to be created")
	}
}

func TestResponseWithNamedBodyReferencesBodyType(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Generate schema for response with named body type
	respType := reflect.TypeOf(TestResponseWithNamedBody{})
	schema := generator.generateResponseComponentSchema(respType, components)

	// Should return a reference schema to the body type directly
	if schema.Ref == "" {
		t.Errorf("Expected schema to have a Ref, got: %+v", schema)
	}

	expectedRef := "#/components/schemas/TestNamedBodyType"
	if schema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, schema.Ref)
	}

	// Should create ONLY the TestNamedBodyType component, not TestResponseWithNamedBody
	if _, hasResponseComponent := components.Schemas["TestResponseWithNamedBody"]; hasResponseComponent {
		t.Errorf("Should not create TestResponseWithNamedBody component when Body is named type. Components: %+v", components.Schemas)
	}

	// Should create the TestNamedBodyType component
	namedBodyComponent, exists := components.Schemas["TestNamedBodyType"]
	if !exists {
		t.Fatal("Expected TestNamedBodyType component to be created")
	}

	// Component schema should have properties from the named body type
	if namedBodyComponent.Properties == nil {
		t.Fatal("Expected TestNamedBodyType component schema to have properties")
	}

	expectedProps := []string{"userID", "username"}
	for _, prop := range expectedProps {
		if _, hasProp := namedBodyComponent.Properties[prop]; !hasProp {
			t.Errorf("Expected TestNamedBodyType component schema to have '%s' property. Properties: %+v", prop, namedBodyComponent.Properties)
		}
	}

	// TestNamedBodyType component should NOT have a 'Body' property
	if _, hasBody := namedBodyComponent.Properties["Body"]; hasBody {
		t.Errorf("TestNamedBodyType component schema should not have 'Body' property. Properties: %+v", namedBodyComponent.Properties)
	}
}

func TestResponseComponentSchemaFallbackToInline(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Create an anonymous struct type (no name)
	anonymousStructType := reflect.StructOf([]reflect.StructField{
		{
			Name: "Body",
			Type: reflect.TypeOf(struct {
				Message string `gork:"message"`
			}{}),
		},
	})

	// Generate schema for anonymous response type
	schema := generator.generateResponseComponentSchema(anonymousStructType, components)

	// Should return an inline schema (no Ref) since typeName is empty
	if schema.Ref != "" {
		t.Errorf("Expected inline schema for anonymous type, got Ref: %q", schema.Ref)
	}

	// Should not create any component schemas since it fell back to inline generation
	if len(components.Schemas) != 0 {
		t.Errorf("Expected no component schemas for anonymous type, got %d schemas: %+v", len(components.Schemas), components.Schemas)
	}

	// The inline schema should have been generated from the Body field
	// For an anonymous struct Body, this should be a component reference to the Body type
	if schema.Ref == "" && schema.Type == "" && schema.Properties == nil {
		t.Error("Expected inline schema to have some content (Ref, Type, or Properties)")
	}
}
