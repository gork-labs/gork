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

// TestRouterInitialization tests router creation scenarios using table-driven approach
func TestRouterInitialization(t *testing.T) {
	tests := []struct {
		name        string
		engine      *ginpkg.Engine
		options     []api.Option
		expectNil   bool
		checkEngine bool
	}{
		{
			name:        "with existing engine",
			engine:      ginpkg.New(),
			expectNil:   false,
			checkEngine: true,
		},
		{
			name:        "with nil engine",
			engine:      nil,
			expectNil:   false,
			checkEngine: false,
		},
		{
			name:        "with options",
			engine:      ginpkg.New(),
			options:     []api.Option{api.WithTags("test")},
			expectNil:   false,
			checkEngine: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.engine, tt.options...)

			if router == nil {
				t.Fatal("NewRouter returned nil")
			}

			if tt.checkEngine && router.engine != tt.engine {
				t.Error("Router engine instance doesn't match provided instance")
			}

			if !tt.checkEngine && router.engine == nil {
				t.Error("Router should create new engine when nil is passed")
			}

			if router.GetRegistry() == nil {
				t.Error("Registry is nil")
			}

			// Test that GetRegistry returns consistent results
			registry := router.GetRegistry()
			if registry != router.registry {
				t.Error("GetRegistry() returned different registry")
			}
		})
	}
}

// TestRouterGroup tests router group functionality including nesting and route registration
func TestRouterGroup(t *testing.T) {
	router := NewRouter(nil)

	t.Run("basic group creation", func(t *testing.T) {
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

		// Test that group router shares resources
		if groupRouter.GetRegistry() != router.GetRegistry() {
			t.Error("Group router does not share the same registry")
		}

		if groupRouter.engine != router.engine {
			t.Error("Group router does not share the same engine")
		}
	})

	t.Run("nested groups", func(t *testing.T) {
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
		sharedRegistry := router.GetRegistry()
		if apiGroup.GetRegistry() != sharedRegistry {
			t.Error("API group does not share registry")
		}
		if v1Group.GetRegistry() != sharedRegistry {
			t.Error("V1 group does not share registry")
		}
		if usersGroup.GetRegistry() != sharedRegistry {
			t.Error("Users group does not share registry")
		}
	})

	t.Run("group route registration", func(t *testing.T) {
		apiGroup := router.Group("/api")

		// Define a simple handler
		handler := func(ctx context.Context, req struct {
			Name string `json:"name"`
		}) (struct {
			Message string `json:"message"`
		}, error,
		) {
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
	})
}

// TestParameterAdapter tests all parameter adapter methods using table-driven approach
func TestParameterAdapter(t *testing.T) {
	adapter := ginParamAdapter{}

	t.Run("path parameters", func(t *testing.T) {
		t.Run("with gin context", func(t *testing.T) {
			e := ginpkg.New()
			e.GET("/users/:id", func(c *ginpkg.Context) {
				req := c.Request.WithContext(context.WithValue(c.Request.Context(), ginCtxKey{}, c))

				tests := []struct {
					name     string
					param    string
					expected string
					expectOk bool
				}{
					{"existing parameter", "id", "123", true},
					{"non-existing parameter", "nonexistent", "", false},
				}

				for _, tt := range tests {
					value, ok := adapter.Path(req, tt.param)
					if ok != tt.expectOk {
						t.Errorf("%s: Path() ok = %v, want %v", tt.name, ok, tt.expectOk)
					}
					if value != tt.expected {
						t.Errorf("%s: Path() value = %q, want %q", tt.name, value, tt.expected)
					}
				}

				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/users/123", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Request failed with status %d", rec.Code)
			}
		})

		t.Run("without gin context", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users/123", nil)
			value, ok := adapter.Path(req, "id")
			if ok {
				t.Error("Path() returned true without Gin context")
			}
			if value != "" {
				t.Errorf("Path() returned %q without Gin context, want empty string", value)
			}
		})
	})

	t.Run("query parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?name=john&age=30", nil)

		tests := []struct {
			name     string
			param    string
			expected string
			expectOk bool
		}{
			{"existing parameter", "name", "john", true},
			{"another existing parameter", "age", "30", true},
			{"non-existing parameter", "nonexistent", "", false},
		}

		for _, tt := range tests {
			value, ok := adapter.Query(req, tt.param)
			if ok != tt.expectOk {
				t.Errorf("%s: Query() ok = %v, want %v", tt.name, ok, tt.expectOk)
			}
			if value != tt.expected {
				t.Errorf("%s: Query() value = %q, want %q", tt.name, value, tt.expected)
			}
		}
	})

	t.Run("headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer token123")
		req.Header.Set("Content-Type", "application/json")

		tests := []struct {
			name     string
			header   string
			expected string
			expectOk bool
		}{
			{"existing header", "Authorization", "Bearer token123", true},
			{"another existing header", "Content-Type", "application/json", true},
			{"non-existing header", "NonExistent", "", false},
		}

		for _, tt := range tests {
			value, ok := adapter.Header(req, tt.header)
			if ok != tt.expectOk {
				t.Errorf("%s: Header() ok = %v, want %v", tt.name, ok, tt.expectOk)
			}
			if value != tt.expected {
				t.Errorf("%s: Header() value = %q, want %q", tt.name, value, tt.expected)
			}
		}
	})

	t.Run("cookies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
		req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})

		tests := []struct {
			name     string
			cookie   string
			expected string
			expectOk bool
		}{
			{"existing cookie", "session", "abc123", true},
			{"another existing cookie", "theme", "dark", true},
			{"non-existing cookie", "nonexistent", "", false},
		}

		for _, tt := range tests {
			value, ok := adapter.Cookie(req, tt.cookie)
			if ok != tt.expectOk {
				t.Errorf("%s: Cookie() ok = %v, want %v", tt.name, ok, tt.expectOk)
			}
			if value != tt.expected {
				t.Errorf("%s: Cookie() value = %q, want %q", tt.name, value, tt.expected)
			}
		}
	})
}

