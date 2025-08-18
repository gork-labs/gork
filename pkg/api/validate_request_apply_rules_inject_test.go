package api

import (
	"context"
	"testing"
)

// Covers the branch where applyRulesFunc is nil and falls back to rules.Apply
func TestConventionValidator_ApplyRulesFunc_NilFallback(t *testing.T) {
	type req struct{}
	var r req
	v := NewConventionValidator()
	v.applyRulesFunc = nil
	if err := v.ValidateRequest(context.Background(), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
