package handlers

import (
	"context"

	"github.com/gork-labs/gork/pkg/unions"
)

// CreditCardPaymentMethod represents a credit card payment method.
type CreditCardPaymentMethod struct {
	// Type is the payment method type discriminator
	Type string `gork:"type" validate:"required,eq=credit_card"`
	// CardNumber is the credit card number
	CardNumber string `gork:"cardNumber" validate:"required"`
}

// DiscriminatorValue implements unions.Discriminator interface
// discriminator:"credit_card"
func (c CreditCardPaymentMethod) DiscriminatorValue() string {
	return "credit_card"
}

// BankPaymentMethod represents a bank account payment method.
type BankPaymentMethod struct {
	// Type is the payment method type discriminator
	Type string `gork:"type" validate:"required,eq=bank_account"`
	// AccountNumber is the bank account number
	AccountNumber string `gork:"accountNumber" validate:"required"`
	// RoutingNumber is the bank routing number
	RoutingNumber string `gork:"routingNumber" validate:"required"`
}

// DiscriminatorValue implements unions.Discriminator interface
// discriminator:"bank_account"
func (b BankPaymentMethod) DiscriminatorValue() string {
	return "bank_account"
}

// PaymentMethodRequest is the request body which is a union of payment methods.
type PaymentMethodRequest struct {
	Path struct {
		// UserID is the ID of the user whose payment method is being updated
		UserID string `gork:"userId" validate:"required"`
	}
	Body unions.Union2[BankPaymentMethod, CreditCardPaymentMethod]
}

// UpdateUserPaymentMethod handles user payment method update requests.
func UpdateUserPaymentMethod(_ context.Context, _ PaymentMethodRequest) (*struct{}, error) {
	// Handle user payment method update logic here
	return nil, nil
}
