package api

import (
	"reflect"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

func TestIsUnionType(t *testing.T) {
	t.Run("nil type", func(t *testing.T) {
		// Test the uncovered nil check
		result := isUnionType(nil)
		if result {
			t.Error("Expected isUnionType(nil) to return false")
		}
	})

	t.Run("valid union type", func(t *testing.T) {
		unionType := reflect.TypeOf(unions.Union2[string, int]{})
		result := isUnionType(unionType)
		if !result {
			t.Error("Expected Union2[string, int] to be recognized as union type")
		}
	})

	t.Run("pointer to union type", func(t *testing.T) {
		unionType := reflect.TypeOf(&unions.Union2[string, int]{})
		result := isUnionType(unionType)
		if !result {
			t.Error("Expected pointer to Union2[string, int] to be recognized as union type")
		}
	})

	t.Run("non-union struct", func(t *testing.T) {
		type RegularStruct struct {
			Field string
		}
		structType := reflect.TypeOf(RegularStruct{})
		result := isUnionType(structType)
		if result {
			t.Error("Expected regular struct to not be recognized as union type")
		}
	})

	t.Run("basic type", func(t *testing.T) {
		stringType := reflect.TypeOf("")
		result := isUnionType(stringType)
		if result {
			t.Error("Expected string type to not be recognized as union type")
		}
	})

	t.Run("union type from different package", func(t *testing.T) {
		// This would be a struct that looks like a union but isn't from unions package
		type FakeUnion2 struct {
			Field1 *string
			Field2 *int
		}
		fakeType := reflect.TypeOf(FakeUnion2{})
		result := isUnionType(fakeType)
		if result {
			t.Error("Expected fake union type to not be recognized as union type")
		}
	})

	t.Run("union3 type", func(t *testing.T) {
		unionType := reflect.TypeOf(unions.Union3[string, int, bool]{})
		result := isUnionType(unionType)
		if !result {
			t.Error("Expected Union3[string, int, bool] to be recognized as union type")
		}
	})

	t.Run("union4 type", func(t *testing.T) {
		unionType := reflect.TypeOf(unions.Union4[string, int, bool, float64]{})
		result := isUnionType(unionType)
		if !result {
			t.Error("Expected Union4[string, int, bool, float64] to be recognized as union type")
		}
	})
}
