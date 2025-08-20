package api

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestSectionValidationTableDriven uses table-driven tests for section validation scenarios
func TestSectionValidationTableDriven(t *testing.T) {
	v := NewConventionValidator()

	tests := []struct {
		name                string
		structDef           func() reflect.StructField
		value               func() reflect.Value
		expectError         bool
		expectValidationErr bool
		validateResult      func(t *testing.T, err error, validationErrors map[string][]string)
	}{
		{
			name: "ByteBody_NoValidationTag",
			structDef: func() reflect.StructField {
				type TestRequest struct {
					Body []byte // No validate tag
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			value: func() reflect.Value {
				return reflect.ValueOf([]byte("test"))
			},
			expectError:         false,
			expectValidationErr: false,
		},
		{
			name: "ByteBody_ValidData",
			structDef: func() reflect.StructField {
				type TestRequest struct {
					Body []byte `validate:"min=1"`
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			value: func() reflect.Value {
				return reflect.ValueOf([]byte("test"))
			},
			expectError:         false,
			expectValidationErr: false,
		},
		{
			name: "ByteBody_ValidationFailure",
			structDef: func() reflect.StructField {
				type TestRequest struct {
					Body []byte `validate:"min=5"`
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			value: func() reflect.Value {
				return reflect.ValueOf([]byte("hi")) // Too short
			},
			expectError:         false,
			expectValidationErr: true,
			validateResult: func(t *testing.T, err error, validationErrors map[string][]string) {
				if len(validationErrors["body"]) == 0 {
					t.Error("Expected validation errors for body section")
				}
			},
		},
		{
			name: "ByteBody_InvalidReflectValue",
			structDef: func() reflect.StructField {
				type TestRequest struct {
					Body []byte `validate:"required"`
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			value: func() reflect.Value {
				return reflect.Value{} // Zero value is invalid
			},
			expectError:         true,
			expectValidationErr: false,
		},
		{
			name: "Struct_ValidData",
			structDef: func() reflect.StructField {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			value: func() reflect.Value {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				return reflect.ValueOf(QuerySection{Limit: 10})
			},
			expectError:         false,
			expectValidationErr: false,
		},
		{
			name: "Struct_ValidationFailure",
			structDef: func() reflect.StructField {
				type QuerySection struct {
					Limit int `validate:"min=10"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			value: func() reflect.Value {
				type QuerySection struct {
					Limit int `validate:"min=10"`
				}
				return reflect.ValueOf(QuerySection{Limit: 5}) // Too small
			},
			expectError:         false,
			expectValidationErr: true,
		},
		{
			name: "Struct_InvalidReflectValue",
			structDef: func() reflect.StructField {
				type QuerySection struct {
					Limit int `validate:"min=1"`
				}
				type TestRequest struct {
					Query QuerySection
				}
				return reflect.TypeOf(TestRequest{}).Field(0)
			},
			value: func() reflect.Value {
				return reflect.Value{} // Zero value is invalid
			},
			expectError:         true,
			expectValidationErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.structDef()
			value := tt.value()
			validationErrors := make(map[string][]string)

			err := v.validateSection(context.Background(), field, value, validationErrors)

			if tt.expectError && err == nil {
				t.Error("Expected server error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no server error but got: %v", err)
			}

			hasValidationErrors := len(validationErrors) > 0
			if tt.expectValidationErr && !hasValidationErrors {
				t.Error("Expected validation errors but got none")
			}
			if !tt.expectValidationErr && hasValidationErrors {
				t.Errorf("Expected no validation errors but got: %v", validationErrors)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, err, validationErrors)
			}
		})
	}
}

// TestCustomValidationTableDriven tests custom validation with table-driven approach
func TestCustomValidationTableDriven(t *testing.T) {
	v := NewConventionValidator()

	tests := []struct {
		name                string
		sectionName         string
		customValidator     interface{}
		expectError         bool
		expectValidationErr bool
		expectedErrorMsg    string
	}{
		{
			name:        "ContextValidator_Success",
			sectionName: "Query",
			customValidator: ComprehensiveCustomValidationSection{
				Data:        "test",
				ShouldError: false,
				ServerError: false,
			},
			expectError:         false,
			expectValidationErr: false,
		},
		{
			name:        "ContextValidator_ValidationError",
			sectionName: "Cookies",
			customValidator: ComprehensiveCustomValidationSection{
				Data:        "test",
				ShouldError: true,
				ServerError: false,
			},
			expectError:         false,
			expectValidationErr: true,
		},
		{
			name:        "ContextValidator_ServerError",
			sectionName: "Query",
			customValidator: ComprehensiveCustomValidationSection{
				Data:        "test",
				ShouldError: false,
				ServerError: true,
			},
			expectError:      true,
			expectedErrorMsg: "database connection failed during section validation",
		},
		{
			name:        "RegularValidator_ServerError",
			sectionName: "Headers",
			customValidator: RegularCustomValidationSection{
				Data:        "test",
				ShouldError: false,
				ServerError: true,
			},
			expectError:      true,
			expectedErrorMsg: "external service unavailable during section validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create field dynamically based on section name
			fieldName := tt.sectionName
			structType := reflect.StructOf([]reflect.StructField{
				{
					Name: fieldName,
					Type: reflect.TypeOf(tt.customValidator),
				},
			})
			field := structType.Field(0)
			value := reflect.ValueOf(tt.customValidator)
			validationErrors := make(map[string][]string)

			err := v.validateSection(context.Background(), field, value, validationErrors)

			if tt.expectError {
				if err == nil {
					t.Error("Expected server error but got none")
				} else if tt.expectedErrorMsg != "" && err.Error() != tt.expectedErrorMsg {
					t.Errorf("Expected error message '%s', got: %v", tt.expectedErrorMsg, err)
				}
				if len(validationErrors) != 0 {
					t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no server error but got: %v", err)
				}

				sectionKey := strings.ToLower(tt.sectionName)
				hasValidationErrors := len(validationErrors[sectionKey]) > 0
				if tt.expectValidationErr && !hasValidationErrors {
					t.Errorf("Expected validation errors for %s section", sectionKey)
				}
				if !tt.expectValidationErr && hasValidationErrors {
					t.Errorf("Expected no validation errors but got: %v", validationErrors)
				}
			}
		})
	}
}

// ComprehensiveCustomValidationSection implements ContextValidator for testing custom validation paths
type ComprehensiveCustomValidationSection struct {
	Data        string `gork:"data"`
	ShouldError bool
	ServerError bool
}

func (c ComprehensiveCustomValidationSection) Validate(ctx context.Context) error {
	if c.ServerError {
		return errors.New("database connection failed during section validation")
	}
	if c.ShouldError {
		return &RequestValidationError{Errors: []string{"custom validation failed"}}
	}
	return nil
}

// RegularCustomValidationSection implements Validator (not ContextValidator) for testing
type RegularCustomValidationSection struct {
	Data        string `gork:"data"`
	ShouldError bool
	ServerError bool
}

func (r RegularCustomValidationSection) Validate() error {
	if r.ServerError {
		return errors.New("external service unavailable during section validation")
	}
	if r.ShouldError {
		return &RequestValidationError{Errors: []string{"regular custom validation failed"}}
	}
	return nil
}
