package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/gork-labs/gork/pkg/gorkson"
)

// JSONMarshaler defines the interface for JSON marshaling.
type JSONMarshaler func(v any) ([]byte, error)

// ConventionHandlerFactory creates HTTP handlers using the Convention Over Configuration approach.
type ConventionHandlerFactory struct {
	parser        *ConventionParser
	validator     *ConventionValidator
	gorkMarshaler JSONMarshaler
	stdMarshaler  JSONMarshaler
}

// NewConventionHandlerFactory creates a new convention handler factory.
func NewConventionHandlerFactory() *ConventionHandlerFactory {
	return &ConventionHandlerFactory{
		parser:        NewConventionParser(),
		validator:     NewConventionValidator(),
		gorkMarshaler: gorkson.Marshal,
		stdMarshaler:  json.Marshal,
	}
}

// CreateHandler creates an HTTP handler using the Convention Over Configuration approach.
func (f *ConventionHandlerFactory) CreateHandler(adapter GenericParameterAdapter[*http.Request], handler any, opts ...Option) (http.HandlerFunc, *RouteInfo) {
	v := reflect.ValueOf(handler)
	t := v.Type()

	// Validate handler signature
	validateHandlerSignature(t)

	reqType := t.In(1)
	var respType reflect.Type

	// Handle cases where ResponseType is nil (error-only handlers)
	if t.NumOut() == 2 {
		respType = t.Out(0)
	}
	// For error-only handlers, respType remains nil

	// Prepare options and build RouteInfo
	info := buildRouteInfo(handler, reqType, respType, opts)

	// Build the http.HandlerFunc using Convention Over Configuration
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		f.executeConventionHandler(w, r, v, reqType, adapter)
	}

	return httpHandler, info
}

// executeConventionHandler executes a handler using the Convention Over Configuration approach.
func (f *ConventionHandlerFactory) executeConventionHandler(w http.ResponseWriter, r *http.Request, handlerValue reflect.Value, reqType reflect.Type, adapter GenericParameterAdapter[*http.Request]) {
	// Instantiate request struct
	reqPtr := reflect.New(reqType)

	// Parse request using Convention Over Configuration
	if err := f.parser.ParseRequest(r.Context(), r, reqPtr, adapter); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate request using Convention Over Configuration
	if err := f.validator.ValidateRequest(r.Context(), reqPtr.Interface()); err != nil {
		writeHandlerError(w, err)
		return
	}

	if handlerValue.Type().NumIn() == 3 {
		serveStream(w, r, handlerValue, reqPtr)
		return
	}

	// Call handler and process response
	f.processConventionResponse(w, r, handlerValue, reqPtr)
}

// writeHandlerError writes the response for an error from request validation or from a handler.
func writeHandlerError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	switch {
	case errors.As(err, &httpErr):
		writeError(w, httpErr.Status, httpErr.Message)
	case IsValidationError(err):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(err)
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// processConventionResponse processes the handler response using Convention Over Configuration.
func (f *ConventionHandlerFactory) processConventionResponse(w http.ResponseWriter, r *http.Request, handlerValue reflect.Value, reqPtr reflect.Value) {
	// Call the handler via reflection
	results := handlerValue.Call([]reflect.Value{
		reflect.ValueOf(r.Context()),
		reqPtr.Elem(),
	})

	// Handle different return patterns
	if len(results) == 1 {
		// Error-only handler
		errInterface := results[0].Interface()
		if errInterface != nil {
			if errVal, ok := errInterface.(error); ok {
				writeHandlerError(w, errVal)
				return
			}
		}
		// Success with no content
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Dual return handler (response, error)
	respVal := results[0]
	errInterface := results[1].Interface()

	if errInterface != nil {
		if errVal, ok := errInterface.(error); ok {
			writeHandlerError(w, errVal)
			return
		}
		writeError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	// Process response sections if the response follows Convention Over Configuration
	f.processResponseSections(w, respVal)
}

// processResponseSections processes response sections (Body, Headers, Cookies).
func (f *ConventionHandlerFactory) processResponseSections(w http.ResponseWriter, respVal reflect.Value) {
	// Check if response is nil (only valid for pointer types)
	if respVal.Kind() == reflect.Pointer && respVal.IsNil() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	respStruct, respType := f.extractResponseStructAndType(respVal)
	bodyValue, hasBody := f.processConventionSections(w, respStruct, respType)
	f.writeResponseBody(w, respVal, bodyValue, hasBody)
}

// extractResponseStructAndType extracts the struct and type from response value.
func (f *ConventionHandlerFactory) extractResponseStructAndType(respVal reflect.Value) (reflect.Value, reflect.Type) {
	if respVal.Kind() == reflect.Pointer {
		return respVal.Elem(), respVal.Elem().Type()
	}
	return respVal, respVal.Type()
}

// processConventionSections processes convention sections and returns body value and hasBody flag.
func (f *ConventionHandlerFactory) processConventionSections(w http.ResponseWriter, respStruct reflect.Value, respType reflect.Type) (reflect.Value, bool) {
	var bodyValue reflect.Value
	hasBody := false

	// Only process response sections if it's actually a struct
	if respType.Kind() != reflect.Struct {
		return bodyValue, hasBody
	}

	if !f.hasConventionSections(respType) {
		return bodyValue, hasBody
	}

	// Process response sections for conventional structs
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)
		fieldValue := respStruct.Field(i)

		switch field.Name {
		case "Body":
			bodyValue = fieldValue
			hasBody = true
		case "Headers":
			f.setResponseHeaders(w, fieldValue)
		case "Cookies":
			f.setResponseCookies(w, fieldValue)
		}
	}

	return bodyValue, hasBody
}

// hasConventionSections checks if struct uses Convention Over Configuration sections.
func (f *ConventionHandlerFactory) hasConventionSections(respType reflect.Type) bool {
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)
		if field.Name == "Body" || field.Name == "Headers" || field.Name == "Cookies" {
			return true
		}
	}
	return false
}

