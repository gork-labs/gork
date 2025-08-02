package api

import (
	"reflect"
	"testing"
)

func TestReflectTypeEquality_Analysis(t *testing.T) {
	// This test analyzes whether lines 24-26 in defaultRouteFilter can ever be reached

	// Get the types we're working with
	specPtrType := reflect.TypeOf(&OpenAPISpec{}) // *OpenAPISpec
	specElemType := specPtrType.Elem()            // OpenAPISpec

	// Test 1: Create a pointer type using reflect.PtrTo
	constructedPtrType := reflect.PtrTo(specElemType) // Should be *OpenAPISpec

	t.Logf("specPtrType: %v", specPtrType)
	t.Logf("constructedPtrType: %v", constructedPtrType)
	t.Logf("Are they equal? %v", specPtrType == constructedPtrType)

	// Test 2: Check element types
	t.Logf("specPtrType.Elem(): %v", specPtrType.Elem())
	t.Logf("constructedPtrType.Elem(): %v", constructedPtrType.Elem())
	t.Logf("Element types equal? %v", specPtrType.Elem() == constructedPtrType.Elem())

	// Test 3: This demonstrates the logic in lines 24-26
	// For constructedPtrType, would the condition be true?
	condition1 := constructedPtrType.Kind() == reflect.Ptr
	condition2 := constructedPtrType.Elem() == specPtrType.Elem()
	wouldTriggerLines24to26 := condition1 && condition2

	t.Logf("constructedPtrType.Kind() == reflect.Ptr: %v", condition1)
	t.Logf("constructedPtrType.Elem() == specPtrType.Elem(): %v", condition2)
	t.Logf("Would trigger lines 24-26: %v", wouldTriggerLines24to26)

	// But would it be caught by line 19 first?
	wouldBeCaughtByLine19 := constructedPtrType == specPtrType
	t.Logf("Would be caught by line 19: %v", wouldBeCaughtByLine19)

	// The hypothesis: if wouldTriggerLines24to26 is true, then wouldBeCaughtByLine19 should also be true
	if wouldTriggerLines24to26 && !wouldBeCaughtByLine19 {
		t.Error("Found a case that would trigger lines 24-26 but not line 19 - lines 24-26 are not dead code")
	} else if wouldTriggerLines24to26 && wouldBeCaughtByLine19 {
		t.Log("Lines 24-26 appear to be unreachable dead code - any case they would catch is already caught by line 19")
	}
}

func TestDeadCodeAnalysis_AlternativeTypes(t *testing.T) {
	// Test with various pointer types to see if any could bypass line 19 but trigger lines 24-26

	specPtrType := reflect.TypeOf(&OpenAPISpec{})
	specElemType := specPtrType.Elem()

	testCases := []struct {
		name     string
		testType reflect.Type
	}{
		{"Direct *OpenAPISpec", reflect.TypeOf((*OpenAPISpec)(nil))},
		{"Constructed *OpenAPISpec", reflect.PtrTo(specElemType)},
		{"Custom type alias", reflect.TypeOf((*CustomSpecAlias)(nil))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check if this type would be caught by line 19
			caughtByLine19 := tc.testType == specPtrType

			// Check if this type would trigger lines 24-26 conditions
			triggersLines24to26 := tc.testType.Kind() == reflect.Ptr && tc.testType.Elem() == specElemType

			t.Logf("Type: %v", tc.testType)
			t.Logf("Caught by line 19: %v", caughtByLine19)
			t.Logf("Would trigger lines 24-26: %v", triggersLines24to26)

			if triggersLines24to26 && !caughtByLine19 {
				t.Logf("This type would make lines 24-26 reachable!")
			}
		})
	}
}

// CustomSpecAlias is a type alias to test edge cases
type CustomSpecAlias = OpenAPISpec
