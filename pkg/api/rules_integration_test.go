package api

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	rules "github.com/gork-labs/gork/pkg/rules"
)

// Define and register a simple equality rule for tests.
var registerOnce sync.Once

type simpleValidationError struct{ msg string }

func (e simpleValidationError) Error() string       { return e.msg }
func (e simpleValidationError) GetErrors() []string { return []string{e.msg} }

func registerTestEqRule() {
	registerOnce.Do(func() {
		// Use a defer/recover to handle the case where the rule might already be registered
		// This can happen when running the full test suite due to test execution order
		defer func() {
			if r := recover(); r != nil {
				// If registration fails due to duplicate, that's fine for tests
				if panicMsg, ok := r.(string); ok && strings.Contains(panicMsg, "already registered") {
					return
				}
				panic(r) // Re-panic if it's a different error
			}
		}()

		rules.Register("test_eq", func(ctx context.Context, entity any, args ...any) (bool, error) {
			ps, _ := entity.(*string)
			if ps == nil || len(args) != 1 {
				return false, simpleValidationError{msg: "bad args"}
			}
			want, _ := args[0].(string)
			if *ps != want {
				return false, nil // validation failed, but no system error
			}
			return true, nil // validation passed
		})
	})
}

type testRulesReq struct {
	Path struct {
		UserID string `rule:"test_eq('u1')"`
	}
}

func TestConventionValidator_RulesIntegration_Passes(t *testing.T) {
	registerTestEqRule()
	var r testRulesReq
	r.Path.UserID = "u1"

	v := NewConventionValidator()
	if err := v.ValidateRequest(context.Background(), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConventionValidator_RulesIntegration_Fails(t *testing.T) {
	registerTestEqRule()
	var r testRulesReq
	r.Path.UserID = "u2"

	v := NewConventionValidator()
	err := v.ValidateRequest(context.Background(), &r)
	if err == nil {
		t.Fatalf("expected validation error from rules engine")
	}
	verr, ok := err.(*ValidationErrorResponse)
	if !ok {
		t.Fatalf("expected ValidationErrorResponse, got %T", err)
	}
	if len(verr.Details["request"]) == 0 {
		t.Fatalf("expected rule errors under 'request' key: %+v", verr.Details)
	}
}

func TestConventionValidator_RulesIntegration_ServerErrorShortCircuit(t *testing.T) {
	// Register a rule that returns a non-validation error (server error)
	rules.Register("test_server_err", func(ctx context.Context, entity any, args ...any) (bool, error) {
		return false, errors.New("db unavailable")
	})

	type req struct {
		Path struct {
			X string `rule:"test_server_err()"`
		}
	}
	var r req
	v := NewConventionValidator()
	err := v.ValidateRequest(context.Background(), &r)
	if err == nil {
		t.Fatalf("expected server error from rule")
	}
	var verr *ValidationErrorResponse
	if errors.As(err, &verr) {
		t.Fatalf("expected server error (non-validation), got ValidationErrorResponse: %#v", verr)
	}
}

func TestConventionValidator_RulesIntegration_NilErrorIgnored(t *testing.T) {
	// Register a rule that returns true to ensure it doesn't add validation errors
	rules.Register("test_nil", func(ctx context.Context, entity any, args ...any) (bool, error) { return true, nil })
	type req struct {
		Path struct {
			X string `rule:"test_nil()"`
		}
	}
	var r req
	v := NewConventionValidator()
	if err := v.ValidateRequest(context.Background(), &r); err != nil {
		t.Fatalf("unexpected error for passing rule: %v", err)
	}
}

func TestConventionValidator_RulesIntegration_ValidationErrorInterfaceCast(t *testing.T) {
	// Register a rule that returns a validation error implementing ValidationError interface
	rules.Register("test_validation_err_interface", func(ctx context.Context, entity any, args ...any) (bool, error) {
		return false, simpleValidationError{msg: "interface validation error"}
	})

	type req struct {
		Path struct {
			X string `rule:"test_validation_err_interface()"`
		}
	}
	var r req
	v := NewConventionValidator()
	err := v.ValidateRequest(context.Background(), &r)
	if err == nil {
		t.Fatalf("expected validation error from rules engine")
	}

	// This should be a ValidationErrorResponse containing the rule validation error
	verr, ok := err.(*ValidationErrorResponse)
	if !ok {
		t.Fatalf("expected ValidationErrorResponse, got %T", err)
	}
	if len(verr.Details["request"]) == 0 {
		t.Fatalf("expected rule errors under 'request' key: %+v", verr.Details)
	}
	if verr.Details["request"][0] != "interface validation error" {
		t.Fatalf("expected 'interface validation error', got '%s'", verr.Details["request"][0])
	}
}
