package fiber

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type groupPathRequest struct {
	Path struct {
		ID string `gork:"id" validate:"required"`
	}
}

type groupPathResponse struct {
	Body struct {
		ID string `gork:"id"`
	}
}

func groupPathHandler(_ context.Context, req groupPathRequest) (*groupPathResponse, error) {
	resp := &groupPathResponse{}
	resp.Body.ID = req.Path.ID
	return resp, nil
}

func TestGroupPathParameter(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)
	router.Group("/v1").Get("/users/{id}", groupPathHandler)
	router.Group("/v2").Group("/admin").Get("/users/{id}", groupPathHandler)

	tests := []struct {
		path string
		want string
	}{
		{"/v1/users/42", "42"},
		{"/v2/admin/users/7", "7"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil))
			if err != nil {
				t.Fatalf("GET %s: %v", tt.path, err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", resp.StatusCode)
			}
			var body struct {
				ID string `json:"id"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.ID != tt.want {
				t.Errorf("expected id %q, got %q", tt.want, body.ID)
			}
		})
	}
}
