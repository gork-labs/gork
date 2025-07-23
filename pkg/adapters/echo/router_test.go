package echo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/labstack/echo/v4"
)

func TestNewRouter(t *testing.T) {
	e := echo.New()
	router := NewRouter(e)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if router.Unwrap() != e {
		t.Error("Router echo instance doesn't match provided instance")
	}

	if router.GetRegistry() == nil {
		t.Error("Registry is nil")
	}
}

func TestNewRouterWithOptions(t *testing.T) {
	e := echo.New()

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

func TestRouterUnwrap(t *testing.T) {
	e := echo.New()
	router := NewRouter(e)

	unwrapped := router.Unwrap()
	if unwrapped != e {
		t.Error("Unwrap() did not return the original Echo instance")
	}
}

func TestRouterGetRegistry(t *testing.T) {
	e := echo.New()
	router := NewRouter(e)

	registry := router.GetRegistry()
	if registry == nil {
		t.Error("GetRegistry() returned nil")
	}

	if registry != router.registry {
		t.Error("GetRegistry() returned different registry")
	}
}

func TestRouterGroup(t *testing.T) {
	e := echo.New()
	router := NewRouter(e)

	// Test creating a group
	groupRouter := router.Group("/api/v1")
	if groupRouter == nil {
		t.Fatal("Group() returned nil")
	}

	if groupRouter.group == nil {
		t.Error("Group router has nil group")
	}

	// Test that group router has registry
	if groupRouter.GetRegistry() == nil {
		t.Error("Group router has nil registry")
	}

	// Test unwrapping group router
	unwrapped := groupRouter.Unwrap()
	if unwrapped != e {
		t.Error("Group router Unwrap() did not return original Echo instance")
	}
}

func TestEchoParamAdapter_Path(t *testing.T) {
	e := echo.New()
	adapter := echoParamAdapter{}

	// Test with Echo context in request
	e.GET("/users/:id", func(c echo.Context) error {
		// Create request with Echo context
		req := c.Request()
		req = req.WithContext(context.WithValue(req.Context(), echoCtxKey{}, c))

		// Set path parameter
		c.SetParamNames("id")
		c.SetParamValues("123")

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

		return nil
	})

	// Make a test request
	req := httptest.NewRequest("GET", "/users/123", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
}

func TestEchoParamAdapter_PathWithoutContext(t *testing.T) {
	adapter := echoParamAdapter{}
	req := httptest.NewRequest("GET", "/users/123", nil)

	// Test without Echo context
	value, ok := adapter.Path(req, "id")
	if ok {
		t.Error("Path() returned true without Echo context")
	}
	if value != "" {
		t.Errorf("Path() returned %q without Echo context, want empty string", value)
	}
}

func TestEchoParamAdapter_Query(t *testing.T) {
	adapter := echoParamAdapter{}
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

func TestEchoParamAdapter_Header(t *testing.T) {
	adapter := echoParamAdapter{}
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

func TestEchoParamAdapter_Cookie(t *testing.T) {
	adapter := echoParamAdapter{}
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

func TestRouterIntegration(t *testing.T) {
	e := echo.New()
	router := NewRouter(e)

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

func TestRouterHTTPMethods(t *testing.T) {
	router := NewRouter(echo.New())

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

	// Test all HTTP methods
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

	if len(routes) < len(expectedMethods) {
		t.Errorf("Expected at least %d routes, got %d", len(expectedMethods), len(routes))
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

func TestRouterGroupWithRouteRegistration(t *testing.T) {
	router := NewRouter(echo.New())

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

func TestRouterNestedGroups(t *testing.T) {
	router := NewRouter(echo.New())

	// Create nested groups
	apiGroup := router.Group("/api")
	v1Group := apiGroup.Group("/v1")
	usersGroup := v1Group.Group("/users")

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

	// Register route in the nested group
	usersGroup.Post("/create", handler)

	// Verify the route was registered with the correct nested path
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	found := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/v1/users/create" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Route was not registered with correct nested group prefix")
	}
}
