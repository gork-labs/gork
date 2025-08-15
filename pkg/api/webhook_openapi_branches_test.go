package api

import (
	"reflect"
	"testing"
)

// Test additional OpenAPI webhook generation branches for 100% coverage
func TestWebhookOpenAPI_GeneratorBranches(t *testing.T) {
	gen := &ConventionOpenAPIGenerator{}

	type StripeWebhookRequest struct{}

	t.Run("getWebhookEventTypes falls back by provider when no handler", func(t *testing.T) {
		route := &RouteInfo{RequestType: reflect.TypeOf(StripeWebhookRequest{})}
		events := gen.getWebhookEventTypes(route)
		if len(events) != 0 {
			t.Fatalf("expected no fallback event types, got %v", events)
		}
	})

	t.Run("processWebhookRequestBody handles interface request type", func(t *testing.T) {
		op := &Operation{Responses: map[string]*Response{}}
		comps := &Components{Schemas: map[string]*Schema{}}
		// Use interface type to trigger generic schema branch
		gen.processWebhookRequestBody(reflect.TypeOf((*WebhookRequest)(nil)).Elem(), op, comps)
		if op.RequestBody == nil || op.RequestBody.Content["application/json"].Schema == nil {
			t.Fatal("expected request body schema to be set")
		}
		if op.RequestBody.Content["application/json"].Schema.Type != "object" {
			t.Fatalf("expected generic object schema, got %s", op.RequestBody.Content["application/json"].Schema.Type)
		}
	})

	t.Run("addWebhookResponses falls back when handler missing", func(t *testing.T) {
		op := &Operation{Responses: map[string]*Response{}}
		comps := &Components{Schemas: map[string]*Schema{}}
		gen.addWebhookResponses(op, comps, &RouteInfo{WebhookHandler: nil})
		if _, ok := op.Responses["200"]; !ok {
			t.Fatal("expected 200 fallback response")
		}
		if _, ok := op.Responses["400"]; !ok {
			t.Fatal("expected 400 fallback response")
		}
	})
}
