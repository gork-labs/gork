package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

// JSONEncoder interface allows dependency injection for testing.
type JSONEncoder interface {
	Encode(v interface{}) error
}

// JSONEncoderFactory creates JSON encoders.
type JSONEncoderFactory interface {
	NewEncoder(w io.Writer) JSONEncoder
}

// defaultJSONEncoderFactory implements JSONEncoderFactory using standard library.
type defaultJSONEncoderFactory struct{}

func (f defaultJSONEncoderFactory) NewEncoder(w io.Writer) JSONEncoder {
	return json.NewEncoder(w)
}

// createHandlerFromAny validates the provided handler, wraps it in an
// http.HandlerFunc that performs request deserialization/parameter extraction,
// and constructs a corresponding RouteInfo structure using Convention Over Configuration.
func createHandlerFromAny(adapter GenericParameterAdapter[*http.Request], handler interface{}, opts ...Option) (http.HandlerFunc, *RouteInfo) {
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

	// Use Convention Over Configuration handler factory
	factory := NewConventionHandlerFactory()
	httpHandler, _ := factory.CreateHandler(adapter, handler, opts...)
	return httpHandler, info
}

func validateHandlerSignature(t reflect.Type) {
	if t.Kind() != reflect.Func {
		panic("handler must be a function")
	}
	if t.NumIn() != 2 {
		panic("handler must accept exactly 2 parameters (context.Context, Request)")
	}
	if !t.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		panic("first handler parameter must be context.Context")
	}

	// Allow either (ResponseType, error) or (error) returns
	numOut := t.NumOut()
	if numOut != 1 && numOut != 2 {
		panic("handler must return either (error) or (*ResponseType, error)")
	}

	// Last return must be error
	lastOut := t.Out(numOut - 1)
	if !lastOut.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("last return value must be error")
	}

	// If two returns, first must be struct or pointer to struct
	if numOut == 2 {
		firstOut := t.Out(0)
		if firstOut.Kind() == reflect.Ptr {
			// Pointer to struct
			if firstOut.Elem().Kind() != reflect.Struct {
				panic("response type must be struct or pointer to struct")
			}
		} else if firstOut.Kind() != reflect.Struct {
			// Value must be struct
			panic("response type must be struct or pointer to struct")
		}
	}
}

func buildRouteInfo(handler interface{}, reqType, respType reflect.Type, opts []Option) *RouteInfo {
	// Prepare options.
	optionCfg := &HandlerOption{}
	for _, o := range opts {
		o(optionCfg)
	}

	return &RouteInfo{
		Handler:      handler,
		HandlerName:  getFunctionName(handler),
		RequestType:  reqType,
		ResponseType: respType,
		Options:      optionCfg,
	}
}

// validateBodyUsageForMethod checks that Body sections are not used with read-only HTTP methods.
func validateBodyUsageForMethod(method string, reqType reflect.Type) {
	// Check if this is a read-only HTTP method
	readOnlyMethods := map[string]bool{
		"GET":     true,
		"HEAD":    true,
		"OPTIONS": true,
	}

	if !readOnlyMethods[method] {
		return // Method allows body, no validation needed
	}

	// For read-only methods, check if the request type has a Body field
	if reqType == nil {
		return // No request type, no validation needed
	}

	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}

	if reqType.Kind() != reflect.Struct {
		return // Not a struct, no Body field possible
	}

	// Check for Body field
	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		if field.Name == "Body" {
			panic(fmt.Sprintf("Handler for %s method cannot have a Body section. Read-only HTTP methods (GET, HEAD, OPTIONS) should use Path, Query, or Headers sections instead.", method))
		}
	}
}
