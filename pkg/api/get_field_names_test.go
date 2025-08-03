package api

import (
	"reflect"
	"strings"
	"testing"
)

func TestConventionOpenAPIGenerator_GetFieldNames(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}

	t.Run("non-struct type returns error message", func(t *testing.T) {
		// Test with string type
		result := generator.getFieldNames(reflect.TypeOf("test string"))
		expected := "not a struct (kind: string)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}

		// Test with int type
		result = generator.getFieldNames(reflect.TypeOf(42))
		expected = "not a struct (kind: int)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}

		// Test with slice type
		result = generator.getFieldNames(reflect.TypeOf([]string{}))
		expected = "not a struct (kind: slice)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}

		// Test with map type
		result = generator.getFieldNames(reflect.TypeOf(map[string]int{}))
		expected = "not a struct (kind: map)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("empty struct returns no fields message", func(t *testing.T) {
		type EmptyStruct struct{}

		result := generator.getFieldNames(reflect.TypeOf(EmptyStruct{}))
		expected := "no fields"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("struct with fields returns field names", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Email string
			Age   int
		}

		result := generator.getFieldNames(reflect.TypeOf(TestStruct{}))
		expected := "Name, Email, Age"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("struct with single field returns single field name", func(t *testing.T) {
		type SingleFieldStruct struct {
			OnlyField string
		}

		result := generator.getFieldNames(reflect.TypeOf(SingleFieldStruct{}))
		expected := "OnlyField"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("struct with many fields returns comma-separated names", func(t *testing.T) {
		type ManyFieldsStruct struct {
			Field1 string
			Field2 int
			Field3 bool
			Field4 float64
			Field5 []string
			Field6 map[string]int
		}

		result := generator.getFieldNames(reflect.TypeOf(ManyFieldsStruct{}))
		// Check that all field names are present
		expectedFields := []string{"Field1", "Field2", "Field3", "Field4", "Field5", "Field6"}
		for _, field := range expectedFields {
			if !strings.Contains(result, field) {
				t.Errorf("Expected result to contain %q, got %q", field, result)
			}
		}

		// Check that fields are comma-separated
		if !strings.Contains(result, ", ") {
			t.Errorf("Expected result to contain comma separators, got %q", result)
		}
	})
}
