package api

import (
	"context"
	"testing"
)

// Simple types to exercise invokeCustomValidation branches directly.
type simpleValidatorOK struct{}

func (simpleValidatorOK) Validate() error { return nil }

func TestInvokeCustomValidation_Branches(t *testing.T) {
	ctx := context.Background()

	// v == nil path
	if msgs, err := invokeCustomValidation(ctx, nil); err != nil || msgs != nil {
		t.Fatalf("expected (nil,nil) for nil value, got (%v,%v)", msgs, err)
	}

	// Regular Validator that returns nil (hits the success return path)
	if msgs, err := invokeCustomValidation(ctx, simpleValidatorOK{}); err != nil || msgs != nil {
		t.Fatalf("expected (nil,nil) for successful validator, got (%v,%v)", msgs, err)
	}
}
