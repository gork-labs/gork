package rules

import "testing"

func TestParseFunctionCall_EmptyArgsStringHandled(t *testing.T) {
	toks, err := tokenize("f(   )")
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	if _, err := p.parseExpr(); err != nil {
		t.Fatalf("parseExpr error: %v", err)
	}
}
