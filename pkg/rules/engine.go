// Package rules implements a lightweight, tag-driven rules engine
// that discovers `rule` tags on request structs and executes
// registered rule functions against the referenced entities.
package rules

import (
	"context"
	"fmt"
	"reflect"
)

// Apply scans a conventional request struct for `rule` tags and executes
// It returns a slice of errors returned by rules, preserving invocation order.
func Apply(ctx context.Context, reqPtr any) []error {
	var errs []error
	rv := reflect.ValueOf(reqPtr)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return []error{fmt.Errorf("rules: request must be a pointer to struct")}
	}
	root := rv.Elem()
	rt := root.Type()

	// Traverse recursively under each top-level struct field
	for i := 0; i < rt.NumField(); i++ {
		secField := rt.Field(i)
		secValue := root.Field(i)
		if secField.Type.Kind() != reflect.Struct {
			continue
		}
		applyRulesInStruct(ctx, root, secValue, &errs)
	}
	return errs
}

func applyRulesInStruct(ctx context.Context, root reflect.Value, parent reflect.Value, errs *[]error) {
	t := parent.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := parent.Field(i)
		if f.Type.Kind() == reflect.Struct {
			applyRulesInStruct(ctx, root, v, errs)
		}
		processFieldRules(ctx, root, parent, f, v, errs)
	}
}

func processFieldRules(ctx context.Context, root, parent reflect.Value, f reflect.StructField, v reflect.Value, errs *[]error) {
	ruleTag := f.Tag.Get("rule")
	if ruleTag == "" {
		return
	}
	entity := valueAsEntity(v)
	// Always treat as boolean expression
	if e := evalBooleanExpr(ctx, root, parent, entity, ruleTag); len(e) > 0 {
		*errs = append(*errs, e...)
	}
}

func valueAsEntity(v reflect.Value) any {
	if v.CanAddr() {
		return v.Addr().Interface()
	}
	return v.Interface()
}
