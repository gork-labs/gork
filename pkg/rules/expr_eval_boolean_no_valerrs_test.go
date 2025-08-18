package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestEvalBooleanExpr_BooleanAnd_NoValidationErrors(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	// true and false -> pass=false, but no validation errors to report
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "true and false"); errs != nil {
		t.Fatalf("expected nil errs for pure boolean false, got %v", errs)
	}
}
