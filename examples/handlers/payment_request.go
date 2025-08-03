package handlers

import (
	"context"

	"github.com/gork-labs/gork/pkg/unions"
)

// PaymentRequest represents a payment request with different payment methods.
type PaymentRequest unions.Union2[CreditCardPaymentMethod, BankPaymentMethod]

// ProcessPaymentRequest represents the full request for processing payment.
type ProcessPaymentRequest struct {
	Body struct {
		// UserID is the ID of the user making payment
		UserID string `gork:"userID" validate:"required"`

		// Amount is the payment amount in cents
		Amount int64 `gork:"amount" validate:"required,min=1"`

		// PaymentMethod contains the payment method details
		PaymentMethod PaymentRequest `gork:"paymentMethod" validate:"required"`
	}
}

// ProcessPaymentResponse represents the payment processing response.
type ProcessPaymentResponse struct {
	Body struct {
		// TransactionID is the unique identifier for this transaction
		TransactionID string `gork:"transactionId"`

		// Status indicates if the payment was successful
		Status string `gork:"status"`
	}
}

// ProcessPayment handles payment processing with different payment methods.
func ProcessPayment(_ context.Context, _ *ProcessPaymentRequest) (*ProcessPaymentResponse, error) {
	// Process payment logic here
	return &ProcessPaymentResponse{
		Body: struct {
			TransactionID string `gork:"transactionId"`
			Status        string `gork:"status"`
		}{
			TransactionID: "txn_123456",
			Status:        "success",
		},
	}, nil
}
