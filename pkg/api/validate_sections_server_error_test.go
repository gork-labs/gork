package api

import (
	"context"
	"errors"
	"testing"
)

// Test section that implements Validator but returns server error
type TestSectionWithServerError struct {
	Name string `gork:"name"`
}

func (s TestSectionWithServerError) Validate() error {
	// Return a non-ValidationError (regular error) - this should be treated as server error
	return errors.New("database connection failed during section validation")
}

// Test request with section that causes server error
type TestRequestWithSectionServerError2 struct {
	Body TestSectionWithServerError
}

// Test section that implements Validator and returns ValidationError (not server error)
type ValidatorSectionWithValidationError struct {
	Data string `gork:"data"`
}

func (s ValidatorSectionWithValidationError) Validate() error {
	if s.Data == "invalid" {
		return &BodyValidationError{
			Errors: []string{"data is invalid"},
		}
	}
	return nil
}

func TestValidateSectionsServerError(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("server error from section-level validation", func(t *testing.T) {
		// This tests the uncovered lines 121-123 in ValidateRequest:
		// if err := v.validateSections(...); err != nil { return err }

		req := &TestRequestWithSectionServerError2{
			Body: TestSectionWithServerError{
				Name: "test",
			},
		}

		err := validator.ValidateRequest(context.Background(), req)

		// Should return the server error directly from validateSections
		if err == nil {
			t.Error("Expected server error from validateSections")
		} else if err.Error() != "database connection failed during section validation" {
			t.Errorf("Expected specific server error, got: %v", err)
		}
	})

	t.Run("normal section validation for comparison", func(t *testing.T) {
		// Test a normal case to ensure our test setup is working
		type NormalSection struct {
			Name string `gork:"name" validate:"required"`
		}

		type TestRequestNormal struct {
			Body NormalSection
		}

		req := &TestRequestNormal{
			Body: NormalSection{
				Name: "valid",
			},
		}

		err := validator.ValidateRequest(context.Background(), req)
		if err != nil {
			t.Errorf("Expected normal validation to succeed, got: %v", err)
		}
	})

	t.Run("section with validation error (not server error)", func(t *testing.T) {
		// Test with a section that has a validation error from go-playground/validator
		type TestSectionWithFieldValidation struct {
			Name string `gork:"name" validate:"required"`
		}

		type TestRequestWithFieldValidationError struct {
			Body TestSectionWithFieldValidation
		}

		req := &TestRequestWithFieldValidationError{
			Body: TestSectionWithFieldValidation{
				Name: "", // Invalid: required field is empty
			},
		}

		err := validator.ValidateRequest(context.Background(), req)

		// Should return ValidationErrorResponse, not server error
		if err == nil {
			t.Error("Expected validation error")
		} else {
			if valErr, ok := err.(*ValidationErrorResponse); ok {
				t.Logf("Got expected validation error: %v", valErr)
			} else {
				t.Errorf("Expected ValidationErrorResponse, got: %T %v", err, err)
			}
		}
	})
}
