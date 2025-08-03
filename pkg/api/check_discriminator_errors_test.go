package api

import (
	"testing"
)

// TestCheckDiscriminatorErrorsMissingCoverage tests specific edge cases to improve coverage
func TestCheckDiscriminatorErrorsMissingCoverage(t *testing.T) {
	t.Run("nil pointer input", func(t *testing.T) {
		var nilPtr *struct{}
		result := CheckDiscriminatorErrors(nilPtr)

		// Should return nil for nil pointer
		if result != nil {
			t.Errorf("Expected nil for nil pointer, got %v", result)
		}
	})

	t.Run("non-struct input", func(t *testing.T) {
		result := CheckDiscriminatorErrors("not a struct")

		// Should return nil for non-struct
		if result != nil {
			t.Errorf("Expected nil for non-struct, got %v", result)
		}
	})

	t.Run("struct with no discriminator fields", func(t *testing.T) {
		type StructWithoutDiscriminator struct {
			Name  string `gork:"name"`
			Value int    `gork:"value"`
		}

		input := StructWithoutDiscriminator{Name: "test", Value: 42}
		result := CheckDiscriminatorErrors(input)

		// Should return nil when no errors found
		if result != nil {
			t.Errorf("Expected nil for struct without discriminator fields, got %v", result)
		}
	})

	t.Run("struct with discriminator but no gork tag name", func(t *testing.T) {
		type StructWithDiscriminatorNoName struct {
			Type string `gork:",discriminator=test"` // No field name, just discriminator
		}

		input := StructWithDiscriminatorNoName{Type: "test"}
		result := CheckDiscriminatorErrors(input)

		// Should use field name when gork tag name is empty
		if result != nil {
			t.Errorf("Expected no errors for valid discriminator, got %v", result)
		}
	})

	t.Run("struct with non-string discriminator field", func(t *testing.T) {
		type StructWithNonStringDiscriminator struct {
			TypeCode int `gork:"type_code,discriminator=123"`
		}

		input := StructWithNonStringDiscriminator{TypeCode: 123}
		result := CheckDiscriminatorErrors(input)

		// Should return nil since non-string fields are skipped
		if result != nil {
			t.Errorf("Expected nil for non-string discriminator field, got %v", result)
		}
	})

	t.Run("struct with empty string discriminator", func(t *testing.T) {
		type StructWithEmptyDiscriminator struct {
			Type string `gork:"type,discriminator=expected"`
		}

		input := StructWithEmptyDiscriminator{Type: ""} // Empty string
		result := CheckDiscriminatorErrors(input)

		// Should have "required" error
		if result == nil {
			t.Fatal("Expected errors for empty discriminator")
		}

		if len(result["type"]) != 1 || result["type"][0] != "required" {
			t.Errorf("Expected 'required' error, got %v", result["type"])
		}
	})

	t.Run("struct with wrong discriminator value", func(t *testing.T) {
		type StructWithWrongDiscriminator struct {
			Type string `gork:"type,discriminator=expected"`
		}

		input := StructWithWrongDiscriminator{Type: "wrong_value"}
		result := CheckDiscriminatorErrors(input)

		// Should have "discriminator" error
		if result == nil {
			t.Fatal("Expected errors for wrong discriminator value")
		}

		if len(result["type"]) != 1 || result["type"][0] != "discriminator" {
			t.Errorf("Expected 'discriminator' error, got %v", result["type"])
		}
	})

	t.Run("struct with correct discriminator value", func(t *testing.T) {
		type StructWithCorrectDiscriminator struct {
			Type string `gork:"type,discriminator=expected"`
		}

		input := StructWithCorrectDiscriminator{Type: "expected"}
		result := CheckDiscriminatorErrors(input)

		// Should return nil for correct discriminator
		if result != nil {
			t.Errorf("Expected nil for correct discriminator, got %v", result)
		}
	})

	t.Run("struct with multiple discriminator fields", func(t *testing.T) {
		type StructWithMultipleDiscriminators struct {
			Type     string `gork:"type,discriminator=expected_type"`
			Category string `gork:"category,discriminator=expected_category"`
		}

		input := StructWithMultipleDiscriminators{
			Type:     "",               // Empty - should error
			Category: "wrong_category", // Wrong value - should error
		}
		result := CheckDiscriminatorErrors(input)

		// Should have errors for both fields
		if result == nil {
			t.Fatal("Expected errors for multiple discriminator fields")
		}

		if len(result["type"]) != 1 || result["type"][0] != "required" {
			t.Errorf("Expected 'required' error for type field, got %v", result["type"])
		}

		if len(result["category"]) != 1 || result["category"][0] != "discriminator" {
			t.Errorf("Expected 'discriminator' error for category field, got %v", result["category"])
		}
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type StructWithDiscriminator struct {
			Type string `gork:"type,discriminator=expected"`
		}

		input := &StructWithDiscriminator{Type: "expected"}
		result := CheckDiscriminatorErrors(input)

		// Should work with pointer to struct
		if result != nil {
			t.Errorf("Expected nil for pointer to struct with correct discriminator, got %v", result)
		}
	})
}
