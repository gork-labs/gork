package gorkson

import "testing"

func TestUnmarshalIntoSlice(t *testing.T) {
	var users []SimpleStruct
	if err := Unmarshal([]byte(`[{"name":"Alice","age":25}]`), &users); err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Name != "Alice" || users[0].Age != 25 {
		t.Errorf("Unmarshal() = %+v, want Alice aged 25", users)
	}

	if err := Unmarshal([]byte(`{"name":"Alice"}`), &users); err == nil {
		t.Error("expected an error for an object")
	}
}
