package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

type nestedUnionCard struct {
	Type   string `gork:"type,discriminator=card"`
	Number string `gork:"number"`
}

type nestedUnionBank struct {
	Type    string `gork:"type,discriminator=bank"`
	Account string `gork:"account"`
}

type nestedUnionPayment = unions.Union2[nestedUnionCard, nestedUnionBank]

type nestedUnionResponse struct {
	Body struct {
		Method nestedUnionPayment `gork:"method"`
	}
}

type nestedUnionPointerRequest struct {
	Body struct {
		Method *nestedUnionPayment `gork:"method"`
	}
}

type nestedUnionBodyRequest struct {
	Body nestedUnionPayment
}

type nestedUnionPointerResponse struct {
	Body *nestedUnionPayment
}

func assertPaymentUnion(t *testing.T, name string, schema *Schema) {
	t.Helper()
	if schema == nil || len(schema.OneOf) != 2 || schema.Discriminator == nil || len(schema.Discriminator.Mapping) != 2 {
		t.Errorf("%s = %+v, want a oneOf of 2 members with a discriminator", name, schema)
	}
}

func TestNestedUnionWritesTheSetMember(t *testing.T) {
	handler := func(context.Context, struct{}) (*nestedUnionResponse, error) {
		resp := &nestedUnionResponse{}
		resp.Body.Method.A = &nestedUnionCard{Type: "card", Number: "4242"}
		return resp, nil
	}
	httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, handler)

	rec := httptest.NewRecorder()
	httpHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if want := `{"method":{"number":"4242","type":"card"}}`; strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("body = %s, want %s", rec.Body.String(), want)
	}
}

func TestPointerAndBodyUnionsGiveOneOf(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Post("/pointer", func(context.Context, nestedUnionPointerRequest) (*nestedUnionPointerResponse, error) { return nil, nil })
	router.Post("/body", func(context.Context, nestedUnionBodyRequest) error { return nil })

	schemas := GenerateOpenAPI(registry).Components.Schemas

	assertPaymentUnion(t, "request body field", schemas["nestedUnionPointerBody"].Properties["method"])
	assertPaymentUnion(t, "request body", schemas["nestedOrnestedBody"])
	assertPaymentUnion(t, "response body", schemas["nestedUnionPointerResponse"])
}
