package examples

import (
	"net/http"

	"github.com/gork-labs/gork/examples/handlers"
	"github.com/gork-labs/gork/pkg/api"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", api.HandlerFunc(handlers.Login, api.WithTags("auth")))

	mux.HandleFunc("GET /api/v1/users", api.HandlerFunc(handlers.ListUsers, api.WithTags("users"), api.WithBearerTokenAuth("read:users")))
	mux.HandleFunc("GET /api/v1/users/{userId}", api.HandlerFunc(handlers.GetUser, api.WithTags("users"), api.WithAPIKeyAuth()))
	mux.HandleFunc("POST /api/v1/users", api.HandlerFunc(handlers.CreateUser, api.WithTags("users"), api.WithBasicAuth()))
	mux.HandleFunc("PUT /api/v1/users/{userId}", api.HandlerFunc(handlers.UpdateUser, api.WithTags("users"), api.WithBearerTokenAuth()))
	mux.HandleFunc("DELETE /api/v1/users/{userId}", api.HandlerFunc(handlers.DeleteUser, api.WithTags("users")))

	mux.HandleFunc("PUT /api/v1/users/{userId}/payment-method", api.HandlerFunc(handlers.UpdateUserPaymentMethod, api.WithTags("users"), api.WithBearerTokenAuth("write:payment")))
	mux.HandleFunc("PUT /api/v1/users/{userId}/preferences", api.HandlerFunc(handlers.UpdateUserPreferences, api.WithTags("users"), api.WithBearerTokenAuth("write:preferences")))
}
