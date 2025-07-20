// Package examples provides example API handlers and routes for the OpenAPI generator.
package examples

import (
	"net/http"

	"github.com/gork-labs/gork/examples/handlers"
	stdlib "github.com/gork-labs/gork/pkg/adapters/stdlib"
	"github.com/gork-labs/gork/pkg/api"
)

// RegisterRoutes registers all API routes.
func RegisterRoutes(mux *http.ServeMux) *stdlib.Router {
	r := stdlib.NewRouter(mux)

	// Auth
	r.Post("/api/v1/auth/login", handlers.Login, api.WithTags("auth"))

	// Users CRUD
	r.Get("/api/v1/users", handlers.ListUsers, api.WithTags("users"), api.WithBearerTokenAuth("read:users"))
	r.Get("/api/v1/users/{userId}", handlers.GetUser, api.WithTags("users"), api.WithAPIKeyAuth())
	r.Post("/api/v1/users", handlers.CreateUser, api.WithTags("users"), api.WithBasicAuth())
	r.Put("/api/v1/users/{userId}", handlers.UpdateUser, api.WithTags("users"), api.WithBearerTokenAuth())
	r.Delete("/api/v1/users/{userId}", handlers.DeleteUser, api.WithTags("users"))

	// Nested resources
	r.Put("/api/v1/users/{userId}/payment-method", handlers.UpdateUserPaymentMethod, api.WithTags("users"), api.WithBearerTokenAuth("write:payment"))
	r.Put("/api/v1/users/{userId}/preferences", handlers.UpdateUserPreferences, api.WithTags("users"), api.WithBearerTokenAuth("write:preferences"))

	return r
}
