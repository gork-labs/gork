package api

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestValidateSection_ByteBodyValidationServerError tests the uncovered path where
// validator.Var() returns a non-ValidationError for []byte Body fields
func TestValidateSection_ByteBodyValidationServerError(t *testing.T) {
	v := NewConventionValidator()

	// Create a struct with []byte Body field that has validation tags
	type TestRequest struct {
		Body []byte `validate:"min=1"`
	}

	// Get the field info for the Body field
	reqType := reflect.TypeOf(TestRequest{})
	bodyField := reqType.Field(0)

	// Try to trigger InvalidValidationError by passing a nil interface{} to validator.Var()
	// This should constitute "bad validation input"
	nilInterface := (*interface{})(nil)
	fieldValue := reflect.ValueOf(nilInterface)

	validationErrors := make(map[string][]string)

	// This should trigger the []byte Body path and call validator.Var() with nil interface
	// which should return InvalidValidationError (a non-ValidationError type)
	err := v.validateSection(context.Background(), bodyField, fieldValue, validationErrors)
	if err != nil {
		t.Logf("Got error from validator.Var() with nil interface: %v (type: %T)", err, err)

		// Check if it's a ValidationError or something else
		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Logf("Successfully triggered non-ValidationError: %v", err)
			// This is the path we're trying to test - server error from validator.Var()

			// Validation errors map should be empty since we got a server error
			if len(validationErrors) != 0 {
				t.Errorf("Expected empty validation errors map, got: %v", validationErrors)
			}
			return
		} else {
			t.Logf("Got ValidationError (not the path we want): %v", valErr.GetErrors())
		}
	}

	// Try another approach - pass an invalid type that might trigger InvalidValidationError
	// Create a reflect.Value of a function type (functions can't be validated)
	funcValue := reflect.ValueOf(func() {})

	err = v.validateSection(context.Background(), bodyField, funcValue, validationErrors)
	if err != nil {
		t.Logf("Got error from validator.Var() with function type: %v (type: %T)", err, err)

		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Logf("Successfully triggered non-ValidationError with function type: %v", err)
			return
		}
	}

	// Try with a channel type (another type that might cause InvalidValidationError)
	chanValue := reflect.ValueOf(make(chan int))

	err = v.validateSection(context.Background(), bodyField, chanValue, validationErrors)
	if err != nil {
		t.Logf("Got error from validator.Var() with channel type: %v (type: %T)", err, err)

		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Logf("Successfully triggered non-ValidationError with channel type: %v", err)
			return
		}
	}

	// If we get here, none of our approaches worked
	t.Error("Could not trigger InvalidValidationError from validator.Var() - this should not happen")
}

