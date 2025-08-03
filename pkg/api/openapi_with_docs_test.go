package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

type Req struct {
	Body struct {
		ID string `gork:"id"`
	}
}

type Resp struct {
	Body struct {
		Message string `gork:"message"`
	}
}

func TestGenerateOpenAPIWithDocs(t *testing.T) {
	registry := NewRouteRegistry()
	info := &RouteInfo{
		Method:       "GET",
		Path:         "/foo",
		HandlerName:  "GetFoo",
		RequestType:  reflect.TypeOf(Req{}),
		ResponseType: reflect.TypeOf((*Resp)(nil)),
	}
	registry.Register(info)

	extractor := NewDocExtractor()
	// fake doc entries
	extractor.docs["GetFoo"] = Documentation{Description: "Returns foo."}

	spec := GenerateOpenAPIWithDocs(registry, extractor)
	p := spec.Paths["/foo"]
	if p == nil || p.Get == nil {
		t.Fatalf("missing get path")
	}
	if p.Get.Description != "Returns foo." {
		t.Errorf("description not propagated: %q", p.Get.Description)
	}
}

func TestGenerateOpenAPIWithDocs_NilExtractor(t *testing.T) {
	registry := NewRouteRegistry()
	info := &RouteInfo{
		Method:       "GET",
		Path:         "/test",
		HandlerName:  "GetTest",
		RequestType:  reflect.TypeOf(Req{}),
		ResponseType: reflect.TypeOf((*Resp)(nil)),
	}
	registry.Register(info)

	// Test with nil extractor - should return spec without docs enhancement
	spec := GenerateOpenAPIWithDocs(registry, nil)

	if spec == nil {
		t.Fatal("GenerateOpenAPIWithDocs() returned nil for nil extractor")
	}

	p := spec.Paths["/test"]
	if p == nil || p.Get == nil {
		t.Fatalf("missing get path")
	}

	// Should not have description since extractor was nil
	if p.Get.Description != "" {
		t.Errorf("description should be empty when extractor is nil, got: %q", p.Get.Description)
	}
}

// Test types for nullable field testing
type TestRequestWithNullables struct {
	RequiredField string  `gork:"requiredField" validate:"required"`
	OptionalField *string `gork:"optionalField"`
	OptionalInt   *int    `gork:"optionalInt"`
}

type TestComplexStruct struct {
	Name string `gork:"name"`
	Age  int    `gork:"age"`
}

type TestRequestWithNullableStruct struct {
	RequiredStruct TestComplexStruct  `gork:"requiredStruct"`
	OptionalStruct *TestComplexStruct `gork:"optionalStruct"`
}

func TestNullableFieldsInOpenAPI(t *testing.T) {
	// Test basic nullable types
	registry := make(map[string]*Schema)
	basicRequest := reflect.TypeOf(TestRequestWithNullables{})
	schemaRef := reflectTypeToSchemaInternal(basicRequest, registry, true)

	// The schema should be registered in the registry, and we get back a reference
	var schema *Schema
	if schemaRef.Ref != "" {
		// Extract the type name from the reference
		refName := schemaRef.Ref[len("#/components/schemas/"):]
		var ok bool
		schema, ok = registry[refName]
		if !ok {
			t.Fatalf("Schema not found in registry for ref: %s", schemaRef.Ref)
		}
	} else {
		schema = schemaRef
	}

	// Check that OptionalField is nullable
	if optionalFieldSchema, ok := schema.Properties["optionalField"]; ok {
		if len(optionalFieldSchema.Types) != 2 || optionalFieldSchema.Types[0] != "string" || optionalFieldSchema.Types[1] != "null" {
			t.Errorf("OptionalField should have types [string, null], got: %v", optionalFieldSchema.Types)
		}
	} else {
		t.Error("OptionalField not found in schema properties")
	}

	// Check that OptionalInt is nullable
	if optionalIntSchema, ok := schema.Properties["optionalInt"]; ok {
		if len(optionalIntSchema.Types) != 2 || optionalIntSchema.Types[0] != "integer" || optionalIntSchema.Types[1] != "null" {
			t.Errorf("OptionalInt should have types [integer, null], got: %v", optionalIntSchema.Types)
		}
	} else {
		t.Error("OptionalInt not found in schema properties")
	}

	// Check that RequiredField is not nullable
	if requiredFieldSchema, ok := schema.Properties["requiredField"]; ok {
		if requiredFieldSchema.Type != "string" || len(requiredFieldSchema.Types) > 0 {
			t.Errorf("RequiredField should have type string (not nullable), got type: %v, types: %v", requiredFieldSchema.Type, requiredFieldSchema.Types)
		}
	} else {
		t.Error("RequiredField not found in schema properties")
	}
}

func TestNullableComplexTypesInOpenAPI(t *testing.T) {
	registry := make(map[string]*Schema)

	// Test nullable complex struct
	complexRequest := reflect.TypeOf(TestRequestWithNullableStruct{})
	schemaRef := reflectTypeToSchemaInternal(complexRequest, registry, true)

	// The schema should be registered in the registry, and we get back a reference
	var schema *Schema
	if schemaRef.Ref != "" {
		// Extract the type name from the reference
		refName := schemaRef.Ref[len("#/components/schemas/"):]
		var ok bool
		schema, ok = registry[refName]
		if !ok {
			t.Fatalf("Schema not found in registry for ref: %s", schemaRef.Ref)
		}
	} else {
		schema = schemaRef
	}

	// Check that OptionalStruct uses anyOf with null
	if optionalStructSchema, ok := schema.Properties["optionalStruct"]; ok {
		if len(optionalStructSchema.AnyOf) != 2 {
			t.Errorf("OptionalStruct should have anyOf with 2 options, got: %d", len(optionalStructSchema.AnyOf))
		} else {
			// One should be a reference to TestComplexStruct, one should be null
			hasRef := false
			hasNull := false
			for _, anyOfSchema := range optionalStructSchema.AnyOf {
				if anyOfSchema.Ref != "" {
					hasRef = true
				}
				if anyOfSchema.Type == "null" {
					hasNull = true
				}
			}
			if !hasRef || !hasNull {
				t.Errorf("OptionalStruct anyOf should have both a ref and null type, hasRef: %v, hasNull: %v", hasRef, hasNull)
			}
		}
	} else {
		t.Error("OptionalStruct not found in schema properties")
	}

	// Check that RequiredStruct is not nullable
	if requiredStructSchema, ok := schema.Properties["requiredStruct"]; ok {
		if requiredStructSchema.Ref == "" || len(requiredStructSchema.AnyOf) > 0 {
			t.Errorf("RequiredStruct should be a simple ref (not nullable), got ref: %v, anyOf: %v", requiredStructSchema.Ref, requiredStructSchema.AnyOf)
		}
	} else {
		t.Error("RequiredStruct not found in schema properties")
	}
}

func TestSchemaJSONMarshalingWithNullableTypes(t *testing.T) {
	// Test that the custom JSON marshaling works correctly
	schema := &Schema{
		Types: []string{"string", "null"},
	}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	typeField, ok := result["type"]
	if !ok {
		t.Error("type field not found in marshaled JSON")
	}

	typeArray, ok := typeField.([]interface{})
	if !ok {
		t.Errorf("type field should be an array, got: %T", typeField)
	}

	if len(typeArray) != 2 || typeArray[0] != "string" || typeArray[1] != "null" {
		t.Errorf("type array should be [string, null], got: %v", typeArray)
	}
}
