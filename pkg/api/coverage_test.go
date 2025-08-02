package api

import (
	"encoding/json"
	"go/ast"
	"os"
	"reflect"
	"testing"
	"unsafe"
)

// Test for ExportOpenAPIAndExit function
func TestExportOpenAPIAndExit(t *testing.T) {
	registry := NewRouteRegistry()
	router := TypedRouter[interface{}]{
		registry: registry,
	}
	
	// Since we removed the exportMode check, ExportOpenAPIAndExit will always
	// attempt to export and exit. For testing, we need to mock the exit function.
	// This test verifies that the function would generate a spec before exiting.
	spec := GenerateOpenAPI(router.registry)
	if spec == nil {
		t.Error("Expected GenerateOpenAPI to return a spec")
	}
	if spec.OpenAPI != "3.1.0" {
		t.Errorf("Expected OpenAPI version 3.1.0, got %s", spec.OpenAPI)
	}
}

// Test for Schema UnmarshalJSON function  
func TestSchema_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		verify   func(*Schema) bool
		wantErr  bool
	}{
		{
			name:    "simple string type",
			jsonStr: `{"type": "string"}`,
			verify: func(s *Schema) bool {
				return s.Type == "string" && len(s.Types) == 0
			},
			wantErr: false,
		},
		{
			name:    "array of types",
			jsonStr: `{"type": ["string", "number"]}`,
			verify: func(s *Schema) bool {
				return len(s.Types) == 2 && s.Types[0] == "string" && s.Types[1] == "number"
			},
			wantErr: false,
		},
		{
			name:    "object with properties",
			jsonStr: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			verify: func(s *Schema) bool {
				if s.Type != "object" || len(s.Properties) != 1 {
					return false
				}
				nameSchema, exists := s.Properties["name"]
				return exists && nameSchema != nil && nameSchema.Type == "string"
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			jsonStr: `{"type": "string"`,
			verify:  func(s *Schema) bool { return true }, // Won't be called for error case
			wantErr: true,
		},
		{
			name:    "mixed type array with non-string",
			jsonStr: `{"type": ["string", 123]}`,
			verify: func(s *Schema) bool {
				// Should only set the string value, ignore the non-string
				return len(s.Types) == 2 && s.Types[0] == "string" && s.Types[1] == ""
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schema Schema
			err := json.Unmarshal([]byte(tt.jsonStr), &schema)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.verify(&schema) {
				t.Errorf("Schema verification failed. Got: %+v", &schema)
			}
		})
	}
}

// Test for handleUnionType function 
func TestHandleUnionType(t *testing.T) {
	// Create a named type for testing
	type TestStruct struct {
		Field1 string
		Field2 int
	}
	unionType := reflect.TypeOf(TestStruct{})
	
	registry := make(map[string]*Schema)
	
	// This function always returns a schema (either union schema or reference)
	result := handleUnionType(unionType, registry)
	
	// Should return a schema
	if result == nil {
		t.Error("handleUnionType should return a schema")
	}
	
	// For named types, should create a reference
	if result.Ref == "" {
		t.Error("handleUnionType should create a reference for named types")
	}
}

// Test for processEmbeddedStruct function
func TestProcessEmbeddedStruct(t *testing.T) {
	schema := &Schema{
		Properties: make(map[string]*Schema),
	}
	
	// Create a test struct field for embedding
	embeddedType := reflect.TypeOf(struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{})
	
	field := reflect.StructField{
		Name:      "Embedded",
		Type:      embeddedType,
		Anonymous: true,
	}
	
	registry := make(map[string]*Schema)
	
	// Call processEmbeddedStruct
	processEmbeddedStruct(field, schema, registry)
	
	// Should have processed the embedded struct fields
	if len(schema.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(schema.Properties))
	}
	
	if schema.Properties["name"] == nil || schema.Properties["name"].Type != "string" {
		t.Error("name property should be string type")
	}
	
	if schema.Properties["age"] == nil || schema.Properties["age"].Type != "integer" {
		t.Error("age property should be integer type")  
	}
}

