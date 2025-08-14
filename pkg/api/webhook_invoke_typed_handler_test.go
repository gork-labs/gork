package api

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

type provObj struct{ ID string }
type userCfg struct {
	A string `json:"a"`
}

func TestInvokeTypedEventHandler_ValidJSONPath(t *testing.T) {
	ctx := context.Background()
	// Handler with correct signature; ensure valid JSON populates user meta and call succeeds
	called := false
	h := func(ctx context.Context, p *provObj, u *userCfg) error {
		if p == nil || p.ID != "42" {
			t.Fatalf("unexpected provider payload: %#v", p)
		}
		if u == nil || u.A != "x" {
			t.Fatalf("unexpected user meta: %#v", u)
		}
		called = true
		return nil
	}
	ev := WebhookEvent{ProviderObject: &provObj{ID: "42"}, UserMetaJSON: json.RawMessage(`{"a":"x"}`)}
	code, err := invokeTypedEventHandler(ctx, h, ev, true)
	if err != nil || code != 0 || !called {
		t.Fatalf("expected success, got code=%d err=%v called=%v", code, err, called)
	}
}

func TestValidateEventHandlerSignature_OutSecondNotError(t *testing.T) {
	// func(ctx context.Context, *provObj, *userCfg) (int, string) -> invalid return type
	f := func(context.Context, *provObj, *userCfg) (int, string) { return 0, "" }
	err := validateEventHandlerSignature(reflect.TypeOf(f))
	if err == nil {
		t.Fatal("expected error for return not error")
	}
}
