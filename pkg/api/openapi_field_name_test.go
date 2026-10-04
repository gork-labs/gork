package api

import (
	"reflect"
	"testing"
)

func TestGetOpenAPIFieldName(t *testing.T) {
	type sample struct {
		Gork       string `gork:"gork_name" json:"other_name"`
		GorkSkip   string `gork:"-"`
		JSON       string `json:"json_name,omitempty"`
		JSONSkip   string `json:"-"`
		JSONNoName string `json:",omitempty"`
		Plain      string
	}

	want := map[string]string{
		"Gork":       "gork_name",
		"GorkSkip":   "",
		"JSON":       "json_name",
		"JSONSkip":   "",
		"JSONNoName": "JSONNoName",
		"Plain":      "Plain",
	}

	typ := reflect.TypeOf(sample{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if got := getOpenAPIFieldName(field); got != want[field.Name] {
			t.Errorf("getOpenAPIFieldName(%s) = %q, want %q", field.Name, got, want[field.Name])
		}
	}
}

func TestBuildStructSchema_SkipsIgnoredFields(t *testing.T) {
	type withIgnored struct {
		Name    string `json:"name" validate:"required"`
		Ignored string `json:"-" validate:"required"`
	}

	registry := map[string]*Schema{}
	ref := reflectTypeToSchema(reflect.TypeOf(withIgnored{}), registry)
	schema := registry[ref.Ref[len("#/components/schemas/"):]]

	if len(schema.Properties) != 1 || schema.Properties["name"] == nil {
		t.Errorf("properties = %v, want only name", schema.Properties)
	}
	if !reflect.DeepEqual(schema.Required, []string{"name"}) {
		t.Errorf("required = %v, want [name]", schema.Required)
	}
}

type FlattenBase struct {
	ID string `gork:"id" validate:"required"`
}

func TestExtractStructPropertiesToSchema_FlattensEmbeddedStructs(t *testing.T) {
	type withEmbedded struct {
		FlattenBase
		Name    string `gork:"name"`
		Ignored string `gork:"-"`
	}

	schema := &Schema{Properties: map[string]*Schema{}}
	generator := &ConventionOpenAPIGenerator{}
	generator.extractStructPropertiesToSchema(reflect.TypeOf(withEmbedded{}), schema, &Components{Schemas: map[string]*Schema{}})

	if len(schema.Properties) != 2 || schema.Properties["id"] == nil || schema.Properties["name"] == nil {
		t.Errorf("properties = %v, want id and name", schema.Properties)
	}
	if !reflect.DeepEqual(schema.Required, []string{"id"}) {
		t.Errorf("required = %v, want [id]", schema.Required)
	}
}
