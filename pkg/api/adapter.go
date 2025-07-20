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

		// Parse path parameters first (from router context)
		parsePathParams(r, req)
		
		// Parse headers second (for all methods)
		parseHeaders(r, req)
		
		// Handle different HTTP methods
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			// Parse query parameters into request struct
			parseQueryParams(r, req)
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			// Parse JSON body first
			if err := json.NewDecoder(r.Body).Decode(req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}
			// Then parse query parameters (they can override or supplement body params)
			parseQueryParams(r, req)
		default:
			// Also parse query parameters for other methods
			parseQueryParams(r, req)
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

// parseOpenAPITag parses a tag value like `my-field,in=query` or `name=X-API-Key,in=header`.
// This is a simplified version that only extracts name and in values.
func parseOpenAPITag(tag string) struct {
	Name string
	In   string
} {
	var info struct {
		Name string
		In   string
	}
	if tag == "" {
		return info
	}
	parts := strings.Split(tag, ",")
	for idx, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if kv := strings.SplitN(p, "=", 2); len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			switch key {
			case "name":
				info.Name = val
			case "in":
				info.In = val
			}
		} else {
			// no '=' present
			if idx == 0 && info.Name == "" {
				info.Name = p
			}
		}
	}
	return info
}

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
		
		// Check openapi tag first
		openapiTag := field.Tag.Get("openapi")
		var paramName string
		
		if openapiTag != "" {
			// Parse openapi tag to check if it's a query parameter
			tagInfo := parseOpenAPITag(openapiTag)
			if tagInfo.In == "query" {
				paramName = tagInfo.Name
				if paramName == "" {
					// If no name override, use json tag
					paramName = field.Tag.Get("json")
				}
			}
		} else {
			// Fall back to json tag for backward compatibility
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			paramName = jsonTag
		}

		if paramName == "" {
			continue
		}

		paramValue := values.Get(paramName)
		if paramValue == "" {
			continue
		}

		setFieldValue(v.Field(i), field, paramValue, values[paramName])
	}
}

func parseHeaders(r *http.Request, req interface{}) {
	v := reflect.ValueOf(req).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Check openapi tag for header parameters
		openapiTag := field.Tag.Get("openapi")
		if openapiTag == "" {
			continue
		}
		
		tagInfo := parseOpenAPITag(openapiTag)
		if tagInfo.In != "header" {
			continue
		}
		
		headerName := tagInfo.Name
		if headerName == "" {
			// If no name specified, use field name
			headerName = field.Name
		}
		
		headerValue := r.Header.Get(headerName)
		if headerValue == "" {
			continue
		}
		
		setFieldValue(v.Field(i), field, headerValue, []string{headerValue})
	}
}

func parsePathParams(r *http.Request, req interface{}) {
	v := reflect.ValueOf(req).Elem()
	t := v.Type()
	
	// Try to get path parameters from various routers
	pathParams := extractPathParams(r)
	if len(pathParams) == 0 {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Check openapi tag for path parameters
		openapiTag := field.Tag.Get("openapi")
		if openapiTag == "" {
			continue
		}
		
		tagInfo := parseOpenAPITag(openapiTag)
		if tagInfo.In != "path" {
			continue
		}
		
		paramName := tagInfo.Name
		if paramName == "" {
			// If no name specified, use json tag or field name
			paramName = field.Tag.Get("json")
			if paramName == "" || paramName == "-" {
				paramName = field.Name
			}
		}
		
		if paramValue, ok := pathParams[paramName]; ok && paramValue != "" {
			setFieldValue(v.Field(i), field, paramValue, []string{paramValue})
		}
	}
}

// extractPathParams tries to extract path parameters from various router implementations
func extractPathParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	
	// Try Chi router (github.com/go-chi/chi/v5)
	if rctx := r.Context().Value(chiRouteCtxKey); rctx != nil {
		if urlParams, ok := rctx.(*chiRouteContext); ok && urlParams.URLParams != nil {
			for i, key := range urlParams.URLParams.Keys {
				if i < len(urlParams.URLParams.Values) {
					params[key] = urlParams.URLParams.Values[i]
				}
			}
		}
	}
	
	// Try Gorilla Mux (github.com/gorilla/mux)
	if vars, ok := r.Context().Value(varsKey).(map[string]string); ok {
		for k, v := range vars {
			params[k] = v
		}
	}
	
	// Try Echo framework (github.com/labstack/echo/v4)
	if echoCtx := r.Context().Value(echoContextKey); echoCtx != nil {
		// Use reflection to safely access echo.Context methods
		ctx := reflect.ValueOf(echoCtx)
		if ctx.IsValid() && !ctx.IsNil() {
			if paramMethod := ctx.MethodByName("Param"); paramMethod.IsValid() {
				// Get all param names from the route
				if paramsMethod := ctx.MethodByName("ParamNames"); paramsMethod.IsValid() {
					if names := paramsMethod.Call(nil); len(names) > 0 {
						if nameSlice, ok := names[0].Interface().([]string); ok {
							for _, name := range nameSlice {
								if result := paramMethod.Call([]reflect.Value{reflect.ValueOf(name)}); len(result) > 0 {
									if val, ok := result[0].Interface().(string); ok && val != "" {
										params[name] = val
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	// Try Gin framework (github.com/gin-gonic/gin)
	if ginCtx := r.Context().Value(ginContextKey); ginCtx != nil {
		ctx := reflect.ValueOf(ginCtx)
		if ctx.IsValid() && !ctx.IsNil() {
			if paramMethod := ctx.MethodByName("Param"); paramMethod.IsValid() {
				// Gin stores params differently, we need to access the Params field
				if paramsField := ctx.Elem().FieldByName("Params"); paramsField.IsValid() {
					// Params is a slice of Param structs
					for i := 0; i < paramsField.Len(); i++ {
						param := paramsField.Index(i)
						if keyField := param.FieldByName("Key"); keyField.IsValid() {
							if valueField := param.FieldByName("Value"); valueField.IsValid() {
								key := keyField.String()
								value := valueField.String()
								if key != "" && value != "" {
									params[key] = value
								}
							}
						}
					}
				}
			}
		}
	}
	
	return params
}

// Context keys for various routers
var (
	chiRouteCtxKey = &contextKey{"RouteContext"}
	varsKey        = &contextKey{"vars"}           // Gorilla mux
	echoContextKey = &contextKey{"echo"}          
	ginContextKey  = &contextKey{"gin"}           
)

type contextKey struct {
	name string
}

// Chi router context structure (simplified)
type chiRouteContext struct {
	URLParams struct {
		Keys   []string
		Values []string
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
