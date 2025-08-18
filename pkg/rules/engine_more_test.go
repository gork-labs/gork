package rules

import (
	"context"
	"reflect"
	"testing"
)

func Test_valueAsEntity_NonAddressable(t *testing.T) {
	v := reflect.ValueOf("immutable") // non-addressable
	got := valueAsEntity(v)
	if s, ok := got.(string); !ok || s != "immutable" {
		t.Fatalf("expected raw interface value, got %#v", got)
	}
}

func TestEngine_RunInvocation_ResolveArgsError(t *testing.T) {
	resetRegistry()
	Register("ok", func(ctx context.Context, e any, args ...any) (bool, error) { return true, nil })

	type req struct {
		Path struct {
			A string `rule:"ok($.NoSuch)"`
		}
	}
	var r req
	errs := Apply(context.Background(), &r)
	if len(errs) == 0 {
		t.Fatalf("expected resolve error to be reported")
	}
}
