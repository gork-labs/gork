package api_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
)

type operationIDRequest struct {
	Path struct {
		ID string `gork:"id"`
	}
}

func wrapHandler[Req any](router *api.TypedRouter[*struct{}], path string, handler func(context.Context, Req) error) {
	router.Get(path, func(ctx context.Context, req Req) error {
		return handler(ctx, req)
	})
}

func newOperationIDRouter() (*api.RouteRegistry, *api.TypedRouter[*struct{}]) {
	registry := api.NewRouteRegistry()
	router := api.NewTypedRouter[*struct{}](nil, registry, "", nil, &api.HTTPParameterAdapter{}, nil)
	return registry, &router
}

func firstOperationIDHandler(context.Context, operationIDRequest) error  { return nil }
func secondOperationIDHandler(context.Context, operationIDRequest) error { return nil }

func TestDuplicateOperationIDPanics(t *testing.T) {
	registry, router := newOperationIDRouter()
	wrapHandler(router, "/first/{id}", firstOperationIDHandler)
	wrapHandler(router, "/second/{id}", secondOperationIDHandler)

	defer func() {
		msg := fmt.Sprint(recover())
		for _, want := range []string{`"func1"`, "GET /first/{id}", "GET /second/{id}", "rename one of the handlers"} {
			if !strings.Contains(msg, want) {
				t.Errorf("panic = %q, want it to contain %q", msg, want)
			}
		}
	}()
	api.GenerateOpenAPI(registry)
	t.Error("GenerateOpenAPI did not panic")
}

func TestDuplicateOperationIDOfAFilteredRouteDoesNotPanic(t *testing.T) {
	registry, router := newOperationIDRouter()
	wrapHandler(router, "/first/{id}", firstOperationIDHandler)
	wrapHandler(router, "/second/{id}", secondOperationIDHandler)

	spec := api.GenerateOpenAPI(registry, api.WithRouteFilter(func(info *api.RouteInfo) bool {
		return info.Path == "/first/{id}"
	}))

	if spec.Paths["/first/{id}"] == nil || spec.Paths["/second/{id}"] != nil {
		t.Errorf("paths = %v, want only /first/{id}", spec.Paths)
	}
}

func TestEmptyOperationIDsDoNotCollide(t *testing.T) {
	registry := api.NewRouteRegistry()
	registry.Register(&api.RouteInfo{Method: "GET", Path: "/first", RequestType: reflect.TypeOf(struct{}{})})
	registry.Register(&api.RouteInfo{Method: "GET", Path: "/second", RequestType: reflect.TypeOf(struct{}{})})

	spec := api.GenerateOpenAPI(registry)

	if len(spec.Paths) != 2 {
		t.Errorf("paths = %v, want 2 paths", spec.Paths)
	}
}
