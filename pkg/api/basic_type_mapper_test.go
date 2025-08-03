package api

import (
	"reflect"
	"testing"
)

func TestMapBasicKind(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		expected *Schema
	}{
		{
			name:     "string type",
			kind:     reflect.String,
			expected: &Schema{Type: "string"},
		},
		{
			name:     "int type",
			kind:     reflect.Int,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "int8 type",
			kind:     reflect.Int8,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "int16 type",
			kind:     reflect.Int16,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "int32 type",
			kind:     reflect.Int32,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "int64 type",
			kind:     reflect.Int64,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "uint type",
			kind:     reflect.Uint,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "uint8 type",
			kind:     reflect.Uint8,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "uint16 type",
			kind:     reflect.Uint16,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "uint32 type",
			kind:     reflect.Uint32,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "uint64 type",
			kind:     reflect.Uint64,
			expected: &Schema{Type: "integer"},
		},
		{
			name:     "float32 type",
			kind:     reflect.Float32,
			expected: &Schema{Type: "number"},
		},
		{
			name:     "float64 type",
			kind:     reflect.Float64,
			expected: &Schema{Type: "number"},
		},
		{
			name:     "bool type",
			kind:     reflect.Bool,
			expected: &Schema{Type: "boolean"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapBasicKind(tt.kind)
			if result.Type != tt.expected.Type {
				t.Errorf("Expected type %s, got %s", tt.expected.Type, result.Type)
			}
		})
	}
}

func TestMapAdvancedKind(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		expected *Schema
	}{
		{
			name:     "uintptr type",
			kind:     reflect.Uintptr,
			expected: &Schema{Type: "integer", Description: "Pointer-sized integer"},
		},
		{
			name:     "complex64 type",
			kind:     reflect.Complex64,
			expected: &Schema{Type: "object", Description: "Complex number"},
		},
		{
			name:     "complex128 type",
			kind:     reflect.Complex128,
			expected: &Schema{Type: "object", Description: "Complex number"},
		},
		{
			name:     "chan type",
			kind:     reflect.Chan,
			expected: &Schema{Type: "object", Description: "Channel"},
		},
		{
			name:     "func type",
			kind:     reflect.Func,
			expected: &Schema{Type: "object", Description: "Function"},
		},
		{
			name:     "interface type",
			kind:     reflect.Interface,
			expected: &Schema{Type: "object", Description: "Interface"},
		},
		{
			name:     "map type",
			kind:     reflect.Map,
			expected: &Schema{Type: "object", Description: "Map with dynamic keys"},
		},
		{
			name:     "unsafe pointer type",
			kind:     reflect.UnsafePointer,
			expected: &Schema{Type: "object", Description: "Unsafe pointer"},
		},
		{
			name:     "unknown type",
			kind:     reflect.Invalid,
			expected: &Schema{Type: "object"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapAdvancedKind(tt.kind)
			if result.Type != tt.expected.Type {
				t.Errorf("Expected type %s, got %s", tt.expected.Type, result.Type)
			}
			if result.Description != tt.expected.Description {
				t.Errorf("Expected description %s, got %s", tt.expected.Description, result.Description)
			}
		})
	}
}

func TestDefaultBasicTypeMapper_MapType(t *testing.T) {
	mapper := defaultBasicTypeMapper{}

	// Test basic types
	stringSchema := mapper.MapType(reflect.String)
	if stringSchema.Type != "string" {
		t.Errorf("Expected string type, got %s", stringSchema.Type)
	}

	// Test advanced types
	chanSchema := mapper.MapType(reflect.Chan)
	if chanSchema.Type != "object" || chanSchema.Description != "Channel" {
		t.Errorf("Expected object type with Channel description, got %s %s", chanSchema.Type, chanSchema.Description)
	}
}

func TestBuildBasicTypeSchema(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{
			name:     "string type",
			typ:      reflect.TypeOf(""),
			expected: "string",
		},
		{
			name:     "int type",
			typ:      reflect.TypeOf(0),
			expected: "integer",
		},
		{
			name:     "bool type",
			typ:      reflect.TypeOf(true),
			expected: "boolean",
		},
		{
			name:     "float64 type",
			typ:      reflect.TypeOf(0.0),
			expected: "number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildBasicTypeSchema(tt.typ)
			if result.Type != tt.expected {
				t.Errorf("Expected type %s, got %s", tt.expected, result.Type)
			}
		})
	}
}

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
			name:     "simple type",
			input:    "User",
			expected: "User",
		},
		{
			name:     "generic type with package paths",
			input:    "Union2[github.com/foo.Bar,github.com/foo.Baz]",
			expected: "Union2_Bar_Baz",
		},
		{
			name:     "type with special characters",
			input:    "*api.Handler",
			expected: "_api.Handler",
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

func TestBuildBasicTypeSchemaWithRegistryPointerTypes(t *testing.T) {
	t.Run("non-pointer type", func(t *testing.T) {
		registry := make(map[string]*Schema)
		stringType := reflect.TypeOf("")

		result := buildBasicTypeSchemaWithRegistry(stringType, registry)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}
		if result.Type != "string" {
			t.Errorf("Expected schema type to be 'string', got '%s'", result.Type)
		}
	})

	t.Run("pointer type", func(t *testing.T) {
		registry := make(map[string]*Schema)
		pointerType := reflect.TypeOf((*string)(nil))

		result := buildBasicTypeSchemaWithRegistry(pointerType, registry)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		t.Logf("Pointer type result: %+v", result)

		if result.Type == "" && result.Ref == "" && result.AnyOf == nil && result.Types == nil {
			t.Error("Expected some schema content to be generated for pointer type")
		}
	})

	t.Run("pointer to complex type", func(t *testing.T) {
		registry := make(map[string]*Schema)
		type CustomStruct struct {
			Field string
		}
		pointerType := reflect.TypeOf((*CustomStruct)(nil))

		result := buildBasicTypeSchemaWithRegistry(pointerType, registry)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		if result.AnyOf != nil && len(result.AnyOf) == 2 {
			foundRef := false
			foundNull := false
			for _, schema := range result.AnyOf {
				if schema.Ref != "" {
					foundRef = true
				} else if schema.Type == "null" {
					foundNull = true
				}
			}
			if !foundRef {
				t.Error("Expected to find reference to struct schema in nullable schema")
			}
			if !foundNull {
				t.Error("Expected to find null type in nullable schema")
			}
		} else {
			t.Logf("Complex pointer type result: %+v", result)
		}
	})

	t.Run("integer type", func(t *testing.T) {
		registry := make(map[string]*Schema)
		intType := reflect.TypeOf(0)

		result := buildBasicTypeSchemaWithRegistry(intType, registry)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}
		if result.Type != "integer" {
			t.Errorf("Expected schema type to be 'integer', got '%s'", result.Type)
		}
	})

	t.Run("pointer to integer", func(t *testing.T) {
		registry := make(map[string]*Schema)
		pointerIntType := reflect.TypeOf((*int)(nil))

		result := buildBasicTypeSchemaWithRegistry(pointerIntType, registry)

		if result == nil {
			t.Fatal("Expected schema to be returned")
		}

		t.Logf("Pointer to integer result: %+v", result)

		if result.Type == "" && result.Ref == "" && result.AnyOf == nil && result.Types == nil {
			t.Error("Expected some schema content to be generated for pointer to integer")
		}
	})
}
