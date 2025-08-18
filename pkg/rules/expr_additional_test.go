package rules

import (
	"context"
	"reflect"
	"testing"
)

// cover tokenizer and parser branches: unexpected char, unterminated string, NOT/AND/OR precedence
func TestExprTokenizerAndParser_Branches(t *testing.T) {
	// unexpected character
	if _, err := tokenize("@"); err == nil {
		t.Fatalf("expected tokenizer error for unexpected char")
	}
	// unterminated string
	if _, err := tokenize("allow('x"); err == nil {
		t.Fatalf("expected unterminated string error")
	}

	// Build a tiny registry to satisfy invokeRule
	resetRegistry()
	Register("trueRule", func(ctx context.Context, _ any) (bool, error) { return true, nil })
	Register("falseRule", func(ctx context.Context, _ any) (bool, error) { return false, nil })

	// Build an expression mixing all operators and parentheses
	expr := "not (falseRule() and trueRule()) or (trueRule() and not falseRule())"
	tokens, err := tokenize(expr)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	p := &parser{toks: tokens}
	n, err := p.parseExpr()
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}

	// Use addressable struct values for root and parent
	var root struct{}
	var parent struct{}
	dummy := "entity"
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &dummy, n)
	if res.serverErr != nil || !res.pass {
		t.Fatalf("expected pass, got %#v", res)
	}
}

// cover eval error branches: unsupported node/op and EOF handling in collectArgsString
func TestExprEvaluator_ErrorBranches(t *testing.T) {
	// Unsupported node
	r := evalNode(context.Background(), reflect.Value{}, reflect.Value{}, nil, 42)
	if r.serverErr == nil {
		t.Fatalf("expected serverErr for unsupported node")
	}

	// collectArgsString EOF (unterminated)
	p := &parser{toks: []token{{kind: tkIdent, text: "f"}, {kind: tkLPar}, {kind: tkIdent, text: "x"}, {kind: tkEOF}}}
	if _, err := p.collectArgsString(); err == nil {
		t.Fatalf("expected error for unterminated argument list")
	}
}
