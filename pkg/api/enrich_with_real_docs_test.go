package api

import (
	"testing"
)

// TestEnrichSchemaWithRealDocs tests enrichSchemaWithTypeDoc with populated DocExtractor
func TestEnrichSchemaWithRealDocs(t *testing.T) {
	t.Run("doc extractor with actual documentation", func(t *testing.T) {
		// Create a DocExtractor and populate it with documentation
		extractor := &DocExtractor{
			docs: map[string]Documentation{
				"TestType": {
					Description: "This is a test type description",
					Fields: map[string]FieldDoc{
						"field1": {Description: "Field 1 description"},
						"field2": {Description: "Field 2 description"},
					},
				},
			},
		}

		// Create a schema without description
		schema := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"field1": {Type: "string", Description: ""},
				"field2": {Type: "integer", Description: ""},
			},
		}

		// This should trigger the doc.Description != "" path (line 33-34)
		enrichSchemaWithTypeDoc(schema, "TestType", extractor)

		// Verify that description was set
		if schema.Description != "This is a test type description" {
			t.Errorf("Expected description to be set to 'This is a test type description', got '%s'", schema.Description)
		}

		// Verify that field descriptions were set
		if schema.Properties["field1"].Description != "Field 1 description" {
			t.Errorf("Expected field1 description to be 'Field 1 description', got '%s'", schema.Properties["field1"].Description)
		}

		if schema.Properties["field2"].Description != "Field 2 description" {
			t.Errorf("Expected field2 description to be 'Field 2 description', got '%s'", schema.Properties["field2"].Description)
		}
	})

	t.Run("schema with existing description gets overwritten", func(t *testing.T) {
		extractor := &DocExtractor{
			docs: map[string]Documentation{
				"TestType": {
					Description: "New description from docs",
				},
			},
		}

		schema := &Schema{
			Description: "Existing description",
		}

		enrichSchemaWithTypeDoc(schema, "TestType", extractor)

		// Should overwrite existing description
		if schema.Description != "New description from docs" {
			t.Errorf("Expected description to be overwritten to 'New description from docs', got '%s'", schema.Description)
		}
	})
}
