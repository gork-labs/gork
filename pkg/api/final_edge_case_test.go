package api

import (
	"reflect"
	"testing"
	"unsafe"

	"gopkg.in/yaml.v3"
)

// These tests target the final remaining uncovered lines for 100% coverage

// Test reflectTypeToSchemaInternal edge case with pointers (openapi_generator.go:241)
func TestReflectTypeToSchemaInternal_ComplexPointerCases(t *testing.T) {
	registry := make(map[string]*Schema)

	// Test nested pointer unwrapping
	nestedPtrType := reflect.TypeOf((**string)(nil)) // Pointer to pointer
	schema := reflectTypeToSchemaInternal(nestedPtrType, registry, false)
	if schema == nil {
		t.Error("Should handle nested pointers")
	}

	// Test with makePointerNullable=true for deep nesting
	schema2 := reflectTypeToSchemaInternal(nestedPtrType, registry, true)
	if schema2 == nil {
		t.Error("Should handle nested pointers with nullable option")
	}

	// Test with makePointerNullable=true on simple pointer (line 240-241 coverage)
	simplePtr := reflect.TypeOf((*string)(nil))
	schema3 := reflectTypeToSchemaInternal(simplePtr, registry, true)
	if schema3 == nil {
		t.Error("Should handle simple pointer with nullable option")
	}

	// The schema should be nullable (either AnyOf or Types array)
	isNullable := schema3.AnyOf != nil || len(schema3.Types) > 1
	if !isNullable {
		t.Error("Pointer with makePointerNullable=true should be nullable")
	}
}

// Test buildStructSchema with named types (openapi_generator.go:344)
func TestBuildStructSchema_NamedTypes(t *testing.T) {
	registry := make(map[string]*Schema)

	// Create a named struct type
	type NamedStruct struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	structType := reflect.TypeOf(NamedStruct{})
	schema := buildStructSchema(structType, registry)

	// Should create a reference for named types
	if schema.Ref == "" {
		t.Error("Named struct should have reference")
	}

	// Verify it was registered
	if _, exists := registry["NamedStruct"]; !exists {
		t.Error("Named struct should be registered in registry")
	}
}

// Test buildStructSchema with unexported fields (openapi_generator.go:347)
func TestBuildStructSchema_UnexportedFields(t *testing.T) {
	registry := make(map[string]*Schema)

	// Create a struct type with both exported and unexported fields
	structType := reflect.StructOf([]reflect.StructField{
		{
			Name: "ExportedField",
			Type: reflect.TypeOf(""),
			Tag:  reflect.StructTag(`json:"exported"`),
		},
		{
			Name:    "unexportedField", // lowercase = unexported
			PkgPath: "some/package",    // This makes it unexported
			Type:    reflect.TypeOf(""),
			Tag:     reflect.StructTag(`json:"unexported"`),
		},
	})

	schema := buildStructSchema(structType, registry)

	// Should only process exported field
	if len(schema.Properties) != 1 {
		t.Errorf("Expected 1 property (unexported should be skipped), got %d", len(schema.Properties))
	}

	if _, exists := schema.Properties["exported"]; !exists {
		t.Error("Expected exported field to be processed")
	}

	if _, exists := schema.Properties["unexported"]; exists {
		t.Error("Unexported field should not be processed")
	}
}

// Test buildBasicTypeSchema with edge cases (openapi_generator.go:432)
func TestBuildBasicTypeSchema_EdgeCases(t *testing.T) {
	// Test with slice kind (should fall through to default)
	sliceType := reflect.TypeOf([]string{})
	schema := buildBasicTypeSchema(sliceType)
	if schema.Type != "object" {
		t.Errorf("Slice type should default to object, got %s", schema.Type)
	}

	// Test with struct kind (should fall through to default)
	structType := reflect.TypeOf(struct{}{})
	schema2 := buildBasicTypeSchema(structType)
	if schema2.Type != "object" {
		t.Errorf("Struct type should default to object, got %s", schema2.Type)
	}

	// Test with ptr kind (should fall through to default)
	ptrType := reflect.TypeOf((*string)(nil))
	schema3 := buildBasicTypeSchema(ptrType)
	if schema3.Type != "object" {
		t.Errorf("Ptr type should default to object, got %s", schema3.Type)
	}

	// Test with Invalid kind (line 438 coverage)
	// reflect.Invalid is the zero value of reflect.Kind and should be impossible
	// to reach in normal usage since buildBasicTypeSchema is only called with valid types
	// However, we can test this by using reflection on a struct field with nil value
	type TestStruct struct {
		Field interface{}
	}

	structVal := reflect.ValueOf(TestStruct{})
	fieldVal := structVal.Field(0)
	if !fieldVal.IsValid() || fieldVal.Kind() == reflect.Invalid {
		t.Skip("Cannot create a valid Invalid type for testing")
	}

	// Alternative: test by checking if the case exists in the function
	// Since Invalid is actually handled by the switch case, it should be reachable
	// The issue is that reflect.TypeOf(nil) returns nil, not a Type with Invalid kind
	t.Log("Invalid case exists in switch but may be unreachable in practice")

	// Test default case by creating an unusual type
	// Use reflect.UnsafePointer to trigger the default case (line 457)
	var ptr unsafe.Pointer
	unsafePtrType := reflect.TypeOf(ptr)
	schema5 := buildBasicTypeSchema(unsafePtrType)
	if schema5.Type != "object" {
		t.Errorf("UnsafePointer type should default to object, got %s", schema5.Type)
	}
	if schema5.Description != "Unsafe pointer" {
		t.Errorf("UnsafePointer should have description 'Unsafe pointer', got %q", schema5.Description)
	}
}

