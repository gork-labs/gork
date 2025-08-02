package api

import (
	"reflect"
	"testing"
)

// Test types for comprehensive validator testing
type DiscriminatorTestStruct struct {
	Type        string `json:"type" openapi:"discriminator=user"`
	UserField   string `json:"userField"`
	RequiredOne string `json:"requiredOne" validate:"required"`
}

type MultiDiscriminatorStruct struct {
	EntityType string `json:"entityType" openapi:"discriminator=entity"`
	Action     string `json:"action" openapi:"discriminator=create"`
	Name       string `json:"name" validate:"required"`
}

type NoDiscriminatorStruct struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"email"`
}

type NestedDiscriminatorStruct struct {
	Outer OuterStruct `json:"outer"`
}

type OuterStruct struct {
	Inner InnerStruct `json:"inner"`
}

type InnerStruct struct {
	Type string `json:"type" openapi:"discriminator=nested"`
}

type PointerDiscriminatorStruct struct {
	Type *string `json:"type" openapi:"discriminator=pointer"`
}

func TestCheckDiscriminatorErrors_ValidDiscriminator(t *testing.T) {
	// Test valid discriminator value
	validStruct := DiscriminatorTestStruct{
		Type:        "user",
		UserField:   "test",
		RequiredOne: "present",
	}

	errors := CheckDiscriminatorErrors(validStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no discriminator errors for valid struct, got %d errors", len(errors))
	}
}

func TestCheckDiscriminatorErrors_InvalidDiscriminator(t *testing.T) {
	// Test invalid discriminator value
	invalidStruct := DiscriminatorTestStruct{
		Type:        "admin", // Wrong discriminator value
		UserField:   "test",
		RequiredOne: "present",
	}

	errors := CheckDiscriminatorErrors(invalidStruct)
	if len(errors) != 1 {
		t.Errorf("Expected 1 discriminator error, got %d", len(errors))
	}

	if fieldErrors, ok := errors["type"]; !ok {
		t.Error("Expected error for 'type' field")
	} else if len(fieldErrors) != 1 || fieldErrors[0] != "discriminator" {
		t.Errorf("Expected 'discriminator' error, got %v", fieldErrors)
	}
}

func TestCheckDiscriminatorErrors_EmptyDiscriminator(t *testing.T) {
	// Test empty discriminator value
	emptyStruct := DiscriminatorTestStruct{
		Type:        "", // Empty discriminator
		UserField:   "test",
		RequiredOne: "present",
	}

	errors := CheckDiscriminatorErrors(emptyStruct)
	if len(errors) != 1 {
		t.Errorf("Expected 1 discriminator error for empty value, got %d", len(errors))
	}

	if fieldErrors, ok := errors["type"]; !ok {
		t.Error("Expected error for 'type' field")
	} else if len(fieldErrors) != 1 || fieldErrors[0] != "required" {
		t.Errorf("Expected 'required' error, got %v", fieldErrors)
	}
}

func TestCheckDiscriminatorErrors_MultipleDiscriminators(t *testing.T) {
	// Test struct with multiple discriminator fields
	tests := []struct {
		name  string
		input MultiDiscriminatorStruct
		want  map[string][]string
	}{
		{
			name: "all valid",
			input: MultiDiscriminatorStruct{
				EntityType: "entity",
				Action:     "create",
				Name:       "test",
			},
			want: map[string][]string{},
		},
		{
			name: "first invalid",
			input: MultiDiscriminatorStruct{
				EntityType: "wrong",
				Action:     "create",
				Name:       "test",
			},
			want: map[string][]string{
				"entityType": {"discriminator"},
			},
		},
		{
			name: "both invalid",
			input: MultiDiscriminatorStruct{
				EntityType: "wrong",
				Action:     "wrong",
				Name:       "test",
			},
			want: map[string][]string{
				"entityType": {"discriminator"},
				"action":     {"discriminator"},
			},
		},
		{
			name: "both empty",
			input: MultiDiscriminatorStruct{
				EntityType: "",
				Action:     "",
				Name:       "test",
			},
			want: map[string][]string{
				"entityType": {"required"},
				"action":     {"required"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := CheckDiscriminatorErrors(tt.input)

			if len(errors) != len(tt.want) {
				t.Errorf("Expected %d errors, got %d", len(tt.want), len(errors))
			}

			for field, expectedErrors := range tt.want {
				if actualErrors, ok := errors[field]; !ok {
					t.Errorf("Expected error for field %s", field)
				} else if len(actualErrors) != len(expectedErrors) {
					t.Errorf("Field %s: expected %d errors, got %d", field, len(expectedErrors), len(actualErrors))
				} else {
					for i, expectedError := range expectedErrors {
						if actualErrors[i] != expectedError {
							t.Errorf("Field %s[%d]: expected %s, got %s", field, i, expectedError, actualErrors[i])
						}
					}
				}
			}
		})
	}
}

func TestCheckDiscriminatorErrors_NoDiscriminatorFields(t *testing.T) {
	// Test struct with no discriminator fields
	noDiscStruct := NoDiscriminatorStruct{
		Name:  "test",
		Email: "test@example.com",
	}

	errors := CheckDiscriminatorErrors(noDiscStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for struct without discriminators, got %d", len(errors))
	}
}

func TestCheckDiscriminatorErrors_PointerStruct(t *testing.T) {
	// Test with pointer to struct
	validStruct := &DiscriminatorTestStruct{
		Type:        "user",
		UserField:   "test",
		RequiredOne: "present",
	}

	errors := CheckDiscriminatorErrors(validStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid pointer struct, got %d", len(errors))
	}

	// Test invalid pointer struct
	invalidStruct := &DiscriminatorTestStruct{
		Type:        "wrong",
		UserField:   "test",
		RequiredOne: "present",
	}

	errors = CheckDiscriminatorErrors(invalidStruct)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error for invalid pointer struct, got %d", len(errors))
	}
}

func TestCheckDiscriminatorErrors_NilPointer(t *testing.T) {
	// Test with nil pointer
	var nilStruct *DiscriminatorTestStruct = nil

	errors := CheckDiscriminatorErrors(nilStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for nil pointer, got %d", len(errors))
	}
}

func TestCheckDiscriminatorErrors_NonStruct(t *testing.T) {
	// Test with non-struct types
	testCases := []interface{}{
		"string",
		42,
		[]int{1, 2, 3},
		map[string]string{"key": "value"},
	}

	for _, testCase := range testCases {
		errors := CheckDiscriminatorErrors(testCase)
		if len(errors) != 0 {
			t.Errorf("Expected no errors for non-struct type %T, got %d", testCase, len(errors))
		}
	}
}

func TestCheckDiscriminatorErrors_NestedStructs(t *testing.T) {
	// Test that nested structs don't affect discriminator checking
	nestedStruct := NestedDiscriminatorStruct{
		Outer: OuterStruct{
			Inner: InnerStruct{
				Type: "wrong", // This shouldn't be checked as it's nested
			},
		},
	}

	errors := CheckDiscriminatorErrors(nestedStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for nested struct discriminators, got %d", len(errors))
	}
}

func TestCheckDiscriminatorErrors_PointerFields(t *testing.T) {
	// Test with pointer fields - current implementation doesn't handle pointer fields
	validValue := "pointer"
	validStruct := PointerDiscriminatorStruct{
		Type: &validValue,
	}

	errors := CheckDiscriminatorErrors(validStruct)
	// Current implementation skips non-string fields (including *string)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for pointer field (not supported), got %d", len(errors))
	}

	// Test with nil pointer field - also should be skipped
	nilStruct := PointerDiscriminatorStruct{
		Type: nil,
	}

	errors = CheckDiscriminatorErrors(nilStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for nil pointer field (not supported), got %d", len(errors))
	}

	// Test with wrong pointer value - also should be skipped
	wrongValue := "wrong"
	wrongStruct := PointerDiscriminatorStruct{
		Type: &wrongValue,
	}

	errors = CheckDiscriminatorErrors(wrongStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for wrong pointer value (not supported), got %d", len(errors))
	}
}

func TestCheckDiscriminatorErrors_EdgeCases(t *testing.T) {
	// Test edge cases
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"nil interface", nil, 0},
		{"interface{} with struct", interface{}(DiscriminatorTestStruct{Type: "user"}), 0},
		{"reflect.Value zero", reflect.Value{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := CheckDiscriminatorErrors(tt.input)
			if len(errors) != tt.expected {
				t.Errorf("Expected %d errors, got %d", tt.expected, len(errors))
			}
		})
	}
}

func TestCheckDiscriminatorErrors_ComplexOpenAPITags(t *testing.T) {
	// Test complex openapi tags with multiple parameters
	type ComplexTagStruct struct {
		Type string `json:"type" openapi:"name=entity_type,in=query,discriminator=complex,required=true"`
	}

	validStruct := ComplexTagStruct{Type: "complex"}
	errors := CheckDiscriminatorErrors(validStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid complex tag struct, got %d", len(errors))
	}

	invalidStruct := ComplexTagStruct{Type: "wrong"}
	errors = CheckDiscriminatorErrors(invalidStruct)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error for invalid complex tag struct, got %d", len(errors))
	}
}

func TestCheckDiscriminatorErrors_CaseInsensitive(t *testing.T) {
	// Discriminator validation should be case-sensitive
	type CaseTestStruct struct {
		Type string `json:"type" openapi:"discriminator=User"`
	}

	// Test exact case match
	validStruct := CaseTestStruct{Type: "User"}
	errors := CheckDiscriminatorErrors(validStruct)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for exact case match, got %d", len(errors))
	}

	// Test different case
	invalidStruct := CaseTestStruct{Type: "user"}
	errors = CheckDiscriminatorErrors(invalidStruct)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error for case mismatch, got %d", len(errors))
	}
}

// Benchmark tests for discriminator validation performance
func BenchmarkCheckDiscriminatorErrors_Valid(b *testing.B) {
	validStruct := DiscriminatorTestStruct{
		Type:        "user",
		UserField:   "test",
		RequiredOne: "present",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckDiscriminatorErrors(validStruct)
	}
}

func BenchmarkCheckDiscriminatorErrors_Invalid(b *testing.B) {
	invalidStruct := DiscriminatorTestStruct{
		Type:        "wrong",
		UserField:   "test",
		RequiredOne: "present",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckDiscriminatorErrors(invalidStruct)
	}
}

func BenchmarkCheckDiscriminatorErrors_NoDiscriminator(b *testing.B) {
	noDiscStruct := NoDiscriminatorStruct{
		Name:  "test",
		Email: "test@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckDiscriminatorErrors(noDiscStruct)
	}
}
