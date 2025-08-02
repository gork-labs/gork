package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorResponse(t *testing.T) {
	// Test ErrorResponse structure
	err := ErrorResponse{
		Error: "Something went wrong",
		Details: map[string]interface{}{
			"field1": "value1",
			"field2": 42,
		},
	}

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal ErrorResponse: %v", jsonErr)
	}

	var unmarshaled ErrorResponse
	if jsonErr := json.Unmarshal(data, &unmarshaled); jsonErr != nil {
		t.Fatalf("Failed to unmarshal ErrorResponse: %v", jsonErr)
	}

	if unmarshaled.Error != "Something went wrong" {
		t.Errorf("Error = %q, want 'Something went wrong'", unmarshaled.Error)
	}

	if len(unmarshaled.Details) != 2 {
		t.Errorf("Details length = %d, want 2", len(unmarshaled.Details))
	}

	if unmarshaled.Details["field1"] != "value1" {
		t.Errorf("Details[field1] = %v, want 'value1'", unmarshaled.Details["field1"])
	}
}

func TestValidationErrorResponse(t *testing.T) {
	// Test ValidationErrorResponse structure
	err := ValidationErrorResponse{
		Error: "Validation failed",
		Details: map[string][]string{
			"name":  {"required", "min"},
			"email": {"email"},
		},
	}

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal ValidationErrorResponse: %v", jsonErr)
	}

	var unmarshaled ValidationErrorResponse
	if jsonErr := json.Unmarshal(data, &unmarshaled); jsonErr != nil {
		t.Fatalf("Failed to unmarshal ValidationErrorResponse: %v", jsonErr)
	}

	if unmarshaled.Error != "Validation failed" {
		t.Errorf("Error = %q, want 'Validation failed'", unmarshaled.Error)
	}

	if len(unmarshaled.Details) != 2 {
		t.Errorf("Details length = %d, want 2", len(unmarshaled.Details))
	}

	nameErrors := unmarshaled.Details["name"]
	if len(nameErrors) != 2 {
		t.Errorf("Name errors length = %d, want 2", len(nameErrors))
	}

	expectedNameErrors := []string{"required", "min"}
	for i, expected := range expectedNameErrors {
		if nameErrors[i] != expected {
			t.Errorf("Name errors[%d] = %q, want %q", i, nameErrors[i], expected)
		}
	}

	emailErrors := unmarshaled.Details["email"]
	if len(emailErrors) != 1 || emailErrors[0] != "email" {
		t.Errorf("Email errors = %v, want [email]", emailErrors)
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		message        string
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:           "client error 400",
			code:           http.StatusBadRequest,
			message:        "Invalid request",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]string{"error": "Invalid request"},
		},
		{
			name:           "client error 404",
			code:           http.StatusNotFound,
			message:        "Resource not found",
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"error": "Resource not found"},
		},
		{
			name:           "server error 500 - hides internal details",
			code:           http.StatusInternalServerError,
			message:        "database connection failed",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]string{"error": "Internal Server Error"},
		},
		{
			name:           "server error 502 - hides internal details",
			code:           http.StatusBadGateway,
			message:        "upstream service error details",
			expectedStatus: http.StatusBadGateway,
			expectedBody:   map[string]string{"error": "Bad Gateway"},
		},
		{
			name:           "validation error 422",
			code:           http.StatusUnprocessableEntity,
			message:        "Validation failed on field X",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"error": "Validation failed on field X"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			writeError(rr, tt.code, tt.message)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rr.Code, tt.expectedStatus)
			}

			// Check content type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}

			// Check response body
			var response map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response["error"] != tt.expectedBody["error"] {
				t.Errorf("Response error = %q, want %q", response["error"], tt.expectedBody["error"])
			}
		})
	}
}

func TestWriteError_ContentTypeAndBody(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusBadRequest, "test error")

	// Verify Content-Type header is set
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	// Verify the response body structure
	var errorResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorResp["error"] != "test error" {
		t.Errorf("Error message = %q, want 'test error'", errorResp["error"])
	}
}

func TestWriteError_ServerErrorLogging(t *testing.T) {
	// This test verifies that server errors (5xx) are handled properly
	// The actual logging behavior requires log output capture which is complex,
	// but we can verify the client-facing behavior
	rr := httptest.NewRecorder()

	// Server error should not expose internal details
	writeError(rr, http.StatusInternalServerError, "sensitive database error")

	var errorResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	// Should return generic message, not the sensitive one
	if errorResp["error"] != "Internal Server Error" {
		t.Errorf("Error message = %q, want 'Internal Server Error'", errorResp["error"])
	}

	// Should still set proper status code
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestErrorResponseOmitEmptyDetails(t *testing.T) {
	// Test that empty Details are omitted from JSON
	err := ErrorResponse{
		Error: "Simple error",
		// Details is nil/empty
	}

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal ErrorResponse: %v", jsonErr)
	}

	// Should not contain "details" field when empty
	jsonStr := string(data)
	if containsDetailsField := json.Valid([]byte(jsonStr)); !containsDetailsField {
		t.Fatal("Invalid JSON generated")
	}

	// Parse back to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if _, hasDetails := result["details"]; hasDetails {
		t.Error("Expected 'details' field to be omitted when empty")
	}

	if result["error"] != "Simple error" {
		t.Errorf("Error = %q, want 'Simple error'", result["error"])
	}
}

func TestValidationErrorResponseOmitEmptyDetails(t *testing.T) {
	// Test that empty Details are omitted from JSON
	err := ValidationErrorResponse{
		Error: "Validation error without details",
		// Details is nil/empty
	}

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal ValidationErrorResponse: %v", jsonErr)
	}

	// Parse back to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if _, hasDetails := result["details"]; hasDetails {
		t.Error("Expected 'details' field to be omitted when empty")
	}

	if result["error"] != "Validation error without details" {
		t.Errorf("Error = %q, want 'Validation error without details'", result["error"])
	}
}
