package rules

import "testing"

func TestParseArgs_UnbalancedDelimiters(t *testing.T) {
	if _, err := parseArgs("('x'"); err == nil {
		t.Fatalf("expected splitTopLevel unbalanced error")
	}
}
