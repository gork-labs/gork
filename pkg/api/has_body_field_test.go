package api

import (
	"reflect"
	"testing"
)

func TestConventionOpenAPIGenerator_HasBodyField(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}

	t.Run("struct with Body field returns true", func(t *testing.T) {
		type StructWithBody struct {
			Body string
		}

		result := generator.hasBodyField(reflect.TypeOf(StructWithBody{}))
		if !result {
			t.Error("Expected hasBodyField to return true for struct with Body field")
		}
	})

	t.Run("struct without Body field returns false", func(t *testing.T) {
		type StructWithoutBody struct {
			Name    string
			Email   string
			Headers map[string]string
			Cookies map[string]string
		}

		result := generator.hasBodyField(reflect.TypeOf(StructWithoutBody{}))
		if result {
			t.Error("Expected hasBodyField to return false for struct without Body field")
		}
	})

	t.Run("empty struct returns false", func(t *testing.T) {
		type EmptyStruct struct{}

		result := generator.hasBodyField(reflect.TypeOf(EmptyStruct{}))
		if result {
			t.Error("Expected hasBodyField to return false for empty struct")
		}
	})

	t.Run("struct with body field (lowercase) returns false", func(t *testing.T) {
		type StructWithLowercaseBody struct {
			body string // lowercase, should not match
		}

		result := generator.hasBodyField(reflect.TypeOf(StructWithLowercaseBody{}))
		if result {
			t.Error("Expected hasBodyField to return false for struct with lowercase 'body' field")
		}
	})
}
