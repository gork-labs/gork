package rules

import "testing"

func TestIsNumberLike(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Empty string cases
		{
			name:  "EmptyString_ReturnsFalse",
			input: "",
			want:  false,
		},

		// Valid positive numbers
		{
			name:  "SingleDigit_ReturnsTrue",
			input: "5",
			want:  true,
		},
		{
			name:  "MultipleDigits_ReturnsTrue",
			input: "123",
			want:  true,
		},
		{
			name:  "PositiveSignWithDigit_ReturnsTrue",
			input: "+5",
			want:  true,
		},
		{
			name:  "PositiveSignWithMultipleDigits_ReturnsTrue",
			input: "+123",
			want:  true,
		},

		// Valid negative numbers
		{
			name:  "NegativeSignWithDigit_ReturnsTrue",
			input: "-5",
			want:  true,
		},
		{
			name:  "NegativeSignWithMultipleDigits_ReturnsTrue",
			input: "-123",
			want:  true,
		},

		// Valid decimal numbers
		{
			name:  "DecimalWithDigits_ReturnsTrue",
			input: "5.2",
			want:  true,
		},
		{
			name:  "DecimalWithLeadingZero_ReturnsTrue",
			input: "0.5",
			want:  true,
		},
		{
			name:  "DecimalWithTrailingZero_ReturnsTrue",
			input: "5.0",
			want:  true,
		},
		{
			name:  "PositiveSignWithDecimal_ReturnsTrue",
			input: "+5.2",
			want:  true,
		},
		{
			name:  "NegativeSignWithDecimal_ReturnsTrue",
			input: "-5.2",
			want:  true,
		},
		{
			name:  "LeadingDot_ReturnsTrue",
			input: ".5",
			want:  true,
		},
		{
			name:  "TrailingDot_ReturnsTrue",
			input: "5.",
			want:  true,
		},
		{
			name:  "PositiveSignWithLeadingDot_ReturnsTrue",
			input: "+.5",
			want:  true,
		},
		{
			name:  "NegativeSignWithLeadingDot_ReturnsTrue",
			input: "-.5",
			want:  true,
		},
		{
			name:  "MultipleDecimalPoints_ReturnsTrue",
			input: "1.2.3",
			want:  true,
		},

		// Invalid sign placement
		{
			name:  "SignInMiddle_ReturnsFalse",
			input: "1+2",
			want:  false,
		},
		{
			name:  "SignAtEnd_ReturnsFalse",
			input: "12+",
			want:  false,
		},
		{
			name:  "NegativeSignInMiddle_ReturnsFalse",
			input: "1-2",
			want:  false,
		},
		{
			name:  "NegativeSignAtEnd_ReturnsFalse",
			input: "12-",
			want:  false,
		},

		// No digits cases
		{
			name:  "OnlyDot_ReturnsFalse",
			input: ".",
			want:  false,
		},
		{
			name:  "OnlyPositiveSign_ReturnsFalse",
			input: "+",
			want:  false,
		},
		{
			name:  "OnlyNegativeSign_ReturnsFalse",
			input: "-",
			want:  false,
		},
		{
			name:  "SignAndDotNoDigits_ReturnsFalse",
			input: "+.",
			want:  false,
		},
		{
			name:  "NegativeSignAndDotNoDigits_ReturnsFalse",
			input: "-.",
			want:  false,
		},
		{
			name:  "MultipleDots_ReturnsFalse",
			input: "...",
			want:  false,
		},

		// Invalid characters
		{
			name:  "ContainsLetter_ReturnsFalse",
			input: "1a2",
			want:  false,
		},
		{
			name:  "ContainsSpace_ReturnsFalse",
			input: "1 2",
			want:  false,
		},
		{
			name:  "ContainsSpecialChar_ReturnsFalse",
			input: "1@2",
			want:  false,
		},
		{
			name:  "OnlyLetters_ReturnsFalse",
			input: "abc",
			want:  false,
		},
		{
			name:  "StartsWithLetter_ReturnsFalse",
			input: "a123",
			want:  false,
		},
		{
			name:  "EndsWithLetter_ReturnsFalse",
			input: "123a",
			want:  false,
		},

		// Edge cases with Unicode
		{
			name:  "UnicodeDigits_ReturnsTrue",
			input: "１２３", // Full-width digits (Unicode)
			want:  true,
		},
		{
			name:  "UnicodeNonDigit_ReturnsFalse",
			input: "1α2",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNumberLike(tt.input)
			if got != tt.want {
				t.Errorf("isNumberLike(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsIdent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Empty string case
		{
			name:  "EmptyString_ReturnsFalse",
			input: "",
			want:  false,
		},

		// Valid identifiers starting with letter
		{
			name:  "SingleLetter_ReturnsTrue",
			input: "a",
			want:  true,
		},
		{
			name:  "UppercaseLetter_ReturnsTrue",
			input: "A",
			want:  true,
		},
		{
			name:  "LetterFollowedByLetter_ReturnsTrue",
			input: "abc",
			want:  true,
		},
		{
			name:  "LetterFollowedByDigit_ReturnsTrue",
			input: "a1",
			want:  true,
		},
		{
			name:  "LetterFollowedByUnderscore_ReturnsTrue",
			input: "a_",
			want:  true,
		},
		{
			name:  "MixedLettersDigitsUnderscores_ReturnsTrue",
			input: "abc_123",
			want:  true,
		},
		{
			name:  "CamelCase_ReturnsTrue",
			input: "camelCase",
			want:  true,
		},
		{
			name:  "PascalCase_ReturnsTrue",
			input: "PascalCase",
			want:  true,
		},

		// Valid identifiers starting with underscore
		{
			name:  "SingleUnderscore_ReturnsTrue",
			input: "_",
			want:  true,
		},
		{
			name:  "UnderscoreFollowedByLetter_ReturnsTrue",
			input: "_a",
			want:  true,
		},
		{
			name:  "UnderscoreFollowedByDigit_ReturnsTrue",
			input: "_1",
			want:  true,
		},
		{
			name:  "UnderscoreFollowedByUnderscore_ReturnsTrue",
			input: "__",
			want:  true,
		},
		{
			name:  "LeadingUnderscoreWithMixed_ReturnsTrue",
			input: "_private123",
			want:  true,
		},
		{
			name:  "MultipleUnderscores_ReturnsTrue",
			input: "__dunder__",
			want:  true,
		},

		// Invalid first characters
		{
			name:  "StartsWithDigit_ReturnsFalse",
			input: "1abc",
			want:  false,
		},
		{
			name:  "StartsWithSpecialChar_ReturnsFalse",
			input: "@abc",
			want:  false,
		},
		{
			name:  "StartsWithSpace_ReturnsFalse",
			input: " abc",
			want:  false,
		},
		{
			name:  "StartsWithHyphen_ReturnsFalse",
			input: "-abc",
			want:  false,
		},
		{
			name:  "StartsWithDot_ReturnsFalse",
			input: ".abc",
			want:  false,
		},

		// Invalid characters in body
		{
			name:  "ContainsSpace_ReturnsFalse",
			input: "abc def",
			want:  false,
		},
		{
			name:  "ContainsHyphen_ReturnsFalse",
			input: "abc-def",
			want:  false,
		},
		{
			name:  "ContainsDot_ReturnsFalse",
			input: "abc.def",
			want:  false,
		},
		{
			name:  "ContainsSpecialChar_ReturnsFalse",
			input: "abc@def",
			want:  false,
		},
		{
			name:  "ContainsPunctuation_ReturnsFalse",
			input: "abc!",
			want:  false,
		},
		{
			name:  "ContainsParentheses_ReturnsFalse",
			input: "abc()",
			want:  false,
		},

		// Only invalid characters
		{
			name:  "OnlyDigits_ReturnsFalse",
			input: "123",
			want:  false,
		},
		{
			name:  "OnlySpecialChars_ReturnsFalse",
			input: "@#$",
			want:  false,
		},
		{
			name:  "OnlySpaces_ReturnsFalse",
			input: "   ",
			want:  false,
		},

		// Unicode cases
		{
			name:  "UnicodeLetterStart_ReturnsTrue",
			input: "αβγ",
			want:  true,
		},
		{
			name:  "UnicodeLetterWithDigit_ReturnsTrue",
			input: "α1",
			want:  true,
		},
		{
			name:  "UnicodeDigitStart_ReturnsFalse",
			input: "１abc", // Full-width digit
			want:  false,
		},
		{
			name:  "UnicodeSpecialChar_ReturnsFalse",
			input: "abc♠",
			want:  false,
		},

		// Edge cases
		{
			name:  "VeryLongIdentifier_ReturnsTrue",
			input: "veryLongIdentifierWithManyCharacters_123",
			want:  true,
		},
		{
			name:  "SingleDigit_ReturnsFalse",
			input: "5",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIdent(tt.input)
			if got != tt.want {
				t.Errorf("isIdent(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTryParseNumber(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectSuccess  bool
		expectedNumber float64
	}{
		{
			name:           "ValidInteger_Success",
			input:          "42",
			expectSuccess:  true,
			expectedNumber: 42,
		},
		{
			name:           "ValidFloat_Success",
			input:          "3.14",
			expectSuccess:  true,
			expectedNumber: 3.14,
		},
		{
			name:           "ValidNegative_Success",
			input:          "-5.2",
			expectSuccess:  true,
			expectedNumber: -5.2,
		},
		{
			name:          "NotNumberLike_ReturnsFalse",
			input:         "abc",
			expectSuccess: false,
		},
		{
			name:          "NumberLikeButInvalidFormat_ReturnsFalse",
			input:         "1.2.3",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, success := tryParseNumber(tt.input)

			if success != tt.expectSuccess {
				t.Errorf("tryParseNumber(%q) success = %v, want %v", tt.input, success, tt.expectSuccess)
			}

			if tt.expectSuccess {
				if token.Kind != argNumber {
					t.Errorf("Expected argNumber kind, got %v", token.Kind)
				}
				if token.Num != tt.expectedNumber {
					t.Errorf("Expected number %f, got %f", tt.expectedNumber, token.Num)
				}
			} else {
				if token.Kind != argInvalid || token.Num != 0 {
					t.Errorf("Expected zero-value token on failure, got %+v", token)
				}
			}
		})
	}
}
