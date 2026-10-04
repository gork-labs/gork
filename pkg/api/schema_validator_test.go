package api

import (
	"strings"
	"testing"
)

// Helper function for creating pointers
func ptr[T any](v T) *T {
	return &v
}

func TestValidateValueAgainstSchema_Pattern(t *testing.T) {
	schema := OpenAPISchema{
		Type:    OpenAPITypeString,
		Pattern: `^[a-zA-Z0-9]+$`, // Alphanumeric only
	}

	testCases := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid alphanumeric", "abc123", false},
		{"valid letters only", "abcdef", false},
		{"valid numbers only", "123456", false},
		{"invalid with space", "abc 123", true},
		{"invalid with special char", "abc@123", true},
		{"empty string", "", true}, // Empty string fails ^[a-zA-Z0-9]+$ (requires at least one char)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateValueAgainstSchema(tc.value, schema, "TestType")

			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for value '%s'", tc.value)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
			}
		})
	}
}

func TestValidateValueAgainstSchema_StringLength(t *testing.T) {
	schema := OpenAPISchema{
		Type:      OpenAPITypeString,
		MinLength: ptr(3),
		MaxLength: ptr(10),
	}

	testCases := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid length", "hello", false},
		{"minimum length", "abc", false},
		{"maximum length", "1234567890", false},
		{"too short", "ab", true},
		{"too long", "12345678901", true},
		{"empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateValueAgainstSchema(tc.value, schema, "TestType")

			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for value '%s'", tc.value)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
			}
		})
	}
}

func TestValidateValueAgainstSchema_NumericRange(t *testing.T) {
	t.Run("integer validation", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:    OpenAPITypeInteger,
			Minimum: ptr(1.0),
			Maximum: ptr(100.0),
		}

		testCases := []struct {
			name      string
			value     string
			expectErr bool
		}{
			{"valid number", "50", false},
			{"minimum value", "1", false},
			{"maximum value", "100", false},
			{"too small", "0", true},
			{"too large", "101", true},
			{"negative", "-5", true},
			{"not a number", "abc", true}, // Schema validation should catch invalid format
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateValueAgainstSchema(tc.value, schema, "TestType")

				if tc.expectErr && err == nil {
					t.Fatalf("Expected error for value '%s'", tc.value)
				}
				if !tc.expectErr && err != nil {
					t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
				}
			})
		}
	})

	t.Run("float validation", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:    OpenAPITypeNumber,
			Minimum: ptr(0.0),
			Maximum: ptr(1.0),
		}

		testCases := []struct {
			name      string
			value     string
			expectErr bool
		}{
			{"valid float", "0.5", false},
			{"minimum value", "0.0", false},
			{"maximum value", "1.0", false},
			{"too small", "-0.1", true},
			{"too large", "1.1", true},
			{"integer in range", "1", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateValueAgainstSchema(tc.value, schema, "TestType")

				if tc.expectErr && err == nil {
					t.Fatalf("Expected error for value '%s'", tc.value)
				}
				if !tc.expectErr && err != nil {
					t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
				}
			})
		}
	})
}

func TestValidateValueAgainstSchema_Enum(t *testing.T) {
	schema := OpenAPISchema{
		Type: OpenAPITypeString,
		Enum: []interface{}{"low", "medium", "high", 42, true},
	}

	testCases := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid string enum", "low", false},
		{"valid string enum", "medium", false},
		{"valid string enum", "high", false},
		{"valid number enum as string", "42", false},
		{"valid bool enum as string", "true", false},
		{"invalid value", "invalid", true},
		{"case sensitive", "LOW", true},
		{"empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateValueAgainstSchema(tc.value, schema, "TestType")

			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for value '%s'", tc.value)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
			}
		})
	}
}

