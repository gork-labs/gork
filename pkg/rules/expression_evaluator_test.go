package rules

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

// TestExpressionEvaluator_ServerErrorBranch tests the server error branch in EvalBooleanExpr
// using the clean method-based approach with dependency injection.
func TestExpressionEvaluator_ServerErrorBranch(t *testing.T) {
	var root, parent struct{}
	ent := "entity"
	ctx := context.Background()
	rootVal := reflect.ValueOf(&root).Elem()
	parentVal := reflect.ValueOf(&parent).Elem()

	// Create evaluator with mocked evalNodeFunc that returns server error
	evaluator := &ExpressionEvaluator{
		evalNodeFunc: func(ctx context.Context, root, parent reflect.Value, entity any, n node) evalResult {
			return evalResult{serverErr: fmt.Errorf("mock server error")}
		},
	}

	// Test the server error branch
	errs := evaluator.EvalBooleanExpr(ctx, rootVal, parentVal, &ent, "true")

	// Verify the server error branch was executed
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if errs[0].Error() != "mock server error" {
		t.Fatalf("expected 'mock server error', got %q", errs[0].Error())
	}

	t.Logf("✓ Successfully tested server error branch with clean method injection")
}

// TestExpressionEvaluator_NormalOperation tests that the default evaluator works correctly
func TestExpressionEvaluator_NormalOperation(t *testing.T) {
	var root, parent struct{}
	ent := "entity"
	ctx := context.Background()
	rootVal := reflect.ValueOf(&root).Elem()
	parentVal := reflect.ValueOf(&parent).Elem()

	evaluator := NewExpressionEvaluator()

	// Test successful evaluation
	errs := evaluator.EvalBooleanExpr(ctx, rootVal, parentVal, &ent, "true")
	if len(errs) != 0 {
		t.Fatalf("expected no errors for 'true', got %v", errs)
	}

	// Test with false (should return no errors since it passes)
	errs = evaluator.EvalBooleanExpr(ctx, rootVal, parentVal, &ent, "false")
	if len(errs) != 0 {
		t.Fatalf("expected no errors for 'false', got %v", errs)
	}
}

// TestExpressionEvaluator_ValidationErrorBranch tests the validation error branch
func TestExpressionEvaluator_ValidationErrorBranch(t *testing.T) {
	resetRegistry()
	Register("alwaysFail", func(ctx context.Context, entity any) (bool, error) {
		return false, nil // Validation failed (business logic)
	})

	var root, parent struct{}
	ent := "entity"
	ctx := context.Background()
	rootVal := reflect.ValueOf(&root).Elem()
	parentVal := reflect.ValueOf(&parent).Elem()

	evaluator := NewExpressionEvaluator()

	// Test validation error path (!res.pass)
	errs := evaluator.EvalBooleanExpr(ctx, rootVal, parentVal, &ent, "alwaysFail()")
	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(errs))
	}

	if errs[0].Error() != "rule 'alwaysFail' validation failed" {
		t.Fatalf("expected 'rule 'alwaysFail' validation failed', got %q", errs[0].Error())
	}
}

// TestExpressionEvaluator_AllBranches tests all code paths in EvalBooleanExpr
func TestExpressionEvaluator_AllBranches(t *testing.T) {
	var root, parent struct{}
	ent := "entity"
	ctx := context.Background()
	rootVal := reflect.ValueOf(&root).Elem()
	parentVal := reflect.ValueOf(&parent).Elem()

	testCases := []struct {
		name         string
		input        string
		mockFunc     func(ctx context.Context, root, parent reflect.Value, entity any, n node) evalResult
		expectErrs   int
		expectErrMsg string
	}{
		{
			name:         "Tokenize Error",
			input:        `"unterminated string`,
			mockFunc:     nil, // Will use default, but tokenize will fail first
			expectErrs:   1,
			expectErrMsg: "rules: expr tokenize:",
		},
		{
			name:         "Parse Error",
			input:        "(",
			mockFunc:     nil, // Will use default, but parse will fail
			expectErrs:   1,
			expectErrMsg: "rules: expr parse:",
		},
		{
			name:  "Server Error",
			input: "true",
			mockFunc: func(ctx context.Context, root, parent reflect.Value, entity any, n node) evalResult {
				return evalResult{serverErr: fmt.Errorf("server error")}
			},
			expectErrs:   1,
			expectErrMsg: "server error",
		},
		{
			name:  "Validation Error",
			input: "true",
			mockFunc: func(ctx context.Context, root, parent reflect.Value, entity any, n node) evalResult {
				return evalResult{pass: false, valErrs: []error{fmt.Errorf("validation error")}}
			},
			expectErrs:   1,
			expectErrMsg: "validation error",
		},
		{
			name:  "Success",
			input: "true",
			mockFunc: func(ctx context.Context, root, parent reflect.Value, entity any, n node) evalResult {
				return evalResult{pass: true}
			},
			expectErrs: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evaluator := NewExpressionEvaluator()
			if tc.mockFunc != nil {
				evaluator.evalNodeFunc = tc.mockFunc
			}

			errs := evaluator.EvalBooleanExpr(ctx, rootVal, parentVal, &ent, tc.input)

			if len(errs) != tc.expectErrs {
				t.Fatalf("expected %d errors, got %d: %v", tc.expectErrs, len(errs), errs)
			}

			if tc.expectErrs > 0 {
				errMsg := errs[0].Error()
				if tc.expectErrMsg != "" && !contains(errMsg, tc.expectErrMsg) {
					t.Fatalf("expected error containing %q, got %q", tc.expectErrMsg, errMsg)
				}
			}

			t.Logf("✓ %s: Correct behavior", tc.name)
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
