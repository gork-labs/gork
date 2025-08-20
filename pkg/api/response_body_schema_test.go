package api

import (
	"reflect"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

func TestConventionOpenAPIGenerator_ExtractBodyPropertiesToResponseSchema(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}

	t.Run("union type copies union properties", func(t *testing.T) {
		components := &Components{
			Schemas: make(map[string]*Schema),
		}

		type EmailAuth struct {
			Type  string `gork:"type,discriminator=email"`
			Email string `gork:"email"`
		}

		type TokenAuth struct {
			Type  string `gork:"type,discriminator=token"`
			Token string `gork:"token"`
		}

		unionType := reflect.TypeOf(unions.Union2[EmailAuth, TokenAuth]{})
		responseSchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractBodyPropertiesToResponseSchema(unionType, responseSchema, components)

		// Should have copied union properties (OneOf)
		if len(responseSchema.OneOf) == 0 {
			t.Error("Expected responseSchema to have OneOf from union type")
		}

		// Type should be cleared for oneOf schemas
		if responseSchema.Type != "" {
			t.Errorf("Expected responseSchema.Type to be empty for union, got: %s", responseSchema.Type)
		}
	})

	t.Run("named struct type extracts properties directly", func(t *testing.T) {
		components := &Components{
			Schemas: make(map[string]*Schema),
		}

		type NamedBodyStruct struct {
			ID   string `gork:"id" validate:"required"`
			Name string `gork:"name"`
		}

		bodyType := reflect.TypeOf(NamedBodyStruct{})
		responseSchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractBodyPropertiesToResponseSchema(bodyType, responseSchema, components)

		// Should have extracted properties directly from the struct
		if _, exists := responseSchema.Properties["id"]; !exists {
			t.Error("Expected responseSchema to have 'id' property")
		}
		if _, exists := responseSchema.Properties["name"]; !exists {
			t.Error("Expected responseSchema to have 'name' property")
		}

		// Should have required fields
		if len(responseSchema.Required) == 0 {
			t.Error("Expected responseSchema to have required fields")
		}
	})

	t.Run("anonymous struct with required fields copies properties and required", func(t *testing.T) {
		components := &Components{
			Schemas: make(map[string]*Schema),
		}

		// Create an anonymous struct type (no name)
		anonymousStructType := reflect.TypeOf(struct {
			UserID string `gork:"user_id" validate:"required"`
			Email  string `gork:"email" validate:"required,email"`
			Name   string `gork:"name"`
		}{})

		responseSchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractBodyPropertiesToResponseSchema(anonymousStructType, responseSchema, components)

		// Should have copied properties from the generated body schema
		expectedProps := []string{"user_id", "email", "name"}
		for _, prop := range expectedProps {
			if _, exists := responseSchema.Properties[prop]; !exists {
				t.Errorf("Expected responseSchema to have '%s' property", prop)
			}
		}

		// This is the specific line we want to cover:
		// if bodySchema.Required != nil { responseSchema.Required = bodySchema.Required }
		if len(responseSchema.Required) == 0 {
			t.Error("Expected responseSchema to have required fields copied from bodySchema")
		}

		// Check that the correct fields are marked as required
		requiredMap := make(map[string]bool)
		for _, req := range responseSchema.Required {
			requiredMap[req] = true
		}

		if !requiredMap["user_id"] {
			t.Error("Expected 'user_id' to be required")
		}
		if !requiredMap["email"] {
			t.Error("Expected 'email' to be required")
		}
		if requiredMap["name"] {
			t.Error("'name' should not be required")
		}
	})

	t.Run("anonymous struct with no required fields", func(t *testing.T) {
		components := &Components{
			Schemas: make(map[string]*Schema),
		}

		// Create an anonymous struct type with no required fields
		anonymousStructType := reflect.TypeOf(struct {
			UserID string `gork:"user_id"`
			Email  string `gork:"email"`
		}{})

		responseSchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractBodyPropertiesToResponseSchema(anonymousStructType, responseSchema, components)

		// Should have copied properties
		if _, exists := responseSchema.Properties["user_id"]; !exists {
			t.Error("Expected responseSchema to have 'user_id' property")
		}
		if _, exists := responseSchema.Properties["email"]; !exists {
			t.Error("Expected responseSchema to have 'email' property")
		}

		// Should not have any required fields
		if len(responseSchema.Required) != 0 {
			t.Errorf("Expected no required fields, got: %v", responseSchema.Required)
		}
	})

	t.Run("non-struct type", func(t *testing.T) {
		components := &Components{
			Schemas: make(map[string]*Schema),
		}

		// Test with a string type (not a struct)
		stringType := reflect.TypeOf("test")
		responseSchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractBodyPropertiesToResponseSchema(stringType, responseSchema, components)

		// For non-struct types, generateSchemaFromType will return a basic type schema
		// which won't have Properties, so nothing should be copied
		if len(responseSchema.Properties) != 0 {
			t.Errorf("Expected no properties to be copied for non-struct type, got: %v", responseSchema.Properties)
		}
	})
}
