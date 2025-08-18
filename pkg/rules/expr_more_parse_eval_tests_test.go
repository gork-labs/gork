package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestParseFunctionCall_SuccessArgs(t *testing.T) {
	toks, err := tokenize("f('x', $.Path.UserID, .Owner, $var)")
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
	if call.name != "f" || len(call.args) != 4 {
		t.Fatalf("unexpected call node: %+v", call)
	}
}

func TestParseOrAnd_Multiple(t *testing.T) {
	for _, s := range []string{"a() or b() or c()", "a() and b() and c()", "a() or b() and ! c()"} {
		toks, err := tokenize(s)
		if err != nil {
			t.Fatalf("tokenize %q: %v", s, err)
		}
		p := &parser{toks: toks}
		if _, err := p.parseExpr(); err != nil {
			t.Fatalf("parseExpr %q: %v", s, err)
		}
	}
}

func TestEvalNode_UnsupportedBinaryOp(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tokKind(999), l: &nodeBool{v: true}, r: &nodeBool{v: false}}
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if res.serverErr == nil {
		t.Fatalf("expected serverErr for unsupported binary op")
	}
}

func TestEvalOrNode_LeftServerErrPropagation(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkOr, l: 123, r: &nodeBool{v: true}}
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if res.serverErr == nil {
		t.Fatalf("expected serverErr propagated from left side")
	}
}
