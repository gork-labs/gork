package api

import (
	"reflect"
	"testing"
)

// TestIsUnionStructComprehensive tests all code paths in isUnionStruct
func TestIsUnionStructComprehensive(t *testing.T) {
	t.Run("non-struct type", func(t *testing.T) {
		// Test with int type
		intType := reflect.TypeOf(int(0))
		result := isUnionStruct(intType)
		if result {
			t.Error("Expected false for non-struct type")
		}

		// Test with slice type
		sliceType := reflect.TypeOf([]string{})
		result = isUnionStruct(sliceType)
		if result {
			t.Error("Expected false for slice type")
		}

		// Test with pointer type
		ptrType := reflect.TypeOf((*string)(nil))
		result = isUnionStruct(ptrType)
		if result {
			t.Error("Expected false for pointer type")
		}
	})

	t.Run("struct with unexported fields", func(t *testing.T) {
		// Create a struct type with unexported fields
		type StructWithUnexported struct {
			Field1 *string
			field2 *int // unexported - should cause function to return false
			Field3 *bool
		}

		structType := reflect.TypeOf(StructWithUnexported{})
		result := isUnionStruct(structType)
		if result {
			t.Error("Expected false for struct with unexported fields")
		}
	})

	t.Run("struct with non-pointer fields", func(t *testing.T) {
		// Create a struct type with non-pointer fields
		type StructWithNonPtr struct {
			Field1 *string
			Field2 int // non-pointer - should cause function to return false
			Field3 *bool
		}

		structType := reflect.TypeOf(StructWithNonPtr{})
		result := isUnionStruct(structType)
		if result {
			t.Error("Expected false for struct with non-pointer fields")
		}
	})

	t.Run("struct with only one pointer field", func(t *testing.T) {
		// Create a struct type with only one pointer field
		type StructWithOnePtr struct {
			Field1 *string
		}

		structType := reflect.TypeOf(StructWithOnePtr{})
		result := isUnionStruct(structType)
		if result {
			t.Error("Expected false for struct with only one pointer field")
		}
	})

	t.Run("valid union struct with two pointer fields", func(t *testing.T) {
		// Create a struct type that should be considered a union
		type ValidUnionStruct struct {
			Field1 *string
			Field2 *int
		}

		structType := reflect.TypeOf(ValidUnionStruct{})
		result := isUnionStruct(structType)
		if !result {
			t.Error("Expected true for valid union struct with two pointer fields")
		}
	})

	t.Run("valid union struct with three pointer fields", func(t *testing.T) {
		// Create a struct type that should be considered a union
		type ValidUnionStruct struct {
			Field1 *string
			Field2 *int
			Field3 *bool
		}

		structType := reflect.TypeOf(ValidUnionStruct{})
		result := isUnionStruct(structType)
		if !result {
			t.Error("Expected true for valid union struct with three pointer fields")
		}
	})

	t.Run("empty struct", func(t *testing.T) {
		// Create an empty struct type
		type EmptyStruct struct{}

		structType := reflect.TypeOf(EmptyStruct{})
		result := isUnionStruct(structType)
		if result {
			t.Error("Expected false for empty struct")
		}
	})

	t.Run("struct with mixed field types", func(t *testing.T) {
		// Create a struct that has some pointer fields but also other types
		type MixedStruct struct {
			PtrField1    *string
			RegularField int
			PtrField2    *bool
			SliceField   []string
		}

		structType := reflect.TypeOf(MixedStruct{})
		result := isUnionStruct(structType)
		if result {
			t.Error("Expected false for struct with mixed field types")
		}
	})
}
