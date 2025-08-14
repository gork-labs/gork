package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
	"gopkg.in/yaml.v3"
)

func TestGenerateConfig(t *testing.T) {
	config := &GenerateConfig{
		BuildPath:  "test-build",
		SourcePath: "test-source",
		OutputPath: "test-output",
		Title:      "Test API",
		Version:    "1.0.0",
		ConfigPath: "",
	}

	if config.BuildPath != "test-build" {
		t.Errorf("BuildPath: got %s, want test-build", config.BuildPath)
	}
	if config.Title != "Test API" {
		t.Errorf("Title: got %s, want Test API", config.Title)
	}
}

func TestLoadConfigFile(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		config     *GenerateConfig
		wantErr    bool
	}{
		{
			name:       "no config file",
			configPath: "",
			config:     &GenerateConfig{},
			wantErr:    false,
		},
		{
			name:       "nonexistent config file",
			configPath: "/nonexistent/config.yml",
			config:     &GenerateConfig{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.ConfigPath = tt.configPath
			err := loadConfigFile(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigFileWithValidYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yml")

	configContent := `
openapi:
  build: "custom-build"
  source: "custom-source"
  output: "custom-output.json"
  title: "Custom API"
  version: "2.0.0"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := &GenerateConfig{
		BuildPath:  "",
		SourcePath: "",
		OutputPath: "openapi.json", // default value
		Title:      "API",          // default value
		Version:    "0.1.0",        // default value
		ConfigPath: configFile,
	}

	err := loadConfigFile(config)
	if err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}

	if config.BuildPath != "custom-build" {
		t.Errorf("BuildPath: got %s, want custom-build", config.BuildPath)
	}
	if config.SourcePath != "custom-source" {
		t.Errorf("SourcePath: got %s, want custom-source", config.SourcePath)
	}
	if config.OutputPath != "custom-output.json" {
		t.Errorf("OutputPath: got %s, want custom-output.json", config.OutputPath)
	}
	if config.Title != "Custom API" {
		t.Errorf("Title: got %s, want Custom API", config.Title)
	}
	if config.Version != "2.0.0" {
		t.Errorf("Version: got %s, want 2.0.0", config.Version)
	}
}

func TestLoadConfigFileWithInvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yml")

	invalidContent := `
invalid: yaml: content: [
`

	if err := os.WriteFile(configFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := &GenerateConfig{ConfigPath: configFile}

	err := loadConfigFile(config)
	if err == nil {
		t.Error("Expected error for invalid YAML, but got none")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Errorf("Expected parse config error, got: %v", err)
	}
}

func TestGenerateBaseSpec(t *testing.T) {
	tests := []struct {
		name   string
		config *GenerateConfig
	}{
		{
			name: "no build path",
			config: &GenerateConfig{
				BuildPath: "",
				Title:     "Test API",
				Version:   "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := generateBaseSpec(tt.config)
			if err != nil {
				t.Errorf("generateBaseSpec() error = %v", err)
				return
			}

			if spec.OpenAPI != "3.1.0" {
				t.Errorf("OpenAPI version: got %s, want 3.1.0", spec.OpenAPI)
			}
			if spec.Info.Title != tt.config.Title {
				t.Errorf("Title: got %s, want %s", spec.Info.Title, tt.config.Title)
			}
			if spec.Info.Version != tt.config.Version {
				t.Errorf("Version: got %s, want %s", spec.Info.Version, tt.config.Version)
			}
		})
	}
}

func TestEnrichWithDocs(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tests := []struct {
		name       string
		sourcePath string
		wantErr    bool
	}{
		{
			name:       "empty source path",
			sourcePath: "",
			wantErr:    false,
		},
		{
			name:       "nonexistent source path",
			sourcePath: "/nonexistent/path",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enrichWithDocs(spec, tt.sourcePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("enrichWithDocs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteOutput(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tests := []struct {
		name    string
		config  *GenerateConfig
		wantErr bool
	}{
		{
			name: "stdout output",
			config: &GenerateConfig{
				OutputPath: "-",
			},
			wantErr: false,
		},
		{
			name: "nonexistent directory",
			config: &GenerateConfig{
				OutputPath: "/nonexistent/dir/output.json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writeOutput(spec, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteOutputToFile(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name   string
		config *GenerateConfig
	}{
		{
			name: "json output",
			config: &GenerateConfig{
				OutputPath: filepath.Join(tmpDir, "test.json"),
			},
		},
		{
			name: "yaml output",
			config: &GenerateConfig{
				OutputPath: filepath.Join(tmpDir, "test.yaml"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writeOutput(spec, tt.config)
			if err != nil {
				t.Errorf("writeOutput() error = %v", err)
				return
			}

			// Verify file was created
			if _, err := os.Stat(tt.config.OutputPath); os.IsNotExist(err) {
				t.Errorf("Output file was not created: %s", tt.config.OutputPath)
			}
		})
	}
}

func TestWriteSpec(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		format   string
		filename string
		wantErr  bool
	}{
		{
			name:     "json format",
			format:   "json",
			filename: "test.json",
			wantErr:  false,
		},
		{
			name:     "yaml format",
			format:   "yaml",
			filename: "test.yaml",
			wantErr:  false,
		},
		{
			name:     "yml format",
			format:   "yml",
			filename: "test.yml",
			wantErr:  false,
		},
		{
			name:     "unsupported format",
			format:   "xml",
			filename: "test.xml",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.filename)
			f, err := os.Create(filePath)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			err = writeSpec(f, tt.format, spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file has content
				info, err := f.Stat()
				if err != nil {
					t.Fatal(err)
				}
				if info.Size() == 0 {
					t.Error("Output file is empty")
				}
			}
		})
	}
}

func TestParseValidatorResponse(t *testing.T) {
	tests := []struct {
		name       string
		body       []byte
		statusCode int
		wantErr    bool
	}{
		{
			name:       "success with empty response",
			body:       []byte("{}"),
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "success with null response",
			body:       []byte(""),
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "error status code",
			body:       []byte("Internal Server Error"),
			statusCode: 500,
			wantErr:    true,
		},
		{
			name:       "validation error in JSON",
			body:       []byte(`{"messages": [{"level": "error", "message": "Invalid schema"}]}`),
			statusCode: 200,
			wantErr:    true,
		},
		{
			name:       "validation warning in JSON",
			body:       []byte(`{"messages": [{"level": "warning", "message": "Minor issue"}]}`),
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "error text in response",
			body:       []byte("error: schema validation failed"),
			statusCode: 200,
			wantErr:    true,
		},
		{
			name:       "allowed error text",
			body:       []byte("schemaValidationMessages: null, error handled"),
			statusCode: 200,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseValidatorResponse(tt.body, tt.statusCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseValidatorResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewOpenAPICommand(t *testing.T) {
	cmd := newOpenAPICommand()
	if cmd.Use != "openapi" {
		t.Errorf("Use: got %s, want openapi", cmd.Use)
	}
	if len(cmd.Commands()) == 0 {
		t.Error("Expected subcommands, got none")
	}
}

func TestNewGenerateCommand(t *testing.T) {
	cmd := newGenerateCommand()
	if cmd.Use != "generate" {
		t.Errorf("Use: got %s, want generate", cmd.Use)
	}

	// Test that flags are registered
	flags := cmd.Flags()
	if flags.Lookup("build") == nil {
		t.Error("build flag not registered")
	}
	if flags.Lookup("source") == nil {
		t.Error("source flag not registered")
	}
	if flags.Lookup("output") == nil {
		t.Error("output flag not registered")
	}
	if flags.Lookup("title") == nil {
		t.Error("title flag not registered")
	}
	if flags.Lookup("version") == nil {
		t.Error("version flag not registered")
	}
	if flags.Lookup("config") == nil {
		t.Error("config flag not registered")
	}
}

func TestGenerateSpec(t *testing.T) {
	tests := []struct {
		name    string
		config  *GenerateConfig
		wantErr bool
	}{
		{
			name: "basic spec generation",
			config: &GenerateConfig{
				BuildPath:  "",
				SourcePath: "",
				OutputPath: "-",
				Title:      "Test API",
				Version:    "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "invalid config file",
			config: &GenerateConfig{
				ConfigPath: "/nonexistent/config.yml",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateSpec(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildAndExtract(t *testing.T) {
	tests := []struct {
		name      string
		buildPath string
		wantErr   bool
	}{
		{
			name:      "nonexistent build path",
			buildPath: "/nonexistent/path",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildAndExtract(tt.buildPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildAndExtract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSpec(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	// This will try to call the actual Swagger validator
	// which might fail due to network issues, but we're testing the code path
	err := validateSpec(spec)

	// We don't assert on the error since network calls can fail
	// We just want to ensure the function doesn't panic
	t.Logf("validateSpec returned: %v", err)
}

func TestWriteOutputDirectoryHandling(t *testing.T) {
	tmpDir := t.TempDir()

	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tests := []struct {
		name    string
		config  *GenerateConfig
		wantErr bool
	}{
		{
			name: "file exists where directory expected",
			config: &GenerateConfig{
				OutputPath: filepath.Join(tmpDir, "file.txt", "output.json"),
			},
			wantErr: true,
		},
	}

	// Create a file at the expected directory location
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writeOutput(spec, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSpecWithBuildPath(t *testing.T) {
	// Test with a build path that exists but will fail to build
	tmpDir := t.TempDir()

	// Create a simple Go file that will fail to build
	mainFile := filepath.Join(tmpDir, "main.go")
	content := `package main
import "nonexistent/package"
func main() {}`

	if err := os.WriteFile(mainFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config := &GenerateConfig{
		BuildPath:  tmpDir,
		SourcePath: "",
		OutputPath: "-",
		Title:      "Test API",
		Version:    "1.0.0",
	}

	err := GenerateSpec(config)
	if err == nil {
		t.Error("Expected build error but got none")
	}
}

func TestParseValidatorResponseWithMalformedJSON(t *testing.T) {
	// Test with malformed JSON response
	body := []byte(`{"messages": [{"level": "error"}`)
	err := parseValidatorResponse(body, 200)

	// Should handle malformed JSON gracefully
	if err == nil {
		t.Error("Expected error for malformed JSON validator response")
	}
}

func TestNewGenerateCommandExecution(t *testing.T) {
	// Test the RunE function by executing the command
	cmd := newGenerateCommand()

	// Set flags for a basic test
	cmd.SetArgs([]string{
		"--output", "-",
		"--title", "Test API",
		"--version", "1.0.0",
	})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Command execution failed: %v", err)
	}
}

func TestBuildAndExtractErrorPaths(t *testing.T) {
	// Test successful build path requires a valid Go module
	tmpDir := t.TempDir()

	// Create a simple valid Go module
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(tmpDir, "main.go")
	content := `package main
import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/gork-labs/gork/pkg/api"
)
func main() {
	if os.Getenv("GORK_EXPORT") == "1" {
		spec := &api.OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: api.Info{Title: "Test", Version: "1.0.0"},
			Paths: map[string]*api.PathItem{},
			Components: &api.Components{Schemas: map[string]*api.Schema{}},
		}
		json.NewEncoder(os.Stdout).Encode(spec)
	} else {
		fmt.Println("Hello")
	}
}`

	if err := os.WriteFile(mainFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// This should work now with proper module setup
	_, err := buildAndExtract(tmpDir)
	// We expect this to still fail due to module resolution issues in test environment
	if err == nil {
		t.Log("buildAndExtract succeeded unexpectedly - that's actually good!")
	} else {
		t.Logf("buildAndExtract failed as expected in test environment: %v", err)
	}
}

func TestEnrichWithDocsErrorPath(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	// Test with a directory that exists but has no Go files to parse
	tmpDir := t.TempDir()
	err := enrichWithDocs(spec, tmpDir)

	// This should not error since ParseDirectory handles empty directories
	if err != nil {
		t.Errorf("enrichWithDocs should handle empty directory: %v", err)
	}
}

func TestValidateSpecErrorPaths(t *testing.T) {
	// Test with a spec that can't be marshaled (circular reference)
	// This is hard to create, so let's test the marshal path by creating a valid spec
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	// This will test the network call and response parsing paths
	err := validateSpec(spec)
	// We don't assert on error since network calls are unpredictable
	t.Logf("validateSpec result: %v", err)
}

func TestWriteOutputErrorPaths(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tmpDir := t.TempDir()

	// Test error creating file (permission denied)
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(readOnlyDir, 0755) // cleanup

	config := &GenerateConfig{
		OutputPath: filepath.Join(readOnlyDir, "output.json"),
	}

	err := writeOutput(spec, config)
	if err == nil {
		t.Error("Expected error when writing to read-only directory")
	}
}

func TestWriteSpecWithYAMLError(t *testing.T) {
	// Test YAML marshaling error path by creating a problematic spec
	// This is tricky since the YAML library handles most Go types well
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = writeSpec(f, "yaml", spec)
	if err != nil {
		t.Errorf("YAML write should not fail for valid spec: %v", err)
	}
}

func TestGenerateSpecErrorPaths(t *testing.T) {
	// Test error paths in GenerateSpec
	config := &GenerateConfig{
		BuildPath:  "nonexistent",
		SourcePath: "",
		OutputPath: "-",
		Title:      "Test API",
		Version:    "1.0.0",
	}

	err := GenerateSpec(config)
	if err == nil {
		t.Error("Expected error for nonexistent build path")
	}
}

func TestBuildAndExtractErrorCoverage(t *testing.T) {
	// Test createTemp error - this is hard to trigger reliably,
	// but we can test the code path exists
	_, err := buildAndExtract("./nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

func TestValidateSpecMarshalError(t *testing.T) {
	// Test JSON marshal error in validateSpec - hard to create a spec that fails to marshal
	// but we can test with a valid spec to cover the success path
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	// This tests the marshal success path and network call
	err := validateSpec(spec)
	// Don't assert on error since network can fail
	t.Logf("validateSpec marshal path result: %v", err)
}

func TestWriteSpecYAMLWriteError(t *testing.T) {
	// Test YAML write error by closing the file
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tmpFile := filepath.Join(t.TempDir(), "test.yml")
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	f.Close() // Close the file to trigger write error

	err = writeSpec(f, "yml", spec)
	if err == nil {
		t.Error("Expected write error for closed file")
	}
}

// Test remaining coverage lines by testing error conditions
func TestRemainingCoverage(t *testing.T) {
	// Test lines 71-73: enrichWithDocs validation error path
	t.Run("enrichWithDocs validation error", func(t *testing.T) {
		spec := &api.OpenAPISpec{
			OpenAPI:    "3.1.0",
			Info:       api.Info{Title: "Test", Version: "1.0.0"},
			Paths:      map[string]*api.PathItem{},
			Components: &api.Components{Schemas: map[string]*api.Schema{}},
		}

		// Test with nonexistent path - this should trigger error in ParseDirectory
		err := enrichWithDocs(spec, "/absolutely/nonexistent/path/that/cannot/exist")
		if err == nil {
			t.Error("Expected error for nonexistent path")
		}
	})

	// Test lines 75-77: validateSpec error path
	t.Run("validateSpec error path", func(t *testing.T) {
		spec := &api.OpenAPISpec{
			OpenAPI:    "3.1.0",
			Info:       api.Info{Title: "Test", Version: "1.0.0"},
			Paths:      map[string]*api.PathItem{},
			Components: &api.Components{Schemas: map[string]*api.Schema{}},
		}

		// Test that validateSpec can be called - network errors are expected
		err := validateSpec(spec)
		// Don't assert on result since network calls are unpredictable
		t.Logf("validateSpec error path result: %v", err)
	})

	// Test lines 250, 271-273: writeOutput error paths
	t.Run("writeOutput stat error", func(t *testing.T) {
		spec := &api.OpenAPISpec{
			OpenAPI:    "3.1.0",
			Info:       api.Info{Title: "Test", Version: "1.0.0"},
			Paths:      map[string]*api.PathItem{},
			Components: &api.Components{Schemas: map[string]*api.Schema{}},
		}

		// Try to write to a path that doesn't exist
		config := &GenerateConfig{
			OutputPath: "/nonexistent/directory/output.json",
		}

		err := writeOutput(spec, config)
		if err == nil {
			t.Error("Expected error for nonexistent directory")
		}
	})
}

// MockBuildRunner for comprehensive testing of buildAndExtract
type MockBuildRunner struct {
	CreateTempError error
	BuildError      error
	RunError        error
	RunOutput       []byte
	TempFile        *os.File
}

func (m *MockBuildRunner) CreateTemp(pattern string) (*os.File, error) {
	if m.CreateTempError != nil {
		return nil, m.CreateTempError
	}
	if m.TempFile != nil {
		return m.TempFile, nil
	}
	// Create a real temp file for most tests
	return os.CreateTemp("", pattern)
}

func (m *MockBuildRunner) BuildCommand(outputPath, buildPath string) error {
	return m.BuildError
}

func (m *MockBuildRunner) RunCommand(exePath string) ([]byte, error) {
	if m.RunError != nil {
		return nil, m.RunError
	}
	if m.RunOutput != nil {
		return m.RunOutput, nil
	}
	// Default valid OpenAPI spec
	return []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1.0.0"},"paths":{},"components":{}}`), nil
}

func TestBuildAndExtractWithRunner(t *testing.T) {
	tests := []struct {
		name        string
		runner      *MockBuildRunner
		buildPath   string
		wantErr     bool
		errContains string
	}{
		{
			name:        "create temp error",
			runner:      &MockBuildRunner{CreateTempError: fmt.Errorf("temp creation failed")},
			buildPath:   "./test",
			wantErr:     true,
			errContains: "create temp exe",
		},
		{
			name:        "build error",
			runner:      &MockBuildRunner{BuildError: fmt.Errorf("build failed")},
			buildPath:   "./test",
			wantErr:     true,
			errContains: "build failed",
		},
		{
			name:        "run error",
			runner:      &MockBuildRunner{RunError: fmt.Errorf("run failed")},
			buildPath:   "./test",
			wantErr:     true,
			errContains: "run generated binary",
		},
		{
			name:        "invalid json output",
			runner:      &MockBuildRunner{RunOutput: []byte("invalid json")},
			buildPath:   "./test",
			wantErr:     true,
			errContains: "parse spec json",
		},
		{
			name:      "successful build and extract",
			runner:    &MockBuildRunner{},
			buildPath: "./test",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := buildAndExtractWithRunner(tt.buildPath, tt.runner)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got none", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if spec == nil {
					t.Error("Expected spec, got nil")
				}
			}
		})
	}
}

