package handlers

import (
	"context"

	"github.com/gork-labs/gork/pkg/unions"
)

// PaymentRequest represents a payment request with different payment methods
type PaymentRequest unions.Union2[CreditCardPaymentMethod, BankPaymentMethod]

// ProcessPaymentRequest represents the full request for processing payment
type ProcessPaymentRequest struct {
	// UserID is the ID of the user making payment
	UserID string `json:"userId" validate:"required"`
	
	// Amount is the payment amount in cents
	Amount int64 `json:"amount" validate:"required,min=1"`
	
	// PaymentMethod contains the payment method details
	PaymentMethod PaymentRequest `json:"paymentMethod" validate:"required"`
}

// ProcessPaymentResponse represents the payment processing response
type ProcessPaymentResponse struct {
	// TransactionID is the unique identifier for this transaction
	TransactionID string `json:"transactionId"`
	
	// Status indicates if the payment was successful
	Status string `json:"status"`
}

// ProcessPayment handles payment processing with different payment methods
func ProcessPayment(ctx context.Context, req *ProcessPaymentRequest) (*ProcessPaymentResponse, error) {
	// Process payment logic here
	return &ProcessPaymentResponse{
		TransactionID: "txn_123456",
		Status:        "success",
	}, nil
}