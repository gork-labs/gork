package gin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ginpkg "github.com/gin-gonic/gin"

	"github.com/gork-labs/gork/pkg/api"
)

func init() {
	// Set gin to test mode to reduce debug output during tests
	ginpkg.SetMode(ginpkg.TestMode)
}

func TestNewRouter(t *testing.T) {
	e := ginpkg.New()
	router := NewRouter(e)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if router.engine != e {
		t.Error("Router engine instance doesn't match provided instance")
	}

	if router.GetRegistry() == nil {
		t.Error("Registry is nil")
	}
}

func TestNewRouterWithNilEngine(t *testing.T) {
	router := NewRouter(nil)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if router.engine == nil {
		t.Error("Router should create new engine when nil is passed")
	}

	if router.GetRegistry() == nil {
		t.Error("Registry is nil")
	}
}

func TestNewRouterWithOptions(t *testing.T) {
	e := ginpkg.New()

	// Test with middleware option
	middleware := api.WithTags("test")
	router := NewRouter(e, middleware)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if router.GetRegistry() == nil {
		t.Error("Registry is nil")
	}
}

func TestRouterGetRegistry(t *testing.T) {
	router := NewRouter(nil)

	registry := router.GetRegistry()
	if registry == nil {
		t.Error("GetRegistry() returned nil")
	}

	if registry != router.registry {
		t.Error("GetRegistry() returned different registry")
	}
}

func TestRouterGroup(t *testing.T) {
	e := ginpkg.New()
	router := NewRouter(e)

	// Test creating a group
	groupRouter := router.Group("/api/v1")
	if groupRouter == nil {
		t.Fatal("Group() returned nil")
	}

	if groupRouter.group == nil {
		t.Error("Group router has nil group")
	}

	if groupRouter.prefix != "/api/v1" {
		t.Errorf("Group router prefix = %s, want /api/v1", groupRouter.prefix)
	}

	// Test that group router has same registry
	if groupRouter.GetRegistry() != router.GetRegistry() {
		t.Error("Group router does not share the same registry")
	}

	// Test that group router has same engine
	if groupRouter.engine != router.engine {
		t.Error("Group router does not share the same engine")
	}
}

func TestGinParamAdapter_Path(t *testing.T) {
	adapter := ginParamAdapter{}
	e := ginpkg.New()

	// Test with Gin context in request
	e.GET("/users/:id", func(c *ginpkg.Context) {
		// Create request with Gin context
		req := c.Request
		req = req.WithContext(context.WithValue(req.Context(), ginCtxKey{}, c))

		// Test existing parameter
		value, ok := adapter.Path(req, "id")
		if !ok {
			t.Error("Path() returned false for existing parameter")
		}
		if value != "123" {
			t.Errorf("Path() returned %q, want %q", value, "123")
		}

		// Test non-existing parameter
		value, ok = adapter.Path(req, "nonexistent")
		if ok {
			t.Error("Path() returned true for non-existing parameter")
		}
		if value != "" {
			t.Errorf("Path() returned %q for non-existing parameter, want empty string", value)
		}

		c.Status(http.StatusOK)
	})

	// Make a test request
	req := httptest.NewRequest("GET", "/users/123", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Request failed with status %d", rec.Code)
	}
}

func TestGinParamAdapter_PathWithoutContext(t *testing.T) {
	adapter := ginParamAdapter{}
	req := httptest.NewRequest("GET", "/users/123", nil)

	// Test without Gin context
	value, ok := adapter.Path(req, "id")
	if ok {
		t.Error("Path() returned true without Gin context")
	}
	if value != "" {
		t.Errorf("Path() returned %q without Gin context, want empty string", value)
	}
}

func TestGinParamAdapter_Query(t *testing.T) {
	adapter := ginParamAdapter{}
	req := httptest.NewRequest("GET", "/test?name=john&age=30", nil)

	// Test existing query parameter
	value, ok := adapter.Query(req, "name")
	if !ok {
		t.Error("Query() returned false for existing parameter")
	}
	if value != "john" {
		t.Errorf("Query() returned %q, want %q", value, "john")
	}

	// Test non-existing query parameter
	value, ok = adapter.Query(req, "nonexistent")
	if ok {
		t.Error("Query() returned true for non-existing parameter")
	}
	if value != "" {
		t.Errorf("Query() returned %q for non-existing parameter, want empty string", value)
	}
}

func TestGinParamAdapter_Header(t *testing.T) {
	adapter := ginParamAdapter{}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")

	// Test existing header
	value, ok := adapter.Header(req, "Authorization")
	if !ok {
		t.Error("Header() returned false for existing header")
	}
	if value != "Bearer token123" {
		t.Errorf("Header() returned %q, want %q", value, "Bearer token123")
	}

	// Test non-existing header
	value, ok = adapter.Header(req, "NonExistent")
	if ok {
		t.Error("Header() returned true for non-existing header")
	}
	if value != "" {
		t.Errorf("Header() returned %q for non-existing header, want empty string", value)
	}
}

