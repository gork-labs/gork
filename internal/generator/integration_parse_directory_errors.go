package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDirectoryErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string
		expectError bool
		errorText   string
	}{
		{
			name: "invalid Go syntax",
			setupFiles: map[string]string{
				"invalid.go": `
package invalid

type User struct {
	ID string ` + "`json:\"id\"" + ` // Missing closing backtick
}
`,
			},
			expectError: true,
		},
		{
			name: "empty directory",
			setupFiles: map[string]string{
				"empty.txt": "not a go file",
			},
			expectError: false, // Should not error on non-Go files
		},
		{
			name:        "non-existent directory",
			setupFiles:  map[string]string{}, // Empty, will test non-existent dir
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "non-existent directory" {
				gen := New("Test API", "1.0.0")
				err := gen.ParseDirectories([]string{"/path/that/does/not/exist"})
				assert.Error(t, err)
				return
			}

			// Create temporary directory structure
			tempDir := t.TempDir()

			// Create files
			for filePath, content := range tt.setupFiles {
				fullPath := filepath.Join(tempDir, filePath)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)

				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}

			// Test parsing
			gen := New("Test API", "1.0.0")
			err := gen.ParseDirectories([]string{tempDir})

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
