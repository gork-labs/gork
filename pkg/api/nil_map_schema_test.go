package api

import (
	"reflect"
	"testing"
)

func TestMapFieldSchemaIsNotNullable(t *testing.T) {
	type labeled struct {
		Labels map[string]string `gork:"labels"`
	}

	registry := map[string]*Schema{}
	reflectTypeToSchema(reflect.TypeOf(labeled{}), registry)

	labels := registry["labeled"].Properties["labels"]
	if labels.Type != "object" || labels.Types != nil || labels.AnyOf != nil {
		t.Errorf("labels = %+v, want an object schema without null, because gorkson writes a nil map as {}", labels)
	}
}
