package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

type mapSchemaValue struct {
	Name string `gork:"name"`
}

func TestMapSchemaAdditionalProperties(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"string to int", reflect.TypeOf(map[string]int{}), `{"type":"object","additionalProperties":{"type":"integer"}}`},
		{"string to any", reflect.TypeOf(map[string]any{}), `{"type":"object","additionalProperties":{}}`},
		{"string to string slice", reflect.TypeOf(map[string][]string{}), `{"type":"object","additionalProperties":{"type":"array","title":"[]string","description":"Array of string","items":{"type":"string"}}}`},
		{"string to struct", reflect.TypeOf(map[string]mapSchemaValue{}), `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/mapSchemaValue"}}`},
		{"string to pointer", reflect.TypeOf(map[string]*int{}), `{"type":"object","additionalProperties":{"type":["integer","null"]}}`},
		{"pointer to map", reflect.TypeOf(&struct {
			M *map[string]bool `gork:"m"`
		}{}).Elem(), `{"type":"object","properties":{"m":{"type":["object","null"],"additionalProperties":{"type":"boolean"}}}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(reflectTypeToSchema(tt.typ, map[string]*Schema{}))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("schema = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestErrorSchemaDetails(t *testing.T) {
	components := &Components{}
	(&ConventionOpenAPIGenerator{}).ensureErrorSchemas(components)

	tests := []struct {
		schema string
		want   string
	}{
		{"ErrorResponse", `{"type":"object","description":"Additional error details","additionalProperties":{}}`},
		{"ValidationErrorResponse", `{"type":"object","description":"Field-level validation errors (maps field names to arrays of error messages)","additionalProperties":{"type":"array","items":{"type":"string"}}}`},
	}

	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			got, err := json.Marshal(components.Schemas[tt.schema].Properties["details"])
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("details = %s, want %s", got, tt.want)
			}
		})
	}
}
