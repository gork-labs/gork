package api

import (
	"context"
	"reflect"
	"testing"
)

// Additional signature checks to hit remaining branches in validateEventHandlerSignature.
func TestValidateEventHandlerSignature_AdditionalCases(t *testing.T) {
	t.Run("second parameter must be pointer", func(t *testing.T) {
		// Wrong: second parameter is not a pointer; exactly one return (error)
		bad := func(ctx context.Context, payload struct{}, metadata *PaymentMetadata) error { return nil }
		err := validateEventHandlerSignature(reflect.TypeOf(bad))
		if err == nil {
			t.Fatalf("expected error for non-pointer provider parameter, got nil")
		}
	})

	t.Run("single return must be error", func(t *testing.T) {
		// Wrong: exactly one return, but not error
		bad := func(ctx context.Context, payload *struct{}, metadata *PaymentMetadata) string { return "" }
		err := validateEventHandlerSignature(reflect.TypeOf(bad))
		if err == nil {
			t.Fatalf("expected error for non-error return type, got nil")
		}
	})
}
