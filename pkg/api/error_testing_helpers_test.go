package api

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// ErrorTestCase represents a test case for error validation
type ErrorTestCase struct {
	Name            string
	ExpectError     bool
	ExpectedMessage string
	ExpectedType    string // "validation", "server", "custom"
	Description     string
}

// ValidateError provides comprehensive error validation for test cases
func ValidateError(t *testing.T, err error, tc ErrorTestCase) {
	t.Helper()

	if tc.ExpectError && err == nil {
		t.Errorf("Test %q: expected error but got none (%s)", tc.Name, tc.Description)
		return
	}

	if !tc.ExpectError && err != nil {
		t.Errorf("Test %q: unexpected error: %v (%s)", tc.Name, err, tc.Description)
		return
	}

	if !tc.ExpectError {
		return // No error expected and none received - success
	}

	// Error is expected and received - validate details
	if err.Error() == "" {
		t.Errorf("Test %q: expected non-empty error message", tc.Name)
	}

	// Validate specific error message if provided
	if tc.ExpectedMessage != "" {
		if err.Error() != tc.ExpectedMessage {
			t.Errorf("Test %q: expected error message %q, got %q", tc.Name, tc.ExpectedMessage, err.Error())
		}
	}

	// Validate error type if specified
	switch tc.ExpectedType {
	case "validation":
		var valErr validator.ValidationErrors
		if !errors.As(err, &valErr) {
			// Also check for custom validation errors
			if !IsValidationError(err) {
				t.Errorf("Test %q: expected validation error, got %T: %v", tc.Name, err, err)
			}
		}
	case "server":
		var valErr validator.ValidationErrors
		if errors.As(err, &valErr) || IsValidationError(err) {
			t.Errorf("Test %q: expected server error but got validation error: %v", tc.Name, err)
		}
	case "custom":
		// Custom error type validation can be added here
		if err == nil {
			t.Errorf("Test %q: expected custom error but got nil", tc.Name)
		}
	}
}

// ValidateErrorContains checks if error message contains expected substring
func ValidateErrorContains(t *testing.T, err error, expectedSubstring string, testName string) {
	t.Helper()

	if err == nil {
		t.Errorf("Test %q: expected error containing %q but got no error", testName, expectedSubstring)
		return
	}

	if !strings.Contains(err.Error(), expectedSubstring) {
		t.Errorf("Test %q: expected error to contain %q, got %q", testName, expectedSubstring, err.Error())
	}
}

// ValidateErrorType checks if error is of expected type
func ValidateErrorType(t *testing.T, err error, expectedType string, testName string) {
	t.Helper()

	if err == nil {
		t.Errorf("Test %q: expected %s error but got no error", testName, expectedType)
		return
	}

	switch expectedType {
	case "validation":
		var valErr validator.ValidationErrors
		if !errors.As(err, &valErr) && !IsValidationError(err) {
			t.Errorf("Test %q: expected validation error, got %T: %v", testName, err, err)
		}
	case "server":
		var valErr validator.ValidationErrors
		if errors.As(err, &valErr) || IsValidationError(err) {
			t.Errorf("Test %q: expected server error but got validation error: %v", testName, err)
		}
	default:
		t.Errorf("Test %q: unknown expected error type: %s", testName, expectedType)
	}
}

// ValidateNoError ensures no error occurred
func ValidateNoError(t *testing.T, err error, testName string) {
	t.Helper()

	if err != nil {
		t.Errorf("Test %q: unexpected error: %v", testName, err)
	}
}

// ValidateErrorWrapping checks if error properly wraps another error
func ValidateErrorWrapping(t *testing.T, err error, expectedWrappedType string, testName string) {
	t.Helper()

	if err == nil {
		t.Errorf("Test %q: expected wrapped error but got no error", testName)
		return
	}

	switch expectedWrappedType {
	case "validation":
		var valErr validator.ValidationErrors
		if !errors.As(err, &valErr) {
			t.Errorf("Test %q: expected error to wrap ValidationErrors, got %T", testName, err)
		}
	default:
		t.Errorf("Test %q: unknown wrapped error type: %s", testName, expectedWrappedType)
	}
}

// TestErrorTestingHelpers validates the error testing helpers themselves
func TestErrorTestingHelpers(t *testing.T) {
	t.Run("ValidateError with expected error", func(t *testing.T) {
		testErr := errors.New("test error")
		tc := ErrorTestCase{
			Name:            "test case",
			ExpectError:     true,
			ExpectedMessage: "test error",
			Description:     "test description",
		}

		// This should not fail
		ValidateError(t, testErr, tc)
	})

	t.Run("ValidateError with no error expected", func(t *testing.T) {
		tc := ErrorTestCase{
			Name:        "test case",
			ExpectError: false,
			Description: "test description",
		}

		// This should not fail
		ValidateError(t, nil, tc)
	})

	t.Run("ValidateErrorContains", func(t *testing.T) {
		testErr := errors.New("this is a test error message")
		ValidateErrorContains(t, testErr, "test error", "contains test")
	})

	t.Run("ValidateNoError", func(t *testing.T) {
		ValidateNoError(t, nil, "no error test")
	})
}
