package rules

import "testing"

func TestParseTag_AbsoluteAndRelative(t *testing.T) {
	invs, err := parse("owned_by($.Path.User),category_matches($.Body.Category),needs(.Peer)")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(invs) != 3 {
		t.Fatalf("expected 3 invocations, got %d", len(invs))
	}
	if invs[0].Name != "owned_by" || len(invs[0].Args) != 1 || !invs[0].Args[0].IsAbsolute {
		t.Fatalf("first invocation not parsed as absolute field ref: %+v", invs[0])
	}
	if invs[1].Name != "category_matches" || !invs[1].Args[0].IsAbsolute {
		t.Fatalf("second invocation not parsed as absolute")
	}
	if invs[2].Name != "needs" || invs[2].Args[0].IsAbsolute {
		t.Fatalf("third invocation not parsed as relative")
	}
}

func TestParseTag_Literals(t *testing.T) {
	invs, err := parse("opt('x'),num(12.5),flag(true),none(null),flag(false)")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(invs) != 5 {
		t.Fatalf("expected 5 invocations, got %d", len(invs))
	}
	if invs[0].Args[0].Kind != argString || invs[0].Args[0].Str != "x" {
		t.Fatal("string literal parse failed")
	}
	if invs[1].Args[0].Kind != argNumber || invs[1].Args[0].Num != 12.5 {
		t.Fatal("number literal parse failed")
	}
	if invs[2].Args[0].Kind != argBool || invs[2].Args[0].Bool != true {
		t.Fatal("bool literal parse failed")
	}
	if invs[3].Args[0].Kind != argNull {
		t.Fatal("null literal parse failed")
	}
	if invs[4].Args[0].Kind != argBool || invs[4].Args[0].Bool != false {
		t.Fatal("bool false literal parse failed")
	}
}

func TestParseTag_Errors(t *testing.T) {
	// Unmatched parenthesis
	if _, err := parse("foo(bar"); err == nil {
		t.Fatal("expected error for unmatched parenthesis")
	}
	// Invalid argument (no prefix)
	if _, err := parse("x(User)"); err == nil {
		t.Fatal("expected error for invalid arg without prefix")
	}
	// Invalid absolute prefix only
	if _, err := parse("x($.)"); err == nil {
		t.Fatal("expected error for invalid absolute ref")
	}
	// Empty string returns no invocations
	invs, err := parse("")
	if err != nil || len(invs) != 0 {
		t.Fatalf("expected empty result, got %v, err=%v", invs, err)
	}
}

func TestParseTag_NoArgsAndComplexStrings(t *testing.T) {
	// Invocation without args
	invs, err := parse("admin,owned_by($.Path.User)")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(invs) != 2 || invs[0].Name != "admin" || len(invs[0].Args) != 0 {
		t.Fatalf("expected admin without args, got %+v", invs)
	}

	// Quoted commas and parentheses
	invs, err = parse("x('a,b(c)'),y(\"c,d\")")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if invs[0].Args[0].Str != "a,b(c)" || invs[1].Args[0].Str != "c,d" {
		t.Fatalf("failed to parse quoted strings with commas: %+v", invs)
	}

	// Signed numbers
	invs, err = parse("n(-12.5),p(+1)")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if invs[0].Args[0].Num != -12.5 || invs[1].Args[0].Num != 1 {
		t.Fatalf("failed to parse signed numbers: %+v", invs)
	}

	// Unmatched right parenthesis
	if _, err := parse("foo()),bar()"); err == nil {
		t.Fatal("expected error for unmatched right parenthesis")
	}
	// Unbalanced parentheses in args string handled by parseArgs/splitTopLevel
	if _, err := parse("a((1,2)"); err == nil {
		t.Fatal("expected error for unbalanced parentheses inside args")
	}

	// Invalid identifier in field ref
	if _, err := parse("x($.Bad-Name)"); err == nil {
		t.Fatal("expected error for invalid identifier with hyphen")
	}
	// Invalid identifier in relative field ref
	if _, err := parse("x(.Bad-Name)"); err == nil {
		t.Fatal("expected error for invalid relative identifier with hyphen")
	}
}

func TestParseTag_EmptyPartsAndNumberEdges(t *testing.T) {
	// Empty part should be skipped
	invs, err := parse("a(), , b()")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(invs) != 2 || invs[0].Name != "a" || invs[1].Name != "b" {
		t.Fatalf("expected two invocations skipping empty, got %+v", invs)
	}

	// Number with trailing dot
	invs, err = parse("n(1.)")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if invs[0].Args[0].Num != 1.0 {
		t.Fatalf("expected 1.0, got %v", invs[0].Args[0].Num)
	}

	// Invalid number with internal plus to hit isNumberLike branch
	if _, err := parse("n(1+2)"); err == nil {
		t.Fatal("expected error for invalid number pattern")
	}

	// Number-like but invalid float (two dots) to hit parseFloat failure branch
	if _, err := parse("n(1..2)"); err == nil {
		t.Fatal("expected error for invalid float with two dots")
	}

	// Invalid identifier starting with digit
	if _, err := parse("x($.1abc)"); err == nil {
		t.Fatal("expected error for invalid identifier starting with digit")
	}
}

func TestSplitInvocation_Whitespace(t *testing.T) {
	invs, err := parse("  foo ( 'x' )  ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(invs) != 1 || invs[0].Name != "foo" || invs[0].Args[0].Str != "x" {
		t.Fatalf("whitespace handling failed: %+v", invs)
	}
}
