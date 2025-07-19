package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
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

// HandlerFunc creates an http.HandlerFunc from a typed handler with options.
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
			parseQueryParams(r, req)
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

// handlerMetadata stores metadata for handlers (used by the generator).
var handlerMetadata = make(map[string]*HandlerOption)

// GetHandlerMetadata returns metadata for a handler by name.
func GetHandlerMetadata(name string) *HandlerOption {
	return handlerMetadata[name]
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

func parseQueryParams(r *http.Request, req interface{}) {
	values := r.URL.Query()
	v := reflect.ValueOf(req).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		paramValue := values.Get(jsonTag)
		if paramValue == "" {
			continue
		}

		setFieldValue(v.Field(i), field, paramValue, values[jsonTag])
	}
}

func setFieldValue(fieldValue reflect.Value, fieldType reflect.StructField, paramValue string, allValues []string) {
	switch fieldType.Type.Kind() {
	case reflect.String:
		fieldValue.SetString(paramValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if iv, err := strconv.ParseInt(paramValue, 10, 64); err == nil {
			fieldValue.SetInt(iv)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uv, err := strconv.ParseUint(paramValue, 10, 64); err == nil {
			fieldValue.SetUint(uv)
		}
	case reflect.Bool:
		if bv, err := strconv.ParseBool(paramValue); err == nil {
			fieldValue.SetBool(bv)
		}
	case reflect.Float32, reflect.Float64:
		if fv, err := strconv.ParseFloat(paramValue, 64); err == nil {
			fieldValue.SetFloat(fv)
		}
	case reflect.Slice:
		setSliceFieldValue(fieldValue, fieldType, paramValue, allValues)
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Ptr, reflect.Struct, reflect.UnsafePointer:
		// Unsupported types are silently ignored
	}
}

func setSliceFieldValue(fieldValue reflect.Value, fieldType reflect.StructField, paramValue string, allValues []string) {
	if fieldType.Type.Elem().Kind() != reflect.String {
		return
	}

	parts := allValues
	if len(parts) == 0 && paramValue != "" {
		parts = strings.Split(paramValue, ",")
	} else if len(parts) == 1 && strings.Contains(parts[0], ",") {
		parts = strings.Split(parts[0], ",")
	}

	if len(parts) > 0 {
		sliceVal := reflect.MakeSlice(fieldType.Type, len(parts), len(parts))
		for idx, p := range parts {
			sliceVal.Index(idx).SetString(p)
		}
		fieldValue.Set(sliceVal)
	}
}

func getFunctionName(i interface{}) string {
	// Use FuncForPC to get the fully-qualified function name, then trim the package path
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer())
	if fn == nil {
		return ""
	}
	fullName := fn.Name() // e.g., github.com/example/project/handlers.CreateUser
	if lastSlash := strings.LastIndex(fullName, "/"); lastSlash != -1 {
		fullName = fullName[lastSlash+1:]
	}
	if lastDot := strings.LastIndex(fullName, "."); lastDot != -1 {
		return fullName[lastDot+1:]
	}
	return fullName
}
