package echo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/labstack/echo/v4"
)

func TestRouter(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		tests := []struct {
			name         string
			echoInstance *echo.Echo
			options      []api.Option
		}{
			{
				name:         "with_provided_echo_instance",
				echoInstance: echo.New(),
				options:      nil,
			},
			{
				name:         "with_nil_echo_instance",
				echoInstance: nil,
				options:      nil,
			},
			{
				name:         "with_middleware_options",
				echoInstance: echo.New(),
				options:      []api.Option{api.WithTags("test")},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				router := NewRouter(tt.echoInstance, tt.options...)

				if router == nil {
					t.Fatal("NewRouter returned nil")
				}

				if router.GetRegistry() == nil {
					t.Error("Registry is nil")
				}

				// Test unwrap functionality
				unwrapped := router.Unwrap()
				if unwrapped == nil {
					t.Error("Unwrap() returned nil")
				}

				// Verify registry consistency
				if registry := router.GetRegistry(); registry != router.registry {
					t.Error("GetRegistry() returned different registry")
				}
			})
		}
	})

	t.Run("groups", func(t *testing.T) {
		t.Run("basic_group_creation", func(t *testing.T) {
			e := echo.New()
			router := NewRouter(e)

			groupRouter := router.Group("/api/v1")
			if groupRouter == nil {
				t.Fatal("Group() returned nil")
			}

			if groupRouter.group == nil {
				t.Error("Group router has nil group")
			}

			if groupRouter.GetRegistry() == nil {
				t.Error("Group router has nil registry")
			}

			if unwrapped := groupRouter.Unwrap(); unwrapped != e {
				t.Error("Group router Unwrap() did not return original Echo instance")
			}
		})

		t.Run("nested_groups", func(t *testing.T) {
			e := echo.New()
			router := NewRouter(e)

			// Create nested groups
			apiGroup := router.Group("/api")
			v1Group := apiGroup.Group("/v1")
			usersGroup := v1Group.Group("/users")

			// Define a simple handler for testing
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
		})

		t.Run("group_route_registration", func(t *testing.T) {
			router := NewRouter(echo.New())
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
	})
}

func TestParameterAdapter(t *testing.T) {
	adapter := echoParamAdapter{}

	t.Run("path_parameters", func(t *testing.T) {
		t.Run("with_echo_context", func(t *testing.T) {
			e := echo.New()

			e.GET("/users/:id", func(c echo.Context) error {
				// Create request with Echo context
				req := c.Request()
				req = req.WithContext(context.WithValue(req.Context(), echoCtxKey{}, c))

				// Set path parameter
				c.SetParamNames("id")
				c.SetParamValues("123")

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

				return nil
			})

			// Make a test request
			req := httptest.NewRequest("GET", "/users/123", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
		})

		t.Run("without_echo_context", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users/123", nil)

			value, ok := adapter.Path(req, "id")
			if ok {
				t.Error("Path() returned true without Echo context")
			}
			if value != "" {
				t.Errorf("Path() returned %q without Echo context, want empty string", value)
			}
		})
	})

	t.Run("query_parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?name=john&age=30", nil)

		tests := []struct {
			name    string
			key     string
			wantVal []string
			wantOk  bool
		}{
			{
				name:    "existing_parameter",
				key:     "name",
				wantVal: []string{"john"},
				wantOk:  true,
			},
			{
				name:    "non_existing_parameter",
				key:     "nonexistent",
				wantVal: nil,
				wantOk:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				value, ok := adapter.Query(req, tt.key)
				if ok != tt.wantOk {
					t.Errorf("Query() ok = %v, want %v", ok, tt.wantOk)
				}
				if !slices.Equal(value, tt.wantVal) {
					t.Errorf("Query() value = %q, want %q", value, tt.wantVal)
				}
			})
		}
	})

	t.Run("headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer token123")
		req.Header.Set("Content-Type", "application/json")

		tests := []struct {
			name    string
			key     string
			wantVal string
			wantOk  bool
		}{
			{
				name:    "existing_header",
				key:     "Authorization",
				wantVal: "Bearer token123",
				wantOk:  true,
			},
			{
				name:    "non_existing_header",
				key:     "NonExistent",
				wantVal: "",
				wantOk:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				value, ok := adapter.Header(req, tt.key)
				if ok != tt.wantOk {
					t.Errorf("Header() ok = %v, want %v", ok, tt.wantOk)
				}
				if value != tt.wantVal {
					t.Errorf("Header() value = %q, want %q", value, tt.wantVal)
				}
			})
		}
	})

	t.Run("cookies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
		req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})

		tests := []struct {
			name    string
			key     string
			wantVal string
			wantOk  bool
		}{
			{
				name:    "existing_cookie",
				key:     "session",
				wantVal: "abc123",
				wantOk:  true,
			},
			{
				name:    "non_existing_cookie",
				key:     "nonexistent",
				wantVal: "",
				wantOk:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				value, ok := adapter.Cookie(req, tt.key)
				if ok != tt.wantOk {
					t.Errorf("Cookie() ok = %v, want %v", ok, tt.wantOk)
				}
				if value != tt.wantVal {
					t.Errorf("Cookie() value = %q, want %q", value, tt.wantVal)
				}
			})
		}
	})
}

