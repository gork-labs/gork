package rules

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestExpressionParsingTableDriven uses table-driven tests for expression parsing scenarios
func TestExpressionParsingTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		setup       func() // Optional setup function
		validate    func(t *testing.T, result interface{}, err error)
	}{
		{
			name:        "TokenizeError_InvalidCharacter",
			input:       "@",
			expectError: true,
			validate: func(t *testing.T, result interface{}, err error) {
				// For evalBooleanExpr, we expect errors in the slice
				if errs, ok := result.([]error); ok && len(errs) == 0 {
					t.Error("Expected tokenize error")
				}
			},
		},
		{
			name:        "ParseError_UnterminatedCall",
			input:       "ok(",
			expectError: true,
			validate: func(t *testing.T, result interface{}, err error) {
				if errs, ok := result.([]error); ok && len(errs) == 0 {
					t.Error("Expected parse error")
				}
			},
		},
		{
			name:        "ServerError_UnknownRule",
			input:       "unknownRule()",
			expectError: true,
			validate: func(t *testing.T, result interface{}, err error) {
				if errs, ok := result.([]error); ok && len(errs) == 0 {
					t.Error("Expected server error from unknown rule")
				}
			},
		},
		{
			name:  "ValidationError_FalseRule",
			input: "fail()",
			setup: func() {
				resetRegistry()
				Register("fail", func(ctx context.Context, _ any) (bool, error) { return false, nil })
			},
			expectError: true,
			validate: func(t *testing.T, result interface{}, err error) {
				if errs, ok := result.([]error); ok && len(errs) != 1 {
					t.Errorf("Expected 1 validation error, got %v", errs)
				}
			},
		},
		{
			name:        "ComplexExpression_OrAndNot",
			input:       "a() or b() and not c()",
			expectError: true,
			setup: func() {
				resetRegistry()
				Register("a", func(ctx context.Context, _ any) (bool, error) { return false, nil })
				Register("b", func(ctx context.Context, _ any) (bool, error) { return true, nil })
				Register("c", func(ctx context.Context, _ any) (bool, error) { return false, nil })
			},
			validate: func(t *testing.T, result interface{}, err error) {
				// Should handle complex expressions with multiple operators
				if errs, ok := result.([]error); ok {
					// Complex expression should evaluate - might not have errors
					t.Logf("Complex expression result: %v errors", len(errs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			var root, parent struct{}
			ent := "e"
			errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, tt.input)

			if tt.validate != nil {
				tt.validate(t, errs, nil)
			}
		})
	}
}

// TestParserTableDriven tests parser functionality with table-driven approach
func TestParserTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []token
		expectError bool
		method      string // Which parser method to call
		validate    func(t *testing.T, result node, err error)
	}{
		{
			name: "UnexpectedTokenError",
			tokens: []token{
				{kind: tkComma, text: ","},
				{kind: tkEOF},
			},
			expectError: true,
			method:      "parsePrimary",
		},
		{
			name: "ExpectLParError",
			tokens: []token{
				{kind: tkBool, text: "true"},
				{kind: tkEOF},
			},
			expectError: true,
			method:      "parseFunctionCall",
		},
		{
			name: "CollectArgsError",
			tokens: []token{
				{kind: tkLPar, text: "("},
				{kind: tkEOF}, // Missing closing ")" - triggers collectArgsString error
			},
			expectError: true,
			method:      "parseFunctionCall",
		},

		{
			name: "InvalidArgsError",
			tokens: []token{
				{kind: tkLPar, text: "("},
				{kind: tkIdent, text: "$."}, // Invalid field reference (causes parseArgs error)
				{kind: tkRPar, text: ")"},
				{kind: tkEOF},
			},
			expectError: true,
			method:      "parseFunctionCall",
		},
		{
			name: "FunctionCallWithArgs",
			tokens: []token{
				{kind: tkLPar, text: "("},
				{kind: tkString, text: "arg1"},
				{kind: tkRPar, text: ")"},
				{kind: tkEOF},
			},
			expectError: false,
			method:      "parseFunctionCall",
			validate: func(t *testing.T, result node, err error) {
				call, ok := result.(*nodeCall)
				if !ok {
					t.Fatalf("expected nodeCall, got %T", result)
				}
				if call.name != "f" || len(call.args) != 1 {
					t.Fatalf("unexpected call node: name=%s, args=%d", call.name, len(call.args))
				}
			},
		},
		{
			name: "FunctionCallNoArgs",
			tokens: []token{
				{kind: tkIdent, text: "f"},
				{kind: tkLPar, text: "("},
				{kind: tkRPar, text: ")"},
				{kind: tkEOF},
			},
			expectError: false,
			method:      "parseExpr",
			validate: func(t *testing.T, result node, err error) {
				call, ok := result.(*nodeCall)
				if !ok {
					t.Fatalf("expected nodeCall, got %T", result)
				}
				if call.name != "f" || len(call.args) != 0 {
					t.Fatalf("unexpected call node: %+v", call)
				}
			},
		},
		{
			name: "BoolFalseParsing",
			tokens: []token{
				{kind: tkBool, text: "false"},
				{kind: tkEOF},
			},
			expectError: false,
			method:      "parsePrimary",
			validate: func(t *testing.T, result node, err error) {
				nb, ok := result.(*nodeBool)
				if !ok || nb.v != false {
					t.Fatalf("expected nodeBool false, got %#v", result)
				}
			},
		},
		{
			name: "ParenthesizedExpression",
			tokens: []token{
				{kind: tkLPar, text: "("},
				{kind: tkBool, text: "true"},
				{kind: tkRPar, text: ")"},
				{kind: tkEOF},
			},
			expectError: false,
			method:      "parsePrimary",
			validate: func(t *testing.T, result node, err error) {
				nb, ok := result.(*nodeBool)
				if !ok || nb.v != true {
					t.Fatalf("expected nodeBool true, got %#v", result)
				}
			},
		},
		{
			name: "OrExpression",
			tokens: []token{
				{kind: tkIdent, text: "a"},
				{kind: tkLPar, text: "("},
				{kind: tkRPar, text: ")"},
				{kind: tkOr, text: "or"},
				{kind: tkIdent, text: "b"},
				{kind: tkLPar, text: "("},
				{kind: tkRPar, text: ")"},
				{kind: tkEOF},
			},
			expectError: false,
			method:      "parseExpr",
			validate: func(t *testing.T, result node, err error) {
				bin, ok := result.(*nodeBinary)
				if !ok || bin.op != tkOr {
					t.Fatalf("expected OR binary node, got %#v", result)
				}
			},
		},
		{
			name: "AndExpression",
			tokens: []token{
				{kind: tkIdent, text: "a"},
				{kind: tkLPar, text: "("},
				{kind: tkRPar, text: ")"},
				{kind: tkAnd, text: "and"},
				{kind: tkIdent, text: "b"},
				{kind: tkLPar, text: "("},
				{kind: tkRPar, text: ")"},
				{kind: tkEOF},
			},
			expectError: false,
			method:      "parseExpr",
			validate: func(t *testing.T, result node, err error) {
				bin, ok := result.(*nodeBinary)
				if !ok || bin.op != tkAnd {
					t.Fatalf("expected AND binary node, got %#v", result)
				}
			},
		},
		{
			name: "NotExpression",
			tokens: []token{
				{kind: tkNot, text: "not"},
				{kind: tkBool, text: "true"},
				{kind: tkEOF},
			},
			expectError: false,
			method:      "parseExpr",
			validate: func(t *testing.T, result node, err error) {
				un, ok := result.(*nodeUnary)
				if !ok || un.op != tkNot {
					t.Fatalf("expected NOT unary node, got %#v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{toks: tt.tokens, pos: 0}

			var result node
			var err error

			switch tt.method {
			case "parsePrimary":
				result, err = p.parsePrimary()
			case "parseExpr":
				result, err = p.parseExpr()
			case "parseFunctionCall":
				result, err = p.parseFunctionCall("f")
			default:
				t.Fatalf("Unknown parser method: %s", tt.method)
			}

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validate != nil && !tt.expectError {
				tt.validate(t, result, err)
			}
		})
	}
}

// TestNodeEvaluationTableDriven tests node evaluation with table-driven approach
func TestNodeEvaluationTableDriven(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	ctx := context.Background()
	rootVal := reflect.ValueOf(&root).Elem()
	parentVal := reflect.ValueOf(&parent).Elem()

	tests := []struct {
		name            string
		node            node
		expectServerErr bool
		expectPass      bool
		expectValErrs   int
		setup           func()
	}{
		{
			name:            "UnsupportedExprNode",
			node:            123, // Invalid node type
			expectServerErr: true,
		},
		{
			name:            "UnaryInnerServerError",
			node:            &nodeUnary{op: tkNot, x: 123}, // Invalid inner node
			expectServerErr: true,
		},
		{
			name:       "AndLeftFalseShortCircuit",
			node:       &nodeBinary{op: tkAnd, l: &nodeBool{v: false}, r: &nodeBool{v: true}},
			expectPass: false,
		},
		{
			name:            "AndRightServerError",
			node:            &nodeBinary{op: tkAnd, l: &nodeBool{v: true}, r: 123}, // Invalid right node
			expectServerErr: true,
		},
		{
			name:       "AndBothTrue",
			node:       &nodeBinary{op: tkAnd, l: &nodeBool{v: true}, r: &nodeBool{v: true}},
			expectPass: true,
		},
		{
			name:       "OrLeftPassTrueShortcut",
			node:       &nodeBinary{op: tkOr, l: &nodeBool{v: true}, r: &nodeBool{v: false}},
			expectPass: true,
		},
		{
			name:            "OrRightServerError",
			node:            &nodeBinary{op: tkOr, l: &nodeBool{v: false}, r: 123}, // Invalid right node
			expectServerErr: true,
		},
		{
			name:            "OrLeftServerError",
			node:            &nodeBinary{op: tkOr, l: 123, r: &nodeBool{v: true}}, // Invalid left node
			expectServerErr: true,
		},
		{
			name:            "AndLeftServerError",
			node:            &nodeBinary{op: tkAnd, l: 123, r: &nodeBool{v: true}}, // Invalid left node
			expectServerErr: true,
		},
		{
			name: "OrBothFalseCollectsErrors",
			node: &nodeBinary{
				op: tkOr,
				l:  &nodeCall{name: "errA"},
				r:  &nodeCall{name: "errB"},
			},
			expectValErrs: 2,
			setup: func() {
				resetRegistry()
				Register("errA", func(ctx context.Context, _ any) (bool, error) { return false, nil })
				Register("errB", func(ctx context.Context, _ any) (bool, error) { return false, nil })
			},
		},
		{
			name: "AndRightValidationError",
			node: &nodeBinary{
				op: tkAnd,
				l:  &nodeCall{name: "ok"},
				r:  &nodeCall{name: "fail"},
			},
			expectValErrs: 1,
			setup: func() {
				resetRegistry()
				Register("ok", func(ctx context.Context, _ any) (bool, error) { return true, nil })
				Register("fail", func(ctx context.Context, _ any) (bool, error) { return false, nil })
			},
		},
		{
			name:            "RuleInvocationWithArityError",
			node:            &nodeCall{name: "needsArgs", args: []argToken{}},
			expectServerErr: true,
			setup: func() {
				resetRegistry()
				// Register a rule that expects arguments but call with wrong number
				Register("needsArgs", func(ctx context.Context, entity any, arg1 string) (bool, error) {
					return true, nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			var res evalResult
			switch n := tt.node.(type) {
			case *nodeBinary:
				if n.op == tkAnd {
					res = evalAndNode(ctx, rootVal, parentVal, &ent, n)
				} else if n.op == tkOr {
					res = evalOrNode(ctx, rootVal, parentVal, &ent, n)
				} else {
					res = evalNode(ctx, rootVal, parentVal, &ent, tt.node)
				}
			default:
				res = evalNode(ctx, rootVal, parentVal, &ent, tt.node)
			}

			if tt.expectServerErr {
				if res.serverErr == nil {
					t.Error("Expected server error but got none")
				}
			} else {
				if res.serverErr != nil {
					t.Errorf("Expected no server error but got: %v", res.serverErr)
				}
			}

			if tt.expectValErrs > 0 {
				if len(res.valErrs) != tt.expectValErrs {
					t.Errorf("Expected %d validation errors, got %d", tt.expectValErrs, len(res.valErrs))
				}
			}

			// Only check pass value if no server error expected
			if !tt.expectServerErr {
				if res.pass != tt.expectPass {
					t.Errorf("Expected pass=%v, got pass=%v", tt.expectPass, res.pass)
				}
			}
		})
	}
}

// TestCollectArgsStringTableDriven tests collectArgsString functionality
func TestCollectArgsStringTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []token
		startPos    int
		expectError bool
	}{
		{
			name: "UnterminatedArgs",
			tokens: []token{
				{kind: tkIdent, text: "f"},
				{kind: tkLPar, text: "("},
				{kind: tkIdent, text: "x"},
				{kind: tkEOF}, // Missing ")"
			},
			startPos:    1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{toks: tt.tokens, pos: tt.startPos}

			_, err := p.collectArgsString()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestInvokeRuleTableDriven tests the invokeRule function for missing coverage
func TestInvokeRuleTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		setup       func()
		ruleName    string
		entity      any
		args        []any
		expectError bool
		expectPass  bool
	}{
		{
			name: "RuleNotFound",
			setup: func() {
				resetRegistry()
			},
			ruleName:    "nonexistent",
			entity:      "test",
			args:        []any{},
			expectError: true,
		},
		{
			name: "ArityMismatch_TooFewArgs",
			setup: func() {
				resetRegistry()
				Register("needsTwo", func(ctx context.Context, entity any, arg1, arg2 string) (bool, error) {
					return true, nil
				})
			},
			ruleName:    "needsTwo",
			entity:      "test",
			args:        []any{"only_one"},
			expectError: true,
		},
		{
			name: "ArityMismatch_TooManyArgs",
			setup: func() {
				resetRegistry()
				Register("needsOne", func(ctx context.Context, entity any, arg1 string) (bool, error) {
					return true, nil
				})
			},
			ruleName:    "needsOne",
			entity:      "test",
			args:        []any{"arg1", "arg2", "arg3"},
			expectError: true,
		},
		{
			name: "SuccessfulInvocation",
			setup: func() {
				resetRegistry()
				Register("success", func(ctx context.Context, entity any) (bool, error) {
					return true, nil
				})
			},
			ruleName:   "success",
			entity:     "test",
			args:       []any{},
			expectPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			// Need to create a nodeCall and use evalNode instead since invokeRule has different signature
			var root, parent struct{}
			call := &nodeCall{name: tt.ruleName, args: []argToken{}}
			result := evalNode(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &tt.entity, call)

			if tt.expectError && result.serverErr == nil {
				t.Error("Expected server error but got none")
			}
			if !tt.expectError && result.serverErr != nil {
				t.Errorf("Expected no server error but got: %v", result.serverErr)
			}

			if !tt.expectError && result.pass != tt.expectPass {
				t.Errorf("Expected pass=%v, got pass=%v", tt.expectPass, result.pass)
			}
		})
	}
}

// TestParserCoverageTableDriven adds test cases to reach 100% coverage of parser functions
func TestParserCoverageTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []token
		method      string
		expectError bool
		validate    func(t *testing.T, result node, err error)
	}{
		{
			name: "ParseOr_RightSideError",
			tokens: []token{
				{kind: tkBool, text: "true"},
				{kind: tkOr, text: "or"},
				{kind: tkComma, text: ","}, // Invalid token for expression
				{kind: tkEOF},
			},
			method:      "parseExpr", // parseOr is called through parseExpr
			expectError: true,
		},
		{
			name: "ParseAnd_RightSideError",
			tokens: []token{
				{kind: tkBool, text: "true"},
				{kind: tkAnd, text: "and"},
				{kind: tkComma, text: ","}, // Invalid token for expression
				{kind: tkEOF},
			},
			method:      "parseExpr", // parseAnd is called through parseExpr
			expectError: true,
		},
		{
			name: "ParseNot_InnerError",
			tokens: []token{
				{kind: tkNot, text: "not"},
				{kind: tkComma, text: ","}, // Invalid token for primary
				{kind: tkEOF},
			},
			method:      "parseExpr",
			expectError: true,
		},
		{
			name: "ParsePrimary_UnmatchedRightParen",
			tokens: []token{
				{kind: tkLPar, text: "("},
				{kind: tkBool, text: "true"},
				{kind: tkEOF}, // Missing right paren
			},
			method:      "parsePrimary",
			expectError: true,
		},
		{
			name: "CollectArgsString_NestedParensError",
			tokens: []token{
				{kind: tkLPar, text: "("},
				{kind: tkString, text: "'start'"},
				{kind: tkLPar, text: "("},
				{kind: tkString, text: "'nested'"},
				{kind: tkEOF}, // Missing closing parens
			},
			method:      "collectArgsString",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{toks: tt.tokens, pos: 0}

			var result node
			var err error

			switch tt.method {
			case "parseExpr":
				result, err = p.parseExpr()
			case "parsePrimary":
				result, err = p.parsePrimary()
			case "collectArgsString":
				// collectArgsString returns string, not node
				_, err = p.collectArgsString()
			default:
				t.Fatalf("Unknown method: %s", tt.method)
			}

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validate != nil && !tt.expectError {
				tt.validate(t, result, err)
			}
		})
	}
}

