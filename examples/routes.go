// Package examples provides example API handlers and routes for the OpenAPI generator.
package examples

import (
	"net/http"

	"github.com/gork-labs/gork/examples/handlers"
	stdlib "github.com/gork-labs/gork/pkg/adapters/stdlib"
	"github.com/gork-labs/gork/pkg/api"
	stripepkg "github.com/gork-labs/gork/pkg/webhooks/stripe"
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

	// Webhooks (Stripe) — served alongside regular API
	r.Post(
		"/webhooks/stripe",
		api.WebhookHandlerFunc(
			stripepkg.NewHandler(getStripeSecret()),
			api.WithEventHandler(
				"payment_intent.succeeded", handlers.HandlePaymentIntentSucceeded,
			),
			api.WithEventHandler(
				"payment_intent.payment_failed", handlers.HandlePaymentIntentFailed,
			),
			api.WithEventHandler(
				"customer.created", handlers.HandleCustomerCreated,
			),
			api.WithEventHandler(
				"invoice.paid", handlers.HandleInvoicePaid,
			),
		),
		api.WithTags("webhooks", "stripe"),
	)

	return r
}

func getStripeSecret() string {
	// example placeholder; in real apps use env/config
	return "whsec_example"
}
