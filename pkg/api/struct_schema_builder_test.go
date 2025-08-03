package api

import (
	"errors"
	"reflect"
	"testing"
)

// Mock implementations for testing

type mockFieldProcessor struct {
	shouldError     bool
	errorMsg        string
	processedFields []string
}

func (m *mockFieldProcessor) ProcessField(field reflect.StructField, schema *Schema, registry map[string]*Schema) error {
	m.processedFields = append(m.processedFields, field.Name)
	if m.shouldError {
		return errors.New(m.errorMsg)
	}
	// Add a test property
	schema.Properties[field.Name] = &Schema{Type: "string"}
	return nil
}

type mockEmbeddedStructProcessor struct {
	shouldError       bool
	errorMsg          string
	processedEmbedded []string
}

func (m *mockEmbeddedStructProcessor) ProcessEmbedded(field reflect.StructField, schema *Schema, registry map[string]*Schema) error {
	m.processedEmbedded = append(m.processedEmbedded, field.Name)
	if m.shouldError {
		return errors.New(m.errorMsg)
	}
	// Add embedded properties
	schema.Properties["embedded_field"] = &Schema{Type: "string"}
	return nil
}

type mockTypeRegistrar struct {
	shouldReturnRef bool
	registeredTypes []string
}

func (m *mockTypeRegistrar) RegisterType(t reflect.Type, schema *Schema, registry map[string]*Schema) *Schema {
	typeName := t.Name()
	m.registeredTypes = append(m.registeredTypes, typeName)

	if m.shouldReturnRef && typeName != "" {
		schema.Title = typeName
		registry[typeName] = schema
		return &Schema{Ref: "#/components/schemas/" + typeName}
	}
	return schema
}

func TestStructSchemaBuilder(t *testing.T) {
	t.Run("build schema with default processors", func(t *testing.T) {
		builder := NewStructSchemaBuilder()
		registry := make(map[string]*Schema)

		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		schema := builder.BuildSchema(reflect.TypeOf(TestStruct{}), registry)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Should return reference for named types
		if schema.Ref == "" {
			t.Error("Expected reference to be returned for named type")
		}

		// Should register the type in registry
		if _, exists := registry["TestStruct"]; !exists {
			t.Error("Expected type to be registered in registry")
		}
	})

	t.Run("build schema with custom processors", func(t *testing.T) {
		fieldProcessor := &mockFieldProcessor{}
		embeddedProcessor := &mockEmbeddedStructProcessor{}
		typeRegistrar := &mockTypeRegistrar{shouldReturnRef: false}

		builder := NewStructSchemaBuilderWithProcessors(
			fieldProcessor,
			embeddedProcessor,
			typeRegistrar,
		)

		registry := make(map[string]*Schema)

		type TestStruct struct {
			Name string
			Age  int
		}

		schema := builder.BuildSchema(reflect.TypeOf(TestStruct{}), registry)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		if schema.Type != "object" {
			t.Errorf("Expected type 'object', got '%s'", schema.Type)
		}

		// Check that field processor was called
		if len(fieldProcessor.processedFields) != 2 {
			t.Errorf("Expected 2 fields processed, got %d", len(fieldProcessor.processedFields))
		}

		// Check that type registrar was called
		if len(typeRegistrar.registeredTypes) != 1 {
			t.Errorf("Expected 1 type registered, got %d", len(typeRegistrar.registeredTypes))
		}
	})

	t.Run("build schema with embedded structs", func(t *testing.T) {
		fieldProcessor := &mockFieldProcessor{}
		embeddedProcessor := &mockEmbeddedStructProcessor{}
		typeRegistrar := &mockTypeRegistrar{shouldReturnRef: false}

		builder := NewStructSchemaBuilderWithProcessors(
			fieldProcessor,
			embeddedProcessor,
			typeRegistrar,
		)

		registry := make(map[string]*Schema)

		type Embedded struct {
			EmbeddedField string
		}

		type TestStruct struct {
			Embedded // Anonymous embedded struct
			Name     string
		}

		schema := builder.BuildSchema(reflect.TypeOf(TestStruct{}), registry)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Check that embedded processor was called
		if len(embeddedProcessor.processedEmbedded) != 1 {
			t.Errorf("Expected 1 embedded struct processed, got %d", len(embeddedProcessor.processedEmbedded))
		}

		// Check that field processor was called for regular fields
		if len(fieldProcessor.processedFields) != 1 {
			t.Errorf("Expected 1 regular field processed, got %d", len(fieldProcessor.processedFields))
		}
	})

	t.Run("build schema with field processor error", func(t *testing.T) {
		fieldProcessor := &mockFieldProcessor{
			shouldError: true,
			errorMsg:    "field processing error",
		}
		embeddedProcessor := &mockEmbeddedStructProcessor{}
		typeRegistrar := &mockTypeRegistrar{shouldReturnRef: false}

		builder := NewStructSchemaBuilderWithProcessors(
			fieldProcessor,
			embeddedProcessor,
			typeRegistrar,
		)

		registry := make(map[string]*Schema)

		type TestStruct struct {
			Name string
			Age  int
		}

		schema := builder.BuildSchema(reflect.TypeOf(TestStruct{}), registry)

		// Should still return a schema even with field processing errors
		if schema == nil {
			t.Fatal("Expected schema to be generated despite field processing errors")
		}

		if schema.Type != "object" {
			t.Errorf("Expected type 'object', got '%s'", schema.Type)
		}
	})

	t.Run("build schema with embedded processor error", func(t *testing.T) {
		fieldProcessor := &mockFieldProcessor{}
		embeddedProcessor := &mockEmbeddedStructProcessor{
			shouldError: true,
			errorMsg:    "embedded processing error",
		}
		typeRegistrar := &mockTypeRegistrar{shouldReturnRef: false}

		builder := NewStructSchemaBuilderWithProcessors(
			fieldProcessor,
			embeddedProcessor,
			typeRegistrar,
		)

		registry := make(map[string]*Schema)

		type Embedded struct {
			EmbeddedField string
		}

		type TestStruct struct {
			Embedded // Anonymous embedded struct
			Name     string
		}

		schema := builder.BuildSchema(reflect.TypeOf(TestStruct{}), registry)

		// Should still return a schema even with embedded processing errors
		if schema == nil {
			t.Fatal("Expected schema to be generated despite embedded processing errors")
		}

		// Should still process regular fields
		if len(fieldProcessor.processedFields) != 1 {
			t.Errorf("Expected 1 regular field processed, got %d", len(fieldProcessor.processedFields))
		}
	})

	t.Run("build schema with unexported fields", func(t *testing.T) {
		fieldProcessor := &mockFieldProcessor{}
		embeddedProcessor := &mockEmbeddedStructProcessor{}
		typeRegistrar := &mockTypeRegistrar{shouldReturnRef: false}

		builder := NewStructSchemaBuilderWithProcessors(
			fieldProcessor,
			embeddedProcessor,
			typeRegistrar,
		)

		registry := make(map[string]*Schema)

		type TestStruct struct {
			Name string
			age  int // unexported field
		}

		schema := builder.BuildSchema(reflect.TypeOf(TestStruct{}), registry)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Should only process exported fields
		if len(fieldProcessor.processedFields) != 1 {
			t.Errorf("Expected 1 exported field processed, got %d", len(fieldProcessor.processedFields))
		}

		if fieldProcessor.processedFields[0] != "Name" {
			t.Errorf("Expected 'Name' field processed, got '%s'", fieldProcessor.processedFields[0])
		}
	})

	t.Run("build schema with embedded struct with json tag", func(t *testing.T) {
		fieldProcessor := &mockFieldProcessor{}
		embeddedProcessor := &mockEmbeddedStructProcessor{}
		typeRegistrar := &mockTypeRegistrar{shouldReturnRef: false}

		builder := NewStructSchemaBuilderWithProcessors(
			fieldProcessor,
			embeddedProcessor,
			typeRegistrar,
		)

		registry := make(map[string]*Schema)

		type Embedded struct {
			EmbeddedField string
		}

		type TestStruct struct {
			Embedded `json:"embedded"` // Has json tag, should be treated as regular field
			Name     string
		}

		schema := builder.BuildSchema(reflect.TypeOf(TestStruct{}), registry)

		if schema == nil {
			t.Fatal("Expected schema to be generated")
		}

		// Should not process as embedded struct (has json tag)
		if len(embeddedProcessor.processedEmbedded) != 0 {
			t.Errorf("Expected 0 embedded structs processed, got %d", len(embeddedProcessor.processedEmbedded))
		}

		// Should process as regular fields
		if len(fieldProcessor.processedFields) != 2 {
			t.Errorf("Expected 2 regular fields processed, got %d", len(fieldProcessor.processedFields))
		}
	})
}

