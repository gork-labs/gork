package api

import (
	"reflect"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

// TestUnionSchemaDoesNotExposeImplementationDetails tests that union types generate
// proper oneOf schemas instead of exposing internal A/B fields
func TestUnionSchemaDoesNotExposeImplementationDetails(t *testing.T) {
	// Define test types similar to the real issue
	type AdminUserType struct {
		UserID    string `gork:"userID"`
		Username  string `gork:"username"`
		CreatedAt string `gork:"createdAt"`
		UpdatedAt string `gork:"updatedAt"`
	}

	type UserType struct {
		UserID   string `gork:"userID"`
		Username string `gork:"username"`
	}

	type ListUsersResponse struct {
		Body unions.Union2[[]AdminUserType, []UserType]
	}

	// Create components registry
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Create generator
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	// Generate schema for ListUsersResponse
	respType := reflect.TypeOf(ListUsersResponse{})
	schema := generator.generateResponseComponentSchema(respType, components)

	// The schema should be a reference to a component
	if schema.Ref == "" {
		t.Error("Expected schema to be a component reference")
	}

	// Get the actual component schema
	componentName := "ListUsersResponse"
	componentSchema, exists := components.Schemas[componentName]
	if !exists {
		t.Fatalf("Expected component schema %s to exist", componentName)
	}

	// The component schema should NOT have "A" and "B" properties
	if _, hasA := componentSchema.Properties["A"]; hasA {
		t.Error("Component schema should not expose internal union field 'A'")
	}

	if _, hasB := componentSchema.Properties["B"]; hasB {
		t.Error("Component schema should not expose internal union field 'B'")
	}

	// The component schema should NOT have a "Body" property wrapper (another implementation detail)
	if _, hasBody := componentSchema.Properties["Body"]; hasBody {
		t.Error("Component schema should not expose 'Body' implementation detail")
	}

	// Instead, the component schema itself should be a oneOf schema with two array options
	if len(componentSchema.OneOf) != 2 {
		t.Errorf("Expected component schema to have oneOf with 2 options, got %d", len(componentSchema.OneOf))
	}

	// Verify that the oneOf options are arrays of the correct types
	for i, option := range componentSchema.OneOf {
		if option.Type != "array" {
			t.Errorf("Expected oneOf option %d to be array type, got %s", i, option.Type)
		}
		if option.Items == nil {
			t.Errorf("Expected oneOf option %d to have items schema", i)
		}
	}
}

// TestUnionSchemaWithNamedUnionType tests union schema generation for named union types
func TestUnionSchemaWithNamedUnionType(t *testing.T) {
	type CreditCard struct {
		Type       string `gork:"type" validate:"required"`
		CardNumber string `gork:"cardNumber" validate:"required"`
	}

	type BankAccount struct {
		Type          string `gork:"type" validate:"required"`
		AccountNumber string `gork:"accountNumber" validate:"required"`
		RoutingNumber string `gork:"routingNumber" validate:"required"`
	}

	// Define a named union type
	type PaymentMethod unions.Union2[CreditCard, BankAccount]

	type UpdatePaymentRequest struct {
		Body struct {
			PaymentMethod PaymentMethod `gork:"paymentMethod" validate:"required"`
		}
	}

	// Create components registry
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Create generator
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	// Generate schema for the request body
	reqType := reflect.TypeOf(UpdatePaymentRequest{})
	bodyField, _ := reqType.FieldByName("Body")

	// Generate the schema for the body section
	bodySchema := generator.generateRequestBodyComponentSchema(bodyField.Type, reqType, components)

	// Should be a component reference
	if bodySchema.Ref == "" {
		t.Error("Expected body schema to be a component reference")
	}

	// Get the component schema - should be named after the request context
	componentName := "UpdatePaymentBody"
	componentSchema, exists := components.Schemas[componentName]
	if !exists {
		t.Fatalf("Expected component schema %s to exist", componentName)
	}

	// The paymentMethod field should be a component reference to a union schema
	paymentMethodProp, exists := componentSchema.Properties["paymentMethod"]
	if !exists {
		t.Fatal("Expected paymentMethod property to exist")
	}

	// Should be a reference to the PaymentMethod component
	if paymentMethodProp.Ref != "#/components/schemas/PaymentMethod" {
		t.Errorf("Expected paymentMethod to reference PaymentMethod component, got %s", paymentMethodProp.Ref)
	}

	// The referenced PaymentMethod component should have oneOf schema
	paymentMethodComponent, exists := components.Schemas["PaymentMethod"]
	if !exists {
		t.Fatal("Expected PaymentMethod component to exist")
	}

	if len(paymentMethodComponent.OneOf) != 2 {
		t.Errorf("Expected PaymentMethod component to have oneOf with 2 options, got %d", len(paymentMethodComponent.OneOf))
	}

	// Should not have A/B properties
	if _, hasA := componentSchema.Properties["A"]; hasA {
		t.Error("Component schema should not expose internal union field 'A'")
	}

	if _, hasB := componentSchema.Properties["B"]; hasB {
		t.Error("Component schema should not expose internal union field 'B'")
	}
}
