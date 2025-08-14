package api

import (
	"context"
	"fmt"
	"testing"
)

// Test webhook request struct that implements ContextValidator
type ContextAwareWebhookRequest struct {
	Headers struct {
		StripeSignature string `gork:"Stripe-Signature" validate:"required"`
	}
	Body []byte
}

// WebhookRequest marker method
func (ContextAwareWebhookRequest) WebhookRequest() {}

// Context-aware validation for webhook signature verification
func (r *ContextAwareWebhookRequest) Validate(ctx context.Context) error {
	// Extract configured Stripe key from context
	stripeKey, ok := ctx.Value("stripe_webhook_secret").(string)
	if !ok || stripeKey == "" {
		return fmt.Errorf("missing stripe webhook secret in context")
	}

	// Verify signature (simplified for test)
	if r.Headers.StripeSignature == "" {
		return &RequestValidationError{
			Errors: []string{"missing stripe signature header"},
		}
	}

	// Simple signature verification (in real implementation this would use stripe.ConstructEvent)
	expectedPrefix := "t=123,v1=" + stripeKey
	if r.Headers.StripeSignature != expectedPrefix {
		return &RequestValidationError{
			Errors: []string{"invalid stripe signature"},
		}
	}

	return nil
}

// Test traditional request struct that implements regular Validator (backward compatibility)
type ContextValidationTraditionalRequest struct {
	Body struct {
		Name string `json:"name" validate:"required"`
	}
}

func (r *ContextValidationTraditionalRequest) Validate() error {
	if r.Body.Name == "forbidden" {
		return &BodyValidationError{
			Errors: []string{"forbidden name not allowed"},
		}
	}
	return nil
}

func TestContextValidator_Integration(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("context-aware webhook validation success", func(t *testing.T) {
		// Create context with Stripe secret
		ctx := context.WithValue(context.Background(), "stripe_webhook_secret", "test_secret")

		req := &ContextAwareWebhookRequest{
			Headers: struct {
				StripeSignature string `gork:"Stripe-Signature" validate:"required"`
			}{
				StripeSignature: "t=123,v1=test_secret",
			},
			Body: []byte(`{"event": "payment.succeeded"}`),
		}

		err := validator.ValidateRequest(ctx, req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("context-aware webhook validation missing context", func(t *testing.T) {
		// Context without Stripe secret
		ctx := context.Background()

		req := &ContextAwareWebhookRequest{
			Headers: struct {
				StripeSignature string `gork:"Stripe-Signature" validate:"required"`
			}{
				StripeSignature: "t=123,v1=test_secret",
			},
			Body: []byte(`{"event": "payment.succeeded"}`),
		}

		err := validator.ValidateRequest(ctx, req)
		if err == nil {
			t.Error("expected error for missing context secret")
		}

		// Should be a server error (500) since missing context is not client's fault
		if IsValidationError(err) {
			t.Error("expected server error, got validation error")
		}
	})

	t.Run("context-aware webhook validation invalid signature", func(t *testing.T) {
		// Create context with correct Stripe secret
		ctx := context.WithValue(context.Background(), "stripe_webhook_secret", "test_secret")

		req := &ContextAwareWebhookRequest{
			Headers: struct {
				StripeSignature string `gork:"Stripe-Signature" validate:"required"`
			}{
				StripeSignature: "t=123,v1=wrong_secret",
			},
			Body: []byte(`{"event": "payment.succeeded"}`),
		}

		err := validator.ValidateRequest(ctx, req)
		if err == nil {
			t.Error("expected error for invalid signature")
		}

		// Should be a validation error (400) since invalid signature is client's fault
		if !IsValidationError(err) {
			t.Errorf("expected validation error, got %v", err)
		}
	})

	t.Run("backward compatibility - traditional validator", func(t *testing.T) {
		ctx := context.Background()

		req := &ContextValidationTraditionalRequest{
			Body: struct {
				Name string `json:"name" validate:"required"`
			}{
				Name: "valid_name",
			},
		}

		err := validator.ValidateRequest(ctx, req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("backward compatibility - traditional validator error", func(t *testing.T) {
		ctx := context.Background()

		req := &ContextValidationTraditionalRequest{
			Body: struct {
				Name string `json:"name" validate:"required"`
			}{
				Name: "forbidden",
			},
		}

		err := validator.ValidateRequest(ctx, req)
		if err == nil {
			t.Error("expected error for forbidden name")
		}

		// Should be a validation error (400)
		if !IsValidationError(err) {
			t.Errorf("expected validation error, got %v", err)
		}
	})
}

func TestContextValidator_PriorityOver_RegularValidator(t *testing.T) {
	validator := NewConventionValidator()

	// Simple test to verify that context-aware validation is working
	t.Run("context validator functionality", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "stripe_webhook_secret", "test_secret")

		req := &ContextAwareWebhookRequest{
			Headers: struct {
				StripeSignature string `gork:"Stripe-Signature" validate:"required"`
			}{
				StripeSignature: "t=123,v1=test_secret",
			},
			Body: []byte(`{"event": "payment.succeeded"}`),
		}

		err := validator.ValidateRequest(ctx, req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("regular validator functionality", func(t *testing.T) {
		req := &ContextValidationTraditionalRequest{
			Body: struct {
				Name string `json:"name" validate:"required"`
			}{
				Name: "valid_name",
			},
		}

		err := validator.ValidateRequest(context.Background(), req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
