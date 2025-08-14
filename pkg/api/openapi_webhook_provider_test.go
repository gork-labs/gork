package api

import (
	"reflect"
	"testing"
)

func TestBuildWebhookOperation_ProviderAndEventsExtensions(t *testing.T) {
	spec := &OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}
	gen := NewConventionOpenAPIGenerator(spec, nil)

	route := &RouteInfo{
		HandlerName: "StripeWebhook",
		Method:      "POST",
		Path:        "/webhooks/stripe",
		Options:     &HandlerOption{Tags: []string{"webhooks"}},
		// Use interface request type to exercise the generic request branch
		RequestType: reflect.TypeOf((*WebhookRequest)(nil)).Elem(),
		// Simulate provider metadata coming from the concrete handler
		WebhookProviderInfo: &WebhookProviderInfo{
			Name:    "Stripe",
			Website: "https://stripe.com",
			DocsURL: "https://stripe.com/docs/webhooks",
		},
		// Simulate explicitly handled events on the route
		WebhookHandledEvents: []string{"payment_intent.succeeded", "invoice.paid"},
	}

	op := gen.buildWebhookOperation(route, spec.Components, &Operation{Responses: map[string]*Response{}})

	if op == nil {
		t.Fatal("operation should not be nil")
	}

	// Verify provider extension
	provRaw, ok := op.Extensions["x-webhook-provider"]
	if !ok {
		t.Fatalf("expected x-webhook-provider extension present")
	}
	prov, ok := provRaw.(map[string]string)
	if !ok {
		t.Fatalf("x-webhook-provider should be map[string]string, got %T", provRaw)
	}
	if prov["name"] != "Stripe" || prov["website"] == "" || prov["docs"] == "" {
		t.Fatalf("unexpected provider info: %#v", prov)
	}

	// Verify events extension is present and is a unified array of objects
	evEntries, ok := op.Extensions["x-webhook-events"].([]map[string]interface{})
	if !ok || len(evEntries) != 2 {
		t.Fatalf("expected x-webhook-events to be []map[string]interface{} with 2 entries, got %T %v", op.Extensions["x-webhook-events"], op.Extensions["x-webhook-events"])
	}

	// Verify request body and responses were populated
	if op.RequestBody == nil || op.RequestBody.Content["application/json"].Schema == nil {
		t.Fatalf("expected request body schema for webhook")
	}
	if op.Responses["200"] == nil || op.Responses["400"] == nil || op.Responses["422"] == nil || op.Responses["500"] == nil {
		t.Fatalf("expected standard webhook responses 200/400/422/500 to be present")
	}
}
