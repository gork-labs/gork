package api

import (
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"gopkg.in/yaml.v3"
)

// Test the continue statement in processDirectoryEntry for non-Go files
func TestProcessDirectoryEntry_ContinueForNonGoFiles(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()
	
	// Create a temp directory with mixed files
	tempDir := t.TempDir()
	
	// Create non-Go files that should trigger continue
	nonGoFiles := []string{"README.md", "config.json", "data.txt", "script.sh"}
	for _, fname := range nonGoFiles {
		if err := os.WriteFile(filepath.Join(tempDir, fname), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	
	// Create a Go file that should be processed
	goFile := filepath.Join(tempDir, "test.go")
	goContent := `package test
// TestType is documented
type TestType struct {}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Process the directory - this will loop through all files
	// and the continue statement will be hit for non-Go files
	err := extractor.processDirectoryEntry(tempDir, &mockDirEntryFinal{name: filepath.Base(tempDir), isDir: true}, fset)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Verify the Go file was processed
	doc := extractor.ExtractTypeDoc("TestType")
	if doc.Description != "TestType is documented" {
		t.Errorf("Expected Go file to be processed, got description: %s", doc.Description)
	}
}

// Test setSliceFieldValue with the specific path for empty allValues and non-empty paramValue
func TestSetSliceFieldValue_EmptyAllValuesNonEmptyParamValue(t *testing.T) {
	type TestStruct struct {
		Tags []string `json:"tags"`
	}
	
	req := TestStruct{}
	reqValue := reflect.ValueOf(&req).Elem()
	field, _ := reqValue.Type().FieldByName("Tags")
	fieldValue := reqValue.FieldByName("Tags")
	
	// Call with empty allValues and non-empty paramValue containing comma-separated values
	// This triggers the len(parts) == 0 && paramValue != "" condition
	setSliceFieldValue(fieldValue, field, "tag1,tag2,tag3", []string{})
	
	if len(req.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(req.Tags))
	}
	
	expected := []string{"tag1", "tag2", "tag3"}
	for i, tag := range req.Tags {
		if tag != expected[i] {
			t.Errorf("Expected tag[%d] = %s, got %s", i, expected[i], tag)
		}
	}
}

// Test reflectTypeToSchemaInternal with channel type
func TestReflectTypeToSchemaInternal_ChannelCase(t *testing.T) {
	registry := make(map[string]*Schema)
	
	// Test with reflect.Chan which should be handled in the explicit cases
	chanType := reflect.TypeOf(make(chan int))
	schema := reflectTypeToSchemaInternal(chanType, registry, false)
	
	if schema == nil {
		t.Error("Expected non-nil schema")
	}
	if schema.Type != "object" {
		t.Errorf("Expected object type for channel, got %s", schema.Type)
	}
}

// Test buildBasicTypeSchema with interface{} type
func TestBuildBasicTypeSchema_InterfaceType(t *testing.T) {
	registry := make(map[string]*Schema)
	
	// Test with interface{} which represents any type
	interfaceType := reflect.TypeOf((*interface{})(nil)).Elem()
	schema := buildBasicTypeSchemaWithRegistry(interfaceType, registry)
	
	if schema == nil {
		t.Error("Expected non-nil schema for interface{}")
	}
}

// Mock dir entry for final tests
type mockDirEntryFinal struct {
	name  string
	isDir bool
}

func (m *mockDirEntryFinal) Name() string               { return m.name }
func (m *mockDirEntryFinal) IsDir() bool                { return m.isDir }
func (m *mockDirEntryFinal) Type() os.FileMode          { return 0 }
func (m *mockDirEntryFinal) Info() (os.FileInfo, error) { return nil, nil }

// The unreachable panic at the end of buildBasicTypeSchema is truly unreachable
// because all reflect.Kind values are handled in the switch statement.
// This is defensive programming for future Go versions.

// Test the default case in reflectTypeToSchemaInternal 
// Now that buildBasicTypeSchema is fixed, we need to find what still triggers the default case
func TestReflectTypeToSchemaInternal_DefaultCase(t *testing.T) {
	registry := make(map[string]*Schema)
	
	// Try to find a type that hits the default case in reflectTypeToSchemaInternal
	// Looking at the switch statement, Ptr should go to default since it's not explicitly listed
	ptrType := reflect.TypeOf((*int)(nil))
	schema := reflectTypeToSchemaInternal(ptrType, registry, false)
	
	if schema == nil {
		t.Error("Expected non-nil schema")
	}
	
	// Since it goes through buildBasicTypeSchemaWithRegistry -> reflectTypeToSchemaInternal(t.Elem())
	// it should resolve to integer for *int
	if schema.Type != "integer" {
		t.Errorf("Expected integer type for *int, got %s", schema.Type)
	}
	
	// Also test other types that might hit the default case
	// Based on the updated switch statement, let's see what's not explicitly covered
	testTypes := []reflect.Type{
		reflect.TypeOf((*string)(nil)),  // *string -> Ptr case, then String
		reflect.TypeOf((**int)(nil)),    // **int -> nested pointers
	}
	
	for _, typ := range testTypes {
		schema := reflectTypeToSchemaInternal(typ, registry, false)
		if schema == nil {
			t.Errorf("Expected non-nil schema for type %v", typ)
		}
	}
}

// Test comprehensive type coverage to ensure we don't miss any cases
func TestBuildBasicTypeSchema_ComprehensiveCoverage(t *testing.T) {
	registry := make(map[string]*Schema)
	
	// Test all the type cases we can to ensure maximum coverage
	testTypes := []struct {
		typ      reflect.Type
		expected string
	}{
		{reflect.TypeOf(true), "boolean"},
		{reflect.TypeOf(int(0)), "integer"},
		{reflect.TypeOf(""), "string"},
		{reflect.TypeOf(float64(0)), "number"},
		{reflect.TypeOf(complex64(0)), "object"}, // Complex -> object
		{reflect.TypeOf(make(chan int)), "object"}, // Chan -> object
		{reflect.TypeOf(func() {}), "object"},      // Func -> object
		{reflect.TypeOf((*any)(nil)).Elem(), "object"}, // Interface -> object
		{reflect.TypeOf(map[string]int{}), "object"},    // Map -> object
		{reflect.TypeOf(unsafe.Pointer(nil)), "object"}, // UnsafePointer -> object
		{reflect.TypeOf(uintptr(0)), "integer"}, // Uintptr -> integer
	}
	
	for _, test := range testTypes {
		schema := buildBasicTypeSchemaWithRegistry(test.typ, registry)
		if schema == nil {
			t.Errorf("Got nil schema for type %v", test.typ)
			continue
		}
		if schema.Type != test.expected {
			t.Errorf("For type %v, expected %s, got %s", test.typ, test.expected, schema.Type)
		}
	}
}

// Test UnmarshalYAML error case
func TestSchema_UnmarshalYAML_ErrorCase(t *testing.T) {
	var s Schema
	
	// Test with a function that returns an error
	err := s.UnmarshalYAML(func(interface{}) error {
		// This will be called by UnmarshalYAML and should return an error
		return &yaml.TypeError{Errors: []string{"test unmarshal error"}}
	})
	
	if err == nil {
		t.Error("Expected error from invalid YAML unmarshal")
	}
}

// Test a subdirectory to ensure we process it correctly
func TestProcessDirectoryEntry_Subdirectory(t *testing.T) {
	extractor := NewDocExtractor()
	fset := token.NewFileSet()
	
	// Create a temp directory with a subdirectory
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Put a Go file in the subdirectory
	goFile := filepath.Join(subDir, "sub.go")
	goContent := `package sub
// SubType is in subdirectory
type SubType struct {}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Also add a non-Go file to ensure continue is hit
	if err := os.WriteFile(filepath.Join(subDir, "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Process the subdirectory
	err := extractor.processDirectoryEntry(subDir, &mockDirEntryFinal{name: "subdir", isDir: true}, fset)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Verify the Go file in subdirectory was processed
	doc := extractor.ExtractTypeDoc("SubType")
	if !strings.Contains(doc.Description, "SubType is in subdirectory") {
		t.Errorf("Expected subdirectory Go file to be processed, got: %s", doc.Description)
	}
}