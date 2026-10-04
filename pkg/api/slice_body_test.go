package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type sliceBodyUser struct {
	Name string `gork:"name" validate:"required"`
}

type sliceBodyRequest struct {
	Body []sliceBodyUser
}

type sliceBodyResponse struct {
	Body []sliceBodyUser
}

type sliceBodyRawRequest struct {
	Body []byte
}

func echoSliceBody(_ context.Context, req sliceBodyRequest) (*sliceBodyResponse, error) {
	return &sliceBodyResponse{Body: req.Body}, nil
}

func TestSliceBodySchemaIsArray(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Post("/users", echoSliceBody)
	router.Post("/raw", func(context.Context, sliceBodyRawRequest) error { return nil })

	spec := GenerateOpenAPI(registry)
	op := spec.Paths["/users"].Post
	want := &Schema{Type: "array", Title: "[]sliceBodyUser", Description: "Array of sliceBodyUser", Items: &Schema{Ref: "#/components/schemas/sliceBodyUser"}}

	if got := op.RequestBody.Content["application/json"].Schema; !reflect.DeepEqual(got, want) {
		t.Errorf("request schema = %+v, want %+v", got, want)
	}
	if got := op.Responses["200"].Content["application/json"].Schema; !reflect.DeepEqual(got, want) {
		t.Errorf("response schema = %+v, want %+v", got, want)
	}
	if spec.Paths["/raw"].Post.RequestBody != nil {
		t.Error("expected no JSON request body for a []byte Body")
	}
}

func TestSliceBodyRoundTrip(t *testing.T) {
	httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, echoSliceBody)

	rec := httptest.NewRecorder()
	httpHandler(rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`[{"name":"a"},{"name":"b"}]`)))
	if want := `[{"name":"a"},{"name":"b"}]`; rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("response = %d %s, want 200 %s", rec.Code, rec.Body.String(), want)
	}

	rec = httptest.NewRecorder()
	httpHandler(rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`[{"name":""}]`)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "body.name") {
		t.Errorf("response = %d %s, want 400 for body.name", rec.Code, rec.Body.String())
	}
}
