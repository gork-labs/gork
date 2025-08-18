package api

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
)

// MockFieldValidator implements FieldValidator for testing
type MockFieldValidator struct {
	VarFunc    func(field interface{}, tag string) error
	StructFunc func(s interface{}) error
}

func (m *MockFieldValidator) Var(field interface{}, tag string) error {
	if m.VarFunc != nil {
		return m.VarFunc(field, tag)
	}
	return nil
}

func (m *MockFieldValidator) Struct(s interface{}) error {
	if m.StructFunc != nil {
		return m.StructFunc(s)
	}
	return nil
}

// TestValidateSection_RefactoredWithMocks tests the refactored validateSection function
// using dependency injection to achieve 100% coverage
func TestValidateSection_RefactoredWithMocks(t *testing.T) {
	t.Run("byte body field - validator.Var returns non-ValidationError", func(t *testing.T) {
		// Create a mock that returns a non-ValidationError from Var()
		mockValidator := &MockFieldValidator{
			VarFunc: func(field interface{}, tag string) error {
				// Return a non-ValidationError (server error)
				return errors.New("server error from validator.Var")
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type TestRequest struct {
			Body []byte `validate:"min=1"`
		}

		reqType := reflect.TypeOf(TestRequest{})
		bodyField := reqType.Field(0)
		bodyValue := reflect.ValueOf(TestRequest{Body: []byte("test")}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), bodyField, bodyValue, validationErrors)

		// Should return the server error
		if err == nil {
			t.Error("Expected server error from validator.Var, got nil")
		}

		if err.Error() != "server error from validator.Var" {
			t.Errorf("Expected 'server error from validator.Var', got: %v", err)
		}

		// Validation errors should be empty for server errors
		if len(validationErrors) != 0 {
			t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
		}
	})

	t.Run("regular struct field - validator.Struct returns non-ValidationError", func(t *testing.T) {
		// Create a mock that returns a non-ValidationError from Struct()
		mockValidator := &MockFieldValidator{
			StructFunc: func(s interface{}) error {
				// Return a non-ValidationError (server error)
				return errors.New("server error from validator.Struct")
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type QuerySection struct {
			Limit int `validate:"min=1"`
		}
		type TestRequest struct {
			Query QuerySection
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: QuerySection{Limit: 10}}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)

		// Should return the server error
		if err == nil {
			t.Error("Expected server error from validator.Struct, got nil")
		}

		if err.Error() != "server error from validator.Struct" {
			t.Errorf("Expected 'server error from validator.Struct', got: %v", err)
		}

		// Validation errors should be empty for server errors
		if len(validationErrors) != 0 {
			t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
		}
	})

	t.Run("panic recovery during struct validation", func(t *testing.T) {
		// Create a mock that panics during Struct()
		mockValidator := &MockFieldValidator{
			StructFunc: func(s interface{}) error {
				panic("validation panic occurred")
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type QuerySection struct {
			Limit int `validate:"min=1"`
		}
		type TestRequest struct {
			Query QuerySection
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: QuerySection{Limit: 10}}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)

		// Should recover from panic and return error
		if err == nil {
			t.Error("Expected panic recovery error, got nil")
		}

		expectedError := "validation panic: validation panic occurred"
		if err.Error() != expectedError {
			t.Errorf("Expected '%s', got: %v", expectedError, err)
		}

		// Validation errors should be empty for panic errors
		if len(validationErrors) != 0 {
			t.Errorf("Expected empty validation errors for panic error, got: %v", validationErrors)
		}
	})

	t.Run("successful byte body validation with ValidationErrors", func(t *testing.T) {
		// Create a mock that returns ValidationErrors from Var()
		mockValidator := &MockFieldValidator{
			VarFunc: func(field interface{}, tag string) error {
				// Create a mock ValidationError
				return &validator.InvalidValidationError{Type: reflect.TypeOf("")}
			},
		}

		// We need to override the mock to return actual ValidationErrors
		realValidator := NewValidator(DefaultValidatorConfig())
		mockValidator.VarFunc = func(field interface{}, tag string) error {
			// Use real validator but with invalid data to trigger ValidationErrors
			return realValidator.Var([]byte{}, "min=1") // Empty slice should fail min=1
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type TestRequest struct {
			Body []byte `validate:"min=1"`
		}

		reqType := reflect.TypeOf(TestRequest{})
		bodyField := reqType.Field(0)
		bodyValue := reflect.ValueOf(TestRequest{Body: []byte{}}).Field(0) // Empty slice

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), bodyField, bodyValue, validationErrors)
		// Should not return error (validation errors are collected)
		if err != nil {
			t.Errorf("Expected no error for ValidationErrors, got: %v", err)
		}

		// Should have validation errors
		if len(validationErrors) == 0 {
			t.Error("Expected validation errors to be collected")
		}

		if _, exists := validationErrors["body"]; !exists {
			t.Error("Expected 'body' section in validation errors")
		}
	})

	t.Run("successful struct validation with ValidationErrors", func(t *testing.T) {
		// Use real validator to generate ValidationErrors
		realValidator := NewValidator(DefaultValidatorConfig())
		mockValidator := &MockFieldValidator{
			StructFunc: func(s interface{}) error {
				// Use real validator with invalid data to trigger ValidationErrors
				type InvalidStruct struct {
					Limit int `validate:"min=10"`
				}
				return realValidator.Struct(&InvalidStruct{Limit: 5}) // Should fail min=10
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type QuerySection struct {
			Limit int `validate:"min=1"`
		}
		type TestRequest struct {
			Query QuerySection
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: QuerySection{Limit: 10}}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)
		// Should not return error (validation errors are collected)
		if err != nil {
			t.Errorf("Expected no error for ValidationErrors, got: %v", err)
		}

		// Should have validation errors
		if len(validationErrors) == 0 {
			t.Error("Expected validation errors to be collected")
		}
	})

	t.Run("byte body field without validation tag", func(t *testing.T) {
		// Mock should not be called for fields without validation tags
		mockValidator := &MockFieldValidator{
			VarFunc: func(field interface{}, tag string) error {
				t.Error("VarFunc should not be called for fields without validation tags")
				return nil
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type TestRequest struct {
			Body []byte // No validate tag
		}

		reqType := reflect.TypeOf(TestRequest{})
		bodyField := reqType.Field(0)
		bodyValue := reflect.ValueOf(TestRequest{Body: []byte("test")}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), bodyField, bodyValue, validationErrors)
		// Should not return error and not call validator
		if err != nil {
			t.Errorf("Expected no error for field without validation tag, got: %v", err)
		}

		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors, got: %v", validationErrors)
		}
	})

	t.Run("successful validation - no errors", func(t *testing.T) {
		// Mock that returns no errors
		mockValidator := &MockFieldValidator{
			StructFunc: func(s interface{}) error {
				return nil // No validation errors
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type QuerySection struct {
			Limit int `validate:"min=1"`
		}
		type TestRequest struct {
			Query QuerySection
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: QuerySection{Limit: 10}}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)
		// Should not return error
		if err != nil {
			t.Errorf("Expected no error for successful validation, got: %v", err)
		}

		// Should have no validation errors
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors, got: %v", validationErrors)
		}
	})
}

// CustomValidationSection implements ContextValidator for testing
type CustomValidationSection struct {
	Value       string
	ShouldError bool
	ServerError bool
}

func (c *CustomValidationSection) Validate(ctx context.Context) error {
	if c.ServerError {
		return errors.New("custom validation server error")
	}
	if c.ShouldError {
		return &RequestValidationError{Errors: []string{"custom validation failed"}}
	}
	return nil
}

// TestValidateSection_CustomValidationCoverage tests custom validation paths
func TestValidateSection_CustomValidationCoverage(t *testing.T) {
	t.Run("custom validation returns server error", func(t *testing.T) {
		mockValidator := &MockFieldValidator{
			StructFunc: func(s interface{}) error {
				return nil // No field validation errors
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type TestRequest struct {
			Query *CustomValidationSection
		}

		customSection := &CustomValidationSection{
			Value:       "test",
			ServerError: true,
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: customSection}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)

		// Should return the custom validation server error
		if err == nil {
			t.Error("Expected server error from custom validation, got nil")
		}

		if err.Error() != "custom validation server error" {
			t.Errorf("Expected 'custom validation server error', got: %v", err)
		}

		// Validation errors should be empty for server errors
		if len(validationErrors) != 0 {
			t.Errorf("Expected empty validation errors for server error, got: %v", validationErrors)
		}
	})

	t.Run("custom validation returns validation errors", func(t *testing.T) {
		mockValidator := &MockFieldValidator{
			StructFunc: func(s interface{}) error {
				return nil // No field validation errors
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type TestRequest struct {
			Query *CustomValidationSection
		}

		customSection := &CustomValidationSection{
			Value:       "test",
			ShouldError: true,
			ServerError: false,
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: customSection}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)
		// Should not return error (validation errors are collected)
		if err != nil {
			t.Errorf("Expected no error for custom validation errors, got: %v", err)
		}

		// Should have validation errors
		if len(validationErrors) == 0 {
			t.Error("Expected validation errors to be collected")
		}

		if _, exists := validationErrors["query"]; !exists {
			t.Error("Expected 'query' section in validation errors")
		}
	})

	t.Run("custom validation succeeds", func(t *testing.T) {
		mockValidator := &MockFieldValidator{
			StructFunc: func(s interface{}) error {
				return nil // No field validation errors
			},
		}

		v := NewConventionValidatorWithFieldValidator(mockValidator)

		type TestRequest struct {
			Query *CustomValidationSection
		}

		customSection := &CustomValidationSection{
			Value:       "test",
			ShouldError: false,
			ServerError: false,
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: customSection}).Field(0)

		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)
		// Should not return error
		if err != nil {
			t.Errorf("Expected no error for successful custom validation, got: %v", err)
		}

		// Should have no validation errors
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors, got: %v", validationErrors)
		}
	})
}

// TestNewConventionValidatorWithFieldValidator tests the new constructor
func TestNewConventionValidatorWithFieldValidator(t *testing.T) {
	mockValidator := &MockFieldValidator{}

	v := NewConventionValidatorWithFieldValidator(mockValidator)

	if v == nil {
		t.Error("Expected non-nil ConventionValidator")
	}

	if v.fieldValidator != mockValidator {
		t.Error("Expected fieldValidator to be set to the provided mock")
	}

	if v.validator == nil {
		t.Error("Expected validator to be initialized")
	}
}

// TestGoPlaygroundValidator tests the wrapper implementation
func TestGoPlaygroundValidator(t *testing.T) {
	realValidator := NewValidator(DefaultValidatorConfig())
	wrapper := &GoPlaygroundValidator{validator: realValidator}

	t.Run("Var method", func(t *testing.T) {
		err := wrapper.Var("test", "min=5")
		if err == nil {
			t.Error("Expected validation error for string shorter than min=5")
		}

		err = wrapper.Var("testing", "min=5")
		if err != nil {
			t.Errorf("Expected no error for valid string, got: %v", err)
		}
	})

	t.Run("Struct method", func(t *testing.T) {
		type TestStruct struct {
			Name string `validate:"required"`
		}

		err := wrapper.Struct(&TestStruct{})
		if err == nil {
			t.Error("Expected validation error for missing required field")
		}

		err = wrapper.Struct(&TestStruct{Name: "test"})
		if err != nil {
			t.Errorf("Expected no error for valid struct, got: %v", err)
		}
	})
}
