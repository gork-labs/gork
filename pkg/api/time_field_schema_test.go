package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type TimeEventWindow struct {
	StartsAt time.Time  `gork:"starts_at"`
	EndsAt   *time.Time `gork:"ends_at"`
}

type TimeEventRequest struct {
	Path struct {
		At time.Time `gork:"at"`
	}
	Query struct {
		Since *time.Time `gork:"since"`
	}
	Body struct {
		CreatedAt time.Time       `gork:"created_at"`
		DeletedAt *time.Time      `gork:"deleted_at"`
		Window    TimeEventWindow `gork:"window"`
	}
}

type TimeEventResponse struct {
	Body struct {
		UpdatedAt time.Time  `gork:"updated_at"`
		ExpiresAt *time.Time `gork:"expires_at"`
	}
}

func TimeEventHandler(_ context.Context, _ TimeEventRequest) (*TimeEventResponse, error) {
	return &TimeEventResponse{}, nil
}

func generateTimeEventSpec(t *testing.T) *OpenAPISpec {
	t.Helper()
	_, info := NewConventionHandlerFactory().CreateHandler(&mockConventionParameterAdapter{}, TimeEventHandler)
	info.Method = "POST"
	info.Path = "/events/{at}"
	registry := NewRouteRegistry()
	registry.Register(info)
	return GenerateOpenAPI(registry)
}

func schemaJSON(t *testing.T, schema *Schema) string {
	t.Helper()
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	return string(data)
}

func TestGenerateOpenAPI_TimeFieldsUseDateTimeSchema(t *testing.T) {
	const dateTime = `{"type":"string","format":"date-time"}`
	const nullableDateTime = `{"type":["string","null"],"format":"date-time"}`

	spec := generateTimeEventSpec(t)
	op := spec.Paths["/events/{at}"].Post

	params := map[string]*Schema{}
	for _, p := range op.Parameters {
		params[p.In+":"+p.Name] = p.Schema
	}
	body := spec.Components.Schemas["TimeEventBody"]
	window := spec.Components.Schemas["TimeEventWindow"]
	response := spec.Components.Schemas["TimeEventResponse"]
	if body == nil || window == nil || response == nil {
		t.Fatalf("missing component schemas: %v", spec.Components.Schemas)
	}

	tests := []struct {
		name   string
		schema *Schema
		want   string
	}{
		{"path time.Time", params["path:at"], dateTime},
		{"query *time.Time", params["query:since"], dateTime},
		{"body time.Time", body.Properties["created_at"], dateTime},
		{"body *time.Time", body.Properties["deleted_at"], dateTime},
		{"nested time.Time", window.Properties["starts_at"], dateTime},
		{"nested *time.Time", window.Properties["ends_at"], nullableDateTime},
		{"response time.Time", response.Properties["updated_at"], dateTime},
		{"response *time.Time", response.Properties["expires_at"], nullableDateTime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := schemaJSON(t, tt.schema); got != tt.want {
				t.Errorf("schema = %s, want %s", got, tt.want)
			}
		})
	}

	if _, exists := spec.Components.Schemas["Time"]; exists {
		t.Error("time.Time must not produce a Time component schema")
	}
}
