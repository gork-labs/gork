package api

import (
	"encoding/json"
	"testing"
)

func TestSchemaUnmarshalJSON(t *testing.T) {
	t.Run("unmarshal valid schema", func(t *testing.T) {
		var schema Schema

		// Test unmarshaling from JSON object
		jsonData := `{"type": "string", "description": "A test string"}`
		err := json.Unmarshal([]byte(jsonData), &schema)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if schema.Type != "string" {
			t.Errorf("Expected type 'string', got '%s'", schema.Type)
		}

		if schema.Description != "A test string" {
			t.Errorf("Expected description 'A test string', got '%s'", schema.Description)
		}
	})

	t.Run("unmarshal schema with array type", func(t *testing.T) {
		var schema Schema

		// Test with array schema
		jsonData := `{"type": "array", "items": {"type": "string"}}`
		err := json.Unmarshal([]byte(jsonData), &schema)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if schema.Type != "array" {
			t.Errorf("Expected type 'array', got '%s'", schema.Type)
		}

		if schema.Items == nil {
			t.Error("Expected items to be set")
		}
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		var schema Schema

		// Test with invalid JSON
		jsonData := `{invalid json`
		err := json.Unmarshal([]byte(jsonData), &schema)

		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("unmarshal null", func(t *testing.T) {
		var schema Schema

		// Test with null value
		jsonData := `null`
		err := json.Unmarshal([]byte(jsonData), &schema)

		if err != nil {
			t.Errorf("Expected no error for null, got %v", err)
		}

		// Should result in zero value
		if schema.Type != "" {
			t.Errorf("Expected empty type, got '%s'", schema.Type)
		}
	})

	t.Run("unmarshal schema with types array", func(t *testing.T) {
		var schema Schema

		// Test with types array (nullable string)
		jsonData := `{"type": ["string", "null"]}`
		err := json.Unmarshal([]byte(jsonData), &schema)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(schema.Types) != 2 {
			t.Errorf("Expected 2 types, got %d", len(schema.Types))
		}

		if schema.Types[0] != "string" || schema.Types[1] != "null" {
			t.Errorf("Expected ['string', 'null'], got %v", schema.Types)
		}
	})

	t.Run("unmarshal schema with mixed array items", func(t *testing.T) {
		var schema Schema

		// Test with mixed types array containing non-string items
		jsonData := `{"type": ["string", 123, "null"]}`
		err := json.Unmarshal([]byte(jsonData), &schema)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Non-string items should be skipped (remain empty)
		if len(schema.Types) != 3 {
			t.Errorf("Expected 3 types slots, got %d", len(schema.Types))
		}

		if schema.Types[0] != "string" || schema.Types[1] != "" || schema.Types[2] != "null" {
			t.Errorf("Expected ['string', '', 'null'], got %v", schema.Types)
		}
	})

	t.Run("unmarshal schema with empty json object", func(t *testing.T) {
		var schema Schema

		// Test with empty object
		jsonData := `{}`
		err := json.Unmarshal([]byte(jsonData), &schema)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Type field should remain unset
		if schema.Type != "" {
			t.Errorf("Expected empty type, got '%s'", schema.Type)
		}
	})
}
