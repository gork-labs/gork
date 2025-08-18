package api

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	rules "github.com/gork-labs/gork/pkg/rules"
)

// Validator interface for custom validation.
type Validator interface {
	Validate() error
}

// ContextValidator interface for custom validation with access to context.
// Useful for webhook validation that needs to access configured secrets from context.
type ContextValidator interface {
	Validate(ctx context.Context) error
}

// ValidationError represents any validation error that should return HTTP 400.
// These errors contain client-side validation issues.
type ValidationError interface {
	error
	GetErrors() []string
}

// RequestValidationError represents validation errors that occur at the request level.
type RequestValidationError struct {
	Errors []string `json:"errors"`
}

func (e *RequestValidationError) Error() string {
	return fmt.Sprintf("request validation failed: %s", strings.Join(e.Errors, ", "))
}

// GetErrors returns the validation errors for the request.
func (e *RequestValidationError) GetErrors() []string {
	return e.Errors
}

// BodyValidationError represents validation errors that occur in the request body.
type BodyValidationError struct {
	Errors []string `json:"errors"`
}

func (e *BodyValidationError) Error() string {
	return fmt.Sprintf("body validation failed: %s", strings.Join(e.Errors, ", "))
}

// GetErrors returns the validation errors for the body.
func (e *BodyValidationError) GetErrors() []string {
	return e.Errors
}

// QueryValidationError represents validation errors that occur in query parameters.
type QueryValidationError struct {
	Errors []string `json:"errors"`
}

func (e *QueryValidationError) Error() string {
	return fmt.Sprintf("query validation failed: %s", strings.Join(e.Errors, ", "))
}

// GetErrors returns the validation errors for the query parameters.
func (e *QueryValidationError) GetErrors() []string {
	return e.Errors
}

// PathValidationError represents validation errors that occur in path parameters.
type PathValidationError struct {
	Errors []string `json:"errors"`
}

func (e *PathValidationError) Error() string {
	return fmt.Sprintf("path validation failed: %s", strings.Join(e.Errors, ", "))
}

// GetErrors returns the validation errors for the path parameters.
func (e *PathValidationError) GetErrors() []string {
	return e.Errors
}

// HeadersValidationError represents validation errors that occur in request headers.
type HeadersValidationError struct {
	Errors []string `json:"errors"`
}

func (e *HeadersValidationError) Error() string {
	return fmt.Sprintf("headers validation failed: %s", strings.Join(e.Errors, ", "))
}

// GetErrors returns the validation errors for the headers.
func (e *HeadersValidationError) GetErrors() []string {
	return e.Errors
}

// CookiesValidationError represents validation errors that occur in request cookies.
type CookiesValidationError struct {
	Errors []string `json:"errors"`
}

func (e *CookiesValidationError) Error() string {
	return fmt.Sprintf("cookies validation failed: %s", strings.Join(e.Errors, ", "))
}

// GetErrors returns the validation errors for the cookies.
func (e *CookiesValidationError) GetErrors() []string {
	return e.Errors
}

// FieldValidator interface abstracts the go-playground/validator functionality for testing.
type FieldValidator interface {
	Var(field interface{}, tag string) error
	Struct(s interface{}) error
}

// GoPlaygroundValidator wraps the go-playground/validator to implement FieldValidator.
type GoPlaygroundValidator struct {
	validator *validator.Validate
}

// Var validates a single field using the provided validation tag.
func (g *GoPlaygroundValidator) Var(field interface{}, tag string) error {
	return g.validator.Var(field, tag)
}

// Struct validates all fields in a struct using their validation tags.
func (g *GoPlaygroundValidator) Struct(s interface{}) error {
	return g.validator.Struct(s)
}

// ConventionValidator handles validation for the Convention Over Configuration approach.
type ConventionValidator struct {
	validator      *validator.Validate
	fieldValidator FieldValidator
	applyRulesFunc func(ctx context.Context, reqPtr interface{}) []error
}

