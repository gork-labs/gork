package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestEvalNode_UnsupportedExprNode(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, 123)
	if res.serverErr == nil {
		t.Fatalf("expected unsupported expr node server error")
	}
}

func TestEvalAndNode_LeftFalseShortCircuit(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkAnd, l: &nodeBool{v: false}, r: &nodeBool{v: true}}
	res := evalAndNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if res.pass {
		t.Fatalf("expected false due to left side false")
	}
}

func TestEvalAndNode_RightServerErr(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkAnd, l: &nodeBool{v: true}, r: 123}
	res := evalAndNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if res.serverErr == nil {
		t.Fatalf("expected server error from right side")
	}
}

func TestEvalOrNode_LeftPassTrueShortcut(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkOr, l: &nodeBool{v: true}, r: &nodeBool{v: false}}
	res := evalOrNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if !res.pass {
		t.Fatalf("expected pass true due to left side true")
	}
}

func TestEvalOrNode_RightServerErr(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkOr, l: &nodeBool{v: false}, r: 123}
	res := evalOrNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if res.serverErr == nil {
		t.Fatalf("expected server error from right side")
	}
}

func TestEvalNode_AndDispatch(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	b := &nodeBinary{op: tkAnd, l: &nodeBool{v: true}, r: &nodeBool{v: true}}
	res := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, b)
	if !res.pass || res.serverErr != nil || len(res.valErrs) != 0 {
		t.Fatalf("expected pass through AND dispatch, got %#v", res)
	}
}

func TestCollectArgsString_Unterminated(t *testing.T) {
	// Build token stream: f( x EOF
	toks := []token{{kind: tkIdent, text: "f"}, {kind: tkLPar, text: "("}, {kind: tkIdent, text: "x"}, {kind: tkEOF}}
	p := &parser{toks: toks, pos: 1} // at '('
	if _, err := p.collectArgsString(); err == nil {
		t.Fatalf("expected unterminated argument list error")
	}
}

func TestParseFunctionCall_ExpectLParError(t *testing.T) {
	toks := []token{{kind: tkBool, text: "true"}, {kind: tkEOF}}
	p := &parser{toks: toks}
	if _, err := p.parseFunctionCall("f"); err == nil {
		t.Fatalf("expected expect(tkLPar) error")
	}
}

func TestParsePrimary_UnexpectedToken(t *testing.T) {
	toks := []token{{kind: tkComma, text: ","}, {kind: tkEOF}}
	p := &parser{toks: toks}
	if _, err := p.parsePrimary(); err == nil {
		t.Fatalf("expected unexpected token error")
	}
}

func TestParseOrAnd_ErrorPaths(t *testing.T) {
	// or with rhs error
	toks, err := tokenize("a() or !")
	if err != nil {
		t.Fatal(err)
	}
	p := &parser{toks: toks}
	if _, err := p.parseExpr(); err == nil {
		t.Fatalf("expected parse error for rhs after or")
	}

	// and with rhs error
	toks, err = tokenize("a() and !")
	if err != nil {
		t.Fatal(err)
	}
	p = &parser{toks: toks}
	if _, err := p.parseExpr(); err == nil {
		t.Fatalf("expected parse error for rhs after and")
	}
}
