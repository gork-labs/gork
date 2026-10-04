package api_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
)

// ErrorResponse has the name of the ErrorResponse component of Gork.
type ErrorResponse struct {
	Code int `gork:"code"`
}

// ValidationErrorResponse has the name of the ValidationErrorResponse component of Gork.
type ValidationErrorResponse struct {
	Fields []string `gork:"fields"`
}

type userErrorResponse struct {
	Body ErrorResponse
}

type userValidationErrorResponse struct {
	Body ValidationErrorResponse
}

type okResponse struct {
	Body struct {
		OK bool `gork:"ok"`
	}
}

func getOK(context.Context, struct{}) (*okResponse, error) { return nil, nil }

func getUserError(context.Context, struct{}) (*userErrorResponse, error) { return nil, nil }

func getUserValidationError(context.Context, struct{}) (*userValidationErrorResponse, error) {
	return nil, nil
}

func TestUserTypeWithTheNameOfAnErrorComponentPanics(t *testing.T) {
	tests := []struct {
		name     string
		handlers []any
		types    []string
	}{
		{
			name:     "user ErrorResponse before the error schemas",
			handlers: []any{getUserError},
			types:    []string{"github.com/gork-labs/gork/pkg/api_test.ErrorResponse", "github.com/gork-labs/gork/pkg/api.ErrorResponse"},
		},
		{
			name:     "user ErrorResponse after the error schemas",
			handlers: []any{getOK, getUserError},
			types:    []string{"github.com/gork-labs/gork/pkg/api_test.ErrorResponse", "github.com/gork-labs/gork/pkg/api.ErrorResponse"},
		},
		{
			name:     "user ValidationErrorResponse",
			handlers: []any{getUserValidationError},
			types:    []string{"github.com/gork-labs/gork/pkg/api_test.ValidationErrorResponse", "github.com/gork-labs/gork/pkg/api.ValidationErrorResponse"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := api.NewRouteRegistry()
			router := api.NewTypedRouter[*struct{}](nil, registry, "", nil, &api.HTTPParameterAdapter{}, nil)
			for i, handler := range tt.handlers {
				router.Get(fmt.Sprintf("/route%d", i), handler)
			}

			defer func() {
				msg := fmt.Sprint(recover())
				for _, want := range tt.types {
					if !strings.Contains(msg, want) {
						t.Errorf("panic = %q, want it to contain %q", msg, want)
					}
				}
			}()
			api.GenerateOpenAPI(registry)
		})
	}
}
