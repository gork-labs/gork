package rules

import (
	"context"
	"fmt"
	"reflect"
)

// RuleValidationError represents a validation error from rule evaluation.
// It implements the ValidationError interface expected by the API layer.
type RuleValidationError struct {
	Rule    string
	Message string
}

func (e *RuleValidationError) Error() string {
	return e.Message
}

// ExpressionEvaluator handles parsing and evaluation of boolean rule expressions.
type ExpressionEvaluator struct {
	// evalNodeFunc can be overridden for testing purposes
	evalNodeFunc func(ctx context.Context, root, parent reflect.Value, entity any, n node) evalResult
}

// NewExpressionEvaluator creates a new expression evaluator with default behavior.
func NewExpressionEvaluator() *ExpressionEvaluator {
	return &ExpressionEvaluator{
		evalNodeFunc: evalNode,
	}
}

// EvalBooleanExpr parses and evaluates a boolean rule expression.
// Returns validation errors (if any) when the expression evaluates to false.
func (e *ExpressionEvaluator) EvalBooleanExpr(ctx context.Context, root, parent reflect.Value, entity any, input string) []error {
	tokens, err := tokenize(input)
	if err != nil {
		return []error{fmt.Errorf("rules: expr tokenize: %w", err)}
	}
	p := &parser{toks: tokens}
	expr, err := p.parseExpr()
	if err != nil {
		return []error{fmt.Errorf("rules: expr parse: %w", err)}
	}
	res := e.evalNodeFunc(ctx, root, parent, entity, expr)
	if res.serverErr != nil {
		return []error{res.serverErr}
	}
	if !res.pass {
		return res.valErrs
	}
	return nil
}

// evalBooleanExpr is a convenience function that uses the default evaluator.
// Returns validation errors (if any) when the expression evaluates to false.
func evalBooleanExpr(ctx context.Context, root, parent reflect.Value, entity any, input string) []error {
	evaluator := NewExpressionEvaluator()
	return evaluator.EvalBooleanExpr(ctx, root, parent, entity, input)
}

// evalResult represents the result of evaluating an expression node.
type evalResult struct {
	pass      bool
	valErrs   []error
	serverErr error
}

// evalNode evaluates an expression node and returns the result.
func evalNode(ctx context.Context, root, parent reflect.Value, entity any, n node) evalResult {
	switch x := n.(type) {
	case *nodeBool:
		return evalResult{pass: x.v}
	case *nodeCall:
		pass, sysErr := invokeRule(ctx, root, parent, entity, x)
		if sysErr != nil {
			// System error (network, DB failure, etc.)
			return evalResult{pass: false, serverErr: sysErr}
		}
		if pass {
			// Validation passed
			return evalResult{pass: true}
		}
		// Validation failed (business logic) - generate validation error message
		validationErr := &RuleValidationError{
			Rule:    x.name,
			Message: fmt.Sprintf("rule '%s' validation failed", x.name),
		}
		return evalResult{pass: false, valErrs: []error{validationErr}}
	case *nodeUnary:
		inner := evalNode(ctx, root, parent, entity, x.x)
		if inner.serverErr != nil {
			return inner
		}
		return evalResult{pass: !inner.pass}
	case *nodeBinary:
		if x.op == tkAnd {
			return evalAndNode(ctx, root, parent, entity, x)
		}
		if x.op == tkOr {
			return evalOrNode(ctx, root, parent, entity, x)
		}
		return evalResult{serverErr: fmt.Errorf("unsupported binary op")}
	default:
		return evalResult{serverErr: fmt.Errorf("unsupported expr node")}
	}
}

// evalAndNode evaluates a binary AND expression.
func evalAndNode(ctx context.Context, root, parent reflect.Value, entity any, b *nodeBinary) evalResult {
	left := evalNode(ctx, root, parent, entity, b.l)
	if left.serverErr != nil || !left.pass {
		return left
	}
	right := evalNode(ctx, root, parent, entity, b.r)
	if right.serverErr != nil || !right.pass {
		return right
	}
	return evalResult{pass: true}
}

// evalOrNode evaluates a binary OR expression.
func evalOrNode(ctx context.Context, root, parent reflect.Value, entity any, b *nodeBinary) evalResult {
	left := evalNode(ctx, root, parent, entity, b.l)
	if left.serverErr != nil || left.pass {
		if left.serverErr != nil {
			return left
		}
		return evalResult{pass: true}
	}
	right := evalNode(ctx, root, parent, entity, b.r)
	if right.serverErr != nil {
		return right
	}
	if right.pass {
		return evalResult{pass: true}
	}
	return evalResult{pass: false, valErrs: append(left.valErrs, right.valErrs...)}
}

// invokeRule invokes a registered rule function with the given arguments.
// Returns (pass, serverError) where:
//
//	pass=true: validation passed
//	pass=false, serverError=nil: validation failed (business logic)
//	pass=false, serverError!=nil: system error
func invokeRule(ctx context.Context, root, parent reflect.Value, entity any, c *nodeCall) (bool, error) {
	desc, ok := getRule(c.name)
	if !ok {
		return false, fmt.Errorf("rules: rule %q not registered", c.name)
	}
	args, err := resolve(ctx, root.Addr().Interface(), parent.Addr().Interface(), c.args)
	if err != nil {
		return false, err
	}
	if !desc.isVariadic {
		expected := desc.numIn - 2
		if len(args) != expected {
			return false, fmt.Errorf("rules: rule %q expects %d args, got %d", c.name, expected, len(args))
		}
	}
	callArgs := make([]reflect.Value, 0, 2+len(args))
	callArgs = append(callArgs, reflect.ValueOf(ctx), reflect.ValueOf(entity))
	for _, a := range args {
		callArgs = append(callArgs, reflect.ValueOf(a))
	}
	res := desc.fn.Call(callArgs)

	// Extract (bool, error) return values
	pass := res[0].Bool()
	var sysErr error
	if !res[1].IsNil() {
		sysErr = res[1].Interface().(error)
	}

	return pass, sysErr
}
