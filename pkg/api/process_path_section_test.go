package api

import (
	"reflect"
	"testing"
)

// TestProcessPathSectionComprehensive tests all paths in processPathSection
func TestProcessPathSectionComprehensive(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	t.Run("non-struct sectionType - early return", func(t *testing.T) {
		operation := &Operation{}

		// This should trigger lines 96-98 (non-struct early return)
		generator.processPathSection(reflect.TypeOf("string"), operation, components)

		// Should not add any parameters
		if len(operation.Parameters) != 0 {
			t.Error("Expected no parameters for non-struct type")
		}
	})

	t.Run("struct with no gork tags", func(t *testing.T) {
		type PathWithoutTags struct {
			Field1 string
			Field2 int
		}

		operation := &Operation{}

		// This should trigger lines 105-107 (continue for empty gork tags)
		generator.processPathSection(reflect.TypeOf(PathWithoutTags{}), operation, components)

		// Should not add any parameters
		if len(operation.Parameters) != 0 {
			t.Error("Expected no parameters for struct without gork tags")
		}
	})

	t.Run("struct with gork tags", func(t *testing.T) {
		type PathWithTags struct {
			UserID string `gork:"user_id" validate:"required"`
			PostID string `gork:"post_id"`
		}

		operation := &Operation{}

		// This should process both fields with gork tags
		generator.processPathSection(reflect.TypeOf(PathWithTags{}), operation, components)

		// Should add parameters for both fields
		if len(operation.Parameters) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(operation.Parameters))
		}

		// Check first parameter
		if operation.Parameters[0].Name != "user_id" {
			t.Errorf("Expected parameter name 'user_id', got '%s'", operation.Parameters[0].Name)
		}

		if operation.Parameters[0].In != "path" {
			t.Errorf("Expected parameter in 'path', got '%s'", operation.Parameters[0].In)
		}

		if !operation.Parameters[0].Required {
			t.Error("Expected path parameter to be required")
		}

		// Check second parameter
		if operation.Parameters[1].Name != "post_id" {
			t.Errorf("Expected parameter name 'post_id', got '%s'", operation.Parameters[1].Name)
		}
	})

	t.Run("struct with mixed fields - some with gork tags, some without", func(t *testing.T) {
		type MixedPathStruct struct {
			ID          string `gork:"id"`
			NormalField string // No gork tag - should be skipped
			Category    string `gork:"category"`
		}

		operation := &Operation{}

		generator.processPathSection(reflect.TypeOf(MixedPathStruct{}), operation, components)

		// Should only process fields with gork tags
		if len(operation.Parameters) != 2 {
			t.Errorf("Expected 2 parameters (only gork-tagged fields), got %d", len(operation.Parameters))
		}

		// Should have id and category parameters
		paramNames := make(map[string]bool)
		for _, param := range operation.Parameters {
			paramNames[param.Name] = true
		}

		if !paramNames["id"] {
			t.Error("Expected 'id' parameter")
		}

		if !paramNames["category"] {
			t.Error("Expected 'category' parameter")
		}

		if paramNames["NormalField"] {
			t.Error("Should not have processed field without gork tag")
		}
	})

	t.Run("struct with various field types", func(t *testing.T) {
		type PathWithVariousTypes struct {
			StringID string  `gork:"string_id"`
			IntID    int     `gork:"int_id"`
			BoolFlag bool    `gork:"bool_flag"`
			FloatVal float64 `gork:"float_val"`
		}

		operation := &Operation{}

		generator.processPathSection(reflect.TypeOf(PathWithVariousTypes{}), operation, components)

		// Should process all fields with gork tags
		if len(operation.Parameters) != 4 {
			t.Errorf("Expected 4 parameters, got %d", len(operation.Parameters))
		}

		// All should be path parameters and required
		for _, param := range operation.Parameters {
			if param.In != "path" {
				t.Errorf("Expected parameter '%s' to be in 'path', got '%s'", param.Name, param.In)
			}

			if !param.Required {
				t.Errorf("Expected parameter '%s' to be required", param.Name)
			}

			if param.Schema == nil {
				t.Errorf("Expected parameter '%s' to have a schema", param.Name)
			}
		}
	})
}
