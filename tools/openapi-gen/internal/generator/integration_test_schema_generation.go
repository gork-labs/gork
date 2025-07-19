package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaGeneration(t *testing.T) {
	tests := []struct {
		name       string
		structCode string
		expected   func(*testing.T, map[string]*Schema)
	}{
		{
			name: "basic struct with all field types",
			structCode: `
package models

import "time"

type TestStruct struct {
	// String fields
	Name        string  ` + "`json:\"name\" validate:\"required,max=100\"`" + `
	Description *string ` + "`json:\"description,omitempty\" validate:\"omitempty,max=500\"`" + `
	
	// Numeric fields
	Age    int     ` + "`json:\"age\" validate:\"gte=0,lte=150\"`" + `
	Score  float64 ` + "`json:\"score\" validate:\"gte=0,lte=100\"`" + `
	Points *int    ` + "`json:\"points,omitempty\" validate:\"omitempty,gte=0\"`" + `
	
	// Boolean
	Active bool ` + "`json:\"active\"`" + `
	
	// Arrays and slices
	Tags    []string ` + "`json:\"tags,omitempty\" validate:\"omitempty,dive,alphanum\"`" + `
	Numbers []int    ` + "`json:\"numbers,omitempty\" validate:\"omitempty,min=1,max=10\"`" + `
	
	// Time
	CreatedAt time.Time  ` + "`json:\"createdAt\"`" + `
	UpdatedAt *time.Time ` + "`json:\"updatedAt,omitempty\"`" + `
	
	// Embedded/nested
	Profile UserProfile ` + "`json:\"profile\"`" + `
	
	// Map
	Metadata map[string]interface{} ` + "`json:\"metadata,omitempty\"`" + `
}

type UserProfile struct {
	Bio      string ` + "`json:\"bio,omitempty\" validate:\"omitempty,max=1000\"`" + `
	Location string ` + "`json:\"location,omitempty\" validate:\"omitempty,max=100\"`" + `
}
`,
			expected: func(t *testing.T, schemas map[string]*Schema) {
				require.Contains(t, schemas, "TestStruct")
				require.Contains(t, schemas, "UserProfile")

				testSchema := schemas["TestStruct"]
				assert.Equal(t, "object", testSchema.Type)

				// Check required fields
				assert.Contains(t, testSchema.Required, "name")
				assert.NotContains(t, testSchema.Required, "description") // Optional pointer
				assert.NotContains(t, testSchema.Required, "points")      // Optional pointer

				// Check string properties
				nameProp := testSchema.Properties["name"]
				assert.Equal(t, "string", nameProp.Type)
				assert.Equal(t, 100, *nameProp.MaxLength)

				descProp := testSchema.Properties["description"]
				assert.Equal(t, "string", descProp.Type)
				assert.True(t, descProp.Nullable)
				assert.Equal(t, 500, *descProp.MaxLength)

				// Check numeric properties
				ageProp := testSchema.Properties["age"]
				assert.Equal(t, "integer", ageProp.Type)
				assert.Equal(t, 0.0, *ageProp.Minimum)
				assert.Equal(t, 150.0, *ageProp.Maximum)

				scoreProp := testSchema.Properties["score"]
				assert.Equal(t, "number", scoreProp.Type)
				assert.Equal(t, 0.0, *scoreProp.Minimum)
				assert.Equal(t, 100.0, *scoreProp.Maximum)

				// Check array properties
				tagsProp := testSchema.Properties["tags"]
				assert.Equal(t, "array", tagsProp.Type)
				assert.Equal(t, "string", tagsProp.Items.Type)
				assert.Equal(t, `^[a-zA-Z0-9]+$`, tagsProp.Items.Pattern)

				numbersProp := testSchema.Properties["numbers"]
				assert.Equal(t, "array", numbersProp.Type)
				assert.Equal(t, "integer", numbersProp.Items.Type)
				assert.Equal(t, 1, *numbersProp.MinItems)
				assert.Equal(t, 10, *numbersProp.MaxItems)

				// Check time properties
				createdProp := testSchema.Properties["createdAt"]
				assert.Equal(t, "string", createdProp.Type)
				assert.Equal(t, "date-time", createdProp.Format)

				updatedProp := testSchema.Properties["updatedAt"]
				assert.Equal(t, "string", updatedProp.Type)
				assert.Equal(t, "date-time", updatedProp.Format)
				assert.True(t, updatedProp.Nullable)

				// Check nested object
				profileProp := testSchema.Properties["profile"]
				assert.Equal(t, "#/components/schemas/UserProfile", profileProp.Ref)

				// Check map property
				metadataProp := testSchema.Properties["metadata"]
				assert.Equal(t, "object", metadataProp.Type)
				assert.NotNil(t, metadataProp.AdditionalProperties)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test.go")

			err := os.WriteFile(filePath, []byte(tt.structCode), 0644)
			require.NoError(t, err)

			// Parse with generator
			gen := New("Test API", "1.0.0")
			err = gen.ParseDirectories([]string{tempDir})
			require.NoError(t, err)

			// Generate the spec to trigger schema generation
			spec := gen.Generate()
			require.NotNil(t, spec)

			// Get generated schemas
			tt.expected(t, spec.Components.Schemas)
		})
	}
}
