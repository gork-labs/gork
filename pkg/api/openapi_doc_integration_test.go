package api

import (
	"testing"
)

func TestEnrichSchemaPropertiesWithDocs_EmptyFields(t *testing.T) {
	schema := &Schema{
		Properties: map[string]*Schema{
			"name": {Type: "string"},
		},
	}
	doc := Documentation{
		Fields: map[string]FieldDoc{}, // Empty fields
	}

	enrichSchemaPropertiesWithDocs(schema, doc)

	// Description should remain empty
	if schema.Properties["name"].Description != "" {
		t.Errorf("Expected empty description, got '%s'", schema.Properties["name"].Description)
	}
}

func TestEnrichSchemaPropertiesWithDocs_NilProperties(t *testing.T) {
	schema := &Schema{
		Properties: nil, // Nil properties
	}
	doc := Documentation{
		Fields: map[string]FieldDoc{
			"name": {Description: "Name field"},
		},
	}

	enrichSchemaPropertiesWithDocs(schema, doc)

	// Should not panic and should handle nil properties gracefully
}

func TestEnrichSchemaPropertiesWithDocs_MatchingField(t *testing.T) {
	schema := &Schema{
		Properties: map[string]*Schema{
			"name": {Type: "string", Description: ""}, // Empty description
		},
	}
	doc := Documentation{
		Fields: map[string]FieldDoc{
			"name": {Description: "Name field description"},
		},
	}

	enrichSchemaPropertiesWithDocs(schema, doc)

	// Description should be set
	if schema.Properties["name"].Description != "Name field description" {
		t.Errorf("Expected 'Name field description', got '%s'", schema.Properties["name"].Description)
	}
}

func TestEnrichSchemaPropertiesWithDocs_ExistingDescription(t *testing.T) {
	schema := &Schema{
		Properties: map[string]*Schema{
			"name": {Type: "string", Description: "Existing description"}, // Already has description
		},
	}
	doc := Documentation{
		Fields: map[string]FieldDoc{
			"name": {Description: "New description"},
		},
	}

	enrichSchemaPropertiesWithDocs(schema, doc)

	// Description should not be overwritten
	if schema.Properties["name"].Description != "Existing description" {
		t.Errorf("Expected 'Existing description', got '%s'", schema.Properties["name"].Description)
	}
}

func TestEnrichSchemaPropertiesWithDocs_NoMatchingField(t *testing.T) {
	schema := &Schema{
		Properties: map[string]*Schema{
			"name": {Type: "string", Description: ""},
		},
	}
	doc := Documentation{
		Fields: map[string]FieldDoc{
			"age": {Description: "Age field description"}, // Different field name
		},
	}

	enrichSchemaPropertiesWithDocs(schema, doc)

	// Description should remain empty
	if schema.Properties["name"].Description != "" {
		t.Errorf("Expected empty description, got '%s'", schema.Properties["name"].Description)
	}
}

func TestEnhanceInlineSchema_NilSchema(t *testing.T) {
	extractor := NewDocExtractor()

	// Should not panic with nil schema
	enhanceInlineSchema(nil, extractor)
}

func TestEnhanceInlineSchema_NilExtractor(t *testing.T) {
	schema := &Schema{Type: "object"}

	// Should not panic with nil extractor
	enhanceInlineSchema(schema, nil)
}

func TestEnhanceInlineSchema_WithRef(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Ref:  "#/components/schemas/User", // Has $ref
	}
	extractor := NewDocExtractor()

	enhanceInlineSchema(schema, extractor)

	// Should return early for $ref schemas
}

func TestEnhanceInlineSchema_ObjectType(t *testing.T) {
	schema := &Schema{
		Type:        "object",
		Description: "", // Empty description
	}
	extractor := NewDocExtractor()

	enhanceInlineSchema(schema, extractor)

	// Should return early for object type with empty description
}

func TestEnhanceInlineSchema_NonObjectType(t *testing.T) {
	schema := &Schema{
		Type:        "string",
		Description: "",
	}
	extractor := NewDocExtractor()

	enhanceInlineSchema(schema, extractor)

	// Should handle non-object types
}

