package api

import (
	"reflect"
	"testing"
)

// Test that request body sections generate component references instead of inline schemas

type TestCreateUserRequestBody struct {
	Username string `gork:"username" validate:"required"`
	Email    string `gork:"email" validate:"email"`
}

type TestAnonymousRequestBody struct {
	UserID string `gork:"userID" validate:"required"`
}

func TestRequestBodyGeneratesComponentReference(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Generate schema for request body
	bodyType := reflect.TypeOf(TestCreateUserRequestBody{})
	schema := generator.generateRequestBodyComponentSchema(bodyType, nil, components)

	// Should return a reference schema
	if schema.Ref == "" {
		t.Errorf("Expected schema to have a Ref, got: %+v", schema)
	}

	expectedRef := "#/components/schemas/TestCreateUserRequestBodyBody"
	if schema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, schema.Ref)
	}

	// Should create the component schema in components
	componentSchema, exists := components.Schemas["TestCreateUserRequestBodyBody"]
	if !exists {
		t.Fatal("Expected TestCreateUserRequestBodyBody component to be created")
	}

	// Component schema should have the properties from the body struct
	if componentSchema.Properties == nil {
		t.Fatal("Expected component schema to have properties")
	}

	expectedProps := []string{"username", "email"}
	for _, prop := range expectedProps {
		if _, hasProp := componentSchema.Properties[prop]; !hasProp {
			t.Errorf("Expected component schema to have '%s' property. Properties: %+v", prop, componentSchema.Properties)
		}
	}

	// Verify schema metadata
	if componentSchema.Type != "object" {
		t.Errorf("Expected component schema type to be 'object', got %q", componentSchema.Type)
	}

	if componentSchema.Title != "TestCreateUserRequestBodyBody" {
		t.Errorf("Expected component schema title to be 'TestCreateUserRequestBodyBody', got %q", componentSchema.Title)
	}

	// Should have required fields based on validation tags
	if len(componentSchema.Required) == 0 {
		t.Error("Expected component schema to have required fields")
	}
}

func TestAnonymousRequestBodyGeneratesComponentName(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Generate schema for anonymous request body
	bodyType := reflect.TypeOf(TestAnonymousRequestBody{})
	schema := generator.generateRequestBodyComponentSchema(bodyType, nil, components)

	// Should return a reference schema
	if schema.Ref == "" {
		t.Errorf("Expected schema to have a Ref, got: %+v", schema)
	}

	expectedRef := "#/components/schemas/TestAnonymousRequestBodyBody"
	if schema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, schema.Ref)
	}

	// Should create the component
	if _, exists := components.Schemas["TestAnonymousRequestBodyBody"]; !exists {
		t.Error("Expected TestAnonymousRequestBodyBody component to be created")
	}
}

func TestGenerateRequestBodyComponentSchema_ExistingComponent(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Pre-populate the component schema to test the existing component path
	existingComponentName := "TestCreateUserRequestBodyBody"
	existingSchema := &Schema{
		Type:  "object",
		Title: existingComponentName,
		Properties: map[string]*Schema{
			"existing_prop": {Type: "string"},
		},
	}
	components.Schemas[existingComponentName] = existingSchema

	bodyType := reflect.TypeOf(TestCreateUserRequestBody{})
	schema := generator.generateRequestBodyComponentSchema(bodyType, nil, components)

	// Should return a reference to the existing component
	expectedRef := "#/components/schemas/" + existingComponentName
	if schema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, schema.Ref)
	}

	// Should not modify the existing component
	if components.Schemas[existingComponentName] != existingSchema {
		t.Error("Existing component should not be modified")
	}

	// Should only have the pre-existing property
	if len(components.Schemas[existingComponentName].Properties) != 1 {
		t.Errorf("Expected existing component to have 1 property, got %d", len(components.Schemas[existingComponentName].Properties))
	}

	if _, exists := components.Schemas[existingComponentName].Properties["existing_prop"]; !exists {
		t.Error("Expected existing component to preserve 'existing_prop'")
	}

	// Should NOT have the properties that would be extracted from TestCreateUserRequestBody
	if _, exists := components.Schemas[existingComponentName].Properties["username"]; exists {
		t.Error("Should not extract new properties when component already exists")
	}
}

