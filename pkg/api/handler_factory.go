package api

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// createHandlerFromAny validates the provided handler, wraps it in an
// http.HandlerFunc that performs request deserialization/parameter extraction,
// and constructs a corresponding RouteInfo structure.
func createHandlerFromAny(adapter ParameterAdapter, handler interface{}, opts ...Option) (http.HandlerFunc, *RouteInfo) {
	v := reflect.ValueOf(handler)
	if v.Kind() != reflect.Func {
		panic("handler must be a function")
	}

	t := v.Type()
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

	reqType := t.In(1)
	respType := t.Out(0)

	// Prepare options.
	optionCfg := &HandlerOption{}
	for _, o := range opts {
		o(optionCfg)
	}

	// Build RouteInfo now so that the caller can enrich it further.
	info := &RouteInfo{
		Handler:      handler,
		HandlerName:  getFunctionName(handler),
		RequestType:  reqType,
		ResponseType: respType,
		Options:      optionCfg,
	}

	// Build the http.HandlerFunc. We copy the logic from HandlerFunc in
	// adapter.go but rely on reflection to call the typed handler.
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		// Instantiate request struct.
		reqPtr := reflect.New(reqType)

		// Parse path parameters using adapter if provided; fall back to generic helper.
		if adapter != nil {
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
				name := tagInfo.Name
				if name == "" {
					name = field.Tag.Get("json")
					if name == "" || name == "-" {
						name = field.Name
					}
				}
				if val, ok := adapter.Path(r, name); ok {
					setFieldValue(vStruct.Field(i), field, val, []string{val})
				}
			}
		}

		// Decode JSON body for methods that typically carry one
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if err := json.NewDecoder(r.Body).Decode(reqPtr.Interface()); err != nil {
				writeError(w, http.StatusUnprocessableEntity, "Unable to parse request body")
				return
			}
		}

		// Populate query, header, cookie parameters using the adapter
		if adapter != nil {
			vStruct := reqPtr.Elem()
			tStruct := vStruct.Type()
			for i := 0; i < tStruct.NumField(); i++ {
				field := tStruct.Field(i)
				openapiTag := field.Tag.Get("openapi")
				if openapiTag == "" {
					continue
				}
				tagInfo := parseOpenAPITag(openapiTag)
				var val string
				var ok bool
				switch tagInfo.In {
				case "query":
					name := tagInfo.Name
					if name == "" {
						name = field.Tag.Get("json")
					}
					val, ok = adapter.Query(r, name)
				case "header":
					name := tagInfo.Name
					if name == "" {
						name = field.Tag.Get("json")
					}
					val, ok = adapter.Header(r, name)
				case "cookie":
					name := tagInfo.Name
					if name == "" {
						name = field.Tag.Get("json")
					}
					val, ok = adapter.Cookie(r, name)
				}
				if ok {
					setFieldValue(vStruct.Field(i), field, val, []string{val})
				}
			}
		}

		// Custom discriminator validation prior to running validator package rules.
		discErrs := CheckDiscriminatorErrors(reqPtr.Interface())
		if len(discErrs) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ValidationErrorResponse{
				Error:   "Validation failed",
				Details: discErrs,
			})
			return
		}

		// Validate the fully populated request struct
		if err := validate.Struct(reqPtr.Interface()); err != nil {
			// Convert validation errors to detailed map
			validationErrors := make(map[string][]string)
			if verrs, ok := err.(validator.ValidationErrors); ok {
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
			return
		}

		// Call the handler via reflection.
		results := v.Call([]reflect.Value{
			reflect.ValueOf(r.Context()),
			reqPtr.Elem(),
		})

		// Extract response and error.
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

	return httpHandler, info
}
