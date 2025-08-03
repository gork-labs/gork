package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

func TestHandleUnionType(t *testing.T) {
	t.Run("union type with valid name", func(t *testing.T) {
		registry := make(map[string]*Schema)

		// Create a union type with a valid name
		unionType := reflect.TypeOf(unions.Union2[string, int]{})

		result := handleUnionType(unionType, registry)

		// Should return a reference since the type name can be sanitized
		if result.Ref == "" {
			t.Error("Expected a reference schema for union type with valid name")
		}

		// Check that the schema was added to registry
		if len(registry) == 0 {
			t.Error("Expected schema to be added to registry")
		}

		// The reference should point to a schema in the registry
		expectedRef := "#/components/schemas/"
		if !strings.HasPrefix(result.Ref, expectedRef) {
			t.Errorf("Expected reference to start with '%s', got '%s'", expectedRef, result.Ref)
		}
	})

	t.Run("union type with empty sanitized name", func(t *testing.T) {
		registry := make(map[string]*Schema)

		// Create an anonymous struct type that will result in empty sanitized name
		// Anonymous structs have empty names and will result in empty sanitized names
		anonUnionType := reflect.TypeOf(struct {
			Field1 *string
			Field2 *int
		}{})

		result := handleUnionType(anonUnionType, registry)

		// Should return the schema directly (not a reference) since typeName is empty
		if result.Ref != "" {
			t.Errorf("Expected direct schema (no reference) for type with empty sanitized name, got ref: %s", result.Ref)
		}

		// Should have OneOf since it's treated as a union
		if result.OneOf == nil {
			t.Error("Expected OneOf schema for union type")
		}

		// Registry should be empty since no schema was added (typeName was empty)
		if len(registry) != 0 {
			t.Errorf("Expected registry to be empty for type with empty sanitized name, got %d entries", len(registry))
		}
	})

	t.Run("union type with special characters in name", func(t *testing.T) {
		registry := make(map[string]*Schema)

		// Create a type that has a name but might result in empty after sanitization
		type SpecialUnion struct {
			Field1 *string
			Field2 *int
		}

		unionType := reflect.TypeOf(SpecialUnion{})

		result := handleUnionType(unionType, registry)

		// Since "SpecialUnion" is a valid name, it should create a reference
		if result.Ref == "" {
			t.Error("Expected a reference schema for union type with valid name")
		}

		// Check that schema was added to registry
		if len(registry) == 0 {
			t.Error("Expected schema to be added to registry")
		}
	})
}
