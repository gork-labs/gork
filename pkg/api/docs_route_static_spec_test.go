package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	stdlib "github.com/gork-labs/gork/pkg/adapters/stdlib"
	"github.com/gork-labs/gork/pkg/api"
)

func TestDocsRoute_WithStaticSpecFile(t *testing.T) {
	// Create a temporary OpenAPI spec file
	specContent := `{
		"openapi": "3.1.0",
		"info": {
			"title": "Static Test API",
			"version": "1.0.0"
		},
		"paths": {}
	}`
	
	tmpFile, err := os.CreateTemp("", "openapi-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(specContent); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	
	// Create router with static spec
	mux := http.NewServeMux()
	router := stdlib.NewRouter(mux)
	
	// Register docs with static spec file
	router.DocsRoute("/docs/*", api.DocsConfig{
		SpecFile: tmpFile.Name(),
		OpenAPIPath: "/openapi.json",
	})
	
	srv := httptest.NewServer(mux)
	defer srv.Close()
	
	// Request the OpenAPI spec - this should trigger the static spec return path
	resp, err := http.Get(srv.URL + "/openapi.json")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	
	// Verify we got the static spec
	var spec map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		t.Fatalf("failed to decode spec: %v", err)
	}
	
	// Check that it's our static spec
	if info, ok := spec["info"].(map[string]interface{}); ok {
		if title, ok := info["title"].(string); ok {
			if title != "Static Test API" {
				t.Errorf("expected static spec title 'Static Test API', got '%s'", title)
			}
		} else {
			t.Error("expected title in info")
		}
	} else {
		t.Error("expected info in spec")
	}
}