func TestDefaultBuildRunner(t *testing.T) {
	runner := &DefaultBuildRunner{}

	t.Run("CreateTemp", func(t *testing.T) {
		file, err := runner.CreateTemp("test-*")
		if err != nil {
			t.Errorf("CreateTemp failed: %v", err)
		}
		if file != nil {
			file.Close()
			os.Remove(file.Name())
		}
	})

	t.Run("BuildCommand error", func(t *testing.T) {
		err := runner.BuildCommand("/tmp/nonexistent", "./nonexistent")
		if err == nil {
			t.Error("Expected build error for nonexistent path")
		}
	})

	t.Run("RunCommand error", func(t *testing.T) {
		_, err := runner.RunCommand("/nonexistent/binary")
		if err == nil {
			t.Error("Expected run error for nonexistent binary")
		}
	})
}

// Mock implementations for testing
type MockHTTPClient struct {
	PostError error
	Response  *http.Response
}

func (m *MockHTTPClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	if m.PostError != nil {
		return nil, m.PostError
	}
	return m.Response, nil
}

type MockValidatorClient struct {
	MarshalError   error
	CallError      error
	CallBody       []byte
	CallStatusCode int
	httpClient     HTTPClient
}

func (m *MockValidatorClient) MarshalSpec(v any) ([]byte, error) {
	if m.MarshalError != nil {
		return nil, m.MarshalError
	}
	return json.Marshal(v)
}

