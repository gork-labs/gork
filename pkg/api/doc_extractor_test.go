package api

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocExtractor(t *testing.T) {
	dir := t.TempDir()
	src := `package fixtures

// Foo represents something.
//
// Deprecated: use Bar instead.
type Foo struct {
    // ID is an identifier.
    ID string ` + "`json:\"id\"`" + `
}

// GetFoo returns foo.
// It does a thing.
func GetFoo() {}
`
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	d := NewDocExtractor()
	if err := d.ParseDirectory(dir); err != nil {
		t.Fatalf("parse: %v", err)
	}

	td := d.ExtractTypeDoc("Foo")
	if td.Description != "Foo represents something." {
		t.Errorf("got desc %q", td.Description)
	}
	fd := d.ExtractFunctionDoc("GetFoo")
	if fd.Description != "GetFoo returns foo. It does a thing." {
		t.Errorf("function desc mismatch: %q", fd.Description)
	}
}

func TestExtractFunctionDoc_Found(t *testing.T) {
	extractor := NewDocExtractor()

	// Add a function doc manually
	extractor.docs["TestFunc"] = Documentation{
		Description: "Test function description",
	}

	doc := extractor.ExtractFunctionDoc("TestFunc")
	if doc.Description != "Test function description" {
		t.Errorf("Expected 'Test function description', got '%s'", doc.Description)
	}
}

func TestExtractFunctionDoc_NotFound(t *testing.T) {
	extractor := NewDocExtractor()

	doc := extractor.ExtractFunctionDoc("NonExistentFunc")
	if doc.Description != "" {
		t.Errorf("Expected empty description for non-existent function, got '%s'", doc.Description)
	}
}

func TestStoreFieldDocByJSONTag_NoTag(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create a field without tag
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   nil,
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should not add any JSON tag entries
	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields stored, got %d", len(doc.Fields))
	}
}

func TestStoreFieldDocByJSONTag_EmptyJSONTag(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create a field with empty json tag
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`validate:\"required\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should not add any JSON tag entries since json tag is empty
	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields stored, got %d", len(doc.Fields))
	}
}

func TestStoreFieldDocByJSONTag_WithJSONTag(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create a field with json tag
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`json:\"test_field,omitempty\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should add entry for the JSON tag name
	if len(doc.Fields) != 1 {
		t.Errorf("Expected 1 field stored, got %d", len(doc.Fields))
	}

	if fieldDoc, exists := doc.Fields["test_field"]; !exists {
		t.Error("Expected field 'test_field' to be stored")
	} else if fieldDoc.Description != "test description" {
		t.Errorf("Expected description 'test description', got '%s'", fieldDoc.Description)
	}
}

func TestStoreFieldDocByJSONTag_WithJSONTagNoComma(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create a field with json tag without comma
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`json:\"test_field\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should add entry for the JSON tag name
	if len(doc.Fields) != 1 {
		t.Errorf("Expected 1 field stored, got %d", len(doc.Fields))
	}

	if fieldDoc, exists := doc.Fields["test_field"]; !exists {
		t.Error("Expected field 'test_field' to be stored")
	} else if fieldDoc.Description != "test description" {
		t.Errorf("Expected description 'test description', got '%s'", fieldDoc.Description)
	}
}

func TestStoreFieldDocByJSONTag_EmptyJSONName(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create a field with empty json name (just comma)
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`json:\",omitempty\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should not add any entries since the JSON name is empty
	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields stored, got %d", len(doc.Fields))
	}
}

func TestProcessDirectoryEntry_NonDirectory(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Write some Go code
	tempFile.WriteString(`package test
// TestFunc does something
func TestFunc() {}
`)
	tempFile.Close()

	// Process the file entry (not a directory)
	err = extractor.processDirectoryEntry(tempFile.Name(), &mockDirEntry{name: filepath.Base(tempFile.Name()), isDir: false}, fset)
	if err != nil {
		t.Errorf("Expected no error for non-directory, got %v", err)
	}
}

func TestProcessDirectoryEntry_VendorDirectory(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()

	// Process vendor directory
	err := extractor.processDirectoryEntry("/some/path/vendor", &mockDirEntry{name: "vendor", isDir: true}, fset)
	if err != filepath.SkipDir {
		t.Errorf("Expected filepath.SkipDir for vendor directory, got %v", err)
	}
}

func TestParseFile_Error(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()

	// Try to parse a non-existent file
	err := extractor.parseFile("/non/existent/file.go", fset)
	if err == nil {
		t.Error("Expected error when parsing non-existent file")
	}
}

func TestParseDirectory_Error(t *testing.T) {
	extractor := NewDocExtractor()

	// Try to parse a non-existent directory
	err := extractor.ParseDirectory("/nonexistent/directory")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestExtractDescription_EmptyComment(t *testing.T) {
	result := extractDescription("")
	if result != "" {
		t.Errorf("Expected empty string for empty comment, got '%s'", result)
	}
}

func TestExtractDescription_WithCommentMarkers(t *testing.T) {
	comment := `// This is a test function
// It does something useful
//
// Second paragraph`

	result := extractDescription(comment)
	// extractDescription should only return the first paragraph
	if !strings.Contains(result, "This is a test function") || !strings.Contains(result, "It does something useful") {
		t.Errorf("Expected first paragraph only, got '%s'", result)
	}
}

func TestExtractDescription_WithBlockComments(t *testing.T) {
	comment := `/* This is a test function
 * It does something useful */`

	result := extractDescription(comment)
	// The function should handle block comments
	if !strings.Contains(result, "This is a test function") || !strings.Contains(result, "It does something useful") {
		t.Errorf("Expected block comment content, got '%s'", result)
	}
}

func TestProcessStructFields_NoFields(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{}

	// Empty struct
	st := &ast.StructType{
		Fields: &ast.FieldList{List: []*ast.Field{}},
	}

	extractor.processStructFields(st, doc)

	if doc.Fields == nil {
		t.Error("Expected Fields map to be initialized")
	}
	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields, got %d", len(doc.Fields))
	}
}

// Mock dir entry for testing
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() os.FileMode          { return 0 }
func (m *mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }
