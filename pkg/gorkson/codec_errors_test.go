package gorkson

import (
	"errors"
	"testing"
)

func TestParseError(t *testing.T) {
	baseErr := errors.New("base error")

	tests := []struct {
		name        string
		parseError  *ParseError
		expectedMsg string
	}{
		{
			name: "with base error",
			parseError: &ParseError{
				Type:    "time.Time",
				Value:   "invalid-date",
				BaseErr: baseErr,
			},
			expectedMsg: `failed to parse time.Time from "invalid-date": base error`,
		},
		{
			name: "with reason",
			parseError: &ParseError{
				Type:   "Priority",
				Value:  "invalid",
				Reason: "must be one of: low, medium, high",
			},
			expectedMsg: `failed to parse Priority from "invalid": must be one of: low, medium, high`,
		},
		{
			name: "minimal error",
			parseError: &ParseError{
				Type:  "CustomType",
				Value: "test",
			},
			expectedMsg: `failed to parse CustomType from "test"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.parseError.Error() != tt.expectedMsg {
				t.Errorf("ParseError.Error() = %q, want %q", tt.parseError.Error(), tt.expectedMsg)
			}
		})
	}
}

func TestParseError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	parseErr := &ParseError{
		Type:    "time.Time",
		Value:   "invalid",
		BaseErr: baseErr,
	}

	if parseErr.Unwrap() != baseErr {
		t.Errorf("ParseError.Unwrap() = %v, want %v", parseErr.Unwrap(), baseErr)
	}

	// Test nil base error
	parseErr.BaseErr = nil
	if parseErr.Unwrap() != nil {
		t.Errorf("ParseError.Unwrap() = %v, want nil", parseErr.Unwrap())
	}
}

func TestNewParseError(t *testing.T) {
	baseErr := errors.New("base error")
	parseErr := NewParseError("time.Time", "invalid-date", baseErr)

	if parseErr.Type != "time.Time" {
		t.Errorf("NewParseError() Type = %q, want %q", parseErr.Type, "time.Time")
	}

	if parseErr.Value != "invalid-date" {
		t.Errorf("NewParseError() Value = %q, want %q", parseErr.Value, "invalid-date")
	}

	if parseErr.BaseErr != baseErr {
		t.Errorf("NewParseError() BaseErr = %v, want %v", parseErr.BaseErr, baseErr)
	}
}

func TestNewParseErrorWithReason(t *testing.T) {
	parseErr := NewParseErrorWithReason("Priority", "invalid", "must be one of: low, medium, high")

	if parseErr.Type != "Priority" {
		t.Errorf("NewParseErrorWithReason() Type = %q, want %q", parseErr.Type, "Priority")
	}

	if parseErr.Value != "invalid" {
		t.Errorf("NewParseErrorWithReason() Value = %q, want %q", parseErr.Value, "invalid")
	}

	if parseErr.Reason != "must be one of: low, medium, high" {
		t.Errorf("NewParseErrorWithReason() Reason = %q, want %q", parseErr.Reason, "must be one of: low, medium, high")
	}
}

func TestFormatError(t *testing.T) {
	baseErr := errors.New("base error")

	tests := []struct {
		name        string
		formatError *FormatError
		expectedMsg string
	}{
		{
			name: "with base error",
			formatError: &FormatError{
				Type:    "time.Time",
				Reason:  "nil value",
				BaseErr: baseErr,
			},
			expectedMsg: "failed to format time.Time: nil value (base error)",
		},
		{
			name: "without base error",
			formatError: &FormatError{
				Type:   "Priority",
				Reason: "invalid enum value",
			},
			expectedMsg: "failed to format Priority: invalid enum value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.formatError.Error() != tt.expectedMsg {
				t.Errorf("FormatError.Error() = %q, want %q", tt.formatError.Error(), tt.expectedMsg)
			}
		})
	}
}

func TestFormatError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	formatErr := &FormatError{
		Type:    "time.Time",
		Reason:  "nil value",
		BaseErr: baseErr,
	}

	if formatErr.Unwrap() != baseErr {
		t.Errorf("FormatError.Unwrap() = %v, want %v", formatErr.Unwrap(), baseErr)
	}

	// Test nil base error
	formatErr.BaseErr = nil
	if formatErr.Unwrap() != nil {
		t.Errorf("FormatError.Unwrap() = %v, want nil", formatErr.Unwrap())
	}
}

func TestNewFormatError(t *testing.T) {
	formatErr := NewFormatError("time.Time", "nil value")

	if formatErr.Type != "time.Time" {
		t.Errorf("NewFormatError() Type = %q, want %q", formatErr.Type, "time.Time")
	}

	if formatErr.Reason != "nil value" {
		t.Errorf("NewFormatError() Reason = %q, want %q", formatErr.Reason, "nil value")
	}
}

func TestNewFormatErrorWithBase(t *testing.T) {
	baseErr := errors.New("base error")
	formatErr := NewFormatErrorWithBase("time.Time", "conversion failed", baseErr)

	if formatErr.Type != "time.Time" {
		t.Errorf("NewFormatErrorWithBase() Type = %q, want %q", formatErr.Type, "time.Time")
	}

	if formatErr.Reason != "conversion failed" {
		t.Errorf("NewFormatErrorWithBase() Reason = %q, want %q", formatErr.Reason, "conversion failed")
	}

	if formatErr.BaseErr != baseErr {
		t.Errorf("NewFormatErrorWithBase() BaseErr = %v, want %v", formatErr.BaseErr, baseErr)
	}
}
