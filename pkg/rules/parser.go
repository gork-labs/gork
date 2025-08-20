package rules

import (
	"fmt"
	"strings"
)

// node represents a node in the expression abstract syntax tree.
type node interface{}

// nodeCall represents a function call in the expression.
type nodeCall struct {
	name string
	args []argToken
}

// nodeUnary represents a unary operation (e.g., NOT).
type nodeUnary struct {
	op tokKind // tkNot
	x  node
}

// nodeBinary represents a binary operation (e.g., AND, OR).
type nodeBinary struct {
	op tokKind // tkAnd / tkOr
	l  node
	r  node
}

// nodeBool represents a boolean literal.
type nodeBool struct{ v bool }

// parser handles parsing of rule expressions using recursive descent.
type parser struct {
	toks []token
	pos  int
}

// cur returns the current token.
func (p *parser) cur() token { return p.toks[p.pos] }

// eat consumes a token of the specified kind and returns true if successful.
func (p *parser) eat(k tokKind) bool {
	if p.cur().kind == k {
		p.pos++
		return true
	}
	return false
}

// expect consumes a token of the specified kind or returns an error.
func (p *parser) expect(k tokKind) error {
	if !p.eat(k) {
		return fmt.Errorf("expected token %v", k)
	}
	return nil
}

// must consumes a token of the specified kind or panics.
// Use this when the token is guaranteed to be present due to program invariants.
func (p *parser) must(k tokKind) {
	if !p.eat(k) {
		panic(fmt.Sprintf("parser invariant violated: expected token %v at position %d, got %v", k, p.pos, p.cur().kind))
	}
}

// parseExpr parses a complete expression.
func (p *parser) parseExpr() (node, error) { return p.parseOr() }

// parseOr parses OR expressions with left associativity.
func (p *parser) parseOr() (node, error) {
	n, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == tkOr {
		p.pos++
		rhs, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		n = &nodeBinary{op: tkOr, l: n, r: rhs}
	}
	return n, nil
}

// parseAnd parses AND expressions with left associativity.
func (p *parser) parseAnd() (node, error) {
	n, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == tkAnd {
		p.pos++
		rhs, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		n = &nodeBinary{op: tkAnd, l: n, r: rhs}
	}
	return n, nil
}

// parseNot parses NOT expressions.
func (p *parser) parseNot() (node, error) {
	if p.cur().kind == tkNot {
		p.pos++
		x, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &nodeUnary{op: tkNot, x: x}, nil
	}
	return p.parsePrimary()
}

// parsePrimary parses primary expressions (literals, function calls, parenthesized expressions).
//
//nolint:exhaustive // the primary parser handles only the tokens used as primaries
func (p *parser) parsePrimary() (node, error) {
	switch p.cur().kind {
	case tkLPar:
		p.pos++
		n, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(tkRPar); err != nil {
			return nil, err
		}
		return n, nil
	case tkBool:
		val := p.cur().text == "true"
		p.pos++
		return &nodeBool{v: val}, nil
	case tkIdent:
		name := p.cur().text
		p.pos++
		return p.parseFunctionCall(name)
	default:
		return nil, fmt.Errorf("unexpected token: %v", p.cur())
	}
}

// parseFunctionCall parses ident '(' [args] ')' and returns a nodeCall.
func (p *parser) parseFunctionCall(name string) (node, error) {
	if err := p.expect(tkLPar); err != nil {
		return nil, err
	}
	argsStr, err := p.collectArgsString()
	if err != nil {
		return nil, err
	}
	var args []argToken
	if argsStr != "" {
		atoks, err := parseArgs(argsStr)
		if err != nil {
			return nil, err
		}
		args = atoks
	}
	// Consume the closing ) - this should always succeed after collectArgsString
	p.must(tkRPar)
	return &nodeCall{name: name, args: args}, nil
}

// collectArgsString consumes tokens until the matching closing ')' and reconstructs the raw args string.
func (p *parser) collectArgsString() (string, error) {
	depth := 1
	var raw strings.Builder
	for depth > 0 {
		t := p.cur()
		if t.kind == tkEOF {
			return "", fmt.Errorf("unterminated argument list")
		}
		if t.kind == tkLPar {
			depth++
			raw.WriteByte('(')
			p.pos++
			continue
		}
		if t.kind == tkRPar {
			depth--
			if depth == 0 {
				break
			}
			raw.WriteByte(')')
			p.pos++
			continue
		}
		if raw.Len() > 0 {
			raw.WriteByte(' ')
		}
		//nolint:exhaustive // only token kinds that can appear inside args need reconstruction
		switch t.kind {
		case tkString:
			raw.WriteByte('"')
			raw.WriteString(t.text)
			raw.WriteByte('"')
		case tkComma:
			raw.WriteByte(',')
		default:
			raw.WriteString(t.text)
		}
		p.pos++
	}
	return strings.TrimSpace(raw.String()), nil
}