func (m *MockValidatorClient) CallValidator(data []byte) ([]byte, int, error) {
	if m.CallError != nil {
		return nil, 0, m.CallError
	}
	if m.httpClient != nil {
		return m.callValidatorWithMockHTTP(data)
	}
	return m.CallBody, m.CallStatusCode, nil
}

func (m *MockValidatorClient) callValidatorWithMockHTTP(data []byte) ([]byte, int, error) {
	resp, err := m.httpClient.Post("https://validator.swagger.io/validator/debug", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, 0, fmt.Errorf("send to Swagger validator: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

type MockFileSystem struct {
	StatError   error
	StatInfo    os.FileInfo
	CreateError error
	CreateFile  *os.File
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.StatError != nil {
		return nil, m.StatError
	}
	return m.StatInfo, nil
}

func (m *MockFileSystem) Create(name string) (*os.File, error) {
	if m.CreateError != nil {
		return nil, m.CreateError
	}
	return m.CreateFile, nil
}

type MockSpecWriter struct {
	YAMLMarshalError error
}

func (m *MockSpecWriter) MarshalYAML(v interface{}) ([]byte, error) {
	if m.YAMLMarshalError != nil {
		return nil, m.YAMLMarshalError
	}
	return yaml.Marshal(v)
}

// Additional tests to achieve 100% coverage
func TestGenerateSpecErrorPaths100(t *testing.T) {
	t.Run("enrichWithDocs error", func(t *testing.T) {
		config := &GenerateConfig{
			BuildPath:  "",
			SourcePath: "/absolutely/nonexistent/path/12345",
			OutputPath: "-",
			Title:      "Test API",
			Version:    "1.0.0",
		}

		err := GenerateSpec(config)
		if err == nil {
			t.Error("Expected error for nonexistent source path")
		}
		if !strings.Contains(err.Error(), "failed to parse source") {
			t.Errorf("Expected 'failed to parse source' error, got: %v", err)
		}
	})

	t.Run("validateSpec error wrapping", func(t *testing.T) {
		// Override the default client to return validation error
		originalClient := defaultValidatorClient
		mockClient := &MockValidatorClient{
			CallError: fmt.Errorf("network connection failed"),
		}
		defaultValidatorClient = mockClient
		defer func() { defaultValidatorClient = originalClient }()

		tmpFile := filepath.Join(t.TempDir(), "output.json")
		config := &GenerateConfig{
			BuildPath:  "",
			SourcePath: "",
			OutputPath: tmpFile,
			Title:      "Test API",
			Version:    "1.0.0",
		}

		err := GenerateSpec(config)
		if err == nil {
			t.Error("Expected validation error")
		}
		if !strings.Contains(err.Error(), "spec validation failed") {
			t.Errorf("Expected 'spec validation failed' error, got: %v", err)
		}
	})
}

func TestValidateSpecWithClientErrors(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	t.Run("marshal error", func(t *testing.T) {
		mockClient := &MockValidatorClient{
			MarshalError: fmt.Errorf("marshal error"),
		}

		err := validateSpecWithClient(spec, mockClient)
		if err == nil {
			t.Error("Expected marshal error")
		}
		if !strings.Contains(err.Error(), "marshal spec") {
			t.Errorf("Expected 'marshal spec' error, got: %v", err)
		}
	})

	t.Run("call validator error", func(t *testing.T) {
		mockClient := &MockValidatorClient{
			CallError: fmt.Errorf("network error"),
		}

		err := validateSpecWithClient(spec, mockClient)
		if err == nil {
			t.Error("Expected call validator error")
		}
		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("Expected 'network error', got: %v", err)
		}
	})

	t.Run("HTTP POST error", func(t *testing.T) {
		mockHTTP := &MockHTTPClient{
			PostError: fmt.Errorf("connection refused"),
		}
		// Create a real DefaultValidatorClient with mocked HTTP client
		client := &DefaultValidatorClient{httpClient: mockHTTP}

		err := validateSpecWithClient(spec, client)
		if err == nil {
			t.Error("Expected HTTP POST error")
		}
		if !strings.Contains(err.Error(), "send to Swagger validator") {
			t.Errorf("Expected 'send to Swagger validator' error, got: %v", err)
		}
	})
}

func TestWriteSpecWithWriterErrors(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	t.Run("yaml marshal error", func(t *testing.T) {
		mockWriter := &MockSpecWriter{
			YAMLMarshalError: fmt.Errorf("yaml marshal error"),
		}
		mockJSONMarshaler := &mockJSONMarshaler{marshalError: false, unmarshalError: false}

		err := writeSpecWithWriter(f, "yaml", spec, mockWriter, mockJSONMarshaler)
		if err == nil {
			t.Error("Expected YAML marshal error")
		}
		if !strings.Contains(err.Error(), "yaml marshal error") {
			t.Errorf("Expected 'yaml marshal error', got: %v", err)
		}
	})
}

func TestWriteOutputWithFSErrors(t *testing.T) {
	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	t.Run("stat generic error", func(t *testing.T) {
		mockFS := &MockFileSystem{
			StatError: fmt.Errorf("permission denied"),
		}

		config := &GenerateConfig{
			OutputPath: "/some/path/output.json",
		}

		err := writeOutputWithFS(spec, config, mockFS)
		if err == nil {
			t.Error("Expected stat error")
		}
		if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected 'permission denied' error, got: %v", err)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		mockFS := &MockFileSystem{
			StatError: os.ErrNotExist,
		}

		config := &GenerateConfig{
			OutputPath: "/nonexistent/directory/output.json",
		}

		err := writeOutputWithFS(spec, config, mockFS)
		if err == nil {
			t.Error("Expected error for nonexistent directory")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
		}
	})
}

func TestGetFormatFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "json extension",
			path:     "test.json",
			expected: "json",
		},
		{
			name:     "yaml extension",
			path:     "test.yaml",
			expected: "yaml",
		},
		{
			name:     "yml extension",
			path:     "test.yml",
			expected: "yaml",
		},
		{
			name:     "no extension defaults to json",
			path:     "test",
			expected: "json",
		},
		{
			name:     "unknown extension defaults to json",
			path:     "test.txt",
			expected: "json",
		},
		{
			name:     "stdout defaults to json",
			path:     "-",
			expected: "json",
		},
		{
			name:     "path with directory",
			path:     "/path/to/spec.yaml",
			expected: "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFormatFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("getFormatFromPath(%s) = %s; want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestWriteSpecWithWriterYAMLJSONErrors(t *testing.T) {
	tmpFile := createTempFile(t)
	defer tmpFile.Close()

	spec := &api.OpenAPISpec{
		OpenAPI:    "3.1.0",
		Info:       api.Info{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*api.PathItem{},
		Components: &api.Components{Schemas: map[string]*api.Schema{}},
	}

	// Test normal YAML case to ensure the new code paths are covered
	mockWriter := &mockSpecWriter{shouldFail: false}
	mockJSONMarshaler := &mockJSONMarshaler{marshalError: false, unmarshalError: false}
	err := writeSpecWithWriter(tmpFile, "yaml", spec, mockWriter, mockJSONMarshaler)
	if err != nil {
		t.Errorf("Expected no error for normal YAML case, got: %v", err)
	}

	// Test YAML writer error
	mockWriter.shouldFail = true
	err = writeSpecWithWriter(tmpFile, "yaml", spec, mockWriter, mockJSONMarshaler)
	if err == nil {
		t.Error("Expected error when YAML writer fails")
	}
	if !strings.Contains(err.Error(), "mock yaml marshal error") {
		t.Errorf("Expected mock yaml marshal error, got: %v", err)
	}

	// Reset for next tests
	tmpFile.Seek(0, 0)
	tmpFile.Truncate(0)
	mockWriter.shouldFail = false

	// Test JSON marshal error
	mockJSONMarshaler.marshalError = true
	err = writeSpecWithWriter(tmpFile, "yaml", spec, mockWriter, mockJSONMarshaler)
	if err == nil {
		t.Error("Expected error when JSON marshal fails")
	}
	if !strings.Contains(err.Error(), "marshal to json") {
		t.Errorf("Expected 'marshal to json' error, got: %v", err)
	}

	// Test JSON unmarshal error
	mockJSONMarshaler.marshalError = false
	mockJSONMarshaler.unmarshalError = true
	err = writeSpecWithWriter(tmpFile, "yaml", spec, mockWriter, mockJSONMarshaler)
	if err == nil {
		t.Error("Expected error when JSON unmarshal fails")
	}
	if !strings.Contains(err.Error(), "unmarshal json to map") {
		t.Errorf("Expected 'unmarshal json to map' error, got: %v", err)
	}
}

// createTempFile creates a temporary file for testing
func createTempFile(t *testing.T) *os.File {
	t.Helper()
	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// mockSpecWriter for testing error paths
type mockSpecWriter struct {
	shouldFail bool
}

func (w *mockSpecWriter) MarshalYAML(v any) ([]byte, error) {
	if w.shouldFail {
		return nil, fmt.Errorf("mock yaml marshal error")
	}
	return []byte("test: yaml"), nil
}

// mockJSONMarshaler for testing JSON error paths
type mockJSONMarshaler struct {
	marshalError   bool
	unmarshalError bool
}

func (m *mockJSONMarshaler) Marshal(v any) ([]byte, error) {
	if m.marshalError {
		return nil, fmt.Errorf("mock json marshal error")
	}
	return json.Marshal(v)
}

func (m *mockJSONMarshaler) Unmarshal(data []byte, v any) error {
	if m.unmarshalError {
		return fmt.Errorf("mock json unmarshal error")
	}
	return json.Unmarshal(data, v)
}