// NewConventionValidator creates a new convention validator.
func NewConventionValidator() *ConventionValidator {
	v := NewValidator(DefaultValidatorConfig())
	return &ConventionValidator{
		validator:      v,
		fieldValidator: &GoPlaygroundValidator{validator: v},
		applyRulesFunc: rules.Apply,
	}
}

// NewConventionValidatorWithFieldValidator creates a new convention validator with a custom field validator.
// This is primarily used for testing to inject mock validators.
func NewConventionValidatorWithFieldValidator(fieldValidator FieldValidator) *ConventionValidator {
	return &ConventionValidator{
		validator:      NewValidator(DefaultValidatorConfig()),
		fieldValidator: fieldValidator,
		applyRulesFunc: rules.Apply,
	}
}

// GetValidator returns the underlying validator for testing purposes.
// This method should only be used in tests.
func (v *ConventionValidator) GetValidator() *validator.Validate {
	return v.validator
}

// ValidateRequest validates a request using the Convention Over Configuration approach.
func (v *ConventionValidator) ValidateRequest(ctx context.Context, reqPtr interface{}) error {
	reqValue := reflect.ValueOf(reqPtr)
	if reqValue.Kind() != reflect.Ptr || reqValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("request must be a pointer to struct")
	}

	reqStruct := reqValue.Elem()
	reqType := reqStruct.Type()
	validationErrors := make(map[string][]string)

	// Step 1: Validate sections
	if err := v.validateSections(ctx, reqStruct, reqType, validationErrors); err != nil {
		return err // Server error
	}

	// Step 2: Custom request-level validation
	if err := v.validateRequestLevel(ctx, reqPtr, validationErrors); err != nil {
		return err // Server error
	}

	// Step 3: Apply rule-based validation (rules engine)
	applyFn := v.applyRulesFunc
	if applyFn == nil {
		applyFn = rules.Apply
	}
	if rerrs := applyFn(ctx, reqPtr); len(rerrs) > 0 {
		for _, re := range rerrs {
			// Distinguish validation vs server errors
			var valErr ValidationError
			if errors.As(re, &valErr) {
				validationErrors["request"] = append(validationErrors["request"], valErr.Error())
				continue
			}
			// Check for RuleValidationError specifically (since it no longer implements ValidationError)
			var ruleErr *rules.RuleValidationError
			if errors.As(re, &ruleErr) {
				validationErrors["request"] = append(validationErrors["request"], ruleErr.Error())
				continue
			}
			// Server error from a rule -> short-circuit
			return re
		}
	}

	// Step 4: Return validation errors if any
	if len(validationErrors) > 0 {
		return &ValidationErrorResponse{
			Message: "Validation failed",
			Details: validationErrors,
		}
	}

	return nil
}

// validateSections validates all sections in the request.
func (v *ConventionValidator) validateSections(ctx context.Context, reqStruct reflect.Value, reqType reflect.Type, validationErrors map[string][]string) error {
	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		fieldValue := reqStruct.Field(i)

		if !AllowedSections[field.Name] {
			continue // Skip non-standard sections
		}

		if err := v.validateSection(ctx, field, fieldValue, validationErrors); err != nil {
			return err // Server error
		}
	}
	return nil
}

// validateSection validates a single section - simplified according to spec.
func (v *ConventionValidator) validateSection(ctx context.Context, field reflect.StructField, fieldValue reflect.Value, validationErrors map[string][]string) (err error) {
	sectionName := strings.ToLower(field.Name)

	// Field-level validation for the section using go-playground/validator
	if err := v.validateFieldLevel(field, fieldValue, sectionName, validationErrors); err != nil {
		return err
	}

	// Custom section-level validation (context-aware or regular)
	return v.validateCustomLevel(ctx, fieldValue, sectionName, validationErrors)
}

// validateFieldLevel handles field-level validation using go-playground/validator.
func (v *ConventionValidator) validateFieldLevel(field reflect.StructField, fieldValue reflect.Value, sectionName string, validationErrors map[string][]string) error {
	if v.isByteSliceBodyField(field, fieldValue) {
		return v.validateByteSliceField(field, fieldValue, sectionName, validationErrors)
	}

	return v.validateStructField(fieldValue, sectionName, validationErrors)
}

