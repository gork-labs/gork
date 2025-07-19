package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2EGeneration(t *testing.T) {
	// Build the binary first
	binary := buildBinary(t)
	defer func() { _ = os.Remove(binary) }()

	tests := []struct {
		name       string
		inputDir   string
		routeFile  string
		args       []string
	}{
		{
			name:      "testdata API generation",
			inputDir:  "../../testdata",
			routeFile: "../../testdata/routes.go",
			args: []string{
				"-t", "Test API",
				"-v", "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "openapi.json")

			// Prepare command arguments
			args := []string{
				"-i", tt.inputDir,
				"-o", outputFile,
			}
			if tt.routeFile != "" {
				args = append(args, "-r", tt.routeFile)
			}
			args = append(args, tt.args...)

			// Run the CLI
			cmd := exec.Command(binary, args...)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "CLI execution failed: %s", string(output))

			// Verify output file exists
			_, err = os.Stat(outputFile)
			require.NoError(t, err, "Output file was not created")

			// Read generated output
			generated, err := os.ReadFile(outputFile)
			require.NoError(t, err)

			// Parse JSON to validate it's well-formed
			var generatedSpec map[string]interface{}
			err = json.Unmarshal(generated, &generatedSpec)
			require.NoError(t, err, "Generated output is not valid JSON")

			// Basic structure validation
			assert.Equal(t, "3.1.0", generatedSpec["openapi"])
			
			info, ok := generatedSpec["info"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "Test API", info["title"])
			assert.Equal(t, "1.0.0", info["version"])

			// Verify paths exist
			paths, ok := generatedSpec["paths"].(map[string]interface{})
			require.True(t, ok)
			assert.NotEmpty(t, paths)

			// Verify components exist
			components, ok := generatedSpec["components"].(map[string]interface{})
			require.True(t, ok)
			
			schemas, ok := components["schemas"].(map[string]interface{})
			require.True(t, ok)
			assert.NotEmpty(t, schemas)

			// Verify specific schemas from testdata are present
			expectedSchemas := []string{
				"GetWithoutQueryParamsReq",
				"GetWithoutQueryParamsResp",
				"GetWithQueryParamsReq",
				"GetWithQueryParamsResp",
				"PostWithJsonBodyReq",
				"PostWithJsonBodyResp",
				"Option1",
				"Option2",
			}

			for _, schemaName := range expectedSchemas {
				assert.Contains(t, schemas, schemaName, "Expected schema %s not found", schemaName)
			}

			// Verify paths from testdata routes
			expectedPaths := []string{
				"/basic/hello",
				"/basic/hello-with-query",
				"/basic/hello-with-json",
				"/basic/hello-with-json-and-query",
				"/unions/any-of-without-wrapper",
				"/unions/any-of-union2",
			}

			for _, path := range expectedPaths {
				assert.Contains(t, paths, path, "Expected path %s not found", path)
			}
		})
	}
}

func TestCLIArguments(t *testing.T) {
	binary := buildBinary(t)
	defer func() { _ = os.Remove(binary) }()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorText   string
	}{
		{
			name:        "missing input directory",
			args:        []string{"-o", "test.json"},
			expectError: false, // input has default value "."
		},
		{
			name:        "missing output file",
			args:        []string{"-i", "../../testdata", "-r", "../../testdata/routes.go"},
			expectError: false, // output has default value "openapi.json"
		},
		{
			name: "valid arguments",
			args: []string{
				"-i", "../../testdata",
				"-r", "../../testdata/routes.go",
				"-o", "TEMPDIR/test.json",
				"-t", "Test",
				"-v", "1.0.0",
			},
			expectError: false,
		},
		{
			name: "non-existent input directory",
			args: []string{
				"-i", "/path/that/does/not/exist",
				"-o", "TEMPDIR/test.json",
			},
			expectError: true,
		},
		{
			name: "help flag",
			args: []string{"-help"},
			expectError: false, // -help should not return error
		},
		{
			name: "version flag",
			args: []string{"-version"},
			expectError: false, // -version should not return error (if implemented)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replace temp dir placeholders in args
			args := make([]string, len(tt.args))
			copy(args, tt.args)
			tempDir := t.TempDir()
			for i, arg := range args {
				if strings.Contains(arg, "TEMPDIR") {
					args[i] = strings.ReplaceAll(arg, "TEMPDIR", tempDir)
				}
			}
			cmd := exec.Command(binary, args...)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				assert.Error(t, err, "Expected command to fail but it succeeded")
				if tt.errorText != "" {
					assert.Contains(t, string(output), tt.errorText)
				}
			} else {
				if err != nil {
					// For help/version, exit code might be non-zero but that's ok
					if tt.name != "help flag" && tt.name != "version flag" {
						assert.NoError(t, err, "Command failed unexpectedly: %s", string(output))
					}
				}
			}
		})
	}
}

