package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	stdlibrouter "github.com/gork-labs/gork/pkg/adapters/stdlib"
	"github.com/gork-labs/gork/pkg/api"
)

// --- Unit tests -----------------------------------------------------------

type sampleRequest struct {
	Name string `json:"name" validate:"required"`
}

type sampleResponse struct {
	OK bool `json:"ok"`
}

func sampleHandler(ctx context.Context, req sampleRequest) (sampleResponse, error) {
	return sampleResponse{OK: true}, nil
}

func TestCheckDiscriminatorErrors(t *testing.T) {
	type Disc struct {
		Kind string `json:"kind" openapi:"discriminator=foo"`
	}
	// Missing field
	if errs := api.CheckDiscriminatorErrors(Disc{}); len(errs) == 0 || errs["kind"][0] != "required" {
		t.Fatalf("expected required error, got %v", errs)
	}
	// Mismatch value
	if errs := api.CheckDiscriminatorErrors(Disc{Kind: "bar"}); len(errs) == 0 || errs["kind"][0] != "discriminator" {
		t.Fatalf("expected discriminator error, got %v", errs)
	}
}

// --- Integration test with stdlib router ----------------------------------

type reqX struct {
	Name string `json:"name" validate:"required,min=2"`
}

type respX struct {
	Msg string `json:"msg"`
}

func handlerX(_ context.Context, r reqX) (respX, error) {
	return respX{Msg: "hi " + r.Name}, nil
}

func TestHTTPValidationFlow(t *testing.T) {
	mux := http.NewServeMux()
	router := stdlibrouter.NewRouter(mux)
	router.Post("/test", handlerX)

	// 1. Valid request
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(reqX{Name: "joe"})
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// 2. Invalid JSON -> 422
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("{"))
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}

	// 3. Validation error (too short name) -> 400
	rr = httptest.NewRecorder()
	body, _ = json.Marshal(reqX{Name: "a"})
	req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
