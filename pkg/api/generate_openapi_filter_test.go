package api

import (
	"reflect"
	"testing"
)

// Test response types following Convention Over Configuration
type TestUserResponse struct {
	Body struct {
		Name string `gork:"name"`
	}
}

type TestEmptyResponse struct {
	Body struct{}
}

func TestGenerateOpenAPIWithRouteFilter(t *testing.T) {
	t.Run("default filter excludes OpenAPISpec endpoints", func(t *testing.T) {
		registry := NewRouteRegistry()

		// Add a regular route
		regularInfo := &RouteInfo{
			Method:       "GET",
			Path:         "/users",
			HandlerName:  "GetUsers",
			RequestType:  reflect.TypeOf(struct{}{}),
			ResponseType: reflect.TypeOf((*TestUserResponse)(nil)),
		}
		registry.Register(regularInfo)

		// Add a documentation endpoint that returns *OpenAPISpec (should be filtered out)
		docInfo := &RouteInfo{
			Method:       "GET",
			Path:         "/docs",
			HandlerName:  "GetDocs",
			RequestType:  reflect.TypeOf(struct{}{}),
			ResponseType: reflect.TypeOf(&OpenAPISpec{}), // This should be filtered out
		}
		registry.Register(docInfo)

		// Generate OpenAPI spec (uses default filter)
		spec := GenerateOpenAPI(registry)

		// Check that regular route is included
		if spec.Paths["/users"] == nil {
			t.Error("Expected /users path to be included")
		}

		// Check that documentation endpoint is filtered out (this tests the uncovered continue line)
		if spec.Paths["/docs"] != nil {
			t.Error("Expected /docs path to be filtered out by default filter")
		}
	})

	t.Run("custom filter excludes specific routes", func(t *testing.T) {
		registry := NewRouteRegistry()

		// Add multiple routes
		user1Info := &RouteInfo{
			Method:       "GET",
			Path:         "/users/1",
			HandlerName:  "GetUser1",
			RequestType:  reflect.TypeOf(struct{}{}),
			ResponseType: reflect.TypeOf((*TestUserResponse)(nil)),
		}
		registry.Register(user1Info)

		user2Info := &RouteInfo{
			Method:       "GET",
			Path:         "/users/2",
			HandlerName:  "GetUser2",
			RequestType:  reflect.TypeOf(struct{}{}),
			ResponseType: reflect.TypeOf((*TestUserResponse)(nil)),
		}
		registry.Register(user2Info)

		adminInfo := &RouteInfo{
			Method:       "GET",
			Path:         "/admin",
			HandlerName:  "GetAdmin",
			RequestType:  reflect.TypeOf(struct{}{}),
			ResponseType: reflect.TypeOf((*TestEmptyResponse)(nil)),
		}
		registry.Register(adminInfo)

		// Create custom filter that excludes admin routes
		customFilter := func(info *RouteInfo) bool {
			return info.Path != "/admin" // Exclude /admin route
		}

		// Generate OpenAPI spec with custom filter
		spec := GenerateOpenAPI(registry, WithRouteFilter(customFilter))

		// Check that user routes are included
		if spec.Paths["/users/1"] == nil {
			t.Error("Expected /users/1 path to be included")
		}
		if spec.Paths["/users/2"] == nil {
			t.Error("Expected /users/2 path to be included")
		}

		// Check that admin route is filtered out (this tests the uncovered continue line)
		if spec.Paths["/admin"] != nil {
			t.Error("Expected /admin path to be filtered out by custom filter")
		}
	})

	t.Run("filter with nil route", func(t *testing.T) {
		// This test verifies that the default filter handles nil routes properly
		// The default filter should return true for nil routes (as seen in defaultRouteFilter)
		result := defaultRouteFilter(nil)
		if !result {
			t.Error("Expected defaultRouteFilter to return true for nil route")
		}
	})

	t.Run("filter with nil response type", func(t *testing.T) {
		registry := NewRouteRegistry()

		// Add a route with nil response type
		nilResponseInfo := &RouteInfo{
			Method:       "POST",
			Path:         "/void",
			HandlerName:  "PostVoid",
			RequestType:  reflect.TypeOf(struct{}{}),
			ResponseType: nil, // Nil response type
		}
		registry.Register(nilResponseInfo)

		// Generate OpenAPI spec (default filter handles nil response types)
		spec := GenerateOpenAPI(registry)

		// Check that route with nil response is included (default filter returns true)
		if spec.Paths["/void"] == nil {
			t.Error("Expected /void path to be included when response type is nil")
		}
	})
}
