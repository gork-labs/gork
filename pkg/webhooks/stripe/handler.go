// Package stripe provides Stripe webhook handling functionality for the Gork framework.
package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

// Handler implements WebhookHandler for Stripe webhooks with event type validation.
type Handler struct {
	secret           string
	tolerance        time.Duration
	customEventTypes []string
}

// NewHandler creates a new Stripe webhook handler with the webhook endpoint secret.
// The webhookSecret is required for signature verification using the official Stripe SDK.
// If customEventTypes is provided, it will be used instead of the default StripeEventTypes.
func NewHandler(secret string, customEventTypes ...string) api.WebhookHandler[WebhookRequest] {
	return &Handler{secret: secret, tolerance: 5 * time.Minute, customEventTypes: customEventTypes}
}

// ParseRequest extracts the event type and data from the Stripe webhook payload using the official Stripe SDK.
// Uses webhook.ConstructEvent() for signature verification and event parsing.
func (h *Handler) ParseRequest(req WebhookRequest) (api.WebhookEvent, error) {
	ev, err := webhook.ConstructEventWithOptions(req.Body, req.Headers.StripeSignature, h.secret, webhook.ConstructEventOptions{
		Tolerance:                h.tolerance,
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return api.WebhookEvent{}, fmt.Errorf("stripe webhook signature verification failed: %w", err)
	}

	// Provider object and optional user metadata
	providerObj, userMeta := h.mapProviderObjectAndMetadata(ev)

	return api.WebhookEvent{
		Type:           string(ev.Type),
		ProviderObject: providerObj,
		UserMetaJSON:   userMeta,
	}, nil
}

func (h *Handler) mapProviderObjectAndMetadata(ev stripe.Event) (any, json.RawMessage) {
	// Default: return the full event
	var userMeta json.RawMessage
	// Attempt to map common prefixes to concrete types and extract metadata
	switch {
	case hasPrefix(string(ev.Type), "payment_intent."):
		var pi stripe.PaymentIntent
		_ = json.Unmarshal(ev.Data.Raw, &pi)
		// Extract metadata if present
		if len(pi.Metadata) > 0 {
			b, _ := json.Marshal(pi.Metadata)
			userMeta = b
		}
		return &pi, userMeta
	case hasPrefix(string(ev.Type), "customer.subscription."):
		var sub stripe.Subscription
		_ = json.Unmarshal(ev.Data.Raw, &sub)
		if len(sub.Metadata) > 0 {
			b, _ := json.Marshal(sub.Metadata)
			userMeta = b
		}
		return &sub, userMeta
	case hasPrefix(string(ev.Type), "invoice."):
		var inv stripe.Invoice
		_ = json.Unmarshal(ev.Data.Raw, &inv)
		if len(inv.Metadata) > 0 {
			b, _ := json.Marshal(inv.Metadata)
			userMeta = b
		}
		return &inv, userMeta
	default:
		return &ev, nil
	}
}

func hasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }

// SuccessResponse returns the standard Stripe webhook success response following Gork conventions.
func (h *Handler) SuccessResponse() interface{} {
	return WebhookResponse{
		Body: struct {
			Received bool `json:"received"`
		}{
			Received: true,
		},
	}
}

// ErrorResponse returns the standard Stripe webhook error response following Gork conventions.
func (h *Handler) ErrorResponse(err error) interface{} {
	return WebhookErrorResponse{
		Body: struct {
			Received bool   `json:"received"`
			Error    string `json:"error,omitempty"`
		}{
			Received: false,
			Error:    err.Error(),
		},
	}
}

// IsValidEventType validates if the event type is a known Stripe event type.
// Uses custom event types if provided, otherwise uses the default StripeEventTypes.
func (h *Handler) IsValidEventType(eventType string) bool {
	eventTypes := h.getValidEventTypes()

	for _, validType := range eventTypes {
		if eventType == validType {
			return true
		}
	}

	return false
}

// GetValidEventTypes returns the list of valid event types for this handler.
func (h *Handler) GetValidEventTypes() []string {
	return h.getValidEventTypes()
}

// getValidEventTypes returns the event types to use (custom or default).
func (h *Handler) getValidEventTypes() []string {
	if len(h.customEventTypes) > 0 {
		return h.customEventTypes
	}
	return StripeEventTypes
}

// ProviderInfo exposes provider metadata required by the WebhookHandler interface.
func (h *Handler) ProviderInfo() api.WebhookProviderInfo {
	return api.WebhookProviderInfo{
		Name:    "Stripe",
		Website: "https://stripe.com",
		DocsURL: "https://stripe.com/docs/webhooks",
	}
}

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const webhookSecretKey contextKey = "stripe_webhook_secret" // #nosec G101 - This is a context key, not a credential

// NewWebhookContext creates a context with the Stripe webhook secret for signature verification.
// This is a convenience function for setting up the webhook handler with proper context.
func NewWebhookContext(webhookSecret string) context.Context {
	return context.WithValue(context.Background(), webhookSecretKey, webhookSecret)
}
