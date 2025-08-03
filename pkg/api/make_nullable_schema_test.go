package api

import (
	"testing"
)

func TestMakeNullableSchemaComprehensive(t *testing.T) {
	t.Run("make nullable schema with nil input", func(t *testing.T) {
		result := makeNullableSchema(nil)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.Type != "null" {
			t.Errorf("Expected type 'null', got '%s'", result.Type)
		}
	})

	t.Run("make nullable schema with ref", func(t *testing.T) {
		originalSchema := &Schema{
			Ref: "#/components/schemas/TestType",
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.AnyOf == nil {
			t.Fatal("Expected AnyOf to be set for ref schema")
		}

		if len(result.AnyOf) != 2 {
			t.Errorf("Expected 2 AnyOf schemas, got %d", len(result.AnyOf))
		}

		if result.AnyOf[0] != originalSchema {
			t.Error("Expected first AnyOf to be original schema")
		}

		if result.AnyOf[1].Type != "null" {
			t.Errorf("Expected second AnyOf to be null, got '%s'", result.AnyOf[1].Type)
		}
	})

	t.Run("make nullable schema with properties", func(t *testing.T) {
		originalSchema := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"name": {Type: "string"},
			},
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.AnyOf == nil {
			t.Fatal("Expected AnyOf to be set for schema with properties")
		}

		if len(result.AnyOf) != 2 {
			t.Errorf("Expected 2 AnyOf schemas, got %d", len(result.AnyOf))
		}

		if result.AnyOf[0] != originalSchema {
			t.Error("Expected first AnyOf to be original schema")
		}

		if result.AnyOf[1].Type != "null" {
			t.Errorf("Expected second AnyOf to be null, got '%s'", result.AnyOf[1].Type)
		}
	})

	t.Run("make nullable schema with OneOf", func(t *testing.T) {
		originalSchema := &Schema{
			OneOf: []*Schema{
				{Type: "string"},
				{Type: "integer"},
			},
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.AnyOf == nil {
			t.Fatal("Expected AnyOf to be set for schema with OneOf")
		}

		if len(result.AnyOf) != 2 {
			t.Errorf("Expected 2 AnyOf schemas, got %d", len(result.AnyOf))
		}

		if result.AnyOf[0] != originalSchema {
			t.Error("Expected first AnyOf to be original schema")
		}

		if result.AnyOf[1].Type != "null" {
			t.Errorf("Expected second AnyOf to be null, got '%s'", result.AnyOf[1].Type)
		}
	})

	t.Run("make nullable schema with AnyOf", func(t *testing.T) {
		originalSchema := &Schema{
			AnyOf: []*Schema{
				{Type: "string"},
				{Type: "integer"},
			},
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.AnyOf == nil {
			t.Fatal("Expected AnyOf to be set for schema with existing AnyOf")
		}

		if len(result.AnyOf) != 2 {
			t.Errorf("Expected 2 AnyOf schemas, got %d", len(result.AnyOf))
		}

		if result.AnyOf[0] != originalSchema {
			t.Error("Expected first AnyOf to be original schema")
		}

		if result.AnyOf[1].Type != "null" {
			t.Errorf("Expected second AnyOf to be null, got '%s'", result.AnyOf[1].Type)
		}
	})

	t.Run("make nullable schema with basic type", func(t *testing.T) {
		originalSchema := &Schema{
			Type:        "string",
			Description: "A test string",
			Title:       "TestString",
			MinLength:   intPtr(1),
			MaxLength:   intPtr(100),
			Pattern:     "^[a-z]+$",
			Enum:        []string{"one", "two", "three"},
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.Types == nil {
			t.Fatal("Expected Types array to be set for basic type")
		}

		if len(result.Types) != 2 {
			t.Errorf("Expected 2 types, got %d", len(result.Types))
		}

		if result.Types[0] != "string" || result.Types[1] != "null" {
			t.Errorf("Expected ['string', 'null'], got %v", result.Types)
		}

		// Should preserve all original properties
		if result.Description != "A test string" {
			t.Errorf("Expected description to be preserved, got '%s'", result.Description)
		}

		if result.Title != "TestString" {
			t.Errorf("Expected title to be preserved, got '%s'", result.Title)
		}

		if result.MinLength == nil || *result.MinLength != 1 {
			t.Error("Expected MinLength to be preserved")
		}

		if result.MaxLength == nil || *result.MaxLength != 100 {
			t.Error("Expected MaxLength to be preserved")
		}

		if result.Pattern != "^[a-z]+$" {
			t.Errorf("Expected pattern to be preserved, got '%s'", result.Pattern)
		}

		if len(result.Enum) != 3 {
			t.Errorf("Expected 3 enum values, got %d", len(result.Enum))
		}
	})

	t.Run("make nullable schema with integer type", func(t *testing.T) {
		originalSchema := &Schema{
			Type:    "integer",
			Minimum: floatPtr(0),
			Maximum: floatPtr(100),
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.Types == nil {
			t.Fatal("Expected Types array to be set for integer type")
		}

		if len(result.Types) != 2 {
			t.Errorf("Expected 2 types, got %d", len(result.Types))
		}

		if result.Types[0] != "integer" || result.Types[1] != "null" {
			t.Errorf("Expected ['integer', 'null'], got %v", result.Types)
		}

		// Should preserve numeric constraints
		if result.Minimum == nil || *result.Minimum != 0 {
			t.Error("Expected Minimum to be preserved")
		}

		if result.Maximum == nil || *result.Maximum != 100 {
			t.Error("Expected Maximum to be preserved")
		}
	})

	t.Run("make nullable schema with array items", func(t *testing.T) {
		originalSchema := &Schema{
			Type:  "array",
			Items: &Schema{Type: "string"},
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.Types == nil {
			t.Fatal("Expected Types array to be set")
		}

		if len(result.Types) != 2 {
			t.Errorf("Expected 2 types, got %d", len(result.Types))
		}

		if result.Types[0] != "array" || result.Types[1] != "null" {
			t.Errorf("Expected ['array', 'null'], got %v", result.Types)
		}

		// Should preserve Items
		if result.Items == nil {
			t.Error("Expected Items to be preserved")
		}

		if result.Items.Type != "string" {
			t.Errorf("Expected Items type 'string', got '%s'", result.Items.Type)
		}
	})

	t.Run("make nullable schema with empty type fallback", func(t *testing.T) {
		originalSchema := &Schema{
			// No type, ref, properties, OneOf, or AnyOf - should use fallback
			Description: "Empty schema",
		}

		result := makeNullableSchema(originalSchema)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.AnyOf == nil {
			t.Fatal("Expected AnyOf to be set for fallback case")
		}

		if len(result.AnyOf) != 2 {
			t.Errorf("Expected 2 AnyOf schemas, got %d", len(result.AnyOf))
		}

		if result.AnyOf[0] != originalSchema {
			t.Error("Expected first AnyOf to be original schema")
		}

		if result.AnyOf[1].Type != "null" {
			t.Errorf("Expected second AnyOf to be null, got '%s'", result.AnyOf[1].Type)
		}
	})
}

// Helper functions for pointer values
func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}
