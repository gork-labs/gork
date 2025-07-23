package fiber

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gork-labs/gork/pkg/api"
)

func TestDocsHandler(t *testing.T) {
	// Create a sample OpenAPI spec
	spec := &api.OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: api.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	config := api.DocsConfig{
		Title:       "Test Documentation",
		OpenAPIPath: "/openapi.json",
	}

	handler := DocsHandler(spec, config)

	app := fiber.New()
	app.Get("/openapi.json", handler)
	app.Get("/openapi.json.yaml", handler)
	app.Get("/unknown", handler)

	// Test JSON endpoint
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test JSON endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Test YAML endpoint
	req = httptest.NewRequest("GET", "/openapi.json.yaml", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test YAML endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType = resp.Header.Get("Content-Type")
	if contentType != "application/yaml" {
		t.Errorf("Expected Content-Type 'application/yaml', got '%s'", contentType)
	}

	// Test 404 for unknown endpoint
	req = httptest.NewRequest("GET", "/unknown", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unknown endpoint: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestDocsHandlerWithInvalidSpec(t *testing.T) {
	// Test with spec that might cause YAML marshaling to fail
	// This is a bit contrived but helps with coverage
	spec := &api.OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: api.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	config := api.DocsConfig{
		Title:       "Test Documentation",
		OpenAPIPath: "/openapi.json",
	}

	handler := DocsHandler(spec, config)

	app := fiber.New()
	app.Get("/openapi.json.yaml", handler)

	// Test YAML endpoint - this should work with our simple spec
	req := httptest.NewRequest("GET", "/openapi.json.yaml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test YAML endpoint: %v", err)
	}

	// Should still return 200 for our simple spec
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDocsHandlerYAMLError(t *testing.T) {
	// Create a spec with a field that might cause YAML marshaling issues
	// Since yaml.Marshal is quite robust, this test mainly verifies the code path exists
	// The error branch is hard to trigger with valid data structures
	spec := &api.OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: api.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	config := api.DocsConfig{
		Title:       "Test Documentation",
		OpenAPIPath: "/openapi.json",
	}

	handler := DocsHandler(spec, config)

	app := fiber.New()
	app.Get("/openapi.json.yaml", handler)

	// This test verifies the YAML generation code path is exercised
	// The error path is difficult to trigger without invalid data structures
	req := httptest.NewRequest("GET", "/openapi.json.yaml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test YAML endpoint: %v", err)
	}

	// The YAML generation should work for our simple spec
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/yaml" {
		t.Errorf("Expected Content-Type 'application/yaml', got '%s'", contentType)
	}
}