func TestGenerateOpenAPIWithDocs_NilExtractor(t *testing.T) {
	registry := NewRouteRegistry()

	spec := GenerateOpenAPIWithDocs(registry, nil)

	if spec == nil {
		t.Error("Expected spec to be generated even with nil extractor")
	}
}

func TestGenerateOpenAPIWithDocs_WithExtractor(t *testing.T) {
	registry := NewRouteRegistry()
	extractor := NewDocExtractor()

	// Add some documentation
	extractor.docs["TestType"] = Documentation{
		Description: "Test type description",
		Fields: map[string]FieldDoc{
			"name": {Description: "Name field"},
		},
	}

	spec := GenerateOpenAPIWithDocs(registry, extractor)

	if spec == nil {
		t.Error("Expected spec to be generated")
	}
}

func TestEnhanceOpenAPISpecWithDocs_NilSpec(t *testing.T) {
	extractor := NewDocExtractor()

	// Should not panic with nil spec
	EnhanceOpenAPISpecWithDocs(nil, extractor)
}

func TestEnhanceOpenAPISpecWithDocs_NilExtractor(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: map[string]*Schema{
				"User": {Type: "object"},
			},
		},
	}

	// Should not panic with nil extractor
	EnhanceOpenAPISpecWithDocs(spec, nil)
}

func TestEnhanceOpenAPISpecWithDocs_NilComponents(t *testing.T) {
	spec := &OpenAPISpec{
		Components: nil, // Nil components
		Paths:      map[string]*PathItem{},
	}
	extractor := NewDocExtractor()

	// Should not panic with nil components
	EnhanceOpenAPISpecWithDocs(spec, extractor)
}

func TestEnhanceOpenAPISpecWithDocs_WithComponents(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: map[string]*Schema{
				"User": {
					Type:        "object",
					Description: "",
					Properties: map[string]*Schema{
						"name": {Type: "string", Description: ""},
					},
				},
			},
		},
		Paths: map[string]*PathItem{},
	}

	extractor := NewDocExtractor()
	extractor.docs["User"] = Documentation{
		Description: "User type description",
		Fields: map[string]FieldDoc{
			"name": {Description: "User name field"},
		},
	}

	EnhanceOpenAPISpecWithDocs(spec, extractor)

	// Check that schema was enriched
	userSchema := spec.Components.Schemas["User"]
	if userSchema.Description != "User type description" {
		t.Errorf("Expected 'User type description', got '%s'", userSchema.Description)
	}
	if userSchema.Properties["name"].Description != "User name field" {
		t.Errorf("Expected 'User name field', got '%s'", userSchema.Properties["name"].Description)
	}
}

func TestUpdateOperationWithDocs_NilOperation(t *testing.T) {
	extractor := NewDocExtractor()

	// Should not panic with nil operation
	updateOperationWithDocs(nil, extractor)
}

func TestUpdateOperationWithDocs_NilExtractor(t *testing.T) {
	op := &Operation{OperationID: "TestOp"}

	// Should not panic with nil extractor
	updateOperationWithDocs(op, nil)
}

func TestUpdateOperationWithDocs_WithRequestBody(t *testing.T) {
	op := &Operation{
		OperationID: "TestOp",
		RequestBody: &RequestBody{
			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{Type: "object"},
				},
			},
		},
	}
	extractor := NewDocExtractor()
	extractor.docs["TestOp"] = Documentation{
		Description: "Test operation",
	}

	updateOperationWithDocs(op, extractor)

	if op.Description != "Test operation" {
		t.Errorf("Expected 'Test operation', got '%s'", op.Description)
	}
}

func TestUpdateOperationWithDocs_WithResponses(t *testing.T) {
	op := &Operation{
		OperationID: "TestOp",
		Responses: map[string]*Response{
			"200": {
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Type: "object"},
					},
				},
			},
		},
	}
	extractor := NewDocExtractor()

	updateOperationWithDocs(op, extractor)

	// Should process responses without error
}
