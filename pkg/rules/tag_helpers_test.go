package rules

import "testing"

func Test_isNumberLike_Edges(t *testing.T) {
	if isNumberLike("") {
		t.Fatal("empty string should not be number-like")
	}
	if isNumberLike(".") {
		t.Fatal("dot alone should not be number-like")
	}
}

func Test_isIdent_Various(t *testing.T) {
	if !isIdent("_x1") {
		t.Fatal("underscore start should be valid identifier")
	}
	if isIdent("") {
		t.Fatal("empty identifier should be invalid")
	}
}

func Test_parseArgs_SkipEmptyElements(t *testing.T) {
	invs, err := parse("fn('a', , 'b')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invs) != 1 || len(invs[0].Args) != 2 || invs[0].Args[0].Str != "a" || invs[0].Args[1].Str != "b" {
		t.Fatalf("unexpected parse result: %+v", invs)
	}
}
