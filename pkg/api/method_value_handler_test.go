package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type methodValueHandlers struct{}

func (methodValueHandlers) ListWorkstreams(context.Context, struct{}) error { return nil }

func TestMethodValueHandlerNameAndDescription(t *testing.T) {
	dir := t.TempDir()
	source := "package x\n\n// ListWorkstreams lists the workstreams.\nfunc (h *Handlers) ListWorkstreams() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "handlers.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	extractor := NewDocExtractor()
	if err := extractor.ParseDirectory(dir); err != nil {
		t.Fatal(err)
	}

	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/workstreams", methodValueHandlers{}.ListWorkstreams)

	op := GenerateOpenAPIWithDocs(registry, extractor).Paths["/workstreams"].Get
	if op.OperationID != "ListWorkstreams" {
		t.Errorf("operationId = %q, want ListWorkstreams", op.OperationID)
	}
	if op.Description != "ListWorkstreams lists the workstreams." {
		t.Errorf("description = %q, want the doc comment of the method", op.Description)
	}
}
