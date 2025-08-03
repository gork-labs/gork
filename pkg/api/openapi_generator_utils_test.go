package api

import (
	"testing"
)

func TestStripPackagePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "type with package path",
			input:    "github.com/gork-labs/gork/pkg/api.Schema",
			expected: "Schema",
		},
		{
			name:     "type without package path",
			input:    "SimpleType",
			expected: "SimpleType",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple dots",
			input:    "package.subpackage.Type",
			expected: "Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripPackagePath(tt.input)
			if result != tt.expected {
				t.Errorf("stripPackagePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
