package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	stdlibrouter "github.com/gork-labs/gork/pkg/adapters/stdlib"
	"github.com/gork-labs/gork/pkg/api"
)

// --- Unit tests -----------------------------------------------------------

type sampleRequest struct {
	Body struct {
		Name string `gork:"name" validate:"required"`
	}
}

type sampleResponse struct {
	Body struct {
		OK bool `gork:"ok"`
	}
}

func sampleHandler(ctx context.Context, req sampleRequest) (sampleResponse, error) {
	return sampleResponse{
		Body: struct {
			OK bool `gork:"ok"`
		}{
			OK: true,
		},
	}, nil
}

func TestCheckDiscriminatorErrors(t *testing.T) {
	type Disc struct {
		Kind string `gork:"kind,discriminator=foo"`
	}
	// Missing field
	if errs := api.CheckDiscriminatorErrors(Disc{}); len(errs) == 0 || errs["kind"][0] != "required" {
		t.Fatalf("expected required error, got %v", errs)
	}
	// Mismatch value
	if errs := api.CheckDiscriminatorErrors(Disc{Kind: "bar"}); len(errs) == 0 || errs["kind"][0] != "discriminator" {
		t.Fatalf("expected discriminator error, got %v", errs)
	}
}

// --- Integration test with stdlib router ----------------------------------

type reqX struct {
	Body struct {
		Name string `gork:"name" validate:"required,min=2"`
	}
}

type respX struct {
	Body struct {
		Msg string `gork:"msg"`
	}
}

func handlerX(_ context.Context, r reqX) (respX, error) {
	return respX{
		Body: struct {
			Msg string `gork:"msg"`
		}{
			Msg: "hi " + r.Body.Name,
		},
	}, nil
}

func TestHTTPValidationFlow(t *testing.T) {
	mux := http.NewServeMux()
	router := stdlibrouter.NewRouter(mux)
	router.Post("/test", handlerX)

	// 1. Valid request
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"name": "john"})
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// 2. Invalid JSON -> 400
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("{"))
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	// 3. Validation error (too short name) -> 400
	rr = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]any{"name": "x"})
	req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// --- Additional Validation Tests -------------------------------------------

func TestValidationWithRealGetUserExample(t *testing.T) {
	validator := api.NewConventionValidator()

	type GetUserRequest struct {
		Path struct{}
		Body struct {
			UserID string `gork:"userID" validate:"required"`
		}
	}

	req := &GetUserRequest{
		Body: struct {
			UserID string `gork:"userID" validate:"required"`
		}{
			UserID: "",
		},
	}

	err := validator.ValidateRequest(context.Background(), req)
	if err == nil {
		t.Fatal("Expected validation error for empty UserID field")
	}

	valErr, ok := err.(*api.ValidationErrorResponse)
	if !ok {
		t.Fatalf("Expected ValidationErrorResponse, got %T", err)
	}

	expectedFieldPath := "body.userID"
	if _, exists := valErr.Details[expectedFieldPath]; !exists {
		t.Errorf("Expected validation error for field '%s', but got errors for: %v", expectedFieldPath, getMapKeys(valErr.Details))
	}

	unexpectedFieldPath := "body.UserID"
	if _, exists := valErr.Details[unexpectedFieldPath]; exists {
		t.Errorf("Should not have validation error for Go field name '%s'", unexpectedFieldPath)
	}

	if errors, exists := valErr.Details[expectedFieldPath]; exists {
		if len(errors) == 0 {
			t.Error("Expected validation errors for userID field")
		} else if errors[0] != "required" {
			t.Errorf("Expected 'required' validation error, got '%s'", errors[0])
		}
	}
}

