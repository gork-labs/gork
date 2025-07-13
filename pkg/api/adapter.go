package api

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
)

// HandlerOption represents an option for configuring a handler
type HandlerOption struct {
	Tags     []string
	Security []SecurityRequirement
}

// SecurityRequirement represents a security requirement for an operation
type SecurityRequirement struct {
	Type   string   // "basic", "bearer", "apiKey"
	Scopes []string // For OAuth2
}

// Option is a function that modifies HandlerOption
type Option func(*HandlerOption)

// WithTags adds tags to the handler
func WithTags(tags ...string) Option {
	return func(h *HandlerOption) {
		h.Tags = append(h.Tags, tags...)
	}
}

// WithBasicAuth adds basic authentication requirement
func WithBasicAuth() Option {
	return func(h *HandlerOption) {
		h.Security = append(h.Security, SecurityRequirement{
			Type: "basic",
		})
	}
}

// WithBearerTokenAuth adds bearer token authentication requirement
func WithBearerTokenAuth(scopes ...string) Option {
	return func(h *HandlerOption) {
		h.Security = append(h.Security, SecurityRequirement{
			Type:   "bearer",
			Scopes: scopes,
		})
	}
}

// WithAPIKeyAuth adds API key authentication requirement
func WithAPIKeyAuth() Option {
	return func(h *HandlerOption) {
		h.Security = append(h.Security, SecurityRequirement{
			Type: "apiKey",
		})
	}
}

// HandlerFunc creates an http.HandlerFunc from a typed handler with options
func HandlerFunc[Req any, Resp any](handler func(context.Context, Req) (Resp, error), opts ...Option) http.HandlerFunc {
	// Apply options
	options := &HandlerOption{}
	for _, opt := range opts {
		opt(options)
	}

	// Store options for later extraction by the generator
	handlerMetadata[getFunctionName(handler)] = options

	return func(w http.ResponseWriter, r *http.Request) {
		// Create a new instance of the request type
		req := new(Req)

		// Handle different HTTP methods
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			// Parse query parameters into request struct
			if err := parseQueryParams(r, req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			// Parse JSON body
			if err := json.NewDecoder(r.Body).Decode(req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}
		}

		// Call the handler
		resp, err := handler(r.Context(), *req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to encode response")
		}
	}
}

// handlerMetadata stores metadata for handlers (used by the generator)
var handlerMetadata = make(map[string]*HandlerOption)

// GetHandlerMetadata returns metadata for a handler by name
func GetHandlerMetadata(name string) *HandlerOption {
	return handlerMetadata[name]
}

// Helper functions

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func parseQueryParams(r *http.Request, req interface{}) error {
	// Simple query parameter parsing
	// In a real implementation, this would use reflection to map query params to struct fields
	values := r.URL.Query()
	
	v := reflect.ValueOf(req).Elem()
	t := v.Type()
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		
		// Get the query parameter value
		paramValue := values.Get(jsonTag)
		if paramValue == "" {
			continue
		}
		
		// Set the field value (simplified - only handles strings for now)
		if field.Type.Kind() == reflect.String {
			v.Field(i).SetString(paramValue)
		}
	}
	
	return nil
}

func getFunctionName(i interface{}) string {
	// Get the function name using reflection
	return reflect.ValueOf(i).Type().Name()
}