func TestRouterHTTPMethods(t *testing.T) {
	router := NewRouter(echo.New())

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

	// Test data for all HTTP methods
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

	// Register routes for each HTTP method
	for _, tt := range tests {
		switch tt.method {
		case "GET":
			router.Get(tt.path, handler)
		case "POST":
			router.Post(tt.path, handler)
		case "PUT":
			router.Put(tt.path, handler)
		case "DELETE":
			router.Delete(tt.path, handler)
		case "PATCH":
			router.Patch(tt.path, handler)
		}
	}

	// Test generic Register method
	router.Register("POST", "/register-test", handler)

	// Verify routes were registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	// Check all expected routes
	for _, tt := range tests {
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
	}

	// Check Register method route
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

func TestRouterIntegration(t *testing.T) {
	e := echo.New()
	router := NewRouter(e)

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

func TestRouterGroupServesPrefixedPath(t *testing.T) {
	router := NewRouter(nil)
	type liveResponse struct {
		Body struct {
			Status string `gork:"status"`
		}
	}
	handler := func(context.Context, struct{}) (*liveResponse, error) { return &liveResponse{}, nil }
	router.Group("/v1").Get("/live", handler)
	router.Group("/api").Group("/v2").Get("/live", handler)

	for path, want := range map[string]int{
		"/v1/live":     http.StatusOK,
		"/api/v2/live": http.StatusOK,
		"/v1/v1/live":  http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		router.echo.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, want)
		}
	}
}

func TestRouterRepeatedQueryParameter(t *testing.T) {
	type queryRequest struct {
		Query struct {
			Model []string `gork:"model"`
			Name  string   `gork:"name"`
		}
	}
	type queryResponse struct {
		Body struct {
			Status string `gork:"status"`
		}
	}

	var got queryRequest
	router := NewRouter(nil)
	router.Get("/query", func(_ context.Context, req queryRequest) (*queryResponse, error) {
		got = req
		return &queryResponse{}, nil
	})

	tests := []struct {
		name      string
		target    string
		wantModel []string
		wantName  string
	}{
		{"repeated parameter", "/query?model=a&model=b", []string{"a", "b"}, ""},
		{"comma separated value", "/query?model=a,b", []string{"a", "b"}, ""},
		{"repeated and comma separated", "/query?model=a,b&model=c", []string{"a", "b", "c"}, ""},
		{"repeated parameter in string field", "/query?name=x&name=y", nil, "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got = queryRequest{}
			rec := httptest.NewRecorder()
			router.echo.ServeHTTP(rec, httptest.NewRequest("GET", tt.target, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if !slices.Equal(got.Query.Model, tt.wantModel) {
				t.Errorf("Query.Model = %q, want %q", got.Query.Model, tt.wantModel)
			}
			if got.Query.Name != tt.wantName {
				t.Errorf("Query.Name = %q, want %q", got.Query.Name, tt.wantName)
			}
		})
	}
}
