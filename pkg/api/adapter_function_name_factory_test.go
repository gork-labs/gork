package api

import (
	"testing"
)

func TestGetFunctionName_FallbackWithFactory(t *testing.T) {
	// Create a mock extractor that returns a name without dots (after slash removal)
	mockExtractor := func(i interface{}) string {
		// Simulate a function name that has no dots after package path removal
		return "functionWithoutDots"
	}
	
	// Test the fallback case using dependency injection
	result := getFunctionNameWithExtractor(func() {}, mockExtractor)
	
	// Should return the name as-is since there are no dots
	expected := "functionWithoutDots"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExtractFunctionNameFromRuntime_FallbackCase(t *testing.T) {
	// Test the original logic but with a mock that simulates the edge case
	// This is harder to test directly, but we can create a scenario
	
	// Use a function to test the normal path
	testFunc := func() {}
	result := extractFunctionNameFromRuntime(testFunc)
	
	// Should get some function name
	if result == "" {
		t.Error("Should extract non-empty function name")
	}
	
	// The key insight is that line 242 (return fullName) is reached
	// when there's no dot in the name after removing package paths
	// This is extremely rare in practice but our factory allows testing it
}