package api

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestWithErrorResponsesInSpec(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	handler := func(context.Context, httpErrorRequest) (*httpErrorResponse, error) { return nil, nil }
	router.Post("/login", handler, WithErrorResponses(http.StatusUnauthorized, http.StatusConflict, http.StatusBadRequest))
	router.Post("/plain", handler)

	spec := GenerateOpenAPI(registry)
	responses := spec.Paths["/login"].Post.Responses

	for status, description := range map[string]string{"401": "Unauthorized", "409": "Conflict"} {
		want := &Response{
			Description: description,
			Content: map[string]*MediaType{
				"application/json": {Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"}},
			},
		}
		if got := responses[status]; !reflect.DeepEqual(got, want) {
			t.Errorf("response %s = %+v, want %+v", status, got, want)
		}
	}
	if got := responses["400"].Ref; got != "#/components/responses/BadRequest" {
		t.Errorf("response 400 ref = %q, want the standard BadRequest response", got)
	}
	if _, ok := spec.Components.Schemas["ErrorResponse"]; !ok {
		t.Error("expected the ErrorResponse schema in components")
	}
	if _, ok := spec.Paths["/plain"].Post.Responses["401"]; ok {
		t.Error("expected no 401 response on a route without WithErrorResponses")
	}
}

func TestRouteErrorResponsesWithoutOptions(t *testing.T) {
	g := NewConventionOpenAPIGenerator(&OpenAPISpec{}, NewDocExtractor())
	op := &Operation{Responses: map[string]*Response{}}
	g.addRouteErrorResponses(op, &RouteInfo{})
	if len(op.Responses) != 0 {
		t.Errorf("responses = %v, want none", op.Responses)
	}
}
