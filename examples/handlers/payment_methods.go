package handlers

import (
	"context"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/gork-labs/gork/pkg/unions"
)

type CreditCardPaymentMethod struct {
	// Type is the payment method type discriminator
	Type string `json:"type" validate:"required,eq=credit_card"`
	// CardNumber is the credit card number
	CardNumber string `json:"cardNumber"`
}

// DiscriminatorValue implements unions.Discriminator interface
// discriminator:"credit_card"
func (c CreditCardPaymentMethod) DiscriminatorValue() string {
	return "credit_card"
}

type BankPaymentMethod struct {
	// Type is the payment method type discriminator
	Type string `json:"type" validate:"required,eq=bank_account"`
	// AccountNumber is the bank account number
	AccountNumber string `json:"accountNumber"`
	// RoutingNumber is the bank routing number
	RoutingNumber string `json:"routingNumber"`
}

// DiscriminatorValue implements unions.Discriminator interface
// discriminator:"bank_account"
func (b BankPaymentMethod) DiscriminatorValue() string {
	return "bank_account"
}

// PaymentMethodRequest is the request body which is a union of payment methods
type PaymentMethodRequest unions.Union2[BankPaymentMethod, CreditCardPaymentMethod]

// UpdateUserPaymentMethod handles user payment method update requests
func UpdateUserPaymentMethod(ctx context.Context, req *PaymentMethodRequest) (*api.NoContentResponse, error) {
	// Handle user payment method update logic here
	// The userId would come from path parameters via the context
	return &api.NoContentResponse{}, nil
}