func TestProcessBodySectionUsesComponentRef(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	operation := &Operation{}

	bodyType := reflect.TypeOf(TestCreateUserRequestBody{})
	generator.processBodySection(bodyType, nil, operation, components)

	// Should create a RequestBody
	if operation.RequestBody == nil {
		t.Fatal("Expected RequestBody to be created")
	}

	// RequestBody should have content
	if operation.RequestBody.Content == nil {
		t.Fatal("Expected RequestBody to have content")
	}

	jsonContent, exists := operation.RequestBody.Content["application/json"]
	if !exists {
		t.Fatal("Expected application/json content")
	}

	// Content schema should be a reference
	if jsonContent.Schema.Ref == "" {
		t.Errorf("Expected request body content to use component reference, got: %+v", jsonContent.Schema)
	}

	expectedRef := "#/components/schemas/TestCreateUserRequestBodyBody"
	if jsonContent.Schema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, jsonContent.Schema.Ref)
	}

	// Should create the component
	if _, exists := components.Schemas["TestCreateUserRequestBodyBody"]; !exists {
		t.Error("Expected TestCreateUserRequestBodyBody component to be created")
	}
}

func TestGenerateRequestBodyComponentName_FallbackToEmptyString(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}

	t.Run("anonymous struct with no fields returns empty string", func(t *testing.T) {
		// Create completely empty anonymous struct
		emptyAnonymousStruct := reflect.StructOf([]reflect.StructField{})

		componentName := generator.generateRequestBodyComponentName(emptyAnonymousStruct, nil)

		if componentName != "" {
			t.Errorf("Expected empty string for empty anonymous struct, got: %q", componentName)
		}
	})

	t.Run("named struct with no exported fields uses name with Body suffix", func(t *testing.T) {
		type PrivateFieldsStruct struct {
			privateField string
		}

		namedStructType := reflect.TypeOf(PrivateFieldsStruct{})
		componentName := generator.generateRequestBodyComponentName(namedStructType, nil)

		expected := "PrivateFieldsStructBody"
		if componentName != expected {
			t.Errorf("Expected %q for named struct with private fields, got: %q", expected, componentName)
		}
	})

	t.Run("anonymous struct with exported fields creates field-based name", func(t *testing.T) {
		// Create anonymous struct with exported fields
		anonymousStructWithFields := reflect.StructOf([]reflect.StructField{
			{
				Name: "Username",
				Type: reflect.TypeOf(""),
			},
			{
				Name: "Email",
				Type: reflect.TypeOf(""),
			},
		})

		componentName := generator.generateRequestBodyComponentName(anonymousStructWithFields, nil)

		expected := "UsernameEmailRequest"
		if componentName != expected {
			t.Errorf("Expected %q for anonymous struct with exported fields, got: %q", expected, componentName)
		}
	})
}

func TestRequestBodyComponentSchemaFallbackToInline(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	t.Run("empty anonymous struct falls back to inline schema", func(t *testing.T) {
		// Create completely empty anonymous struct
		emptyAnonymousStruct := reflect.StructOf([]reflect.StructField{})

		schema := generator.generateRequestBodyComponentSchema(emptyAnonymousStruct, nil, components)

		// Should return inline schema (no Ref) since componentName is empty
		if schema.Ref != "" {
			t.Errorf("Expected inline schema for empty anonymous struct, got Ref: %q", schema.Ref)
		}

		// Should not create any component schemas since it fell back to inline generation
		if len(components.Schemas) != 0 {
			t.Errorf("Expected no component schemas for empty anonymous struct, got %d schemas: %+v", len(components.Schemas), components.Schemas)
		}

		// Should have generated an object schema
		if schema.Type != "object" {
			t.Errorf("Expected object type for empty struct inline schema, got: %q", schema.Type)
		}
	})

	t.Run("named struct creates component reference even with no exported fields", func(t *testing.T) {
		type PrivateFieldsStruct struct {
			privateField string
		}

		bodyType := reflect.TypeOf(PrivateFieldsStruct{})
		schema := generator.generateRequestBodyComponentSchema(bodyType, nil, components)

		// Should create a component reference since the struct has a name
		if schema.Ref == "" {
			t.Error("Expected component reference for named struct, got inline schema")
		}

		expectedRef := "#/components/schemas/PrivateFieldsStructBody"
		if schema.Ref != expectedRef {
			t.Errorf("Expected ref %q, got %q", expectedRef, schema.Ref)
		}

		// Should create the component
		if _, exists := components.Schemas["PrivateFieldsStructBody"]; !exists {
			t.Error("Expected PrivateFieldsStructBody component to be created")
		}
	})
}
