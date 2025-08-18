package rules

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// argKind enumerates possible argument token types parsed from a rule tag.
type argKind int

const (
	// argFieldRef represents a field reference like $.Path or .Header.
	argFieldRef argKind = iota
	// argString is a quoted string literal.
	argString
	// argNumber is a numeric literal (float64).
	argNumber
	// argBool is a boolean literal: true or false.
	argBool
	// argNull represents a null literal.
	argNull
	// argContextVar represents a context variable like $current_user.
	// Context variables are prefixed with $ but not $. (which is field reference).
	// They provide per-request dynamic values accessible across all rules.
	// Example: owned_by($current_user), has_permission($user_role).
	argContextVar
)

// argToken represents a parsed argument token.
type argToken struct {
	Kind argKind
	// For field refs
	IsAbsolute bool
	Segments   []string
	// For context vars
	ContextVar string
	// For literals
	Str  string
	Num  float64
	Bool bool
}

// invocation represents a single rule invocation parsed from a tag.
type invocation struct {
	Name string
	Args []argToken
}

// parse parses the content of a `rule:"..."` tag into invocations.
// Grammar:
//
//	rule = invocation *("," invocation)
//	invocation = ident ["(" [args] ")"]
//	args = arg *("," arg)
//	arg = FieldRef | String | Number | Bool | Null
//	FieldRef = Section '.' Identifier {'.' Identifier}
//
// Sections: Path, Query, Body, Headers, Cookies.
func parse(s string) ([]invocation, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts, err := splitTopLevel(s, ',')
	if err != nil {
		return nil, err
	}
	invs := make([]invocation, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		name, argstr, hasArgs, _ := splitInvocation(p)
		inv := invocation{Name: name}
		if hasArgs {
			argTokens, err := parseArgs(argstr)
			if err != nil {
				return nil, err
			}
			inv.Args = argTokens
		}
		invs = append(invs, inv)
	}
	return invs, nil
}

func splitInvocation(s string) (name, argstr string, hasArgs bool, err error) {
	// find first '(' if any
	i := strings.IndexRune(s, '(')
	if i < 0 {
		return s, "", false, nil
	}
	if !strings.HasSuffix(s, ")") {
		return "", "", false, fmt.Errorf("rules: unmatched '(' in %q", s)
	}
	return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1 : len(s)-1]), true, nil
}

// parseArgs parses a comma-separated list of rule arguments.
// Supports field refs $.X.Y and .X.Y, string/number/bool/null literals, and $var context variables.
// Example: owned_by($current_user), has_permission($user_role).
func parseArgs(s string) ([]argToken, error) {
	parts, err := splitTopLevel(s, ',')
	if err != nil {
		return nil, err
	}
	out := make([]argToken, 0, len(parts))
	for _, p := range parts {
		tok, err := parseSingleArg(p)
		if err != nil {
			return nil, err
		}
		out = append(out, tok)
	}
	return out, nil
}

func parseSingleArg(p string) (argToken, error) {
	if tok, ok, err := tryParseFieldRef(p); err != nil {
		return argToken{}, err
	} else if ok {
		return tok, nil
	}
	if tok, ok, err := tryParseContextVar(p); err != nil {
		return argToken{}, err
	} else if ok {
		return tok, nil
	}
	if tok, ok := tryParseString(p); ok {
		return tok, nil
	}
	if tok, ok := tryParseBool(p); ok {
		return tok, nil
	}
	if tok, ok := tryParseNull(p); ok {
		return tok, nil
	}
	if tok, ok := tryParseNumber(p); ok {
		return tok, nil
	}
	return argToken{}, fmt.Errorf("rules: invalid argument: %q", p)
}

func tryParseString(s string) (argToken, bool) {
	if (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) || (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) {
		unq := s[1 : len(s)-1]
		return argToken{Kind: argString, Str: unq}, true
	}
	return argToken{}, false
}

func tryParseBool(s string) (argToken, bool) {
	if s == "true" {
		return argToken{Kind: argBool, Bool: true}, true
	}
	if s == "false" {
		return argToken{Kind: argBool, Bool: false}, true
	}
	return argToken{}, false
}