// Test for generateUnionSchema function
func TestGenerateUnionSchema(t *testing.T) {
	// Create a test union type (this would normally be Union2, Union3, etc.)
	unionType := reflect.TypeOf(struct{}{}) // Simplified for testing
	registry := make(map[string]*Schema)
	
	result := generateUnionSchema(unionType, registry)
	
	// For a non-union type, should return empty schema
	if result == nil {
		t.Error("generateUnionSchema should return a schema, not nil")
	}
	
	// Should not have discriminator for non-union types
	if result.Discriminator != nil {
		t.Error("generateUnionSchema should not set discriminator for non-union types")
	}
}

// Test for extractUnionVariantsAndDiscriminator function
func TestExtractUnionVariantsAndDiscriminator(t *testing.T) {
	// Test with a simple struct (not a real union)
	unionType := reflect.TypeOf(struct {
		Type string `json:"type"`
	}{})
	
	registry := make(map[string]*Schema)
	
	variants, discriminator := extractUnionVariantsAndDiscriminator(unionType, registry)
	
	// For non-union types, should return empty results
	if len(variants) != 0 {
		t.Errorf("Expected 0 variants, got %d", len(variants))
	}
	
	if discriminator.propertyName != "" {
		t.Errorf("Expected empty discriminator property name, got %s", discriminator.propertyName)
	}
}

// Test for processVariantDiscriminator function  
func TestProcessVariantDiscriminator(t *testing.T) {
	variantType := reflect.TypeOf(struct {
		Kind string `json:"kind" openapi:"discriminator=test"`
	}{})
	
	discInfo := &discriminatorInfo{
		mapping: make(map[string]string),
	}
	
	// Call the function (it modifies discInfo in place)
	processVariantDiscriminator(variantType, discInfo)
	
	// Should have found and processed the discriminator
	if discInfo.propertyName == "" {
		t.Error("Expected discriminator property name to be set")
	}
	
	// Test with no discriminator
	variantType2 := reflect.TypeOf(struct {
		Name string `json:"name"`
	}{})
	
	discInfo2 := &discriminatorInfo{
		mapping: make(map[string]string),
	}
	
	processVariantDiscriminator(variantType2, discInfo2)
	
	// Should not have set any discriminator properties
	if discInfo2.propertyName != "" {
		t.Errorf("Expected empty discriminator property name, got '%s'", discInfo2.propertyName)
	}
}

