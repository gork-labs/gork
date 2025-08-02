// Package api provides HTTP handler wrappers and OpenAPI generation capabilities.
package api

import (
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
		} else if idx == 0 && info.Name == "" {
			// no '=' present
			info.Name = p
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
		paramName := resolveQueryParamName(field)
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

func resolveQueryParamName(field reflect.StructField) string {
	// Check openapi tag first
	openapiTag := field.Tag.Get("openapi")
	if openapiTag != "" {
		// Parse openapi tag to check if it's a query parameter
		tagInfo := parseOpenAPITag(openapiTag)
		if tagInfo.In == "query" {
			paramName := tagInfo.Name
			if paramName == "" {
				// If no name override, use json tag
				paramName = field.Tag.Get("json")
			}
			return paramName
		}
		return ""
	}

	// Fall back to json tag for backward compatibility
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return ""
	}
	return jsonTag
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
	return fullName // This is the fallback case we want to test
}
