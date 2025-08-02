package api

import (
	"net/http"
	"reflect"
	"testing"
	"unsafe"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Test the 83.3% coverage gap in setSliceFieldValue - the non-string slice type early return
func TestSetSliceFieldValue_NonStringSliceType(t *testing.T) {
	// Create a struct with a non-string slice field
	type TestStruct struct {
		IntSlice []int `json:"int_slice"`
	}

	req := TestStruct{}
	fieldValue := reflect.ValueOf(&req).Elem().Field(0)
	fieldType := reflect.TypeOf(req).Field(0)

	// This should trigger the early return at line 196-197 because it's []int, not []string
	setSliceFieldValue(fieldValue, fieldType, "1,2,3", []string{"1", "2", "3"})

	// The field should remain unchanged (empty) because the function returns early
	if len(req.IntSlice) != 0 {
		t.Error("Expected IntSlice to remain empty for non-string slice type")
	}
}

// Test parseOpenAPIParam edge cases
func TestParseOpenAPIParam_MoreEdgeCases(t *testing.T) {
	// Test field with no openapi tag
	name, in, required := parseOpenAPIParam("")

	if name != "" || in != "" || required {
		t.Error("Expected empty values for missing openapi tag")
	}

	// Test malformed openapi tag
	name, in, required = parseOpenAPIParam("malformed")

	if name != "" || in != "" || required {
		t.Error("Expected empty values for malformed openapi tag")
	}
}

// Test extractParameters with edge cases
func TestExtractParameters_MoreEdgeCases(t *testing.T) {
	// Test with struct that has no valid parameter fields
	type EmptyStruct struct{}

	params := extractParameters(reflect.TypeOf(EmptyStruct{}), nil)
	if len(params) != 0 {
		t.Error("Expected no parameters for empty struct")
	}

	// Test with struct containing only unexported fields
	type UnexportedFieldsStruct struct {
		private string
		hidden  int
	}

	params = extractParameters(reflect.TypeOf(UnexportedFieldsStruct{}), nil)
	if len(params) != 0 {
		t.Error("Expected no parameters for struct with only unexported fields")
	}
}

// Test isUnionType and isUnionStruct functions to reach the handleUnionType path correctly
func TestUnionTypeDetection_MissingCoverage(t *testing.T) {
	// Test isUnionType with non-union types
	type RegularStruct struct {
		Field string
	}

	if isUnionType(reflect.TypeOf(RegularStruct{})) {
		t.Error("Expected false for non-union struct")
	}

	if isUnionType(reflect.TypeOf("")) {
		t.Error("Expected false for primitive type")
	}

	if isUnionType(nil) {
		t.Error("Expected false for nil type")
	}

	// Test isUnionStruct with non-struct types
	if isUnionStruct(reflect.TypeOf("")) {
		t.Error("Expected false for primitive type")
	}

	if isUnionStruct(reflect.TypeOf([]string{})) {
		t.Error("Expected false for slice type")
	}
}

// Test CheckDiscriminatorErrors with more edge cases
func TestCheckDiscriminatorErrors_FinalEdgeCases(t *testing.T) {
	// Test with interface{} value
	var interfaceVal interface{} = "test"
	errors := CheckDiscriminatorErrors(reflect.ValueOf(interfaceVal))
	if len(errors) != 0 {
		t.Error("Expected no errors for interface{} value")
	}

	// Test with nil interface
	var nilInterface interface{}
	errors = CheckDiscriminatorErrors(reflect.ValueOf(&nilInterface).Elem())
	if len(errors) != 0 {
		t.Error("Expected no errors for nil interface")
	}
}

// Test reflectTypeToSchemaInternal with makePointerNullable false
func TestReflectTypeToSchemaInternal_NonNullablePointer(t *testing.T) {
	registry := make(map[string]*Schema)

	type TestStruct struct {
		Field string
	}

	// Test pointer type with makePointerNullable = false
	schema := reflectTypeToSchemaInternal(reflect.TypeOf(&TestStruct{}), registry, false)

	// Schema should not be nullable (no Type array with "null")
	if schema.Type != "" && schema.Type == "null" {
		t.Error("Expected pointer to not be nullable when makePointerNullable is false")
	}
}

// Test handleUnionType with anonymous union type (no name) to cover line 325
func TestHandleUnionType_AnonymousUnion(t *testing.T) {
	registry := make(map[string]*Schema)

	// Create an anonymous struct that looks like a union (2+ pointer fields)
	anonType := reflect.StructOf([]reflect.StructField{
		{Name: "Option1", Type: reflect.TypeOf((*string)(nil)), Tag: `openapi:"discriminator=type"`},
		{Name: "Option2", Type: reflect.TypeOf((*int)(nil))},
	})

	// This should trigger the path where typeName is empty and returns u directly
	schema := handleUnionType(anonType, registry)

	// Schema should be returned directly, not as a reference
	if schema.Ref != "" {
		t.Error("Expected anonymous union to return schema directly, not as reference")
	}

	if schema.OneOf == nil {
		t.Error("Expected anonymous union to have OneOf property")
	}
}

// Test processPackages error handling - this function is not easily testable
// as it requires mocking ast.Package which is complex. Skip for now.

// Test processVariantDiscriminator with conflicting discriminator property names
func TestProcessVariantDiscriminator_ConflictingPropertyNames(t *testing.T) {
	// Create a discriminatorInfo with an existing property name
	discInfo := &discriminatorInfo{
		propertyName: "type",
		mapping:      make(map[string]string),
		isValid:      true,
	}

	// Create a type with a discriminator field that has a different JSON name
	type TestVariant struct {
		Kind string `json:"kind" openapi:"discriminator=test"`
	}

	// This should set isValid to false because "kind" != "type"
	processVariantDiscriminator(reflect.TypeOf(TestVariant{}), discInfo)

	if discInfo.isValid {
		t.Error("Expected isValid to be false when discriminator property names don't match")
	}
}

// Test extractDescription with empty comment to cover line 223
func TestExtractDescription_EmptyAfterTrim(t *testing.T) {
	// Test with only whitespace which results in empty after trim
	desc := extractDescription("   \n\t  ")
	if desc != "" {
		t.Errorf("Expected empty string, got %q", desc)
	}
}

// Test extractDescription with empty string that hits line 223
func TestExtractDescription_EmptyString(t *testing.T) {
	desc := extractDescription("")
	if desc != "" {
		t.Errorf("Expected empty string, got %q", desc)
	}
}

// Test applySecurityToOperation with unknown security type (default case)
func TestApplySecurityToOperation_UnknownType(t *testing.T) {
	route := &RouteInfo{
		Options: &HandlerOption{
			Security: []SecurityRequirement{
				{Type: "unknown"},
				{Type: "basic"}, // Include a valid one to ensure processing continues
			},
		},
	}

	spec := &OpenAPISpec{
		Components: &Components{},
	}

	op := &Operation{}

	applySecurityToOperation(route, spec, op)

	// Should only have added BasicAuth, not the unknown type
	if len(spec.Components.SecuritySchemes) != 1 {
		t.Errorf("Expected 1 security scheme, got %d", len(spec.Components.SecuritySchemes))
	}

	if _, ok := spec.Components.SecuritySchemes["BasicAuth"]; !ok {
		t.Error("Expected BasicAuth to be added")
	}
}

// Test setSliceFieldValue with empty parts and empty paramValue
func TestSetSliceFieldValue_EmptyPartsAndParam(t *testing.T) {
	type TestStruct struct {
		Tags []string `json:"tags"`
	}

	req := TestStruct{}
	fieldValue := reflect.ValueOf(&req).Elem().Field(0)
	fieldType := reflect.TypeOf(req).Field(0)

	// Call with empty paramValue and empty allValues - should not modify the field
	setSliceFieldValue(fieldValue, fieldType, "", []string{})

	if req.Tags != nil {
		t.Error("Expected Tags to remain nil")
	}
}

// Test setSliceFieldValue with single value containing commas (line 203-204)
func TestSetSliceFieldValue_SingleValueWithCommas(t *testing.T) {
	type TestStruct struct {
		Tags []string `json:"tags"`
	}

	req := TestStruct{}
	fieldValue := reflect.ValueOf(&req).Elem().Field(0)
	fieldType := reflect.TypeOf(req).Field(0)

	// Call with single value containing commas - should split on commas
	setSliceFieldValue(fieldValue, fieldType, "", []string{"tag1,tag2,tag3"})

	if len(req.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(req.Tags))
	}

	if req.Tags[0] != "tag1" || req.Tags[1] != "tag2" || req.Tags[2] != "tag3" {
		t.Errorf("Expected [tag1, tag2, tag3], got %v", req.Tags)
	}
}

