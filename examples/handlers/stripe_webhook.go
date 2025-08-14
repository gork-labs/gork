package handlers

import (
	"context"
	"log"

	"github.com/stripe/stripe-go/v76"
)

// PaymentMetadata represents example user-defined metadata types validated by the webhook dispatcher.
type PaymentMetadata struct {
	UserID    string `json:"user_id" validate:"required"`
	ProjectID string `json:"project_id"`
}

// CustomerMetadata represents customer-specific metadata for webhook events.
type CustomerMetadata struct {
	UserID   string `json:"user_id" validate:"required"`
	PlanType string `json:"plan_type"`
}

// InvoiceMetadata represents invoice-specific metadata for webhook events.
type InvoiceMetadata struct {
	UserID         string `json:"user_id" validate:"required"`
	SubscriptionID string `json:"subscription_id" validate:"required"`
	BillingPeriod  string `json:"billing_period"`
}

// HandlePaymentIntentSucceeded processes successful payment intent events.
func HandlePaymentIntentSucceeded(_ context.Context, _ *stripe.PaymentIntent, meta *PaymentMetadata) error {
	if meta != nil {
		log.Printf("Payment succeeded for user %s", meta.UserID)
	}
	return nil
}

// HandlePaymentIntentFailed processes failed payment intent events.
func HandlePaymentIntentFailed(_ context.Context, _ *stripe.PaymentIntent, meta *PaymentMetadata) error {
	if meta != nil {
		log.Printf("Payment failed for user %s", meta.UserID)
	}
	return nil
}

// HandleCustomerCreated processes customer creation events.
func HandleCustomerCreated(_ context.Context, _ *stripe.Customer, meta *CustomerMetadata) error {
	if meta != nil {
		log.Printf("Customer created for user %s", meta.UserID)
	}
	return nil
}

// HandleInvoicePaid processes invoice payment events.
func HandleInvoicePaid(_ context.Context, _ *stripe.Invoice, meta *InvoiceMetadata) error {
	if meta != nil {
		log.Printf("Invoice paid for subscription %s", meta.SubscriptionID)
	}
	return nil
}

// Route wiring is now done inline in examples/routes.go so everything is visible in one place.
