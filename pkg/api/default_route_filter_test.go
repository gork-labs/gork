package api

import (
	"reflect"
	"testing"
)

func TestDefaultRouteFilter_PointerToOpenAPISpec(t *testing.T) {
	// Test the specific case where ResponseType is a pointer to OpenAPISpec
	// This should trigger lines 24-26 in openapi_generator.go

	// The condition is: info.ResponseType.Kind() == reflect.Ptr && info.ResponseType.Elem() == specPtrType.Elem()
	// We need a type that is Ptr and whose Elem() equals OpenAPISpec (not *OpenAPISpec)

	// Create a RouteInfo with ResponseType as *OpenAPISpec
	route := &RouteInfo{
		ResponseType: reflect.TypeOf((*OpenAPISpec)(nil)),
	}

	result := defaultRouteFilter(route)

	// This should return false because it's a pointer to OpenAPISpec
	if result {
		t.Error("Expected defaultRouteFilter to return false for *OpenAPISpec")
	}
}

func TestDefaultRouteFilter_ComprehensivePointerCases(t *testing.T) {
	testCases := []struct {
		name         string
		responseType reflect.Type
		expected     bool
		description  string
	}{
		{
			name:         "direct_openapi_spec_pointer",
			responseType: reflect.TypeOf((*OpenAPISpec)(nil)),
			expected:     false,
			description:  "Direct pointer to OpenAPISpec should be filtered out",
		},
		{
			name:         "pointer_to_openapi_spec_pointer",
			responseType: reflect.TypeOf((**OpenAPISpec)(nil)),
			expected:     true,
			description:  "Pointer to pointer to OpenAPISpec should NOT match the condition (element is *OpenAPISpec, not OpenAPISpec)",
		},
		{
			name:         "normal_struct_pointer",
			responseType: reflect.TypeOf((*struct{ Message string })(nil)),
			expected:     true,
			description:  "Normal struct pointer should pass through",
		},
		{
			name:         "nil_response_type",
			responseType: nil,
			expected:     true,
			description:  "Nil response type should pass through",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			route := &RouteInfo{
				ResponseType: tc.responseType,
			}

			result := defaultRouteFilter(route)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for %s", tc.expected, result, tc.description)
			}
		})
	}
}

func TestDefaultRouteFilter_Lines24to26Coverage(t *testing.T) {
	// This test is specifically designed to trigger lines 24-26 in openapi_generator.go
	// The condition is: info.ResponseType.Kind() == reflect.Ptr && info.ResponseType.Elem() == specPtrType.Elem()

	// Let's create a mock RouteInfo that bypasses the line 19 check but triggers lines 24-26
	// We need a pointer type whose element matches OpenAPISpec but isn't exactly *OpenAPISpec

	// Actually, let's verify if lines 24-26 are even reachable
	// by creating the exact scenario they're designed to catch

	specType := reflect.TypeOf(OpenAPISpec{})
	pointerToSpec := reflect.PtrTo(specType)

	// This should be identical to *OpenAPISpec, so line 19 should catch it
	route1 := &RouteInfo{ResponseType: pointerToSpec}
	result1 := defaultRouteFilter(route1)
	if result1 {
		t.Error("Expected false for constructed *OpenAPISpec")
	}

	// Now test the case that might reach lines 24-26:
	// What if we create a type that has the same element but different identity?
	// In Go's type system, this shouldn't be possible, but let's test edge cases

	// Test with a custom type that embeds OpenAPISpec
	type CustomOpenAPISpec OpenAPISpec
	customType := reflect.TypeOf((*CustomOpenAPISpec)(nil))

	route2 := &RouteInfo{ResponseType: customType}
	result2 := defaultRouteFilter(route2)
	// This should return true (not filtered) because it's not exactly *OpenAPISpec
	if !result2 {
		t.Error("Expected true for *CustomOpenAPISpec (should not be filtered)")
	}
}
