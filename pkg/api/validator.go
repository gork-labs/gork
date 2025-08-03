package api

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// ValidatorConfig allows for dependency injection of validator behavior.
type ValidatorConfig struct {
	TagNameFunc func(reflect.StructField) string
}

// DefaultValidatorConfig returns the default validator configuration.
func DefaultValidatorConfig() ValidatorConfig {
	return ValidatorConfig{
		TagNameFunc: defaultTagNameFunc,
	}
}

// defaultTagNameFunc is the default tag name function.
func defaultTagNameFunc(fld reflect.StructField) string {
	// Use gork tag for field naming
	if gorkTag := fld.Tag.Get("gork"); gorkTag != "" {
		tagInfo := parseGorkTag(gorkTag)
		if tagInfo.Name != "" {
			return tagInfo.Name
		}
	}

	// Fall back to field name if no gork tag
	return fld.Name
}

// NewValidator creates a new validator instance with the given configuration.
func NewValidator(config ValidatorConfig) *validator.Validate {
	v := validator.New()
	if config.TagNameFunc != nil {
		v.RegisterTagNameFunc(config.TagNameFunc)
	}
	return v
}

// CheckDiscriminatorErrors inspects v (struct pointer or struct) for fields
// carrying a `gork:"field_name,discriminator=<value>"` tag and returns a map of
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
		gorkTag := field.Tag.Get("gork")
		discVal, ok := parseDiscriminator(gorkTag)
		if !ok {
			continue
		}

		// Parse gork tag to get field name
		tagInfo := parseGorkTag(gorkTag)
		fieldName := tagInfo.Name
		if fieldName == "" {
			fieldName = field.Name
		}

		fv := rv.Field(i)

		// Only handle string discriminators for now.
		if fv.Kind() != reflect.String {
			continue
		}

		if fv.String() == "" {
			errs[fieldName] = append(errs[fieldName], "required")
		} else if fv.String() != discVal {
			errs[fieldName] = append(errs[fieldName], "discriminator")
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