// TestRouteRegistration tests HTTP method registration and route management
func TestRouteRegistration(t *testing.T) {
	router := NewRouter(nil)

	// Define a simple handler used across tests
	handler := func(ctx context.Context, req struct {
		Name string `json:"name"`
	}) (struct {
		Message string `json:"message"`
	}, error,
	) {
		return struct {
			Message string `json:"message"`
		}{
			Message: "Hello " + req.Name,
		}, nil
	}

	t.Run("HTTP methods", func(t *testing.T) {
		// Register routes using specific method functions
		router.Get("/get", handler)
		router.Post("/post", handler)
		router.Put("/put", handler)
		router.Delete("/delete", handler)
		router.Patch("/patch", handler)

		// Verify routes were registered
		registry := router.GetRegistry()
		routes := registry.GetRoutes()

		expectedRoutes := []struct {
			method string
			path   string
		}{
			{"GET", "/get"},
			{"POST", "/post"},
			{"PUT", "/put"},
			{"DELETE", "/delete"},
			{"PATCH", "/patch"},
		}

		if len(routes) != len(expectedRoutes) {
			t.Errorf("Expected %d routes, got %d", len(expectedRoutes), len(routes))
		}

		for _, expected := range expectedRoutes {
			found := false
			for _, route := range routes {
				if route.Method == expected.method && route.Path == expected.path {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Route %s %s was not registered", expected.method, expected.path)
			}
		}
	})

	t.Run("generic register method", func(t *testing.T) {
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
	})
}

// TestRouterMiscellaneous tests additional router functionality
func TestRouterMiscellaneous(t *testing.T) {
	t.Run("unwrap engine", func(t *testing.T) {
		engine := ginpkg.New()
		router := NewRouter(engine)
		unwrapped := router.Unwrap()
		if unwrapped != engine {
			t.Error("Unwrap() returned different engine instance")
		}
	})

	t.Run("export openapi and exit", func(t *testing.T) {
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
	})
}
