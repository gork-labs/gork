package rules

import "testing"

// TestParseArgs_SplitTopLevelError tests the missing error branch in parseArgs
// where splitTopLevel returns an error
func TestParseArgs_SplitTopLevelError(t *testing.T) {
	// Test with unbalanced quotes that would cause splitTopLevel to fail
	if _, err := parseArgs("'unclosed"); err == nil {
		t.Fatalf("expected splitTopLevel error for unclosed quote")
	}
}

// TestParseArgs_ParseSingleArgError tests the missing error branch in parseArgs
// where parseSingleArg returns an error for a non-empty trimmed part
func TestParseArgs_ParseSingleArgError(t *testing.T) {
	// Test with invalid field reference that causes parseSingleArg to fail
	if _, err := parseArgs("$.123invalid"); err == nil {
		t.Fatalf("expected parseSingleArg error for invalid field reference")
	}
}
