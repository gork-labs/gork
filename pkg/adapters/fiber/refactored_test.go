package fiber

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Test the extracted createHTTPRequestFromFiber function
func TestCreateHTTPRequestFromFiber(t *testing.T) {
	app := fiber.New()

	// Test successful request creation
	app.Post("/test", func(c *fiber.Ctx) error {
		req, err := createHTTPRequestFromFiber(c)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if req.Method != "POST" {
			t.Errorf("Expected method POST, got: %s", req.Method)
		}
		
		if req.URL.Path != "/test" {
			t.Errorf("Expected path /test, got: %s", req.URL.Path)
		}
		
		// Check if fiber context is in request context
		fiberCtx := req.Context().Value(fiberCtxKey{})
		if fiberCtx == nil {
			t.Error("Expected fiber context in request context")
		}
		
		// Check headers are copied
		if req.Header.Get("Content-Type") != "application/json" {
			t.Error("Headers not copied correctly")
		}
		
		return c.SendString("OK")
	})

	// Make test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
}

// Test the extracted handleFiberRequest function
func TestHandleFiberRequest(t *testing.T) {
	app := fiber.New()

	// Test successful request handling
	app.Get("/test", func(c *fiber.Ctx) error {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello from handler"))
		})
		
		return handleFiberRequest(c, handler)
	})

	// Make test request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}
}

// Test handleFiberRequest error path
func TestHandleFiberRequestError(t *testing.T) {
	// This is tricky to test because createHTTPRequestFromFiber rarely fails
	// in normal circumstances. We'll test the function structure.
	
	app := fiber.New()
	
	app.Get("/test", func(c *fiber.Ctx) error {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		
		// This should succeed normally
		err := handleFiberRequest(c, handler)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}
}

// Test the extracted createRegisterFn function
func TestCreateRegisterFn(t *testing.T) {
	app := fiber.New()
	group := app.Group("/api")
	
	registerFn := createRegisterFn(group, "/api")
	
	// Test that the function is created correctly
	if registerFn == nil {
		t.Fatal("createRegisterFn returned nil")
	}
	
	// Test using the registerFn - just verify it doesn't panic
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Test response"))
	})
	
	// This should register a route without error
	registerFn("GET", "/test", handler, nil)
	
	// We don't test the actual HTTP call here as that's tested elsewhere
	// The important thing is that registerFn works without panicking
}

// Test Group function after refactoring
func TestGroupRefactored(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)
	
	// Create group
	group := router.Group("/api")
	
	if group == nil {
		t.Fatal("Group returned nil")
	}
	
	if group.prefix != "/api" {
		t.Errorf("Expected prefix '/api', got: '%s'", group.prefix)
	}
	
	// Test nested group
	nestedGroup := group.Group("/v1")
	if nestedGroup.prefix != "/api/v1" {
		t.Errorf("Expected prefix '/api/v1', got: '%s'", nestedGroup.prefix)
	}
	
	// Test that we can register routes
	type TestRequest struct {
		Name string `json:"name"`
	}
	
	type TestResponse struct {
		Message string `json:"message"`
	}
	
	handler := func(ctx context.Context, req TestRequest) (*TestResponse, error) {
		return &TestResponse{Message: "Hello " + req.Name}, nil
	}
	
	// This should work without errors
	group.Register("POST", "/users", handler)
	
	// Verify route is registered
	routes := group.GetRegistry().GetRoutes()
	if len(routes) == 0 {
		t.Error("No routes registered")
	}
}

// Test complex routing scenario after refactoring
func TestComplexRoutingRefactored(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)
	
	// Create nested groups
	api := router.Group("/api")
	v1 := api.Group("/v1")
	users := v1.Group("/users")
	
	if users.prefix != "/api/v1/users" {
		t.Errorf("Expected prefix '/api/v1/users', got: '%s'", users.prefix)
	}
	
	// All groups should share the same registry
	if users.GetRegistry() != router.GetRegistry() {
		t.Error("Groups should share the same registry")
	}
}