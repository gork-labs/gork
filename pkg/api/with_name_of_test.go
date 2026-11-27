package api

import (
	"context"
	"testing"
)

type nameOfHandlers struct{}

type nameOfRequest struct {
	Path struct {
		Kind string `gork:"kind"`
	}
}

func (nameOfHandlers) SetSwitch(context.Context, nameOfRequest) (string, error) { return "", nil }

func (nameOfHandlers) SetMode(context.Context, nameOfRequest) (string, error) { return "", nil }

func registerWithRecord[Req any](method func(string, interface{}, ...Option), path string,
	handler func(context.Context, Req) (string, error), opts ...Option) {
	method(path, func(ctx context.Context, req Req) error {
		_, err := handler(ctx, req)
		return err
	}, opts...)
}

const nameOfSource = `package api

// SetSwitch turns a global switch on or off.
func (nameOfHandlers) SetSwitch() {}

// SetMode sets the mode.
func (nameOfHandlers) SetMode() {}
`

func TestWithNameOfGenericWrapper(t *testing.T) {
	extractor := parseFixtures(t, map[string]string{"pkg/api/handlers.go": nameOfSource})
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	h := nameOfHandlers{}

	registerWithRecord(router.Put, "/switches/{kind}", h.SetSwitch, WithNameOf(h.SetSwitch))
	registerWithRecord(router.Put, "/modes/{kind}", h.SetMode, WithNameOf(h.SetMode))

	paths := GenerateOpenAPIWithDocs(registry, extractor).Paths
	tests := []struct{ path, id, description string }{
		{"/switches/{kind}", "SetSwitch", "SetSwitch turns a global switch on or off."},
		{"/modes/{kind}", "SetMode", "SetMode sets the mode."},
	}
	for _, tc := range tests {
		op := paths[tc.path].Put
		if op.OperationID != tc.id {
			t.Errorf("%s: operationId = %q, want %q", tc.path, op.OperationID, tc.id)
		}
		if op.Description != tc.description {
			t.Errorf("%s: description = %q, want %q", tc.path, op.Description, tc.description)
		}
	}
}

func TestWithNameOfKeepsRegisteredHandlerTypes(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	h := nameOfHandlers{}

	registerWithRecord(router.Put, "/switches/{kind}", h.SetSwitch, WithNameOf(h.SetSwitch))

	routes := registry.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(routes))
	}
	if routes[0].RequestType.Name() != "nameOfRequest" {
		t.Errorf("request type = %v, want nameOfRequest", routes[0].RequestType)
	}
}

func TestWithoutNameOfUsesRegisteredHandler(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)

	registerWithRecord(router.Put, "/switches/{kind}", nameOfHandlers{}.SetSwitch)

	op := GenerateOpenAPI(registry).Paths["/switches/{kind}"].Put
	if op.OperationID != "func1" {
		t.Errorf("operationId = %q, want func1", op.OperationID)
	}
}
