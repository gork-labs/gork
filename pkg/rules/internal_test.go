package rules

import (
	"reflect"
	"testing"
)

func TestFieldByName_WithPointerType(t *testing.T) {
	type X struct{ A int }
	idx, f, ok := fieldByName(reflect.TypeOf(&X{}), "A")
	if !ok || idx != 0 || f.Name != "A" {
		t.Fatalf("expected to find field A via pointer type, got ok=%v idx=%d name=%s", ok, idx, f.Name)
	}
}
