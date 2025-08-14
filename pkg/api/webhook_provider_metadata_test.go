package api

import (
	"context"
	"net/http"
	"testing"
)

// Use the minimal webhook request type from handler_factory_httpfunc_test.go

// Provider payload/user meta types (unused beyond typing)
type acmeProviderPayload struct{ ID string }
type acmeUserMeta struct{ Note string }

// Provider implementing handler + provider info exposure
type testProvider struct{}

func (p *testProvider) ParseRequest(req testWebhookReq) (WebhookEvent, error) {
	return WebhookEvent{Type: "ignored", ProviderObject: &acmeProviderPayload{ID: "1"}}, nil
}
func (p *testProvider) SuccessResponse() interface{} { return map[string]any{"ok": true} }
func (p *testProvider) ErrorResponse(err error) interface{} {
	return map[string]any{"error": err.Error()}
}
func (p *testProvider) GetValidEventTypes() []string { return []string{"evt.a", "evt.b", "evt.c"} }

// Expose provider metadata via required method
func (p *testProvider) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{
		Name:    "AcmePay",
		Website: "https://acme.example.com",
		DocsURL: "https://docs.acme.example.com/webhooks",
	}
}

func TestWebhookProviderMetadataAndHandledEvents(t *testing.T) {
	// Build webhook HTTP handler with two registered events
	h := &testProvider{}
	httpHandler := WebhookHandlerFunc[testWebhookReq](
		h,
		WithEventHandler[acmeProviderPayload, acmeUserMeta]("evt.a", func(ctx context.Context, p *acmeProviderPayload, u *acmeUserMeta) error {
			return nil
		}),
		WithEventHandler[acmeProviderPayload, acmeUserMeta]("evt.b", func(ctx context.Context, p *acmeProviderPayload, u *acmeUserMeta) error {
			return nil
		}),
	)

	// Create RouteInfo using internal factory
	mockAdapter := &mockTypedRouterAdapter{}
	_, info := createHandlerFromAny(mockAdapter, httpHandler)
	info.Method = http.MethodPost
	info.Path = "/webhooks/acme"

	// Validate RouteInfo carries metadata
	if info.WebhookProviderInfo == nil || info.WebhookProviderInfo.Name != "AcmePay" {
		t.Fatalf("expected provider info with name 'AcmePay', got %#v", info.WebhookProviderInfo)
	}
	if got := len(info.WebhookHandledEvents); got != 2 {
		t.Fatalf("expected 2 handled events, got %d: %v", got, info.WebhookHandledEvents)
	}

	// Generate OpenAPI and verify extensions are attached to the webhook route
	reg := NewRouteRegistry()
	reg.Register(info)
	spec := GenerateOpenAPI(reg)
	path := spec.Paths["/webhooks/acme"]
	if path == nil || path.Post == nil {
		t.Fatal("expected POST operation for /webhooks/acme")
	}
	op := path.Post
	if op.Extensions == nil {
		t.Fatal("expected extensions to be present for webhook metadata")
	}
	prov, ok := op.Extensions["x-webhook-provider"].(map[string]string)
	if !ok {
		t.Fatalf("expected x-webhook-provider map, got %T", op.Extensions["x-webhook-provider"])
	}
	if prov["name"] != "AcmePay" || prov["website"] == "" || prov["docs"] == "" {
		t.Fatalf("unexpected provider extension: %#v", prov)
	}
	// With registered handlers, x-webhook-events is a rich array of objects
	entries, ok := op.Extensions["x-webhook-events"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected x-webhook-events to be []map[string]interface{}, got %T", op.Extensions["x-webhook-events"])
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 event entries, got %d", len(entries))
	}
	// Verify presence of both events and basic metadata
	seenA, seenB := false, false
	for _, e := range entries {
		ev, _ := e["event"].(string)
		if ev == "evt.a" || ev == "evt.b" {
			if ev == "evt.a" {
				seenA = true
			} else {
				seenB = true
			}
			// operationId should be a non-empty string
			if opid, ok := e["operationId"].(string); !ok || opid == "" {
				t.Fatalf("expected non-empty operationId, got %#v", e["operationId"])
			}
			// userPayloadSchema should be a *Schema; allow either direct object type or $ref
			sch, ok := e["userPayloadSchema"].(*Schema)
			if !ok || sch == nil {
				t.Fatalf("expected userPayloadSchema to be *Schema, got %T", e["userPayloadSchema"])
			}
			if sch.Ref == "" && sch.Type != "object" {
				t.Fatalf("expected user metadata schema to be object or $ref, got type=%s ref=%s", sch.Type, sch.Ref)
			}
		}
	}
	if !(seenA && seenB) {
		t.Fatalf("missing expected events in %v", entries)
	}
}
