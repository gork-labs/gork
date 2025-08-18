package rules

import "testing"

func TestTokenize_WhitespaceVariants(t *testing.T) {
	s := " \t\rand  \n or \t not ( true ) , 'x' "
	if _, err := tokenize(s); err != nil {
		t.Fatalf("tokenize error with whitespace variants: %v", err)
	}
}
