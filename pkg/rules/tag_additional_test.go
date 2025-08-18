package rules

import "testing"

func TestParseArgs_EmptyAndInvalid(t *testing.T) {
	// empty args string -> empty slice
	toks, err := parseArgs("")
	if err != nil || len(toks) != 0 {
		t.Fatalf("expected empty slice, got %v, %v", toks, err)
	}
	// whitespace only -> empty slice
	toks, err = parseArgs("   ")
	if err != nil || len(toks) != 0 {
		t.Fatalf("expected empty slice, got %v, %v", toks, err)
	}
	// invalid token -> error
	if _, err := parseArgs("!"); err == nil {
		t.Fatalf("expected error for invalid arg")
	}
}

func TestTryParseContextVar_InvalidNames(t *testing.T) {
	// empty name
	if _, ok, err := tryParseContextVar("$"); err == nil || ok {
		t.Fatalf("expected error for empty var name")
	}
	// starts with digit
	if _, ok, err := tryParseContextVar("$1abc"); err == nil || ok {
		t.Fatalf("expected error for invalid identifier")
	}
}
