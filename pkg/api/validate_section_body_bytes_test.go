package api

import (
	"context"
	"testing"
)

// Ensure []byte Body section is skipped by go-playground validation path
func TestValidateSection_SkipByteBody(t *testing.T) {
	v := NewConventionValidator()
	type Req struct{ Body []byte }
	req := &Req{Body: []byte("raw")}
	if err := v.ValidateRequest(context.Background(), req); err != nil {
		t.Fatalf("expected no error for []byte Body, got %v", err)
	}
}
