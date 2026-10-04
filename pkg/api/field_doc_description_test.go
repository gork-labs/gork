package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// docEnvelope wraps a response payload.
type docEnvelope[T any] struct {
	// Data is the payload of the response
	Data T `gork:"data"`
}

type docHealth struct {
	Status string `gork:"status"`
}

type docWorkstream struct {
	ID string `gork:"id"`
}

// docFields has a field of each composite kind.
type docFields struct {
	// Tags are the labels of the item
	Tags []string `gork:"tags"`
	// Scores are the last three scores
	Scores [3]int `gork:"scores"`
	// Labels map a key to a value
	Labels map[string]string `gork:"labels"`
	// Owner is the workstream that owns the item
	Owner *docWorkstream `gork:"owner"`
	Notes []string       `gork:"notes"`
}

const docFieldsSource = `package fixtures

// docEnvelope wraps a response payload.
type docEnvelope[T any] struct {
	// Data is the payload of the response
	Data T ` + "`gork:\"data\"`" + `
}

// docFields has a field of each composite kind.
type docFields struct {
	// Tags are the labels of the item
	Tags []string ` + "`gork:\"tags\"`" + `
	// Scores are the last three scores
	Scores [3]int ` + "`gork:\"scores\"`" + `
	// Labels map a key to a value
	Labels map[string]string ` + "`gork:\"labels\"`" + `
	// Owner is the workstream that owns the item
	Owner *docWorkstream ` + "`gork:\"owner\"`" + `
	Notes []string ` + "`gork:\"notes\"`" + `
}
`

type docHealthResponse struct {
	Body docEnvelope[docHealth]
}

type docWorkstreamsResponse struct {
	Body docEnvelope[[]docWorkstream]
}

type docFieldsResponse struct {
	Body docFields
}

func TestFieldDocIsDescriptionForEveryFieldType(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(docFieldsSource), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	extractor := NewDocExtractor()
	if err := extractor.ParseDirectory(dir); err != nil {
		t.Fatalf("parse: %v", err)
	}

	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/health", func(context.Context, struct{}) (*docHealthResponse, error) { return nil, nil })
	router.Get("/workstreams", func(context.Context, struct{}) (*docWorkstreamsResponse, error) { return nil, nil })
	router.Get("/fields", func(context.Context, struct{}) (*docFieldsResponse, error) { return nil, nil })

	schemas := GenerateOpenAPIWithDocs(registry, extractor).Components.Schemas

	tests := []struct {
		component, property, want string
	}{
		{"docEnvelope_docHealth", "data", "Data is the payload of the response"},
		{"docEnvelope_Array_docWorkstream", "data", "Data is the payload of the response"},
		{"docFields", "tags", "Tags are the labels of the item"},
		{"docFields", "scores", "Scores are the last three scores"},
		{"docFields", "labels", "Labels map a key to a value"},
		{"docFields", "owner", "Owner is the workstream that owns the item"},
		{"docFields", "notes", "Array of string"},
	}
	for _, tt := range tests {
		t.Run(tt.component+"."+tt.property, func(t *testing.T) {
			if got := schemas[tt.component].Properties[tt.property].Description; got != tt.want {
				t.Errorf("description = %q, want %q", got, tt.want)
			}
		})
	}
}
