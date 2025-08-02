package api

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

// Test uncovered lines in adapter.go

// Test parseQueryParams with empty parameter value (line 133-134)
func TestParseQueryParams_EmptyParameterValue(t *testing.T) {
	type TestRequest struct {
		Name string `json:"name"`
	}

	// Create request with empty query parameter
	req := &http.Request{
		URL: &url.URL{
			RawQuery: "name=", // Empty value
		},
	}

	testReq := &TestRequest{}
	parseQueryParams(req, testReq)

	// The field should remain empty since the parameter value was empty
	if testReq.Name != "" {
		t.Errorf("Expected empty name, got %s", testReq.Name)
	}
}

// Test setFieldValue with float32/float64 (lines 183-185)
func TestSetFieldValue_FloatTypes(t *testing.T) {
	type TestStruct struct {
		Float32Field float32
		Float64Field float64
	}

	testStruct := TestStruct{}
	v := reflect.ValueOf(&testStruct).Elem()

	// Test float32
	field32 := v.Type().Field(0)
	fieldValue32 := v.Field(0)
	setFieldValue(fieldValue32, field32, "3.14", []string{"3.14"})

	expected32 := float32(3.14)
	if testStruct.Float32Field != expected32 {
		t.Errorf("Expected %f, got %f", expected32, testStruct.Float32Field)
	}

	// Test float64
	field64 := v.Type().Field(1)
	fieldValue64 := v.Field(1)
	setFieldValue(fieldValue64, field64, "2.718281828", []string{"2.718281828"})

	expected64 := 2.718281828
	if testStruct.Float64Field != expected64 {
		t.Errorf("Expected %f, got %f", expected64, testStruct.Float64Field)
	}
}

// Test setFieldValue with invalid float values (error case for line 183)
func TestSetFieldValue_InvalidFloatValues(t *testing.T) {
	type TestStruct struct {
		FloatField float64
	}

	testStruct := TestStruct{}
	v := reflect.ValueOf(&testStruct).Elem()

	field := v.Type().Field(0)
	fieldValue := v.Field(0)

	// Test with invalid float value - should not set the field
	setFieldValue(fieldValue, field, "not-a-float", []string{"not-a-float"})

	// Field should remain zero since parsing failed
	if testStruct.FloatField != 0 {
		t.Errorf("Expected 0, got %f", testStruct.FloatField)
	}
}

// Test the case where ParseFloat fails but doesn't crash
func TestSetFieldValue_ParseFloatError(t *testing.T) {
	type TestStruct struct {
		Float32Field float32
		Float64Field float64
	}

	testStruct := TestStruct{}
	v := reflect.ValueOf(&testStruct).Elem()

	// Test float32 with invalid value
	field32 := v.Type().Field(0)
	fieldValue32 := v.Field(0)
	setFieldValue(fieldValue32, field32, "invalid", []string{"invalid"})

	if testStruct.Float32Field != 0 {
		t.Errorf("Expected 0 for invalid float32, got %f", testStruct.Float32Field)
	}

	// Test float64 with invalid value
	field64 := v.Type().Field(1)
	fieldValue64 := v.Field(1)
	setFieldValue(fieldValue64, field64, "also-invalid", []string{"also-invalid"})

	if testStruct.Float64Field != 0 {
		t.Errorf("Expected 0 for invalid float64, got %f", testStruct.Float64Field)
	}
}

// Test uncovered lines in other files

// Test doc_extractor.go line 72.58,74.4 (processPackageFiles error handling)
func TestDocExtractor_ProcessPackageFilesErrorHandling(t *testing.T) {
	extractor := NewDocExtractor()

	// This will test the error path in processPackageFiles
	// where the file processing could encounter issues

	// processPackageFiles is internal, but we can test through ParseDirectory
	// which calls it internally. We've already tested this indirectly through
	// other tests, but let's ensure the error path is covered
	err := extractor.ParseDirectory("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

// Test resolveQueryParamName with different openapi tag scenarios
func TestResolveQueryParamName_OpenAPITag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "openapi tag with in=query",
			tag:      `openapi:"name=test_param,in=query"`,
			expected: "test_param",
		},
		{
			name:     "openapi tag with in=path (should be empty)",
			tag:      `openapi:"name=test_param,in=path"`,
			expected: "",
		},
		{
			name:     "openapi tag with in=header (should be empty)",
			tag:      `openapi:"name=test_param,in=header"`,
			expected: "",
		},
		{
			name:     "json tag fallback",
			tag:      `json:"json_name"`,
			expected: "json_name",
		},
		{
			name:     "field name fallback",
			tag:      ``,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: "FieldName",
				Tag:  reflect.StructTag(tt.tag),
			}

			result := resolveQueryParamName(field)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Test to cover line 229.2,229.17 (setFieldValue default case return)
func TestSetFieldValue_UnsupportedTypes(t *testing.T) {
	type TestStruct struct {
		UnsafePtr    uintptr
		ComplexField complex64
	}

	testStruct := TestStruct{}
	v := reflect.ValueOf(&testStruct).Elem()

	// Test uintptr (unsupported type)
	field := v.Type().Field(0)
	fieldValue := v.Field(0)

	// This should hit the default case and return early
	setFieldValue(fieldValue, field, "123", []string{"123"})

	// Value should remain zero since it's unsupported
	if testStruct.UnsafePtr != 0 {
		t.Errorf("Expected 0 for unsupported uintptr type, got %v", testStruct.UnsafePtr)
	}

	// Test complex64 (unsupported type)
	field2 := v.Type().Field(1)
	fieldValue2 := v.Field(1)

	setFieldValue(fieldValue2, field2, "1+2i", []string{"1+2i"})

	// Value should remain zero since it's unsupported
	if testStruct.ComplexField != 0 {
		t.Errorf("Expected 0 for unsupported complex64 type, got %v", testStruct.ComplexField)
	}
}

// Test to hit more uncovered lines in validation
func TestValidation_EdgeCases(t *testing.T) {
	type TestStruct struct {
		RequiredField string `validate:"required"`
		OptionalField string
	}

	// Test with valid struct
	validStruct := TestStruct{RequiredField: "valid"}
	errs := CheckDiscriminatorErrors(validStruct)
	if len(errs) != 0 {
		t.Errorf("Expected no errors for valid struct, got %d", len(errs))
	}
}
