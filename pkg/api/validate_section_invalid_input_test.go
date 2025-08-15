package api

import (
	"context"
	"testing"
)

// Non-struct section types should trigger validator.InvalidValidationError path
func TestValidateSection_NonStructSectionServerError(t *testing.T) {
	v := NewConventionValidator()
	type badReq struct {
		Headers string // not a struct; allowed section name but invalid type
	}
	err := v.ValidateRequest(context.Background(), &badReq{Headers: "x"})
	if err == nil {
		t.Fatal("expected server error for non-struct section type")
	}
}
