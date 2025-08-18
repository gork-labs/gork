package api

import (
	"context"
	"encoding/json"
	"testing"
)

type (
	providerPayload struct{ ID string }
	userMeta        struct{ X string }
)

func TestInvokeTypedEventHandler_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("provider type mismatch", func(t *testing.T) {
		// Handler expects *providerPayload
		h := func(ctx context.Context, p *providerPayload, u *userMeta) error { return nil }
		// Event carries wrong type
		ev := WebhookEvent{ProviderObject: &struct{ Z int }{Z: 1}}
		if _, err := invokeTypedEventHandler(ctx, h, ev, false); err == nil {
			t.Fatal("expected provider type mismatch error")
		}
	})

	t.Run("user meta must be pointer", func(t *testing.T) {
		// Handler expects non-pointer user meta -> should fail signature validation before call
		h := func(ctx context.Context, p *providerPayload, u userMeta) error { return nil }
		ev := WebhookEvent{ProviderObject: &providerPayload{ID: "1"}}
		if _, err := invokeTypedEventHandler(ctx, h, ev, false); err == nil {
			t.Fatal("expected signature validation error for non-pointer user meta")
		}
	})

	t.Run("invalid user meta JSON strict vs non-strict", func(t *testing.T) {
		// Valid signature; invalid JSON
		h := func(ctx context.Context, p *providerPayload, u *userMeta) error { return nil }
		ev := WebhookEvent{ProviderObject: &providerPayload{ID: "1"}, UserMetaJSON: json.RawMessage(`{"x": 1}`)}

		// Non-strict: should ignore and not error
		if _, err := invokeTypedEventHandler(ctx, h, ev, false); err != nil {
			t.Fatalf("non-strict should not error, got %v", err)
		}

		// Strict: should return 400
		if code, err := invokeTypedEventHandler(ctx, h, ev, true); err == nil || code != 400 {
			t.Fatalf("strict should return 400 error, got code=%d err=%v", code, err)
		}
	})

	t.Run("nil provider and nil user meta are accepted", func(t *testing.T) {
		// Accept interface pointer for provider -> allow nil
		h := func(ctx context.Context, p *interface{}, u *userMeta) error { return nil }
		ev := WebhookEvent{ProviderObject: nil, UserMetaJSON: nil}
		if _, err := invokeTypedEventHandler(ctx, h, ev, false); err != nil {
			t.Fatalf("expected success with nil args, got err=%v", err)
		}
	})
}
