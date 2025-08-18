package rules

import "testing"

func TestParsePrimary_Paren(t *testing.T) {
	toks, err := tokenize("(true)")
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	n, err := p.parsePrimary()
	if err != nil {
		t.Fatalf("expected primary parse, got %v", err)
	}
	if _, ok := n.(*nodeBool); !ok {
		t.Fatalf("expected nodeBool from paren primary, got %T", n)
	}
}

func TestParseFunctionCall_ParseArgsError(t *testing.T) {
	toks, err := tokenize("bad(!)")
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	if _, err := p.parseExpr(); err == nil {
		t.Fatalf("expected parse error for invalid arg in function call")
	}
}
