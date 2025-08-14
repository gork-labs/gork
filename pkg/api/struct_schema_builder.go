package api

import "reflect"

// FieldProcessor handles processing of struct fields.
type FieldProcessor interface {
	ProcessField(field reflect.StructField, schema *Schema, registry map[string]*Schema) error
}

// EmbeddedStructProcessor handles processing of embedded structs.
type EmbeddedStructProcessor interface {
	ProcessEmbedded(field reflect.StructField, schema *Schema, registry map[string]*Schema) error
}

// TypeRegistrar handles registration of named types.
type TypeRegistrar interface {
	RegisterType(t reflect.Type, schema *Schema, registry map[string]*Schema) *Schema
}

// StructSchemaBuilder builds struct schemas using dependency injection.
type StructSchemaBuilder struct {
	fieldProcessor          FieldProcessor
	embeddedStructProcessor EmbeddedStructProcessor
	typeRegistrar           TypeRegistrar
}

// NewStructSchemaBuilder creates a new builder with default processors.
func NewStructSchemaBuilder() *StructSchemaBuilder {
	return &StructSchemaBuilder{
		fieldProcessor:          &defaultFieldProcessor{},
		embeddedStructProcessor: &defaultEmbeddedStructProcessor{},
		typeRegistrar:           &defaultTypeRegistrar{},
	}
}

// NewStructSchemaBuilderWithProcessors creates a builder with custom processors.
func NewStructSchemaBuilderWithProcessors(
	fieldProcessor FieldProcessor,
	embeddedStructProcessor EmbeddedStructProcessor,
	typeRegistrar TypeRegistrar,
) *StructSchemaBuilder {
	return &StructSchemaBuilder{
		fieldProcessor:          fieldProcessor,
		embeddedStructProcessor: embeddedStructProcessor,
		typeRegistrar:           typeRegistrar,
	}
}

// BuildSchema builds a schema for the given struct type.
func (b *StructSchemaBuilder) BuildSchema(t reflect.Type, registry map[string]*Schema) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: map[string]*Schema{},
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}

		// Handle embedded structs
		if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("json") == "" {
			if err := b.embeddedStructProcessor.ProcessEmbedded(f, s, registry); err != nil {
				// Log error but continue processing other fields
				continue
			}
			continue
		}

		// Process regular field
		if err := b.fieldProcessor.ProcessField(f, s, registry); err != nil {
			// Log error but continue processing other fields
			continue
		}
	}

	// Register named types
	return b.typeRegistrar.RegisterType(t, s, registry)
}

// Default implementations

type defaultFieldProcessor struct{}

func (p *defaultFieldProcessor) ProcessField(field reflect.StructField, schema *Schema, registry map[string]*Schema) error {
	processStructField(field, schema, registry)
	return nil
}

type defaultEmbeddedStructProcessor struct{}

func (p *defaultEmbeddedStructProcessor) ProcessEmbedded(field reflect.StructField, schema *Schema, registry map[string]*Schema) error {
	processEmbeddedStruct(field, schema, registry)
	return nil
}

type defaultTypeRegistrar struct{}

func (r *defaultTypeRegistrar) RegisterType(t reflect.Type, schema *Schema, registry map[string]*Schema) *Schema {
	rawName := t.Name()
	typeName := sanitizeSchemaName(rawName)
	if typeName != "" {
		// Pick a human-friendly unique name to avoid collisions
		unique := uniqueSchemaNameForType(t, registry)
		schema.Title = unique
		registry[unique] = schema
		return &Schema{Ref: "#/components/schemas/" + unique}
	}
	return schema
}