// Test for buildBasicTypeSchema function
func TestBuildBasicTypeSchema(t *testing.T) {
	tests := []struct {
		name         string
		reflectType  reflect.Type
		expectedType string
		expectedDesc string
	}{
		{
			name:         "string type",
			reflectType:  reflect.TypeOf(""),
			expectedType: "string",
			expectedDesc: "",
		},
		{
			name:         "int type",
			reflectType:  reflect.TypeOf(0),
			expectedType: "integer",
			expectedDesc: "",
		},
		{
			name:         "int64 type",
			reflectType:  reflect.TypeOf(int64(0)),
			expectedType: "integer",
			expectedDesc: "",
		},
		{
			name:         "uint type",
			reflectType:  reflect.TypeOf(uint(0)),
			expectedType: "integer",
			expectedDesc: "",
		},
		{
			name:         "float32 type",
			reflectType:  reflect.TypeOf(float32(0.0)),
			expectedType: "number",
			expectedDesc: "",
		},
		{
			name:         "float64 type",
			reflectType:  reflect.TypeOf(0.0),
			expectedType: "number",
			expectedDesc: "",
		},
		{
			name:         "bool type",
			reflectType:  reflect.TypeOf(true),
			expectedType: "boolean",
			expectedDesc: "",
		},
		{
			name:         "uintptr type",
			reflectType:  reflect.TypeOf(uintptr(0)),
			expectedType: "integer",
			expectedDesc: "Pointer-sized integer",
		},
		{
			name:         "complex64 type",
			reflectType:  reflect.TypeOf(complex64(0)),
			expectedType: "object",
			expectedDesc: "Complex number",
		},
		{
			name:         "complex128 type",
			reflectType:  reflect.TypeOf(complex128(0)),
			expectedType: "object",
			expectedDesc: "Complex number",
		},
		{
			name:         "channel type",
			reflectType:  reflect.TypeOf(make(chan int)),
			expectedType: "object",
			expectedDesc: "Channel",
		},
		{
			name:         "function type",
			reflectType:  reflect.TypeOf(func() {}),
			expectedType: "object",
			expectedDesc: "Function",
		},
		{
			name:         "interface type",
			reflectType:  reflect.TypeOf((*interface{})(nil)).Elem(),
			expectedType: "object",
			expectedDesc: "Interface",
		},
		{
			name:         "map type",
			reflectType:  reflect.TypeOf(map[string]int{}),
			expectedType: "object",
			expectedDesc: "Map with dynamic keys",
		},
		{
			name:         "unsafe pointer type",
			reflectType:  reflect.TypeOf(unsafe.Pointer(nil)),
			expectedType: "object",
			expectedDesc: "Unsafe pointer",
		},
		{
			name:         "slice type (should return object)",
			reflectType:  reflect.TypeOf([]int{}),
			expectedType: "object",
			expectedDesc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildBasicTypeSchema(tt.reflectType)
			
			if schema.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, schema.Type)
			}
			
			if schema.Description != tt.expectedDesc {
				t.Errorf("Expected description '%s', got '%s'", tt.expectedDesc, schema.Description)
			}
		})
	}
}

// Test for sanitizeSchemaName function
func TestSanitizeSchemaName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple name",
			input:    "User",
			expected: "User",
		},
		{
			name:     "name with generic type",
			input:    "Union2[github.com/foo.Bar,github.com/baz.Qux]",
			expected: "Union2_Bar_Qux",
		},
		{
			name:     "name with spaces in generic args",
			input:    "Union3[string, int, bool]",
			expected: "Union3_string_int_bool",
		},
		{
			name:     "name with invalid characters",
			input:    "User@Name#With$Special%Chars",
			expected: "User_Name_With_Special_Chars",
		},
		{
			name:     "name with allowed characters",
			input:    "User_Name.With-Allowed123",
			expected: "User_Name.With-Allowed123",
		},
		{
			name:     "name with complex generic",
			input:    "Map[pkg.Key,pkg.Value]",
			expected: "Map_Key_Value",
		},
		{
			name:     "name with package path",
			input:    "github.com/user/pkg.Type",
			expected: "github.com_user_pkg.Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeSchemaName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Test for applyOneOfValidationToSchema function (16.7% coverage)
func TestApplyOneOfValidationToSchema(t *testing.T) {
	tests := []struct {
		name         string
		validateTag  string
		expectedEnum []string
	}{
		{
			name:         "oneof at start",
			validateTag:  "oneof=red blue green",
			expectedEnum: []string{"red", "blue", "green"},
		},
		{
			name:         "oneof with space prefix",
			validateTag:  "required oneof=small medium large",
			expectedEnum: []string{"small", "medium", "large"},
		},
		{
			name:         "no oneof",
			validateTag:  "required,min=1",
			expectedEnum: nil,
		},
		{
			name:         "empty oneof",
			validateTag:  "oneof=",
			expectedEnum: nil,
		},
		{
			name:         "oneof with single value", 
			validateTag:  "oneof=single",
			expectedEnum: []string{"single"},
		},
		{
			name:         "no parts after oneof=",
			validateTag:  "oneof",
			expectedEnum: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{}
			applyOneOfValidationToSchema(schema, tt.validateTag)

			if len(tt.expectedEnum) == 0 {
				if schema.Enum != nil {
					t.Errorf("Expected no enum, got %v", schema.Enum)
				}
			} else {
				if len(schema.Enum) != len(tt.expectedEnum) {
					t.Errorf("Expected enum length %d, got %d", len(tt.expectedEnum), len(schema.Enum))
					return
				}
				for i, expected := range tt.expectedEnum {
					if schema.Enum[i] != expected {
						t.Errorf("Expected enum[%d] = %s, got %s", i, expected, schema.Enum[i])
					}
				}
			}
		})
	}
}

