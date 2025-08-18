package rules

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
)

// RuleFunc is the canonical rule function type alias.
// The function may choose to ignore variadic args.
// Signature: func(ctx context.Context, entity any, args ...any) (bool, error).
// Return values:
//
//	(true, nil)   = validation passed
//	(false, nil)  = validation failed (business logic)
//	(false, error) = system error (network, DB failure, etc.)
type RuleFunc = any

type ruleDescriptor struct {
	fn         reflect.Value
	fnName     string
	isVariadic bool
	numIn      int
}

var (
	regMu    sync.RWMutex
	registry = make(map[string]ruleDescriptor)
)

// Register registers a rule by name. Panics on duplicate name or invalid signature.
// Accepted signatures:
//   - func(context.Context, any) (bool, error)
//   - func(context.Context, any, ...any) (bool, error)
//   - func(context.Context, any, any, any, ...) (bool, error) (fixed arity >=2)
func Register(name string, fn RuleFunc) {
	if name == "" {
		panic("rules: rule name must not be empty")
	}

	v := reflect.ValueOf(fn)
	t := v.Type()
	if t.Kind() != reflect.Func {
		panic(fmt.Sprintf("rules: rule %q must be a function", name))
	}
	if t.NumIn() < 2 {
		panic(fmt.Sprintf("rules: rule %q must accept at least (context.Context, any)", name))
	}
	// First parameter must implement context.Context
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !t.In(0).Implements(ctxType) {
		panic(fmt.Sprintf("rules: rule %q first parameter must be context.Context", name))
	}
	// Second parameter is entity (any)
	// Return exactly 2 values: (bool, error)
	if t.NumOut() != 2 {
		panic(fmt.Sprintf("rules: rule %q must return exactly two values (bool, error)", name))
	}
	if t.Out(0).Kind() != reflect.Bool {
		panic(fmt.Sprintf("rules: rule %q first return value must be bool", name))
	}
	if !t.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic(fmt.Sprintf("rules: rule %q second return value must implement error", name))
	}

	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("rules: rule %q is already registered", name))
	}
	fnName := runtime.FuncForPC(v.Pointer()).Name()
	registry[name] = ruleDescriptor{fn: v, fnName: fnName, isVariadic: t.IsVariadic(), numIn: t.NumIn()}
}

// getRule returns the registered rule descriptor.
func getRule(name string) (ruleDescriptor, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	d, ok := registry[name]
	return d, ok
}

// resetRegistry clears the registry (for tests only).
func resetRegistry() {
	regMu.Lock()
	defer regMu.Unlock()
	registry = make(map[string]ruleDescriptor)
}
