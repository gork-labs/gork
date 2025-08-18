package rules

import (
	"context"
	"reflect"
	"testing"
)

func TestEvalBooleanExpr_PassTrueLiteral(t *testing.T) {
	var root, parent struct{}
	ent := "e"
	if errs := evalBooleanExpr(context.Background(), reflect.ValueOf(&root).Elem(), reflect.ValueOf(&parent).Elem(), &ent, "true"); errs != nil {
		t.Fatalf("expected nil errs, got %v", errs)
	}
}