// TestValidateSection_StructValidationServerError tests the uncovered path where
// validator.Struct() returns a non-ValidationError for regular struct fields
func TestValidateSection_StructValidationServerError(t *testing.T) {
	v := NewConventionValidator()

	// Create a struct with a regular struct field (not []byte Body)
	type QuerySection struct {
		Limit int `validate:"min=1"`
	}

	type TestRequest struct {
		Query QuerySection
	}

	// Get the field info for the Query field
	reqType := reflect.TypeOf(TestRequest{})
	queryField := reqType.Field(0)

	// Try to trigger InvalidValidationError by passing a nil interface{} to validator.Struct()
	// This should constitute "bad validation input" similar to the Var() case
	nilInterface := (*interface{})(nil)
	fieldValue := reflect.ValueOf(nilInterface)

	validationErrors := make(map[string][]string)

	// This should trigger the regular struct validation path and call validator.Struct()
	// with nil interface, which should return InvalidValidationError
	err := v.validateSection(context.Background(), queryField, fieldValue, validationErrors)
	if err != nil {
		t.Logf("Got error from validator.Struct() with nil interface: %v (type: %T)", err, err)

		// Check if it's a ValidationError or something else
		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Logf("Successfully triggered non-ValidationError: %v", err)
			// This is the path we're trying to test - server error from validator.Struct()

			// Validation errors map should be empty since we got a server error
			if len(validationErrors) != 0 {
				t.Errorf("Expected empty validation errors map, got: %v", validationErrors)
			}
			return
		} else {
			t.Logf("Got ValidationError (not the path we want): %v", valErr.GetErrors())
		}
	}

	// Try another approach - pass a non-struct type to validator.Struct()
	// This should definitely trigger InvalidValidationError
	stringValue := reflect.ValueOf("not a struct")

	err = v.validateSection(context.Background(), queryField, stringValue, validationErrors)
	if err != nil {
		t.Logf("Got error from validator.Struct() with string type: %v (type: %T)", err, err)

		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Logf("Successfully triggered non-ValidationError with string type: %v", err)
			return
		}
	}

	// Try with a function type (another invalid type for struct validation)
	funcValue := reflect.ValueOf(func() {})

	err = v.validateSection(context.Background(), queryField, funcValue, validationErrors)
	if err != nil {
		t.Logf("Got error from validator.Struct() with function type: %v (type: %T)", err, err)

		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Logf("Successfully triggered non-ValidationError with function type: %v", err)
			return
		}
	}

	// If we get here, none of our approaches worked
	t.Error("Could not trigger InvalidValidationError from validator.Struct() - this should not happen")
}

// ServerErrorSection implements ContextValidator and returns a server error
type ServerErrorSection struct {
	Data string `gork:"data"`
}

// Validate implements ContextValidator interface to return server error
func (s ServerErrorSection) Validate(ctx context.Context) error {
	// Return a non-ValidationError (server error)
	return errors.New("database connection failed during section validation")
}

// ServerErrorSectionRegular implements Validator and returns a server error
type ServerErrorSectionRegular struct {
	Data string `gork:"data"`
}

// Validate implements Validator interface to return server error
func (s ServerErrorSectionRegular) Validate() error {
	// Return a non-ValidationError (server error)
	return errors.New("external service unavailable during section validation")
}

// ValidationErrorSection implements ContextValidator and returns a ValidationError
type ValidationErrorSection struct {
	Data string `gork:"data"`
}

// Validate implements ContextValidator interface to return ValidationError
func (s ValidationErrorSection) Validate(ctx context.Context) error {
	return &RequestValidationError{Errors: []string{"custom validation failed"}}
}

// TestValidateSection_CustomValidationServerError tests the uncovered path where
// invokeCustomValidation() returns a server error for section-level custom validation
func TestValidateSection_CustomValidationServerError(t *testing.T) {
	v := NewConventionValidator()

	type TestRequest struct {
		Query ServerErrorSection
	}

	// Get the field info for the Query field
	reqType := reflect.TypeOf(TestRequest{})
	queryField := reqType.Field(0)

	// Create a field value with the custom validator section
	reqValue := reflect.ValueOf(TestRequest{
		Query: ServerErrorSection{Data: "test-data"},
	})
	queryValue := reqValue.Field(0)

	validationErrors := make(map[string][]string)

	// This should trigger the custom validation path in validateSection
	// where invokeCustomValidation() returns a server error
	err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)

	// Should return a server error (not nil)
	if err == nil {
		t.Error("Expected server error from custom validation, got nil")
	}

	// Should not be a ValidationError
	var valErr ValidationError
	if errors.As(err, &valErr) {
		t.Error("Expected non-ValidationError, got ValidationError")
	}

	// Should be our specific server error
	expectedError := "database connection failed during section validation"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got: %v", expectedError, err)
	}

	// Validation errors map should be empty since we got a server error
	if len(validationErrors) != 0 {
		t.Errorf("Expected empty validation errors map, got: %v", validationErrors)
	}
}

