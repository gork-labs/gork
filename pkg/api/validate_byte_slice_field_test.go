package api

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// mockFieldValidator is a test double that can be configured to return different types of errors
type mockFieldValidator struct {
	varError    error
	structError error
}

func (m *mockFieldValidator) Var(field interface{}, tag string) error {
	return m.varError
}

func (m *mockFieldValidator) Struct(s interface{}) error {
	return m.structError
}

func TestValidateByteSliceField_NonValidationError(t *testing.T) {
	// Create a mock field validator that returns a non-ValidationErrors error
	serverError := errors.New("database connection failed")
	mockValidator := &mockFieldValidator{
		varError: serverError,
	}

	// Create a convention validator with the mock field validator
	v := &ConventionValidator{
		validator:      NewValidator(DefaultValidatorConfig()),
		fieldValidator: mockValidator,
		applyRulesFunc: func(ctx context.Context, reqPtr interface{}) []error { return nil },
	}

	// Create a test struct field for a []byte Body with validation tag
	type TestRequest struct {
		Body []byte `validate:"required"`
	}
	field := reflect.TypeOf(TestRequest{}).Field(0)
	fieldValue := reflect.ValueOf([]byte("test data"))
	validationErrors := make(map[string][]string)

	// Call validateByteSliceField directly through validateSection
	err := v.validateSection(context.Background(), field, fieldValue, validationErrors)

	// Verify that the server error was returned directly
	if err == nil {
		t.Fatal("Expected server error to be returned, got nil")
	}

	if err != serverError {
		t.Errorf("Expected server error '%v', got '%v'", serverError, err)
	}

	// Validation errors map should remain empty since this is a server error
	if len(validationErrors) != 0 {
		t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
	}
}

func TestValidateByteSliceField_ReflectionPanic(t *testing.T) {
	// Create a mock field validator that panics when called
	mockValidator := &mockFieldValidator{
		varError: nil, // Will never be reached due to panic
	}

	// Create a convention validator with the mock field validator
	v := &ConventionValidator{
		validator:      NewValidator(DefaultValidatorConfig()),
		fieldValidator: mockValidator,
		applyRulesFunc: func(ctx context.Context, reqPtr interface{}) []error { return nil },
	}

	// Create a test struct field for a []byte Body with validation tag
	type TestRequest struct {
		Body []byte `validate:"required"`
	}
	field := reflect.TypeOf(TestRequest{}).Field(0)

	// Use an invalid reflect.Value that might cause issues
	invalidValue := reflect.Value{}
	validationErrors := make(map[string][]string)

	// Call validateByteSliceField through validateSection
	err := v.validateSection(context.Background(), field, invalidValue, validationErrors)

	// This should result in a server error due to reflection issues
	if err == nil {
		t.Fatal("Expected server error due to invalid reflection value, got nil")
	}

	// Should be a server error, not validation errors
	if len(validationErrors) != 0 {
		t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
	}
}

func TestValidateByteSliceField_ComprehensivePaths(t *testing.T) {
	tests := []struct {
		name                   string
		mockError              error
		expectServerError      bool
		expectValidationErrors bool
	}{
		{
			name:                   "NonValidationError_DatabaseError",
			mockError:              errors.New("database connection timeout"),
			expectServerError:      true,
			expectValidationErrors: false,
		},
		{
			name:                   "NonValidationError_NetworkError",
			mockError:              errors.New("network unreachable"),
			expectServerError:      true,
			expectValidationErrors: false,
		},
		{
			name:                   "NonValidationError_CustomError",
			mockError:              errors.New("custom field validator error"),
			expectServerError:      true,
			expectValidationErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock validator with the test error
			mockValidator := &mockFieldValidator{
				varError: tt.mockError,
			}

			v := &ConventionValidator{
				validator:      NewValidator(DefaultValidatorConfig()),
				fieldValidator: mockValidator,
				applyRulesFunc: func(ctx context.Context, reqPtr interface{}) []error { return nil },
			}

			// Create test field and value
			type TestRequest struct {
				Body []byte `validate:"min=1"`
			}
			field := reflect.TypeOf(TestRequest{}).Field(0)
			fieldValue := reflect.ValueOf([]byte("test"))
			validationErrors := make(map[string][]string)

			// Execute the test
			err := v.validateSection(context.Background(), field, fieldValue, validationErrors)

			// Validate results
			if tt.expectServerError {
				if err == nil {
					t.Errorf("Expected server error, got nil")
				}
				if err != tt.mockError {
					t.Errorf("Expected error '%v', got '%v'", tt.mockError, err)
				}
			}

			hasValidationErrors := len(validationErrors) > 0
			if tt.expectValidationErrors && !hasValidationErrors {
				t.Errorf("Expected validation errors, got none")
			}
			if !tt.expectValidationErrors && hasValidationErrors {
				t.Errorf("Expected no validation errors, got: %v", validationErrors)
			}
		})
	}
}

