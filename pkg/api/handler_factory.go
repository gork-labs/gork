package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// createHandlerFromAny validates the provided handler, wraps it in an
// http.HandlerFunc that performs request deserialization/parameter extraction,
// and constructs a corresponding RouteInfo structure.
func createHandlerFromAny(adapter ParameterAdapter, handler interface{}, opts ...Option) (http.HandlerFunc, *RouteInfo) {
	v := reflect.ValueOf(handler)
	t := v.Type()

	// Validate handler signature
	validateHandlerSignature(t)

	reqType := t.In(1)
	respType := t.Out(0)

	// Prepare options and build RouteInfo
	info := buildRouteInfo(handler, reqType, respType, opts)

	// Build the http.HandlerFunc
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		executeHandler(w, r, v, reqType, adapter)
	}

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
	if t.NumOut() != 2 {
		panic("handler must return (Response, error)")
	}
	if !t.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("second handler return value must be error")
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

func executeHandler(w http.ResponseWriter, r *http.Request, handlerValue reflect.Value, reqType reflect.Type, adapter ParameterAdapter) {
	// Instantiate request struct
	reqPtr := reflect.New(reqType)

	// Process request parameters
	if err := processRequestParameters(reqPtr, r, adapter); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Validate request
	if err := validateRequest(w, reqPtr.Interface()); err != nil {
		return // Error already written to response
	}

	// Call handler and process response
	processHandlerResponse(w, r, handlerValue, reqPtr)
}

func processRequestParameters(reqPtr reflect.Value, r *http.Request, adapter ParameterAdapter) error {
	// Parse path parameters
	if adapter != nil {
		parsePathParameters(reqPtr, r, adapter)
	}

	// Decode JSON body for methods that typically carry one
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		if err := json.NewDecoder(r.Body).Decode(reqPtr.Interface()); err != nil {
			return errors.New("unable to parse request body")
		}
	}

	// Parse query, header, cookie parameters
	if adapter != nil {
		parseOtherParameters(reqPtr, r, adapter)
	}

	return nil
}

func parsePathParameters(reqPtr reflect.Value, r *http.Request, adapter ParameterAdapter) {
	vStruct := reqPtr.Elem()
	tStruct := vStruct.Type()
	for i := 0; i < tStruct.NumField(); i++ {
		field := tStruct.Field(i)
		openapiTag := field.Tag.Get("openapi")
		if openapiTag == "" {
			continue
		}
		tagInfo := parseOpenAPITag(openapiTag)
		if tagInfo.In != "path" {
			continue
		}
		name := getParameterName(tagInfo, field)
		if val, ok := adapter.Path(r, name); ok {
			setFieldValue(vStruct.Field(i), field, val, []string{val})
		}
	}
}

func parseOtherParameters(reqPtr reflect.Value, r *http.Request, adapter ParameterAdapter) {
	vStruct := reqPtr.Elem()
	tStruct := vStruct.Type()
	for i := 0; i < tStruct.NumField(); i++ {
		field := tStruct.Field(i)
		openapiTag := field.Tag.Get("openapi")
		if openapiTag == "" {
			continue
		}
		tagInfo := parseOpenAPITag(openapiTag)
		name := getParameterName(tagInfo, field)

		var val string
		var ok bool
		switch tagInfo.In {
		case "query":
			val, ok = adapter.Query(r, name)
		case "header":
			val, ok = adapter.Header(r, name)
		case "cookie":
			val, ok = adapter.Cookie(r, name)
		}
		if ok {
			setFieldValue(vStruct.Field(i), field, val, []string{val})
		}
	}
}

func getParameterName(tagInfo struct{ Name, In string }, field reflect.StructField) string {
	name := tagInfo.Name
	if name == "" {
		name = field.Tag.Get("json")
		if name == "" || name == "-" {
			name = field.Name
		}
	}
	return name
}

func validateRequest(w http.ResponseWriter, req interface{}) error {
	// Custom discriminator validation
	discErrs := CheckDiscriminatorErrors(req)
	if len(discErrs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ValidationErrorResponse{
			Error:   "Validation failed",
			Details: discErrs,
		})
		return errors.New("discriminator validation failed")
	}

	// Standard validation
	if err := validate.Struct(req); err != nil {
		validationErrors := make(map[string][]string)
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			for _, ve := range verrs {
				field := ve.Field()
				validationErrors[field] = append(validationErrors[field], ve.Tag())
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ValidationErrorResponse{
			Error:   "Validation failed",
			Details: validationErrors,
		})
		return errors.New("validation failed")
	}

	return nil
}

func processHandlerResponse(w http.ResponseWriter, r *http.Request, handlerValue reflect.Value, reqPtr reflect.Value) {
	// Call the handler via reflection
	results := handlerValue.Call([]reflect.Value{
		reflect.ValueOf(r.Context()),
		reqPtr.Elem(),
	})

	// Extract response and error
	respVal := results[0]
	errInterface := results[1].Interface()

	if errInterface != nil {
		if errVal, ok := errInterface.(error); ok {
			writeError(w, http.StatusInternalServerError, errVal.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(respVal.Interface()); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}
