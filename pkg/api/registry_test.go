package api_test

import (
	"context"
	"testing"

	"net/http"

	stdlibrouter "github.com/gork-labs/gork/pkg/adapters/stdlib"
	"github.com/gork-labs/gork/pkg/api"
)

type testReq struct {
	ID string `json:"id" openapi:"name=id,in=path"`
}

type testResp struct {
	OK bool `json:"ok"`
}

func getUser(ctx context.Context, r testReq) (testResp, error) {
	return testResp{OK: true}, nil
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
