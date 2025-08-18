package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestEvalBooleanExpr_FalseValidationErrors(t *testing.T) {
	resetRegistry()
	Register("fail", func(ctx context.Context, _ any) (bool, error) { return false, nil })
	var root, parent struct{}
	ent := "e"
	errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "fail()")
	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %v", errs)
	}
}

func TestEvalAndNode_RightValidationError(t *testing.T) {
	resetRegistry()
	Register("ok", func(ctx context.Context, _ any) (bool, error) { return true, nil })
	Register("fail", func(ctx context.Context, _ any) (bool, error) { return false, nil })
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkAnd, l: &nodeCall{name: "ok"}, r: &nodeCall{name: "fail"}}
	res := evalAndNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if res.pass || res.serverErr != nil || len(res.valErrs) == 0 {
		t.Fatalf("expected right-side validation error, got %#v", res)
	}
}

func TestParsePrimary_BoolFalse(t *testing.T) {
	toks, err := tokenize("false")
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	n, err := p.parsePrimary()
	if err != nil {
		t.Fatalf("parsePrimary error: %v", err)
	}
	nb, ok := n.(*nodeBool)
	if !ok || nb.v != false {
		t.Fatalf("expected nodeBool false, got %#v", n)
	}
}