// isByteSliceBodyField checks if this is a []byte Body field (webhook).
func (v *ConventionValidator) isByteSliceBodyField(field reflect.StructField, fieldValue reflect.Value) bool {
	return field.Name == "Body" &&
		fieldValue.Kind() == reflect.Slice &&
		fieldValue.Type().Elem().Kind() == reflect.Uint8
}

// validateByteSliceField validates []byte Body fields using Var() method.
func (v *ConventionValidator) validateByteSliceField(field reflect.StructField, fieldValue reflect.Value, sectionName string, validationErrors map[string][]string) error {
	tag := field.Tag.Get("validate")
	if tag == "" {
		return nil
	}

	validationErr := v.fieldValidator.Var(fieldValue.Interface(), tag)
	if validationErr == nil {
		return nil
	}

	var verrs validator.ValidationErrors
	if errors.As(validationErr, &verrs) {
		for _, ve := range verrs {
			// Aggregate under the section name (e.g., "body") since there is no nested field
			validationErrors[sectionName] = append(validationErrors[sectionName], ve.Tag())
		}
		return nil
	}

	return validationErr
}

// validateStructField validates regular struct fields.
func (v *ConventionValidator) validateStructField(fieldValue reflect.Value, sectionName string, validationErrors map[string][]string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("validation panic: %v", r)
		}
	}()

	validationErr := v.fieldValidator.Struct(fieldValue.Interface())
	if validationErr == nil {
		return nil
	}

	var verrs validator.ValidationErrors
	if errors.As(validationErr, &verrs) {
		for _, ve := range verrs {
			// Simple field path: section.field (as per spec)
			fieldPath := fmt.Sprintf("%s.%s", sectionName, ve.Field())
			validationErrors[fieldPath] = append(validationErrors[fieldPath], ve.Tag())
		}
		return nil
	}

	// Non-validation errors are server errors
	return validationErr
}

// validateCustomLevel handles custom section-level validation.
func (v *ConventionValidator) validateCustomLevel(ctx context.Context, fieldValue reflect.Value, sectionName string, validationErrors map[string][]string) error {
	verrs, serverErr := invokeCustomValidation(ctx, fieldValue.Interface())
	if serverErr != nil {
		return serverErr
	}

	if len(verrs) > 0 {
		validationErrors[sectionName] = append(validationErrors[sectionName], verrs...)
	}

	return nil
}

// validateRequestLevel validates at the request level.
func (v *ConventionValidator) validateRequestLevel(ctx context.Context, reqPtr interface{}, validationErrors map[string][]string) error {
	if verrs, serverErr := invokeCustomValidation(ctx, reqPtr); serverErr != nil {
		return serverErr
	} else if len(verrs) > 0 {
		validationErrors["request"] = append(validationErrors["request"], verrs...)
	}
	return nil
}

// invokeCustomValidation runs either context-aware or regular validation, normalizing outputs.
// Returns a slice of field-agnostic validation error strings (client errors) or a server error.
func invokeCustomValidation(ctx context.Context, v interface{}) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	if cv, ok := v.(ContextValidator); ok {
		if err := cv.Validate(ctx); err != nil {
			var valErr ValidationError
			if errors.As(err, &valErr) {
				return valErr.GetErrors(), nil
			}
			return nil, err
		}
		return nil, nil
	}
	if rv, ok := v.(Validator); ok {
		if err := rv.Validate(); err != nil {
			var valErr ValidationError
			if errors.As(err, &valErr) {
				return valErr.GetErrors(), nil
			}
			return nil, err
		}
	}
	return nil, nil
}

// IsValidationError checks if an error is a client validation error (HTTP 400).
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	// Check for ValidationErrorResponse
	var verr *ValidationErrorResponse
	if errors.As(err, &verr) {
		return true
	}

	// Check for any ValidationError interface
	var valErr ValidationError
	return errors.As(err, &valErr)
}
