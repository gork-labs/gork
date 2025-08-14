package api

import (
	"context"
	"reflect"
	"testing"
)

// Dummy types for handler signature tests
type testProviderPayload struct{ ID string }
type testUserMeta struct{ Name string }

func TestValidateEventHandlerSignature_ProviderParamMustBePointer(t *testing.T) {
	// Non-pointer provider parameter should fail validation
	bad := func(ctx context.Context, p testProviderPayload, u *testUserMeta) (interface{}, error) {
		return nil, nil
	}

	err := validateEventHandlerSignature(reflect.TypeOf(bad))
	if err == nil {
		t.Fatalf("expected error for non-pointer provider parameter, got nil")
	}
}

func TestValidateEventHandlerSignature_UserParamMustBePointer(t *testing.T) {
	// Non-pointer user metadata parameter should fail validation
	bad := func(ctx context.Context, p *testProviderPayload, u testUserMeta) (interface{}, error) {
		return nil, nil
	}

	err := validateEventHandlerSignature(reflect.TypeOf(bad))
	if err == nil {
		t.Fatalf("expected error for non-pointer user metadata parameter, got nil")
	}
}