// Test parseOpenAPIParam with edge case where tag part equals "in="
func TestParseOpenAPIParam_InEqualsEdgeCase(t *testing.T) {
	// This case should return false because loc is empty
	name, in, ok := parseOpenAPIParam("name,in=,required")

	if ok {
		t.Errorf("Expected ok to be false for empty 'in=' value, got true with (%s, %s)", name, in)
	}
}

// Test extractParameters with pointer type
func TestExtractParameters_PointerType(t *testing.T) {
	type TestStruct struct {
		ID string `json:"id" openapi:"id,in=path"`
	}

	// Test with pointer to struct
	params := extractParameters(reflect.TypeOf(&TestStruct{}), nil)

	if len(params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(params))
	}

	if params[0].Name != "id" {
		t.Errorf("Expected parameter name 'id', got '%s'", params[0].Name)
	}
}

// Test CheckDiscriminatorErrors edge case line 610
func TestCheckDiscriminatorErrors_NoDiscriminatorField(t *testing.T) {
	// Test struct without discriminator openapi tag
	type NoDiscriminator struct {
		Type string `json:"type"` // No openapi tag
	}

	val := reflect.ValueOf(NoDiscriminator{Type: "test"})
	errors := CheckDiscriminatorErrors(val)

	// Should return no errors when no discriminator field exists
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %v", errors)
	}
}

