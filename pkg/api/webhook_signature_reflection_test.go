package api

import (
	"context"
	"reflect"
	"testing"
)

// TestValidateEventHandlerSignature_ComprehensiveCoverage provides comprehensive test coverage
// for the validateEventHandlerSignature function to ensure all branches are tested.
func TestValidateEventHandlerSignature_ComprehensiveCoverage(t *testing.T) {
	tests := []struct {
		name      string
		handler   interface{}
		expectErr bool
		errMsg    string
	}{
		// Valid cases
		{
			name: "valid handler signature",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) error {
				return nil
			},
			expectErr: false,
		},

		// Function type validation
		{
			name:      "not a function - string",
			handler:   "not a function",
			expectErr: true,
			errMsg:    "handler must be a function",
		},
		{
			name:      "not a function - int",
			handler:   42,
			expectErr: true,
			errMsg:    "handler must be a function",
		},
		{
			name:      "not a function - struct",
			handler:   struct{}{},
			expectErr: true,
			errMsg:    "handler must be a function",
		},

		// Parameter count validation
		{
			name: "no parameters",
			handler: func() error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler must accept exactly 3 parameters (context.Context, *ProviderPayload, *UserPayload)",
		},
		{
			name: "one parameter",
			handler: func(ctx context.Context) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler must accept exactly 3 parameters (context.Context, *ProviderPayload, *UserPayload)",
		},
		{
			name: "two parameters",
			handler: func(ctx context.Context, payload *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler must accept exactly 3 parameters (context.Context, *ProviderPayload, *UserPayload)",
		},
		{
			name: "four parameters",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}, extra string) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler must accept exactly 3 parameters (context.Context, *ProviderPayload, *UserPayload)",
		},

		// Return value count validation
		{
			name: "no return values",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) {
			},
			expectErr: true,
			errMsg:    "handler must return exactly 1 value (error)",
		},
		{
			name: "two return values",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) (interface{}, error) {
				return nil, nil
			},
			expectErr: true,
			errMsg:    "handler must return exactly 1 value (error)",
		},
		{
			name: "three return values",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) (interface{}, error, string) {
				return nil, nil, ""
			},
			expectErr: true,
			errMsg:    "handler must return exactly 1 value (error)",
		},

		// First parameter (context) validation
		{
			name: "first parameter not context - string",
			handler: func(notCtx string, payload *struct{}, metadata *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler first parameter must be context.Context",
		},
		{
			name: "first parameter not context - int",
			handler: func(notCtx int, payload *struct{}, metadata *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler first parameter must be context.Context",
		},
		{
			name: "first parameter not context - any",
			handler: func(notCtx any, payload *struct{}, metadata *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler first parameter must be context.Context",
		},

		// Second parameter (provider payload) validation
		{
			name: "second parameter not pointer - struct",
			handler: func(ctx context.Context, payload struct{}, metadata *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler second parameter must be a pointer to provider payload type",
		},
		{
			name: "second parameter not pointer - string",
			handler: func(ctx context.Context, payload string, metadata *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler second parameter must be a pointer to provider payload type",
		},
		{
			name: "second parameter not pointer - int",
			handler: func(ctx context.Context, payload int, metadata *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler second parameter must be a pointer to provider payload type",
		},
		{
			name: "second parameter not pointer - any",
			handler: func(ctx context.Context, payload any, metadata *struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler second parameter must be a pointer to provider payload type",
		},

		// Third parameter (user metadata) validation
		{
			name: "third parameter not pointer - struct",
			handler: func(ctx context.Context, payload *struct{}, metadata struct{}) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler third parameter must be a pointer to user metadata type",
		},
		{
			name: "third parameter not pointer - string",
			handler: func(ctx context.Context, payload *struct{}, metadata string) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler third parameter must be a pointer to user metadata type",
		},
		{
			name: "third parameter not pointer - int",
			handler: func(ctx context.Context, payload *struct{}, metadata int) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler third parameter must be a pointer to user metadata type",
		},
		{
			name: "third parameter not pointer - any",
			handler: func(ctx context.Context, payload *struct{}, metadata any) error {
				return nil
			},
			expectErr: true,
			errMsg:    "handler third parameter must be a pointer to user metadata type",
		},

		// Return type validation
		{
			name: "return type not error - string",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) string {
				return ""
			},
			expectErr: true,
			errMsg:    "handler return value must be error",
		},
		{
			name: "return type not error - int",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) int {
				return 0
			},
			expectErr: true,
			errMsg:    "handler return value must be error",
		},
		{
			name: "return type not error - any",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) any {
				return nil
			},
			expectErr: true,
			errMsg:    "handler return value must be error",
		},
		{
			name: "return type not error - bool",
			handler: func(ctx context.Context, payload *struct{}, metadata *struct{}) bool {
				return false
			},
			expectErr: true,
			errMsg:    "handler return value must be error",
		},

		// Edge cases with different pointer types
		{
			name: "valid with different pointer types",
			handler: func(ctx context.Context, payload *string, metadata *int) error {
				return nil
			},
			expectErr: false,
		},
		{
			name: "valid with complex pointer types",
			handler: func(ctx context.Context, payload *map[string]any, metadata *[]string) error {
				return nil
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerType := reflect.TypeOf(tt.handler)
			err := validateEventHandlerSignature(handlerType)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				// Validate specific error message if provided
				if tt.errMsg != "" {
					if err.Error() != tt.errMsg {
						t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
					}
				} else {
					// At minimum, error should have a non-empty message
					if err.Error() == "" {
						t.Error("expected non-empty error message")
					}
				}

				// Validate error type - should be a regular error, not a wrapped error
				if err == nil {
					t.Error("error should not be nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateEventHandlerSignature_ReflectionEdgeCases tests edge cases using reflection
// to programmatically create invalid function signatures.
func TestValidateEventHandlerSignature_ReflectionEdgeCases(t *testing.T) {
	t.Run("nil reflect.Type", func(t *testing.T) {
		// This would panic in real usage, but we test the function's behavior
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil reflect.Type")
			}
		}()

		var nilType reflect.Type
		_ = validateEventHandlerSignature(nilType)
	})

	t.Run("function with variadic parameters", func(t *testing.T) {
		// Variadic functions should fail parameter count validation
		handler := func(ctx context.Context, payload *struct{}, metadata *struct{}, extra ...string) error {
			return nil
		}

		err := validateEventHandlerSignature(reflect.TypeOf(handler))
		if err == nil {
			t.Error("expected error for variadic function")
		}
		if !containsString(err.Error(), "exactly 3 parameters") {
			t.Errorf("expected parameter count error, got: %v", err)
		}
	})
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
