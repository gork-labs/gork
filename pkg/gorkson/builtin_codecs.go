package gorkson

import (
	"context"
	"strconv"
	"time"
)

// ptr is a helper function to create pointers to values.
func ptr[T any](v T) *T {
	return &v
}

// TimeCodec handles RFC3339 parsing/formatting with OpenAPI schema.
// It is the registered codec for time.Time unless RegisterCodec replaces it.
type TimeCodec struct{}

// Parse converts an RFC3339 formatted string, with or without fractional
// seconds, to a time.Time.
func (c TimeCodec) Parse(_ context.Context, value string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, NewParseError("time.Time", value, err)
	}
	return &t, nil
}

// Format converts a time.Time to an RFC3339 string with the fractional
// seconds of the value, so that Parse gives the same instant back.
func (c TimeCodec) Format(_ context.Context, value *time.Time) (string, error) {
	if value == nil {
		return "", nil
	}
	return value.Format(time.RFC3339Nano), nil
}

// Schema returns the OpenAPI schema for RFC3339 time representation.
func (c TimeCodec) Schema() OpenAPISchema {
	return OpenAPISchema{
		Type:   OpenAPITypeString,
		Format: OpenAPIFormatDateTime,
	}
}

// UnixTimeCodec handles Unix timestamp parsing/formatting.
type UnixTimeCodec struct{}

// Parse converts a Unix timestamp string to a time.Time.
func (c UnixTimeCodec) Parse(_ context.Context, value string) (*time.Time, error) {
	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, NewParseError("time.Time", value, err)
	}
	t := time.Unix(timestamp, 0)
	return &t, nil
}

// Format converts a time.Time to Unix timestamp string.
func (c UnixTimeCodec) Format(_ context.Context, value *time.Time) (string, error) {
	if value == nil {
		return "", nil
	}
	return strconv.FormatInt(value.Unix(), 10), nil
}

// Schema returns the OpenAPI schema for Unix timestamp representation.
func (c UnixTimeCodec) Schema() OpenAPISchema {
	return OpenAPISchema{
		Type:        OpenAPITypeInteger,
		Format:      "int64",
		Example:     1703505000,
		Description: "Unix timestamp (seconds since epoch)",
		Minimum:     ptr(0.0), // No negative timestamps
	}
}
