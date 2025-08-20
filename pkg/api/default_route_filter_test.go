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
