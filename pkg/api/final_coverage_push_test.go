package api

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestBuildBasicTypeSchema_FinalCoveragePush(t *testing.T) {
	// These tests are specifically designed to hit the final uncovered lines
	
	// Test all the cases we know are covered first
	coveredCases := []struct {
		name     string
		testType reflect.Type
		expected string
	}{
		{"String", reflect.TypeOf(""), "string"},
		{"Int", reflect.TypeOf(int(0)), "integer"},
		{"Bool", reflect.TypeOf(true), "boolean"},
		{"Float64", reflect.TypeOf(float64(0)), "number"},
		{"Complex128", reflect.TypeOf(complex128(0)), "object"},
		{"Channel", reflect.TypeOf(make(chan int)), "object"},
		{"Function", reflect.TypeOf(func() {}), "object"},
		{"Interface", reflect.TypeOf((*interface{})(nil)).Elem(), "object"},
		{"Map", reflect.TypeOf(map[string]int{}), "object"},
		{"Slice (fallthrough)", reflect.TypeOf([]int{}), "object"},
		{"Struct (fallthrough)", reflect.TypeOf(struct{}{}), "object"},
		{"Pointer (fallthrough)", reflect.TypeOf((*int)(nil)), "object"},
	}
	
	for _, tc := range coveredCases {
		t.Run(tc.name, func(t *testing.T) {
			schema := buildBasicTypeSchema(tc.testType)
			if schema.Type != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, schema.Type)
			}
		})
	}
	
	// Try to create an Invalid type for line 455-457 coverage
	// The challenge is that reflect.Invalid is very hard to create
	t.Run("Invalid_type_attempt", func(t *testing.T) {
		// One approach: try to create a Value and get its Type when it's invalid
		var zeroValue reflect.Value // This creates an invalid Value
		if !zeroValue.IsValid() {
			// But zeroValue.Type() would panic, so we can't use this approach
			t.Skip("Cannot safely create Invalid reflect.Type for testing")
		}
	})
	
	// Test UnsafePointer explicitly
	t.Run("UnsafePointer", func(t *testing.T) {
		var unsafe unsafe.Pointer
		unsafeType := reflect.TypeOf(unsafe)
		schema := buildBasicTypeSchema(unsafeType)
		if schema.Type != "object" {
			t.Errorf("Expected object for UnsafePointer, got %s", schema.Type)
		}
		if schema.Description != "Unsafe pointer" {
			t.Errorf("Expected 'Unsafe pointer' description, got %s", schema.Description)
		}
	})
}