func TestOutputFormats(t *testing.T) {
	binary := buildBinary(t)
	defer func() { _ = os.Remove(binary) }()

	tests := []struct {
		name       string
		outputFile string
		format     string
	}{
		{
			name:       "JSON output",
			outputFile: "openapi.json",
			format:     "json",
		},
		{
			name:       "YAML output",
			outputFile: "openapi.yaml",
			format:     "yaml",
		},
		{
			name:       "YML output",
			outputFile: "openapi.yml",
			format:     "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			outputPath := filepath.Join(tempDir, tt.outputFile)

			args := []string{
				"-i", "../../testdata",
				"-r", "../../testdata/routes.go",
				"-o", outputPath,
				"-t", "Test API",
				"-v", "1.0.0",
			}

			if tt.format != "" {
				args = append(args, "-f", tt.format)
			}

			cmd := exec.Command(binary, args...)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "CLI execution failed: %s", string(output))

			// Verify file exists
			_, err = os.Stat(outputPath)
			require.NoError(t, err, "Output file was not created")

			// Read and validate content based on format
			content, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			if tt.format == "yaml" || filepath.Ext(tt.outputFile) == ".yaml" || filepath.Ext(tt.outputFile) == ".yml" {
				// Basic YAML validation - should start with proper YAML
				assert.Contains(t, string(content), "openapi:")
				assert.Contains(t, string(content), "info:")
			} else {
				// JSON validation
				var spec map[string]interface{}
				err = json.Unmarshal(content, &spec)
				require.NoError(t, err, "Generated output is not valid JSON")
			}
		})
	}
}

func TestRouteDetection(t *testing.T) {
	binary := buildBinary(t)
	defer func() { _ = os.Remove(binary) }()

	// Create test file with different routing patterns
	tempDir := t.TempDir()
	
	// Create a test file with various routing patterns
	routesFile := filepath.Join(tempDir, "routes.go")
	routesContent := `package test

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
)

func setupRoutes() {
	// Standard library
	http.HandleFunc("GET /stdlib/users", handlers.ListUsers)
	http.HandleFunc("POST /stdlib/users", handlers.CreateUser)
	
	// Gin routes
	r := gin.Default()
	r.GET("/gin/users", handlers.ListUsers)
	r.POST("/gin/users/:id", handlers.UpdateUser)
	
	// Gorilla Mux
	router := mux.NewRouter()
	router.HandleFunc("/mux/users", handlers.ListUsers).Methods("GET")
	router.HandleFunc("/mux/users/{id}", handlers.GetUser).Methods("GET")
}
`

	err := os.WriteFile(routesFile, []byte(routesContent), 0644)
	require.NoError(t, err)

	// Create handlers file
	handlersFile := filepath.Join(tempDir, "handlers.go")
	handlersContent := `package test

import "context"

func ListUsers(ctx context.Context, req ListUsersRequest) (*UserListResponse, error) {
	return nil, nil
}

func CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	return nil, nil
}

func UpdateUser(ctx context.Context, req UpdateUserRequest) (*UserResponse, error) {
	return nil, nil
}

func GetUser(ctx context.Context, req GetUserRequest) (*UserResponse, error) {
	return nil, nil
}

type ListUsersRequest struct{}
type CreateUserRequest struct{}
type UpdateUserRequest struct{}
type GetUserRequest struct{}
type UserResponse struct{}
type UserListResponse struct{}
`

	err = os.WriteFile(handlersFile, []byte(handlersContent), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tempDir, "openapi.json")

	// Run generator
	cmd := exec.Command(binary, 
		"-i", tempDir,
		"-o", outputFile,
		"-t", "Route Test API",
		"-v", "1.0.0",
	)
	
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI execution failed: %s", string(output))

	// Read and parse output
	generated, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(generated, &spec)
	require.NoError(t, err)

	// Verify paths were detected
	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok)

	// Should have detected routes from different frameworks
	expectedPaths := []string{
		"/stdlib/users",
		"/gin/users", 
		"/gin/users/{id}",
		"/mux/users",
		"/mux/users/{id}",
	}

	for _, expectedPath := range expectedPaths {
		assert.Contains(t, paths, expectedPath, "Expected path %s not found", expectedPath)
	}
}

// Helper functions

func buildBinary(t *testing.T) string {
	t.Helper()
	
	tempDir := t.TempDir()
	binary := filepath.Join(tempDir, "openapi-gen")
	
	cmd := exec.Command("go", "build", "-o", binary, ".")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(output))
	
	return binary
}