package api

import (
	"reflect"
	"testing"
)

// TestValidateSectionAllPaths ensures all code paths in validateSection are covered
func TestValidateSectionAllPaths(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("normal successful path", func(t *testing.T) {
		type NormalStruct struct {
			Value string
		}

		field := reflect.StructField{
			Name: "Body",
			Type: reflect.TypeOf(NormalStruct{}),
		}

		fieldValue := reflect.ValueOf(NormalStruct{Value: "test"})
		validationErrors := make(map[string][]string)

		err := validator.validateSection(field, fieldValue, validationErrors)

		if err != nil {
			t.Errorf("Expected no error for normal struct, got %v", err)
		}
	})

	t.Run("struct with validation errors", func(t *testing.T) {
		type ValidationStruct struct {
			RequiredField string `validate:"required"`
		}

		field := reflect.StructField{
			Name: "Body",
			Type: reflect.TypeOf(ValidationStruct{}),
		}

		// Empty required field should trigger validation
		fieldValue := reflect.ValueOf(ValidationStruct{RequiredField: ""})
		validationErrors := make(map[string][]string)

		err := validator.validateSection(field, fieldValue, validationErrors)

		// Should not return error, but should populate validationErrors
		if err != nil {
			t.Errorf("Expected no error, validation errors should be collected, got %v", err)
		}

		if len(validationErrors) == 0 {
			t.Error("Expected validation errors to be populated")
		}
	})

}