func TestDefaultProcessors(t *testing.T) {
	t.Run("default field processor", func(t *testing.T) {
		processor := &defaultFieldProcessor{}
		schema := &Schema{Properties: make(map[string]*Schema)}
		registry := make(map[string]*Schema)

		field := reflect.StructField{
			Name: "TestField",
			Type: reflect.TypeOf(""),
		}

		err := processor.ProcessField(field, schema, registry)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("default embedded struct processor", func(t *testing.T) {
		processor := &defaultEmbeddedStructProcessor{}
		schema := &Schema{Properties: make(map[string]*Schema)}
		registry := make(map[string]*Schema)

		field := reflect.StructField{
			Name: "TestEmbedded",
			Type: reflect.TypeOf(struct{}{}),
		}

		err := processor.ProcessEmbedded(field, schema, registry)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("default type registrar with named type", func(t *testing.T) {
		registrar := &defaultTypeRegistrar{}
		schema := &Schema{Type: "object"}
		registry := make(map[string]*Schema)

		type NamedType struct{}

		result := registrar.RegisterType(reflect.TypeOf(NamedType{}), schema, registry)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.Ref == "" {
			t.Error("Expected reference to be returned for named type")
		}

		if _, exists := registry["NamedType"]; !exists {
			t.Error("Expected type to be registered in registry")
		}
	})

	t.Run("default type registrar with anonymous type", func(t *testing.T) {
		registrar := &defaultTypeRegistrar{}
		schema := &Schema{Type: "object"}
		registry := make(map[string]*Schema)

		// Anonymous struct
		result := registrar.RegisterType(reflect.TypeOf(struct{}{}), schema, registry)

		if result != schema {
			t.Error("Expected original schema to be returned for anonymous type")
		}

		if result.Ref != "" {
			t.Error("Expected no reference for anonymous type")
		}
	})
}
