package stdlib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
)

// Test helper functions
func createTestHandler() func(ctx context.Context, req struct {
	Name string `json:"name"`
}) (struct {
	Message string `json:"message"`
}, error) {
	return func(ctx context.Context, req struct {
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
}

func TestNewRouter(t *testing.T) {
	tests := []struct {
		name         string
		mux          *http.ServeMux
		opts         []api.Option
		expectNilMux bool
	}{
		{
			name:         "with provided mux",
			mux:          http.NewServeMux(),
			opts:         nil,
			expectNilMux: false,
		},
		{
			name:         "with nil mux (should create new)",
			mux:          nil,
			opts:         nil,
			expectNilMux: false,
		},
		{
			name:         "with options",
			mux:          http.NewServeMux(),
			opts:         []api.Option{api.WithTags("test")},
			expectNilMux: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.mux, tt.opts...)

			if router == nil {
				t.Fatal("NewRouter returned nil")
			}

			if tt.expectNilMux && router.mux != nil {
				t.Error("Expected nil mux but got non-nil")
			} else if !tt.expectNilMux && router.mux == nil {
				t.Error("Expected non-nil mux but got nil")
			}

			if tt.mux != nil && router.mux != tt.mux {
				t.Error("Router mux instance doesn't match provided instance")
			}

			if router.GetRegistry() == nil {
				t.Error("Registry is nil")
			}
		})
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

func TestStdlibParamAdapter(t *testing.T) {
	adapter := stdlibParamAdapter{}

	t.Run("Path", func(t *testing.T) {
		t.Run("with path values", func(t *testing.T) {
			mux := http.NewServeMux()
			// Test with stdlib path values (Go 1.22+)
			mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
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

			req := httptest.NewRequest("GET", "/users/123", nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Request failed with status %d", rec.Code)
			}
		})

		t.Run("without path values", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users/123", nil)
			value, ok := adapter.Path(req, "id")
			if ok {
				t.Error("Path() returned true without path values")
			}
			if value != "" {
				t.Errorf("Path() returned %q without path values, want empty string", value)
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

func TestRouterHTTPMethods(t *testing.T) {
	router := NewRouter(nil)
	handler := createTestHandler()

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

func TestRouterGroup(t *testing.T) {
	router := NewRouter(nil)

	t.Run("basic group creation", func(t *testing.T) {
		groupRouter := router.Group("/api/v1")
		if groupRouter == nil {
			t.Fatal("Group() returned nil")
		}

		if groupRouter.prefix != "/api/v1" {
			t.Errorf("Group router prefix = %s, want /api/v1", groupRouter.prefix)
		}

		// Test that group router shares same registry and mux
		if groupRouter.GetRegistry() != router.GetRegistry() {
			t.Error("Group router does not share the same registry")
		}
		if groupRouter.mux != router.mux {
			t.Error("Group router does not share the same mux")
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
		registries := []interface{}{
			apiGroup.GetRegistry(),
			v1Group.GetRegistry(),
			usersGroup.GetRegistry(),
		}
		for i, reg := range registries {
			if reg != router.GetRegistry() {
				t.Errorf("Group %d does not share registry", i)
			}
		}
	})

	t.Run("route registration with group prefix", func(t *testing.T) {
		apiGroup := router.Group("/api")
		handler := createTestHandler()

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

func TestRouterRegister(t *testing.T) {
	router := NewRouter(nil)
	handler := createTestHandler()

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
}

func TestRouterDocsRoute(t *testing.T) {
	router := NewRouter(nil)

	// Test DocsRoute method - it should delegate to the underlying TypedRouter
	// and should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DocsRoute should not panic: %v", r)
		}
	}()

	router.DocsRoute("/docs/*")

	// DocsRoute should not add routes to the metadata registry
	// as they are infrastructure routes, not business API routes
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	// Registry should be empty since we haven't registered any business API routes
	if len(routes) != 0 {
		t.Errorf("Expected no routes in registry after DocsRoute calls, got %d", len(routes))
	}
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
