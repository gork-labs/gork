package api

import (
	"testing"
)

// TestEnhanceOpenAPISpecWithDocsComprehensive tests all paths in EnhanceOpenAPISpecWithDocs
func TestEnhanceOpenAPISpecWithDocsComprehensive(t *testing.T) {
	t.Run("nil spec - early return", func(t *testing.T) {
		extractor := &DocExtractor{}

		// This should trigger line 106-108 (nil spec check)
		EnhanceOpenAPISpecWithDocs(nil, extractor)

		// Should not panic
	})

	t.Run("nil extractor - early return", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
		}

		// This should trigger line 106-108 (nil extractor check)
		EnhanceOpenAPISpecWithDocs(spec, nil)

		// Should not panic
	})

	t.Run("spec with nil components", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI:    "3.1.0",
			Info:       Info{Title: "Test", Version: "1.0.0"},
			Components: nil, // This should skip line 112
		}

		extractor := &DocExtractor{}

		EnhanceOpenAPISpecWithDocs(spec, extractor)

		// Should not panic and should skip component enrichment
	})

	t.Run("spec with components", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Components: &Components{
				Schemas: map[string]*Schema{
					"TestSchema": {Type: "object"},
				},
			},
		}

		extractor := &DocExtractor{
			docs: map[string]Documentation{
				"TestSchema": {
					Description: "Test schema description",
				},
			},
		}

		EnhanceOpenAPISpecWithDocs(spec, extractor)

		// Should enrich component schemas (line 112)
		if spec.Components.Schemas["TestSchema"].Description != "Test schema description" {
			t.Error("Expected component schema to be enriched with documentation")
		}
	})

	t.Run("spec with paths", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Paths: map[string]*PathItem{
				"/test": {
					Get: &Operation{
						Description: "Test operation",
					},
				},
			},
		}

		extractor := &DocExtractor{}

		EnhanceOpenAPISpecWithDocs(spec, extractor)

		// Should call enrichPathOperations (line 116)
		// The behavior depends on the enrichPathOperations implementation
	})

	t.Run("complete spec with both components and paths", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Components: &Components{
				Schemas: map[string]*Schema{
					"TestSchema": {Type: "object"},
				},
			},
			Paths: map[string]*PathItem{
				"/test": {
					Get: &Operation{
						Description: "Test operation",
					},
				},
			},
		}

		extractor := &DocExtractor{
			docs: map[string]Documentation{
				"TestSchema": {
					Description: "Component schema description",
				},
			},
		}

		EnhanceOpenAPISpecWithDocs(spec, extractor)

		// Should enrich both components and paths
		if spec.Components.Schemas["TestSchema"].Description != "Component schema description" {
			t.Error("Expected component schema to be enriched")
		}
	})
}