// Test generateUnionSchema with discriminator edge cases (openapi_generator.go:475)
func TestGenerateUnionSchema_DiscriminatorEdgeCases(t *testing.T) {
	registry := make(map[string]*Schema)

	// Create a union type with valid discriminator
	unionType := reflect.StructOf([]reflect.StructField{
		{
			Name: "Option1",
			Type: reflect.TypeOf((*struct {
				Type string `openapi:"discriminator=option1" json:"type"`
				Data string `json:"data"`
			})(nil)),
		},
	})

	schema := generateUnionSchema(unionType, registry)

	// Should have oneOf variants
	if schema.OneOf == nil {
		t.Error("Should have oneOf variants")
	}

	// With only one variant, discriminator might not be valid
	if schema.Discriminator != nil && len(schema.OneOf) == 1 {
		t.Log("Single variant union has discriminator info")
	}
}

// Test isUnionStruct with edge case (openapi_generator.go:569)
func TestIsUnionStruct_EdgeCase(t *testing.T) {
	// Test with a struct that has exactly 2 pointer fields (minimum for union)
	structType := reflect.StructOf([]reflect.StructField{
		{
			Name: "Option1",
			Type: reflect.TypeOf((*string)(nil)),
		},
		{
			Name: "Option2",
			Type: reflect.TypeOf((*int)(nil)),
		},
	})

	result := isUnionStruct(structType)
	if !result {
		t.Error("Should consider struct with exactly 2 pointer fields as union")
	}

	// Test with struct that has unexported field (line 573-574 coverage)
	structWithUnexported := reflect.StructOf([]reflect.StructField{
		{
			Name: "Option1",
			Type: reflect.TypeOf((*string)(nil)),
		},
		{
			Name:    "unexported", // This will have PkgPath set, making it unexported
			PkgPath: "some/package",
			Type:    reflect.TypeOf((*int)(nil)),
		},
	})

	result2 := isUnionStruct(structWithUnexported)
	if result2 {
		t.Error("Should NOT consider struct with unexported fields as union")
	}
}

// Test extractParameters with various field types (openapi_generator.go:730)
func TestExtractParameters_VariousFieldTypes(t *testing.T) {
	registry := make(map[string]*Schema)

	// Test with unexported field (should be skipped)
	type RequestWithUnexported struct {
		PublicParam  string `openapi:"public,in=query"`
		privateParam string `openapi:"private,in=query"` // Unexported, should be skipped
	}

	structType := reflect.TypeOf(RequestWithUnexported{})
	params := extractParameters(structType, registry)

	// Should only extract the public parameter
	if len(params) != 1 {
		t.Errorf("Expected 1 parameter (unexported should be skipped), got %d", len(params))
	}

	if len(params) > 0 && params[0].Name != "public" {
		t.Errorf("Expected parameter name 'public', got '%s'", params[0].Name)
	}
}

// Test UnmarshalYAML with complex types (openapi_types.go:179)
func TestSchema_UnmarshalYAML_ComplexTypes(t *testing.T) {
	var schema Schema

	// Test with complex YAML structure
	yamlData := `
type: object
properties:
  field1:
    type: string
  field2:
    type: 
      - string
      - "null"
oneOf:
  - type: string
  - type: integer
`

	err := yaml.Unmarshal([]byte(yamlData), &schema)
	if err != nil {
		t.Errorf("Should handle complex YAML: %v", err)
	}

	// Verify it parsed correctly
	if schema.Type != "object" {
		t.Errorf("Expected object type, got %s", schema.Type)
	}
}

// Test validator init edge case (validator.go:13)
func TestValidator_InitEdgeCase(t *testing.T) {
	// Test that discriminator validation works with complex patterns
	field := reflect.StructField{
		Name: "ComplexField",
		Tag:  reflect.StructTag(`openapi:"discriminator=complex_value_123"`),
	}

	err := CheckDiscriminatorErrors([]reflect.StructField{field})
	if err != nil {
		t.Errorf("Should handle complex discriminator values: %v", err)
	}
}

// Test CheckDiscriminatorErrors with additional edge cases (validator.go:43)
func TestCheckDiscriminatorErrors_AdditionalEdgeCases(t *testing.T) {
	// Test with empty fields slice
	err := CheckDiscriminatorErrors([]reflect.StructField{})
	if err != nil {
		t.Errorf("Should handle empty fields slice: %v", err)
	}

	// Test with field that has no openapi tag
	field := reflect.StructField{
		Name: "NoTag",
		Tag:  reflect.StructTag(`json:"no_tag"`),
	}

	err2 := CheckDiscriminatorErrors([]reflect.StructField{field})
	if err2 != nil {
		t.Errorf("Should handle fields without openapi tags: %v", err2)
	}

	// Test with malformed discriminator tag
	field2 := reflect.StructField{
		Name: "MalformedTag",
		Tag:  reflect.StructTag(`openapi:"discriminator="`), // Empty value
	}

	err3 := CheckDiscriminatorErrors([]reflect.StructField{field2})
	if err3 != nil {
		t.Errorf("Should handle malformed discriminator tags: %v", err3)
	}
}
