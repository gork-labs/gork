package api

import (
	"context"
	"errors"
	"testing"
)

// Test request that implements ContextValidator but returns server error
type ContextValidatorServerErrorRequest struct {
	Headers struct {
		TestHeader string `gork:"X-Test-Header"`
	}
	Body []byte
}

func (ContextValidatorServerErrorRequest) WebhookRequest() {}

func (r *ContextValidatorServerErrorRequest) Validate(ctx context.Context) error {
	// Return a non-ValidationError (server error)
	return errors.New("database connection failed during webhook validation")
}

// Test section that implements ContextValidator but returns server error
type ContextValidatorServerErrorSection struct {
	Data string `gork:"data"`
}

func (s ContextValidatorServerErrorSection) Validate(ctx context.Context) error {
	// Return a non-ValidationError (server error)
	return errors.New("external service unavailable during section validation")
}

func TestContextValidator_ServerErrorHandling(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("context validator returning server error at request level", func(t *testing.T) {
		req := &ContextValidatorServerErrorRequest{
			Headers: struct {
				TestHeader string `gork:"X-Test-Header"`
			}{
				TestHeader: "test-value",
			},
			Body: []byte(`{"test": "data"}`),
		}

		err := validator.ValidateRequest(context.Background(), req)

		if err == nil {
			t.Error("expected server error from context validator")
		}

		// Should be a server error, not a validation error
		if IsValidationError(err) {
			t.Error("expected server error, got validation error")
		}

		if err.Error() != "database connection failed during webhook validation" {
			t.Errorf("expected specific server error message, got %v", err)
		}
	})

	t.Run("context validator returning server error at section level", func(t *testing.T) {
		// Create a request with a section that implements ContextValidator
		type TestRequestWithContextValidatorSection struct {
			Headers ContextValidatorServerErrorSection
		}

		req := &TestRequestWithContextValidatorSection{
			Headers: ContextValidatorServerErrorSection{
				Data: "test-data",
			},
		}

		err := validator.ValidateRequest(context.Background(), req)

		if err == nil {
			t.Error("expected server error from context validator section")
		}

		// Should be a server error, not a validation error
		if IsValidationError(err) {
			t.Error("expected server error, got validation error")
		}

		if err.Error() != "external service unavailable during section validation" {
			t.Errorf("expected specific server error message, got %v", err)
		}
	})
}
