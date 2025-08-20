package rules

import (
	"context"
	"testing"
)

// TestContextVariableResolution consolidates tests for context variable resolution,
// including missing variables, nil variables, and error handling.
func TestContextVariableResolution(t *testing.T) {
	t.Run("ContextVariableNotFound", func(t *testing.T) {
		ctx := context.Background()
		// Test when context variable is not found
		token := argToken{Kind: argContextVar, ContextVar: "nonexistent"}
		_, err := resolveContextVar(ctx, token)
		if err == nil {
			t.Fatalf("expected error for nonexistent context variable")
		}
	})

	t.Run("ContextVariableNil", func(t *testing.T) {
		ctx := context.Background()
		vars := ContextVars{"nilvar": nil}
		ctx = WithContextVars(ctx, vars)

		token := argToken{Kind: argContextVar, ContextVar: "nilvar"}
		_, err := resolveContextVar(ctx, token)
		if err == nil {
			t.Fatalf("expected error for nil context variable")
		}
	})

	t.Run("ContextVariableFound", func(t *testing.T) {
		ctx := context.Background()
		vars := ContextVars{"found": "value"}
		ctx = WithContextVars(ctx, vars)

		token := argToken{Kind: argContextVar, ContextVar: "found"}
		result, err := resolveContextVar(ctx, token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "value" {
			t.Fatalf("expected 'value', got %v", result)
		}
	})
}
