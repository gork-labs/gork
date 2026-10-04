package gorkson

import (
	"context"
	"testing"
	"time"
)

func TestTimeCodec_Parse(t *testing.T) {
	codec := TimeCodec{}
	ctx := context.Background()

	tests := []struct {
		name      string
		value     string
		wantTime  *time.Time
		wantErr   bool
		checkTime bool
	}{
		{
			name:      "valid RFC3339 time",
			value:     "2023-12-25T10:30:00Z",
			checkTime: true,
			wantErr:   false,
		},
		{
			name:      "valid RFC3339 time with milliseconds",
			value:     "2023-12-25T10:30:00.123Z",
			checkTime: true,
			wantErr:   false,
		},
		{
			name:    "invalid time format",
			value:   "2023-12-25",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			value:   "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.Parse(ctx, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("TimeCodec.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if result == nil {
				t.Error("TimeCodec.Parse() returned nil result")
				return
			}

			if tt.checkTime {
				expected, parseErr := time.Parse(time.RFC3339, tt.value)
				if parseErr != nil {
					t.Fatalf("Test setup error: %v", parseErr)
				}
				if !result.Equal(expected) {
					t.Errorf("TimeCodec.Parse() = %v, want %v", *result, expected)
				}
			}
		})
	}
}

func TestTimeCodec_Format(t *testing.T) {
	codec := TimeCodec{}
	ctx := context.Background()

	testTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
	expected := "2023-12-25T10:30:00Z"

	tests := []struct {
		name     string
		value    *time.Time
		expected string
		wantErr  bool
	}{
		{
			name:     "valid time",
			value:    &testTime,
			expected: expected,
			wantErr:  false,
		},
		{
			name:     "nil time",
			value:    nil,
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.Format(ctx, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("TimeCodec.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("TimeCodec.Format() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTimeCodec_Schema(t *testing.T) {
	codec := TimeCodec{}
	schema := codec.Schema()

	if schema.Type != OpenAPITypeString {
		t.Errorf("TimeCodec.Schema() Type = %v, want %v", schema.Type, OpenAPITypeString)
	}

	if schema.Format != OpenAPIFormatDateTime {
		t.Errorf("TimeCodec.Schema() Format = %v, want %v", schema.Format, OpenAPIFormatDateTime)
	}

	if schema.Pattern != "" || schema.Example != nil || schema.Description != "" {
		t.Errorf("TimeCodec.Schema() = %+v, want only type and format", schema)
	}
}

func TestUnixTimeCodec_Parse(t *testing.T) {
	codec := UnixTimeCodec{}
	ctx := context.Background()

	tests := []struct {
		name         string
		value        string
		expectedUnix int64
		wantErr      bool
	}{
		{
			name:         "valid unix timestamp",
			value:        "1703505000",
			expectedUnix: 1703505000,
			wantErr:      false,
		},
		{
			name:         "zero timestamp",
			value:        "0",
			expectedUnix: 0,
			wantErr:      false,
		},
		{
			name:    "invalid format",
			value:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "float number",
			value:   "123.456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.Parse(ctx, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnixTimeCodec.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if result == nil {
				t.Error("UnixTimeCodec.Parse() returned nil result")
				return
			}

			if result.Unix() != tt.expectedUnix {
				t.Errorf("UnixTimeCodec.Parse() Unix() = %v, want %v", result.Unix(), tt.expectedUnix)
			}
		})
	}
}

func TestUnixTimeCodec_Format(t *testing.T) {
	codec := UnixTimeCodec{}
	ctx := context.Background()

	testTime := time.Unix(1703505000, 0)
	expected := "1703505000"

	tests := []struct {
		name     string
		value    *time.Time
		expected string
		wantErr  bool
	}{
		{
			name:     "valid time",
			value:    &testTime,
			expected: expected,
			wantErr:  false,
		},
		{
			name:     "nil time",
			value:    nil,
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.Format(ctx, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnixTimeCodec.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("UnixTimeCodec.Format() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUnixTimeCodec_Schema(t *testing.T) {
	codec := UnixTimeCodec{}
	schema := codec.Schema()

	if schema.Type != OpenAPITypeInteger {
		t.Errorf("UnixTimeCodec.Schema() Type = %v, want %v", schema.Type, OpenAPITypeInteger)
	}

	if schema.Format != "int64" {
		t.Errorf("UnixTimeCodec.Schema() Format = %v, want %v", schema.Format, "int64")
	}

	if schema.Example == nil {
		t.Error("UnixTimeCodec.Schema() Example should not be nil")
	}

	if schema.Description == "" {
		t.Error("UnixTimeCodec.Schema() Description should not be empty")
	}

	if schema.Minimum == nil || *schema.Minimum != 0.0 {
		t.Error("UnixTimeCodec.Schema() should have Minimum = 0")
	}
}
