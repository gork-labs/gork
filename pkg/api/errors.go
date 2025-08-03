package api

// ErrorResponse represents a generic error response structure.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ValidationErrorResponse represents validation error responses with field-level details.
type ValidationErrorResponse struct {
	Message string              `json:"error"`
	Details map[string][]string `json:"details,omitempty"`
}

// Error implements the error interface for ValidationErrorResponse.
func (v *ValidationErrorResponse) Error() string {
	return v.Message
}
