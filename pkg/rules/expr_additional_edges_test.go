package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestParseFunctionCall_NoArgs(t *testing.T) {
	toks, err := tokenize("f()")
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	n, err := p.parseExpr()
	if err != nil {
		t.Fatalf("parseExpr error: %v", err)
	}
	call, ok := n.(*nodeCall)
	if !ok {
		t.Fatalf("expected nodeCall, got %T", n)
	}
	if call.name != "f" || len(call.args) != 0 {
		t.Fatalf("unexpected call node: %+v", call)
	}
}

func TestEvalAndNode_BothTrue(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkAnd, l: &nodeBool{v: true}, r: &nodeBool{v: true}}
	res := evalAndNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if !res.pass || res.serverErr != nil || len(res.valErrs) != 0 {
		t.Fatalf("expected pass true, got %#v", res)
	}
}

func TestEvalBooleanExpr_ServerError(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "unknownRule()")
	if len(errs) == 0 {
		t.Fatalf("expected server error from unknown rule")
	}
}
