package gorkson

import "testing"

func TestMarshalWritesNilMapAsEmptyObject(t *testing.T) {
	type wrapper struct {
		Labels map[string]string `gork:"labels"`
		Tags   []string          `gork:"tags"`
	}

	data, err := Marshal(wrapper{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"labels":{},"tags":[]}`; string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}
}
