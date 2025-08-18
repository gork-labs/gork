package rules

import "testing"

func TestTokenize_Kinds(t *testing.T) {
	// numbers are classified later via tag parser, tokenizer doesn't emit tkNumber directly from bare digits here
	s := "true false null $.Path.Field .Local $ctx ! , ( ) 'str' \"q\" and or not foo()"
	toks, err := tokenize(s)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	// Spot check presence of various token kinds
	var hasBool, hasNull, hasFieldRefAbs, hasFieldRefRel, hasCtx, hasNot, hasLPar, hasRPar, hasComma, hasString, hasAnd, hasOr, hasIdent bool
	for _, tk := range toks {
		switch tk.kind {
		case tkBool:
			hasBool = true
		case tkNull:
			hasNull = true
		case tkFieldRef:
			if tk.text == "$.Path.Field" {
				hasFieldRefAbs = true
			}
			if tk.text == ".Local" {
				hasFieldRefRel = true
			}
		case tkContextVar:
			hasCtx = true
		case tkNot:
			hasNot = true
		case tkLPar:
			hasLPar = true
		case tkRPar:
			hasRPar = true
		case tkComma:
			hasComma = true
		case tkString:
			hasString = true
		case tkAnd:
			hasAnd = true
		case tkOr:
			hasOr = true
		case tkIdent:
			hasIdent = true
		}
	}
	if !(hasBool && hasNull && hasFieldRefAbs && hasFieldRefRel && hasCtx && hasNot && hasLPar && hasRPar && hasComma && hasString && hasAnd && hasOr && hasIdent) {
		t.Fatalf("missing kinds: bool=%v null=%v abs=%v rel=%v ctx=%v !%v (=%v )=%v ,=%v str=%v and=%v or=%v ident=%v",
			hasBool, hasNull, hasFieldRefAbs, hasFieldRefRel, hasCtx, hasNot, hasLPar, hasRPar, hasComma, hasString, hasAnd, hasOr, hasIdent)
	}
}