func TestValidateValueAgainstSchema_ComplexScenarios(t *testing.T) {
	t.Run("pattern and length combined", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:      OpenAPITypeString,
			Pattern:   `^[A-Z]+$`, // Uppercase letters only
			MinLength: ptr(2),
			MaxLength: ptr(5),
		}

		testCases := []struct {
			name      string
			value     string
			expectErr bool
		}{
			{"valid", "ABC", false},
			{"valid at minimum", "AB", false},
			{"valid at maximum", "ABCDE", false},
			{"too short", "A", true},
			{"too long", "ABCDEF", true},
			{"wrong pattern", "abc", true},
			{"wrong pattern and length", "abcdef", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateValueAgainstSchema(tc.value, schema, "TestType")

				if tc.expectErr && err == nil {
					t.Fatalf("Expected error for value '%s'", tc.value)
				}
				if !tc.expectErr && err != nil {
					t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
				}
			})
		}
	})

	t.Run("numeric range with invalid regex pattern", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:    OpenAPITypeInteger,
			Pattern: `[unclosed`, // Invalid regex
			Minimum: ptr(1.0),
		}

		err := validateValueAgainstSchema("5", schema, "TestType")
		if err == nil {
			t.Fatal("Expected error for invalid regex pattern")
		}
		if !strings.Contains(err.Error(), "invalid regex pattern") {
			t.Fatalf("Expected regex error, got: %v", err)
		}
	})

	t.Run("no validation for empty schema", func(t *testing.T) {
		schema := OpenAPISchema{
			Type: OpenAPITypeString,
			// No constraints
		}

		err := validateValueAgainstSchema("any-value", schema, "TestType")
		if err != nil {
			t.Fatalf("Expected no error for unconstrained schema, got: %v", err)
		}
	})
}

func TestValidateValueAgainstSchema_EdgeCases(t *testing.T) {
	t.Run("only minimum length", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:      OpenAPITypeString,
			MinLength: ptr(5),
		}

		testCases := []struct {
			value     string
			expectErr bool
		}{
			{"12345", false},
			{"123456", false},
			{"1234", true},
		}

		for _, tc := range testCases {
			err := validateValueAgainstSchema(tc.value, schema, "TestType")
			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for value '%s'", tc.value)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
			}
		}
	})

	t.Run("only maximum length", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:      OpenAPITypeString,
			MaxLength: ptr(3),
		}

		testCases := []struct {
			value     string
			expectErr bool
		}{
			{"123", false},
			{"12", false},
			{"", false},
			{"1234", true},
		}

		for _, tc := range testCases {
			err := validateValueAgainstSchema(tc.value, schema, "TestType")
			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for value '%s'", tc.value)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
			}
		}
	})

	t.Run("only minimum value", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:    OpenAPITypeInteger,
			Minimum: ptr(10.0),
		}

		testCases := []struct {
			value     string
			expectErr bool
		}{
			{"10", false},
			{"15", false},
			{"9", true},
		}

		for _, tc := range testCases {
			err := validateValueAgainstSchema(tc.value, schema, "TestType")
			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for value '%s'", tc.value)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
			}
		}
	})

	t.Run("only maximum value", func(t *testing.T) {
		schema := OpenAPISchema{
			Type:    OpenAPITypeNumber,
			Maximum: ptr(100.0),
		}

		testCases := []struct {
			value     string
			expectErr bool
		}{
			{"100", false},
			{"99.9", false},
			{"100.1", true},
		}

		for _, tc := range testCases {
			err := validateValueAgainstSchema(tc.value, schema, "TestType")
			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for value '%s'", tc.value)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
			}
		}
	})
}

func TestValidateNumeric_ParseFloatError(t *testing.T) {
	// Test the specific uncovered line 56: ParseFloat error case
	err := validateNumeric("not-a-float", OpenAPITypeNumber, nil, nil, "TestType")
	if err == nil {
		t.Fatal("Expected error for invalid float format")
	}

	// Should be an InvalidFormatError with specific message
	if !strings.Contains(err.Error(), "invalid number format") {
		t.Fatalf("Expected 'invalid number format' in error, got: %s", err.Error())
	}
}
