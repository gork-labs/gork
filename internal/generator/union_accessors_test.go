package generator

import (
	"strings"
	"testing"
)

func TestGenerateAccessors(t *testing.T) {
	gen := NewUnionAccessorGenerator("testpkg")
	
	union := UserDefinedUnion{
		TypeName:    "PaymentMethod",
		UnionType:   "unions.Union2[CreditCard, BankAccount]",
		UnionSize:   2,
		OptionTypes: []string{"CreditCard", "BankAccount"},
		PackageName: "testpkg",
	}
	
	result, err := gen.GenerateAccessors(union)
	if err != nil {
		t.Fatalf("GenerateAccessors failed: %v", err)
	}
	
	// Check for expected methods
	expectedMethods := []string{
		"func (u *PaymentMethod) IsCreditCard() bool",
		"func (u *PaymentMethod) CreditCard() CreditCard",
		"func (u *PaymentMethod) IsBankAccount() bool",
		"func (u *PaymentMethod) BankAccount() BankAccount",
		"func (u *PaymentMethod) Value() interface{}",
		"func (u *PaymentMethod) SetCreditCard(value CreditCard)",
		"func (u *PaymentMethod) SetBankAccount(value BankAccount)",
	}
	
	for _, expected := range expectedMethods {
		if !strings.Contains(result, expected) {
			t.Errorf("Generated code missing expected method: %s", expected)
		}
	}
}

func TestGenerateConstructors(t *testing.T) {
	gen := NewUnionAccessorGenerator("testpkg")
	
	union := UserDefinedUnion{
		TypeName:    "PaymentMethod",
		UnionType:   "unions.Union2[CreditCard, BankAccount]",
		UnionSize:   2,
		OptionTypes: []string{"CreditCard", "BankAccount"},
		PackageName: "testpkg",
	}
	
	result, err := gen.GenerateConstructors(union)
	if err != nil {
		t.Fatalf("GenerateConstructors failed: %v", err)
	}
	
	// Check for expected constructor functions
	expectedFuncs := []string{
		"func NewPaymentMethodFromCreditCard(value CreditCard) PaymentMethod",
		"func NewPaymentMethodFromBankAccount(value BankAccount) PaymentMethod",
	}
	
	for _, expected := range expectedFuncs {
		if !strings.Contains(result, expected) {
			t.Errorf("Generated code missing expected function: %s", expected)
		}
	}
}

func TestCleanTypeName(t *testing.T) {
	gen := NewUnionAccessorGenerator("testpkg")
	
	tests := []struct {
		input    string
		expected string
	}{
		{"CreditCard", "CreditCard"},
		{"*CreditCard", "CreditCard"},
		{"models.CreditCard", "CreditCard"},
		{"*models.CreditCard", "CreditCard"},
		{"[]User", "UserSlice"},
		{"[]models.User", "UserSlice"},
		{"map[string]int", "stringTointMap"},
	}
	
	for _, test := range tests {
		result := gen.cleanTypeName(test.input)
		if result != test.expected {
			t.Errorf("cleanTypeName(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestUnion3Generation(t *testing.T) {
	gen := NewUnionAccessorGenerator("testpkg")
	
	union := UserDefinedUnion{
		TypeName:    "AuthMethod",
		UnionType:   "unions.Union3[Password, OAuth, APIKey]",
		UnionSize:   3,
		OptionTypes: []string{"Password", "OAuth", "APIKey"},
		PackageName: "testpkg",
	}
	
	result, err := gen.GenerateAccessors(union)
	if err != nil {
		t.Fatalf("GenerateAccessors for Union3 failed: %v", err)
	}
	
	// Check that all three options have methods
	expectedMethods := []string{
		"IsPassword",
		"IsOAuth", 
		"IsAPIKey",
		"SetPassword",
		"SetOAuth",
		"SetAPIKey",
	}
	
	for _, method := range expectedMethods {
		if !strings.Contains(result, method) {
			t.Errorf("Generated code missing method: %s", method)
		}
	}
	
	// Check field access
	if !strings.Contains(result, "u.A") && !strings.Contains(result, "u.B") && !strings.Contains(result, "u.C") {
		t.Error("Generated code should access fields A, B, and C for Union3")
	}
}