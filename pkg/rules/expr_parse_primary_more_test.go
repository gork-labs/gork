package rules

import "testing"

func TestParsePrimary_UnexpectedAfterParen(t *testing.T) {
	// (true without closing ) to force error in expect(tkRPar)
	toks, err := tokenize("(true")
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	if _, err := p.parsePrimary(); err == nil {
		t.Fatalf("expected missing right paren error")
	}
}
