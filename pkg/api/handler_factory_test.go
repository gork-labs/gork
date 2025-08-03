package api

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// TestDefaultJSONEncoderFactory_NewEncoder tests the NewEncoder method
func TestDefaultJSONEncoderFactory_NewEncoder(t *testing.T) {
	factory := defaultJSONEncoderFactory{}
	var buf bytes.Buffer
	encoder := factory.NewEncoder(&buf)

	// Test that it creates a valid JSON encoder
	testData := map[string]string{"key": "value"}
	err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Verify the encoded data
	var decoded map[string]string
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["key"] != "value" {
		t.Errorf("Expected key=value, got %v", decoded)
	}
}

type TestRequestWithBody struct {
	Path struct {
		ID string `gork:"id"`
	}
	Body struct {
		Name string `gork:"name"`
	}
}

type TestRequestWithoutBody struct {
	Path struct {
		ID string `gork:"id"`
	}
	Query struct {
		Filter string `gork:"filter"`
	}
}

func TestValidateBodyUsageForMethod(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		reqType     reflect.Type
		shouldPanic bool
		panicMsg    string
	}{
		{
			name:        "GET with Body field should panic",
			method:      "GET",
			reqType:     reflect.TypeOf(TestRequestWithBody{}),
			shouldPanic: true,
			panicMsg:    "Handler for GET method cannot have a Body section",
		},
		{
			name:        "HEAD with Body field should panic",
			method:      "HEAD",
			reqType:     reflect.TypeOf(TestRequestWithBody{}),
			shouldPanic: true,
			panicMsg:    "Handler for HEAD method cannot have a Body section",
		},
		{
			name:        "OPTIONS with Body field should panic",
			method:      "OPTIONS",
			reqType:     reflect.TypeOf(TestRequestWithBody{}),
			shouldPanic: true,
			panicMsg:    "Handler for OPTIONS method cannot have a Body section",
		},
		{
			name:        "POST with Body field should not panic",
			method:      "POST",
			reqType:     reflect.TypeOf(TestRequestWithBody{}),
			shouldPanic: false,
		},
		{
			name:        "PUT with Body field should not panic",
			method:      "PUT",
			reqType:     reflect.TypeOf(TestRequestWithBody{}),
			shouldPanic: false,
		},
		{
			name:        "PATCH with Body field should not panic",
			method:      "PATCH",
			reqType:     reflect.TypeOf(TestRequestWithBody{}),
			shouldPanic: false,
		},
		{
			name:        "DELETE with Body field should not panic",
			method:      "DELETE",
			reqType:     reflect.TypeOf(TestRequestWithBody{}),
			shouldPanic: false,
		},
		{
			name:        "GET without Body field should not panic",
			method:      "GET",
			reqType:     reflect.TypeOf(TestRequestWithoutBody{}),
			shouldPanic: false,
		},
		{
			name:        "GET with nil request type should not panic",
			method:      "GET",
			reqType:     nil,
			shouldPanic: false,
		},
		{
			name:        "GET with non-struct type should not panic",
			method:      "GET",
			reqType:     reflect.TypeOf("string"),
			shouldPanic: false,
		},
		{
			name:        "GET with pointer to struct with Body should panic",
			method:      "GET",
			reqType:     reflect.TypeOf(&TestRequestWithBody{}),
			shouldPanic: true,
			panicMsg:    "Handler for GET method cannot have a Body section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("Expected panic but none occurred")
						return
					}
					panicMsg, ok := r.(string)
					if !ok {
						t.Errorf("Expected string panic message, got %T: %v", r, r)
						return
					}
					if tt.panicMsg != "" && len(panicMsg) > 0 && panicMsg[:len(tt.panicMsg)] != tt.panicMsg {
						t.Errorf("Expected panic message to start with %q, got %q", tt.panicMsg, panicMsg)
					}
				}()
			}

			validateBodyUsageForMethod(tt.method, tt.reqType)

			if tt.shouldPanic {
				t.Error("Expected panic but function completed normally")
			}
		})
	}
}