// writeResponseBody writes the response body based on whether convention sections are used.
func (f *ConventionHandlerFactory) writeResponseBody(w http.ResponseWriter, respVal reflect.Value, bodyValue reflect.Value, hasBody bool) {
	if hasBody {
		f.writeConventionBody(w, bodyValue)
		return
	}

	f.writeNonConventionBody(w, respVal)
}

// writeConventionBody writes body from convention Body field.
func (f *ConventionHandlerFactory) writeConventionBody(w http.ResponseWriter, bodyValue reflect.Value) {
	w.Header().Set("Content-Type", "application/json")
	data, err := f.gorkMarshaler(bodyValue.Interface())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	_, _ = w.Write(data)
}

// writeNonConventionBody writes non-convention response body.
func (f *ConventionHandlerFactory) writeNonConventionBody(w http.ResponseWriter, respVal reflect.Value) {
	responseInterface := respVal.Interface()
	if _, canMarshal := responseInterface.(json.Marshaler); canMarshal {
		// Response implements json.Marshaler - use standard JSON marshaling
		w.Header().Set("Content-Type", "application/json")
		data, err := f.stdMarshaler(responseInterface)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to encode response")
			return
		}
		_, _ = w.Write(data)
	} else {
		// Non-conventional response without json.Marshaler - return 204 No Content
		w.WriteHeader(http.StatusNoContent)
	}
}

// setResponseHeaders sets HTTP headers from the Headers section.
func (f *ConventionHandlerFactory) setResponseHeaders(w http.ResponseWriter, headersValue reflect.Value) {
	if headersValue.Kind() != reflect.Struct {
		return
	}

	headersType := headersValue.Type()
	for i := 0; i < headersType.NumField(); i++ {
		field := headersType.Field(i)
		fieldValue := headersValue.Field(i)

		gorkTag := field.Tag.Get("gork")
		if gorkTag == "" {
			continue
		}

		headerName := parseGorkTag(gorkTag).Name
		headerValue := f.getStringValue(fieldValue)

		if headerValue != "" {
			w.Header().Set(headerName, headerValue)
		}
	}
}

// setResponseCookies sets HTTP cookies from the Cookies section.
// An http.Cookie or *http.Cookie field is sent unchanged.
func (f *ConventionHandlerFactory) setResponseCookies(w http.ResponseWriter, cookiesValue reflect.Value) {
	if cookiesValue.Kind() != reflect.Struct {
		return
	}

	cookiesType := cookiesValue.Type()
	for i := 0; i < cookiesType.NumField(); i++ {
		field := cookiesType.Field(i)
		fieldValue := cookiesValue.Field(i)

		switch field.Type {
		case reflect.TypeOf((*http.Cookie)(nil)):
			if !fieldValue.IsNil() {
				http.SetCookie(w, fieldValue.Interface().(*http.Cookie))
			}
			continue
		case reflect.TypeOf((*http.Cookie)(nil)).Elem():
			cookie := fieldValue.Interface().(http.Cookie)
			http.SetCookie(w, &cookie)
			continue
		}

		gorkTag := field.Tag.Get("gork")
		if gorkTag == "" {
			continue
		}

		cookieName := parseGorkTag(gorkTag).Name
		cookieValue := f.getStringValue(fieldValue)

		if cookieValue != "" {
			cookie := &http.Cookie{
				Name:     cookieName,
				Value:    cookieValue,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, cookie)
		}
	}
}

// getStringValue converts a reflect.Value to string representation.
// It returns an empty string when the value cannot be formatted.
func (f *ConventionHandlerFactory) getStringValue(value reflect.Value) string {
	formatted, err := gorkson.FormatFieldValueToString(context.Background(), value.Interface())
	if err != nil {
		return ""
	}
	return formatted
}