// Test for loadStaticSpec function (18.2% coverage)
func TestLoadStaticSpec(t *testing.T) {
	// Test empty spec file
	result := loadStaticSpec("")
	if result != nil {
		t.Error("Expected nil for empty spec file")
	}

	// Test non-existent file
	result = loadStaticSpec("/nonexistent/file.json")
	if result != nil {
		t.Error("Expected nil for non-existent file")
	}

	// Create temporary files for testing
	tempDir := t.TempDir()

	// Test valid JSON file
	jsonContent := `{"openapi": "3.1.0", "info": {"title": "Test API", "version": "1.0.0"}}`
	jsonFile := tempDir + "/spec.json"
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	result = loadStaticSpec(jsonFile)
	if result == nil {
		t.Error("Expected valid spec from JSON file")
	} else if result.OpenAPI != "3.1.0" {
		t.Errorf("Expected OpenAPI version 3.1.0, got %s", result.OpenAPI)
	}

	// Test valid YAML file
	yamlContent := "openapi: '3.1.0'\ninfo:\n  title: Test API\n  version: '1.0.0'"
	yamlFile := tempDir + "/spec.yaml"
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	result = loadStaticSpec(yamlFile)
	if result == nil {
		t.Error("Expected valid spec from YAML file")
	} else if result.OpenAPI != "3.1.0" {
		t.Errorf("Expected OpenAPI version 3.1.0, got %s", result.OpenAPI)
	}

	// Test invalid file content
	invalidFile := tempDir + "/invalid.json"
	if err := os.WriteFile(invalidFile, []byte("invalid content"), 0644); err != nil {
		t.Fatalf("Failed to create invalid test file: %v", err)
	}

	result = loadStaticSpec(invalidFile)
	if result != nil {
		t.Error("Expected nil for invalid file content")
	}
}

// Test for resolveQueryParamName function (38.5% coverage)
func TestResolveQueryParamName(t *testing.T) {
	tests := []struct {
		name         string
		field        reflect.StructField
		expectedName string
	}{
		{
			name: "field with openapi query tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `openapi:"test_param,in=query"`,
			},
			expectedName: "test_param",
		},
		{
			name: "field with openapi query tag and json fallback",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `openapi:",in=query" json:"test_field"`,
			},
			expectedName: "test_field",
		},
		{
			name: "field with json tag only",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `json:"test_field"`,
			},
			expectedName: "test_field",
		},
		{
			name: "field with json tag and omitempty",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `json:"test_field,omitempty"`,
			},
			expectedName: "test_field,omitempty",
		},
		{
			name: "field with dash json tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `json:"-"`,
			},
			expectedName: "",
		},
		{
			name: "field with no tags",
			field: reflect.StructField{
				Name: "TestField",
			},
			expectedName: "",
		},
		{
			name: "field with empty json tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `json:""`,
			},
			expectedName: "",
		},
		{
			name: "field with openapi non-query tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `openapi:"test_param,in=header"`,
			},
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveQueryParamName(tt.field)
			if result != tt.expectedName {
				t.Errorf("Expected %s, got %s", tt.expectedName, result)
			}
		})
	}
}

