package api

import (
	"context"
	"testing"
)

// Test section that implements Validator and returns ValidationError
type TestSectionWithValidationError struct {
	Data string `gork:"data"`
}

func (s TestSectionWithValidationError) Validate() error {
	if s.Data == "invalid" {
		// Return a ValidationError - this should be handled by the uncovered lines
		return &BodyValidationError{
			Errors: []string{"custom section validation failed", "data cannot be 'invalid'"},
		}
	}
	return nil
}

// Test request with section that returns ValidationError
type TestRequestWithSectionValidationError struct {
	Body TestSectionWithValidationError
}

// Test section for Query that returns ValidationError
type TestQuerySectionWithValidationError struct {
	Term string `gork:"term"`
}

func (s TestQuerySectionWithValidationError) Validate() error {
	if s.Term == "forbidden" {
		return &QueryValidationError{
			Errors: []string{"term 'forbidden' is not allowed"},
		}
	}
	return nil
}

type TestRequestWithQueryValidationError struct {
	Query TestQuerySectionWithValidationError
}

func TestValidateSectionValidationError(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("section returns ValidationError - body section", func(t *testing.T) {
		// This tests the uncovered lines 180-182 in validateSection:
		// if valErr, ok := err.(ValidationError); ok {
		//     validationErrors[sectionName] = append(validationErrors[sectionName], valErr.GetErrors()...)
		// }

		req := &TestRequestWithSectionValidationError{
			Body: TestSectionWithValidationError{
				Data: "invalid", // This will trigger the ValidationError
			},
		}

		err := validator.ValidateRequest(context.Background(), req)

		// Should return ValidationErrorResponse with section-level errors
		if err == nil {
			t.Error("Expected validation error")
		} else {
			if valErr, ok := err.(*ValidationErrorResponse); ok {
				// Check that the errors are added at the section level ("body")
				if bodyErrors, exists := valErr.Details["body"]; !exists {
					t.Error("Expected 'body' section errors in validation response")
				} else {
					if len(bodyErrors) < 2 {
						t.Errorf("Expected at least 2 errors in body section, got %d", len(bodyErrors))
					}

					// Check that the custom validation errors are included
					foundFirst := false
					foundSecond := false
					for _, errMsg := range bodyErrors {
						if errMsg == "custom section validation failed" {
							foundFirst = true
						}
						if errMsg == "data cannot be 'invalid'" {
							foundSecond = true
						}
					}

					if !foundFirst {
						t.Error("Expected 'custom section validation failed' error in body section")
					}
					if !foundSecond {
						t.Error("Expected 'data cannot be 'invalid'' error in body section")
					}
				}
			} else {
				t.Errorf("Expected ValidationErrorResponse, got: %T %v", err, err)
			}
		}
	})

	t.Run("section returns ValidationError - query section", func(t *testing.T) {
		// Test with Query section to ensure section-level validation works for different sections
		req := &TestRequestWithQueryValidationError{
			Query: TestQuerySectionWithValidationError{
				Term: "forbidden", // This will trigger the ValidationError
			},
		}

		err := validator.ValidateRequest(context.Background(), req)

		// Should return ValidationErrorResponse with section-level errors
		if err == nil {
			t.Error("Expected validation error")
		} else {
			if valErr, ok := err.(*ValidationErrorResponse); ok {
				// Check that the errors are added at the query section level
				if queryErrors, exists := valErr.Details["query"]; !exists {
					t.Error("Expected 'query' section errors in validation response")
				} else {
					if len(queryErrors) == 0 {
						t.Error("Expected errors in query section")
					}

					// Check that the custom validation error is included
					found := false
					for _, errMsg := range queryErrors {
						if errMsg == "term 'forbidden' is not allowed" {
							found = true
							break
						}
					}

					if !found {
						t.Error("Expected 'term 'forbidden' is not allowed' error in query section")
					}
				}
			} else {
				t.Errorf("Expected ValidationErrorResponse, got: %T %v", err, err)
			}
		}
	})

	t.Run("section with successful validation", func(t *testing.T) {
		// Test that sections with valid data don't trigger errors
		req := &TestRequestWithSectionValidationError{
			Body: TestSectionWithValidationError{
				Data: "valid", // This should not trigger any validation error
			},
		}

		err := validator.ValidateRequest(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error for valid section, got: %v", err)
		}
	})

	t.Run("multiple section validation errors", func(t *testing.T) {
		// Use the existing TestSectionWithValidationError but test multiple append operations
		req := &TestRequestWithSectionValidationError{
			Body: TestSectionWithValidationError{
				Data: "invalid", // This returns 2 errors
			},
		}

		err := validator.ValidateRequest(context.Background(), req)

		if err == nil {
			t.Error("Expected validation errors")
		} else {
			if valErr, ok := err.(*ValidationErrorResponse); ok {
				if bodyErrors, exists := valErr.Details["body"]; !exists {
					t.Error("Expected 'body' section errors")
				} else if len(bodyErrors) != 2 {
					t.Errorf("Expected 2 errors in body section, got %d: %v", len(bodyErrors), bodyErrors)
				}
			} else {
				t.Errorf("Expected ValidationErrorResponse, got: %T", err)
			}
		}
	})
}
