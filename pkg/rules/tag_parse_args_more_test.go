package rules

import "testing"

func TestParseArgs_OnlyWhitespaceAndSeparators(t *testing.T) {
	toks, err := parseArgs(" ,  , \t ,\n ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(toks) != 0 {
		t.Fatalf("expected 0 tokens, got %d", len(toks))
	}
}

func TestParseArgs_MixedValidAndEmptyParts(t *testing.T) {
	// Test that splitTopLevel properly filters out empty parts between commas
	// Input: "true, , false,   , null" where empty parts are filtered by splitTopLevel
	toks, err := parseArgs("true, , false,   , null")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(toks))
	}

	// Verify the parsed tokens are correct (non-empty parts)
	if toks[0].Kind != argBool || !toks[0].Bool {
		t.Fatalf("expected first token to be true bool, got %v", toks[0])
	}
	if toks[1].Kind != argBool || toks[1].Bool {
		t.Fatalf("expected second token to be false bool, got %v", toks[1])
	}
	if toks[2].Kind != argNull {
		t.Fatalf("expected third token to be null, got %v", toks[2])
	}
}