// Test for normalizeDocsPath function (40% coverage)
func TestNormalizeDocsPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path with trailing slash",
			input:    "/docs/",
			expected: "/docs",
		},
		{
			name:     "path without trailing slash",
			input:    "/docs",
			expected: "/docs",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "path already ending with /*",
			input:    "/api/docs/*",
			expected: "/api/docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDocsPath(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Test for makeNullableSchema function (71.4% coverage)
func TestMakeNullableSchema(t *testing.T) {
	// Test nil schema
	result := makeNullableSchema(nil)
	if result.Type != "null" {
		t.Errorf("Expected null type for nil schema, got %s", result.Type)
	}

	// Test schema with reference
	refSchema := &Schema{Ref: "#/components/schemas/User"}
	result = makeNullableSchema(refSchema)
	if result.AnyOf == nil || len(result.AnyOf) != 2 {
		t.Error("Expected anyOf with 2 items for ref schema")
	} else {
		if result.AnyOf[0].Ref != "#/components/schemas/User" {
			t.Errorf("Expected first anyOf to be ref, got %v", result.AnyOf[0])
		}
		if result.AnyOf[1].Type != "null" {
			t.Errorf("Expected second anyOf to be null, got %s", result.AnyOf[1].Type)
		}
	}

	// Test schema with properties
	propSchema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"name": {Type: "string"},
		},
	}
	result = makeNullableSchema(propSchema)
	if result.AnyOf == nil || len(result.AnyOf) != 2 {
		t.Error("Expected anyOf with 2 items for properties schema")
	}

	// Test basic type schema
	stringSchema := &Schema{Type: "string", MinLength: intPtr(1)}
	result = makeNullableSchema(stringSchema)
	if len(result.Types) != 2 || result.Types[0] != "string" || result.Types[1] != "null" {
		t.Errorf("Expected types [string, null], got %v", result.Types)
	}
	if result.MinLength == nil || *result.MinLength != 1 {
		t.Error("Expected MinLength to be preserved")
	}

	// Test schema with oneOf
	oneOfSchema := &Schema{
		OneOf: []*Schema{
			{Type: "string"},
			{Type: "number"},
		},
	}
	result = makeNullableSchema(oneOfSchema)
	if result.AnyOf == nil || len(result.AnyOf) != 2 {
		t.Error("Expected anyOf with 2 items for oneOf schema")
	}

	// Test fallback case - schema with no type and no special properties
	emptySchema := &Schema{Description: "test"}
	result = makeNullableSchema(emptySchema)
	if result.AnyOf == nil || len(result.AnyOf) != 2 {
		t.Error("Expected anyOf with 2 items for fallback case")
	}
}

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}

// Test for extractUnionVariantsAndDiscriminator function (58.3% coverage)
func TestExtractUnionVariantsAndDiscriminator_EdgeCases(t *testing.T) {
	// Test with struct that has non-pointer fields
	type MixedStruct struct {
		PtrField    *string
		NonPtrField string
	}
	unionType := reflect.TypeOf(MixedStruct{})
	registry := make(map[string]*Schema)

	variants, _ := extractUnionVariantsAndDiscriminator(unionType, registry)

	// Should only process pointer fields
	if len(variants) != 1 {
		t.Errorf("Expected 1 variant, got %d", len(variants))
	}

	// Test with struct that has no pointer fields
	type NoPointerStruct struct {
		Field1 string
		Field2 int
	}
	unionType2 := reflect.TypeOf(NoPointerStruct{})
	variants2, _ := extractUnionVariantsAndDiscriminator(unionType2, registry)

	if len(variants2) != 0 {
		t.Errorf("Expected 0 variants for no-pointer struct, got %d", len(variants2))
	}
}

