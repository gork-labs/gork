package rules

import "testing"

func TestParseArgs_AllKindsAndErrors(t *testing.T) {
	// happy path: all kinds
	toks, err := parseArgs("'s', 12, true, false, null, $.Path.Id, .Local.Name, $ctx")
	if err != nil || len(toks) != 8 {
		t.Fatalf("expected 8 tokens, got %v err=%v", len(toks), err)
	}
	// invalid field ref segment
	if _, err := parseArgs("$."); err == nil {
		t.Fatalf("expected error for invalid absolute field ref")
	}
	// invalid relative ref segment
	if _, err := parseArgs("."); err == nil {
		t.Fatalf("expected error for invalid relative field ref")
	}
	// invalid context var name
	if _, err := parseArgs("$"); err == nil {
		t.Fatalf("expected error for invalid context var")
	}
}