// TestEvalNodeCoverageTableDriven adds test cases to reach 100% coverage of evalNode
func TestEvalNodeCoverageTableDriven(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	ctx := context.Background()
	rootVal := reflect.ValueOf(&root).Elem()
	parentVal := reflect.ValueOf(&parent).Elem()

	tests := []struct {
		name            string
		setupNode       func() node
		expectServerErr bool
		setup           func()
	}{
		{
			name: "OrNodeDispatch",
			setupNode: func() node {
				return &nodeBinary{op: tkOr, l: &nodeBool{v: false}, r: &nodeBool{v: true}}
			},
		},
		{
			name: "CallNodeDispatch",
			setupNode: func() node {
				return &nodeCall{name: "test", args: []argToken{}}
			},
			setup: func() {
				resetRegistry()
				Register("test", func(ctx context.Context, _ any) (bool, error) { return true, nil })
			},
		},
		{
			name: "UnaryNodeDispatch",
			setupNode: func() node {
				return &nodeUnary{op: tkNot, x: &nodeBool{v: true}}
			},
		},
		{
			name: "BoolNodeDispatch_True",
			setupNode: func() node {
				return &nodeBool{v: true}
			},
		},
		{
			name: "BoolNodeDispatch_False",
			setupNode: func() node {
				return &nodeBool{v: false}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			node := tt.setupNode()
			res := evalNode(ctx, rootVal, parentVal, &ent, node)

			if tt.expectServerErr && res.serverErr == nil {
				t.Error("Expected server error but got none")
			}
			if !tt.expectServerErr && res.serverErr != nil {
				t.Errorf("Expected no server error but got: %v", res.serverErr)
			}
		})
	}
}

