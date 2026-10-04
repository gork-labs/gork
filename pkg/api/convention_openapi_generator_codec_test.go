package api

import (
	"reflect"
	"testing"
	"time"

	"github.com/gork-labs/gork/pkg/gorkson"
)

// Helper function for creating pointers
func gorksonPtr[T any](v T) *T {
	return &v
}

func TestConvertCodecSchemaToOpenAPI(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}

	t.Run("Basic schema conversion", func(t *testing.T) {
		codecSchema := gorkson.OpenAPISchema{
			Type:        gorkson.OpenAPITypeString,
			Format:      gorkson.OpenAPIFormatDateTime,
			Pattern:     "^\\d{4}-\\d{2}",
			MinLength:   gorksonPtr(10),
			MaxLength:   gorksonPtr(30),
			Minimum:     gorksonPtr(1.0),
			Maximum:     gorksonPtr(100.0),
			Example:     "2023-12-25T10:30:00Z",
			Description: "Test date time",
			Enum:        []interface{}{"value1", "value2"},
		}

		result := generator.convertCodecSchemaToOpenAPI(codecSchema)

		if result.Type != codecSchema.Type {
			t.Fatalf("Expected type %s, got %s", codecSchema.Type, result.Type)
		}
		if result.Format != codecSchema.Format {
			t.Fatalf("Expected format %s, got %s", codecSchema.Format, result.Format)
		}
		if result.Pattern != codecSchema.Pattern {
			t.Fatalf("Expected pattern %s, got %s", codecSchema.Pattern, result.Pattern)
		}
		if *result.MinLength != *codecSchema.MinLength {
			t.Fatalf("Expected minLength %d, got %d", *codecSchema.MinLength, *result.MinLength)
		}
		if *result.MaxLength != *codecSchema.MaxLength {
			t.Fatalf("Expected maxLength %d, got %d", *codecSchema.MaxLength, *result.MaxLength)
		}
		if *result.Minimum != *codecSchema.Minimum {
			t.Fatalf("Expected minimum %f, got %f", *codecSchema.Minimum, *result.Minimum)
		}
		if *result.Maximum != *codecSchema.Maximum {
			t.Fatalf("Expected maximum %f, got %f", *codecSchema.Maximum, *result.Maximum)
		}
		if result.Description != codecSchema.Description {
			t.Fatalf("Expected description %s, got %s", codecSchema.Description, result.Description)
		}
		if len(result.Enum) != 2 {
			t.Fatalf("Expected 2 enum values, got %d", len(result.Enum))
		}
		if result.Enum[0] != "value1" || result.Enum[1] != "value2" {
			t.Fatalf("Expected enum [value1, value2], got %v", result.Enum)
		}
	})

	t.Run("Schema with object properties", func(t *testing.T) {
		codecSchema := OpenAPISchema{
			Type: gorkson.OpenAPITypeObject,
			Properties: map[string]*OpenAPISchema{
				"name": {
					Type:        gorkson.OpenAPITypeString,
					Description: "Name field",
				},
				"age": {
					Type:        gorkson.OpenAPITypeInteger,
					Description: "Age field",
				},
			},
		}

		result := generator.convertCodecSchemaToOpenAPI(codecSchema)

		if result.Type != gorkson.OpenAPITypeObject {
			t.Fatalf("Expected type object, got %s", result.Type)
		}
		if len(result.Properties) != 2 {
			t.Fatalf("Expected 2 properties, got %d", len(result.Properties))
		}
		if result.Properties["name"].Type != gorkson.OpenAPITypeString {
			t.Fatalf("Expected name property to be string, got %s", result.Properties["name"].Type)
		}
		if result.Properties["age"].Type != gorkson.OpenAPITypeInteger {
			t.Fatalf("Expected age property to be integer, got %s", result.Properties["age"].Type)
		}
	})

	t.Run("Schema with array items", func(t *testing.T) {
		codecSchema := OpenAPISchema{
			Type: gorkson.OpenAPITypeArray,
			Items: &OpenAPISchema{
				Type:        gorkson.OpenAPITypeString,
				Description: "Array item",
			},
		}

		result := generator.convertCodecSchemaToOpenAPI(codecSchema)

		if result.Type != gorkson.OpenAPITypeArray {
			t.Fatalf("Expected type array, got %s", result.Type)
		}
		if result.Items == nil {
			t.Fatal("Expected items schema to be set")
		}
		if result.Items.Type != gorkson.OpenAPITypeString {
			t.Fatalf("Expected items type string, got %s", result.Items.Type)
		}
	})

	t.Run("Enum with mixed types", func(t *testing.T) {
		codecSchema := OpenAPISchema{
			Type: gorkson.OpenAPITypeString,
			Enum: []interface{}{"string", 123, true, nil},
		}

		result := generator.convertCodecSchemaToOpenAPI(codecSchema)

		if len(result.Enum) != 4 {
			t.Fatalf("Expected 4 enum values, got %d", len(result.Enum))
		}
		if result.Enum[0] != "string" {
			t.Fatalf("Expected first enum to be 'string', got %s", result.Enum[0])
		}
		if result.Enum[1] != "123" {
			t.Fatalf("Expected second enum to be '123', got %s", result.Enum[1])
		}
		if result.Enum[2] != "true" {
			t.Fatalf("Expected third enum to be 'true', got %s", result.Enum[2])
		}
		if result.Enum[3] != "<nil>" {
			t.Fatalf("Expected fourth enum to be '<nil>', got %s", result.Enum[3])
		}
	})

	t.Run("Empty schema", func(t *testing.T) {
		codecSchema := OpenAPISchema{}

		result := generator.convertCodecSchemaToOpenAPI(codecSchema)

		if result.Type != "" {
			t.Fatalf("Expected empty type, got %s", result.Type)
		}
		if result.Properties != nil {
			t.Fatal("Expected properties to be nil")
		}
		if result.Items != nil {
			t.Fatal("Expected items to be nil")
		}
		if result.Enum != nil {
			t.Fatal("Expected enum to be nil")
		}
	})
}

