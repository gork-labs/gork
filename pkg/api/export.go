package api

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

// ExitFunc allows dependency injection for testing.
type ExitFunc func(int)

// LogFatalfFunc allows dependency injection for testing.
type LogFatalfFunc func(string, ...interface{})

// ExportConfig holds configuration for export functionality.
type ExportConfig struct {
	Output    io.Writer
	ExitFunc  ExitFunc
	LogFatalf LogFatalfFunc
}

// DefaultExportConfig returns the default export configuration.
func DefaultExportConfig() ExportConfig {
	return ExportConfig{
		Output:    os.Stdout,
		ExitFunc:  os.Exit,
		LogFatalf: log.Fatalf,
	}
}

// exportConfig holds the current export configuration.
var exportConfig = DefaultExportConfig()

// SetExportConfig allows setting a custom export configuration for testing.
func SetExportConfig(config ExportConfig) {
	exportConfig = config
}

// exportOpenAPISpec generates and writes the OpenAPI spec using the provided configuration.
func exportOpenAPISpec(registry *RouteRegistry, config ExportConfig, opts ...OpenAPIOption) error {
	spec := GenerateOpenAPI(registry, opts...)

	enc := json.NewEncoder(config.Output)
	enc.SetIndent("", "  ")
	if err := enc.Encode(spec); err != nil {
		config.LogFatalf("failed to encode OpenAPI spec: %v", err)
		return err
	}
	return nil
}

// ExportOpenAPIAndExit generates an OpenAPI specification from the router's
// internal RouteRegistry, writes it to stdout as pretty-printed JSON and then
// terminates the process with exit code 0.
//
// This function always exports and exits when called. Users should call it
// only when they want to export the OpenAPI specification (e.g., when
// GORK_EXPORT=1 environment variable is set).
func (r *TypedRouter[T]) ExportOpenAPIAndExit(opts ...OpenAPIOption) {
	if err := exportOpenAPISpec(r.registry, exportConfig, opts...); err != nil {
		return // Error already logged by exportOpenAPISpec
	}

	// Ensure graceful termination so that calling scripts can rely on the
	// process exiting once the specification has been emitted.
	exportConfig.ExitFunc(0)
}
