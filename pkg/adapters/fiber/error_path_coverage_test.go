package fiber

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gork-labs/gork/pkg/api"
)

// Test createHTTPRequestFromFiberWithCreator error path
func TestCreateHTTPRequestFromFiberWithCreatorError(t *testing.T) {
	app := fiber.New()
	
	// Create a failing HTTP request creator
	failingCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("mock HTTP request creation error")
	}
	
	app.Post("/test", func(c *fiber.Ctx) error {
		req, err := createHTTPRequestFromFiberWithCreator(c, failingCreator)
		if err == nil {
			t.Error("Expected error from failing creator")
			return c.SendStatus(fiber.StatusOK)
		}
		if req != nil {
			t.Error("Expected nil request when creator fails")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	
	// Make test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
}

// Test handleFiberRequestWithCreator error path
func TestHandleFiberRequestWithCreatorError(t *testing.T) {
	app := fiber.New()
	
	// Create a failing HTTP request creator
	failingCreator := func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("mock HTTP request creation error")
	}
	
	app.Post("/test", func(c *fiber.Ctx) error {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		
		err := handleFiberRequestWithCreator(c, handler, failingCreator)
		// The error should be handled internally by returning fiber error response
		if err != nil {
			t.Errorf("Expected no error from handleFiberRequestWithCreator, got: %v", err)
		}
		
		return nil
	})
	
	// Make test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	
	// Should return 500 error due to failing creator
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got: %d", resp.StatusCode)
	}
}

// Test createHTTPRequestFromFiber normal path (ensuring we still test the wrapper)
func TestCreateHTTPRequestFromFiberNormalPath(t *testing.T) {
	app := fiber.New()
	
	app.Post("/test", func(c *fiber.Ctx) error {
		req, err := createHTTPRequestFromFiber(c)
		if err != nil {
			t.Errorf("Expected no error from normal createHTTPRequestFromFiber, got: %v", err)
		}
		if req == nil {
			t.Error("Expected non-nil request from normal path")
		}
		if req.Method != "POST" {
			t.Errorf("Expected POST method, got: %s", req.Method)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	
	// Make test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
}

// Test handleFiberRequest normal path (ensuring we still test the wrapper)
func TestHandleFiberRequestNormalPath(t *testing.T) {
	app := fiber.New()
	
	app.Post("/test", func(c *fiber.Ctx) error {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		
		err := handleFiberRequest(c, handler)
		if err != nil {
			t.Errorf("Expected no error from normal handleFiberRequest, got: %v", err)
		}
		
		return nil
	})
	
	// Make test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}
}

// Test createRegisterFn coverage (test the returned function execution)
func TestCreateRegisterFnExecution(t *testing.T) {
	app := fiber.New()
	group := app.Group("/api")
	
	registerFn := createRegisterFn(group, "/api")
	
	// Test using the returned function
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})
	
	// Register a route using the function
	registerFn("GET", "/test", handler, nil)
	
	// Test the registered route
	// The createRegisterFn uses the group path, so the route should be at /api/api/test
	req := httptest.NewRequest("GET", "/api/api/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test registered route: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}
}

// Test NewRouter with nil app coverage
func TestNewRouterNilAppCoverage(t *testing.T) {
	// This should create a new app when nil is passed
	router := NewRouter(nil)
	
	if router == nil {
		t.Fatal("Expected router to be created")
	}
	
	if router.app == nil {
		t.Error("Expected app to be created when nil is passed")
	}
	
	// Test that the router works
	app := router.Unwrap()
	if app == nil {
		t.Error("Expected non-nil app from Unwrap")
	}
}

// Test DocsHandler error path for YAML marshaling
func TestDocsHandlerYAMLErrorPath(t *testing.T) {
	// Create an OpenAPISpec
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
	
	// Create a failing YAML marshaler
	failingMarshaler := func(interface{}) ([]byte, error) {
		return nil, errors.New("mock YAML marshal error")
	}
	
	handler := DocsHandlerWithMarshaler(spec, config, failingMarshaler)
	
	app := fiber.New()
	app.Get("/openapi.json.yaml", handler)
	
	// Test YAML endpoint - should return 500 error due to failing marshaler
	req := httptest.NewRequest("GET", "/openapi.json.yaml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test YAML endpoint: %v", err)
	}
	
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for failing YAML marshal, got: %d", resp.StatusCode)
	}
}

// Test NewRouter registerFn error path by testing through a router
func TestNewRouterRegisterFnErrorPath(t *testing.T) {
	// Create a router which uses the refactored registerFn
	router := NewRouter(nil)
	
	// Create a proper handler with the expected signature
	handler := func(ctx context.Context, req struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	}
	
	// Register a route that will trigger the error path we're testing
	router.Post("/test", handler)
	
	app := router.Unwrap()
	
	// Now we need to test that error path in handleFiberRequest - 
	// we'll override the defaultHTTPRequestCreator temporarily for this test
	originalCreator := defaultHTTPRequestCreator
	defer func() {
		defaultHTTPRequestCreator = originalCreator
	}()
	
	// Set a failing creator
	defaultHTTPRequestCreator = func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("mock HTTP request creation error in NewRouter")
	}
	
	// Make test request - this should trigger the error path
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	
	// Should return 500 error due to failing HTTP request creation
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 due to failing HTTP request creation, got: %d", resp.StatusCode)
	}
}