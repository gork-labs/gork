package api

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// TestDocsRouteComprehensive tests all paths in DocsRoute
func TestDocsRouteComprehensive(t *testing.T) {
	t.Run("DocsRoute with absolute OpenAPIPath", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
			registerFn: func(method, path string, handler http.HandlerFunc, info *RouteInfo) {
				// Mock register function
			},
		}

		// Test with absolute OpenAPIPath (starts with "/")
		cfg := DocsConfig{
			OpenAPIPath: "/absolute/openapi.json",
			Title:       "Custom Title",
		}

		// This should trigger lines 70-71 (absolute path handling)
		router.DocsRoute("/docs", cfg)
	})

	t.Run("DocsRoute with relative OpenAPIPath", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
			registerFn: func(method, path string, handler http.HandlerFunc, info *RouteInfo) {
				// Mock register function
			},
		}

		// Test with relative OpenAPIPath (doesn't start with "/")
		cfg := DocsConfig{
			OpenAPIPath: "relative/openapi.json",
			Title:       "Custom Title",
		}

		// This should trigger lines 72-74 (relative path handling)
		router.DocsRoute("/docs", cfg)
	})

	t.Run("openAPIHandler with nil staticSpec", func(t *testing.T) {
		registry := NewRouteRegistry()

		// Add a test route to the registry so GenerateOpenAPI has something to process
		type TestReq struct {
			ID string `gork:"id"`
		}
		type TestResp struct {
			Body struct {
				Message string `gork:"message"`
			}
		}

		info := &RouteInfo{
			Method:       "GET",
			Path:         "/test",
			HandlerName:  "GetTest",
			RequestType:  reflect.TypeOf(TestReq{}),
			ResponseType: reflect.TypeOf((*TestResp)(nil)),
		}
		registry.Register(info)

		router := &TypedRouter[*TestResp]{
			registry: registry,
		}

		// Test the extracted openAPIHandler method with nil staticSpec
		// This should trigger the GenerateOpenAPI(r.registry) path
		spec, err := router.openAPIHandler(nil)
		if err != nil {
			t.Errorf("openAPIHandler returned error: %v", err)
		}
		if spec == nil {
			t.Error("openAPIHandler returned nil spec")
		}
		// Verify it generated from registry (should have our test route)
		if len(spec.Paths) == 0 {
			t.Error("Generated spec should have paths from registry")
		}
	})

	t.Run("generateDocsHTML with template substitution", func(t *testing.T) {
		router := &TypedRouter[*TestResponse]{}

		cfg := DocsConfig{
			Title:       "Test API Docs",
			OpenAPIPath: "/api/openapi.json",
			UITemplate:  UITemplate("<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body><div data-openapi-url=\"{{.OpenAPIPath}}\" data-base=\"{{.BasePath}}\">Content</div></body></html>"),
		}
		basePath := "/docs"

		html := router.generateDocsHTML(basePath, cfg)

		// Verify template substitution worked
		if !strings.Contains(html, "Test API Docs") {
			t.Error("Title was not substituted in HTML")
		}
		if !strings.Contains(html, "/api/openapi.json") {
			t.Error("OpenAPIPath was not substituted in HTML")
		}
		if !strings.Contains(html, "/docs") {
			t.Error("BasePath was not substituted in HTML")
		}

		// Verify no template placeholders remain
		if strings.Contains(html, "{{.Title}}") {
			t.Error("Title placeholder was not replaced")
		}
		if strings.Contains(html, "{{.OpenAPIPath}}") {
			t.Error("OpenAPIPath placeholder was not replaced")
		}
		if strings.Contains(html, "{{.BasePath}}") {
			t.Error("BasePath placeholder was not replaced")
		}
	})

	t.Run("serveDocsHTML returns handler with proper headers and content", func(t *testing.T) {
		router := &TypedRouter[*TestResponse]{}

		// Create a test ResponseWriter
		recorder := httptest.NewRecorder()
		testHTML := "<html><head><title>Test</title></head><body>Test Content</body></html>"

		// Get the handler from serveDocsHTML
		handler := router.serveDocsHTML(testHTML)

		// Call the handler
		req := httptest.NewRequest("GET", "/docs", nil)
		handler(recorder, req)

		// Verify the response
		if recorder.Code != 200 {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		// Verify Content-Type header was set
		contentType := recorder.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("Expected Content-Type 'text/html; charset=utf-8', got '%s'", contentType)
		}

		// Verify HTML content was written
		body := recorder.Body.String()
		if body != testHTML {
			t.Errorf("Expected body '%s', got '%s'", testHTML, body)
		}
	})

	t.Run("DocsRoute with SpecFile provided", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
			registerFn: func(method, path string, handler http.HandlerFunc, info *RouteInfo) {
				// Mock register function
			},
		}

		// Test with SpecFile provided (for static spec)
		cfg := DocsConfig{
			SpecFile: "nonexistent-spec.json", // This will test LoadStaticSpec behavior
		}

		// This should trigger line 81 (LoadStaticSpec call)
		router.DocsRoute("/docs", cfg)
	})

	t.Run("DocsRoute with nil registerFn", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry:   registry,
			registerFn: nil, // This should skip the UI route registration
		}

		cfg := DocsConfig{
			Title: "No Register Test",
		}

		// This should trigger lines 87-89 (skip UI registration when registerFn is nil)
		router.DocsRoute("/docs", cfg)
	})

	t.Run("DocsRoute with no config provided", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
			registerFn: func(method, path string, handler http.HandlerFunc, info *RouteInfo) {
				// Mock register function
			},
		}

		// This should use default configuration
		router.DocsRoute("/docs")
	})

	t.Run("DocsRoute with empty config values", func(t *testing.T) {
		registry := NewRouteRegistry()
		router := &TypedRouter[*TestResponse]{
			registry: registry,
			registerFn: func(method, path string, handler http.HandlerFunc, info *RouteInfo) {
				// Mock register function
			},
		}

		// Test config with empty values to trigger defaults
		cfg := DocsConfig{
			// All empty values should trigger defaults in PrepareDocsConfig
		}

		router.DocsRoute("/docs", cfg)
	})
}

// TestResponse for testing purposes
type TestResponse struct {
	Message string `json:"message"`
}