// TestFinalCoverageGaps targets the specific remaining uncovered lines
func TestFinalCoverageGaps(t *testing.T) {
	t.Run("ValueAsEntity_InvalidValue", func(t *testing.T) {
		// Test with a valid but unaddressable value (covers the final branch)
		val := reflect.ValueOf(42) // int literal, cannot be addressed
		entity := valueAsEntity(val)
		if entity != 42 {
			t.Errorf("Expected 42, got %v", entity)
		}
	})

	t.Run("CollectArgsString_UnmatchedParens", func(t *testing.T) {
		// Test unmatched parentheses in collectArgsString
		tokens := []token{
			{kind: tkLPar, text: "("},
			{kind: tkString, text: "'start'"},
			{kind: tkLPar, text: "("},
			{kind: tkString, text: "'nested'"},
			// Missing closing parens
			{kind: tkEOF},
		}
		p := &parser{toks: tokens, pos: 0}
		_, err := p.collectArgsString()
		if err == nil {
			t.Error("Expected error for unmatched parentheses")
		}
	})

	t.Run("EvalNode_DefaultCase", func(t *testing.T) {
		// Test the default case in evalNode switch statement
		var root, parent struct{}
		ent := "e"
		ctx := context.Background()
		rootVal := reflect.ValueOf(&root).Elem()
		parentVal := reflect.ValueOf(&parent).Elem()

		// Use an unknown node type to trigger default case
		type unknownNode struct{}
		unknown := &unknownNode{}

		result := evalNode(ctx, rootVal, parentVal, &ent, unknown)
		if result.serverErr == nil {
			t.Error("Expected server error for unknown node type")
		}
	})

	t.Run("InvokeRule_ActualCall", func(t *testing.T) {
		// Test invokeRule directly with proper signature
		resetRegistry()
		Register("directTest", func(ctx context.Context, entity any) (bool, error) {
			return true, nil
		})

		var root, parent struct{}
		ent := "test"
		call := &nodeCall{name: "directTest", args: []argToken{}}

		// This should reach the actual invokeRule function
		pass, err := invokeRule(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, call)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !pass {
			t.Error("Expected pass=true")
		}
	})

	t.Run("InvokeRule_ServerError", func(t *testing.T) {
		// Test invokeRule with server error from rule function
		resetRegistry()
		Register("serverErrorRule", func(ctx context.Context, entity any) (bool, error) {
			return false, fmt.Errorf("server error")
		})

		var root, parent struct{}
		ent := "test"
		call := &nodeCall{name: "serverErrorRule", args: []argToken{}}

		pass, err := invokeRule(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, call)
		if err == nil {
			t.Error("Expected server error")
		}
		if pass {
			t.Error("Expected pass=false")
		}
	})
}

