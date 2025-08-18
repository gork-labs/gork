package rules

import (
	"fmt"
	"strings"
)

// tokKind represents the kind of token in rule expressions.
type tokKind int

const (
	tkIdent tokKind = iota
	tkLPar
	tkRPar
	tkComma
	tkAnd
	tkOr
	tkNot
	tkString
	tkNumber
	tkBool
	tkNull
	tkFieldRef
	tkContextVar
	tkEOF
)

// token represents a single token in rule expressions.
type token struct {
	kind tokKind
	text string
}

// tokenize parses a string into a sequence of tokens.
func tokenize(s string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(s) {
		skipWhitespace(s, &i)
		if i >= len(s) {
			break
		}
		if tok, advanced, err := scanSpecialToken(s, &i); advanced {
			if err != nil {
				return nil, err
			}
			if tok.kind != 0 {
				toks = append(toks, tok)
			}
			continue
		}
		ident, ok := scanIdent(s, &i)
		if !ok {
			return nil, fmt.Errorf("unexpected character: %q", s[i])
		}
		toks = append(toks, classifyIdentToken(ident))
	}
	toks = append(toks, token{kind: tkEOF})
	return toks, nil
}

// skipWhitespace advances the position past any whitespace characters.
func skipWhitespace(s string, i *int) {
	for *i < len(s) {
		c := s[*i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			*i++
			continue
		}
		break
	}
}

// isIdentChar returns true if the character can be part of an identifier.
func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '$'
}

// scanIdent scans an identifier from the current position.
func scanIdent(s string, i *int) (string, bool) {
	start := *i
	for *i < len(s) {
		if isIdentChar(s[*i]) {
			*i++
			continue
		}
		break
	}
	if *i == start {
		return "", false
	}
	return s[start:*i], true
}

// scanSpecialToken scans special tokens like parentheses, operators, and string literals.
func scanSpecialToken(s string, i *int) (token, bool, error) {
	switch s[*i] {
	case '(':
		*i++
		return token{kind: tkLPar, text: "("}, true, nil
	case ')':
		*i++
		return token{kind: tkRPar, text: ")"}, true, nil
	case ',':
		*i++
		return token{kind: tkComma, text: ","}, true, nil
	case '!':
		*i++
		return token{kind: tkNot, text: "!"}, true, nil
	case '\'', '"':
		lit, err := scanQuotedString(s, i)
		if err != nil {
			return token{}, true, err
		}
		return token{kind: tkString, text: lit}, true, nil
	default:
		return token{}, false, nil
	}
}

// scanQuotedString scans a quoted string literal.
func scanQuotedString(s string, i *int) (string, error) {
	quote := s[*i]
	*i++
	start := *i
	for *i < len(s) && s[*i] != quote {
		*i++
	}
	if *i >= len(s) {
		return "", fmt.Errorf("unterminated string literal")
	}
	lit := s[start:*i]
	*i++
	return lit, nil
}

// classifyIdentToken classifies an identifier token based on its content.
func classifyIdentToken(ident string) token {
	low := strings.ToLower(ident)
	switch low {
	case "and", "&&":
		return token{kind: tkAnd, text: ident}
	case "or", "||":
		return token{kind: tkOr, text: ident}
	case "not":
		return token{kind: tkNot, text: ident}
	case "true", "false":
		return token{kind: tkBool, text: low}
	case "null":
		return token{kind: tkNull, text: low}
	default:
		switch {
		case strings.HasPrefix(ident, "$.") || strings.HasPrefix(ident, "."):
			return token{kind: tkFieldRef, text: ident}
		case strings.HasPrefix(ident, "$"):
			return token{kind: tkContextVar, text: ident}
		default:
			return token{kind: tkIdent, text: ident}
		}
	}
}
