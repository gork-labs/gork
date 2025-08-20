package rules

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type engineReq struct {
	Path struct {
		UserID string `rule:"must_equal('u1')"`
		Item   struct {
			Owner string `rule:"owned_by($.Path.UserID)"`
		}
	}
}

func TestEngine_Apply(t *testing.T) {
	resetRegistry()
	// Register a simple rule that asserts entity string equals provided string
	Register("must_equal", func(ctx context.Context, entity any, args ...any) (bool, error) {
		s, _ := entity.(*string)
		if s == nil {
			return false, errors.New("not a *string") // System error
		}
		if len(args) != 1 {
			return false, errors.New("arg required") // System error
		}
		want, _ := args[0].(string)
		if *s != want {
			return false, nil // Validation failed (business logic)
		}
		return true, nil // Validation passed
	})
	Register("owned_by", func(ctx context.Context, entity any, args ...any) (bool, error) {
		s, _ := entity.(*string)
		if s == nil || len(args) != 1 {
			return false, errors.New("bad args") // System error
		}
		owner, _ := args[0].(string)
		if *s != owner {
			return false, nil // Validation failed (business logic)
		}
		return true, nil // Validation passed
	})

	var r engineReq
	r.Path.UserID = "u1"
	r.Path.Item.Owner = "u1"

	errs := Apply(context.Background(), &r)
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}

	r.Path.Item.Owner = "u2"
	errs = Apply(context.Background(), &r)
	if len(errs) == 0 {
		t.Fatalf("expected an error for ownership mismatch")
	}
}

func TestEngine_ApplyRules_ErrorPaths(t *testing.T) {
	resetRegistry()
	// Register one rule to avoid unknown for the first field
	Register("ok", func(ctx context.Context, entity any, args ...any) (bool, error) { return true, nil })

	type badReq struct {
		Path struct {
			A string `rule:"unknown()"`         // unknown rule name
			B string `rule:"owned_by($.Path.A"` // parse error (missing ')')
			C string `rule:"ok('x')"`           // ok
		}
	}

	var r badReq
	errs := Apply(context.Background(), &r)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}

	// Non-pointer should return error
	errs2 := Apply(context.Background(), r)
	if len(errs2) != 1 {
		t.Fatalf("expected 1 error for non-pointer, got %d", len(errs2))
	}
}

func TestEngine_RuleOnStructField(t *testing.T) {
	resetRegistry()
	Register("ok", func(ctx context.Context, entity any, args ...any) (bool, error) { return true, nil })

	type req struct {
		Path struct {
			S struct{ X string } `rule:"ok()"`
		}
	}
	var r req
	errs := Apply(context.Background(), &r)
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
}

func TestEngine_TopLevelNonStruct(t *testing.T) {
	resetRegistry()
	type req struct {
		A int
		B struct{ X string }
	}
	var r req
	errs := Apply(context.Background(), &r)
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
}

func TestEngine_ArityMismatch(t *testing.T) {
	resetRegistry()
	// fixed-arity rule: expects exactly one extra arg in addition to ctx, entity
	Register("need_one", func(ctx context.Context, e any, a1 any) (bool, error) { return true, nil })

	type req struct {
		Path struct {
			X string `rule:"need_one()"`
		}
	}
	var r req
	errs := Apply(context.Background(), &r)
	if len(errs) == 0 {
		t.Fatalf("expected arity mismatch error")
	}
}

func TestEngine_ContextVariables(t *testing.T) {
	resetRegistry()

	// Register a rule that checks ownership against current user
	Register("owned_by_current", func(ctx context.Context, entity any, args ...any) (bool, error) {
		ownerPtr, ok := entity.(*string)
		if !ok {
			return false, errors.New("entity must be *string") // System error
		}

		vars := GetContextVars(ctx)
		currentUser, ok := vars["current_user"]
		if !ok {
			return false, errors.New("current_user not found in context") // System error
		}

		if ownerPtr == nil {
			return false, errors.New("owner is nil") // System error
		}

		if *ownerPtr != currentUser {
			return false, nil // Validation failed (business logic)
		}
		return true, nil // Validation passed
	})

	Register("has_permission", func(ctx context.Context, entity any, args ...any) (bool, error) {
		if len(args) != 1 {
			return false, errors.New("permission argument required") // System error
		}

		requiredPermission, ok := args[0].(string)
		if !ok {
			return false, errors.New("permission must be string") // System error
		}

		vars := GetContextVars(ctx)
		permissions, ok := vars["permissions"]
		if !ok {
			return false, errors.New("permissions not found in context") // System error
		}

		// Simple string check for demo - in real usage, this might be a slice
		permsStr, ok := permissions.(string)
		if !ok {
			return false, errors.New("permissions must be string") // System error
		}

		if !strings.Contains(permsStr, requiredPermission) {
			return false, nil // Validation failed (business logic)
		}
		return true, nil // Validation passed
	})

	type req struct {
		Path struct {
			Owner string `rule:"owned_by_current()"`
		}
		Query struct {
			Action string `rule:"has_permission('write')"`
		}
	}

	var r req
	r.Path.Owner = "alice"
	r.Query.Action = "write"

	// Test with context variables
	ctx := context.Background()
	vars := ContextVars{
		"current_user": "alice",
		"permissions":  "read,write,admin",
	}
	ctx = WithContextVars(ctx, vars)

	errs := Apply(ctx, &r)
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}

	// Test with different user
	r.Path.Owner = "bob"
	errs = Apply(ctx, &r)
	if len(errs) == 0 {
		t.Fatalf("expected error for ownership mismatch")
	}

	// Test missing context variable
	ctxMissing := context.Background()
	errs = Apply(ctxMissing, &r)
	if len(errs) == 0 {
		t.Fatalf("expected error for missing context variables")
	}
}

