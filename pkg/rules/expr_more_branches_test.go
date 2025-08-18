package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestEvalAndNode_RightFailureShortCircuit(t *testing.T) {
	resetRegistry()
	Register("ok", func(ctx context.Context, _ any) (bool, error) { return true, nil })
	Register("bad", func(ctx context.Context, _ any) (bool, error) { return false, nil })
	expr := "ok() and bad()"
	toks, err := tokenize(expr)
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	n, err := p.parseExpr()
	if err != nil {
		t.Fatal(err)
	}
	var root, parent struct{}
	ent := "e"
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, n)
	if res.pass || len(res.valErrs) == 0 {
		t.Fatalf("expected right-side failure, got %#v", res)
	}
}

func TestResolveContextVar_NilValueAndNotFound(t *testing.T) {
	// not found
	if _, err := resolveContextVar(context.Background(), argToken{Kind: argContextVar, ContextVar: "nope"}); err == nil {
		t.Fatalf("expected not found error")
	}
	// present but nil
	ctx := WithContextVars(context.Background(), ContextVars{"k": nil})
	if _, err := resolveContextVar(ctx, argToken{Kind: argContextVar, ContextVar: "k"}); err == nil {
		t.Fatalf("expected nil value error")
	}
}

func TestParserExpectAndPrimaryErrors(t *testing.T) {
	// expect error when token does not match
	p := &parser{toks: []token{{kind: tkIdent, text: "f"}}}
	if err := p.expect(tkLPar); err == nil {
		t.Fatalf("expected expect() error")
	}
	// parsePrimary unexpected token (string)
	p2 := &parser{toks: []token{{kind: tkString, text: "x"}}}
	if _, err := p2.parsePrimary(); err == nil {
		t.Fatalf("expected parsePrimary error for tkString")
	}
}

func TestParseNot_WithExclamation(t *testing.T) {
	resetRegistry()
	Register("bad", func(ctx context.Context, _ any) (bool, error) { return false, nil })
	expr := "!bad()"
	toks, err := tokenize(expr)
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	n, err := p.parseExpr()
	if err != nil {
		t.Fatal(err)
	}
	var root, parent struct{}
	ent := "e"
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, n)
	if !res.pass {
		t.Fatalf("expected pass for !bad()")
	}
}
