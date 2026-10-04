package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/gork-labs/gork/pkg/gorkson"
)

func TestConvertCodecSchema(t *testing.T) {
	minLength, maxLength := 2, 8
	minimum, maximum := 1.0, 9.0
	codecSchema := gorkson.OpenAPISchema{
		Type:        gorkson.OpenAPITypeObject,
		Format:      "custom",
		Pattern:     "^a",
		Example:     map[string]any{"id": 1},
		Description: "An object",
		MinLength:   &minLength,
		MaxLength:   &maxLength,
		Minimum:     &minimum,
		Maximum:     &maximum,
		Enum:        []interface{}{"low", 2},
		Properties: map[string]*gorkson.OpenAPISchema{
			"id": {Type: gorkson.OpenAPITypeInteger},
		},
		Items: &gorkson.OpenAPISchema{Type: gorkson.OpenAPITypeString},
	}

	got := convertCodecSchema(codecSchema)

	want := &Schema{
		Type:        "object",
		Format:      "custom",
		Pattern:     "^a",
		Example:     map[string]any{"id": 1},
		Description: "An object",
		MinLength:   &minLength,
		MaxLength:   &maxLength,
		Minimum:     &minimum,
		Maximum:     &maximum,
		Enum:        []string{"low", "2"},
		Properties:  map[string]*Schema{"id": {Type: "integer"}},
		Items:       &Schema{Type: "string"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("convertCodecSchema() = %+v, want %+v", got, want)
	}
}

type codecSchemaLevel int

type codecSchemaLevelCodec struct{}

func (codecSchemaLevelCodec) Parse(context.Context, string) (*codecSchemaLevel, error) {
	level := codecSchemaLevel(1)
	return &level, nil
}

func (codecSchemaLevelCodec) Format(context.Context, *codecSchemaLevel) (string, error) {
	return "low", nil
}

func (codecSchemaLevelCodec) Schema() gorkson.OpenAPISchema {
	return gorkson.OpenAPISchema{Type: gorkson.OpenAPITypeString, Enum: []interface{}{"low", "high"}}
}

func TestCodecTypeHandler_UsedForCodecTypes(t *testing.T) {
	if err := gorkson.RegisterCodec[codecSchemaLevel](codecSchemaLevelCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	type withLevel struct {
		Level    codecSchemaLevel  `gork:"level"`
		Optional *codecSchemaLevel `gork:"optional"`
	}

	registry := map[string]*Schema{}
	ref := reflectTypeToSchema(reflect.TypeOf(withLevel{}), registry)
	schema := registry[ref.Ref[len("#/components/schemas/"):]]

	if level := schema.Properties["level"]; level.Type != "string" || !reflect.DeepEqual(level.Enum, []string{"low", "high"}) {
		t.Errorf("level schema = %+v, want the codec schema", level)
	}
	if optional := schema.Properties["optional"]; !reflect.DeepEqual(optional.Types, []string{"string", "null"}) || len(optional.Enum) != 2 {
		t.Errorf("optional schema = %+v, want the nullable codec schema", optional)
	}
	if (&CodecTypeHandler{}).CanHandle(reflect.TypeOf(0)) {
		t.Error("CanHandle(int) = true, want false")
	}
}
