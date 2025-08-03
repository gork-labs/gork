// Package api provides HTTP handler wrappers and OpenAPI generation capabilities.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

// HandlerOption represents an option for configuring a handler.
type HandlerOption struct {
	Tags     []string
	Security []SecurityRequirement
}

// SecurityRequirement represents a security requirement for an operation.
type SecurityRequirement struct {
	Type   string   // "basic", "bearer", "apiKey"
	Scopes []string // For OAuth2
}

// Option is a function that modifies HandlerOption.
type Option func(*HandlerOption)

// WithTags adds tags to the handler.
func WithTags(tags ...string) Option {
	return func(h *HandlerOption) {
		h.Tags = append(h.Tags, tags...)
	}
}

// WithBasicAuth adds basic authentication requirement.
func WithBasicAuth() Option {
	return func(h *HandlerOption) {
		h.Security = append(h.Security, SecurityRequirement{
			Type: "basic",
		})
	}
}

// WithBearerTokenAuth adds bearer token authentication requirement.
func WithBearerTokenAuth(scopes ...string) Option {
	return func(h *HandlerOption) {
		h.Security = append(h.Security, SecurityRequirement{
			Type:   "bearer",
			Scopes: scopes,
		})
	}
}

// WithAPIKeyAuth adds API key authentication requirement.
func WithAPIKeyAuth() Option {
	return func(h *HandlerOption) {
		h.Security = append(h.Security, SecurityRequirement{
			Type: "apiKey",
		})
	}
}

// Helper functions

func writeError(w http.ResponseWriter, code int, message string) {
	// For 5xx errors, avoid leaking internal details to clients
	clientMessage := message
	if code >= 500 {
		clientMessage = http.StatusText(code)
	}

	// Log server-side for observability
	if code >= 500 {
		log.Printf("http %d: %s", code, message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": clientMessage,
	})
}

// FunctionNameExtractor allows dependency injection for testing.
type FunctionNameExtractor func(interface{}) string

var defaultFunctionNameExtractor FunctionNameExtractor = extractFunctionNameFromRuntime

func getFunctionName(i interface{}) string {
	return getFunctionNameWithExtractor(i, defaultFunctionNameExtractor)
}

func getFunctionNameWithExtractor(i interface{}, extractor FunctionNameExtractor) string {
	return extractor(i)
}

func extractFunctionNameFromRuntime(i interface{}) string {
	return extractFunctionNameFromRuntimeWithFunc(i, runtime.FuncForPC)
}

// FuncForPCProvider allows dependency injection for testing.
type FuncForPCProvider func(uintptr) *runtime.Func

func extractFunctionNameFromRuntimeWithFunc(i interface{}, funcProvider FuncForPCProvider) string {
	// Use FuncForPC to get the fully-qualified function name, then trim the package path
	fn := funcProvider(reflect.ValueOf(i).Pointer())
	if fn == nil {
		return ""
	}
	fullName := fn.Name() // e.g., github.com/example/project/handlers.CreateUser
	return trimFunctionName(fullName)
}

func trimFunctionName(fullName string) string {
	if lastSlash := strings.LastIndex(fullName, "/"); lastSlash != -1 {
		fullName = fullName[lastSlash+1:]
	}
	if lastDot := strings.LastIndex(fullName, "."); lastDot != -1 {
		return fullName[lastDot+1:]
	}
	return fullName
}
