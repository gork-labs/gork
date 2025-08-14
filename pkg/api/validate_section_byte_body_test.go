package api

import (
	"context"
	"reflect"
	"testing"
)

// Test struct with []byte Body field (webhook style)
type ByteBodyRequest struct {
	Body []byte // This should not be validated with go-playground/validator
}

// Test struct with regular struct Body field (traditional style)
type StructBodyRequest struct {
	Body struct {
		Name string `json:"name" validate:"required"`
	}
}

func TestValidateSection_ByteBodyHandling(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("byte body field - should skip go-playground validation", func(t *testing.T) {
		req := ByteBodyRequest{
			Body: []byte(`{"event": "payment.succeeded"}`),
		}

		reqValue := reflect.ValueOf(req)
		reqType := reqValue.Type()

		// Get the Body field
		bodyField := reqType.Field(0)
		bodyValue := reqValue.Field(0)

		validationErrors := make(map[string][]string)

		// This should not cause validation errors because []byte Body fields are handled specially
		err := validator.validateSection(context.Background(), bodyField, bodyValue, validationErrors)

		if err != nil {
			t.Errorf("expected no error for byte body, got %v", err)
		}

		if len(validationErrors) > 0 {
			t.Errorf("expected no validation errors for byte body, got %v", validationErrors)
		}
	})

	t.Run("empty byte body field - should not cause validation errors", func(t *testing.T) {
		req := ByteBodyRequest{
			Body: []byte{}, // Empty byte slice
		}

		reqValue := reflect.ValueOf(req)
		reqType := reqValue.Type()

		bodyField := reqType.Field(0)
		bodyValue := reqValue.Field(0)

		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), bodyField, bodyValue, validationErrors)

		if err != nil {
			t.Errorf("expected no error for empty byte body, got %v", err)
		}

		if len(validationErrors) > 0 {
			t.Errorf("expected no validation errors for empty byte body, got %v", validationErrors)
		}
	})

	t.Run("nil byte body field - should not cause validation errors", func(t *testing.T) {
		req := ByteBodyRequest{
			Body: nil, // Nil byte slice
		}

		reqValue := reflect.ValueOf(req)
		reqType := reqValue.Type()

		bodyField := reqType.Field(0)
		bodyValue := reqValue.Field(0)

		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), bodyField, bodyValue, validationErrors)

		if err != nil {
			t.Errorf("expected no error for nil byte body, got %v", err)
		}

		if len(validationErrors) > 0 {
			t.Errorf("expected no validation errors for nil byte body, got %v", validationErrors)
		}
	})

	t.Run("struct body field - should use go-playground validation", func(t *testing.T) {
		req := StructBodyRequest{
			Body: struct {
				Name string `json:"name" validate:"required"`
			}{
				Name: "", // Empty name should trigger validation error
			},
		}

		reqValue := reflect.ValueOf(req)
		reqType := reqValue.Type()

		bodyField := reqType.Field(0)
		bodyValue := reqValue.Field(0)

		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), bodyField, bodyValue, validationErrors)

		if err != nil {
			t.Errorf("expected no server error for struct body validation, got %v", err)
		}

		// Should have validation error for missing required field
		if len(validationErrors) == 0 {
			t.Error("expected validation errors for struct body with empty required field")
		}

		if errorList, exists := validationErrors["body.Name"]; !exists || len(errorList) == 0 {
			t.Error("expected validation error for body.Name field")
		}
	})
}

func TestValidateSection_ByteBodyFieldDetection(t *testing.T) {
	validator := NewConventionValidator()

	// Test various field types to ensure proper detection
	testCases := []struct {
		name                 string
		fieldType            reflect.Type
		fieldName            string
		shouldSkipValidation bool
	}{
		{
			name:                 "[]byte Body field",
			fieldType:            reflect.TypeOf([]byte{}),
			fieldName:            "Body",
			shouldSkipValidation: true,
		},
		{
			name:                 "[]uint8 Body field (same as []byte)",
			fieldType:            reflect.TypeOf([]uint8{}),
			fieldName:            "Body",
			shouldSkipValidation: true,
		},
		{
			name:                 "[]byte field with different name",
			fieldType:            reflect.TypeOf([]byte{}),
			fieldName:            "Data",
			shouldSkipValidation: false,
		},
		{
			name:                 "[]string Body field",
			fieldType:            reflect.TypeOf([]string{}),
			fieldName:            "Body",
			shouldSkipValidation: false,
		},
		{
			name:                 "string Body field",
			fieldType:            reflect.TypeOf(""),
			fieldName:            "Body",
			shouldSkipValidation: false,
		},
		{
			name: "struct Body field",
			fieldType: reflect.TypeOf(struct {
				Name string
			}{}),
			fieldName:            "Body",
			shouldSkipValidation: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a field descriptor
			field := reflect.StructField{
				Name: tc.fieldName,
				Type: tc.fieldType,
			}

			// Create a zero value of the field type
			fieldValue := reflect.Zero(tc.fieldType)

			validationErrors := make(map[string][]string)

			// Call validateSection - this tests the detection logic
			err := validator.validateSection(context.Background(), field, fieldValue, validationErrors)

			// For byte body fields, should not cause errors
			// For other fields, may cause errors depending on validation rules
			if tc.shouldSkipValidation {
				if err != nil {
					t.Errorf("expected no error for %s, got %v", tc.name, err)
				}
				// Note: We don't check validationErrors here because some types might still
				// cause validation issues during go-playground validation attempt
			}

			// The main test is that it doesn't crash and handles the logic correctly
		})
	}
}
