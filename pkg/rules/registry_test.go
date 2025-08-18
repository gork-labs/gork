package rules

import (
	"context"
	"testing"
)

func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}

func TestRegisterRule_ValidAndDuplicate(t *testing.T) {
	resetRegistry()
	Register("r1", func(ctx context.Context, e any, args ...any) (bool, error) { return true, nil })
	if _, ok := getRule("r1"); !ok {
		t.Fatal("rule not registered")
	}
	mustPanic(t, func() {
		Register("r1", func(ctx context.Context, e any, args ...any) (bool, error) { return true, nil })
	})
}

func TestRegisterRule_SignaturePanics(t *testing.T) {
	resetRegistry()
	// empty name
	mustPanic(t, func() { Register("", func(ctx context.Context, e any, args ...any) (bool, error) { return true, nil }) })
	// non-function
	mustPanic(t, func() { Register("x", 123) })
	// too few params
	mustPanic(t, func() { Register("x1", func() error { return nil }) })
	// wrong first param
	mustPanic(t, func() { Register("x2", func(s string, e any) error { return nil }) })
	// wrong returns (none)
	mustPanic(t, func() { Register("x3", func(ctx context.Context, e any) {}) })
	// wrong return type
	mustPanic(t, func() { Register("x4", func(ctx context.Context, e any) int { return 0 }) })
	// first return value not bool
	mustPanic(t, func() { Register("x5", func(ctx context.Context, e any) (string, error) { return "", nil }) })
	// second return value not error
	mustPanic(t, func() { Register("x6", func(ctx context.Context, e any) (bool, string) { return true, "" }) })
}