// Test validator TagNameFunc with json:"-" tag
func TestValidator_JsonDashTag(t *testing.T) {
	type TestStruct struct {
		Ignored string `json:"-" validate:"required"`
		Normal  string `json:"normal" validate:"required"`
	}

	val := TestStruct{}
	err := validate.Struct(val)

	// The validation should fail for "normal" field only
	if err == nil {
		t.Error("Expected validation error")
	}

	// Check that the error is for "normal" field, not the ignored one
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			if e.Field() == "" {
				// This would be the ignored field with json:"-"
				t.Error("Field with json:\"-\" should not appear in validation errors")
			}
		}
	}
}

// Test reflectTypeToSchemaInternal with union types
func TestReflectTypeToSchemaInternal_UnionType(t *testing.T) {
	registry := make(map[string]*Schema)

	// Create a union-like struct (2+ pointer fields)
	type UnionStruct struct {
		Option1 *string `openapi:"discriminator=option1"`
		Option2 *int    `openapi:"discriminator=option2"`
	}

	// This should trigger isUnionStruct and call handleUnionType
	schema := reflectTypeToSchemaInternal(reflect.TypeOf(UnionStruct{}), registry, false)

	if schema == nil {
		t.Error("Expected schema for union type")
	}
}

// Test applyValidationConstraints with nil schema case
func TestApplyValidationConstraints_NilSchema(t *testing.T) {
	// This should return early without panic
	applyValidationConstraints(nil, "required", reflect.TypeOf(""), nil, reflect.StructField{})
}

