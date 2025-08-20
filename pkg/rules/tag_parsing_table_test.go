package rules

import "testing"

// TestTagParsingTableDriven uses table-driven tests for tag parsing scenarios
func TestTagParsingTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedLen int // For successful parses, expected number of tokens
		validate    func(t *testing.T, tokens []argToken, err error)
	}{
		{
			name:        "AllTokenKinds_Success",
			input:       "'s', 12, true, false, null, $.Path.Id, .Local.Name, $ctx",
			expectError: false,
			expectedLen: 8,
		},
		{
			name:        "InvalidAbsoluteFieldRef",
			input:       "$.",
			expectError: true,
		},
		{
			name:        "InvalidRelativeFieldRef",
			input:       ".",
			expectError: true,
		},
		{
			name:        "InvalidContextVar",
			input:       "$",
			expectError: true,
		},
		{
			name:        "UnbalancedQuotes_SplitTopLevelError",
			input:       "'unclosed",
			expectError: true,
		},
		{
			name:        "InvalidFieldReference_ParseSingleArgError",
			input:       "$.123invalid",
			expectError: true,
		},
		{
			name:        "UnbalancedDelimiters",
			input:       "('x'",
			expectError: true,
		},
		{
			name:        "EmptyInput_Success",
			input:       "",
			expectError: false,
			expectedLen: 0,
		},
		{
			name:        "OnlyWhitespace_Success",
			input:       "   ",
			expectError: false,
			expectedLen: 0,
		},
		{
			name:        "SingleString_Success",
			input:       "'hello'",
			expectError: false,
			expectedLen: 1,
			validate: func(t *testing.T, tokens []argToken, err error) {
				if len(tokens) == 1 && tokens[0].Kind == argString && tokens[0].Str != "hello" {
					t.Errorf("Expected string 'hello', got '%s'", tokens[0].Str)
				}
			},
		},
		{
			name:        "SingleNumber_Success",
			input:       "42",
			expectError: false,
			expectedLen: 1,
			validate: func(t *testing.T, tokens []argToken, err error) {
				if len(tokens) == 1 && tokens[0].Kind == argNumber && tokens[0].Num != 42 {
					t.Errorf("Expected number 42, got %f", tokens[0].Num)
				}
			},
		},
		{
			name:        "SingleBoolTrue_Success",
			input:       "true",
			expectError: false,
			expectedLen: 1,
			validate: func(t *testing.T, tokens []argToken, err error) {
				if len(tokens) == 1 && tokens[0].Kind == argBool && !tokens[0].Bool {
					t.Errorf("Expected bool true, got %v", tokens[0].Bool)
				}
			},
		},
		{
			name:        "SingleBoolFalse_Success",
			input:       "false",
			expectError: false,
			expectedLen: 1,
			validate: func(t *testing.T, tokens []argToken, err error) {
				if len(tokens) == 1 && tokens[0].Kind == argBool && tokens[0].Bool {
					t.Errorf("Expected bool false, got %v", tokens[0].Bool)
				}
			},
		},
		{
			name:        "AbsoluteFieldRef_Success",
			input:       "$.Path.Id",
			expectError: false,
			expectedLen: 1,
			validate: func(t *testing.T, tokens []argToken, err error) {
				if len(tokens) == 1 {
					token := tokens[0]
					if token.Kind != argFieldRef {
						t.Errorf("Expected argFieldRef, got %v", token.Kind)
					}
					if !token.IsAbsolute {
						t.Error("Expected absolute field reference")
					}
					if len(token.Segments) != 2 || token.Segments[0] != "Path" || token.Segments[1] != "Id" {
						t.Errorf("Expected segments [Path, Id], got %v", token.Segments)
					}
				}
			},
		},
		{
			name:        "RelativeFieldRef_Success",
			input:       ".Local.Name",
			expectError: false,
			expectedLen: 1,
			validate: func(t *testing.T, tokens []argToken, err error) {
				if len(tokens) == 1 {
					token := tokens[0]
					if token.Kind != argFieldRef {
						t.Errorf("Expected argFieldRef, got %v", token.Kind)
					}
					if token.IsAbsolute {
						t.Error("Expected relative field reference")
					}
					if len(token.Segments) != 2 || token.Segments[0] != "Local" || token.Segments[1] != "Name" {
						t.Errorf("Expected segments [Local, Name], got %v", token.Segments)
					}
				}
			},
		},
		{
			name:        "ContextVar_Success",
			input:       "$ctx",
			expectError: false,
			expectedLen: 1,
			validate: func(t *testing.T, tokens []argToken, err error) {
				if len(tokens) == 1 {
					token := tokens[0]
					if token.Kind != argContextVar {
						t.Errorf("Expected argContextVar, got %v", token.Kind)
					}
					if token.ContextVar != "ctx" {
						t.Errorf("Expected context var 'ctx', got '%s'", token.ContextVar)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parseArgs(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(tokens) != tt.expectedLen {
					t.Errorf("Expected %d tokens, got %d", tt.expectedLen, len(tokens))
				}
			}

			if tt.validate != nil && !tt.expectError {
				tt.validate(t, tokens, err)
			}
		})
	}
}

// TestHandleCloseParen tests the handleCloseParen function for all branches
func TestHandleCloseParen(t *testing.T) {
	t.Run("InsideSingleQuotes_NoOp", func(t *testing.T) {
		depth := 1
		err := handleCloseParen(&depth, true, false, "test')'")

		if err != nil {
			t.Errorf("Expected no error when inside single quotes, got: %v", err)
		}
		if depth != 1 {
			t.Errorf("Expected depth to remain unchanged (1), got: %d", depth)
		}
	})

	t.Run("InsideDoubleQuotes_NoOp", func(t *testing.T) {
		depth := 1
		err := handleCloseParen(&depth, false, true, "test\")\"")

		if err != nil {
			t.Errorf("Expected no error when inside double quotes, got: %v", err)
		}
		if depth != 1 {
			t.Errorf("Expected depth to remain unchanged (1), got: %d", depth)
		}
	})

	t.Run("DepthZero_UnmatchedParen", func(t *testing.T) {
		depth := 0
		err := handleCloseParen(&depth, false, false, "test)")

		if err == nil {
			t.Error("Expected error for unmatched closing paren, got nil")
		}
		expectedMsg := "rules: unmatched ')' in \"test)\""
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
		if depth != 0 {
			t.Errorf("Expected depth to remain 0, got: %d", depth)
		}
	})

	t.Run("DepthGreaterThanZero_DecrementDepth", func(t *testing.T) {
		depth := 2
		err := handleCloseParen(&depth, false, false, "test(x,y)")

		if err != nil {
			t.Errorf("Expected no error for matched paren, got: %v", err)
		}
		if depth != 1 {
			t.Errorf("Expected depth to be decremented to 1, got: %d", depth)
		}
	})

	t.Run("DepthOne_DecrementToZero", func(t *testing.T) {
		depth := 1
		err := handleCloseParen(&depth, false, false, "test(x)")

		if err != nil {
			t.Errorf("Expected no error for matched paren, got: %v", err)
		}
		if depth != 0 {
			t.Errorf("Expected depth to be decremented to 0, got: %d", depth)
		}
	})
}

// TestSplitTopLevel_CloseParenError tests the error path in splitTopLevel when handleCloseParen fails
func TestSplitTopLevel_CloseParenError(t *testing.T) {
	t.Run("UnmatchedClosingParen", func(t *testing.T) {
		// Test case: unmatched closing parenthesis should trigger error in handleCloseParen
		input := "test)" // No opening paren, so depth=0 when we hit ')'

		parts, err := splitTopLevel(input)

		// Should get the error from handleCloseParen
		if err == nil {
			t.Error("Expected error for unmatched closing paren, got nil")
		}

		// Should return nil for parts when error occurs
		if parts != nil {
			t.Errorf("Expected nil parts on error, got: %v", parts)
		}

		// Check the error message matches what handleCloseParen returns
		expectedMsg := "rules: unmatched ')' in \"test)\""
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("UnmatchedClosingParenInMiddle", func(t *testing.T) {
		// Test case: unmatched closing paren in middle of string
		input := "a,b),c" // Closing paren without matching opening paren

		parts, err := splitTopLevel(input)

		if err == nil {
			t.Error("Expected error for unmatched closing paren, got nil")
		}

		if parts != nil {
			t.Errorf("Expected nil parts on error, got: %v", parts)
		}

		expectedMsg := "rules: unmatched ')' in \"a,b),c\""
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("ValidClosingParen_NoError", func(t *testing.T) {
		// Test case: properly matched parentheses should work fine
		input := "func(a,b),other"

		parts, err := splitTopLevel(input)

		if err != nil {
			t.Errorf("Expected no error for matched parens, got: %v", err)
		}

		expectedParts := []string{"func(a,b)", "other"}
		if len(parts) != len(expectedParts) {
			t.Errorf("Expected %d parts, got %d: %v", len(expectedParts), len(parts), parts)
		}

		for i, expected := range expectedParts {
			if i >= len(parts) || parts[i] != expected {
				t.Errorf("Expected part[%d] = %q, got %q", i, expected, parts[i])
			}
		}
	})
}
