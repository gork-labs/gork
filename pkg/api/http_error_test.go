package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type httpErrorRequest struct {
	Body struct {
		Password string `gork:"password"`
	}
}

type httpErrorResponse struct {
	Body struct {
		Token string `gork:"token"`
	}
}

func serveHTTPError(t *testing.T, handler any) (int, string) {
	t.Helper()
	httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, handler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	httpHandler(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

func TestNewHTTPError(t *testing.T) {
	err := NewHTTPError(http.StatusUnauthorized, "The access password is wrong.")
	if err.Status != http.StatusUnauthorized || err.Message != "The access password is wrong." {
		t.Errorf("NewHTTPError = %+v", err)
	}
	if err.Error() != "The access password is wrong." {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestHandlerReturnsHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "client status sends the message",
			err:        NewHTTPError(http.StatusUnauthorized, "The access password is wrong."),
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error":"The access password is wrong."}`,
		},
		{
			name:       "wrapped error keeps the status",
			err:        fmt.Errorf("login: %w", NewHTTPError(http.StatusNotFound, "No such user.")),
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"No such user."}`,
		},
		{
			name:       "server status hides the message",
			err:        NewHTTPError(http.StatusServiceUnavailable, "database is down"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"error":"Service Unavailable"}`,
		},
		{
			name:       "validation error response gives 400",
			err:        &ValidationErrorResponse{Message: "Validation failed", Details: map[string][]string{"password": {"too short"}}},
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"Validation failed","details":{"password":["too short"]}}`,
		},
		{
			name:       "unknown error stays 500 without the internal text",
			err:        errors.New("secret connection string"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Internal Server Error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]any{
				"response and error": func(context.Context, httpErrorRequest) (*httpErrorResponse, error) { return nil, tt.err },
				"error only":         func(context.Context, httpErrorRequest) error { return tt.err },
			}
			for kind, handler := range handlers {
				status, body := serveHTTPError(t, handler)
				if status != tt.wantStatus || body != tt.wantBody {
					t.Errorf("%s: response = %d %s, want %d %s", kind, status, body, tt.wantStatus, tt.wantBody)
				}
			}
		})
	}
}
