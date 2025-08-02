package fiber

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gork-labs/gork/pkg/api"
)

// Test Group function comprehensively to improve coverage
func TestGroup_Coverage(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)
	
	// Create a group - this tests the Group function
	group := router.Group("/api")
	
	if group == nil {
		t.Fatal("Group returned nil")
	}
	
	if group.prefix != "/api" {
		t.Errorf("Expected prefix '/api', got '%s'", group.prefix)
	}
	
	// Test nested group
	v1Group := group.Group("/v1")
	if v1Group.prefix != "/api/v1" {
		t.Errorf("Expected nested prefix '/api/v1', got '%s'", v1Group.prefix)
	}
	
	// Test that group shares registry
	if group.GetRegistry() != router.GetRegistry() {
		t.Error("Group should share registry with parent")
	}
}

// Test Group registerFn by registering a handler
func TestGroup_RegisterFn(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)
	group := router.Group("/api")
	
	// Define test types
	type TestRequest struct {
		Name string `json:"name"`
	}
	
	type TestResponse struct {
		Message string `json:"message"`
	}
	
	// Define handler
	handler := func(ctx context.Context, req TestRequest) (*TestResponse, error) {
		return &TestResponse{Message: "Hello " + req.Name}, nil
	}
	
	// Register handler using the group - this exercises the registerFn
	group.Register("POST", "/test", handler)
	
	// Verify registration
	routes := group.GetRegistry().GetRoutes()
	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}
	
	route := routes[0]
	if route.Method != "POST" {
		t.Errorf("Expected method POST, got %s", route.Method)
	}
	if route.Path != "/api/test" {
		t.Errorf("Expected path '/api/test', got %s", route.Path)
	}
}

// Test DocsHandler different code paths
func TestDocsHandler_AllPaths(t *testing.T) {
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
	
	handler := DocsHandler(spec, config)
	
	app := fiber.New()
	app.Get("/openapi.json", handler)
	app.Get("/openapi.json.yaml", handler)
	app.Get("/notfound", handler)
	
	// Test JSON path
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test JSON path: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for JSON path, got %d", resp.StatusCode)
	}
	
	// Test YAML path
	req = httptest.NewRequest(http.MethodGet, "/openapi.json.yaml", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test YAML path: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for YAML path, got %d", resp.StatusCode)
	}
	
	// Test not found path (default case)
	req = httptest.NewRequest(http.MethodGet, "/notfound", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test not found path: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for not found path, got %d", resp.StatusCode)
	}
}

// Test DocsHandler YAML marshal error path
func TestDocsHandler_YAMLError(t *testing.T) {
	// Create a problematic spec that could cause YAML marshaling issues
	// We'll use a spec with circular references or complex data
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
		OpenAPIPath: "/spec.json",
	}
	
	handler := DocsHandler(spec, config)
	
	app := fiber.New()
	app.Get("/spec.json.yaml", handler)
	
	// Test YAML conversion - should work fine for valid spec
	req := httptest.NewRequest(http.MethodGet, "/spec.json.yaml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test YAML conversion: %v", err)
	}
	
	// Should succeed for valid spec
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for valid YAML conversion, got %d", resp.StatusCode)
	}
}

// Test NewRouter with nil app to cover that branch
func TestNewRouter_NilApp(t *testing.T) {
	router := NewRouter(nil)
	
	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
	
	if router.Unwrap() == nil {
		t.Error("Router should have created new fiber app")
	}
}

// Test NewRouter with middleware options
func TestNewRouter_WithMiddleware(t *testing.T) {
	app := fiber.New()
	
	// Create options
	opt1 := api.WithTags("tag1")
	opt2 := api.WithTags("tag2")
	
	router := NewRouter(app, opt1, opt2)
	
	if router == nil {
		t.Fatal("NewRouter with options returned nil")
	}
	
	if len(router.middleware) != 2 {
		t.Errorf("Expected 2 middleware options, got %d", len(router.middleware))
	}
}