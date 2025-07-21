package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	stdlib "github.com/gork-labs/gork/pkg/adapters/stdlib"
)

func TestDocsRoute_ServesOpenAPISpecAndUI(t *testing.T) {
	mux := http.NewServeMux()
	router := stdlib.NewRouter(mux)

	// Register docs
	router.DocsRoute("/docs/*")

	for _, rt := range router.GetRegistry().GetRoutes() {
		t.Logf("registered initially: %s %s", rt.Method, rt.Path)
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 1) OpenAPI JSON
	resp, err := http.Get(srv.URL + "/docs/openapi.json")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		routes := router.GetRegistry().GetRoutes()
		for _, r := range routes {
			t.Logf("registered: %s %s", r.Method, r.Path)
		}
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var doc struct {
		OpenAPI string `json:"openapi"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if doc.OpenAPI == "" {
		t.Fatalf("openapi field empty in spec")
	}

	// 2) Docs UI HTML
	resp, err = http.Get(srv.URL + "/docs/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ctype := resp.Header.Get("Content-Type")
	if ctype == "" || ctype[:9] != "text/html" {
		t.Fatalf("expected text/html content type, got %s", ctype)
	}
}