func TestValidationFieldNamesUseGorkTags(t *testing.T) {
	validator := api.NewConventionValidator()

	type FieldNamesRequest struct {
		Body struct {
			UserEmail string `gork:"user_email" validate:"required,email"`
		}
	}

	req := &FieldNamesRequest{
		Body: struct {
			UserEmail string `gork:"user_email" validate:"required,email"`
		}{
			UserEmail: "invalid-email",
		},
	}

	err := validator.ValidateRequest(context.Background(), req)
	if err == nil {
		t.Fatal("Expected validation error for invalid email")
	}

	valErr, ok := err.(*api.ValidationErrorResponse)
	if !ok {
		t.Fatalf("Expected ValidationErrorResponse, got %T", err)
	}

	expectedFieldPath := "body.user_email"
	if _, exists := valErr.Details[expectedFieldPath]; !exists {
		t.Errorf("Expected validation error for gork field name '%s', but got errors for: %v", expectedFieldPath, getMapKeys(valErr.Details))
	}

	unexpectedFieldPath := "body.UserEmail"
	if _, exists := valErr.Details[unexpectedFieldPath]; exists {
		t.Errorf("Should not have validation error for Go field name '%s'", unexpectedFieldPath)
	}
}

func TestValidationErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		err      api.ValidationError
		expected string
	}{
		{
			name: "RequestValidationError",
			err: &api.RequestValidationError{
				Errors: []string{"request error 1", "request error 2"},
			},
			expected: "request validation failed: request error 1, request error 2",
		},
		{
			name: "BodyValidationError",
			err: &api.BodyValidationError{
				Errors: []string{"body error"},
			},
			expected: "body validation failed",
		},
		{
			name: "QueryValidationError",
			err: &api.QueryValidationError{
				Errors: []string{"query error"},
			},
			expected: "query validation failed",
		},
		{
			name: "PathValidationError",
			err: &api.PathValidationError{
				Errors: []string{"path error"},
			},
			expected: "path validation failed",
		},
		{
			name: "HeadersValidationError",
			err: &api.HeadersValidationError{
				Errors: []string{"headers error"},
			},
			expected: "headers validation failed",
		},
		{
			name: "CookiesValidationError",
			err: &api.CookiesValidationError{
				Errors: []string{"cookies error"},
			},
			expected: "cookies validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !contains(tt.err.Error(), tt.expected) {
				t.Errorf("Expected error message to contain '%s', got '%s'", tt.expected, tt.err.Error())
			}

			if len(tt.err.GetErrors()) == 0 {
				t.Error("Expected validation error to have errors")
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "ValidationErrorResponse",
			err: &api.ValidationErrorResponse{
				Message: "Validation failed",
				Details: map[string][]string{"field": {"error"}},
			},
			expected: true,
		},
		{
			name: "RequestValidationError",
			err: &api.RequestValidationError{
				Errors: []string{"error"},
			},
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.IsValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateRequestEdgeCases(t *testing.T) {
	validator := api.NewConventionValidator()

	t.Run("validate request with non-standard sections", func(t *testing.T) {
		type RequestWithNonStandardSection struct {
			Body struct {
				Name string `validate:"required"`
			}
			CustomSection struct {
				Value string
			}
		}

		req := &RequestWithNonStandardSection{
			Body: struct {
				Name string `validate:"required"`
			}{
				Name: "test",
			},
			CustomSection: struct {
				Value string
			}{
				Value: "custom",
			},
		}

		err := validator.ValidateRequest(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("validate with empty request", func(t *testing.T) {
		type EmptyRequest struct{}

		req := &EmptyRequest{}

		err := validator.ValidateRequest(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error for empty request, got %v", err)
		}
	})
}

// Test types with custom validation that provide coverage
type TestValidationRequest struct {
	Query struct {
		Limit int `gork:"limit" validate:"required,min=1,max=100"`
	}
	Body struct {
		Name  string `gork:"name" validate:"required,min=1"`
		Email string `gork:"email" validate:"required,email"`
	}
}

type TestCustomValidationRequest struct {
	Query struct {
		Force bool `gork:"force"`
	}
	Body struct {
		Name string `gork:"name" validate:"required"`
	}
}

func (r *TestCustomValidationRequest) Validate() error {
	if r.Query.Force && r.Body.Name == "" {
		return &api.RequestValidationError{
			Errors: []string{"name is required when force flag is set"},
		}
	}
	return nil
}

type TestSectionValidationBody struct {
	Password        string `gork:"password" validate:"required"`
	ConfirmPassword string `gork:"confirm_password" validate:"required"`
}

func (b *TestSectionValidationBody) Validate() error {
	if b.Password != b.ConfirmPassword {
		return &api.BodyValidationError{
			Errors: []string{"passwords do not match"},
		}
	}
	return nil
}

type TestSectionValidationRequest struct {
	Body struct {
		Password        string `gork:"password" validate:"required"`
		ConfirmPassword string `gork:"confirm_password" validate:"required"`
	}
}

func (b *TestSectionValidationRequest) Validate() error {
	if b.Body.Password != b.Body.ConfirmPassword {
		return &api.BodyValidationError{
			Errors: []string{"passwords do not match"},
		}
	}
	return nil
}

func TestConventionValidator_ValidateRequest(t *testing.T) {
	validator := api.NewConventionValidator()

	tests := []struct {
		name      string
		request   interface{}
		wantError bool
	}{
		{
			name: "valid request",
			request: &TestValidationRequest{
				Query: struct {
					Limit int `gork:"limit" validate:"required,min=1,max=100"`
				}{
					Limit: 10,
				},
				Body: struct {
					Name  string `gork:"name" validate:"required,min=1"`
					Email string `gork:"email" validate:"required,email"`
				}{
					Name:  "John Doe",
					Email: "john@example.com",
				},
			},
			wantError: false,
		},
		{
			name: "field validation errors",
			request: &TestValidationRequest{
				Query: struct {
					Limit int `gork:"limit" validate:"required,min=1,max=100"`
				}{
					Limit: 0,
				},
				Body: struct {
					Name  string `gork:"name" validate:"required,min=1"`
					Email string `gork:"email" validate:"required,email"`
				}{
					Name:  "",
					Email: "invalid-email",
				},
			},
			wantError: true,
		},
		{
			name: "custom request validation",
			request: &TestCustomValidationRequest{
				Query: struct {
					Force bool `gork:"force"`
				}{
					Force: true,
				},
				Body: struct {
					Name string `gork:"name" validate:"required"`
				}{
					Name: "",
				},
			},
			wantError: true,
		},
		{
			name: "custom section validation",
			request: &TestSectionValidationRequest{
				Body: struct {
					Password        string `gork:"password" validate:"required"`
					ConfirmPassword string `gork:"confirm_password" validate:"required"`
				}{
					Password:        "password123",
					ConfirmPassword: "different",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRequest(context.Background(), tt.request)

			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRequest() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil {
				// Validate error message is not empty
				if err.Error() == "" {
					t.Error("Expected non-empty error message")
				}

				// Check that it's a validation error (client error, not server error)
				if !api.IsValidationError(err) {
					t.Errorf("Expected validation error, got %T: %v", err, err)
				}

				// For validation errors, ensure they contain meaningful information
				if valErr, ok := err.(*api.ValidationErrorResponse); ok {
					if len(valErr.Details) == 0 {
						t.Error("Expected validation error details to be populated")
					}

					// Validate that error details contain actual validation messages
					for field, errors := range valErr.Details {
						if len(errors) == 0 {
							t.Errorf("Expected validation errors for field %q", field)
						}
						for _, errMsg := range errors {
							if errMsg == "" {
								t.Errorf("Expected non-empty validation error message for field %q", field)
							}
						}
					}
				}
			}
		})
	}
}

// Helper functions
func getMapKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func contains(str, substr string) bool {
	return strings.Contains(str, substr)
}
