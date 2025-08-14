package api

import (
	"context"
	"errors"
	"testing"
)

// Test request that implements Validator but returns ValidationError
type TestRequestWithValidationError struct {
	Body struct {
		Name string `gork:"name"`
	}
}

func (r *TestRequestWithValidationError) Validate() error {
	if r.Body.Name == "invalid" {
		return &RequestValidationError{
			Errors: []string{"name cannot be 'invalid'"},
		}
	}
	return nil
}

// Test request that implements Validator but returns non-ValidationError (server error)
type TestRequestWithServerError struct {
	Body struct {
		Data string `gork:"data"`
	}
}

func (r *TestRequestWithServerError) Validate() error {
	if r.Body.Data == "cause-server-error" {
		// Return a regular error (not ValidationError) - this should be treated as server error
		return errors.New("internal validation failed - database connection lost")
	}
	return nil
}

// Test request that implements Validator and succeeds
type TestRequestValidationSuccess struct {
	Body struct {
		Valid string `gork:"valid"`
	}
}

func (r *TestRequestValidationSuccess) Validate() error {
	return nil // Always valid
}

// Test request that does NOT implement Validator
type TestRequestNoValidator struct {
	Body struct {
		Field string `gork:"field"`
	}
}

func TestValidateRequestLevel(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("request without validator interface", func(t *testing.T) {
		req := &TestRequestNoValidator{
			Body: struct {
				Field string `gork:"field"`
			}{Field: "test"},
		}
		validationErrors := make(map[string][]string)

		err := validator.validateRequestLevel(context.Background(), req, validationErrors)

		if err != nil {
			t.Errorf("Expected no error for request without validator, got: %v", err)
		}
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors, got: %v", validationErrors)
		}
	})

	t.Run("request with successful validation", func(t *testing.T) {
		req := &TestRequestValidationSuccess{
			Body: struct {
				Valid string `gork:"valid"`
			}{Valid: "good-data"},
		}
		validationErrors := make(map[string][]string)

		err := validator.validateRequestLevel(context.Background(), req, validationErrors)

		if err != nil {
			t.Errorf("Expected no error for successful validation, got: %v", err)
		}
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors, got: %v", validationErrors)
		}
	})

	t.Run("request with validation error (client error)", func(t *testing.T) {
		req := &TestRequestWithValidationError{
			Body: struct {
				Name string `gork:"name"`
			}{Name: "invalid"},
		}
		validationErrors := make(map[string][]string)

		err := validator.validateRequestLevel(context.Background(), req, validationErrors)

		// Should not return error (validation errors are added to map, not returned)
		if err != nil {
			t.Errorf("Expected no error return for validation errors, got: %v", err)
		}

		// Should add validation errors to the map
		if len(validationErrors["request"]) == 0 {
			t.Error("Expected validation errors to be added to request level")
		} else if validationErrors["request"][0] != "name cannot be 'invalid'" {
			t.Errorf("Expected specific validation error, got: %v", validationErrors["request"])
		}
	})

	t.Run("request with server error (non-ValidationError)", func(t *testing.T) {
		req := &TestRequestWithServerError{
			Body: struct {
				Data string `gork:"data"`
			}{Data: "cause-server-error"},
		}
		validationErrors := make(map[string][]string)

		// This tests the uncovered line: return err for non-ValidationError types
		err := validator.validateRequestLevel(context.Background(), req, validationErrors)

		// Should return the server error directly
		if err == nil {
			t.Error("Expected server error to be returned")
		} else if err.Error() != "internal validation failed - database connection lost" {
			t.Errorf("Expected specific server error, got: %v", err)
		}

		// Should not add anything to validation errors map (server errors are returned, not collected)
		if len(validationErrors) != 0 {
			t.Errorf("Expected no validation errors in map for server error, got: %v", validationErrors)
		}
	})
}
