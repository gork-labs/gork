package gorkson

import (
	"fmt"
)

// ParseError represents an error that occurred during parsing.
type ParseError struct {
	Type     string // The type being parsed (e.g., "time.Time", "Priority")
	Value    string // The input value that failed to parse
	BaseErr  error  // The underlying error
	Reason   string // Human-readable reason (optional)
	HTTPCode int    // Suggested HTTP status code (400 for client errors)
}

// Error returns the error message.
func (e *ParseError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("failed to parse %s from %q: %s", e.Type, e.Value, e.Reason)
	}
	if e.BaseErr != nil {
		return fmt.Sprintf("failed to parse %s from %q: %v", e.Type, e.Value, e.BaseErr)
	}
	return fmt.Sprintf("failed to parse %s from %q", e.Type, e.Value)
}

// Unwrap returns the underlying error for error chain compatibility.
func (e *ParseError) Unwrap() error {
	return e.BaseErr
}

// NewParseError creates a new ParseError with a base error.
func NewParseError(typeName, value string, err error) *ParseError {
	return &ParseError{
		Type:     typeName,
		Value:    value,
		BaseErr:  err,
		HTTPCode: 400, // Bad Request
	}
}

// NewParseErrorWithReason creates a new ParseError with a custom reason.
func NewParseErrorWithReason(typeName, value, reason string) *ParseError {
	return &ParseError{
		Type:     typeName,
		Value:    value,
		Reason:   reason,
		HTTPCode: 400, // Bad Request
	}
}

// FormatError represents an error that occurred during formatting.
type FormatError struct {
	Type    string // The type being formatted (e.g., "time.Time", "Priority")
	Reason  string // Human-readable reason
	BaseErr error  // The underlying error (optional)
}

// Error returns the error message.
func (e *FormatError) Error() string {
	if e.BaseErr != nil {
		return fmt.Sprintf("failed to format %s: %s (%v)", e.Type, e.Reason, e.BaseErr)
	}
	return fmt.Sprintf("failed to format %s: %s", e.Type, e.Reason)
}

// Unwrap returns the underlying error for error chain compatibility.
func (e *FormatError) Unwrap() error {
	return e.BaseErr
}

// NewFormatError creates a new FormatError.
func NewFormatError(typeName, reason string) *FormatError {
	return &FormatError{
		Type:   typeName,
		Reason: reason,
	}
}

// NewFormatErrorWithBase creates a new FormatError with a base error.
func NewFormatErrorWithBase(typeName, reason string, err error) *FormatError {
	return &FormatError{
		Type:    typeName,
		Reason:  reason,
		BaseErr: err,
	}
}
