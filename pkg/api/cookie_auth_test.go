package api

import (
	"context"
	"reflect"
	"testing"
)

func TestWithCookieAuthInSpec(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Post("/me", func(context.Context, httpErrorRequest) (*httpErrorResponse, error) { return nil, nil }, WithCookieAuth("mobius_session"))

	spec := GenerateOpenAPI(registry)

	want := &SecurityScheme{Type: "apiKey", In: "cookie", Name: "mobius_session"}
	if got := spec.Components.SecuritySchemes["mobius_session"]; !reflect.DeepEqual(got, want) {
		t.Errorf("security scheme = %+v, want %+v", got, want)
	}
	wantSecurity := []map[string][]string{{"mobius_session": {}}}
	if got := spec.Paths["/me"].Post.Security; !reflect.DeepEqual(got, wantSecurity) {
		t.Errorf("operation security = %v, want %v", got, wantSecurity)
	}
}