// TestValidateStructField_NonValidationError tests the return validationErr path in validateStructField
func TestValidateStructField_NonValidationError(t *testing.T) {
	// Create a mock field validator that returns a non-ValidationErrors error
	serverError := errors.New("external service unavailable during struct validation")
	mockValidator := &mockFieldValidator{
		structError: serverError, // This will be returned by Struct() method
	}

	// Create a convention validator with the mock field validator
	v := &ConventionValidator{
		validator:      NewValidator(DefaultValidatorConfig()),
		fieldValidator: mockValidator,
		applyRulesFunc: func(ctx context.Context, reqPtr interface{}) []error { return nil },
	}

	// Create a test struct field for a regular struct (not []byte)
	type QuerySection struct {
		Limit int `validate:"min=1"`
	}
	type TestRequest struct {
		Query QuerySection
	}
	field := reflect.TypeOf(TestRequest{}).Field(0)
	fieldValue := reflect.ValueOf(QuerySection{Limit: 10})
	validationErrors := make(map[string][]string)

	// Call validateStructField through validateSection
	err := v.validateSection(context.Background(), field, fieldValue, validationErrors)

	// Verify that the server error was returned directly
	if err == nil {
		t.Fatal("Expected server error to be returned, got nil")
	}

	if err != serverError {
		t.Errorf("Expected server error '%v', got '%v'", serverError, err)
	}

	// Validation errors map should remain empty since this is a server error
	if len(validationErrors) != 0 {
		t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
	}
}

func TestValidateStructField_StructValidationServerErrors(t *testing.T) {
	tests := []struct {
		name                   string
		mockError              error
		expectServerError      bool
		expectValidationErrors bool
	}{
		{
			name:                   "StructValidation_DatabaseError",
			mockError:              errors.New("database connection failed during struct validation"),
			expectServerError:      true,
			expectValidationErrors: false,
		},
		{
			name:                   "StructValidation_ExternalServiceError",
			mockError:              errors.New("external validation service timeout"),
			expectServerError:      true,
			expectValidationErrors: false,
		},
		{
			name:                   "StructValidation_CustomValidatorError",
			mockError:              errors.New("custom struct validator panic recovery"),
			expectServerError:      true,
			expectValidationErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock validator with the test error for Struct() method
			mockValidator := &mockFieldValidator{
				structError: tt.mockError,
			}

			v := &ConventionValidator{
				validator:      NewValidator(DefaultValidatorConfig()),
				fieldValidator: mockValidator,
				applyRulesFunc: func(ctx context.Context, reqPtr interface{}) []error { return nil },
			}

			// Create test field and value for struct validation
			type PathSection struct {
				ID string `validate:"required"`
			}
			type TestRequest struct {
				Path PathSection
			}
			field := reflect.TypeOf(TestRequest{}).Field(0)
			fieldValue := reflect.ValueOf(PathSection{ID: "123"})
			validationErrors := make(map[string][]string)

			// Execute the test
			err := v.validateSection(context.Background(), field, fieldValue, validationErrors)

			// Validate results
			if tt.expectServerError {
				if err == nil {
					t.Errorf("Expected server error, got nil")
				}
				if err != tt.mockError {
					t.Errorf("Expected error '%v', got '%v'", tt.mockError, err)
				}
			}

			hasValidationErrors := len(validationErrors) > 0
			if tt.expectValidationErrors && !hasValidationErrors {
				t.Errorf("Expected validation errors, got none")
			}
			if !tt.expectValidationErrors && hasValidationErrors {
				t.Errorf("Expected no validation errors, got: %v", validationErrors)
			}
		})
	}
}

func TestValidateStructField_PanicRecovery(t *testing.T) {
	// Create a mock field validator that will cause the struct validation to panic
	mockValidator := &mockFieldValidator{
		structError: nil, // Will never be reached due to panic in reflection
	}

	v := &ConventionValidator{
		validator:      NewValidator(DefaultValidatorConfig()),
		fieldValidator: mockValidator,
		applyRulesFunc: func(ctx context.Context, reqPtr interface{}) []error { return nil },
	}

	// Create a test struct field with an invalid reflect.Value that causes panic
	type TestRequest struct {
		Query interface{} // Using interface{} to test edge cases
	}
	field := reflect.TypeOf(TestRequest{}).Field(0)

	// Use an invalid reflect.Value that might cause panic during validation
	invalidValue := reflect.Value{}
	validationErrors := make(map[string][]string)

	// Call validateStructField through validateSection
	err := v.validateSection(context.Background(), field, invalidValue, validationErrors)

	// This should result in a server error due to panic recovery or reflection issues
	if err == nil {
		t.Fatal("Expected server error due to invalid reflection value, got nil")
	}

	// Should be a server error, not validation errors
	if len(validationErrors) != 0 {
		t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
	}
}
