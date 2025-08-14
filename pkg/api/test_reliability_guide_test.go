package api

import (
	"context"
	"testing"
	"time"
)

// TestReliabilityGuide demonstrates best practices for reliable and maintainable tests
func TestReliabilityGuide(t *testing.T) {
	t.Run("deterministic test data", func(t *testing.T) {
		// ✅ Good: Use fixed, realistic test data
		testUser := struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			ID:    "user-123",
			Name:  "John Doe",
			Email: "john.doe@example.com",
		}

		// ❌ Bad: Using random or time-dependent data
		// ID: fmt.Sprintf("user-%d", rand.Int())
		// CreatedAt: time.Now()

		if testUser.ID == "" {
			t.Error("Test data should be deterministic and non-empty")
		}
	})

	t.Run("fixed time values", func(t *testing.T) {
		// ✅ Good: Use fixed time for deterministic tests
		fixedTime := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)

		// Test time-related functionality with fixed time
		if fixedTime.Year() != 2023 {
			t.Errorf("Expected year 2023, got %d", fixedTime.Year())
		}

		// ❌ Bad: Using time.Now() which makes tests non-deterministic
		// currentTime := time.Now()
	})

	t.Run("isolated test environment", func(t *testing.T) {
		// ✅ Good: Tests should not depend on external state
		validator := NewConventionValidator()

		// Each test creates its own validator instance
		if validator == nil {
			t.Error("Validator should be created successfully")
		}

		// ❌ Bad: Sharing global state between tests
		// Using global variables or singletons that persist between tests
	})

	t.Run("realistic test data", func(t *testing.T) {
		// ✅ Good: Use realistic data that represents actual usage
		type UserRequest struct {
			Body struct {
				Name    string `gork:"name" validate:"required,min=2,max=50"`
				Email   string `gork:"email" validate:"required,email"`
				Age     int    `gork:"age" validate:"min=13,max=120"`
				Country string `gork:"country" validate:"required,len=2"`
			}
		}

		validRequest := UserRequest{
			Body: struct {
				Name    string `gork:"name" validate:"required,min=2,max=50"`
				Email   string `gork:"email" validate:"required,email"`
				Age     int    `gork:"age" validate:"min=13,max=120"`
				Country string `gork:"country" validate:"required,len=2"`
			}{
				Name:    "Alice Johnson",
				Email:   "alice.johnson@example.com",
				Age:     28,
				Country: "US",
			},
		}

		// Test with realistic data
		validator := NewConventionValidator()
		err := validator.ValidateRequest(context.Background(), &validRequest)
		if err != nil {
			t.Errorf("Valid realistic request should pass validation: %v", err)
		}

		// ❌ Bad: Using unrealistic test data
		// Name: "x", Email: "a", Age: 999, Country: "INVALID"
	})

	t.Run("comprehensive error scenarios", func(t *testing.T) {
		// ✅ Good: Test realistic error scenarios
		type UserRequest struct {
			Body struct {
				Email string `gork:"email" validate:"required,email"`
			}
		}

		errorCases := []struct {
			name     string
			request  UserRequest
			expected string
		}{
			{
				name: "missing email",
				request: UserRequest{
					Body: struct {
						Email string `gork:"email" validate:"required,email"`
					}{
						Email: "",
					},
				},
				expected: "required",
			},
			{
				name: "invalid email format",
				request: UserRequest{
					Body: struct {
						Email string `gork:"email" validate:"required,email"`
					}{
						Email: "not-an-email",
					},
				},
				expected: "email",
			},
			{
				name: "email with spaces",
				request: UserRequest{
					Body: struct {
						Email string `gork:"email" validate:"required,email"`
					}{
						Email: "user @example.com",
					},
				},
				expected: "email",
			},
		}

		validator := NewConventionValidator()
		for _, tc := range errorCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validator.ValidateRequest(context.Background(), &tc.request)
				if err == nil {
					t.Errorf("Expected validation error for %s", tc.name)
					return
				}

				if !IsValidationError(err) {
					t.Errorf("Expected validation error, got %T: %v", err, err)
				}
			})
		}
	})

	t.Run("resource cleanup", func(t *testing.T) {
		// ✅ Good: Proper resource cleanup with defer
		// This is demonstrated in the improved temp file handling above

		// Always use defer for cleanup
		// Always check cleanup errors and log them
		// Use anonymous functions for complex cleanup logic

		t.Log("Resource cleanup patterns are demonstrated in other test files")
	})
}

// TestDataPatterns demonstrates good test data patterns
func TestDataPatterns(t *testing.T) {
	// ✅ Good: Define test data as constants or variables at package level
	// for reuse across multiple tests
	const (
		ValidUserID   = "user-12345"
		ValidEmail    = "test.user@example.com"
		ValidUserName = "Test User"
		InvalidEmail  = "not-an-email"
		EmptyString   = ""
	)

	t.Run("reusable test data", func(t *testing.T) {
		if ValidUserID == "" {
			t.Error("Valid user ID should not be empty")
		}

		if ValidEmail == "" {
			t.Error("Valid email should not be empty")
		}
	})

	t.Run("boundary value testing", func(t *testing.T) {
		// ✅ Good: Test boundary values explicitly
		type LimitRequest struct {
			Query struct {
				Limit int `gork:"limit" validate:"min=1,max=100"`
			}
		}

		boundaryTests := []struct {
			name    string
			limit   int
			isValid bool
		}{
			{"minimum valid", 1, true},
			{"maximum valid", 100, true},
			{"below minimum", 0, false},
			{"above maximum", 101, false},
			{"typical value", 25, true},
		}

		validator := NewConventionValidator()
		for _, tc := range boundaryTests {
			t.Run(tc.name, func(t *testing.T) {
				req := LimitRequest{
					Query: struct {
						Limit int `gork:"limit" validate:"min=1,max=100"`
					}{
						Limit: tc.limit,
					},
				}

				err := validator.ValidateRequest(context.Background(), &req)

				if tc.isValid && err != nil {
					t.Errorf("Expected valid request to pass, got error: %v", err)
				}

				if !tc.isValid && err == nil {
					t.Error("Expected invalid request to fail validation")
				}
			})
		}
	})
}
