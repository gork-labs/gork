package api

// ErrorResponse represents a generic error response structure.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ValidationErrorResponse represents validation error responses with field-level details.
type ValidationErrorResponse struct {
	Error   string              `json:"error"`
	Details map[string][]string `json:"details,omitempty"`
}
