package api

import (
	"reflect"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

func TestConventionOpenAPIGenerator_ExtractStructPropertiesToSchema(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	t.Run("union type early return", func(t *testing.T) {
		// Test that union types return early without processing fields
		unionType := reflect.TypeOf(unions.Union2[EmailAuth, TokenAuth]{})
		schema := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractStructPropertiesToSchema(unionType, schema, spec.Components)

		// Should have no properties because of early return
		if len(schema.Properties) != 0 {
			t.Errorf("Expected no properties for union type, got %d", len(schema.Properties))
		}
		if len(schema.Required) != 0 {
			t.Errorf("Expected no required fields for union type, got %d", len(schema.Required))
		}
	})

	t.Run("skip unexported fields", func(t *testing.T) {
		// Create a type with both exported and unexported fields
		type TestStructWithUnexportedFields struct {
			ExportedField   string `gork:"exported" validate:"required"`
			unexportedField string `gork:"unexported"`
			AnotherExported int    `gork:"another"`
		}

		structType := reflect.TypeOf(TestStructWithUnexportedFields{})
		schema := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractStructPropertiesToSchema(structType, schema, spec.Components)

		// Should only have exported fields
		if len(schema.Properties) != 2 {
			t.Errorf("Expected 2 properties (only exported fields), got %d", len(schema.Properties))
		}

		// Check that exported fields are present
		if _, exists := schema.Properties["exported"]; !exists {
			t.Error("Expected 'exported' property to be present")
		}
		if _, exists := schema.Properties["another"]; !exists {
			t.Error("Expected 'another' property to be present")
		}

		// Check that unexported field is not present
		if _, exists := schema.Properties["unexported"]; exists {
			t.Error("Unexported field should not be present in schema")
		}

		// Check required fields (only exported field with required validation)
		if len(schema.Required) != 1 {
			t.Errorf("Expected 1 required field, got %d", len(schema.Required))
		}
		if len(schema.Required) > 0 && schema.Required[0] != "exported" {
			t.Errorf("Expected required field 'exported', got %s", schema.Required[0])
		}
	})

	t.Run("field name from gork tag or field name", func(t *testing.T) {
		type TestFieldNames struct {
			WithGorkTag    string `gork:"custom_name"`
			WithoutGorkTag string
		}

		structType := reflect.TypeOf(TestFieldNames{})
		schema := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractStructPropertiesToSchema(structType, schema, spec.Components)

		// Should have both fields with correct names
		if len(schema.Properties) != 2 {
			t.Errorf("Expected 2 properties, got %d", len(schema.Properties))
		}

		// Field with gork tag should use custom name
		if _, exists := schema.Properties["custom_name"]; !exists {
			t.Error("Expected property 'custom_name' from gork tag")
		}

		// Field without gork tag should use field name
		if _, exists := schema.Properties["WithoutGorkTag"]; !exists {
			t.Error("Expected property 'WithoutGorkTag' from field name")
		}
	})

	t.Run("required field detection", func(t *testing.T) {
		type TestRequiredFields struct {
			RequiredField    string `gork:"required_field" validate:"required"`
			OptionalField    string `gork:"optional_field"`
			RequiredEmail    string `gork:"email" validate:"required,email"`
			NonRequiredEmail string `gork:"non_req_email" validate:"email"`
		}

		structType := reflect.TypeOf(TestRequiredFields{})
		schema := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractStructPropertiesToSchema(structType, schema, spec.Components)

		// Should have all 4 properties
		if len(schema.Properties) != 4 {
			t.Errorf("Expected 4 properties, got %d", len(schema.Properties))
		}

		// Should have 2 required fields
		if len(schema.Required) != 2 {
			t.Errorf("Expected 2 required fields, got %d", len(schema.Required))
		}

		// Check that correct fields are marked as required
		requiredMap := make(map[string]bool)
		for _, req := range schema.Required {
			requiredMap[req] = true
		}

		if !requiredMap["required_field"] {
			t.Error("Expected 'required_field' to be required")
		}
		if !requiredMap["email"] {
			t.Error("Expected 'email' to be required")
		}
		if requiredMap["optional_field"] {
			t.Error("'optional_field' should not be required")
		}
		if requiredMap["non_req_email"] {
			t.Error("'non_req_email' should not be required")
		}
	})

	t.Run("nil field schema handling", func(t *testing.T) {
		// This test ensures that if generateSchemaFromType returns nil,
		// the field is not added to properties
		type TestNilSchema struct {
			ValidField string `gork:"valid"`
			// We can't easily test generateSchemaFromType returning nil without mocking,
			// but we can test the normal behavior
		}

		structType := reflect.TypeOf(TestNilSchema{})
		schema := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractStructPropertiesToSchema(structType, schema, spec.Components)

		// Should have the valid field (assuming generateSchemaFromType doesn't return nil for string)
		if len(schema.Properties) == 0 {
			t.Error("Expected at least one property")
		}
		if _, exists := schema.Properties["valid"]; !exists {
			t.Error("Expected 'valid' property to be present")
		}
	})

	t.Run("complex validation tags", func(t *testing.T) {
		type TestComplexValidation struct {
			MultiValidation string `gork:"multi" validate:"required,min=5,max=100"`
			NoRequired      string `gork:"no_req" validate:"min=1,max=50"`
			OnlyRequired    string `gork:"only_req" validate:"required"`
		}

		structType := reflect.TypeOf(TestComplexValidation{})
		schema := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		generator.extractStructPropertiesToSchema(structType, schema, spec.Components)

		// Should have all 3 properties
		if len(schema.Properties) != 3 {
			t.Errorf("Expected 3 properties, got %d", len(schema.Properties))
		}

		// Should have 2 required fields (multi and only_req)
		if len(schema.Required) != 2 {
			t.Errorf("Expected 2 required fields, got %d", len(schema.Required))
		}

		// Check required fields
		requiredMap := make(map[string]bool)
		for _, req := range schema.Required {
			requiredMap[req] = true
		}

		if !requiredMap["multi"] {
			t.Error("Expected 'multi' to be required")
		}
		if !requiredMap["only_req"] {
			t.Error("Expected 'only_req' to be required")
		}
		if requiredMap["no_req"] {
			t.Error("'no_req' should not be required")
		}
	})
}
