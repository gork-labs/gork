package gorkson

import (
	"testing"
)

type EmbeddedBase struct {
	ID string `gork:"id"`
}

type EmbeddedFailing struct {
	Item failingMarshaler `gork:"item"`
}

func TestEmbeddedStructFieldsAreFlattened(t *testing.T) {
	type wrapper struct {
		EmbeddedBase
		Meta     EmbeddedBase `gork:"meta"`
		Name     string
		Internal string `json:"-"`
		secret   string
	}

	data, err := Marshal(wrapper{EmbeddedBase: EmbeddedBase{ID: "1"}, Meta: EmbeddedBase{ID: "2"}, Name: "n", Internal: "i", secret: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"Name":"n","id":"1","meta":{"id":"2"}}`; string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}

	var back wrapper
	if err := Unmarshal([]byte(`{"Name":"n","id":"1","meta":{"id":"2"},"Internal":"i","secret":"s"}`), &back); err != nil {
		t.Fatal(err)
	}
	if back != (wrapper{EmbeddedBase: EmbeddedBase{ID: "1"}, Meta: EmbeddedBase{ID: "2"}, Name: "n"}) {
		t.Errorf("Unmarshal() = %+v, want id 1, meta id 2 and name n", back)
	}
}

func TestEmbeddedStructReturnsMarshalError(t *testing.T) {
	type wrapper struct {
		EmbeddedFailing
	}

	if _, err := Marshal(wrapper{}); err == nil {
		t.Error("expected the error of the embedded field")
	}
}
