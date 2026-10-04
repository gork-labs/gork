package api

import (
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/gork-labs/gork/pkg/gorkson"
)

type WireNamesBase struct {
	ID string `gork:"id"`
}

type wireNamesPayload struct {
	WireNamesBase
	Meta     WireNamesBase `gork:"meta"`
	Name     string
	Internal string `json:"-"`
}

type wireNamesResponse struct {
	Body wireNamesPayload
}

func TestSchemaPropertiesMatchTheWrittenKeys(t *testing.T) {
	data, err := gorkson.Marshal(wireNamesPayload{})
	if err != nil {
		t.Fatal(err)
	}
	var written map[string]any
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatal(err)
	}

	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/names", func(context.Context, struct{}) (*wireNamesResponse, error) { return nil, nil })
	schema := GenerateOpenAPI(registry).Components.Schemas["wireNamesPayload"]

	want := slices.Sorted(maps.Keys(written))
	if got := slices.Sorted(maps.Keys(schema.Properties)); !reflect.DeepEqual(got, want) {
		t.Errorf("properties = %v, want the written keys %v", got, want)
	}
	if got := slices.Sorted(slices.Values(schema.Required)); !reflect.DeepEqual(got, want) {
		t.Errorf("required = %v, want the written keys %v", got, want)
	}
}
