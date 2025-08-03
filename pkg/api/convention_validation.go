package api

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator interface for custom validation.
type Validator interface {
	Validate() error
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

// ConventionValidator handles validation for the Convention Over Configuration approach.
type ConventionValidator struct {
	validator *validator.Validate
}

// NewConventionValidator creates a new convention validator.
func NewConventionValidator() *ConventionValidator {
	return &ConventionValidator{
		validator: NewValidator(DefaultValidatorConfig()),
	}
}

// ValidateRequest validates a request using the Convention Over Configuration approach.
func (v *ConventionValidator) ValidateRequest(reqPtr interface{}) error {
	reqValue := reflect.ValueOf(reqPtr)
	if reqValue.Kind() != reflect.Ptr || reqValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("request must be a pointer to struct")
	}

	reqStruct := reqValue.Elem()
	reqType := reqStruct.Type()
	validationErrors := make(map[string][]string)

	// Step 1: Validate sections
	if err := v.validateSections(reqStruct, reqType, validationErrors); err != nil {
		return err // Server error
	}

	// Step 2: Custom request-level validation
	if err := v.validateRequestLevel(reqPtr, validationErrors); err != nil {
		return err // Server error
	}

	// Step 3: Return validation errors if any
	if len(validationErrors) > 0 {
		return &ValidationErrorResponse{
			Message: "Validation failed",
			Details: validationErrors,
		}
	}

	return nil
}

// validateSections validates all sections in the request.
func (v *ConventionValidator) validateSections(reqStruct reflect.Value, reqType reflect.Type, validationErrors map[string][]string) error {
	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		fieldValue := reqStruct.Field(i)

		if !AllowedSections[field.Name] {
			continue // Skip non-standard sections
		}

		if err := v.validateSection(field, fieldValue, validationErrors); err != nil {
			return err // Server error
		}
	}
	return nil
}

// validateSection validates a single section - simplified according to spec.
func (v *ConventionValidator) validateSection(field reflect.StructField, fieldValue reflect.Value, validationErrors map[string][]string) error {
	sectionName := strings.ToLower(field.Name)

	// Field-level validation for the section using go-playground/validator
	if err := v.validator.Struct(fieldValue.Interface()); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			for _, ve := range verrs {
				// Simple field path: section.field (as per spec)
				fieldPath := fmt.Sprintf("%s.%s", sectionName, ve.Field())
				validationErrors[fieldPath] = append(validationErrors[fieldPath], ve.Tag())
			}
		} else {
			// Non-validation errors are server errors
			return err
		}
	}

	// Custom section-level validation
	if sectionValidator, ok := fieldValue.Interface().(Validator); ok {
		if err := sectionValidator.Validate(); err != nil {
			var valErr ValidationError
			if errors.As(err, &valErr) {
				// Custom validation errors go to the section level
				validationErrors[sectionName] = append(validationErrors[sectionName], valErr.GetErrors()...)
			} else {
				// Non-ValidationError types are server errors (HTTP 500)
				return err
			}
		}
	}

	return nil
}

// validateRequestLevel validates at the request level.
func (v *ConventionValidator) validateRequestLevel(reqPtr interface{}, validationErrors map[string][]string) error {
	if requestValidator, ok := reqPtr.(Validator); ok {
		if err := requestValidator.Validate(); err != nil {
			var valErr ValidationError
			if errors.As(err, &valErr) {
				// Custom validation errors go to the request level
				validationErrors["request"] = append(validationErrors["request"], valErr.GetErrors()...)
			} else {
				// Non-ValidationError types are server errors (HTTP 500)
				return err
			}
		}
	}
	return nil
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
