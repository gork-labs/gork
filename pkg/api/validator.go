package api

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is a globally accessible validator instance configured to use JSON tag names.
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Use JSON tag names in validation errors so that field names match request JSON.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// Prefer explicit name from `openapi` tag if provided.
		if openapiTag := fld.Tag.Get("openapi"); openapiTag != "" {
			info := parseOpenAPITag(openapiTag)
			if info.Name != "" {
				return info.Name
			}
		}

		// Fall back to JSON tag.
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// CheckDiscriminatorErrors inspects v (struct pointer or struct) for fields
// carrying an `openapi:"discriminator=<value>"` tag and returns a map of
// field names -> slice of validation error codes. The returned slice will
// contain either "required" (when the field is empty) or "discriminator"
// (when value does not match the expected discriminator constant).
//
// The map format matches ValidationErrorResponse.Details to plug directly into
// response rendering.
func CheckDiscriminatorErrors(v interface{}) map[string][]string {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Type()
	errs := make(map[string][]string)

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		discVal, ok := parseDiscriminator(field.Tag.Get("openapi"))
		if !ok {
			continue
		}

		jsonName := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if jsonName == "" || jsonName == "-" {
			jsonName = field.Name
		}

		fv := rv.Field(i)

		// Only handle string discriminators for now.
		if fv.Kind() != reflect.String {
			continue
		}

		if fv.String() == "" {
			errs[jsonName] = append(errs[jsonName], "required")
		} else if fv.String() != discVal {
			errs[jsonName] = append(errs[jsonName], "discriminator")
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
