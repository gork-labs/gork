package api

import (
	"reflect"
	"testing"
)

type fakeEventTypes2 struct{}

func (fakeEventTypes2) GetValidEventTypes() []string { return []string{"foo", "bar"} }

type simpleStruct2 struct {
	ID string `json:"id"`
}

type noOut2 struct{}

func (noOut2) SuccessResponse() {}

type sucConcrete2 struct{}

type sucProvider2 struct{}

func (sucProvider2) SuccessResponse() interface{} { return sucConcrete2{} }

type nilErr2 struct{}

func (nilErr2) ErrorResponse(err error) interface{} { return nil }

func TestBuildWebhookOperation_FallbackToReflectedEvents(t *testing.T) {
	route := &RouteInfo{
		HandlerName: "wh2",
		Options:     &HandlerOption{Tags: []string{"webhooks"}},
		// No WebhookHandledEvents provided -> should fall back to provider-advertised list
		WebhookHandler: fakeEventTypes2{},
	}

	spec := &OpenAPISpec{Paths: map[string]*PathItem{}, Components: &Components{Schemas: map[string]*Schema{}}}
	gen := NewConventionOpenAPIGenerator(spec, nil)
	op := &Operation{Responses: map[string]*Response{}}

	out := gen.buildWebhookOperation(route, spec.Components, op)
	if out.Extensions == nil {
		t.Fatalf("expected extensions to be present")
	}
	// No registered handlers → should still be array of objects
	entries, ok := out.Extensions["x-webhook-events"].([]map[string]interface{})
	if !ok || len(entries) != 2 {
		t.Fatalf("expected reflected events as []map[string]interface{}, got %T %v", out.Extensions["x-webhook-events"], out.Extensions["x-webhook-events"])
	}
}

func TestBuildWebhookRequestBodySchema_InterfaceVsStruct(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, nil)

	s := gen.buildWebhookRequestBodySchema(reflect.TypeOf((*WebhookRequest)(nil)).Elem(), gen.spec.Components)
	if s == nil || s.Type != "object" {
		t.Fatalf("expected generic object schema for interface, got %#v", s)
	}

	s2 := gen.buildWebhookRequestBodySchema(reflect.TypeOf(simpleStruct2{}), gen.spec.Components)
	if s2 == nil {
		t.Fatalf("expected non-nil schema for struct type")
	}
}

func TestGetWebhookResponseType_AdditionalBranches(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, nil)

	if tp := gen.getWebhookResponseType(noOut2{}, "SuccessResponse"); tp != nil {
		t.Fatalf("expected nil for no-output method")
	}

	if tp := gen.getWebhookResponseType(sucProvider2{}, "SuccessResponse"); tp == nil || tp.Kind() != reflect.Struct {
		t.Fatalf("expected struct type for interface-concrete return, got %v", tp)
	}

	if tp := gen.getWebhookResponseType(nilErr2{}, "ErrorResponse"); tp == nil || tp.Kind() != reflect.Interface || tp.String() != "interface {}" {
		t.Fatalf("expected interface{} for nil interface return, got %v", tp)
	}
}