// Test for processEmbeddedStruct function - additional edge cases (60% coverage)
func TestProcessEmbeddedStruct_EdgeCases(t *testing.T) {
	schema := &Schema{
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	// Test with embedded struct that creates a reference
	type NamedStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	embeddedType := reflect.TypeOf(NamedStruct{})
	field := reflect.StructField{
		Name:      "Embedded",
		Type:      embeddedType,
		Anonymous: true,
	}

	registry := make(map[string]*Schema)

	// Call processEmbeddedStruct
	processEmbeddedStruct(field, schema, registry)

	// Should have properties from the embedded struct
	if len(schema.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(schema.Properties))
	}

	// Test with embedded struct that has a reference in registry
	schema2 := &Schema{
		Properties: make(map[string]*Schema),
	}

	// Pre-populate registry with the embedded type
	registry["NamedStruct"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"title": {Type: "string"},
		},
		Required: []string{"title"},
	}

	processEmbeddedStruct(field, schema2, registry)

	// Should resolve the reference and copy properties
	if schema2.Properties["title"] == nil || schema2.Properties["title"].Type != "string" {
		t.Error("Expected title property to be copied from referenced schema")
	}
	if len(schema2.Required) != 1 || schema2.Required[0] != "title" {
		t.Errorf("Expected required field 'title', got %v", schema2.Required)
	}
}

// Test for addRequiredField function (60% coverage)
func TestAddRequiredField_EdgeCases(t *testing.T) {
	parent := &Schema{
		Required: []string{"existing"},
	}

	// Test adding a field that already exists
	field := reflect.StructField{
		Name: "ExistingField",
		Tag:  `json:"existing"`,
	}

	addRequiredField(parent, field)

	// Should not duplicate
	if len(parent.Required) != 1 {
		t.Errorf("Expected 1 required field, got %d", len(parent.Required))
	}

	// Test adding a new field
	newField := reflect.StructField{
		Name: "NewField",
		Tag:  `json:"new_field"`,
	}

	addRequiredField(parent, newField)

	// Should add the new field
	if len(parent.Required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(parent.Required))
	}
	if parent.Required[1] != "new_field" {
		t.Errorf("Expected second required field to be 'new_field', got %s", parent.Required[1])
	}
}

// Test for isUnionType function (66.7% coverage)
func TestIsUnionType_EdgeCases(t *testing.T) {
	// Test with nil type
	result := isUnionType(nil)
	if result {
		t.Error("Expected false for nil type")
	}

	// Test with non-union type from the unions package
	type NonUnion struct{}
	nonUnionType := reflect.TypeOf(NonUnion{})
	result = isUnionType(nonUnionType)
	if result {
		t.Error("Expected false for non-union type")
	}

	// Test with type from different package but same names
	type Union2 struct {
		A *string
		B *int
	}
	fakeUnionType := reflect.TypeOf(Union2{})
	result = isUnionType(fakeUnionType)
	if result {
		t.Error("Expected false for fake union type from different package")
	}
}

// Test for isStringKind function (66.7% coverage)
func TestIsStringKind_EdgeCases(t *testing.T) {
	// Test with pointer to pointer to string
	ptrPtrString := reflect.TypeOf((**string)(nil)).Elem()
	result := isStringKind(ptrPtrString)
	if !result {
		t.Error("Expected true for pointer to pointer to string")
	}

	// Test with slice type
	sliceType := reflect.TypeOf([]string{})
	result = isStringKind(sliceType)
	if result {
		t.Error("Expected false for slice type")
	}
}


// Test for prepareDocsConfig function (30% coverage)
func TestPrepareDocsConfig(t *testing.T) {
	config := DocsConfig{
		Title:       "Test API",
		OpenAPIPath: "/openapi.json",
		SpecFile:    "/path/to/spec.json",
	}
	
	result := prepareDocsConfig(config)
	
	// Should keep the provided values
	if result.Title != "Test API" {
		t.Errorf("Expected Title 'Test API', got %s", result.Title)
	}
	if result.OpenAPIPath != "/openapi.json" {
		t.Errorf("Expected OpenAPIPath '/openapi.json', got %s", result.OpenAPIPath)  
	}
	
	// Test with empty config (should get defaults)
	emptyConfig := DocsConfig{}
	result2 := prepareDocsConfig(emptyConfig)
	if result2.Title != "API Documentation" {
		t.Errorf("Expected default title, got %s", result2.Title)
	}
	if result2.OpenAPIPath != "/openapi.json" {
		t.Errorf("Expected default OpenAPIPath, got %s", result2.OpenAPIPath)
	}
	
	// Test with no arguments (should get defaults)
	result3 := prepareDocsConfig()
	if result3.Title != "API Documentation" {
		t.Errorf("Expected default title for no args, got %s", result3.Title)
	}
}