// TestValidateSection_CustomValidationServerErrorRegularValidator tests server error
// from regular Validator interface (not ContextValidator)
func TestValidateSection_CustomValidationServerErrorRegularValidator(t *testing.T) {
	v := NewConventionValidator()

	type TestRequest struct {
		Headers ServerErrorSectionRegular
	}

	// Get the field info for the Headers field
	reqType := reflect.TypeOf(TestRequest{})
	headersField := reqType.Field(0)

	// Create a field value with the custom validator section
	reqValue := reflect.ValueOf(TestRequest{
		Headers: ServerErrorSectionRegular{Data: "test-data"},
	})
	headersValue := reqValue.Field(0)

	validationErrors := make(map[string][]string)

	// This should trigger the custom validation path in validateSection
	// where invokeCustomValidation() returns a server error from regular Validator
	err := v.validateSection(context.Background(), headersField, headersValue, validationErrors)

	// Should return a server error (not nil)
	if err == nil {
		t.Error("Expected server error from regular custom validation, got nil")
	}

	// Should not be a ValidationError
	var valErr ValidationError
	if errors.As(err, &valErr) {
		t.Error("Expected non-ValidationError, got ValidationError")
	}

	// Should be our specific server error
	expectedError := "external service unavailable during section validation"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got: %v", expectedError, err)
	}

	// Validation errors map should be empty since we got a server error
	if len(validationErrors) != 0 {
		t.Errorf("Expected empty validation errors map, got: %v", validationErrors)
	}
}

// TestValidateSection_AllConditionalBranches ensures all conditional branches are covered
func TestValidateSection_AllConditionalBranches(t *testing.T) {
	v := NewConventionValidator()

	t.Run("[]byte Body field with no validation tag", func(t *testing.T) {
		// Test the []byte Body path where tag is empty (tag == "")
		type TestRequest struct {
			Body []byte // No validate tag
		}

		reqType := reflect.TypeOf(TestRequest{})
		bodyField := reqType.Field(0)
		bodyValue := reflect.ValueOf(TestRequest{Body: []byte("test")}).Field(0)
		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), bodyField, bodyValue, validationErrors)
		// Should not return error and should not add validation errors
		if err != nil {
			t.Errorf("Expected no error for []byte Body without validation tag, got: %v", err)
		}
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors, got: %v", validationErrors)
		}
	})

	t.Run("[]byte Body field with validation tag - success case", func(t *testing.T) {
		// Test the []byte Body path where validation succeeds
		type TestRequest struct {
			Body []byte `validate:"min=1"`
		}

		reqType := reflect.TypeOf(TestRequest{})
		bodyField := reqType.Field(0)
		bodyValue := reflect.ValueOf(TestRequest{Body: []byte("test")}).Field(0)
		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), bodyField, bodyValue, validationErrors)
		// Should not return error and should not add validation errors (validation passes)
		if err != nil {
			t.Errorf("Expected no error for valid []byte Body, got: %v", err)
		}
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors for valid data, got: %v", validationErrors)
		}
	})

	t.Run("[]byte Body field with validation tag - validation failure", func(t *testing.T) {
		// Test the []byte Body path where validation fails (ValidationError path)
		type TestRequest struct {
			Body []byte `validate:"min=5"`
		}

		reqType := reflect.TypeOf(TestRequest{})
		bodyField := reqType.Field(0)
		bodyValue := reflect.ValueOf(TestRequest{Body: []byte("hi")}).Field(0) // Too short
		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), bodyField, bodyValue, validationErrors)
		// Should not return error but should add validation errors
		if err != nil {
			t.Errorf("Expected no server error for validation failure, got: %v", err)
		}
		if len(validationErrors["body"]) == 0 {
			t.Error("Expected validation errors for body section")
		}
	})

	t.Run("regular struct field - success case", func(t *testing.T) {
		// Test the regular struct validation path where validation succeeds
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
		// Should not return error and should not add validation errors
		if err != nil {
			t.Errorf("Expected no error for valid struct, got: %v", err)
		}
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors for valid struct, got: %v", validationErrors)
		}
	})

	t.Run("regular struct field - validation failure", func(t *testing.T) {
		// Test the regular struct validation path where validation fails (ValidationError path)
		type QuerySection struct {
			Limit int `validate:"min=10"`
		}
		type TestRequest struct {
			Query QuerySection
		}

		reqType := reflect.TypeOf(TestRequest{})
		queryField := reqType.Field(0)
		queryValue := reflect.ValueOf(TestRequest{Query: QuerySection{Limit: 5}}).Field(0) // Too small
		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), queryField, queryValue, validationErrors)
		// Should not return error but should add validation errors
		if err != nil {
			t.Errorf("Expected no server error for validation failure, got: %v", err)
		}
		if len(validationErrors) == 0 {
			t.Error("Expected validation errors for query section")
		}
	})

	t.Run("custom validation - success case", func(t *testing.T) {
		// Test custom validation path where validation succeeds
		// Use an existing type that doesn't implement any validator interface
		type SuccessSection struct {
			Data string `gork:"data"`
		}

		type TestRequest struct {
			Headers SuccessSection
		}

		reqType := reflect.TypeOf(TestRequest{})
		headersField := reqType.Field(0)
		headersValue := reflect.ValueOf(TestRequest{Headers: SuccessSection{Data: "test"}}).Field(0)
		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), headersField, headersValue, validationErrors)
		// Should not return error and should not add validation errors
		if err != nil {
			t.Errorf("Expected no error for successful custom validation, got: %v", err)
		}
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors for successful custom validation, got: %v", validationErrors)
		}
	})

	t.Run("custom validation - validation error case", func(t *testing.T) {
		// Test custom validation path where validation returns ValidationError
		// Use the ValidationErrorSection type defined at package level

		type TestRequest struct {
			Cookies ValidationErrorSection
		}

		reqType := reflect.TypeOf(TestRequest{})
		cookiesField := reqType.Field(0)
		cookiesValue := reflect.ValueOf(TestRequest{Cookies: ValidationErrorSection{Data: "test"}}).Field(0)
		validationErrors := make(map[string][]string)

		err := v.validateSection(context.Background(), cookiesField, cookiesValue, validationErrors)
		// Should not return error but should add validation errors
		if err != nil {
			t.Errorf("Expected no server error for validation error, got: %v", err)
		}
		if len(validationErrors["cookies"]) == 0 {
			t.Error("Expected validation errors for cookies section")
		}
	})
}

