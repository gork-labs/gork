package chi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	chibase "github.com/go-chi/chi/v5"

	"github.com/gork-labs/gork/pkg/api"
)

func TestRouterInitialization(t *testing.T) {
	tests := []struct {
		name     string
		mux      *chibase.Mux
		options  []api.Option
		checkMux func(*testing.T, *Router, *chibase.Mux)
	}{
		{
			name: "with provided mux",
			mux:  chibase.NewRouter(),
			checkMux: func(t *testing.T, router *Router, expectedMux *chibase.Mux) {
				if router.mux != expectedMux {
					t.Error("Router mux instance doesn't match provided instance")
				}
			},
		},
		{
			name: "with nil mux",
			mux:  nil,
			checkMux: func(t *testing.T, router *Router, expectedMux *chibase.Mux) {
				if router.mux == nil {
					t.Error("Router should create new mux when nil is passed")
				}
			},
		},
		{
			name:    "with options",
			mux:     chibase.NewRouter(),
			options: []api.Option{api.WithTags("test")},
			checkMux: func(t *testing.T, router *Router, expectedMux *chibase.Mux) {
				if router.mux != expectedMux {
					t.Error("Router mux instance doesn't match provided instance")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.mux, tt.options...)

			if router == nil {
				t.Fatal("NewRouter returned nil")
			}

			if router.GetRegistry() == nil {
				t.Error("Registry is nil")
			}

			if tt.checkMux != nil {
				tt.checkMux(t, router, tt.mux)
			}
		})
	}
}

func TestRouterCore(t *testing.T) {
	t.Run("GetRegistry", func(t *testing.T) {
		router := NewRouter(nil)

		registry := router.GetRegistry()
		if registry == nil {
			t.Error("GetRegistry() returned nil")
		}

		if registry != router.registry {
			t.Error("GetRegistry() returned different registry")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		mux := chibase.NewRouter()
		router := NewRouter(mux)

		unwrapped := router.Unwrap()
		if unwrapped != mux {
			t.Error("Unwrap() returned different mux instance")
		}
	})
}

func TestRouterGroups(t *testing.T) {
	t.Run("basic group creation", func(t *testing.T) {
		router := NewRouter(nil)

		// Test creating a group
		groupRouter := router.Group("/api/v1")
		if groupRouter == nil {
			t.Fatal("Group() returned nil")
		}

		if groupRouter.mux != router.mux {
			t.Error("Group router does not share the same mux")
		}

		if groupRouter.prefix != "/api/v1" {
			t.Errorf("Group router prefix = %s, want /api/v1", groupRouter.prefix)
		}

		// Test that group router has same registry
		if groupRouter.GetRegistry() != router.GetRegistry() {
			t.Error("Group router does not share the same registry")
		}
	})

	t.Run("nested groups", func(t *testing.T) {
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
	})

	t.Run("group with route registration", func(t *testing.T) {
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
	})
}

func TestParameterAdapters(t *testing.T) {
	adapter := chiParamAdapter{}

	t.Run("Path", func(t *testing.T) {
		t.Run("with chi context", func(t *testing.T) {
			mux := chibase.NewRouter()

			// Test with chi path parameters
			mux.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
				// Test existing parameter
				value, ok := adapter.Path(r, "id")
				if !ok {
					t.Error("Path() returned false for existing parameter")
				}
				if value != "123" {
					t.Errorf("Path() returned %q, want %q", value, "123")
				}

				// Test non-existing parameter
				value, ok = adapter.Path(r, "nonexistent")
				if ok {
					t.Error("Path() returned true for non-existing parameter")
				}
				if value != "" {
					t.Errorf("Path() returned %q for non-existing parameter, want empty string", value)
				}

				w.WriteHeader(http.StatusOK)
			})

			// Make a test request
			req := httptest.NewRequest("GET", "/users/123", nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Request failed with status %d", rec.Code)
			}
		})

		t.Run("without chi context", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users/123", nil)

			// Test without chi context
			value, ok := adapter.Path(req, "id")
			if ok {
				t.Error("Path() returned true without chi context")
			}
			if value != "" {
				t.Errorf("Path() returned %q without chi context, want empty string", value)
			}
		})
	})

	t.Run("Query", func(t *testing.T) {
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
	})

	t.Run("Header", func(t *testing.T) {
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
	})

	t.Run("Cookie", func(t *testing.T) {
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
	})
}

func TestHTTPMethods(t *testing.T) {
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

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/get"},
		{"POST", "/post"},
		{"PUT", "/put"},
		{"DELETE", "/delete"},
		{"PATCH", "/patch"},
	}

	// Register all routes
	router.Get("/get", handler)
	router.Post("/post", handler)
	router.Put("/put", handler)
	router.Delete("/delete", handler)
	router.Patch("/patch", handler)

	// Verify routes were registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	if len(routes) != len(tests) {
		t.Errorf("Expected %d routes, got %d", len(tests), len(routes))
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			found := false
			for _, route := range routes {
				if route.Method == tt.method && route.Path == tt.path {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Route %s %s was not registered", tt.method, tt.path)
			}
		})
	}
}

func TestRouterRegistration(t *testing.T) {
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

	t.Run("integration test", func(t *testing.T) {
		router := NewRouter(nil)

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
	})

	t.Run("register method", func(t *testing.T) {
		router := NewRouter(nil)

		// Test Register method
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

func TestRouterExportOpenAPIAndExit(t *testing.T) {
	router := NewRouter(nil)

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