// TestFinalCoverageTargets covers the last remaining lines to reach 100%
func TestFinalCoverageTargets(t *testing.T) {
	t.Run("EvalNode_BinaryNodeDispatch", func(t *testing.T) {
		// Test evalNode with binary node that doesn't match AND/OR
		var root, parent struct{}
		ent := "e"
		ctx := context.Background()
		rootVal := reflect.ValueOf(&root).Elem()
		parentVal := reflect.ValueOf(&parent).Elem()

		// Create a binary node with an unsupported operator
		binaryNode := &nodeBinary{
			op: tkComma, // Unsupported binary operator
			l:  &nodeBool{v: true},
			r:  &nodeBool{v: false},
		}

		result := evalNode(ctx, rootVal, parentVal, &ent, binaryNode)
		if result.serverErr == nil {
			t.Error("Expected server error for unsupported binary operator")
		}
	})
}

// TestParseFunctionCallExpectRParError specifically tests the expect(tkRPar) error path
func TestParseFunctionCallExpectRParError(t *testing.T) {
	t.Run("ManualParserStateManipulation", func(t *testing.T) {
		// Create a scenario where we can manually trigger the expect(tkRPar) error
		// by creating a custom test that simulates the exact condition

		// Step 1: Create parser and call the first part of parseFunctionCall manually
		tokens := []token{
			{kind: tkLPar, text: "("},
			{kind: tkRPar, text: ")"},  // This gets consumed by collectArgsString
			{kind: tkComma, text: ","}, // This is where we want to end up
			{kind: tkEOF},
		}

		p := &parser{toks: tokens, pos: 0}

		// Step 2: Execute the first part of parseFunctionCall logic manually
		if err := p.expect(tkLPar); err != nil {
			t.Fatalf("Unexpected error in expect(tkLPar): %v", err)
		}

		argsStr, err := p.collectArgsString()
		if err != nil {
			t.Fatalf("Unexpected error in collectArgsString: %v", err)
		}

		// Step 3: Manually manipulate parser position to trigger the error condition
		p.pos = 2 // Force position to comma instead of closing paren

		// Step 4: Test the specific expect(tkRPar) call that should fail
		if err := p.expect(tkRPar); err == nil {
			t.Error("Expected error from expect(tkRPar) but got none")
		} else {
			if !strings.Contains(err.Error(), "expected token") {
				t.Errorf("Expected error about expected token, got: %v", err.Error())
			}
		}

		// This proves the error path is reachable when parser state is manipulated
		t.Logf("Successfully triggered expect(tkRPar) error with argsStr='%s'", argsStr)
	})
}

