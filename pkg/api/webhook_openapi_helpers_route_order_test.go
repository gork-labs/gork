package api

import "testing"

func TestBuildWebhookOperation_PrefersRouteHandledEventsOverReflected(t *testing.T) {
	// Prepare route with explicit handled events and handler that returns different valid events
	route := &RouteInfo{
		HandlerName:          "wh",
		RequestType:          nil,
		ResponseType:         nil,
		Options:              &HandlerOption{Tags: []string{"webhooks"}},
		WebhookHandledEvents: []string{"evt.a", "evt.b"},
		WebhookHandler:       &CustomWebhookHandler{},
	}

	spec := &OpenAPISpec{Paths: map[string]*PathItem{}, Components: &Components{Schemas: map[string]*Schema{}}}
	gen := NewConventionOpenAPIGenerator(spec, nil)
	op := &Operation{Responses: map[string]*Response{}}

	out := gen.buildWebhookOperation(route, spec.Components, op)
	if out.Extensions == nil {
		t.Fatalf("expected extensions")
	}
	// With explicit handled events and a handler, generator must emit rich entries
	entries, ok := out.Extensions["x-webhook-events"].([]map[string]interface{})
	if !ok || len(entries) != 2 {
		t.Fatalf("expected x-webhook-events to be []map[string]interface{} with 2 entries, got %T %v", out.Extensions["x-webhook-events"], out.Extensions["x-webhook-events"])
	}
}
