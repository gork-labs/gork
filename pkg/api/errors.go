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

// HTTPError is an error that a handler returns to send an HTTP status.
// Gork writes the body {"error": Message}. For a 5xx status, the body
// contains the status text instead of Message.
type HTTPError struct {
	Status  int
	Message string
}

// NewHTTPError returns an HTTPError with the status and the message.
func NewHTTPError(status int, message string) *HTTPError {
	return &HTTPError{Status: status, Message: message}
}

// Error implements the error interface for HTTPError.
func (e *HTTPError) Error() string {
	return e.Message
}
