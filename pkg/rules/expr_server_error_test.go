package rules

import (
	"context"
	"reflect"
	"testing"
)

type testServerErr struct{ msg string }

func (e testServerErr) Error() string { return e.msg }

func TestEvalBooleanExpr_ServerErrorMarkerShortCircuits(t *testing.T) {
	resetRegistry()
	Register("serverFail", func(ctx context.Context, _ any) (bool, error) { return false, testServerErr{msg: "boom"} })
	var root, parent struct{}
	ent := "e"
	errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "serverFail()")
	if len(errs) != 1 || errs[0].Error() != "boom" {
		t.Fatalf("expected server error short-circuit, got %#v", errs)
	}
}
