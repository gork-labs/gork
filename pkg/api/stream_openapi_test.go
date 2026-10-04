package api

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func newStreamTestRegistry(t *testing.T, handlers ...interface{}) *RouteRegistry {
	t.Helper()
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	for i, handler := range handlers {
		router.Get("/route"+string(rune('a'+i)), handler)
	}
	return registry
}

func TestStreamOpenAPIResponse(t *testing.T) {
	registry := newStreamTestRegistry(t, func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error { return nil })

	spec := GenerateOpenAPI(registry)

	if spec.OpenAPI != "3.2.0" {
		t.Errorf("expected openapi 3.2.0 for a spec with a stream route, got %s", spec.OpenAPI)
	}
	op := spec.Paths["/routea"].Get
	for _, code := range []string{"400", "422", "500"} {
		if op.Responses[code] == nil {
			t.Errorf("expected standard %s response on a stream route", code)
		}
	}
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "topic" {
		t.Errorf("expected the topic query parameter, got %+v", op.Parameters)
	}

	media := op.Responses["200"].Content["text/event-stream"]
	if media == nil || media.Schema != nil || media.ItemSchema == nil {
		t.Fatalf("expected text/event-stream with itemSchema, got %+v", op.Responses["200"].Content)
	}
	events := media.ItemSchema.OneOf
	if len(events) != 2 {
		t.Fatalf("expected 2 event schemas, got %d", len(events))
	}

	row := events[0]
	if row.Type != "object" || !reflect.DeepEqual(row.Required, []string{"event", "data"}) {
		t.Errorf("unexpected event schema: %+v", row)
	}
	if row.Properties["event"].Const != "row" {
		t.Errorf("expected event const row, got %q", row.Properties["event"].Const)
	}
	data := row.Properties["data"]
	if data.ContentMediaType != "application/json" || data.ContentSchema.Ref != "#/components/schemas/streamTestRow" {
		t.Errorf("unexpected data schema: %+v", data)
	}
	if spec.Components.Schemas["streamTestRow"] == nil {
		t.Error("expected the payload type as a component schema")
	}
	if events[1].Properties["event"].Const != "done" || events[1].Properties["data"].ContentSchema == nil {
		t.Errorf("unexpected empty event schema: %+v", events[1])
	}
}

func TestStreamOpenAPIVersion(t *testing.T) {
	plain := GenerateOpenAPI(newStreamTestRegistry(t, func(context.Context, streamTestRequest) error { return nil }))
	if plain.OpenAPI != "3.1.0" {
		t.Errorf("expected openapi 3.1.0 without stream routes, got %s", plain.OpenAPI)
	}

	filtered := GenerateOpenAPI(
		newStreamTestRegistry(t, func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error { return nil }),
		WithRouteFilter(func(*RouteInfo) bool { return false }),
	)
	if filtered.OpenAPI != "3.1.0" {
		t.Errorf("expected openapi 3.1.0 when the stream route is filtered out, got %s", filtered.OpenAPI)
	}
}

func TestStreamOpenAPIJSON(t *testing.T) {
	spec := GenerateOpenAPI(newStreamTestRegistry(t, func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error { return nil }))

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	var doc struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]map[string]struct {
			Responses map[string]struct {
				Content map[string]json.RawMessage `json:"content"`
			} `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}

	want := `{"itemSchema":{"oneOf":[
		{"type":"object","required":["event","data"],"properties":{
			"event":{"const":"row"},
			"data":{"contentMediaType":"application/json","contentSchema":{"$ref":"#/components/schemas/streamTestRow"}}}},
		{"type":"object","required":["event","data"],"properties":{
			"event":{"const":"done"},
			"data":{"contentMediaType":"application/json","contentSchema":{"type":"object"}}}}
	]}}`
	var gotValue, wantValue interface{}
	got := doc.Paths["/routea"]["get"].Responses["200"].Content["text/event-stream"]
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal media type: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("unmarshal expected media type: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("unexpected text/event-stream media type:\n%s\nwant:\n%s", got, want)
	}
}
