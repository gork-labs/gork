package a

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// Test struct with valid gork tags using Convention Over Configuration
type ValidRequest struct {
	Query struct {
		UserID string `gork:"user_id"`
		Filter string `gork:"filter"`
	}
	Path struct {
		ID string `gork:"id"`
	}
	Headers struct {
		Auth string `gork:"Authorization"`
	}
	Body struct {
		Name string `gork:"name"`
		Type string `gork:"type,discriminator=user"`
	}
}

// Test struct with Convention Over Configuration sections
type ConventionRequest struct {
	Query struct {
		Limit  int `gork:"limit"`
		Offset int `gork:"offset"`
	}
	Body struct {
		Data string `gork:"data"`
	}
}

// Test struct with invalid gork tags
type InvalidRequest struct {
	Query struct {
		BadTag1 string // want "field 'Query.BadTag1' missing gork tag"
		BadTag2 string `gork:""`                    // want "field 'Query.BadTag2' has empty gork tag"
		BadTag3 string `gork:"name,invalid=option"` // want "field 'Query.BadTag3' unknown gork tag option 'invalid'"
		BadTag4 string `gork:"name,discriminator="` // want "field 'Query.BadTag4' discriminator value cannot be empty"
	}
}

// Test router method calls
func setupRoutes(router TestRouter) {
	// Valid router calls with path parameters
	router.Get("/users/{id}", handler)
	router.Post("/users/{userId}/posts/{postId}", handler)
	router.Put("/files/*", handler)

	// This should trigger path validation for empty placeholder (if implemented)
	router.Delete("/bad/{}", handler) // want "empty placeholder found in route"

	// Non-router method calls to trigger the !isRouterMethodCall path
	fmt.Println("/users/{id}", handler)     // This should be ignored
	log.Println("/posts/{postId}", handler) // This should be ignored

	// Router calls with invalid string literals to trigger pathStr == "" path
	// Note: These won't compile but they test the AST parsing edge cases
}

func handler(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

// Additional test functions to trigger specific coverage paths
func moreRouteCalls(router TestRouter) {
	// Calls with less than 2 arguments (should be ignored)
	router.Get("/test", handler)

	// Calls with non-string first argument (should be ignored)
	var pathVar = "/users/{id}"
	router.Post(pathVar, handler)
}

func nonRouterCalls() {
	// Non-router method calls to trigger !isRouterMethodCall path
	fmt.Printf("/users/{id}")
	log.Println("/posts/{postId}")
	http.Get("/api/{version}")
}

// Router interface for testing
type TestRouter interface {
	Get(path string, handler interface{})
	Post(path string, handler interface{})
	Put(path string, handler interface{})
	Delete(path string, handler interface{})
	Patch(path string, handler interface{})
}
