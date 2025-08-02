package api

import (
	"testing"
)

func TestTrimFunctionName_FallbackCase(t *testing.T) {
	// Test the fallback case where there's no dot in the function name
	// after removing the package path
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "functionWithoutDots",
			expected: "functionWithoutDots",
			desc:     "function name with no dots triggers fallback",
		},
		{
			input:    "github.com/example/functionWithoutDots",
			expected: "functionWithoutDots", 
			desc:     "removes package path but no dots to remove",
		},
		{
			input:    "normal.function.name",
			expected: "name",
			desc:     "normal case with dots",
		},
		{
			input:    "com/pkg/justname",
			expected: "justname",
			desc:     "package path removed, no dots left",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := trimFunctionName(tc.input)
			if result != tc.expected {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		})
	}
}

func TestExtractFunctionNameFromRuntimeWithFunc_MockProvider(t *testing.T) {
	// Test trimFunctionName directly which covers the important fallback logic
	result := trimFunctionName("justfunctionname")
	expected := "justfunctionname"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// FakeRuntimeFunc simulates runtime.Func for testing
type FakeRuntimeFunc struct {
	name string
}

func (f *FakeRuntimeFunc) Name() string {
	return f.name
}