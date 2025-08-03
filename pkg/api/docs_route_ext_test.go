package api

import (
	"errors"
	"testing"
)

func TestPrepareDocsConfig(t *testing.T) {
	t.Run("no config provided", func(t *testing.T) {
		config := PrepareDocsConfig()
		if config.Title != "API Documentation" {
			t.Errorf("Expected Title to be 'API Documentation', got %s", config.Title)
		}
		if config.OpenAPIPath != "/openapi.json" {
			t.Errorf("Expected OpenAPIPath to be '/openapi.json', got %s", config.OpenAPIPath)
		}
		if config.UITemplate != StoplightUITemplate {
			t.Errorf("Expected UITemplate to be StoplightUITemplate, got %s", config.UITemplate)
		}
	})

	t.Run("empty config provided", func(t *testing.T) {
		config := PrepareDocsConfig(DocsConfig{})
		if config.Title != "API Documentation" {
			t.Errorf("Expected Title to be 'API Documentation', got %s", config.Title)
		}
		if config.OpenAPIPath != "/openapi.json" {
			t.Errorf("Expected OpenAPIPath to be '/openapi.json', got %s", config.OpenAPIPath)
		}
		if config.UITemplate != StoplightUITemplate {
			t.Errorf("Expected UITemplate to be StoplightUITemplate, got %s", config.UITemplate)
		}
	})

	t.Run("all fields provided", func(t *testing.T) {
		customConfig := DocsConfig{
			Title:       "Custom Title",
			OpenAPIPath: "/custom/openapi.json",
			SpecFile:    "/custom/spec.yaml",
			UITemplate:  SwaggerUITemplate,
		}
		config := PrepareDocsConfig(customConfig)

		if config.Title != "Custom Title" {
			t.Errorf("Expected Title 'Custom Title', got %s", config.Title)
		}
		if config.OpenAPIPath != "/custom/openapi.json" {
			t.Errorf("Expected OpenAPIPath '/custom/openapi.json', got %s", config.OpenAPIPath)
		}
		if config.SpecFile != "/custom/spec.yaml" {
			t.Errorf("Expected SpecFile '/custom/spec.yaml', got %s", config.SpecFile)
		}
		if config.UITemplate != SwaggerUITemplate {
			t.Errorf("Expected UITemplate SwaggerUITemplate, got %s", config.UITemplate)
		}
	})
}

func TestNormalizeDocsPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{"no trailing slash", "/docs", "/docs"},
		{"trailing slash", "/docs/", "/docs"},
		{"trailing wildcard", "/docs/*", "/docs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath := normalizeDocsPath(tt.path)
			if gotPath != tt.wantPath {
				t.Errorf("normalizeDocsPath(%q) = %q, want %q", tt.path, gotPath, tt.wantPath)
			}
		})
	}
}

// Mock implementations for testing
type mockFileReader struct {
	data []byte
	err  error
}

func (m mockFileReader) ReadFile(filename string) ([]byte, error) {
	return m.data, m.err
}

type mockSpecParser struct {
	jsonSpec *OpenAPISpec
	jsonErr  error
	yamlSpec *OpenAPISpec
	yamlErr  error
}

func (m mockSpecParser) ParseJSON(data []byte) (*OpenAPISpec, error) {
	return m.jsonSpec, m.jsonErr
}

func (m mockSpecParser) ParseYAML(data []byte) (*OpenAPISpec, error) {
	return m.yamlSpec, m.yamlErr
}

func TestLoadStaticSpecWithDeps(t *testing.T) {
	t.Run("empty spec file", func(t *testing.T) {
		reader := mockFileReader{}
		parser := mockSpecParser{}
		spec := LoadStaticSpecWithDeps("", reader, parser)
		if spec != nil {
			t.Error("Expected nil for empty spec file")
		}
	})

	t.Run("file read error", func(t *testing.T) {
		reader := mockFileReader{err: errors.New("file not found")}
		parser := mockSpecParser{}
		spec := LoadStaticSpecWithDeps("test.json", reader, parser)
		if spec != nil {
			t.Error("Expected nil when file read fails")
		}
	})

	t.Run("successful JSON parse", func(t *testing.T) {
		expectedSpec := &OpenAPISpec{OpenAPI: "3.1.0"}
		reader := mockFileReader{data: []byte(`{"openapi":"3.1.0"}`)}
		parser := mockSpecParser{jsonSpec: expectedSpec}

		spec := LoadStaticSpecWithDeps("test.json", reader, parser)
		if spec == nil {
			t.Fatal("Expected spec, got nil")
		}
		if spec.OpenAPI != "3.1.0" {
			t.Errorf("Expected OpenAPI 3.1.0, got %s", spec.OpenAPI)
		}
	})

	t.Run("JSON parse fails, YAML succeeds", func(t *testing.T) {
		expectedSpec := &OpenAPISpec{OpenAPI: "3.1.0"}
		reader := mockFileReader{data: []byte(`openapi: "3.1.0"`)}
		parser := mockSpecParser{
			jsonErr:  errors.New("invalid JSON"),
			yamlSpec: expectedSpec,
		}

		spec := LoadStaticSpecWithDeps("test.yaml", reader, parser)
		if spec == nil {
			t.Fatal("Expected spec, got nil")
		}
		if spec.OpenAPI != "3.1.0" {
			t.Errorf("Expected OpenAPI 3.1.0, got %s", spec.OpenAPI)
		}
	})

	t.Run("both parsers fail", func(t *testing.T) {
		reader := mockFileReader{data: []byte(`invalid data`)}
		parser := mockSpecParser{
			jsonErr: errors.New("invalid JSON"),
			yamlErr: errors.New("invalid YAML"),
		}

		spec := LoadStaticSpecWithDeps("test.txt", reader, parser)
		if spec != nil {
			t.Error("Expected nil when both parsers fail")
		}
	})
}

func TestDefaultSpecParser(t *testing.T) {
	parser := defaultSpecParser{}

	t.Run("parse valid JSON", func(t *testing.T) {
		data := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1.0.0"}}`)
		spec, err := parser.ParseJSON(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if spec.OpenAPI != "3.1.0" {
			t.Errorf("Expected OpenAPI 3.1.0, got %s", spec.OpenAPI)
		}
	})

	t.Run("parse invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid json}`)
		spec, err := parser.ParseJSON(data)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		if spec == nil {
			t.Error("Expected spec struct even on error")
		}
	})

	t.Run("parse valid YAML", func(t *testing.T) {
		data := []byte(`openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"`)
		spec, err := parser.ParseYAML(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if spec.OpenAPI != "3.1.0" {
			t.Errorf("Expected OpenAPI 3.1.0, got %s", spec.OpenAPI)
		}
	})

	t.Run("parse invalid YAML", func(t *testing.T) {
		data := []byte(`{invalid: yaml: content`)
		spec, err := parser.ParseYAML(data)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
		if spec == nil {
			t.Error("Expected spec struct even on error")
		}
	})
}

func TestOsFileReader(t *testing.T) {
	reader := osFileReader{}

	// Test with non-existent file
	data, err := reader.ReadFile("/non/existent/file.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if data != nil {
		t.Error("Expected nil data for non-existent file")
	}
}
