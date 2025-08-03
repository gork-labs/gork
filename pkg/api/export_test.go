package api

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSetExportConfig(t *testing.T) {
	// Save original config
	originalConfig := exportConfig
	defer func() {
		exportConfig = originalConfig
	}()

	// Create test config
	var output bytes.Buffer
	var logMessages []string

	testConfig := ExportConfig{
		Output: &output,
		ExitFunc: func(code int) {
			// Test function
		},
		LogFatalf: func(format string, args ...interface{}) {
			logMessages = append(logMessages, format)
		},
	}

	// Test SetExportConfig
	SetExportConfig(testConfig)

	// Verify config was set
	if exportConfig.Output != testConfig.Output {
		t.Error("Output was not set correctly")
	}
	if exportConfig.ExitFunc == nil {
		t.Error("ExitFunc was not set correctly")
	}
	if exportConfig.LogFatalf == nil {
		t.Error("LogFatalf was not set correctly")
	}
}

func TestExportOpenAPISpec(t *testing.T) {
	// Convention Over Configuration test types
	type ExportReq struct {
		Body struct {
			Name string `gork:"name"`
		}
	}
	type ExportResp struct {
		Body struct {
			Message string `gork:"message"`
		}
	}

	handler := func(ctx context.Context, req ExportReq) (*ExportResp, error) {
		return &ExportResp{
			Body: struct {
				Message string `gork:"message"`
			}{
				Message: "Hello " + req.Body.Name,
			},
		}, nil
	}

	// Create test registry with proper RouteInfo
	registry := NewRouteRegistry()
	registry.Register(&RouteInfo{
		Method:       "GET",
		Path:         "/test",
		Handler:      handler,
		HandlerName:  "handler",
		RequestType:  reflect.TypeOf(ExportReq{}),
		ResponseType: reflect.TypeOf((*ExportResp)(nil)),
		Options:      &HandlerOption{},
	})

	// Create test config
	var output bytes.Buffer
	var logMessages []string

	config := ExportConfig{
		Output: &output,
		ExitFunc: func(code int) {
			// Should not be called in this test
			t.Error("ExitFunc should not be called in exportOpenAPISpec")
		},
		LogFatalf: func(format string, args ...interface{}) {
			logMessages = append(logMessages, format)
		},
	}

	// Test successful export
	err := exportOpenAPISpec(registry, config)
	if err != nil {
		t.Errorf("exportOpenAPISpec failed: %v", err)
	}

	// Verify output contains OpenAPI JSON
	outputStr := output.String()
	if !strings.Contains(outputStr, `"openapi"`) {
		t.Error("Output does not contain OpenAPI specification")
	}
	if !strings.Contains(outputStr, `"paths"`) {
		t.Error("Output does not contain paths")
	}
}

func TestExportOpenAPISpecWithOptions(t *testing.T) {
	// Create test registry
	registry := NewRouteRegistry()

	// Create test config
	var output bytes.Buffer
	config := ExportConfig{
		Output:    &output,
		ExitFunc:  func(code int) {},
		LogFatalf: func(format string, args ...interface{}) {},
	}

	// Test with custom options
	opts := []OpenAPIOption{
		WithTitle("Test API"),
		WithVersion("2.0.0"),
	}

	err := exportOpenAPISpec(registry, config, opts...)
	if err != nil {
		t.Errorf("exportOpenAPISpec with options failed: %v", err)
	}

	// Verify custom title and version in output
	outputStr := output.String()
	if !strings.Contains(outputStr, `"Test API"`) {
		t.Error("Output does not contain custom title")
	}
	if !strings.Contains(outputStr, `"2.0.0"`) {
		t.Error("Output does not contain custom version")
	}
}

func TestTypedRouterExportOpenAPIAndExit(t *testing.T) {
	// Create test router
	registry := NewRouteRegistry()
	router := &TypedRouter[any]{
		registry: registry,
	}

	// Save original config
	originalConfig := exportConfig
	defer func() {
		exportConfig = originalConfig
	}()

	// Set up test config
	var output bytes.Buffer
	var exitCalled bool
	var exitCode int

	testConfig := ExportConfig{
		Output: &output,
		ExitFunc: func(code int) {
			exitCalled = true
			exitCode = code
		},
		LogFatalf: func(format string, args ...interface{}) {
			// Should not be called in successful case
			t.Errorf("LogFatalf called unexpectedly: %s", format)
		},
	}

	SetExportConfig(testConfig)

	// Test ExportOpenAPIAndExit
	router.ExportOpenAPIAndExit()

	// Verify exit was called with code 0
	if !exitCalled {
		t.Error("ExitFunc was not called")
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Verify OpenAPI spec was written
	outputStr := output.String()
	if !strings.Contains(outputStr, `"openapi"`) {
		t.Error("Output does not contain OpenAPI specification")
	}
}

func TestDefaultExportConfig(t *testing.T) {
	config := DefaultExportConfig()

	if config.Output == nil {
		t.Error("Default output should not be nil")
	}
	if config.ExitFunc == nil {
		t.Error("Default ExitFunc should not be nil")
	}
	if config.LogFatalf == nil {
		t.Error("Default LogFatalf should not be nil")
	}
}

// Test error path in exportOpenAPISpec
func TestExportOpenAPISpecError(t *testing.T) {
	// Create a writer that always fails
	failingWriter := &failingWriter{}

	registry := NewRouteRegistry()
	var logCalled bool
	var logFormat string

	config := ExportConfig{
		Output: failingWriter,
		ExitFunc: func(code int) {
			t.Error("ExitFunc should not be called when testing error path")
		},
		LogFatalf: func(format string, args ...interface{}) {
			logCalled = true
			logFormat = format
		},
	}

	// Test error case
	err := exportOpenAPISpec(registry, config)

	if err == nil {
		t.Error("Expected error from failing writer")
	}
	if !logCalled {
		t.Error("LogFatalf should have been called")
	}
	if !strings.Contains(logFormat, "failed to encode OpenAPI spec") {
		t.Error("LogFatalf should contain expected error message")
	}
}

// Test error path in ExportOpenAPIAndExit
func TestTypedRouterExportOpenAPIAndExitError(t *testing.T) {
	// Create test router
	registry := NewRouteRegistry()
	router := &TypedRouter[any]{
		registry: registry,
	}

	// Save original config
	originalConfig := exportConfig
	defer func() {
		exportConfig = originalConfig
	}()

	// Set up test config with failing writer
	failingWriter := &failingWriter{}
	var exitCalled bool
	var logCalled bool

	testConfig := ExportConfig{
		Output: failingWriter,
		ExitFunc: func(code int) {
			exitCalled = true
			t.Error("ExitFunc should not be called when exportOpenAPISpec fails")
		},
		LogFatalf: func(format string, args ...interface{}) {
			logCalled = true
		},
	}

	SetExportConfig(testConfig)

	// Test ExportOpenAPIAndExit with error
	router.ExportOpenAPIAndExit()

	// Verify exit was NOT called (early return on error)
	if exitCalled {
		t.Error("ExitFunc should not have been called due to early return on error")
	}

	// Verify logging occurred in exportOpenAPISpec
	if !logCalled {
		t.Error("LogFatalf should have been called in exportOpenAPISpec")
	}
}

// failingWriter is a writer that always returns an error
type failingWriter struct{}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}
