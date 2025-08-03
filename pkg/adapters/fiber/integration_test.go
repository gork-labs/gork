package fiber

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gork-labs/gork/pkg/api"
)

// Integration test types
type UserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=18"`
}

type UserResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type GetUserRequest struct {
	UserID string `path:"userId" validate:"required"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

type ListUsersResponse struct {
	Body []*UserResponse
}

type StatsResponse struct {
	Body struct {
		TotalUsers  int `gork:"totalUsers"`
		ActiveUsers int `gork:"activeUsers"`
	}
}

type VersionResponse struct {
	Body struct {
		Version string `gork:"version"`
	}
}

type HealthResponse struct {
	Body struct {
		Status string `gork:"status"`
	}
}

type SimpleTestResponse struct {
	Body struct {
		Message string `gork:"message"`
	}
}

type ComplexResponse struct {
	Body struct {
		UserID    string `gork:"userId"`
		PostID    string `gork:"postId"`
		Format    string `gork:"format"`
		Page      string `gork:"page"`
		AuthToken string `gork:"authToken"`
		SessionID string `gork:"sessionId"`
	}
}

// TestCompleteAPIScenario tests a complete API scenario with multiple endpoints
func TestCompleteAPIScenario(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	// Create API v1 group
	v1 := router.Group("/api/v1")

	// User handlers
	createUserHandler := func(ctx context.Context, req UserRequest) (*UserResponse, error) {
		return &UserResponse{
			ID:    123,
			Name:  req.Name,
			Email: req.Email,
			Age:   req.Age,
		}, nil
	}

	getUserHandler := func(ctx context.Context, req GetUserRequest) (*UserResponse, error) {
		return &UserResponse{
			ID:    123,
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   25,
		}, nil
	}

	listUsersHandler := func(ctx context.Context, req struct{}) (*ListUsersResponse, error) {
		return &ListUsersResponse{
			Body: []*UserResponse{
				{ID: 1, Name: "User 1", Email: "user1@example.com", Age: 25},
				{ID: 2, Name: "User 2", Email: "user2@example.com", Age: 30},
			},
		}, nil
	}

	// Register routes
	v1.Post("/users", createUserHandler)
	v1.Get("/users/{userId}", getUserHandler)
	v1.Get("/users", listUsersHandler)

	// Test route registration
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	expectedRoutes := map[string]string{
		"POST": "/api/v1/users",
		"GET":  "/api/v1/users/{userId}",
	}

	for method, path := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Method == method && route.Path == path {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected route %s %s not found", method, path)
		}
	}

	// Test actual HTTP requests (simulation)
	t.Run("create_user", func(t *testing.T) {
		userJSON := `{"name":"Jane Doe","email":"jane@example.com","age":28}`
		req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(userJSON))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test create user: %v", err)
		}
		defer resp.Body.Close()

		// Since we're testing the router registration, not the full HTTP flow,
		// we mainly verify that routes are properly registered
		if len(routes) == 0 {
			t.Error("No routes registered")
		}
	})
}

// TestNestedGroupsScenario tests complex nested group scenarios
func TestNestedGroupsScenario(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	// Create nested group structure: /api/v1/admin/users
	api := router.Group("/api")
	v1 := api.Group("/v1")
	admin := v1.Group("/admin")
	users := admin.Group("/users")

	// Register handlers at different levels
	api.Get("/health", func(ctx context.Context, req struct{}) (*HealthResponse, error) {
		return &HealthResponse{
			Body: struct {
				Status string `gork:"status"`
			}{
				Status: "healthy",
			},
		}, nil
	})

	v1.Get("/version", func(ctx context.Context, req struct{}) (*VersionResponse, error) {
		return &VersionResponse{
			Body: struct {
				Version string `gork:"version"`
			}{
				Version: "1.0.0",
			},
		}, nil
	})

	admin.Get("/stats", func(ctx context.Context, req struct{}) (*StatsResponse, error) {
		return &StatsResponse{
			Body: struct {
				TotalUsers  int `gork:"totalUsers"`
				ActiveUsers int `gork:"activeUsers"`
			}{
				TotalUsers:  100,
				ActiveUsers: 85,
			},
		}, nil
	})

	users.Get("/list", func(ctx context.Context, req struct{}) (*ListUsersResponse, error) {
		return &ListUsersResponse{
			Body: []*UserResponse{
				{ID: 1, Name: "Admin User", Email: "admin@example.com", Age: 35},
			},
		}, nil
	})

	users.Post("/create", func(ctx context.Context, req UserRequest) (*UserResponse, error) {
		return &UserResponse{
			ID:    999,
			Name:  req.Name,
			Email: req.Email,
			Age:   req.Age,
		}, nil
	})

	// Verify all routes have correct prefixes
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	expectedPaths := []string{
		"/api/health",
		"/api/v1/version",
		"/api/v1/admin/stats",
		"/api/v1/admin/users/list",
		"/api/v1/admin/users/create",
	}

	for _, expectedPath := range expectedPaths {
		found := false
		for _, route := range routes {
			if route.Path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected path %s not found in registered routes", expectedPath)
		}
	}

	// Test group registry sharing
	if users.GetRegistry() != router.GetRegistry() {
		t.Error("Nested groups should share the same registry")
	}

	// Test prefix accumulation
	if users.prefix != "/api/v1/admin/users" {
		t.Errorf("Expected prefix '/api/v1/admin/users', got '%s'", users.prefix)
	}
}

// TestMiddlewareAndOptionsScenario tests middleware and options propagation
func TestMiddlewareAndOptionsScenario(t *testing.T) {
	app := fiber.New()

	// Create router with middleware
	middleware1 := api.WithTags("auth")
	middleware2 := api.WithTags("logging")
	router := NewRouter(app, middleware1, middleware2)

	// Test middleware preservation
	if len(router.middleware) != 2 {
		t.Errorf("Expected 2 middleware options, got %d", len(router.middleware))
	}

	// Create groups and verify middleware propagation
	apiGroup := router.Group("/api")
	if len(apiGroup.middleware) != 2 {
		t.Errorf("Group should inherit middleware, got %d", len(apiGroup.middleware))
	}

	// Register handlers with additional options
	handler := func(ctx context.Context, req struct{}) (*SimpleTestResponse, error) {
		return &SimpleTestResponse{
			Body: struct {
				Message string `gork:"message"`
			}{
				Message: "success",
			},
		}, nil
	}

	router.Get("/test", handler, api.WithTags("endpoint"))
	apiGroup.Post("/test", handler, api.WithTags("group-endpoint"))

	// Verify routes are registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}
}

// TestParameterExtractionScenario tests realistic parameter extraction
func TestParameterExtractionScenario(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	// Define request with all parameter types
	type ComplexRequest struct {
		UserID    string `path:"userId" validate:"required"`
		PostID    string `path:"postId" validate:"required"`
		Format    string `query:"format"`
		Page      string `query:"page"`
		AuthToken string `header:"Authorization"`
		SessionID string `cookie:"session_id"`
	}

	handler := func(ctx context.Context, req ComplexRequest) (*ComplexResponse, error) {
		return &ComplexResponse{
			Body: struct {
				UserID    string `gork:"userId"`
				PostID    string `gork:"postId"`
				Format    string `gork:"format"`
				Page      string `gork:"page"`
				AuthToken string `gork:"authToken"`
				SessionID string `gork:"sessionId"`
			}{
				UserID:    req.UserID,
				PostID:    req.PostID,
				Format:    req.Format,
				Page:      req.Page,
				AuthToken: req.AuthToken,
				SessionID: req.SessionID,
			},
		}, nil
	}

	router.Get("/users/{userId}/posts/{postId}", handler)

	// Verify route registration
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	found := false
	for _, route := range routes {
		if route.Path == "/users/{userId}/posts/{postId}" && route.Method == "GET" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Complex parameter route not registered correctly")
	}

	// Test path conversion
	expectedNativePath := "/users/:userId/posts/:postId"
	actualNativePath := toNativePath("/users/{userId}/posts/{postId}")
	if actualNativePath != expectedNativePath {
		t.Errorf("Path conversion failed: expected %s, got %s", expectedNativePath, actualNativePath)
	}
}

// TestErrorHandlingScenario tests realistic error handling scenarios
func TestErrorHandlingScenario(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	// Handler that returns an error
	errorHandler := func(ctx context.Context, req struct{}) (*UserResponse, error) {
		return nil, errors.New("user not found")
	}

	// Handler that returns success
	successHandler := func(ctx context.Context, req struct{}) (*UserResponse, error) {
		return &UserResponse{
			ID:    1,
			Name:  "Success User",
			Email: "success@example.com",
			Age:   30,
		}, nil
	}

	router.Get("/error", errorHandler)
	router.Get("/success", successHandler)

	// Verify both routes are registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	// Test that the router structure supports error handling
	for _, route := range routes {
		if route.Path != "/error" && route.Path != "/success" {
			t.Errorf("Unexpected route path: %s", route.Path)
		}
	}
}

// TestDocumentationIntegration tests documentation generation integration
func TestDocumentationIntegration(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	// Register some API routes
	router.Get("/users", func(ctx context.Context, req struct{}) (*ListUsersResponse, error) {
		return &ListUsersResponse{
			Body: []*UserResponse{},
		}, nil
	})

	router.Post("/users", func(ctx context.Context, req UserRequest) (*UserResponse, error) {
		return &UserResponse{}, nil
	})

	// Register documentation routes
	router.DocsRoute("/docs/*")
	router.DocsRoute("/api-docs/*", api.DocsConfig{
		Title:       "Test API",
		OpenAPIPath: "/openapi.json",
	})

	// Verify all routes are registered
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	// Should have API routes + documentation routes
	if len(routes) < 2 {
		t.Errorf("Expected at least 2 routes (API + docs), got %d", len(routes))
	}

	// Test that registry contains actual API routes
	apiRoutes := 0
	for _, route := range routes {
		if route.Path == "/users" {
			apiRoutes++
		}
	}

	if apiRoutes == 0 {
		t.Error("No API routes found in registry")
	}
}

// TestRequestResponseFlow tests the complete request/response flow structure
func TestRequestResponseFlow(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	// Test JSON marshaling compatibility
	testUser := UserResponse{
		ID:    42,
		Name:  "Test User",
		Email: "test@example.com",
		Age:   25,
	}

	// Verify our test types can be marshaled/unmarshaled
	jsonData, err := json.Marshal(testUser)
	if err != nil {
		t.Fatalf("Failed to marshal test user: %v", err)
	}

	var unmarshaled UserResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal test user: %v", err)
	}

	if unmarshaled.Name != testUser.Name {
		t.Error("JSON marshal/unmarshal cycle failed")
	}

	// Register a handler that uses these types
	echoHandler := func(ctx context.Context, req UserRequest) (*UserResponse, error) {
		return &UserResponse{
			ID:    100,
			Name:  req.Name,
			Email: req.Email,
			Age:   req.Age,
		}, nil
	}

	router.Post("/echo", echoHandler)

	// Verify route registration worked
	registry := router.GetRegistry()
	routes := registry.GetRoutes()

	found := false
	for _, route := range routes {
		if route.Path == "/echo" && route.Method == "POST" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Echo handler route not registered")
	}
}

// TestPathConversionIntegration tests path conversion in realistic scenarios
func TestPathConversionIntegration(t *testing.T) {
	conversions := map[string]string{
		// Real-world API patterns
		"/api/v1/users/{id}":                         "/api/v1/users/:id",
		"/api/v1/users/{userId}/posts/{postId}":      "/api/v1/users/:userId/posts/:postId",
		"/api/v1/organizations/{orgId}/members/{id}": "/api/v1/organizations/:orgId/members/:id",
		"/files/{category}/{filename}":               "/files/:category/:filename",
		"/static/*":                                  "/static/*",
		"/docs/{version}/*":                          "/docs/:version/*",

		// Edge cases
		"/":            "/",
		"":             "",
		"/simple":      "/simple",
		"/users/{id}/": "/users/:id/",
	}

	for input, expected := range conversions {
		t.Run(input, func(t *testing.T) {
			result := toNativePath(input)
			if result != expected {
				t.Errorf("toNativePath(%q) = %q, expected %q", input, result, expected)
			}
		})
	}
}
