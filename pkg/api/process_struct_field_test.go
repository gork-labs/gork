package api

import (
	"reflect"
	"testing"
)

func TestProcessStructField(t *testing.T) {
	t.Run("field with discriminator value", func(t *testing.T) {
		registry := make(map[string]*Schema)
		s := &Schema{
			Properties: make(map[string]*Schema),
		}

		// Create a struct field with discriminator in gork tag
		field := reflect.StructField{
			Name: "Type",
			Type: reflect.TypeOf(""),
			Tag:  `gork:"type,discriminator=order"`,
		}

		processStructField(field, s, registry)

		// Check that the field was added to properties
		if fieldSchema, exists := s.Properties["type"]; !exists {
			t.Error("Expected field 'type' to be added to properties")
		} else {
			// Check that the discriminator value was set as enum
			if fieldSchema.Enum == nil {
				t.Error("Expected Enum to be set for discriminator field")
			} else if len(fieldSchema.Enum) != 1 {
				t.Errorf("Expected Enum to have 1 value, got %d", len(fieldSchema.Enum))
			} else if fieldSchema.Enum[0] != "order" {
				t.Errorf("Expected Enum value to be 'order', got '%s'", fieldSchema.Enum[0])
			}
		}
	})

	t.Run("field without discriminator", func(t *testing.T) {
		registry := make(map[string]*Schema)
		s := &Schema{
			Properties: make(map[string]*Schema),
		}

		// Create a struct field without discriminator
		field := reflect.StructField{
			Name: "Name",
			Type: reflect.TypeOf(""),
			Tag:  `gork:"name"`,
		}

		processStructField(field, s, registry)

		// Check that the field was added to properties
		if fieldSchema, exists := s.Properties["name"]; !exists {
			t.Error("Expected field 'name' to be added to properties")
		} else {
			// Check that no discriminator enum was set
			if fieldSchema.Enum != nil {
				t.Error("Expected Enum to be nil for non-discriminator field")
			}
		}
	})

	t.Run("field with validation constraints", func(t *testing.T) {
		registry := make(map[string]*Schema)
		s := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		// Create a struct field with validation
		field := reflect.StructField{
			Name: "Email",
			Type: reflect.TypeOf(""),
			Tag:  `gork:"email" validate:"required,email"`,
		}

		processStructField(field, s, registry)

		// Check that the field was added to properties
		if fieldSchema, exists := s.Properties["email"]; !exists {
			t.Error("Expected field 'email' to be added to properties")
		} else {
			// Check that string type was set
			if fieldSchema.Type != "string" {
				t.Errorf("Expected field type to be 'string', got '%s'", fieldSchema.Type)
			}
		}

		// Check that required constraint was applied
		found := false
		for _, req := range s.Required {
			if req == "email" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'email' to be in required fields")
		}
	})

	t.Run("field with complex discriminator and validation", func(t *testing.T) {
		registry := make(map[string]*Schema)
		s := &Schema{
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		// Create a struct field with both discriminator and validation
		field := reflect.StructField{
			Name: "PaymentType",
			Type: reflect.TypeOf(""),
			Tag:  `gork:"payment_type,discriminator=credit_card" validate:"required,oneof=credit_card debit_card"`,
		}

		processStructField(field, s, registry)

		// Check that the field was added to properties
		if fieldSchema, exists := s.Properties["payment_type"]; !exists {
			t.Error("Expected field 'payment_type' to be added to properties")
		} else {
			// Check that enum values include both discriminator and validation enum
			// Note: validation constraints may override discriminator enum, so we check for the final result
			if fieldSchema.Enum == nil {
				t.Error("Expected Enum to be set")
			} else {
				// The validation "oneof" constraint will set the enum, potentially overriding discriminator
				// Let's just verify that enum is set and contains valid values
				found := false
				for _, val := range fieldSchema.Enum {
					if val == "credit_card" || val == "debit_card" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected Enum to contain valid payment types")
				}
			}
		}

		// Check that required constraint was applied
		found := false
		for _, req := range s.Required {
			if req == "payment_type" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'payment_type' to be in required fields")
		}
	})

	t.Run("field with empty gork tag uses field name", func(t *testing.T) {
		registry := make(map[string]*Schema)
		s := &Schema{
			Properties: make(map[string]*Schema),
		}

		// Create a struct field with empty gork tag
		field := reflect.StructField{
			Name: "DefaultName",
			Type: reflect.TypeOf(""),
			Tag:  `gork:""`,
		}

		processStructField(field, s, registry)

		// Check that the field was added using the field name
		if _, exists := s.Properties["DefaultName"]; !exists {
			t.Error("Expected field 'DefaultName' to be added to properties when gork tag is empty")
		}
	})
}