func TestEngine_ContextVariableSyntax(t *testing.T) {
	resetRegistry()

	Register("test_rule", func(ctx context.Context, entity any, args ...any) (bool, error) {
		return true, nil // Always validation passed
	})

	type req struct {
		Path struct {
			UserID string `rule:"test_rule($current_user)"`
			ItemID string `rule:"test_rule($item_id)"`
		}
	}

	var r req
	r.Path.UserID = "test"
	r.Path.ItemID = "item123"

	ctx := context.Background()
	vars := ContextVars{
		"current_user": "alice",
		"item_id":      "item123",
	}
	ctx = WithContextVars(ctx, vars)

	errs := Apply(ctx, &r)
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
}

func TestEngine_BooleanExpression_AND_OR_NOT(t *testing.T) {
	resetRegistry()
	Register("owned_by", func(ctx context.Context, entity any, args ...any) (bool, error) {
		ownerPtr, _ := entity.(*string)
		if len(args) != 1 || ownerPtr == nil {
			return false, errors.New("bad") // System error
		}
		exp, _ := args[0].(string)
		if *ownerPtr != exp {
			return false, nil // Validation failed (business logic)
		}
		return true, nil // Validation passed
	})
	Register("system_readonly", func(ctx context.Context, entity any, args ...any) (bool, error) {
		return false, nil // Always validation failure
	})
	Register("godmode_enabled", func(ctx context.Context, entity any, args ...any) (bool, error) {
		return true, nil // Always validation success
	})

	type req struct {
		Path struct {
			ItemID string `rule:"(owned_by($current_user) and not system_readonly()) or godmode_enabled()"`
		}
	}
	var r req
	r.Path.ItemID = "i1"

	ctx := context.Background()
	ctx = WithContextVars(ctx, ContextVars{"current_user": "u1"})

	// owned_by fails, system_readonly returns validation error (not => pass), or godmode true => overall true
	errs := Apply(ctx, &r)
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
}

func TestEngine_TypedFixedArityRules(t *testing.T) {
	resetRegistry()

	// Register a typed, fixed-arity rule (no manual len(args) checks needed)
	Register("typed_owned_by", func(ctx context.Context, itemID *string, currentUser string) (bool, error) {
		if itemID == nil {
			return false, fmt.Errorf("item id is nil")
		}
		// Simple ownership check for test
		if *itemID == "item123" && currentUser == "alice" {
			return true, nil
		}
		return false, nil // Validation failed (business logic)
	})

	// Register a typed variadic rule
	Register("typed_in_list", func(ctx context.Context, value *string, allowed ...string) (bool, error) {
		if value == nil {
			return false, fmt.Errorf("value is nil")
		}
		v := *value
		for _, a := range allowed {
			if a == v {
				return true, nil
			}
		}
		return false, nil // Not in allowed list
	})

	type typedReq struct {
		Path struct {
			ItemID string `rule:"typed_owned_by($current_user)"`
			Status string `rule:"typed_in_list('active', 'inactive', 'pending')"`
		}
	}

	var r typedReq
	r.Path.ItemID = "item123"
	r.Path.Status = "active"

	ctx := context.Background()
	ctx = WithContextVars(ctx, ContextVars{"current_user": "alice"})

	errs := Apply(ctx, &r)
	if len(errs) != 0 {
		t.Fatalf("unexpected errs for valid typed rules: %v", errs)
	}

	// Test validation failure
	r.Path.Status = "invalid_status"
	errs = Apply(ctx, &r)
	if len(errs) == 0 {
		t.Fatalf("expected validation error for invalid status")
	}

	// Test arity enforcement - this should fail at call time, not in user code
	Register("typed_needs_two", func(ctx context.Context, entity *string, arg1 string, arg2 string) (bool, error) {
		return true, nil
	})

	type arityTestReq struct {
		Path struct {
			Field string `rule:"typed_needs_two('only_one_arg')"`
		}
	}

	var arityReq arityTestReq
	arityReq.Path.Field = "test"

	errs = Apply(ctx, &arityReq)
	if len(errs) == 0 {
		t.Fatalf("expected arity mismatch error")
	}
	// Verify it's a server error (arity mismatch), not a validation error
	if !strings.Contains(errs[0].Error(), "expects 2 args, got 1") {
		t.Fatalf("expected arity error, got: %v", errs[0])
	}
}
