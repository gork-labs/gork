package api_test

import (
	"context"
	"net/http"
	"testing"

	stdlibrouter "github.com/gork-labs/gork/pkg/adapters/stdlib"
	"github.com/gork-labs/gork/pkg/api"
)

type testReq struct {
	Path struct {
		ID string `gork:"id"`
	}
}

type testResp struct {
	Body struct {
		OK bool `gork:"ok"`
	}
}

func getUser(ctx context.Context, r testReq) (*testResp, error) {
	return &testResp{
		Body: struct {
			OK bool `gork:"ok"`
		}{
			OK: true,
		},
	}, nil
}

func TestRegistryAndOpenAPI(t *testing.T) {
	mux := http.NewServeMux()
	router := stdlibrouter.NewRouter(mux)
	router.Get("/users/{id}", getUser)

	spec := api.GenerateOpenAPI(router.GetRegistry())
	if spec.Paths["/users/{id}"].Get == nil {
		t.Fatalf("expected GET operation registered")
	}
	if spec.Paths["/users/{id}"].Get.OperationID != "getUser" {
		t.Fatalf("unexpected operation id: %s", spec.Paths["/users/{id}"].Get.OperationID)
	}
}
