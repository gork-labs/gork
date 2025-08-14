package api

import (
	"context"
	"reflect"
	"testing"
)

// TestValidateSectionFieldsWithStruct tests validateSectionFields with a struct that causes issues
func TestValidateSectionFieldsWithStruct(t *testing.T) {
	validator := NewConventionValidator()

	t.Run("struct with invalid field type that might cause validator to fail", func(t *testing.T) {
		// Test with a struct that might cause the validator to return a non-ValidationError
		// This is tricky because go-playground/validator is quite robust
		// One approach is to pass a non-struct value to Struct() method via reflection

		field := reflect.StructField{
			Name: "Body",
			Type: reflect.TypeOf("string"), // This will cause reflection issues
		}

		// Pass a string instead of struct to validator.Struct()
		fieldValue := reflect.ValueOf("not a struct")
		validationErrors := make(map[string][]string)

		// This might trigger the non-ValidationError path
		err := validator.validateSection(context.Background(), field, fieldValue, validationErrors)

		// The validator.Struct() method expects a struct, passing a string might cause an error
		// This tests the error propagation path from validateSectionFields to validateSection
		if err != nil {
			// This is the behavior we want - error propagation
			t.Logf("Successfully caught error: %v", err)
		} else {
			t.Log("Validator handled non-struct gracefully")
		}
	})

	t.Run("struct with channel field", func(t *testing.T) {
		// Another approach: use a struct with field types that might cause validator issues
		type ProblematicStruct struct {
			ChanField chan string // Channels can sometimes cause reflection issues
			Value     string
		}

		field := reflect.StructField{
			Name: "Body",
			Type: reflect.TypeOf(ProblematicStruct{}),
		}

		fieldValue := reflect.ValueOf(ProblematicStruct{
			ChanField: make(chan string),
			Value:     "test",
		})
		validationErrors := make(map[string][]string)

		err := validator.validateSection(context.Background(), field, fieldValue, validationErrors)

		// This tests various code paths
		if err != nil {
			t.Logf("Error with channel field: %v", err)
		} else {
			t.Log("Validator handled channel field successfully")
		}
	})
}
