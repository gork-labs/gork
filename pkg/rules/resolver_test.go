package rules

import (
	"context"
	"testing"
)

type testReq struct {
	Path struct {
		UserID string
		Item   struct {
			Owner string
		}
	}
	Extra struct {
		P *struct{ V string }
	}
	Raw struct {
		Body []byte
	}
	Headers struct {
		Authorization string
	}
	Body struct {
		Amount float64
		Type   string
	}
}

func TestResolveArgsWithContext(t *testing.T) {
	var r testReq
	r.Path.UserID = "u1"
	r.Path.Item.Owner = "u1"
	r.Headers.Authorization = "Bearer abc"
	r.Body.Amount = 42.0
	r.Body.Type = "admin"
	r.Extra.P = &struct{ V string }{V: "ok"}

	absUser := argToken{Kind: argFieldRef, IsAbsolute: true, Segments: []string{"Path", "UserID"}}
	relOwner := argToken{Kind: argFieldRef, IsAbsolute: false, Segments: []string{"Owner"}}
	absAmt := argToken{Kind: argFieldRef, IsAbsolute: true, Segments: []string{"Body", "Amount"}}
	absPtr := argToken{Kind: argFieldRef, IsAbsolute: true, Segments: []string{"Extra", "P", "V"}}

	litBool := argToken{Kind: argBool, Bool: true}
	litBoolFalse := argToken{Kind: argBool, Bool: false}
	litStr := argToken{Kind: argString, Str: "hello"}
	litNum := argToken{Kind: argNumber, Num: 3.14}
	litNull := argToken{Kind: argNull}
	ctx := context.Background()
	args, err := resolve(ctx, &r, &r.Path.Item, []argToken{absUser, relOwner, absAmt, absPtr, litBool, litNull, litStr, litNum, litBoolFalse})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if args[0].(string) != "u1" {
		t.Fatalf("want u1, got %v", args[0])
	}
	if args[1].(string) != "u1" {
		t.Fatalf("want u1, got %v", args[1])
	}
	if args[2].(float64) != 42.0 {
		t.Fatalf("want 42.0, got %v", args[2])
	}
	if args[3].(string) != "ok" {
		t.Fatalf("want ok, got %v", args[3])
	}
	if args[4].(bool) != true {
		t.Fatalf("want true, got %v", args[4])
	}
	if args[5] != nil {
		t.Fatalf("want nil, got %v", args[5])
	}
	if args[6].(string) != "hello" {
		t.Fatalf("want 'hello', got %v", args[6])
	}
	if args[7].(float64) != 3.14 {
		t.Fatalf("want 3.14, got %v", args[7])
	}
	if args[8].(bool) != false {
		t.Fatalf("want false, got %v", args[8])
	}

	// Call again to hit cache path
	args2, err := resolve(ctx, &r, &r.Path.Item, []argToken{absUser})
	if err != nil || args2[0].(string) != "u1" {
		t.Fatalf("cache path failed: %v %v", args2, err)
	}
}

func TestResolveArgsWithContext_Errors(t *testing.T) {
	var r testReq
	r.Raw.Body = []byte("{}")
	ctx := context.Background()
	// Attempt to traverse into raw body bytes
	tok := argToken{Kind: argFieldRef, IsAbsolute: true, Segments: []string{"Raw", "Body", "X"}}
	if _, err := resolve(ctx, &r, &r.Path, []argToken{tok}); err == nil {
		t.Fatal("expected error when traversing into raw body bytes")
	}

	// Start not a struct
	tok2 := argToken{Kind: argFieldRef, IsAbsolute: false, Segments: []string{"X"}}
	if _, err := resolve(ctx, &r, 42, []argToken{tok2}); err == nil {
		t.Fatal("expected error for non-struct parent start")
	}

	// Nil pointer parent
	type P struct{ X int }
	var p *P
	if _, err := resolve(ctx, &r, p, []argToken{tok2}); err == nil {
		t.Fatal("expected error for nil parent pointer")
	}

	// Parent is nil interface (invalid start)
	var nilIface any
	if _, err := resolve(ctx, &r, nilIface, []argToken{tok2}); err == nil {
		t.Fatal("expected error for invalid parent start")
	}

	// Field not found
	tok3 := argToken{Kind: argFieldRef, IsAbsolute: true, Segments: []string{"Path", "Nope"}}
	if _, err := resolve(ctx, &r, &r.Path, []argToken{tok3}); err == nil {
		t.Fatal("expected error for missing field")
	}

	// Nil root pointer with absolute ref
	var rnil *testReq
	if _, err := resolve(ctx, rnil, &r.Path, []argToken{{Kind: argFieldRef, IsAbsolute: true, Segments: []string{"Path", "UserID"}}}); err == nil {
		t.Fatal("expected error for nil root pointer")
	}

	// Nil pointer encountered mid-traversal
	r.Extra.P = nil
	tok4 := argToken{Kind: argFieldRef, IsAbsolute: true, Segments: []string{"Extra", "P", "V"}}
	if _, err := resolve(ctx, &r, &r.Path, []argToken{tok4}); err == nil {
		t.Fatal("expected error for nil pointer while resolving")
	}
}

func TestResolveArgsWithContext_UnsupportedKind(t *testing.T) {
	var r testReq
	ctx := context.Background()

	// Test unsupported kind (invalid enum value)
	bad := argToken{Kind: argKind(999)}
	if _, err := resolve(ctx, &r, &r.Path, []argToken{bad}); err == nil {
		t.Fatal("expected error for unsupported arg kind")
	}

	// Test argInvalid specifically
	invalid := argToken{Kind: argInvalid}
	if _, err := resolve(ctx, &r, &r.Path, []argToken{invalid}); err == nil {
		t.Fatal("expected error for argInvalid token")
	}

	// Verify the error message contains "invalid argument token"
	_, err := resolve(ctx, &r, &r.Path, []argToken{invalid})
	if err == nil {
		t.Fatal("expected error for argInvalid token")
	}
	expectedMsg := "rules: cannot resolve invalid argument token"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}
