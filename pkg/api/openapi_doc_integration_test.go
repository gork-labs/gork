package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// parseFixtures writes the files into a temporary copy of the module
// github.com/gork-labs/gork and parses it. Thus a fixture in pkg/api declares
// the doc of a type of this test package.
func parseFixtures(t *testing.T, files map[string]string) *DocExtractor {
	t.Helper()
	extractor := NewDocExtractor()
	if err := extractor.ParseDirectory(writeFixtures(t, files)); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return extractor
}

// writeFixtures writes the files into a temporary copy of the module
// github.com/gork-labs/gork and returns its directory.
func writeFixtures(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module github.com/gork-labs/gork\n"
	for name, src := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// docRow is a row with a field that has the name of a property of ErrorResponse.
type docRow struct {
	// Error is true for a row that failed
	Error bool `gork:"error"`
}

type docRowsResponse struct {
	Body []docRow
}

const docRowSource = `package api

// docRow is a row with a field that has the name of a property of ErrorResponse.
type docRow struct {
	// Error is true for a row that failed
	Error bool ` + "`gork:\"error\"`" + `
}
`

const otherDocRowSource = `package other

// docRow is a type of another package.
type docRow struct {
	// Error is the doc of another package
	Error bool ` + "`gork:\"error\"`" + `
}
`

func docRowsRegistry() *RouteRegistry {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/rows", func(context.Context, struct{}) (*docRowsResponse, error) { return nil, nil })
	return registry
}

var docRowFixtures = map[string]string{
	"pkg/api/fixture.go":   docRowSource,
	"pkg/other/fixture.go": otherDocRowSource,
}

func TestFieldDocAppliesOnlyToItsType(t *testing.T) {
	t.Setenv("GORK_SOURCE", writeFixtures(t, docRowFixtures))
	exported := GenerateOpenAPI(docRowsRegistry())
	t.Setenv("GORK_SOURCE", "")

	specs := map[string]*OpenAPISpec{
		"GenerateOpenAPIWithDocs":          GenerateOpenAPIWithDocs(docRowsRegistry(), parseFixtures(t, docRowFixtures)),
		"GenerateOpenAPI with GORK_SOURCE": exported,
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			schemas := spec.Components.Schemas
			tests := []struct {
				component, want string
			}{
				{"docRow", "Error is true for a row that failed"},
				{"ErrorResponse", "Error message"},
				{"ValidationErrorResponse", "Error message"},
			}
			for _, tt := range tests {
				if got := schemas[tt.component].Properties["error"].Description; got != tt.want {
					t.Errorf("%s.error description = %q, want %q", tt.component, got, tt.want)
				}
			}
			if got := schemas["docRow"].Description; got != "docRow is a row with a field that has the name of a property of ErrorResponse." {
				t.Errorf("docRow description = %q", got)
			}
		})
	}
}

func TestSpecHasNoInternalData(t *testing.T) {
	t.Setenv("GORK_SOURCE", writeFixtures(t, docRowFixtures))
	out, err := json.Marshal(GenerateOpenAPI(docRowsRegistry()))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, internal := range []string{"x-gork-", "github.com/"} {
		if strings.Contains(string(out), internal) {
			t.Errorf("spec contains %q: %s", internal, out)
		}
	}
}

func TestGenerateOpenAPIPanicsForSourceWithoutGoMod(t *testing.T) {
	t.Setenv("GORK_SOURCE", t.TempDir())
	defer func() {
		if msg := fmt.Sprint(recover()); !strings.Contains(msg, "no go.mod file") {
			t.Errorf("panic = %q, want an error about the missing go.mod file", msg)
		}
	}()
	GenerateOpenAPI(docRowsRegistry())
}

// DocBase is embedded in docUser.
type DocBase struct {
	// ID is the id of the record
	ID string `gork:"id"`
}

type docUser struct {
	DocBase
	Name string `gork:"name"`
}

// docCreateUserRequest has an inline Body.
type docCreateUserRequest struct {
	Path struct {
		// Team is the team of the new user
		Team string `gork:"team"`
	}
	Body struct {
		DocBase
		// Name is the name of the new user
		Name string `gork:"name"`
	}
}

// docCreateUserResponse has an inline Body.
type docCreateUserResponse struct {
	Body struct {
		// User is the new user
		User docUser `gork:"user"`
		// Team is the team of the new user
		Team string `gork:"team"`
	}
}

