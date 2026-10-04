package api

import (
	"context"
	"testing"
)

type methodValueHandlers struct{}

func (methodValueHandlers) ListWorkstreams(context.Context, struct{}) error { return nil }

func TestMethodValueHandlerNameAndDescription(t *testing.T) {
	source := "package api\n\n// ListWorkstreams lists the workstreams.\nfunc (methodValueHandlers) ListWorkstreams() {}\n"
	extractor := parseFixtures(t, map[string]string{"pkg/api/handlers.go": source})

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