func TestProcessPathSectionWithCodecs(t *testing.T) {
	// Register a codec for testing
	gorkson.RegisterCodec[time.Time](gorkson.TimeCodec{})

	spec := &OpenAPISpec{}
	extractor := &DocExtractor{}
	generator := NewConventionOpenAPIGenerator(spec, extractor)

	operation := &Operation{
		Parameters: []Parameter{},
	}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Create a struct type with time.Time field that has a codec
	sectionType := reflect.StructOf([]reflect.StructField{
		{
			Name: "CreatedAt",
			Type: reflect.TypeOf(time.Time{}),
			Tag:  reflect.StructTag(`gork:"createdAt"`),
		},
		{
			Name: "Name",
			Type: reflect.TypeOf(""),
			Tag:  reflect.StructTag(`gork:"name"`),
		},
	})

	generator.processPathSection(sectionType, operation, components)

	if len(operation.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(operation.Parameters))
	}

	// Find the createdAt parameter
	var createdAtParam *Parameter
	for i, param := range operation.Parameters {
		if param.Name == "createdAt" {
			createdAtParam = &operation.Parameters[i]
			break
		}
	}

	if createdAtParam == nil {
		t.Fatal("Expected to find createdAt parameter")
	}

	// Verify it uses the codec schema
	if createdAtParam.Schema.Type != "string" {
		t.Fatalf("Expected createdAt schema type to be string, got %s", createdAtParam.Schema.Type)
	}
	if createdAtParam.Schema.Format != "date-time" {
		t.Fatalf("Expected createdAt schema format to be date-time, got %s", createdAtParam.Schema.Format)
	}
}

func TestProcessQuerySectionWithCodecs(t *testing.T) {
	// Register a codec for testing
	gorkson.RegisterCodec[time.Time](gorkson.TimeCodec{})

	spec := &OpenAPISpec{}
	extractor := &DocExtractor{}
	generator := NewConventionOpenAPIGenerator(spec, extractor)

	operation := &Operation{
		Parameters: []Parameter{},
	}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Create a struct type with time.Time field that has a codec
	sectionType := reflect.StructOf([]reflect.StructField{
		{
			Name: "UpdatedAt",
			Type: reflect.TypeOf(time.Time{}),
			Tag:  reflect.StructTag(`gork:"updatedAt" validate:"required"`),
		},
		{
			Name: "Category",
			Type: reflect.TypeOf(""),
			Tag:  reflect.StructTag(`gork:"category"`),
		},
	})

	generator.processQuerySection(sectionType, operation, components)

	if len(operation.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(operation.Parameters))
	}

	// Find the updatedAt parameter
	var updatedAtParam *Parameter
	for i, param := range operation.Parameters {
		if param.Name == "updatedAt" {
			updatedAtParam = &operation.Parameters[i]
			break
		}
	}

	if updatedAtParam == nil {
		t.Fatal("Expected to find updatedAt parameter")
	}

	// Verify it uses the codec schema
	if updatedAtParam.Schema.Type != "string" {
		t.Fatalf("Expected updatedAt schema type to be string, got %s", updatedAtParam.Schema.Type)
	}
	if updatedAtParam.Schema.Format != "date-time" {
		t.Fatalf("Expected updatedAt schema format to be date-time, got %s", updatedAtParam.Schema.Format)
	}
	if !updatedAtParam.Required {
		t.Fatal("Expected updatedAt parameter to be required")
	}
	if updatedAtParam.In != "query" {
		t.Fatalf("Expected updatedAt parameter to be in query, got %s", updatedAtParam.In)
	}
}
