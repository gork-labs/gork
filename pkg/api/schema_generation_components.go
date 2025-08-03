package api

import "reflect"

// TypeSchemaHandler handles schema generation for specific types.
type TypeSchemaHandler interface {
	CanHandle(t reflect.Type) bool
	GenerateSchema(t reflect.Type, registry map[string]*Schema, makePointerNullable bool) *Schema
}

// PointerTypeHandler handles pointer types.
type PointerTypeHandler struct{}

// CanHandle returns true if this handler can process the given type.
func (p *PointerTypeHandler) CanHandle(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr
}

// GenerateSchema generates a schema for pointer types.
func (p *PointerTypeHandler) GenerateSchema(t reflect.Type, registry map[string]*Schema, makePointerNullable bool) *Schema {
	if makePointerNullable {
		generator := NewSchemaGenerator()
		underlyingSchema := generator.GenerateSchema(t.Elem(), registry, true)
		return makeNullableSchema(underlyingSchema)
	}
	// For top-level types, just unwrap the pointer without making it nullable
	generator := NewSchemaGenerator()
	return generator.GenerateSchema(t.Elem(), registry, true)
}

// UnionTypeHandler handles union types.
type UnionTypeHandler struct{}

// CanHandle returns true if this handler can process the given type.
func (u *UnionTypeHandler) CanHandle(t reflect.Type) bool {
	return isUnionType(t) || isUnionStruct(t)
}

// GenerateSchema generates a schema for union types.
func (u *UnionTypeHandler) GenerateSchema(t reflect.Type, registry map[string]*Schema, _ bool) *Schema {
	return handleUnionType(t, registry)
}

// ExistingTypeHandler checks for already registered types.
type ExistingTypeHandler struct{}

// CanHandle returns true if this handler can process the given type.
func (e *ExistingTypeHandler) CanHandle(_ reflect.Type) bool {
	// This is checked via checkExistingType, not by kind
	return false // Always return false, handled specially
}

// GenerateSchema generates a schema for existing registered types.
func (e *ExistingTypeHandler) GenerateSchema(t reflect.Type, registry map[string]*Schema, _ bool) *Schema {
	return checkExistingType(t, registry)
}

// StructTypeHandler handles struct types.
type StructTypeHandler struct{}

// CanHandle returns true if this handler can process the given type.
func (s *StructTypeHandler) CanHandle(t reflect.Type) bool {
	return t.Kind() == reflect.Struct
}

// GenerateSchema generates a schema for struct types.
func (s *StructTypeHandler) GenerateSchema(t reflect.Type, registry map[string]*Schema, _ bool) *Schema {
	return buildStructSchema(t, registry)
}

// ArrayTypeHandler handles slice and array types.
type ArrayTypeHandler struct{}

// CanHandle returns true if this handler can process the given type.
func (a *ArrayTypeHandler) CanHandle(t reflect.Type) bool {
	return t.Kind() == reflect.Slice || t.Kind() == reflect.Array
}

// GenerateSchema generates a schema for array and slice types.
func (a *ArrayTypeHandler) GenerateSchema(t reflect.Type, registry map[string]*Schema, _ bool) *Schema {
	return buildArraySchema(t, registry)
}

// BasicTypeHandler handles basic types (string, int, etc.).
type BasicTypeHandler struct{}

// CanHandle returns true if this handler can process the given type.
func (b *BasicTypeHandler) CanHandle(_ reflect.Type) bool {
	// Handle everything not handled by other handlers
	return true
}

// GenerateSchema generates a schema for basic types.
func (b *BasicTypeHandler) GenerateSchema(t reflect.Type, registry map[string]*Schema, _ bool) *Schema {
	return buildBasicTypeSchemaWithRegistry(t, registry)
}

// SchemaGenerator orchestrates schema generation using handlers.
type SchemaGenerator struct {
	handlers []TypeSchemaHandler
}

// NewSchemaGenerator creates a new SchemaGenerator with default handlers.
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{
		handlers: []TypeSchemaHandler{
			&PointerTypeHandler{},
			&UnionTypeHandler{},
			&StructTypeHandler{},
			&ArrayTypeHandler{},
			&BasicTypeHandler{}, // Must be last as it accepts everything
		},
	}
}

// NewSchemaGeneratorWithHandlers creates a SchemaGenerator with custom handlers.
func NewSchemaGeneratorWithHandlers(handlers []TypeSchemaHandler) *SchemaGenerator {
	return &SchemaGenerator{
		handlers: handlers,
	}
}

// GenerateSchema generates a schema using the appropriate handler.
func (s *SchemaGenerator) GenerateSchema(t reflect.Type, registry map[string]*Schema, makePointerNullable bool) *Schema {
	// Special case: check for existing types first
	existingHandler := &ExistingTypeHandler{}
	if schema := existingHandler.GenerateSchema(t, registry, makePointerNullable); schema != nil {
		return schema
	}

	// Find the first handler that can handle this type
	for _, handler := range s.handlers {
		if handler.CanHandle(t) {
			return handler.GenerateSchema(t, registry, makePointerNullable)
		}
	}

	// Fallback to basic type handler
	basicHandler := &BasicTypeHandler{}
	return basicHandler.GenerateSchema(t, registry, makePointerNullable)
}
