package api

import (
	"context"
	"reflect"
	"testing"
)

// TestWebhookSignatureValidation consolidates tests for webhook event handler signature validation,
// focusing on parameter types, return types, and function signature requirements.
func TestWebhookSignatureValidation(t *testing.T) {
	t.Run("SecondParameterMustBePointer", func(t *testing.T) {
		// Wrong: second parameter is not a pointer; exactly one return (error)
		bad := func(ctx context.Context, payload struct{}, metadata *PaymentMetadata) error { return nil }
		err := validateEventHandlerSignature(reflect.TypeOf(bad))
		if err == nil {
			t.Fatalf("expected error for non-pointer provider parameter, got nil")
		}
	})

	t.Run("SingleReturnMustBeError", func(t *testing.T) {
		// Wrong: exactly one return, but not error
		bad := func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) string { return "" }
		err := validateEventHandlerSignature(reflect.TypeOf(bad))
		if err == nil {
			t.Fatalf("expected error for non-error return type, got nil")
		}
	})
}
