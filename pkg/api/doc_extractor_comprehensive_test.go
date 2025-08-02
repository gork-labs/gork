package api

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// Test parseFile with valid Go code
func TestParseFile_ValidCode(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()

	// Create a temporary file with valid Go code
	tempFile, err := os.CreateTemp("", "valid*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Write valid Go code with documentation
	content := `package test

// TestType is a test type
type TestType struct {
	// Field1 is the first field
	Field1 string
}

// TestFunc is a test function
func TestFunc() {}
`
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tempFile.Close()

	// Parse the file
	err = extractor.parseFile(tempFile.Name(), fset)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify that documentation was extracted
	typeDoc := extractor.ExtractTypeDoc("TestType")
	if typeDoc.Description != "TestType is a test type" {
		t.Errorf("Expected 'TestType is a test type', got '%s'", typeDoc.Description)
	}

	funcDoc := extractor.ExtractFunctionDoc("TestFunc")
	if funcDoc.Description != "TestFunc is a test function" {
		t.Errorf("Expected 'TestFunc is a test function', got '%s'", funcDoc.Description)
	}
}

// Test processGenDecl with various scenarios
func TestProcessGenDecl_EdgeCases(t *testing.T) {
	extractor := NewDocExtractor()

	// Test with GenDecl that has no doc comment
	genDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Doc: nil, // No doc comment
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{Name: "TestType"},
				Type: &ast.StructType{},
			},
		},
	}

	extractor.processGenDecl(genDecl)

	// Should not have added any documentation
	doc := extractor.ExtractTypeDoc("TestType")
	if doc.Description != "" {
		t.Error("Expected empty description for type without doc comment")
	}

	// Test with GenDecl that is not a TYPE
	varDecl := &ast.GenDecl{
		Tok: token.VAR,
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// This is a variable"},
			},
		},
	}

	extractor.processGenDecl(varDecl)
	// Should not process non-TYPE declarations

	// Test with GenDecl that has doc but no TypeSpec
	emptyTypeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// This should be ignored"},
			},
		},
		Specs: []ast.Spec{}, // No specs
	}

	extractor.processGenDecl(emptyTypeDecl)
	// Should handle empty specs gracefully
}

// Test processStructFields with various field configurations
func TestProcessStructFields_EdgeCases(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{}

	// Test with struct that has fields with no comments
	st := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Names:   []*ast.Ident{{Name: "Field1"}},
					Doc:     nil,
					Comment: nil,
				},
				{
					Names: []*ast.Ident{{Name: "Field2"}},
					Doc: &ast.CommentGroup{
						List: []*ast.Comment{
							{Text: "// Valid comment"},
						},
					},
				},
			},
		},
	}

	extractor.processStructFields(st, doc)

	// Should initialize Fields map
	if doc.Fields == nil {
		t.Error("Expected Fields map to be initialized")
	}

	// Should add documentation for Field2 which has a comment
	if len(doc.Fields) != 1 {
		t.Errorf("Expected 1 field documentation, got %d", len(doc.Fields))
	}

	if fieldDoc, exists := doc.Fields["Field2"]; !exists {
		t.Error("Expected Field2 to have documentation")
	} else if fieldDoc.Description != "Valid comment" {
		t.Errorf("Expected 'Valid comment', got '%s'", fieldDoc.Description)
	}
}