func tryParseNull(s string) (argToken, bool) {
	if s == "null" {
		return argToken{Kind: argNull}, true
	}
	return argToken{}, false
}

func tryParseNumber(s string) (argToken, bool) {
	if !isNumberLike(s) {
		return argToken{}, false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return argToken{Kind: argNumber, Num: f}, true
	}
	return argToken{}, false
}

func tryParseFieldRef(s string) (argToken, bool, error) {
	if strings.HasPrefix(s, "$.") {
		segs := strings.Split(s[2:], ".")
		if err := validateSegments(segs); err != nil {
			return argToken{}, false, err
		}
		return argToken{Kind: argFieldRef, IsAbsolute: true, Segments: segs}, true, nil
	}
	if strings.HasPrefix(s, ".") {
		segs := strings.Split(strings.TrimPrefix(s, "."), ".")
		if err := validateSegments(segs); err != nil {
			return argToken{}, false, err
		}
		return argToken{Kind: argFieldRef, IsAbsolute: false, Segments: segs}, true, nil
	}
	return argToken{}, false, nil
}

func tryParseContextVar(s string) (argToken, bool, error) {
	if strings.HasPrefix(s, "$") && !strings.HasPrefix(s, "$.") {
		varName := strings.TrimPrefix(s, "$")
		if varName == "" || !isIdent(varName) {
			return argToken{}, false, fmt.Errorf("rules: invalid context variable name %q", varName)
		}
		return argToken{Kind: argContextVar, ContextVar: varName}, true, nil
	}
	return argToken{}, false, nil
}

func validateSegments(segs []string) error {
	for _, seg := range segs {
		if seg == "" || !isIdent(seg) {
			return fmt.Errorf("rules: invalid field path segment %q", seg)
		}
	}
	return nil
}

func isNumberLike(s string) bool {
	if s == "" {
		return false
	}
	hasDigit := false
	for i, r := range s {
		if r == '+' || r == '-' {
			if i != 0 {
				return false
			}
			continue
		}
		if r == '.' {
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
		hasDigit = true
	}
	return hasDigit
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// no section validation; only absolute $. and relative . prefixes are allowed

// splitTopLevel splits by sep while respecting parentheses and quotes.
func splitTopLevel(s string, sep rune) ([]string, error) {
	var (
		parts    []string
		cur      strings.Builder
		depth    int
		inSingle bool
		inDouble bool
	)
	pushPart := func() {
		p := strings.TrimSpace(cur.String())
		if p != "" {
			parts = append(parts, p)
		}
		cur.Reset()
	}
	for _, r := range s {
		switch r {
		case '\'':
			handleSingleQuote(&inSingle, inDouble)
			cur.WriteRune(r)
		case '"':
			handleDoubleQuote(&inDouble, inSingle)
			cur.WriteRune(r)
		case '(':
			handleOpenParen(&depth, inSingle, inDouble)
			cur.WriteRune(r)
		case ')':
			if err := handleCloseParen(&depth, inSingle, inDouble, s); err != nil {
				return nil, err
			}
			cur.WriteRune(r)
		default:
			if shouldSplit(r, depth, inSingle, inDouble, sep) {
				pushPart()
				continue
			}
			cur.WriteRune(r)
		}
	}
	if depth != 0 || inSingle || inDouble {
		return nil, fmt.Errorf("rules: unbalanced delimiters in %q", s)
	}
	pushPart()
	return parts, nil
}

func handleSingleQuote(inSingle *bool, inDouble bool) {
	if !inDouble {
		*inSingle = !*inSingle
	}
}

func handleDoubleQuote(inDouble *bool, inSingle bool) {
	if !inSingle {
		*inDouble = !*inDouble
	}
}

func handleOpenParen(depth *int, inSingle, inDouble bool) {
	if !inSingle && !inDouble {
		(*depth)++
	}
}

func handleCloseParen(depth *int, inSingle, inDouble bool, whole string) error {
	if !inSingle && !inDouble {
		if *depth == 0 {
			return fmt.Errorf("rules: unmatched ')' in %q", whole)
		}
		(*depth)--
	}
	return nil
}

func shouldSplit(r rune, depth int, inSingle, inDouble bool, sep rune) bool {
	return r == sep && depth == 0 && !inSingle && !inDouble
}