const docUserSource = `package api

// DocBase is embedded in docUser.
type DocBase struct {
	// ID is the id of the record
	ID string ` + "`gork:\"id\"`" + `
}

// docUser is a user.
type docUser struct {
	DocBase
	Name string ` + "`gork:\"name\"`" + `
}

// docCreateUserRequest has an inline Body.
type docCreateUserRequest struct {
	Path struct {
		// Team is the team of the new user
		Team string ` + "`gork:\"team\"`" + `
	}
	Body struct {
		DocBase
		// Name is the name of the new user
		Name string ` + "`gork:\"name\"`" + `
	}
}

// docCreateUserResponse has an inline Body.
type docCreateUserResponse struct {
	Body struct {
		// User is the new user
		User docUser ` + "`gork:\"user\"`" + `
		// Team is the team of the new user
		Team string ` + "`gork:\"team\"`" + `
	}
}

// docCreateUser creates a user.
func docCreateUser() {}
`

func docCreateUser(context.Context, docCreateUserRequest) (*docCreateUserResponse, error) {
	return nil, nil
}

func TestDocsOfEmbeddedTypesAndSections(t *testing.T) {
	extractor := parseFixtures(t, map[string]string{"pkg/api/fixture.go": docUserSource})

	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Post("/teams/{team}/users", docCreateUser)
	spec := GenerateOpenAPIWithDocs(registry, extractor)
	schemas := spec.Components.Schemas

	tests := []struct {
		name, got, want string
	}{
		{"docUser", schemas["docUser"].Description, "docUser is a user."},
		{"docUser.id", schemas["docUser"].Properties["id"].Description, "ID is the id of the record"},
		{"docCreateUserBody", schemas["docCreateUserBody"].Description, ""},
		{"docCreateUserBody.id", schemas["docCreateUserBody"].Properties["id"].Description, "ID is the id of the record"},
		{"docCreateUserBody.name", schemas["docCreateUserBody"].Properties["name"].Description, "Name is the name of the new user"},
		{"docCreateUserResponse", schemas["docCreateUserResponse"].Description, "docCreateUserResponse has an inline Body."},
		{"docCreateUserResponse.user", schemas["docCreateUserResponse"].Properties["user"].Description, "User is the new user"},
		{"operation", spec.Paths["/teams/{team}/users"].Post.Description, "docCreateUser creates a user."},
		{"parameter team", spec.Paths["/teams/{team}/users"].Post.Parameters[0].Description, "Team is the team of the new user"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestEnrichSchemaWithTypeDocs(t *testing.T) {
	extractor := NewDocExtractor()
	extractor.docs["example.com/app.A"] = Documentation{
		Description: "A is a type",
		Fields:      map[string]FieldDoc{"x": {Description: "x of A"}},
	}
	extractor.docs["example.com/app.B"] = Documentation{
		Description: "B is a type",
		Fields:      map[string]FieldDoc{"x": {Description: "x of B"}, "y": {Description: "y of B"}},
	}

	newSchema := func() *Schema {
		return &Schema{
			Description: "generated",
			Properties:  map[string]*Schema{"x": {Description: "Array of X"}, "y": {}, "z": {Description: "Array of Z"}},
		}
	}

	t.Run("no doc types", func(t *testing.T) {
		schema := newSchema()
		enrichSchemaWithTypeDocs(schema, nil, extractor)
		if schema.Description != "generated" || schema.Properties["x"].Description != "Array of X" {
			t.Errorf("schema changed: %+v", schema)
		}
	})

	t.Run("first type documents the schema", func(t *testing.T) {
		schema := newSchema()
		enrichSchemaWithTypeDocs(schema, []string{"example.com/app.A", "example.com/app.B"}, extractor)
		tests := []struct {
			name, got, want string
		}{
			{"schema", schema.Description, "A is a type"},
			{"x", schema.Properties["x"].Description, "x of A"},
			{"y", schema.Properties["y"].Description, "y of B"},
			{"z", schema.Properties["z"].Description, "Array of Z"},
		}
		for _, tt := range tests {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		}
	})

	t.Run("first type without doc", func(t *testing.T) {
		schema := newSchema()
		enrichSchemaWithTypeDocs(schema, []string{"example.com/app.Missing"}, extractor)
		if schema.Description != "generated" {
			t.Errorf("description = %q, want generated", schema.Description)
		}
	})
}