// Test processDirectoryEntry error paths
func TestProcessDirectoryEntry_ErrorPaths(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()

	// Test with directory that doesn't exist
	tempDir := t.TempDir()
	nonExistentDir := filepath.Join(tempDir, "nonexistent")

	err := extractor.processDirectoryEntry(nonExistentDir, &mockDirEntry{name: "nonexistent", isDir: true}, fset)
	// Should handle read errors gracefully
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	// Test with directory containing invalid Go files
	invalidGoDir := t.TempDir()
	invalidGoFile := filepath.Join(invalidGoDir, "invalid.go")
	err = os.WriteFile(invalidGoFile, []byte("this is not valid go code {{{"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// The function should not return an error, it just skips files that fail to parse
	err = extractor.processDirectoryEntry(invalidGoDir, &mockDirEntry{name: filepath.Base(invalidGoDir), isDir: true}, fset)
	if err != nil {
		t.Errorf("Expected no error (invalid files are skipped), got %v", err)
	}
}

// Test extractDescription with various comment formats
func TestExtractDescription_ComprehensiveCases(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected string
	}{
		{
			name:     "single line comment",
			comment:  "// Simple comment",
			expected: "Simple comment",
		},
		{
			name:     "multi-line comment with markers",
			comment:  "// Line 1\n// Line 2\n// Line 3",
			expected: "Line 1 Line 2 Line 3",
		},
		{
			name:     "block comment with stars",
			comment:  "/* Block comment\n * with stars\n * on each line */",
			expected: "Block comment * with stars * on each line",
		},
		{
			name:     "comment with double newlines (multiple paragraphs)",
			comment:  "// First paragraph\n// continues here\n//\n// Second paragraph\n// continues here",
			expected: "First paragraph continues here  Second paragraph continues here",
		},
		{
			name:     "mixed comment markers",
			comment:  "/* Start of block\n// Mixed markers\n * More block\n */",
			expected: "Start of block Mixed markers * More block",
		},
		{
			name:     "comment with extra whitespace",
			comment:  "//   Lots   of   spaces   ",
			expected: "Lots   of   spaces",
		},
		{
			name:     "empty paragraph",
			comment:  "\n\n",
			expected: "",
		},
		{
			name:     "only comment markers",
			comment:  "//\n//\n//",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDescription(tt.comment)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test parseDirectory with various directory structures
func TestParseDirectory_ErrorScenarios(t *testing.T) {
	extractor := NewDocExtractor()

	// Test with non-existent directory
	err := extractor.ParseDirectory("/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	// Test with file instead of directory
	tempFile, err := os.CreateTemp("", "notadir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	err = extractor.ParseDirectory(tempFile.Name())
	// ParseDirectory handles files gracefully by calling WalkDir, which processes the file
	// but processDirectoryEntry returns nil for non-directories, so no error is expected
	if err != nil {
		t.Errorf("Expected no error when parsing a file (it should be skipped), got %v", err)
	}
}

// Test concurrent access safety
func TestDocExtractor_ConcurrentAccess(t *testing.T) {
	extractor := NewDocExtractor()

	// Add some initial documentation
	extractor.docs["TestType"] = Documentation{
		Description: "Test type",
		Fields: map[string]FieldDoc{
			"field1": {Description: "Field 1"},
		},
	}

	// Test concurrent reads
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Perform multiple read operations
			for j := 0; j < 100; j++ {
				_ = extractor.ExtractTypeDoc("TestType")
				_ = extractor.ExtractFunctionDoc("TestFunc")
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test edge cases in field documentation processing
func TestStoreFieldDocByJSONTag_ComplexTags(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Test with complex JSON tag
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "ComplexField"}},
		Tag:   &ast.BasicLit{Value: "`json:\"complex_field,omitempty\" validate:\"required\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "Complex field description", doc)

	if fieldDoc, exists := doc.Fields["complex_field"]; !exists {
		t.Error("Expected field 'complex_field' to be stored")
	} else if fieldDoc.Description != "Complex field description" {
		t.Errorf("Expected description 'Complex field description', got '%s'", fieldDoc.Description)
	}

	// Test with JSON tag that has multiple options
	field2 := &ast.Field{
		Names: []*ast.Ident{{Name: "MultiOptionField"}},
		Tag:   &ast.BasicLit{Value: "`json:\"multi_option,omitempty,string\"`"},
	}

	extractor.storeFieldDocByJSONTag(field2, "Multi option field", doc)

	if fieldDoc, exists := doc.Fields["multi_option"]; !exists {
		t.Error("Expected field 'multi_option' to be stored")
	} else if fieldDoc.Description != "Multi option field" {
		t.Errorf("Expected description 'Multi option field', got '%s'", fieldDoc.Description)
	}
}

// Test function declaration processing
func TestProcessFuncDecl_EdgeCases(t *testing.T) {
	extractor := NewDocExtractor()

	// Test with function that has no doc comment
	funcDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: "TestFunc"},
		Doc:  nil,
	}

	extractor.processFuncDecl(funcDecl)

	doc := extractor.ExtractFunctionDoc("TestFunc")
	if doc.Description != "" {
		t.Error("Expected empty description for function without doc comment")
	}

	// Test with function that has doc comment
	funcDeclWithDoc := &ast.FuncDecl{
		Name: &ast.Ident{Name: "DocumentedFunc"},
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// This function does something"},
				{Text: "// and continues here"},
			},
		},
	}

	extractor.processFuncDecl(funcDeclWithDoc)

	doc = extractor.ExtractFunctionDoc("DocumentedFunc")
	if doc.Description == "" {
		t.Error("Expected description for function with doc comment")
	}
}

// Test inspectNode with various node types
func TestInspectNode_NodeTypes(t *testing.T) {
	extractor := NewDocExtractor()

	// Test with non-GenDecl, non-FuncDecl node
	ident := &ast.Ident{Name: "test"}
	result := extractor.inspectNode(ident)
	if !result {
		t.Error("inspectNode should return true to continue traversal")
	}

	// Test with nil node
	result = extractor.inspectNode(nil)
	if !result {
		t.Error("inspectNode should handle nil node gracefully")
	}
}

// Test field documentation storage with multiple names
func TestStoreFieldDocumentation_MultipleNames(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create field with multiple names (like in var declarations)
	field := &ast.Field{
		Names: []*ast.Ident{
			{Name: "Field1"},
			{Name: "Field2"},
			{Name: "Field3"},
		},
		Tag: &ast.BasicLit{Value: "`json:\"json_name\"`"},
	}

	extractor.storeFieldDocumentation(field, "Multi-name field", doc)

	// Should store documentation for all field names
	for _, name := range []string{"Field1", "Field2", "Field3"} {
		if fieldDoc, exists := doc.Fields[name]; !exists {
			t.Errorf("Expected field '%s' to be stored", name)
		} else if fieldDoc.Description != "Multi-name field" {
			t.Errorf("Expected description 'Multi-name field' for %s, got '%s'", name, fieldDoc.Description)
		}
	}

	// Should also store by JSON tag name
	if fieldDoc, exists := doc.Fields["json_name"]; !exists {
		t.Error("Expected field 'json_name' to be stored")
	} else if fieldDoc.Description != "Multi-name field" {
		t.Errorf("Expected description 'Multi-name field' for json_name, got '%s'", fieldDoc.Description)
	}
}
