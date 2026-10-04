package gorkson

import (
	"errors"
	"testing"
)

type failingMarshaler struct{}

func (failingMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func TestMarshalUsesMarshalJSONOfNestedValues(t *testing.T) {
	type wrapper struct {
		Item  CustomMarshaler   `gork:"item"`
		Items []CustomMarshaler `gork:"items"`
	}

	data, err := Marshal(wrapper{Item: CustomMarshaler{Value: "a"}, Items: []CustomMarshaler{{Value: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"item":{"custom":"a"},"items":[{"custom":"b"}]}`; string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}

	var back wrapper
	if err := Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Item.Value != "a" || back.Items[0].Value != "b" {
		t.Errorf("Unmarshal() = %+v, want the values a and b", back)
	}
}

func TestMarshalReturnsErrorOfNestedMarshalJSON(t *testing.T) {
	type wrapper struct {
		Item failingMarshaler `gork:"item"`
	}

	if _, err := Marshal(wrapper{}); err == nil || err.Error() != "marshal failed" {
		t.Errorf("Marshal() error = %v, want marshal failed", err)
	}
}
