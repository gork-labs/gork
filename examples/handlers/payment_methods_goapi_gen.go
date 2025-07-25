// Code generated by openapi-gen. DO NOT EDIT.

package handlers

// IsBankPaymentMethod returns true if the union contains BankPaymentMethod.
func (u *PaymentMethodRequest) IsBankPaymentMethod() bool {
	return u.A != nil
}

// BankPaymentMethod returns the BankPaymentMethod value if present, nil otherwise.
func (u *PaymentMethodRequest) BankPaymentMethod() *BankPaymentMethod {
	return u.A
}

// IsCreditCardPaymentMethod returns true if the union contains CreditCardPaymentMethod.
func (u *PaymentMethodRequest) IsCreditCardPaymentMethod() bool {
	return u.B != nil
}

// CreditCardPaymentMethod returns the CreditCardPaymentMethod value if present, nil otherwise.
func (u *PaymentMethodRequest) CreditCardPaymentMethod() *CreditCardPaymentMethod {
	return u.B
}

// Value returns the non-nil value from the union.
func (u *PaymentMethodRequest) Value() interface{} {
	if u.A != nil {
		return u.A
	}
	if u.B != nil {
		return u.B
	}
	return nil
}

// SetBankPaymentMethod sets the union to contain BankPaymentMethod.
func (u *PaymentMethodRequest) SetBankPaymentMethod(value *BankPaymentMethod) {
	// Clear all fields first
	u.A = nil
	u.B = nil

	// Set the appropriate field
	u.A = value
}

// SetCreditCardPaymentMethod sets the union to contain CreditCardPaymentMethod.
func (u *PaymentMethodRequest) SetCreditCardPaymentMethod(value *CreditCardPaymentMethod) {
	// Clear all fields first
	u.A = nil
	u.B = nil

	// Set the appropriate field
	u.B = value
}

// NewPaymentMethodRequestFromBankPaymentMethod creates a new PaymentMethodRequest containing BankPaymentMethod.
func NewPaymentMethodRequestFromBankPaymentMethod(value *BankPaymentMethod) PaymentMethodRequest {
	return PaymentMethodRequest{
		A: value,
	}
}

// NewPaymentMethodRequestFromCreditCardPaymentMethod creates a new PaymentMethodRequest containing CreditCardPaymentMethod.
func NewPaymentMethodRequestFromCreditCardPaymentMethod(value *CreditCardPaymentMethod) PaymentMethodRequest {
	return PaymentMethodRequest{
		B: value,
	}
}