// TestValidateSection_PanicRecovery tests the panic recovery path in validateSection
func TestValidateSection_PanicRecovery(t *testing.T) {
	v := NewConventionValidator()

	// Create a struct field that will cause a panic when validator.Struct() is called
	type TestRequest struct {
		Query interface{} // interface{} type
	}

	// Get the field info for the Query field
	reqType := reflect.TypeOf(TestRequest{})
	queryField := reqType.Field(0)

	// Create a field value that will cause a panic when Interface() is called
	// We need to create a reflect.Value that will panic when Interface() is called
	// One way is to create an invalid reflect.Value

	// Create a zero Value which should cause a panic when Interface() is called
	var invalidValue reflect.Value // This is a zero Value

	validationErrors := make(map[string][]string)

	// This should trigger the panic recovery path in validateSection
	// The defer function should catch the panic and return a formatted error
	err := v.validateSection(context.Background(), queryField, invalidValue, validationErrors)

	// Should return a panic error (not nil)
	if err == nil {
		t.Error("Expected panic recovery error, got nil")
	}

	// Should be a formatted panic error
	if !strings.Contains(err.Error(), "validation panic:") {
		t.Errorf("Expected panic error message, got: %v", err)
	}

	// Validation errors map should be empty since we got a panic error
	if len(validationErrors) != 0 {
		t.Errorf("Expected empty validation errors map, got: %v", validationErrors)
	}
}
