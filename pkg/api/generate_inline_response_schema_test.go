package api

import (
	"reflect"
	"testing"
)

func TestConventionOpenAPIGenerator_GenerateInlineResponseSchema(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	t.Run("struct with Body field returns schema for Body type", func(t *testing.T) {
		type ResponseBody struct {
			ID   string `gork:"id"`
			Name string `gork:"name"`
		}

		type TestResponse struct {
			Body    ResponseBody
			Headers map[string]string
		}

		respType := reflect.TypeOf(TestResponse{})
		schema := generator.generateInlineResponseSchema(respType, components)

		if schema == nil {
			t.Fatal("Expected schema to be generated for Body field")
		}

		// The schema should be a component reference since ResponseBody is a struct
		if schema.Ref == "" {
			t.Error("Expected schema to be a component reference")
		}

		expectedRef := "#/components/schemas/ResponseBody"
		if schema.Ref != expectedRef {
			t.Errorf("Expected ref %q, got %q", expectedRef, schema.Ref)
		}

		// Check that the component was created
		if _, exists := components.Schemas["ResponseBody"]; !exists {
			t.Error("Expected ResponseBody component to be created")
		}
	})

	t.Run("struct without Body field returns nil", func(t *testing.T) {
		type TestResponseWithoutBody struct {
			Headers map[string]string
			Cookies map[string]string
			Status  int
		}

		respType := reflect.TypeOf(TestResponseWithoutBody{})
		schema := generator.generateInlineResponseSchema(respType, components)

		if schema != nil {
			t.Errorf("Expected nil schema for struct without Body field, got: %+v", schema)
		}
	})

	t.Run("struct with multiple fields but Body field returns Body schema", func(t *testing.T) {
		type TestResponseMultipleFields struct {
			Headers map[string]string
			Body    string // Simple string body
			Cookies map[string]string
			Status  int
		}

		respType := reflect.TypeOf(TestResponseMultipleFields{})
		schema := generator.generateInlineResponseSchema(respType, components)

		if schema == nil {
			t.Fatal("Expected schema to be generated for Body field")
		}

		// For a string Body, should get a direct string schema
		if schema.Type != "string" {
			t.Errorf("Expected string type schema for string Body field, got type: %q", schema.Type)
		}
	})

	t.Run("struct with Body field of primitive type", func(t *testing.T) {
		type TestResponsePrimitiveBody struct {
			Body int
		}

		respType := reflect.TypeOf(TestResponsePrimitiveBody{})
		schema := generator.generateInlineResponseSchema(respType, components)

		if schema == nil {
			t.Fatal("Expected schema to be generated for primitive Body field")
		}

		if schema.Type != "integer" {
			t.Errorf("Expected integer type schema for int Body field, got type: %q", schema.Type)
		}
	})

	t.Run("struct with Body field of slice type", func(t *testing.T) {
		type TestResponseSliceBody struct {
			Body []string
		}

		respType := reflect.TypeOf(TestResponseSliceBody{})
		schema := generator.generateInlineResponseSchema(respType, components)

		if schema == nil {
			t.Fatal("Expected schema to be generated for slice Body field")
		}

		if schema.Type != "array" {
			t.Errorf("Expected array type schema for slice Body field, got type: %q", schema.Type)
		}

		if schema.Items == nil {
			t.Error("Expected array schema to have items")
		} else if schema.Items.Type != "string" {
			t.Errorf("Expected array items to be string type, got: %q", schema.Items.Type)
		}
	})

	t.Run("empty struct returns nil", func(t *testing.T) {
		type EmptyResponse struct{}

		respType := reflect.TypeOf(EmptyResponse{})
		schema := generator.generateInlineResponseSchema(respType, components)

		if schema != nil {
			t.Errorf("Expected nil schema for empty struct, got: %+v", schema)
		}
	})
}
