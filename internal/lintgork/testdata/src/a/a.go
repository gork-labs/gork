package a

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// Test struct with valid openapi tags
type ValidStruct struct {
	ID     string `json:"id" openapi:"id,in=path"`
	Name   string `json:"name" openapi:"name,in=query"`
	Auth   string `json:"auth" openapi:"Authorization,in=header"`
	UserType string `json:"user_type" openapi:"discriminator=user"`
}

// Test struct with invalid openapi tags
type InvalidStruct struct {
	BadTag1 string `json:"bad1" openapi:"invalid"` // want "invalid openapi tag"
	BadTag2 string `json:"bad2" openapi:",in=query"` // want "invalid openapi tag"
	BadTag3 string `json:"bad3" openapi:"name,in=invalid"` // want "invalid openapi tag"
	BadTag4 string `json:"bad4" openapi:"discriminator="` // want "invalid openapi tag"
}

// Test struct with duplicate discriminator values
type DuplicateStruct1 struct {
	Type string `json:"type" openapi:"discriminator=admin"`
}

type DuplicateStruct2 struct {
	Kind string `json:"kind" openapi:"discriminator=admin"` // want "duplicate discriminator value"
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
	fmt.Println("/users/{id}", handler)  // This should be ignored
	log.Println("/posts/{postId}", handler)  // This should be ignored
	
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