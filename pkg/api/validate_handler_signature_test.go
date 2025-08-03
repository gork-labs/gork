package api

import (
	"context"
	"reflect"
	"testing"
)

func TestValidateHandlerSignature(t *testing.T) {
	t.Run("valid handler signature", func(t *testing.T) {
		type TestResponse struct {
			Body struct {
				Message string `gork:"message"`
			}
		}
		validHandler := func(ctx context.Context, req string) (*TestResponse, error) {
			return &TestResponse{}, nil
		}

		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Valid handler signature should not panic: %v", r)
			}
		}()

		validateHandlerSignature(reflect.TypeOf(validHandler))
	})

	t.Run("valid error-only handler signature", func(t *testing.T) {
		validErrorOnlyHandler := func(ctx context.Context, req string) error {
			return nil
		}

		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Valid error-only handler signature should not panic: %v", r)
			}
		}()

		validateHandlerSignature(reflect.TypeOf(validErrorOnlyHandler))
	})

	t.Run("non-function type", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for non-function type")
			} else if r != "handler must be a function" {
				t.Errorf("Expected 'handler must be a function', got: %v", r)
			}
		}()

		validateHandlerSignature(reflect.TypeOf("not a function"))
	})

	t.Run("wrong number of input parameters - zero", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for wrong number of input parameters")
			} else if r != "handler must accept exactly 2 parameters (context.Context, Request)" {
				t.Errorf("Expected parameter count error, got: %v", r)
			}
		}()

		invalidHandler := func() (string, error) {
			return "", nil
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("wrong number of input parameters - one", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for wrong number of input parameters")
			} else if r != "handler must accept exactly 2 parameters (context.Context, Request)" {
				t.Errorf("Expected parameter count error, got: %v", r)
			}
		}()

		invalidHandler := func(ctx context.Context) (string, error) {
			return "", nil
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("wrong number of input parameters - three", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for wrong number of input parameters")
			} else if r != "handler must accept exactly 2 parameters (context.Context, Request)" {
				t.Errorf("Expected parameter count error, got: %v", r)
			}
		}()

		invalidHandler := func(ctx context.Context, req string, extra int) (string, error) {
			return "", nil
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("first parameter not context.Context", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for first parameter not being context.Context")
			} else if r != "first handler parameter must be context.Context" {
				t.Errorf("Expected context error, got: %v", r)
			}
		}()

		invalidHandler := func(notCtx string, req string) (string, error) {
			return "", nil
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("wrong number of return values - zero", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for wrong number of return values")
			} else if r != "handler must return either (error) or (*ResponseType, error)" {
				t.Errorf("Expected return count error, got: %v", r)
			}
		}()

		invalidHandler := func(ctx context.Context, req string) {
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("single return value not error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for single non-error return")
			} else if r != "last return value must be error" {
				t.Errorf("Expected error type error, got: %v", r)
			}
		}()

		invalidHandler := func(ctx context.Context, req string) string {
			return ""
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("wrong number of return values - three", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for wrong number of return values")
			} else if r != "handler must return either (error) or (*ResponseType, error)" {
				t.Errorf("Expected return count error, got: %v", r)
			}
		}()

		invalidHandler := func(ctx context.Context, req string) (string, error, int) {
			return "", nil, 0
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("second return value not error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for second return value not being error")
			} else if r != "last return value must be error" {
				t.Errorf("Expected error type error, got: %v", r)
			}
		}()

		invalidHandler := func(ctx context.Context, req string) (string, string) {
			return "", ""
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})

	t.Run("response type pointer to non-struct", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for pointer to non-struct response type")
			} else if r != "response type must be struct or pointer to struct" {
				t.Errorf("Expected struct type error, got: %v", r)
			}
		}()

		// Handler returns pointer to string instead of pointer to struct
		invalidHandler := func(ctx context.Context, req string) (*string, error) {
			result := "test"
			return &result, nil
		}

		validateHandlerSignature(reflect.TypeOf(invalidHandler))
	})
}
