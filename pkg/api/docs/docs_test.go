package docs

import (
	"testing"
)

func TestExtractDocs(t *testing.T) {
	// Test with nil packages
	docs, err := ExtractDocs(nil)
	if err != nil {
		t.Errorf("ExtractDocs(nil) returned error: %v", err)
	}
	if docs == nil {
		t.Error("ExtractDocs(nil) returned nil docs")
	}
	if docs.Types == nil {
		t.Error("ExtractDocs(nil) returned nil Types map")
	}
	if docs.Fields == nil {
		t.Error("ExtractDocs(nil) returned nil Fields map")
	}
	if len(docs.Types) != 0 {
		t.Errorf("ExtractDocs(nil) returned non-empty Types map: %v", docs.Types)
	}
	if len(docs.Fields) != 0 {
		t.Errorf("ExtractDocs(nil) returned non-empty Fields map: %v", docs.Fields)
	}

	// Test with empty packages slice
	docs, err = ExtractDocs([]string{})
	if err != nil {
		t.Errorf("ExtractDocs([]) returned error: %v", err)
	}
	if docs == nil {
		t.Error("ExtractDocs([]) returned nil docs")
	}
	if len(docs.Types) != 0 {
		t.Errorf("ExtractDocs([]) returned non-empty Types map: %v", docs.Types)
	}
	if len(docs.Fields) != 0 {
		t.Errorf("ExtractDocs([]) returned non-empty Fields map: %v", docs.Fields)
	}

	// Test with some packages (should still return empty due to stub implementation)
	docs, err = ExtractDocs([]string{"fmt", "os"})
	if err != nil {
		t.Errorf("ExtractDocs([\"fmt\", \"os\"]) returned error: %v", err)
	}
	if docs == nil {
		t.Error("ExtractDocs([\"fmt\", \"os\"]) returned nil docs")
	}
	if len(docs.Types) != 0 {
		t.Errorf("ExtractDocs([\"fmt\", \"os\"]) returned non-empty Types map: %v", docs.Types)
	}
	if len(docs.Fields) != 0 {
		t.Errorf("ExtractDocs([\"fmt\", \"os\"]) returned non-empty Fields map: %v", docs.Fields)
	}
}

func TestTypeDocs_Struct(t *testing.T) {
	// Test TypeDocs struct construction and field access
	docs := &TypeDocs{
		Types:  map[string]string{"TestType": "Test type documentation"},
		Fields: map[string]map[string]string{"TestType": {"Field1": "Field documentation"}},
	}

	if docs.Types["TestType"] != "Test type documentation" {
		t.Errorf("Expected type doc 'Test type documentation', got %q", docs.Types["TestType"])
	}

	if docs.Fields["TestType"]["Field1"] != "Field documentation" {
		t.Errorf("Expected field doc 'Field documentation', got %q", docs.Fields["TestType"]["Field1"])
	}

	// Test missing keys
	if docs.Types["NonExistent"] != "" {
		t.Errorf("Expected empty string for non-existent type, got %q", docs.Types["NonExistent"])
	}

	if docs.Fields["NonExistent"] != nil {
		t.Errorf("Expected nil for non-existent type fields, got %v", docs.Fields["NonExistent"])
	}
}
