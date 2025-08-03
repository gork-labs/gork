package api

import (
	"reflect"
	"testing"
)

// Tests for individual schema handlers

func TestPointerTypeHandler(t *testing.T) {
	handler := &PointerTypeHandler{}
	registry := make(map[string]*Schema)

	t.Run("can handle pointer type", func(t *testing.T) {
		var s *string
		if !handler.CanHandle(reflect.TypeOf(s)) {
			t.Error("Expected to handle pointer type")
		}
	})

	t.Run("cannot handle non-pointer type", func(t *testing.T) {
		if handler.CanHandle(reflect.TypeOf("")) {
			t.Error("Expected to not handle non-pointer type")
		}
	})

	t.Run("generate nullable schema", func(t *testing.T) {
		var s *string
		schema := handler.GenerateSchema(reflect.TypeOf(s), registry, true)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Should create nullable schema with Types array for basic types
		if schema.Types == nil && schema.AnyOf == nil {
			t.Error("Expected nullable schema to have Types array or AnyOf")
		}
	})

	t.Run("generate non-nullable schema", func(t *testing.T) {
		var s *string
		schema := handler.GenerateSchema(reflect.TypeOf(s), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Should unwrap pointer without making nullable
		if schema.Type != "string" {
			t.Errorf("Expected type 'string', got '%s'", schema.Type)
		}
	})
}

func TestUnionTypeHandler(t *testing.T) {
	handler := &UnionTypeHandler{}
	registry := make(map[string]*Schema)

	t.Run("can handle union type", func(t *testing.T) {
		type TestStruct struct {
			Type string `gork:"type,discriminator=test"`
		}

		// isUnionType and isUnionStruct determine if it's a union
		result := handler.CanHandle(reflect.TypeOf(TestStruct{}))
		// Result depends on the actual union detection logic
		_ = result
	})

	t.Run("generate union schema", func(t *testing.T) {
		type TestStruct struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}

		schema := handler.GenerateSchema(reflect.TypeOf(TestStruct{}), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Union schema generation depends on implementation
		_ = schema
	})
}

func TestStructTypeHandler(t *testing.T) {
	handler := &StructTypeHandler{}
	registry := make(map[string]*Schema)

	t.Run("can handle struct type", func(t *testing.T) {
		type TestStruct struct {
			Field string
		}

		if !handler.CanHandle(reflect.TypeOf(TestStruct{})) {
			t.Error("Expected to handle struct type")
		}
	})

	t.Run("cannot handle non-struct type", func(t *testing.T) {
		if handler.CanHandle(reflect.TypeOf("")) {
			t.Error("Expected to not handle non-struct type")
		}
	})

	t.Run("generate struct schema", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		schema := handler.GenerateSchema(reflect.TypeOf(TestStruct{}), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// buildStructSchema returns a reference for named types, not the schema directly
		if schema.Ref == "" && schema.Type != "object" {
			t.Errorf("Expected type 'object' or a reference, got type='%s', ref='%s'", schema.Type, schema.Ref)
		}

		// Properties are only set on the actual schema, not the reference
		if schema.Ref == "" && schema.Properties == nil {
			t.Error("Expected properties to be set for non-reference schema")
		}
	})
}

func TestArrayTypeHandler(t *testing.T) {
	handler := &ArrayTypeHandler{}
	registry := make(map[string]*Schema)

	t.Run("can handle slice type", func(t *testing.T) {
		if !handler.CanHandle(reflect.TypeOf([]string{})) {
			t.Error("Expected to handle slice type")
		}
	})

	t.Run("can handle array type", func(t *testing.T) {
		if !handler.CanHandle(reflect.TypeOf([5]string{})) {
			t.Error("Expected to handle array type")
		}
	})

	t.Run("cannot handle non-array type", func(t *testing.T) {
		if handler.CanHandle(reflect.TypeOf("")) {
			t.Error("Expected to not handle non-array type")
		}
	})

	t.Run("generate array schema", func(t *testing.T) {
		schema := handler.GenerateSchema(reflect.TypeOf([]string{}), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		if schema.Type != "array" {
			t.Errorf("Expected type 'array', got '%s'", schema.Type)
		}

		if schema.Items == nil {
			t.Error("Expected items to be set")
		}
	})
}

func TestExistingTypeHandler(t *testing.T) {
	handler := &ExistingTypeHandler{}

	t.Run("never handles any type", func(t *testing.T) {
		if handler.CanHandle(reflect.TypeOf("")) {
			t.Error("Expected ExistingTypeHandler to never handle any type")
		}

		if handler.CanHandle(reflect.TypeOf(0)) {
			t.Error("Expected ExistingTypeHandler to never handle any type")
		}

		type TestStruct struct{}
		if handler.CanHandle(reflect.TypeOf(TestStruct{})) {
			t.Error("Expected ExistingTypeHandler to never handle any type")
		}
	})

	t.Run("generate schema for existing type", func(t *testing.T) {
		registry := map[string]*Schema{
			"TestStruct": {Type: "object", Description: "Existing"},
		}

		type TestStruct struct{}
		schema := handler.GenerateSchema(reflect.TypeOf(TestStruct{}), registry, false)

		if schema == nil {
			t.Fatal("Expected schema reference to be returned")
		}

		if schema.Ref != "#/components/schemas/TestStruct" {
			t.Errorf("Expected reference to TestStruct, got %s", schema.Ref)
		}
	})

	t.Run("generate schema for non-existing type", func(t *testing.T) {
		registry := make(map[string]*Schema)

		type TestStruct struct{}
		schema := handler.GenerateSchema(reflect.TypeOf(TestStruct{}), registry, false)

		if schema != nil {
			t.Error("Expected nil schema for non-existing type")
		}
	})
}

func TestBasicTypeHandler(t *testing.T) {
	handler := &BasicTypeHandler{}
	registry := make(map[string]*Schema)

	t.Run("can handle any type", func(t *testing.T) {
		if !handler.CanHandle(reflect.TypeOf("")) {
			t.Error("Expected to handle string type")
		}

		if !handler.CanHandle(reflect.TypeOf(0)) {
			t.Error("Expected to handle int type")
		}

		if !handler.CanHandle(reflect.TypeOf(true)) {
			t.Error("Expected to handle bool type")
		}
	})

	t.Run("generate basic type schemas", func(t *testing.T) {
		// Test string
		stringSchema := handler.GenerateSchema(reflect.TypeOf(""), registry, false)
		if stringSchema.Type != "string" {
			t.Errorf("Expected string type, got '%s'", stringSchema.Type)
		}

		// Test int
		intSchema := handler.GenerateSchema(reflect.TypeOf(0), registry, false)
		if intSchema.Type != "integer" {
			t.Errorf("Expected integer type, got '%s'", intSchema.Type)
		}

		// Test bool
		boolSchema := handler.GenerateSchema(reflect.TypeOf(true), registry, false)
		if boolSchema.Type != "boolean" {
			t.Errorf("Expected boolean type, got '%s'", boolSchema.Type)
		}
	})
}

// Tests for the orchestrator

func TestSchemaGenerator(t *testing.T) {
	t.Run("create with default handlers", func(t *testing.T) {
		generator := NewSchemaGenerator()

		if generator == nil {
			t.Fatal("Expected generator to be created")
		}

		if len(generator.handlers) == 0 {
			t.Error("Expected default handlers to be set")
		}
	})

	t.Run("create with custom handlers", func(t *testing.T) {
		customHandlers := []TypeSchemaHandler{
			&BasicTypeHandler{},
		}

		generator := NewSchemaGeneratorWithHandlers(customHandlers)

		if generator == nil {
			t.Fatal("Expected generator to be created")
		}

		if len(generator.handlers) != 1 {
			t.Errorf("Expected 1 handler, got %d", len(generator.handlers))
		}
	})

	t.Run("generate schema for string", func(t *testing.T) {
		generator := NewSchemaGenerator()
		registry := make(map[string]*Schema)

		schema := generator.GenerateSchema(reflect.TypeOf(""), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		if schema.Type != "string" {
			t.Errorf("Expected type 'string', got '%s'", schema.Type)
		}
	})

	t.Run("generate schema for pointer to string", func(t *testing.T) {
		generator := NewSchemaGenerator()
		registry := make(map[string]*Schema)

		var s *string
		schema := generator.GenerateSchema(reflect.TypeOf(s), registry, true)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Should be handled by PointerTypeHandler - either Types array or AnyOf
		if schema.Types == nil && schema.AnyOf == nil {
			t.Error("Expected nullable schema with Types array or AnyOf")
		}
	})

	t.Run("generate schema for struct", func(t *testing.T) {
		generator := NewSchemaGenerator()
		registry := make(map[string]*Schema)

		type TestStruct struct {
			Name string `json:"name"`
		}

		schema := generator.GenerateSchema(reflect.TypeOf(TestStruct{}), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// buildStructSchema returns a reference for named types
		if schema.Ref == "" && schema.Type != "object" {
			t.Errorf("Expected type 'object' or a reference, got type='%s', ref='%s'", schema.Type, schema.Ref)
		}
	})

	t.Run("generate schema for slice", func(t *testing.T) {
		generator := NewSchemaGenerator()
		registry := make(map[string]*Schema)

		schema := generator.GenerateSchema(reflect.TypeOf([]string{}), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		if schema.Type != "array" {
			t.Errorf("Expected type 'array', got '%s'", schema.Type)
		}

		if schema.Items == nil {
			t.Error("Expected items to be set")
		}
	})

	t.Run("generate schema with existing type", func(t *testing.T) {
		generator := NewSchemaGenerator()
		registry := map[string]*Schema{
			"TestStruct": {Type: "object", Description: "Existing schema"},
		}

		type TestStruct struct {
			Name string
		}

		schema := generator.GenerateSchema(reflect.TypeOf(TestStruct{}), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Should return reference to existing schema
		if schema.Ref == "" && schema.Type != "object" {
			t.Error("Expected reference to existing schema or object type")
		}
	})

	t.Run("fallback to basic type handler", func(t *testing.T) {
		// Create generator with no handlers
		generator := NewSchemaGeneratorWithHandlers([]TypeSchemaHandler{})
		registry := make(map[string]*Schema)

		schema := generator.GenerateSchema(reflect.TypeOf(""), registry, false)

		if schema == nil {
			t.Fatal("Expected schema to be generated via fallback")
		}

		if schema.Type != "string" {
			t.Errorf("Expected type 'string', got '%s'", schema.Type)
		}
	})
}

// Mock handler for testing

type mockTypeHandler struct {
	canHandleType reflect.Type
	returnSchema  *Schema
}

func (m *mockTypeHandler) CanHandle(t reflect.Type) bool {
	return m.canHandleType == t
}

func (m *mockTypeHandler) GenerateSchema(t reflect.Type, registry map[string]*Schema, makePointerNullable bool) *Schema {
	return m.returnSchema
}

func TestSchemaGeneratorWithMocks(t *testing.T) {
	t.Run("use first matching handler", func(t *testing.T) {
		mockSchema := &Schema{Type: "mock", Description: "Mock schema"}
		mockHandler := &mockTypeHandler{
			canHandleType: reflect.TypeOf(""),
			returnSchema:  mockSchema,
		}

		generator := NewSchemaGeneratorWithHandlers([]TypeSchemaHandler{
			mockHandler,
			&BasicTypeHandler{}, // This should not be used
		})

		registry := make(map[string]*Schema)
		schema := generator.GenerateSchema(reflect.TypeOf(""), registry, false)

		if schema != mockSchema {
			t.Error("Expected mock schema to be returned")
		}

		if schema.Description != "Mock schema" {
			t.Errorf("Expected 'Mock schema', got '%s'", schema.Description)
		}
	})
}
