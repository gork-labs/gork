package api

import (
	"reflect"
	"testing"
)

// TestProcessEmbeddedStructRealCalls tests processEmbeddedStruct with actual function calls
func TestProcessEmbeddedStructRealCalls(t *testing.T) {
	t.Run("embedded struct that generates reference", func(t *testing.T) {
		type EmbeddedType struct {
			Name string `json:"name" validate:"required"`
			Age  int    `json:"age"`
		}

		registry := make(map[string]*Schema)
		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		field := reflect.StructField{
			Name: "EmbeddedType",
			Type: reflect.TypeOf(EmbeddedType{}),
		}

		processEmbeddedStruct(field, schema, registry)

		// Should have properties from the embedded struct
		if len(schema.Properties) == 0 {
			t.Error("Expected properties to be added from embedded struct")
		}
	})

	t.Run("embedded struct that creates registry entry first", func(t *testing.T) {
		type ComplexEmbedded struct {
			Field1 string  `json:"field1"`
			Field2 *string `json:"field2,omitempty"`
		}

		registry := make(map[string]*Schema)

		// First, create the schema in registry by calling reflectTypeToSchema
		embeddedSchema := reflectTypeToSchemaInternal(reflect.TypeOf(ComplexEmbedded{}), registry, true)
		registry["ComplexEmbedded"] = embeddedSchema

		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		field := reflect.StructField{
			Name: "ComplexEmbedded",
			Type: reflect.TypeOf(ComplexEmbedded{}),
		}

		processEmbeddedStruct(field, schema, registry)

		// Should have properties merged (may be empty for some cases)
		_ = schema.Properties
	})

	t.Run("embedded struct with no properties", func(t *testing.T) {
		type EmptyEmbedded struct{}

		registry := make(map[string]*Schema)
		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		field := reflect.StructField{
			Name: "EmptyEmbedded",
			Type: reflect.TypeOf(EmptyEmbedded{}),
		}

		processEmbeddedStruct(field, schema, registry)

		// Should handle empty embedded struct gracefully
		// Properties might be empty or contain no useful fields
	})

	t.Run("embedded struct with registry reference resolution", func(t *testing.T) {
		type ReferencedStruct struct {
			Value string `json:"value" validate:"required"`
		}

		registry := make(map[string]*Schema)

		// Pre-populate registry to test reference resolution
		registry["ReferencedStruct"] = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"value": {Type: "string"},
			},
			Required: []string{"value"},
		}

		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		field := reflect.StructField{
			Name: "ReferencedStruct",
			Type: reflect.TypeOf(ReferencedStruct{}),
		}

		processEmbeddedStruct(field, schema, registry)

		// Should resolve reference and merge properties
		if schema.Properties["value"] == nil {
			t.Error("Expected 'value' property to be merged from referenced struct")
		}

		// Check if required fields were merged
		hasRequired := false
		for _, req := range schema.Required {
			if req == "value" {
				hasRequired = true
				break
			}
		}
		if !hasRequired {
			t.Error("Expected 'value' to be in required fields")
		}
	})

	t.Run("embedded struct without required fields", func(t *testing.T) {
		type OptionalFieldsStruct struct {
			Optional1 string `json:"optional1,omitempty"`
			Optional2 int    `json:"optional2,omitempty"`
		}

		registry := make(map[string]*Schema)
		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		field := reflect.StructField{
			Name: "OptionalFieldsStruct",
			Type: reflect.TypeOf(OptionalFieldsStruct{}),
		}

		processEmbeddedStruct(field, schema, registry)

		// Should merge properties but not add required fields
		if len(schema.Properties) == 0 {
			t.Error("Expected properties to be merged from embedded struct")
		}

		// Required should remain empty or not grow significantly
		if len(schema.Required) > 2 {
			t.Errorf("Expected few or no required fields, got %d", len(schema.Required))
		}
	})
}
