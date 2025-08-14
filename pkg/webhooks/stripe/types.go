package stripe

import (
	"context"
	"fmt"
)

// WebhookRequest represents a Stripe webhook request using Gork's convention-over-configuration.
// This struct automatically maps the Stripe-Signature header and raw body for signature verification.
type WebhookRequest struct {
	Headers struct {
		// Stripe-Signature header contains the webhook signature
		StripeSignature string `gork:"Stripe-Signature" validate:"required"`

		// Content-Type should be application/json for Stripe webhooks (optional validation)
		ContentType string `gork:"Content-Type"`
	}

	// Raw body bytes needed for signature verification
	Body []byte
}

// RequestValidationError represents validation errors in webhook requests.
type RequestValidationError struct {
	Errors []string
}

func (e *RequestValidationError) Error() string {
	return fmt.Sprintf("validation errors: %v", e.Errors)
}

// WebhookRequest marker method to implement api.WebhookRequest interface.
func (WebhookRequest) WebhookRequest() {}

// Validate implements basic request validation.
// Signature verification is handled by the Handler.ParseRequest method using the official Stripe SDK.
func (r *WebhookRequest) Validate(_ context.Context) error {
	// Basic validation - signature verification is handled by Handler.ParseRequest
	if r.Headers.StripeSignature == "" {
		return &RequestValidationError{
			Errors: []string{"missing stripe signature header"},
		}
	}

	if len(r.Body) == 0 {
		return &RequestValidationError{
			Errors: []string{"empty webhook body"},
		}
	}

	return nil
}

// WebhookResponse represents standard Stripe webhook response format following Gork conventions.
type WebhookResponse struct {
	Body struct {
		Received bool `json:"received"`
	}
}

// WebhookErrorResponse represents error response for Stripe webhooks following Gork conventions.
type WebhookErrorResponse struct {
	Body struct {
		Received bool   `json:"received"`
		Error    string `json:"error,omitempty"`
	}
}

// StripeEventTypes contains common Stripe event types for validation - will be replaced with official SDK validation.
var StripeEventTypes = []string{
	// Payment intents
	"payment_intent.amount_capturable_updated",
	"payment_intent.canceled",
	"payment_intent.created",
	"payment_intent.partially_funded",
	"payment_intent.payment_failed",
	"payment_intent.processing",
	"payment_intent.requires_action",
	"payment_intent.succeeded",

	// Charges
	"charge.captured",
	"charge.expired",
	"charge.failed",
	"charge.pending",
	"charge.succeeded",
	"charge.updated",

	// Customers
	"customer.created",
	"customer.deleted",
	"customer.updated",
	"customer.subscription.created",
	"customer.subscription.deleted",
	"customer.subscription.updated",
	"customer.subscription.trial_will_end",

	// Invoices
	"invoice.created",
	"invoice.deleted",
	"invoice.finalized",
	"invoice.paid",
	"invoice.payment_action_required",
	"invoice.payment_failed",
	"invoice.payment_succeeded",
	"invoice.sent",
	"invoice.upcoming",
	"invoice.updated",
	"invoice.voided",

	// Subscriptions
	"subscription_schedule.aborted",
	"subscription_schedule.canceled",
	"subscription_schedule.completed",
	"subscription_schedule.created",
	"subscription_schedule.expiring",
	"subscription_schedule.released",
	"subscription_schedule.updated",

	// Products and prices
	"price.created",
	"price.deleted",
	"price.updated",
	"product.created",
	"product.deleted",
	"product.updated",

	// Checkout
	"checkout.session.completed",
	"checkout.session.expired",

	// Setup intents
	"setup_intent.canceled",
	"setup_intent.created",
	"setup_intent.requires_action",
	"setup_intent.setup_failed",
	"setup_intent.succeeded",
}