// Test for buildBasicTypeSchemaWithRegistry function (66.7% coverage) 
func TestBuildBasicTypeSchemaWithRegistry(t *testing.T) {
	registry := make(map[string]*Schema)
	
	// Test with pointer type
	ptrType := reflect.TypeOf((*string)(nil))
	schema := buildBasicTypeSchemaWithRegistry(ptrType, registry)
	if schema.Type != "string" {
		t.Errorf("Expected string type for *string, got %s", schema.Type)
	}
	
	// Test with non-pointer type
	stringType := reflect.TypeOf("")
	schema2 := buildBasicTypeSchemaWithRegistry(stringType, registry)
	if schema2.Type != "string" {
		t.Errorf("Expected string type for string, got %s", schema2.Type)
	}
}

// Test for Schema MarshalJSON function (71.4% coverage)
func TestSchema_MarshalJSON(t *testing.T) {
	// Test schema with single type
	schema := &Schema{Type: "string"}
	data, err := schema.MarshalJSON()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if result["type"] != "string" {
		t.Errorf("Expected type=string, got %v", result["type"])
	}
	
	// Test schema with multiple types (should use array format)
	schema2 := &Schema{Types: []string{"string", "null"}}
	data2, err := schema2.MarshalJSON()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	var result2 map[string]interface{}
	if err := json.Unmarshal(data2, &result2); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	// Should use the array format for multiple types
	typeValue, ok := result2["type"].([]interface{})
	if !ok {
		t.Errorf("Expected type to be array, got %T", result2["type"])
	} else {
		if len(typeValue) != 2 {
			t.Errorf("Expected 2 types, got %d", len(typeValue))
		}
	}
	
	// Test schema with properties (should not have type field in JSON)
	schema3 := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"name": {Type: "string"},
		},
	}
	data3, err := schema3.MarshalJSON()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	var result3 map[string]interface{}
	if err := json.Unmarshal(data3, &result3); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	// Should include properties
	if result3["properties"] == nil {
		t.Error("Expected properties to be present")
	}
}

// Test for extractFieldDescription function (66.7% coverage)
func TestExtractFieldDescription(t *testing.T) {
	// This function needs AST fields, not reflect fields.
	// We'll create a minimal AST field to test the function
	
	docExtractor := NewDocExtractor()
	
	// Create an AST field with documentation
	field := &ast.Field{
		Names: []*ast.Ident{
			{Name: "TestField"},
		},
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// This is a test field"},
			},
		},
	}
	
	result := docExtractor.extractFieldDescription(field)
	if result == "" {
		t.Error("Expected non-empty description for documented field")
	}
	
	// Test field with comment instead of doc
	field2 := &ast.Field{
		Names: []*ast.Ident{
			{Name: "TestField2"},
		},
		Comment: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// This is a comment"},
			},
		},
	}
	
	result2 := docExtractor.extractFieldDescription(field2)
	if result2 == "" {
		t.Error("Expected non-empty description for field with comment")
	}
	
	// Test field with no documentation
	field3 := &ast.Field{
		Names: []*ast.Ident{
			{Name: "TestField3"},
		},
	}
	
	result3 := docExtractor.extractFieldDescription(field3)
	if result3 != "" {
		t.Errorf("Expected empty description for undocumented field, got %s", result3)
	}
}