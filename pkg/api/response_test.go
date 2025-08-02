package api

import (
	"encoding/json"
	"testing"
)

func TestNoContentResponse(t *testing.T) {
	// Test NoContentResponse structure
	response := NoContentResponse{}

	// Should marshal to empty JSON object
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal NoContentResponse: %v", err)
	}

	expectedJSON := "{}"
	if string(data) != expectedJSON {
		t.Errorf("NoContentResponse JSON = %q, want %q", string(data), expectedJSON)
	}

	// Should unmarshal from empty JSON object
	var unmarshaled NoContentResponse
	err = json.Unmarshal([]byte("{}"), &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal NoContentResponse: %v", err)
	}

	// Should be equal to original (both are empty structs)
	if unmarshaled != response {
		t.Error("Unmarshaled NoContentResponse does not match original")
	}
}

func TestNoContentResponse_Usage(t *testing.T) {
	// Test that NoContentResponse can be used as a return type
	var response interface{} = NoContentResponse{}

	if _, ok := response.(NoContentResponse); !ok {
		t.Error("NoContentResponse type assertion failed")
	}

	// Test zero value
	var zeroResponse NoContentResponse
	data, err := json.Marshal(zeroResponse)
	if err != nil {
		t.Fatalf("Failed to marshal zero NoContentResponse: %v", err)
	}

	if string(data) != "{}" {
		t.Errorf("Zero NoContentResponse JSON = %q, want {}", string(data))
	}
}
