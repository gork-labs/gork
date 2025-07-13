package generator

import (
	"go/ast"
	"go/parser"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectGinRoute(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected *ExtractedRoute
	}{
		{
			name: "simple GET route",
			code: `router.GET("/users", handlers.ListUsers)`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users",
				HandlerName: "handlers.ListUsers",
			},
		},
		{
			name: "POST route with path params",
			code: `router.POST("/users/:id/posts", handlers.CreatePost)`,
			expected: &ExtractedRoute{
				Method:      "POST",
				Path:        "/users/{id}/posts",
				HandlerName: "handlers.CreatePost",
			},
		},
		{
			name: "DELETE route with multiple params",
			code: `app.DELETE("/users/:userId/posts/:postId", handlers.DeletePost)`,
			expected: &ExtractedRoute{
				Method:      "DELETE",
				Path:        "/users/{userId}/posts/{postId}",
				HandlerName: "handlers.DeletePost",
			},
		},
		{
			name: "PUT route",
			code: `r.PUT("/api/v1/users/:id", UpdateUser)`,
			expected: &ExtractedRoute{
				Method:      "PUT",
				Path:        "/api/v1/users/{id}",
				HandlerName: "UpdateUser",
			},
		},
		{
			name: "PATCH route",
			code: `router.PATCH("/users/:id", handlers.PatchUser)`,
			expected: &ExtractedRoute{
				Method:      "PATCH",
				Path:        "/users/{id}",
				HandlerName: "handlers.PatchUser",
			},
		},
		{
			name:     "invalid method",
			code:     `router.INVALID("/users", handlers.ListUsers)`,
			expected: nil,
		},
		{
			name:     "missing path",
			code:     `router.GET(handlers.ListUsers)`,
			expected: nil,
		},
		{
			name:     "missing handler",
			code:     `router.GET("/users")`,
			expected: nil,
		},
		{
			name:     "non-string path",
			code:     `router.GET(pathVar, handlers.ListUsers)`,
			expected: nil,
		},
	}

	rd := NewRouteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			callExpr, ok := expr.(*ast.CallExpr)
			require.True(t, ok)
			
			result := rd.detectGinRoute(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectEchoRoute(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected *ExtractedRoute
	}{
		{
			name: "echo GET route",
			code: `e.GET("/users", getUserHandler)`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users",
				HandlerName: "getUserHandler",
			},
		},
		{
			name: "echo POST with params",
			code: `e.POST("/users/:id", handlers.UpdateUser)`,
			expected: &ExtractedRoute{
				Method:      "POST",
				Path:        "/users/{id}",
				HandlerName: "handlers.UpdateUser",
			},
		},
	}

	rd := NewRouteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			callExpr, ok := expr.(*ast.CallExpr)
			require.True(t, ok)
			
			result := rd.detectEchoRoute(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectGorillaMuxRoute(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected *ExtractedRoute
	}{
		{
			name: "gorilla mux with Methods",
			code: `router.HandleFunc("/users", handlers.ListUsers).Methods("GET")`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users",
				HandlerName: "handlers.ListUsers",
			},
		},
		{
			name: "gorilla mux with path params",
			code: `mux.HandleFunc("/users/{id}", handlers.GetUser).Methods("GET")`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users/{id}",
				HandlerName: "handlers.GetUser",
			},
		},
		{
			name: "gorilla mux POST",
			code: `r.HandleFunc("/api/v1/users", CreateUser).Methods("POST")`,
			expected: &ExtractedRoute{
				Method:      "POST",
				Path:        "/api/v1/users",
				HandlerName: "CreateUser",
			},
		},
		{
			name: "gorilla mux with complex path",
			code: `router.HandleFunc("/users/{userId}/posts/{postId}", handlers.GetPost).Methods("GET")`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users/{userId}/posts/{postId}",
				HandlerName: "handlers.GetPost",
			},
		},
		{
			name:     "not a Methods call",
			code:     `router.HandleFunc("/users", handlers.ListUsers).Subrouter()`,
			expected: nil,
		},
		{
			name:     "not HandleFunc",
			code:     `router.Handle("/users", handlers.ListUsers).Methods("GET")`,
			expected: nil,
		},
		{
			name:     "missing method argument",
			code:     `router.HandleFunc("/users", handlers.ListUsers).Methods()`,
			expected: nil,
		},
	}

	rd := NewRouteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			callExpr, ok := expr.(*ast.CallExpr)
			require.True(t, ok)
			
			result := rd.detectGorillaMuxRoute(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectChiRoute(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected *ExtractedRoute
	}{
		{
			name: "chi GET route",
			code: `r.Get("/users", handlers.ListUsers)`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users",
				HandlerName: "handlers.ListUsers",
			},
		},
		{
			name: "chi POST with params",
			code: `r.Post("/users/{id}", handlers.UpdateUser)`,
			expected: &ExtractedRoute{
				Method:      "POST",
				Path:        "/users/{id}",
				HandlerName: "handlers.UpdateUser",
			},
		},
		{
			name: "chi DELETE",
			code: `router.Delete("/users/{id}", handlers.DeleteUser)`,
			expected: &ExtractedRoute{
				Method:      "DELETE",
				Path:        "/users/{id}",
				HandlerName: "handlers.DeleteUser",
			},
		},
	}

	rd := NewRouteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			callExpr, ok := expr.(*ast.CallExpr)
			require.True(t, ok)
			
			result := rd.detectChiRoute(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectFiberRoute(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected *ExtractedRoute
	}{
		{
			name: "fiber GET route",
			code: `app.Get("/users", handlers.ListUsers)`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users",
				HandlerName: "handlers.ListUsers",
			},
		},
		{
			name: "fiber POST with params",
			code: `app.Post("/users/:id", handlers.UpdateUser)`,
			expected: &ExtractedRoute{
				Method:      "POST",
				Path:        "/users/{id}",
				HandlerName: "handlers.UpdateUser",
			},
		},
		{
			name: "fiber with multiple params",
			code: `app.Put("/users/:userId/posts/:postId", handlers.UpdatePost)`,
			expected: &ExtractedRoute{
				Method:      "PUT",
				Path:        "/users/{userId}/posts/{postId}",
				HandlerName: "handlers.UpdatePost",
			},
		},
	}

	rd := NewRouteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			callExpr, ok := expr.(*ast.CallExpr)
			require.True(t, ok)
			
			result := rd.detectFiberRoute(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectStdLibRoute(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected *ExtractedRoute
	}{
		{
			name: "http.HandleFunc simple",
			code: `http.HandleFunc("/users", handlers.ListUsers)`,
			expected: &ExtractedRoute{
				Method:      "",
				Path:        "/users",
				HandlerName: "handlers.ListUsers",
			},
		},
		{
			name: "http.HandleFunc with method prefix",
			code: `http.HandleFunc("GET /users", handlers.ListUsers)`,
			expected: &ExtractedRoute{
				Method:      "GET",
				Path:        "/users",
				HandlerName: "handlers.ListUsers",
			},
		},
		{
			name: "mux.HandleFunc",
			code: `mux.HandleFunc("/api/v1/users", handlers.ListUsers)`,
			expected: &ExtractedRoute{
				Method:      "",
				Path:        "/api/v1/users",
				HandlerName: "handlers.ListUsers",
			},
		},
		{
			name: "mux with method prefix",
			code: `mux.HandleFunc("POST /api/v1/users", handlers.CreateUser)`,
			expected: &ExtractedRoute{
				Method:      "POST",
				Path:        "/api/v1/users",
				HandlerName: "handlers.CreateUser",
			},
		},
		{
			name: "http with complex path",
			code: `http.HandleFunc("PUT /users/{id}/profile", handlers.UpdateProfile)`,
			expected: &ExtractedRoute{
				Method:      "PUT",
				Path:        "/users/{id}/profile",
				HandlerName: "handlers.UpdateProfile",
			},
		},
		{
			name:     "not HandleFunc",
			code:     `http.Handle("/users", handlers.ListUsers)`,
			expected: nil,
		},
		{
			name:     "wrong package",
			code:     `foo.HandleFunc("/users", handlers.ListUsers)`,
			expected: nil,
		},
		{
			name:     "missing arguments",
			code:     `http.HandleFunc("/users")`,
			expected: nil,
		},
	}

	rd := NewRouteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			callExpr, ok := expr.(*ast.CallExpr)
			require.True(t, ok)
			
			result := rd.detectStdLibRoute(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractWrappedHandler(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "api.HandlerFunc with simple handler",
			code:     `api.HandlerFunc(handlers.Login)`,
			expected: "handlers.Login",
		},
		{
			name:     "api.HandlerFunc with multiple args",
			code:     `api.HandlerFunc(handlers.Login, middleware.Auth)`,
			expected: "handlers.Login",
		},
		{
			name:     "not api.HandlerFunc",
			code:     `foo.HandlerFunc(handlers.Login)`,
			expected: "",
		},
		{
			name:     "different function name",
			code:     `api.Wrapper(handlers.Login)`,
			expected: "",
		},
		{
			name:     "no arguments",
			code:     `api.HandlerFunc()`,
			expected: "",
		},
		{
			name:     "not a call expression",
			code:     `handlers.Login`,
			expected: "",
		},
	}

	rd := NewRouteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			result := rd.extractWrappedHandler(expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHTTPMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"GET", true},
		{"POST", true},
		{"PUT", true},
		{"DELETE", true},
		{"PATCH", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"CONNECT", true},
		{"TRACE", true},
		{"get", true},
		{"post", true},
		{"INVALID", false},
		{"LIST", false},
		{"CREATE", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := isHTTPMethod(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple string",
			code:     `"/users"`,
			expected: "/users",
		},
		{
			name:     "string with spaces",
			code:     `"GET /api/v1/users"`,
			expected: "GET /api/v1/users",
		},
		{
			name:     "empty string",
			code:     `""`,
			expected: "",
		},
		{
			name:     "not a string literal",
			code:     `pathVar`,
			expected: "",
		},
		{
			name:     "number literal",
			code:     `123`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			result := extractStringLiteral(expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractHandlerName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple identifier",
			code:     `ListUsers`,
			expected: "ListUsers",
		},
		{
			name:     "qualified name",
			code:     `handlers.ListUsers`,
			expected: "handlers.ListUsers",
		},
		{
			name:     "nested package",
			code:     `api.handlers.ListUsers`,
			expected: "",
		},
		{
			name:     "function literal",
			code:     `func(w http.ResponseWriter, r *http.Request) {}`,
			expected: "anonymous",
		},
		{
			name:     "complex expression",
			code:     `getHandler()`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			result := extractHandlerName(expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertPathParams(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		framework string
		expected  string
	}{
		{
			name:      "gin single param",
			path:      "/users/:id",
			framework: "gin",
			expected:  "/users/{id}",
		},
		{
			name:      "gin multiple params",
			path:      "/users/:userId/posts/:postId",
			framework: "gin",
			expected:  "/users/{userId}/posts/{postId}",
		},
		{
			name:      "echo params",
			path:      "/users/:id/comments/:commentId",
			framework: "echo",
			expected:  "/users/{id}/comments/{commentId}",
		},
		{
			name:      "fiber params",
			path:      "/api/:version/users/:id",
			framework: "fiber",
			expected:  "/api/{version}/users/{id}",
		},
		{
			name:      "gorilla already correct",
			path:      "/users/{id}",
			framework: "gorilla",
			expected:  "/users/{id}",
		},
		{
			name:      "chi already correct",
			path:      "/users/{id}/posts/{postId}",
			framework: "chi",
			expected:  "/users/{id}/posts/{postId}",
		},
		{
			name:      "no params",
			path:      "/users",
			framework: "gin",
			expected:  "/users",
		},
		{
			name:      "unknown framework",
			path:      "/users/:id",
			framework: "unknown",
			expected:  "/users/:id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPathParams(tt.path, tt.framework)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPathParameters(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "single parameter",
			path:     "/users/{id}",
			expected: []string{"id"},
		},
		{
			name:     "multiple parameters",
			path:     "/users/{userId}/posts/{postId}",
			expected: []string{"userId", "postId"},
		},
		{
			name:     "no parameters",
			path:     "/users",
			expected: []string{},
		},
		{
			name:     "parameter at start",
			path:     "/{version}/users/{id}",
			expected: []string{"version", "id"},
		},
		{
			name:     "parameter at end",
			path:     "/users/{id}",
			expected: []string{"id"},
		},
		{
			name:     "mixed with static segments",
			path:     "/api/v1/users/{userId}/posts/{postId}/comments",
			expected: []string{"userId", "postId"},
		},
		{
			name:     "parameter with numbers",
			path:     "/users/{userId123}",
			expected: []string{"userId123"},
		},
		{
			name:     "underscore in parameter",
			path:     "/users/{user_id}",
			expected: []string{"user_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPathParameters(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferMethodFromHandler(t *testing.T) {
	tests := []struct {
		name         string
		handlerName  string
		expectedHTTP string
	}{
		// GET patterns
		{
			name:         "GetUser",
			handlerName:  "GetUser",
			expectedHTTP: "GET",
		},
		{
			name:         "ListUsers",
			handlerName:  "ListUsers",
			expectedHTTP: "GET",
		},
		{
			name:         "FetchUser",
			handlerName:  "FetchUser",
			expectedHTTP: "GET",
		},
		{
			name:         "handlers.GetUser",
			handlerName:  "handlers.GetUser",
			expectedHTTP: "GET",
		},
		
		// POST patterns
		{
			name:         "CreateUser",
			handlerName:  "CreateUser",
			expectedHTTP: "POST",
		},
		{
			name:         "PostMessage",
			handlerName:  "PostMessage",
			expectedHTTP: "POST",
		},
		{
			name:         "AddUser",
			handlerName:  "AddUser",
			expectedHTTP: "POST",
		},
		
		// PUT patterns
		{
			name:         "UpdateUser",
			handlerName:  "UpdateUser",
			expectedHTTP: "PUT",
		},
		{
			name:         "PutUser",
			handlerName:  "PutUser",
			expectedHTTP: "PUT",
		},
		{
			name:         "EditUser",
			handlerName:  "EditUser",
			expectedHTTP: "PUT",
		},
		
		// PATCH patterns
		{
			name:         "PatchUser",
			handlerName:  "PatchUser",
			expectedHTTP: "PATCH",
		},
		{
			name:         "ModifyUser",
			handlerName:  "ModifyUser",
			expectedHTTP: "PATCH",
		},
		
		// DELETE patterns
		{
			name:         "DeleteUser",
			handlerName:  "DeleteUser",
			expectedHTTP: "DELETE",
		},
		{
			name:         "RemoveUser",
			handlerName:  "RemoveUser",
			expectedHTTP: "DELETE",
		},
		
		// Handler prefix patterns
		{
			name:         "HandleGetUser",
			handlerName:  "HandleGetUser",
			expectedHTTP: "GET",
		},
		{
			name:         "HandlerCreateUser",
			handlerName:  "HandlerCreateUser",
			expectedHTTP: "POST",
		},
		
		// Unknown patterns default to POST
		{
			name:         "ProcessUser",
			handlerName:  "ProcessUser",
			expectedHTTP: "POST",
		},
		{
			name:         "HandleUser",
			handlerName:  "HandleUser",
			expectedHTTP: "POST",
		},
		{
			name:         "UserHandler",
			handlerName:  "UserHandler",
			expectedHTTP: "POST",
		},
		{
			name:         "anonymous",
			handlerName:  "anonymous",
			expectedHTTP: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferMethodFromHandler(tt.handlerName)
			assert.Equal(t, tt.expectedHTTP, result)
		})
	}
}

func TestDetectRoutesFromFile(t *testing.T) {
	// Create a temporary file with route definitions
	content := `package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
)

func setupRoutes() {
	// Gin routes
	r := gin.Default()
	r.GET("/users", handlers.ListUsers)
	r.POST("/users", handlers.CreateUser)
	r.GET("/users/:id", handlers.GetUser)
	r.PUT("/users/:id", handlers.UpdateUser)
	r.DELETE("/users/:id", handlers.DeleteUser)

	// Gorilla Mux routes  
	router := mux.NewRouter()
	router.HandleFunc("/api/users", handlers.ListUsers).Methods("GET")
	router.HandleFunc("/api/users/{id}", handlers.GetUser).Methods("GET")
	router.HandleFunc("/api/users", handlers.CreateUser).Methods("POST")

	// Standard library
	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("GET /version", handlers.Version)

	// Chi routes
	r.Get("/posts", handlers.ListPosts)
	r.Post("/posts", handlers.CreatePost)
}`

	// Write to temporary file
	tmpfile, err := createTempFile(content)
	require.NoError(t, err)
	defer tmpfile.Close()

	rd := NewRouteDetector()
	routes, err := rd.DetectRoutesFromFile(tmpfile.Name())
	require.NoError(t, err)

	// Should detect multiple routes
	assert.Greater(t, len(routes), 5)

	// Check for specific routes
	expectedRoutes := []ExtractedRoute{
		{Method: "GET", Path: "/users", HandlerName: "handlers.ListUsers"},
		{Method: "POST", Path: "/users", HandlerName: "handlers.CreateUser"},
		{Method: "GET", Path: "/users/{id}", HandlerName: "handlers.GetUser"},
		{Method: "PUT", Path: "/users/{id}", HandlerName: "handlers.UpdateUser"},
		{Method: "DELETE", Path: "/users/{id}", HandlerName: "handlers.DeleteUser"},
		{Method: "GET", Path: "/api/users", HandlerName: "handlers.ListUsers"},
		{Method: "GET", Path: "/api/users/{id}", HandlerName: "handlers.GetUser"},
		{Method: "POST", Path: "/api/users", HandlerName: "handlers.CreateUser"},
		{Method: "", Path: "/health", HandlerName: "handlers.Health"},
		{Method: "GET", Path: "/version", HandlerName: "handlers.Version"},
		{Method: "GET", Path: "/posts", HandlerName: "handlers.ListPosts"},
		{Method: "POST", Path: "/posts", HandlerName: "handlers.CreatePost"},
	}

	// Check that all expected routes are found
	for _, expected := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Method == expected.Method && 
			   route.Path == expected.Path && 
			   route.HandlerName == expected.HandlerName {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected route not found: %+v", expected)
	}
}

// Helper function to create temporary files for testing
func createTempFile(content string) (*os.File, error) {
	tmpfile, err := os.CreateTemp("", "route_test_*.go")
	if err != nil {
		return nil, err
	}
	
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		return nil, err
	}
	
	return tmpfile, nil
}