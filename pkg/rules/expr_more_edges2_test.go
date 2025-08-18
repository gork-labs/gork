package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestEvalBooleanExpr_TokenizeAndParseErrors(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "@"); len(errs) == 0 {
		t.Fatalf("expected tokenize error")
	}
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "ok("); len(errs) == 0 {
		t.Fatalf("expected parse error")
	}
}

func TestEvalNode_UnaryInnerServerErr(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	n := &nodeUnary{op: tkNot, x: 123}
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, n)
	if res.serverErr == nil {
		t.Fatalf("expected server error from inner node")
	}
}

func TestEvalOrNode_BothFalseCollectsErrors(t *testing.T) {
	resetRegistry()
	Register("errA", func(ctx context.Context, _ any) (bool, error) { return false, nil })
	Register("errB", func(ctx context.Context, _ any) (bool, error) { return false, nil })
	var root, parent struct{}
	ent := "e"
	left := &nodeCall{name: "errA"}
	right := &nodeCall{name: "errB"}
	b := &nodeBinary{op: tkOr, l: left, r: right}
	res := evalOrNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if res.pass || len(res.valErrs) != 2 {
		t.Fatalf("expected two validation errors, got pass=%v errs=%v", res.pass, res.valErrs)
	}
}

func TestCollectArgsString_NestedAndStrings(t *testing.T) {
	toks, err := tokenize("f(g('x'), h(1, 2))")
	if err != nil {
		t.Fatal(err)
	}
	// pos after '('
	p := &parser{toks: toks, pos: 2}
	s, err := p.collectArgsString()
	if err != nil || s == "" {
		t.Fatalf("expected args string, got %q err=%v", s, err)
	}
}