// Test UnmarshalJSON error path (line 139)
func TestSchema_UnmarshalJSON_ErrorPath(t *testing.T) {
	var schema Schema
	err := schema.UnmarshalJSON([]byte("invalid json"))

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Test UnmarshalYAML error path (line 190)
func TestSchema_UnmarshalYAML_ErrorPath(t *testing.T) {
	var schema Schema
	err := yaml.Unmarshal([]byte("invalid: [yaml"), &schema)

	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

// Test extractParameters with non-struct type after unwrapping pointer
func TestExtractParameters_NonStructAfterPointer(t *testing.T) {
	// Test with pointer to non-struct (e.g., pointer to string)
	var str string
	params := extractParameters(reflect.TypeOf(&str), nil)

	if len(params) != 0 {
		t.Error("Expected no parameters for pointer to non-struct")
	}
}

// Test buildBasicTypeSchema with UnsafePointer type
func TestBuildBasicTypeSchema_UnsafePointer(t *testing.T) {
	var p unsafe.Pointer
	schema := buildBasicTypeSchema(reflect.TypeOf(p))

	if schema.Type != "object" {
		t.Errorf("Expected object type for UnsafePointer, got %s", schema.Type)
	}

	if schema.Description != "Unsafe pointer" {
		t.Errorf("Expected 'Unsafe pointer' description, got %s", schema.Description)
	}
}

// Test buildBasicTypeSchema with Complex128 type
func TestBuildBasicTypeSchema_Complex128(t *testing.T) {
	var c complex128
	schema := buildBasicTypeSchema(reflect.TypeOf(c))

	if schema.Type != "object" {
		t.Errorf("Expected object type for Complex128, got %s", schema.Type)
	}

	if schema.Description != "Complex number" {
		t.Errorf("Expected 'Complex number' description, got %s", schema.Description)
	}
}

// Test CheckDiscriminatorErrors with json:"-" tag
func TestCheckDiscriminatorErrors_JsonDashTag(t *testing.T) {
	type TestStruct struct {
		Type  string `json:"-" openapi:"discriminator=expected"`
		Other string
	}

	// Set Type to a different value than expected
	val := TestStruct{Type: "wrong"}
	errors := CheckDiscriminatorErrors(val)

	// Should have error for field "Type" (uses field name when json:"-")
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	if _, ok := errors["Type"]; !ok {
		t.Error("Expected error for field 'Type'")
	}

	// Verify it's a discriminator error
	if errs, ok := errors["Type"]; ok && len(errs) > 0 {
		if errs[0] != "discriminator" {
			t.Errorf("Expected 'discriminator' error, got %s", errs[0])
		}
	}
}

// Test DocsRoute with nil registerFn
func TestDocsRoute_NilRegisterFn(t *testing.T) {
	// Create a router with nil registerFn
	router := &TypedRouter[struct{}]{
		registry:   NewRouteRegistry(),
		registerFn: nil, // This will cause the if check to fail
	}

	// Call DocsRoute - should not panic
	router.DocsRoute("/docs")
}

// Test DocsRoute with relative OpenAPIPath (no leading slash)
func TestDocsRoute_RelativeOpenAPIPath(t *testing.T) {
	registry := NewRouteRegistry()
	router := &TypedRouter[struct{}]{
		registry: registry,
		registerFn: func(method, path string, handler http.HandlerFunc, info *RouteInfo) {
			// Just track that it was called
		},
	}

	// Call DocsRoute with config that has relative OpenAPIPath
	router.DocsRoute("/docs", DocsConfig{
		Title:       "Test API",
		OpenAPIPath: "spec.json", // No leading slash - relative path
		UITemplate:  StoplightUITemplate,
	})
}

// Test buildBasicTypeSchema with Chan type
func TestBuildBasicTypeSchema_Chan(t *testing.T) {
	ch := make(chan int)
	schema := buildBasicTypeSchema(reflect.TypeOf(ch))

	if schema.Type != "object" {
		t.Errorf("Expected object type for Chan, got %s", schema.Type)
	}

	if schema.Description != "Channel" {
		t.Errorf("Expected 'Channel' description, got %s", schema.Description)
	}
}

// Test buildBasicTypeSchema with Func type
func TestBuildBasicTypeSchema_Func(t *testing.T) {
	var fn func()
	schema := buildBasicTypeSchema(reflect.TypeOf(fn))

	if schema.Type != "object" {
		t.Errorf("Expected object type for Func, got %s", schema.Type)
	}

	if schema.Description != "Function" {
		t.Errorf("Expected 'Function' description, got %s", schema.Description)
	}
}

// Test buildBasicTypeSchema with Interface type
func TestBuildBasicTypeSchema_Interface(t *testing.T) {
	var i interface{}
	schema := buildBasicTypeSchema(reflect.TypeOf(&i).Elem())

	if schema.Type != "object" {
		t.Errorf("Expected object type for Interface, got %s", schema.Type)
	}

	if schema.Description != "Interface" {
		t.Errorf("Expected 'Interface' description, got %s", schema.Description)
	}
}

// Test buildBasicTypeSchema with Map type
func TestBuildBasicTypeSchema_Map(t *testing.T) {
	var m map[string]int
	schema := buildBasicTypeSchema(reflect.TypeOf(m))

	if schema.Type != "object" {
		t.Errorf("Expected object type for Map, got %s", schema.Type)
	}

	if schema.Description != "Map with dynamic keys" {
		t.Errorf("Expected 'Map with dynamic keys' description, got %s", schema.Description)
	}
}

// Test buildBasicTypeSchema with Uintptr type
func TestBuildBasicTypeSchema_Uintptr(t *testing.T) {
	var p uintptr
	schema := buildBasicTypeSchema(reflect.TypeOf(p))

	if schema.Type != "integer" {
		t.Errorf("Expected integer type for Uintptr, got %s", schema.Type)
	}

	if schema.Description != "Pointer-sized integer" {
		t.Errorf("Expected 'Pointer-sized integer' description, got %s", schema.Description)
	}
}

// Test buildBasicTypeSchema with Complex64 type
func TestBuildBasicTypeSchema_Complex64(t *testing.T) {
	var c complex64
	schema := buildBasicTypeSchema(reflect.TypeOf(c))

	if schema.Type != "object" {
		t.Errorf("Expected object type for Complex64, got %s", schema.Type)
	}

	if schema.Description != "Complex number" {
		t.Errorf("Expected 'Complex number' description, got %s", schema.Description)
	}
}

// Test buildBasicTypeSchema with Invalid type would require a type with Kind() == Invalid
// which is not easily achievable in normal Go code. The Invalid case in the switch
// statement is there for completeness but is essentially unreachable in practice.

// Test openAPISpec unmarshal with type array handling
func TestSchema_UnmarshalYAML_TypeArray(t *testing.T) {
	yamlContent := `
type:
  - string
  - "null"
description: test
`
	var schema Schema
	err := yaml.Unmarshal([]byte(yamlContent), &schema)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(schema.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(schema.Types))
	}

	if schema.Type != "" {
		t.Error("Expected Type to be empty when Types array is used")
	}
}
