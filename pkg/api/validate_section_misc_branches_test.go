package api

import (
	"context"
	"reflect"
	"testing"
)

// TestValidateSection_FinalCoverage provides comprehensive test coverage
// for edge cases in the validateSection function
func TestValidateSection_FinalCoverage(t *testing.T) {
	validator := NewConventionValidator()

	tests := []struct {
		name            string
		setupTest       func() (reflect.StructField, reflect.Value)
		expectError     bool
		expectServerErr bool // Expect server error (not validation error)
		description     string
	}{
		{
			name: "byte body with nil interface",
			setupTest: func() (reflect.StructField, reflect.Value) {
				type TestRequest struct {
					Body []byte `validate:"min=1"`
				}
				field := reflect.TypeOf(TestRequest{}).Field(0)

				// Try with a nil interface - this might trigger InvalidValidationError
				var nilInterface any
				nilValue := reflect.ValueOf(&nilInterface).Elem()

				return field, nilValue
			},
			expectError:     true,
			expectServerErr: true,
			description:     "Nil interface should trigger server error",
		},
		{
			name: "struct section with nil interface",
			setupTest: func() (reflect.StructField, reflect.Value) {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				field := reflect.TypeOf(TestRequest{}).Field(0)

				// Try with a nil interface - this might trigger InvalidValidationError
				var nilInterface any
				nilValue := reflect.ValueOf(&nilInterface).Elem()

				return field, nilValue
			},
			expectError:     true,
			expectServerErr: true,
			description:     "Nil interface for struct should trigger server error",
		},
		{
			name: "invalid reflect value for byte body",
			setupTest: func() (reflect.StructField, reflect.Value) {
				type TestRequest struct {
					Body []byte `validate:"required"`
				}
				field := reflect.TypeOf(TestRequest{}).Field(0)
				invalidValue := reflect.Value{} // Zero value is invalid

				return field, invalidValue
			},
			expectError:     true,
			expectServerErr: true,
			description:     "Invalid reflect.Value should trigger server error",
		},
		{
			name: "invalid reflect value for struct section",
			setupTest: func() (reflect.StructField, reflect.Value) {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				field := reflect.TypeOf(TestRequest{}).Field(0)
				invalidValue := reflect.Value{} // Zero value is invalid

				return field, invalidValue
			},
			expectError:     true,
			expectServerErr: true,
			description:     "Invalid reflect.Value for struct should trigger server error",
		},
		{
			name: "valid byte body",
			setupTest: func() (reflect.StructField, reflect.Value) {
				type TestRequest struct {
					Body []byte `validate:"min=1"`
				}
				field := reflect.TypeOf(TestRequest{}).Field(0)
				value := reflect.ValueOf([]byte("test"))

				return field, value
			},
			expectError: false,
			description: "Valid byte body should pass",
		},
		{
			name: "valid struct section",
			setupTest: func() (reflect.StructField, reflect.Value) {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				field := reflect.TypeOf(TestRequest{}).Field(0)
				value := reflect.ValueOf(QuerySection{Limit: 10})

				return field, value
			},
			expectError: false,
			description: "Valid struct section should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, value := tt.setupTest()
			validationErrors := make(map[string][]string)

			err := validator.validateSection(context.Background(), field, value, validationErrors)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none: %s", tt.description)
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v (%s)", err, tt.description)
				return
			}

			if tt.expectError && err != nil {
				// Just verify we got an error as expected
				if tt.expectServerErr {
					// For server errors, we expect the error to be returned directly
					t.Logf("Got expected server error: %v", err)
				} else {
					// For validation errors, just log that we got an error
					t.Logf("Got expected validation error: %v", err)
				}

				// Server errors should not populate validation errors map
				if tt.expectServerErr && len(validationErrors) != 0 {
					t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
				}
			}
		})
	}
}
