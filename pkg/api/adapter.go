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
	Tags           []string
	Security       []SecurityRequirement
	ErrorResponses []int
	Status         int

	ResponseContentTypes []string
}

// SecurityRequirement represents a security requirement for an operation.
type SecurityRequirement struct {
	Type   string   // "basic", "bearer", "apiKey", "cookie"
	Scopes []string // For OAuth2
	Name   string   // Cookie name for "cookie"
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

// WithCookieAuth adds an API key requirement that the client sends in the cookie with this name.
func WithCookieAuth(name string) Option {
	return func(h *HandlerOption) {
		h.Security = append(h.Security, SecurityRequirement{
			Type: "cookie",
			Name: name,
		})
	}
}

// WithErrorResponses adds responses with the ErrorResponse schema for these
// HTTP statuses to the OpenAPI operation. A handler sends them with HTTPError.
func WithErrorResponses(statuses ...int) Option {
	return func(h *HandlerOption) {
		h.ErrorResponses = append(h.ErrorResponses, statuses...)
	}
}

// WithStatus sets the HTTP status of a successful response. Without this
// option, Gork sends 200 for a response with a Body and 204 for a response
// without a Body. The OpenAPI operation shows the success response with this status.
func WithStatus(status int) Option {
	return func(h *HandlerOption) {
		h.Status = status
	}
}

// WithResponseContentTypes sets the media types of a Binary success response in the
// OpenAPI operation. Without this option, the operation shows application/octet-stream.
func WithResponseContentTypes(types ...string) Option {
	return func(h *HandlerOption) {
		h.ResponseContentTypes = append(h.ResponseContentTypes, types...)
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

// trimFunctionName returns the function or method name. The runtime name of a
// method value such as h.List ends with "-fm".
func trimFunctionName(fullName string) string {
	fullName = strings.TrimSuffix(fullName, "-fm")
	if lastSlash := strings.LastIndex(fullName, "/"); lastSlash != -1 {
		fullName = fullName[lastSlash+1:]
	}
	if lastDot := strings.LastIndex(fullName, "."); lastDot != -1 {
		return fullName[lastDot+1:]
	}
	return fullName
}
