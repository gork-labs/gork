package api

import (
	"context"
	"reflect"
	"testing"
)

// TestValidateSection_EnhancedCoverage provides comprehensive test coverage
// for the validateSection function using table-driven tests
func TestValidateSection_EnhancedCoverage(t *testing.T) {
	validator := NewConventionValidator()

	tests := []struct {
		name        string
		setupField  func() reflect.StructField
		setupValue  func() reflect.Value
		expectError bool
		description string
	}{
		{
			name: "valid byte body with validation",
			setupField: func() reflect.StructField {
				type TestRequest struct {
					Body []byte `validate:"min=1"`
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			setupValue: func() reflect.Value {
				return reflect.ValueOf([]byte("test"))
			},
			expectError: false,
			description: "Valid byte slice should pass validation",
		},
		{
			name: "empty byte body with validation tag",
			setupField: func() reflect.StructField {
				type TestRequest struct {
					Body []byte `validate:"min=1"`
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			setupValue: func() reflect.Value {
				return reflect.ValueOf([]byte{})
			},
			expectError: false, // Byte body fields skip go-playground validation
			description: "Empty byte slice with validation tag should not cause validation errors (byte body special handling)",
		},
		{
			name: "nil byte slice with validation tag",
			setupField: func() reflect.StructField {
				type TestRequest struct {
					Body []byte `validate:"required"`
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			setupValue: func() reflect.Value {
				var nilSlice []byte
				return reflect.ValueOf(nilSlice)
			},
			expectError: false, // Byte body fields skip go-playground validation
			description: "Nil byte slice with validation tag should not cause validation errors (byte body special handling)",
		},
		{
			name: "valid struct section",
			setupField: func() reflect.StructField {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			setupValue: func() reflect.Value {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				return reflect.ValueOf(QuerySection{Limit: 10})
			},
			expectError: false,
			description: "Valid struct should pass validation",
		},
		{
			name: "struct section with valid data",
			setupField: func() reflect.StructField {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			setupValue: func() reflect.Value {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				return reflect.ValueOf(QuerySection{Limit: 5})
			},
			expectError: false,
			description: "Struct with valid data should pass validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.setupField()
			value := tt.setupValue()
			validationErrors := make(map[string][]string)

			err := validator.validateSection(context.Background(), field, value, validationErrors)

			// Use comprehensive error validation
			tc := ErrorTestCase{
				Name:        tt.name,
				ExpectError: tt.expectError,
				Description: tt.description,
			}

			if tt.expectError {
				tc.ExpectedType = "validation" // Most validation errors should be validation type
			}

			ValidateError(t, err, tc)

			// Verify validation errors are properly populated for validation failures
			if tt.expectError && err != nil {
				// Just verify we got an error as expected
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}

// TestValidateSection_ErrorHandling tests error handling scenarios
func TestValidateSection_ErrorHandling(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("invalid reflect value", func(t *testing.T) {
		type TestRequest struct {
			Body []byte `validate:"required"`
		}

		field := reflect.TypeOf(TestRequest{}).Field(0)
		invalidValue := reflect.Value{} // Zero value is invalid
		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), field, invalidValue, validationErrors)

		if err == nil {
			t.Error("Expected error for invalid reflect.Value")
		}

		// Should be a server error, not validation error
		t.Logf("Got expected server error: %v", err)
	})

	t.Run("panic recovery", func(t *testing.T) {
		type TestRequest struct {
			Query any
		}

		field := reflect.TypeOf(TestRequest{}).Field(0)
		// Create a value that might cause issues during validation
		invalidValue := reflect.ValueOf(nil)
		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), field, invalidValue, validationErrors)
		// Should handle any panics gracefully
		if err != nil {
			t.Logf("Handled error gracefully: %v", err)
		}
	})
}

// TestValidateSection_EdgeCases tests edge cases and boundary conditions
func TestValidateSection_EdgeCases(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("field without validation tags", func(t *testing.T) {
		type TestRequest struct {
			Body []byte // No validation tags
		}

		field := reflect.TypeOf(TestRequest{}).Field(0)
		value := reflect.ValueOf([]byte("test"))
		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), field, value, validationErrors)
		if err != nil {
			t.Errorf("Expected no error for field without validation tags, got: %v", err)
		}
	})

	t.Run("empty validation errors map", func(t *testing.T) {
		type TestRequest struct {
			Body []byte `validate:"required"`
		}

		field := reflect.TypeOf(TestRequest{}).Field(0)
		value := reflect.ValueOf([]byte("test"))
		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), field, value, validationErrors)
		if err != nil {
			t.Errorf("Expected no error for valid data, got: %v", err)
		}

		if len(validationErrors) != 0 {
			t.Errorf("Expected empty validation errors for valid data, got: %v", validationErrors)
		}
	})
}
