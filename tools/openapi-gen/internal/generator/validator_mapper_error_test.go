package generator

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorMapperErrors(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		fieldType   string
		expectError string
	}{
		{
			name:        "min without value",
			tag:         "min",
			fieldType:   "string",
			expectError: "validator tag 'min' requires a value",
		},
		{
			name:        "max without value",
			tag:         "max",
			fieldType:   "number",
			expectError: "validator tag 'max' requires a value",
		},
		{
			name:        "len without value",
			tag:         "len",
			fieldType:   "string",
			expectError: "validator tag 'len' requires a value",
		},
		{
			name:        "min with invalid value",
			tag:         "min=abc",
			fieldType:   "number",
			expectError: "invalid value for 'min': abc",
		},
		{
			name:        "max with invalid value",
			tag:         "max=xyz",
			fieldType:   "number",
			expectError: "invalid value for 'max': xyz",
		},
		{
			name:        "gt without value",
			tag:         "gt",
			fieldType:   "number",
			expectError: "validator tag 'gt' requires a value",
		},
		{
			name:        "lt without value",
			tag:         "lt",
			fieldType:   "number",
			expectError: "validator tag 'lt' requires a value",
		},
		{
			name:        "len with invalid value",
			tag:         "len=notanumber",
			fieldType:   "string",
			expectError: "invalid value for 'len': notanumber",
		},
	}
	
	mapper := NewValidatorMapper()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: tt.fieldType}
			err := mapper.MapValidatorTags(tt.tag, schema, tt.fieldType)
			
			if tt.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatorMapperErrorsInSchema(t *testing.T) {
	// Test that errors are properly handled in the generator
	g := New("Test API", "1.0.0")
	
	field := ExtractedField{
		Name:         "TestField",
		Type:         "string",
		JSONTag:      "test_field",
		ValidateTags: "min", // Invalid - missing value
	}
	
	// Call the private method via reflection or test the public API
	// For now, let's test through the validator mapper directly
	schema := &Schema{Type: "string"}
	err := g.validatorMapper.MapValidatorTags(field.ValidateTags, schema, field.Type)
	
	// Should return an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validator tag 'min' requires a value")
}

func TestValidatorMapperORConditionsWithErrors(t *testing.T) {
	mapper := NewValidatorMapper()
	schema := &Schema{Type: "string"}
	
	// Test OR conditions where one condition has an error
	err := mapper.MapValidatorTags("email|min", schema, "string")
	
	// Should not return error but should process what it can
	require.NoError(t, err)
	
	// Should have anyOf with two schemas
	require.NotNil(t, schema.AnyOf)
	require.Len(t, schema.AnyOf, 2)
	
	// The second schema should have an error description
	assert.Contains(t, schema.AnyOf[1].Description, "Validation error:")
}

func TestValidatorMapperValidTags(t *testing.T) {
	// Ensure valid tags still work correctly
	tests := []struct {
		name      string
		tag       string
		fieldType string
		check     func(t *testing.T, schema *Schema)
	}{
		{
			name:      "valid min",
			tag:       "min=5",
			fieldType: "string",
			check: func(t *testing.T, schema *Schema) {
				require.NotNil(t, schema.MinLength)
				assert.Equal(t, 5, *schema.MinLength)
			},
		},
		{
			name:      "valid max",
			tag:       "max=10.5",
			fieldType: "number",
			check: func(t *testing.T, schema *Schema) {
				require.NotNil(t, schema.Maximum)
				assert.Equal(t, 10.5, *schema.Maximum)
			},
		},
		{
			name:      "valid len",
			tag:       "len=8",
			fieldType: "string",
			check: func(t *testing.T, schema *Schema) {
				require.NotNil(t, schema.MinLength)
				require.NotNil(t, schema.MaxLength)
				assert.Equal(t, 8, *schema.MinLength)
				assert.Equal(t, 8, *schema.MaxLength)
			},
		},
	}
	
	mapper := NewValidatorMapper()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: tt.fieldType}
			err := mapper.MapValidatorTags(tt.tag, schema, tt.fieldType)
			
			require.NoError(t, err)
			tt.check(t, schema)
		})
	}
}

func TestValidatorMapperComplexTagsWithErrors(t *testing.T) {
	mapper := NewValidatorMapper()
	schema := &Schema{Type: "string"}
	
	// Mix of valid and invalid tags
	err := mapper.MapValidatorTags("required,min=3,max,email", schema, "string")
	
	// Should return error for the first invalid tag
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validator tag 'max' requires a value")
	
	// Should have processed tags before the error
	require.NotNil(t, schema.MinLength)
	assert.Equal(t, 3, *schema.MinLength)
}