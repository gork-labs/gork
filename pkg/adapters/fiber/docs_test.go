package fiber

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gork-labs/gork/pkg/api"
)

// TestDocsHandlerConsolidated tests the documentation handler functionality
func TestDocsHandlerConsolidated(t *testing.T) {
	// Create a sample OpenAPI spec
	spec := &api.OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: api.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*api.PathItem{
			"/test": {
				Get: &api.Operation{
					Description: "Test operation",
				},
			},
		},
	}

	config := api.DocsConfig{
		Title:       "Test Documentation",
		OpenAPIPath: "/openapi.json",
	}

	handler := DocsHandler(spec, config)
	app := fiber.New()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "json_endpoint",
			path:           "/openapi.json",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "yaml_endpoint",
			path:           "/openapi.json.yaml",
			expectedStatus: http.StatusOK,
			expectedType:   "application/yaml",
		},
		{
			name:           "not_found",
			path:           "/unknown",
			expectedStatus: http.StatusNotFound,
			expectedType:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.Get(tt.path, handler)

			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test %s endpoint: %v", tt.name, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedType != "" {
				contentType := resp.Header.Get("Content-Type")
				if contentType != tt.expectedType {
					t.Errorf("Expected Content-Type '%s', got '%s'", tt.expectedType, contentType)
				}
			}
		})
	}
}

// TestDocsHandlerWithComplexSpec tests with more complex OpenAPI specifications
func TestDocsHandlerWithComplexSpec(t *testing.T) {
	// Create a more complex spec to test YAML marshaling thoroughly
	spec := &api.OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: api.Info{
			Title:   "Complex Test API",
			Version: "2.0.0",
		},
		Paths: map[string]*api.PathItem{
			"/users/{id}": {
				Get: &api.Operation{
					Description: "Get user by ID",
				},
				Post: &api.Operation{
					Description: "Update user",
				},
			},
			"/users": {
				Get: &api.Operation{
					Description: "List all users",
				},
			},
		},
	}

	config := api.DocsConfig{
		Title:       "Complex API Documentation",
		OpenAPIPath: "/spec.json",
	}

	handler := DocsHandler(spec, config)
	app := fiber.New()
	app.Get("/spec.json.yaml", handler)

	// Test YAML conversion with complex spec
	req := httptest.NewRequest("GET", "/spec.json.yaml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test complex YAML endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for complex YAML, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/yaml" {
		t.Errorf("Expected Content-Type 'application/yaml', got '%s'", contentType)
	}
}

// TestDocsHandlerWithMarshaler tests custom marshaler functionality
func TestDocsHandlerWithMarshaler(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: api.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	config := api.DocsConfig{
		OpenAPIPath: "/openapi.json",
	}

	t.Run("successful_custom_marshaler", func(t *testing.T) {
		// Create a custom marshaler that adds a comment
		customMarshaler := func(v interface{}) ([]byte, error) {
			return []byte("# Custom YAML\ntest: value"), nil
		}

		handler := DocsHandlerWithMarshaler(spec, config, customMarshaler)
		app := fiber.New()
		app.Get("/openapi.json.yaml", handler)

		req := httptest.NewRequest("GET", "/openapi.json.yaml", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test custom marshaler: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("failing_marshaler", func(t *testing.T) {
		// Create a failing marshaler to test error path
		failingMarshaler := func(v interface{}) ([]byte, error) {
			return nil, errors.New("mock YAML marshal error")
		}

		handler := DocsHandlerWithMarshaler(spec, config, failingMarshaler)
		app := fiber.New()
		app.Get("/openapi.json.yaml", handler)

		req := httptest.NewRequest("GET", "/openapi.json.yaml", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test failing marshaler: %v", err)
		}
		defer resp.Body.Close()

		// Should return 500 error due to failing marshaler
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for failing marshaler, got: %d", resp.StatusCode)
		}
	})
}

// TestDocsHandlerEdgeCases tests edge cases and configurations
func TestDocsHandlerEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		spec           *api.OpenAPISpec
		config         api.DocsConfig
		requestPath    string
		expectedStatus int
	}{
		{
			name: "minimal_spec",
			spec: &api.OpenAPISpec{
				OpenAPI: "3.1.0",
				Info: api.Info{
					Title:   "Minimal API",
					Version: "1.0.0",
				},
			},
			config: api.DocsConfig{
				OpenAPIPath: "/minimal.json",
			},
			requestPath:    "/minimal.json",
			expectedStatus: http.StatusOK,
		},
		{
			name: "different_openapi_path",
			spec: &api.OpenAPISpec{
				OpenAPI: "3.1.0",
				Info: api.Info{
					Title:   "Custom Path API",
					Version: "1.0.0",
				},
			},
			config: api.DocsConfig{
				OpenAPIPath: "/custom/spec.json",
			},
			requestPath:    "/custom/spec.json.yaml",
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty_config_title",
			spec: &api.OpenAPISpec{
				OpenAPI: "3.1.0",
				Info: api.Info{
					Title:   "No Title Config API",
					Version: "1.0.0",
				},
			},
			config: api.DocsConfig{
				OpenAPIPath: "/notitle.json",
				// Title intentionally empty
			},
			requestPath:    "/notitle.json",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := DocsHandler(tt.spec, tt.config)
			app := fiber.New()
			app.Get(tt.requestPath, handler)

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test %s: %v", tt.name, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// TestDocsHandlerIntegration tests integration with router DocsRoute
func TestDocsHandlerIntegration(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	// Test DocsRoute registration - should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DocsRoute should not panic: %v", r)
		}
	}()

	router.DocsRoute("/docs/*")
	router.DocsRoute("/api-docs/*", api.DocsConfig{
		Title: "Custom API Documentation",
	})

	// DocsRoute should not add routes to the metadata registry
	// as they are infrastructure routes, not business API routes
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	// Registry should be empty since we haven't registered any business API routes
	if len(routes) != 0 {
		t.Errorf("Expected no routes in registry after DocsRoute calls, got %d", len(routes))
	}
}
