package rules

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// resolve resolves argument tokens using root (request), parent (same level), and context variables.
func resolve(ctx context.Context, root any, parent any, tokens []argToken) ([]any, error) {
	out := make([]any, 0, len(tokens))
	for _, t := range tokens {
		v, err := resolveSingle(ctx, root, parent, t)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func resolveSingle(ctx context.Context, root any, parent any, t argToken) (any, error) {
	switch t.Kind {
	case argInvalid:
		return nil, fmt.Errorf("rules: cannot resolve invalid argument token")
	case argFieldRef:
		return resolveFieldRef(root, parent, t)
	case argContextVar:
		return resolveContextVar(ctx, t)
	case argString:
		return t.Str, nil
	case argBool:
		return t.Bool, nil
	case argNumber:
		return t.Num, nil
	case argNull:
		return nil, nil
	default:
		return nil, fmt.Errorf("rules: unsupported arg kind %v", t.Kind)
	}
}

func resolveFieldRef(root any, parent any, t argToken) (any, error) {
	var start reflect.Value
	if t.IsAbsolute {
		start = reflect.ValueOf(root)
	} else {
		start = reflect.ValueOf(parent)
	}
	v, err := resolveFrom(start, t.Segments)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

func resolveContextVar(ctx context.Context, t argToken) (any, error) {
	vars := GetContextVars(ctx)
	if val, ok := vars[t.ContextVar]; ok {
		if val == nil {
			return nil, fmt.Errorf("rules: context variable %q is nil", t.ContextVar)
		}
		return val, nil
	}
	return nil, fmt.Errorf("rules: context variable %q not found", t.ContextVar)
}

// ContextVars holds per-request variables for rule evaluation.
type ContextVars map[string]any

// WithContextVars returns a new context carrying the provided ContextVars.
func WithContextVars(ctx context.Context, vars ContextVars) context.Context {
	return context.WithValue(ctx, contextVarsKey, vars)
}

// GetContextVars retrieves ContextVars from the context; returns empty if none.
func GetContextVars(ctx context.Context) ContextVars {
	if vars, ok := ctx.Value(contextVarsKey).(ContextVars); ok {
		return vars
	}
	return make(ContextVars)
}

// contextKey is the internal key type for storing context variables in context.Context.
type contextKey string

const contextVarsKey contextKey = "gork_rules_context_vars"

// cached accessor for performance.
var accCache sync.Map // key:string -> accessorFunc

func makeKey(rootType reflect.Type, isAbs bool, segs []string) string {
	return fmt.Sprintf("%p|%t|%s", rootType, isAbs, strings.Join(segs, ","))
}

func resolveFrom(start reflect.Value, segs []string) (reflect.Value, error) {
	base, err := normalizeStart(start)
	if err != nil {
		return reflect.Value{}, err
	}
	// include whether the original start was a pointer to avoid subtle cache collisions.
	key := makeKey(base.Type(), start.Kind() == reflect.Ptr, segs)
	if fn, ok := accCache.Load(key); ok {
		return fn.(func(reflect.Value) (reflect.Value, error))(base)
	}
	fn, err := buildAccessor(base.Type(), segs)
	if err != nil {
		return reflect.Value{}, err
	}
	accCache.Store(key, fn)
	return fn(base)
}

func normalizeStart(start reflect.Value) (reflect.Value, error) {
	if !start.IsValid() {
		return reflect.Value{}, fmt.Errorf("rules: invalid start for field resolution")
	}
	if start.Kind() == reflect.Ptr {
		if start.IsNil() {
			return reflect.Value{}, fmt.Errorf("rules: nil start value")
		}
		start = start.Elem()
	}
	if start.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("rules: start value must be struct")
	}
	return start, nil
}

func buildAccessor(startType reflect.Type, segs []string) (func(reflect.Value) (reflect.Value, error), error) {
	indices := make([]int, 0, len(segs))
	cur := startType
	for _, name := range segs {
		next, idx, err := advanceAccessor(cur, name, len(indices), len(segs))
		if err != nil {
			return nil, err
		}
		indices = append(indices, idx)
		cur = next
	}
	return func(root reflect.Value) (reflect.Value, error) {
		v := root
		for _, idx := range indices {
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return reflect.Value{}, fmt.Errorf("rules: nil pointer while resolving")
				}
				v = v.Elem()
			}
			v = v.Field(idx)
		}
		return v, nil
	}, nil
}

func advanceAccessor(cur reflect.Type, name string, pos, total int) (next reflect.Type, idx int, err error) {
	i, f, ok := fieldByName(cur, name)
	if !ok {
		return nil, -1, fmt.Errorf("rules: field %q not found", name)
	}
	if f.Type.Kind() == reflect.Slice && f.Type.Elem().Kind() == reflect.Uint8 && pos < total-1 {
		return nil, -1, fmt.Errorf("rules: cannot traverse into raw body bytes")
	}
	next = f.Type
	if next.Kind() == reflect.Ptr {
		next = next.Elem()
	}
	return next, i, nil
}

func fieldByName(t reflect.Type, name string) (int, reflect.StructField, bool) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Name == name {
			return i, f, true
		}
	}
	return -1, reflect.StructField{}, false
}
