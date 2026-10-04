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
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fixtures\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	d := NewDocExtractor()
	if err := d.ParseDirectory(dir); err != nil {
		t.Fatalf("parse: %v", err)
	}

	td := d.ExtractTypeDoc("example.com/fixtures.Foo")
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

	// Create a field with gork tag (the function name is misleading but kept for compatibility)
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`gork:\"test_field,omitempty\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should add entry for the gork tag name
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

	// Create a field with gork tag without comma
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`gork:\"test_field\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should add entry for the gork tag name
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

	// Create a field with empty gork name (just comma)
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`gork:\",omitempty\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	// Should not add any entries since the gork name is empty
	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields stored, got %d", len(doc.Fields))
	}
}

func TestStoreFieldDocByJSONTag_JSONName(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`json:\"test_field,omitempty\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	if fieldDoc := doc.Fields["test_field"]; fieldDoc.Description != "test description" {
		t.Errorf("Expected description for 'test_field', got %v", doc.Fields)
	}
}

func TestStoreFieldDocByJSONTag_SkippedJSONName(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	field := &ast.Field{
		Names: []*ast.Ident{{Name: "TestField"}},
		Tag:   &ast.BasicLit{Value: "`json:\"-\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "test description", doc)

	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields stored, got %v", doc.Fields)
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
	err := extractor.parseFile("/non/existent/file.go", "example.com/fixtures", fset)
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

	extractor.processStructFields(st, "example.com/fixtures.T", doc)

	if doc.Fields == nil {
		t.Error("Expected Fields map to be initialized")
	}
	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields, got %d", len(doc.Fields))
	}
}

func TestExtractFieldDescription_WithCommentOnly(t *testing.T) {
	extractor := NewDocExtractor()
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "CommentField"}},
		Doc:   nil, // Ensure Doc is nil
		Comment: &ast.CommentGroup{List: []*ast.Comment{{
			Text: "// This is a line comment.",
		}}},
	}

	desc := extractor.extractFieldDescription(field)
	if desc != "This is a line comment." {
		t.Errorf("Expected 'This is a line comment.', got %q", desc)
	}
}

func TestProcessStructFields_EmptyDescription(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Struct with a field that has no doc or comment
	st := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Names:   []*ast.Ident{{Name: "NoDocField"}},
					Doc:     nil,
					Comment: nil,
				},
			},
		},
	}

	extractor.processStructFields(st, "example.com/fixtures.T", doc)

	// No documentation should be stored for NoDocField
	if len(doc.Fields) != 0 {
		t.Errorf("Expected no fields stored, got %d", len(doc.Fields))
	}
}

func TestProcessDirectoryEntry_ParseFileError(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()

	// Create a temporary directory
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fixtures\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Create a malformed Go file in the temporary directory
	malformedFilePath := filepath.Join(dir, "malformed.go")
	if err := os.WriteFile(malformedFilePath, []byte(`package main
func main { // Syntax error: missing parentheses
	}
`), 0o644); err != nil {
		t.Fatalf("Failed to write malformed file: %v", err)
	}

	// Create a valid Go file in the same directory
	validFilePath := filepath.Join(dir, "valid.go")
	if err := os.WriteFile(validFilePath, []byte(`package main

// MyFunc is a valid function
func MyFunc() {}
`), 0o644); err != nil {
		t.Fatalf("Failed to write valid file: %v", err)
	}

	// Process the directory entry (which is a directory)
	err := extractor.processDirectoryEntry(dir, &mockDirEntry{name: filepath.Base(dir), isDir: true}, fset)
	if err != nil {
		t.Errorf("Expected no error for directory with malformed file, got %v", err)
	}

	// Verify that only the valid file's documentation was extracted
	doc := extractor.ExtractFunctionDoc("MyFunc")
	if doc.Description != "MyFunc is a valid function" {
		t.Errorf("Expected 'MyFunc is a valid function', got %q", doc.Description)
	}
}

func TestProcessGenDecl_NoDocComment(t *testing.T) {
	extractor := NewDocExtractor()
	decl := &ast.GenDecl{
		Doc: nil, // No doc comment
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("MyType"),
				Type: &ast.StructType{},
			},
		},
	}

	// Expect no panic or error, and no documentation stored
	extractor.processGenDecl(decl, "example.com/fixtures")
	if _, ok := extractor.docs["example.com/fixtures.MyType"]; ok {
		t.Error("Expected no documentation to be stored for a type without a doc comment")
	}
}

func TestProcessGenDecl_NonTypeDeclaration(t *testing.T) {
	extractor := NewDocExtractor()
	decl := &ast.GenDecl{
		Doc: &ast.CommentGroup{List: []*ast.Comment{{
			Text: "// Some var",
		}}},
		Tok: token.VAR, // Not token.TYPE
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("myVar")},
			},
		},
	}

	// Expect no panic or error, and no documentation stored
	extractor.processGenDecl(decl, "example.com/fixtures")
	if _, ok := extractor.docs["example.com/fixtures.myVar"]; ok {
		t.Error("Expected no documentation to be stored for a non-type declaration")
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

func TestStoreFieldDocByJSONTag_WithGorkTag(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create a field with gork tag
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "UserID"}},
		Tag:   &ast.BasicLit{Value: "`gork:\"userID\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "ID of the user", doc)

	// Should add entry for the gork tag name
	if len(doc.Fields) != 1 {
		t.Errorf("Expected 1 field stored, got %d", len(doc.Fields))
	}

	if fieldDoc, exists := doc.Fields["userID"]; !exists {
		t.Error("Expected field 'userID' to be stored")
	} else if fieldDoc.Description != "ID of the user" {
		t.Errorf("Expected description 'ID of the user', got '%s'", fieldDoc.Description)
	}
}

func TestStoreFieldDocByJSONTag_WithGorkTagAndOptions(t *testing.T) {
	extractor := NewDocExtractor()
	doc := &Documentation{Fields: make(map[string]FieldDoc)}

	// Create a field with gork tag with options
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "Username"}},
		Tag:   &ast.BasicLit{Value: "`gork:\"username,omitempty\"`"},
	}

	extractor.storeFieldDocByJSONTag(field, "Username of the user", doc)

	// Should add entry for the gork tag name (before comma)
	if len(doc.Fields) != 1 {
		t.Errorf("Expected 1 field stored, got %d", len(doc.Fields))
	}

	if fieldDoc, exists := doc.Fields["username"]; !exists {
		t.Error("Expected field 'username' to be stored")
	} else if fieldDoc.Description != "Username of the user" {
		t.Errorf("Expected description 'Username of the user', got '%s'", fieldDoc.Description)
	}
}

func TestParseDirectory_NoGoMod(t *testing.T) {
	err := NewDocExtractor().ParseDirectory(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "no go.mod file") {
		t.Errorf("err = %v, want an error about the missing go.mod file", err)
	}
}

func TestDocExtractor_ImportPathOfSubdirectory(t *testing.T) {
	extractor := parseFixtures(t, map[string]string{
		"internal/store/store.go": "package store\n\n// Row is a row.\ntype Row struct{}\n",
	})
	if got := extractor.ExtractTypeDoc("github.com/gork-labs/gork/internal/store.Row").Description; got != "Row is a row." {
		t.Errorf("description = %q, want the doc of Row", got)
	}
}