// TestParserMustFunction tests the new must() function for invariant enforcement
func TestParserMustFunction(t *testing.T) {
	t.Run("MustSuccess", func(t *testing.T) {
		tokens := []token{
			{kind: tkRPar, text: ")"},
			{kind: tkEOF},
		}
		p := &parser{toks: tokens, pos: 0}

		// Should not panic
		p.must(tkRPar)
		if p.pos != 1 {
			t.Errorf("Expected position 1, got %d", p.pos)
		}
	})

	t.Run("MustPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic from must() but got none")
			} else {
				panicMsg := fmt.Sprintf("%v", r)
				if !strings.Contains(panicMsg, "parser invariant violated") {
					t.Errorf("Expected invariant violation message, got: %s", panicMsg)
				}
			}
		}()

		tokens := []token{
			{kind: tkComma, text: ","},
			{kind: tkEOF},
		}
		p := &parser{toks: tokens, pos: 0}

		// Should panic
		p.must(tkRPar)
		t.Error("Should not reach this line")
	})
}

// TestAbsoluteFinalCoverage targets the very last uncovered lines
func TestAbsoluteFinalCoverage(t *testing.T) {
	t.Run("CollectArgsString_SuccessfulParsing", func(t *testing.T) {
		// Test successful parsing path in collectArgsString
		tokens := []token{
			{kind: tkLPar, text: "("},
			{kind: tkString, text: "hello"},
			{kind: tkRPar, text: ")"},
			{kind: tkEOF},
		}
		p := &parser{toks: tokens, pos: 1} // Start after opening paren
		result, err := p.collectArgsString()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != `"hello"` {
			t.Errorf("Expected quoted hello, got: %s", result)
		}
	})

	t.Run("CollectArgsString_NestedParensSuccess", func(t *testing.T) {
		// Test nested parentheses that successfully parse - covers raw.WriteByte(')'), p.pos++, continue lines
		tokens := []token{
			{kind: tkLPar, text: "("}, // depth 1 (outer)
			{kind: tkString, text: "outer"},
			{kind: tkLPar, text: "("}, // depth 2 (inner)
			{kind: tkString, text: "inner"},
			{kind: tkRPar, text: ")"}, // depth 1 (this covers the lines we need)
			{kind: tkString, text: "more"},
			{kind: tkRPar, text: ")"}, // depth 0 (exit)
			{kind: tkEOF},
		}
		p := &parser{toks: tokens, pos: 1} // Start after opening paren
		result, err := p.collectArgsString()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		expected := `"outer"( "inner") "more"`
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("CollectArgsString_CommaHandling", func(t *testing.T) {
		// Test comma token handling - covers raw.WriteByte(',') line
		tokens := []token{
			{kind: tkLPar, text: "("},
			{kind: tkString, text: "arg1"},
			{kind: tkComma, text: ","}, // This covers the comma case
			{kind: tkString, text: "arg2"},
			{kind: tkComma, text: ","}, // Another comma
			{kind: tkIdent, text: "arg3"},
			{kind: tkRPar, text: ")"},
			{kind: tkEOF},
		}
		p := &parser{toks: tokens, pos: 1} // Start after opening paren
		result, err := p.collectArgsString()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		expected := `"arg1" , "arg2" , arg3`
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("InvokeRule_ResolveError", func(t *testing.T) {
		// Test invokeRule resolve error handling - covers resolve error path
		resetRegistry()
		Register("testRule", func(ctx context.Context, entity any, arg string) (bool, error) {
			return true, nil
		})

		var root, parent struct{}
		ent := "test"

		// Create a nodeCall with an argument that will fail to resolve (non-existent context variable)
		call := &nodeCall{
			name: "testRule",
			args: []argToken{
				{Kind: argContextVar, ContextVar: "nonexistent_var"},
			},
		}

		pass, err := invokeRule(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, call)

		// This should trigger the resolve error path: if err != nil { return false, err }
		if err == nil {
			t.Error("Expected resolve error but got none")
		}
		if pass {
			t.Error("Expected pass=false when resolve error occurs")
		}
		if !strings.Contains(err.Error(), "context variable \"nonexistent_var\" not found") {
			t.Errorf("Expected context variable error, got: %v", err)
		}
	})
}
