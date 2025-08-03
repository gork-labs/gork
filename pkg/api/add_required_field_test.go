package api

import (
	"reflect"
	"testing"
)

func TestAddRequiredFieldComprehensive(t *testing.T) {
	t.Run("add required field to empty schema", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: []string{},
		}

		field := reflect.StructField{
			Name: "TestField",
			Tag:  `gork:"test_field"`,
		}

		addRequiredField(schema, field)

		if len(schema.Required) != 1 {
			t.Errorf("Expected 1 required field, got %d", len(schema.Required))
		}

		if schema.Required[0] != "test_field" {
			t.Errorf("Expected 'test_field', got '%s'", schema.Required[0])
		}
	})

	t.Run("add required field to schema with existing fields", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: []string{"existing_field"},
		}

		field := reflect.StructField{
			Name: "TestField",
			Tag:  `gork:"test_field"`,
		}

		addRequiredField(schema, field)

		if len(schema.Required) != 2 {
			t.Errorf("Expected 2 required fields, got %d", len(schema.Required))
		}

		if schema.Required[0] != "existing_field" {
			t.Errorf("Expected first field 'existing_field', got '%s'", schema.Required[0])
		}

		if schema.Required[1] != "test_field" {
			t.Errorf("Expected second field 'test_field', got '%s'", schema.Required[1])
		}
	})

	t.Run("do not add duplicate required field", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: []string{"test_field", "other_field"},
		}

		field := reflect.StructField{
			Name: "TestField",
			Tag:  `gork:"test_field"`,
		}

		addRequiredField(schema, field)

		// Should not add duplicate - should remain 2 fields
		if len(schema.Required) != 2 {
			t.Errorf("Expected 2 required fields (no duplicate), got %d", len(schema.Required))
		}

		if schema.Required[0] != "test_field" {
			t.Errorf("Expected first field 'test_field', got '%s'", schema.Required[0])
		}

		if schema.Required[1] != "other_field" {
			t.Errorf("Expected second field 'other_field', got '%s'", schema.Required[1])
		}
	})

	t.Run("add required field with no gork tag", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: []string{},
		}

		field := reflect.StructField{
			Name: "TestField",
			// No gork tag - should use field name
		}

		addRequiredField(schema, field)

		// Should add field using field name when no gork tag present
		if len(schema.Required) != 1 {
			t.Errorf("Expected 1 required field, got %d", len(schema.Required))
		}

		if schema.Required[0] != "TestField" {
			t.Errorf("Expected 'TestField', got '%s'", schema.Required[0])
		}
	})

	t.Run("add required field with gork tag", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: []string{},
		}

		field := reflect.StructField{
			Name: "TestField",
			Tag:  `gork:"test_field"`,
		}

		addRequiredField(schema, field)

		if len(schema.Required) != 1 {
			t.Errorf("Expected 1 required field, got %d", len(schema.Required))
		}

		if schema.Required[0] != "test_field" {
			t.Errorf("Expected 'test_field', got '%s'", schema.Required[0])
		}
	})

	t.Run("add multiple required fields in order", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: []string{},
		}

		field1 := reflect.StructField{
			Name: "FirstField",
			Tag:  `gork:"first"`,
		}

		field2 := reflect.StructField{
			Name: "SecondField",
			Tag:  `gork:"second"`,
		}

		field3 := reflect.StructField{
			Name: "ThirdField",
			Tag:  `gork:"third"`,
		}

		addRequiredField(schema, field1)
		addRequiredField(schema, field2)
		addRequiredField(schema, field3)

		if len(schema.Required) != 3 {
			t.Errorf("Expected 3 required fields, got %d", len(schema.Required))
		}

		expectedOrder := []string{"first", "second", "third"}
		for i, expected := range expectedOrder {
			if schema.Required[i] != expected {
				t.Errorf("Expected field %d to be '%s', got '%s'", i, expected, schema.Required[i])
			}
		}
	})

	t.Run("add required field with duplicate in middle", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: []string{"first", "duplicate", "third"},
		}

		field := reflect.StructField{
			Name: "DuplicateField",
			Tag:  `gork:"duplicate"`,
		}

		addRequiredField(schema, field)

		// Should not add duplicate - should remain 3 fields
		if len(schema.Required) != 3 {
			t.Errorf("Expected 3 required fields (no duplicate), got %d", len(schema.Required))
		}

		expectedOrder := []string{"first", "duplicate", "third"}
		for i, expected := range expectedOrder {
			if schema.Required[i] != expected {
				t.Errorf("Expected field %d to be '%s', got '%s'", i, expected, schema.Required[i])
			}
		}
	})

	t.Run("add required field to nil required slice", func(t *testing.T) {
		schema := &Schema{
			Type:     "object",
			Required: nil, // nil slice
		}

		field := reflect.StructField{
			Name: "TestField",
			Tag:  `gork:"test_field"`,
		}

		addRequiredField(schema, field)

		if len(schema.Required) != 1 {
			t.Errorf("Expected 1 required field, got %d", len(schema.Required))
		}

		if schema.Required[0] != "test_field" {
			t.Errorf("Expected 'test_field', got '%s'", schema.Required[0])
		}
	})
}
