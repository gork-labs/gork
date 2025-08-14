package api

import (
	"context"
	"testing"
)

// Section type that implements ContextValidator and returns a ValidationError
type sectionContextValidator struct {
	Data string `gork:"data"`
}

func (s sectionContextValidator) Validate(ctx context.Context) error {
	return &HeadersValidationError{Errors: []string{"missing X-Test header"}}
}

func TestContextValidator_SectionValidationErrorCollected(t *testing.T) {
	type Req struct {
		Headers sectionContextValidator
	}

	v := NewConventionValidator()
	err := v.ValidateRequest(context.Background(), &Req{Headers: sectionContextValidator{Data: "x"}})
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	if !IsValidationError(err) {
		t.Fatalf("expected validation error type, got %T: %v", err, err)
	}

	verr := err.(*ValidationErrorResponse)
	if _, ok := verr.Details["headers"]; !ok {
		t.Fatalf("expected section-level errors under 'headers', got: %#v", verr.Details)
	}
}
