package rules

import "testing"

func TestParseFunctionCall_UnterminatedArgs_ViaExpr(t *testing.T) {
	// Build tokens: f ( x EOF — so collectArgsString errors inside parseFunctionCall
	toks := []token{{kind: tkIdent, text: "f"}, {kind: tkLPar, text: "("}, {kind: tkIdent, text: "x"}, {kind: tkEOF}}
	p := &parser{toks: toks}
	if _, err := p.parseExpr(); err == nil {
		t.Fatalf("expected unterminated argument list error")
	}
}