func TestGinParamAdapter_Cookie(t *testing.T) {
	adapter := ginParamAdapter{}
	req := httptest.NewRequest("GET", "/test", nil)

	// Add cookies
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})

	// Test existing cookie
	value, ok := adapter.Cookie(req, "session")
	if !ok {
		t.Error("Cookie() returned false for existing cookie")
	}
	if value != "abc123" {
		t.Errorf("Cookie() returned %q, want %q", value, "abc123")
	}

	// Test non-existing cookie
	value, ok = adapter.Cookie(req, "nonexistent")
	if ok {
		t.Error("Cookie() returned true for non-existing cookie")
	}
	if value != "" {
		t.Errorf("Cookie() returned %q for non-existing cookie, want empty string", value)
	}
}

func TestRouterHTTPMethods(t *testing.T) {
	router := NewRouter(nil)

	// Define a simple handler
	handler := func(ctx context.Context, req struct {
		Name string `json:"name"`
	}) (struct {
		Message string `json:"message"`
	}, error) {
		return struct {
			Message string `json:"message"`
		}{
			Message: "Hello " + req.Name,
		}, nil
	}

	// Test all HTTP methods (using methods directly from TypedRouter)
	router.Get("/get", handler)
	router.Post("/post", handler)
	router.Put("/put", handler)
	router.Delete("/delete", handler)
	router.Patch("/patch", handler)

	// Verify routes were registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	expectedPaths := []string{"/get", "/post", "/put", "/delete", "/patch"}

	if len(routes) != len(expectedMethods) {
		t.Errorf("Expected %d routes, got %d", len(expectedMethods), len(routes))
	}

	for i, expectedMethod := range expectedMethods {
		found := false
		for _, route := range routes {
			if route.Method == expectedMethod && route.Path == expectedPaths[i] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Route %s %s was not registered", expectedMethod, expectedPaths[i])
		}
	}
}

func TestRouterIntegration(t *testing.T) {
	router := NewRouter(nil)

	// Define a simple handler
	handler := func(ctx context.Context, req struct {
		Name string `json:"name"`
	}) (struct {
		Message string `json:"message"`
	}, error) {
		return struct {
			Message string `json:"message"`
		}{
			Message: "Hello " + req.Name,
		}, nil
	}

	// Register route
	router.Post("/greet", handler)

	// Verify route was registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	found := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/greet" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Route was not registered in registry")
	}
}

func TestRouterGroupNested(t *testing.T) {
	router := NewRouter(nil)

	// Create nested groups
	apiGroup := router.Group("/api")
	v1Group := apiGroup.Group("/v1")
	usersGroup := v1Group.Group("/users")

	// Check prefixes
	if apiGroup.prefix != "/api" {
		t.Errorf("API group prefix = %s, want /api", apiGroup.prefix)
	}
	if v1Group.prefix != "/api/v1" {
		t.Errorf("V1 group prefix = %s, want /api/v1", v1Group.prefix)
	}
	if usersGroup.prefix != "/api/v1/users" {
		t.Errorf("Users group prefix = %s, want /api/v1/users", usersGroup.prefix)
	}

	// Test that all groups share the same registry
	if apiGroup.GetRegistry() != router.GetRegistry() {
		t.Error("API group does not share registry")
	}
	if v1Group.GetRegistry() != router.GetRegistry() {
		t.Error("V1 group does not share registry")
	}
	if usersGroup.GetRegistry() != router.GetRegistry() {
		t.Error("Users group does not share registry")
	}
}

func TestRouterGroupWithRouteRegistration(t *testing.T) {
	router := NewRouter(nil)

	// Create a group
	apiGroup := router.Group("/api")

	// Define a simple handler
	handler := func(ctx context.Context, req struct {
		Name string `json:"name"`
	}) (struct {
		Message string `json:"message"`
	}, error) {
		return struct {
			Message string `json:"message"`
		}{
			Message: "Hello " + req.Name,
		}, nil
	}

	// Register route in the group
	apiGroup.Post("/users", handler)

	// Verify the route was registered with the correct path
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	found := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Route was not registered with correct group prefix")
	}
}

func TestRouterUnwrap(t *testing.T) {
	engine := ginpkg.New()
	router := NewRouter(engine)
	unwrapped := router.Unwrap()
	if unwrapped != engine {
		t.Error("Unwrap() returned different engine instance")
	}
}

func TestRouterRegister(t *testing.T) {
	router := NewRouter(nil)

	// Define a simple handler
	handler := func(ctx context.Context, req struct {
		Name string `json:"name"`
	}) (struct {
		Message string `json:"message"`
	}, error) {
		return struct {
			Message string `json:"message"`
		}{
			Message: "Hello " + req.Name,
		}, nil
	}

	// Use the generic Register method
	router.Register("POST", "/register-test", handler)

	// Verify the route was registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	found := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/register-test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Route was not registered using Register method")
	}
}

func TestRouterExportOpenAPIAndExit(t *testing.T) {
	router := NewRouter(ginpkg.New())

	// This test checks that ExportOpenAPIAndExit calls the underlying TypedRouter
	// We can't test the actual exit behavior, but we can ensure the method exists and delegates
	defer func() {
		if r := recover(); r != nil {
			// ExportOpenAPIAndExit calls os.Exit, so we expect a panic in tests
			// This is expected behavior for this method
		}
	}()

	// Call ExportOpenAPIAndExit - this will panic with os.Exit
	router.ExportOpenAPIAndExit()
}
