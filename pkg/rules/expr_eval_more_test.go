package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestEvalBooleanExpr_CoversAll(t *testing.T) {
	resetRegistry()
	Register("ok", func(ctx context.Context, _ any) (bool, error) { return true, nil })
	Register("fail", func(ctx context.Context, _ any) (bool, error) { return false, nil })

	var root, parent struct{}
	ent := "e"

	// pass
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "ok()"); errs != nil {
		t.Fatalf("unexpected errs: %v", errs)
	}
	// validation error (returns slice)
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "fail()"); len(errs) == 0 {
		t.Fatalf("expected validation errs")
	}
	// tokenize error
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "@"); len(errs) == 0 {
		t.Fatalf("expected tokenize error")
	}
	// parse error (mismatched parentheses)
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "(ok("); len(errs) == 0 {
		t.Fatalf("expected parse error")
	}
}
