package main

import (
	"fmt"

	"github.com/gork-labs/gork/examples/handlers"
)

func main() {
	// Example 1: Creating unions using constructor functions

	// Create a credit card payment using constructor
	creditCardPayment := handlers.NewPaymentRequestFromCreditCardPaymentMethod(
		&handlers.CreditCardPaymentMethod{
			CardNumber: "4111111111111111",
		},
	)

	// Using the generated accessor methods
	if creditCardPayment.IsCreditCardPaymentMethod() {
		fmt.Println("Payment method is credit card")
		card := creditCardPayment.CreditCardPaymentMethod()
		fmt.Printf("Card number: %s\n", card.CardNumber)
	}

	// Example 2: Using setter methods to change union value
	var payment handlers.PaymentRequest

	// Set to bank payment
	payment.SetBankPaymentMethod(&handlers.BankPaymentMethod{
		AccountNumber: "123456789",
		RoutingNumber: "987654321",
	})

	// Check payment type using generated methods
	if payment.IsBankPaymentMethod() {
		fmt.Println("Payment method is bank transfer")
		bank := payment.BankPaymentMethod()
		fmt.Printf("Account: %s, Routing: %s\n", bank.AccountNumber, bank.RoutingNumber)
	}

	// Now change it to credit card
	payment.SetCreditCardPaymentMethod(&handlers.CreditCardPaymentMethod{
		CardNumber: "5555555555554444",
	})
	fmt.Printf("Changed to credit card: %s\n", payment.CreditCardPaymentMethod().CardNumber)

	// Example 3: Using the Value() method to get the active value
	processPayment(&creditCardPayment)
	processPayment(&payment)

	// Example 4: Working with ListUsersResponse union using constructor
	adminUsers := []handlers.AdminUserResponse{
		{
			UserResponse: handlers.UserResponse{
				UserId:   "admin1",
				Username: "superadmin",
			},
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-01-15",
		},
	}
	adminResponse := handlers.NewListUsersResponseFromAdminUserResponseSlice(&adminUsers)

	if adminResponse.IsAdminUserResponseSlice() {
		admins := adminResponse.AdminUserResponseSlice()
		if admins != nil {
			fmt.Printf("Found %d admin users\n", len(*admins))
		}
	}
}

// processPayment demonstrates using the Value() method
func processPayment(payment *handlers.PaymentRequest) {
	value := payment.Value()

	switch v := value.(type) {
	case *handlers.CreditCardPaymentMethod:
		fmt.Printf("Processing credit card payment: %s\n", v.CardNumber)
	case *handlers.BankPaymentMethod:
		fmt.Printf("Processing bank payment: Account %s\n", v.AccountNumber)
	default:
		fmt.Println("Unknown payment method")
	}
}
