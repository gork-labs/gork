package api

import (
	"context"
	"reflect"
	"testing"
)

// Covers the []byte Body special-case path that uses validator.Var with tags on Body.
func TestValidateSection_ByteBodyWithTags(t *testing.T) {
	v := NewConventionValidator()

	// Body is []byte with a length constraint; should aggregate errors under "body"
	type Req struct {
		Body []byte `validate:"min=5"`
	}

	// Too short to satisfy min=5
	req := Req{Body: []byte("abcd")}

	rv := reflect.ValueOf(req)
	rt := rv.Type()
	field := rt.Field(0)
	val := rv.Field(0)

	errs := make(map[string][]string)
	if err := v.validateSection(context.Background(), field, val, errs); err != nil {
		t.Fatalf("unexpected server error: %v", err)
	}

	// Expect a validation error recorded for the section name "body"
	messages, ok := errs["body"]
	if !ok || len(messages) == 0 {
		t.Fatalf("expected validation errors for body, got: %v", errs)
	}

	// Now satisfy the constraint and expect no errors
	reqOk := Req{Body: []byte("abcde")}
	rvOk := reflect.ValueOf(reqOk)
	valOk := rvOk.Field(0)
	errsOk := make(map[string][]string)
	if err := v.validateSection(context.Background(), field, valOk, errsOk); err != nil {
		t.Fatalf("unexpected server error: %v", err)
	}
	if len(errsOk) != 0 {
		t.Fatalf("expected no validation errors, got: %v", errsOk)
	}
}
