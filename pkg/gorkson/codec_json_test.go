package gorkson

import (
	"strings"
	"testing"
	"time"
)

type timeFields struct {
	CreatedAt time.Time   `gork:"created_at"`
	UpdatedAt *time.Time  `gork:"updated_at"`
	DeletedAt *time.Time  `gork:"deleted_at"`
	History   []time.Time `gork:"history"`
}

func TestMarshal_TimeUsesRFC3339(t *testing.T) {
	at := time.Date(2024, 1, 2, 3, 4, 5, 0, time.FixedZone("CET", 3600))
	data, err := Marshal(timeFields{CreatedAt: at, UpdatedAt: &at, History: []time.Time{at}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := `{"created_at":"2024-01-02T03:04:05+01:00","deleted_at":null,"history":["2024-01-02T03:04:05+01:00"],"updated_at":"2024-01-02T03:04:05+01:00"}`
	if string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}
}

func TestUnmarshal_TimeFromRFC3339(t *testing.T) {
	var got timeFields
	err := Unmarshal([]byte(`{"created_at":"2024-01-02T03:04:05Z","updated_at":"2024-01-02T03:04:05+01:00","deleted_at":null}`), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !got.CreatedAt.Equal(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("CreatedAt = %v", got.CreatedAt)
	}
	if got.UpdatedAt == nil || !got.UpdatedAt.Equal(time.Date(2024, 1, 2, 2, 4, 5, 0, time.UTC)) {
		t.Errorf("UpdatedAt = %v", got.UpdatedAt)
	}
	if got.DeletedAt != nil {
		t.Errorf("DeletedAt = %v, want nil", got.DeletedAt)
	}
}

func TestUnmarshal_CodecErrors(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Level](levelCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	tests := []struct {
		name   string
		data   string
		target any
		errMsg string
	}{
		{"invalid time", `{"created_at":"yesterday"}`, &timeFields{}, "failed to parse time.Time"},
		{"invalid pointer time", `{"updated_at":"yesterday"}`, &timeFields{}, "failed to parse time.Time"},
		{"time as number", `{"created_at":12}`, &timeFields{}, "failed to parse time.Time"},
		{"schema validation", `{"level":-1}`, &struct {
			Level *Level `gork:"level"`
		}{}, "number too small"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.data), tt.target)
			if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Unmarshal() error = %v, want it to contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestCodecJSON_NonStringSchemaUsesJSONValue(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Level](levelCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	type payload struct {
		Level  Level   `gork:"level"`
		Levels []Level `gork:"levels"`
	}

	data, err := Marshal(payload{Level: Level{Value: 3}, Levels: []Level{{Value: 1}}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if want := `{"level":3,"levels":[1]}`; string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}

	var got payload
	if err := Unmarshal([]byte(`{"level":5}`), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Level.Value != 5 {
		t.Errorf("Level = %d, want 5", got.Level.Value)
	}

	if err := Unmarshal([]byte(`{"level":"6"}`), &got); err != nil {
		t.Fatalf("Unmarshal() from JSON string error = %v", err)
	}
	if got.Level.Value != 6 {
		t.Errorf("Level = %d, want 6", got.Level.Value)
	}
}

func TestMarshal_CodecFormatErrors(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Level](levelCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	bad := Level{Value: -1}
	tests := []struct {
		name  string
		value any
	}{
		{"top-level value", bad},
		{"struct field", struct {
			Level Level `gork:"level"`
		}{bad}},
		{"slice item", []Level{bad}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Marshal(tt.value); err == nil {
				t.Error("Marshal() expected codec format error")
			}
		})
	}
}

func TestMarshal_InterfaceField(t *testing.T) {
	at := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	data, err := Marshal(struct {
		Value any `gork:"value"`
		Empty any `gork:"empty"`
	}{Value: at})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if want := `{"empty":null,"value":"2024-01-02T03:04:05Z"}`; string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}
}

func TestUnmarshal_PointerToBasicType(t *testing.T) {
	var got struct {
		Name *string `gork:"name"`
	}
	if err := Unmarshal([]byte(`{"name":"x"}`), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Name == nil || *got.Name != "x" {
		t.Errorf("Name = %v, want x", got.Name)
	}
}

func TestMarshal_Nil(t *testing.T) {
	data, err := Marshal(nil)
	if err != nil || string(data) != "null" {
		t.Errorf("Marshal(nil) = %s, %v, want null", data, err)
	}
}

func TestUnmarshal_SliceItemsUseCodecs(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Level](levelCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	var got struct {
		Levels []Level        `gork:"levels"`
		Users  []SimpleStruct `gork:"users"`
		Times  []time.Time    `gork:"times"`
		Bad    []Level        `gork:"bad"`
	}
	data := `{"levels":[1,2],"users":[{"name":"Ann"}],"times":["2024-01-02T03:04:05Z"]}`
	if err := Unmarshal([]byte(data), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(got.Levels) != 2 || got.Levels[1].Value != 2 {
		t.Errorf("Levels = %v", got.Levels)
	}
	if len(got.Users) != 1 || got.Users[0].Name != "Ann" {
		t.Errorf("Users = %v", got.Users)
	}
	if len(got.Times) != 1 || !got.Times[0].Equal(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("Times = %v", got.Times)
	}

	if err := Unmarshal([]byte(`{"bad":[-1]}`), &got); err == nil {
		t.Error("Unmarshal() expected a schema validation error for a slice item")
	}
}

func TestUnmarshal_MapField(t *testing.T) {
	var got struct {
		Tags map[string]string `gork:"tags"`
	}
	if err := Unmarshal([]byte(`{"tags":{"a":"b"}}`), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Tags["a"] != "b" {
		t.Errorf("Tags = %v", got.Tags)
	}
}
