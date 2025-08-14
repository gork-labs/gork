package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// sample types for user metadata
type sampleUserMeta struct {
	Note string `json:"note"`
}

// minimal webhook request and provider for metadata tests
type metaReq struct{}

func (metaReq) WebhookRequest() {}

type metaProvider struct{}

func (p *metaProvider) ParseRequest(req metaReq) (WebhookEvent, error) {
	return WebhookEvent{Type: "evt.x"}, nil
}
func (p *metaProvider) SuccessResponse() interface{} { return map[string]any{"ok": true} }
func (p *metaProvider) ErrorResponse(err error) interface{} {
	return map[string]any{"error": err.Error()}
}
func (p *metaProvider) GetValidEventTypes() []string { return []string{"evt.x", "evt.y"} }
func (p *metaProvider) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "Meta", Website: "w", DocsURL: "d"}
}

func TestBuildWebhookEventsMetadata_Branches(t *testing.T) {
	spec := &OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}

	// Prepare a DocExtractor with a documented function name
	tmp := t.TempDir()
	src := []byte("package tmp\n\n// MyDocHandler does something useful\nfunc MyDocHandler() {}\n")
	if err := os.WriteFile(filepath.Join(tmp, "doc.go"), src, 0o600); err != nil {
		t.Fatalf("write temp doc: %v", err)
	}
	extractor := NewDocExtractor()
	if err := extractor.ParseDirectory(tmp); err != nil {
		t.Fatalf("parse temp dir: %v", err)
	}

	gen := NewConventionOpenAPIGenerator(spec, extractor)

	// Build route with two handlers: one with user meta, one without
	route := &RouteInfo{
		WebhookHandlersMeta: []RegisteredEventHandler{
			{
				EventType:           "evt.x",
				HandlerName:         "MyDocHandler",
				UserMetadataType:    reflect.TypeOf((*sampleUserMeta)(nil)),
				ProviderPayloadType: reflect.TypeOf((*struct{})(nil)),
			},
			{
				EventType:   "evt.y",
				HandlerName: "",
				// No user meta type -> exercise nil branch
			},
		},
	}

	entries := gen.buildWebhookEventsMetadata(route, spec.Components)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Validate first entry has description and schema
	if entries[0]["event"] != "evt.x" {
		t.Fatalf("unexpected event: %v", entries[0]["event"])
	}
	if desc, _ := entries[0]["description"].(string); desc == "" {
		t.Fatalf("expected non-empty description for documented handler")
	}
	if sch, ok := entries[0]["userPayloadSchema"].(*Schema); !ok || sch == nil {
		t.Fatalf("expected userPayloadSchema to be present and a *Schema, got %T", entries[0]["userPayloadSchema"])
	}

	// Second entry should not include description or schema
	if _, has := entries[1]["description"]; has {
		t.Fatalf("did not expect description on second entry")
	}
	if _, has := entries[1]["userPayloadSchema"]; has {
		t.Fatalf("did not expect userPayloadSchema on second entry")
	}
}

func TestGetWebhookHandlersMetadata_ReturnsRegistered(t *testing.T) {
	prov := &metaProvider{}
	// register a single handler
	h := WebhookHandlerFunc[metaReq](prov, WithEventHandler[sampleUserMeta, sampleUserMeta]("evt.x", func(ctx context.Context, p *sampleUserMeta, u *sampleUserMeta) error { return nil }))

	metas := GetWebhookHandlersMetadata(h)
	if len(metas) != 1 {
		t.Fatalf("expected 1 registered handler meta, got %d", len(metas))
	}
	if metas[0].EventType != "evt.x" || metas[0].HandlerName == "" || metas[0].UserMetadataType == nil {
		t.Fatalf("unexpected meta: %#v", metas[0])
	}
}

func TestBuildWebhookEventsMetadata_NoDocFound(t *testing.T) {
	spec := &OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}
	gen := NewConventionOpenAPIGenerator(spec, NewDocExtractor()) // extractor has no entries

	route := &RouteInfo{
		WebhookHandlersMeta: []RegisteredEventHandler{
			{
				EventType:           "evt.z",
				HandlerName:         "NonExistingFuncName",
				UserMetadataType:    reflect.TypeOf((*sampleUserMeta)(nil)),
				ProviderPayloadType: reflect.TypeOf((*struct{})(nil)),
			},
		},
	}
	entries := gen.buildWebhookEventsMetadata(route, spec.Components)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if _, ok := entries[0]["description"]; ok {
		t.Fatalf("did not expect description when doc not found")
	}
}

func TestGetWebhookHandlersMetadata_UnregisteredReturnsNil(t *testing.T) {
	// Plain http handler not registered via WebhookHandlerFunc should have no metadata
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	if metas := GetWebhookHandlersMetadata(h); metas != nil {
		t.Fatalf("expected nil metadata for unregistered handler, got %#v", metas)
	}
}

func TestBuildWebhookEventsMetadata_NoHandlersReturnsNil(t *testing.T) {
	spec := &OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}
	gen := NewConventionOpenAPIGenerator(spec, NewDocExtractor())
	route := &RouteInfo{WebhookHandlersMeta: nil}
	entries := gen.buildWebhookEventsMetadata(route, spec.Components)
	if entries != nil {
		t.Fatalf("expected nil entries when no handlers, got %#v", entries)
	}
}

func TestBuildWebhookEventsMetadata_NoExtractorNoDescription(t *testing.T) {
	spec := &OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}
	gen := NewConventionOpenAPIGenerator(spec, nil)
	route := &RouteInfo{WebhookHandlersMeta: []RegisteredEventHandler{{EventType: "evt.nd", HandlerName: "SomeFunc", UserMetadataType: reflect.TypeOf((*sampleUserMeta)(nil))}}}
	entries := gen.buildWebhookEventsMetadata(route, spec.Components)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if _, ok := entries[0]["description"]; ok {
		t.Fatalf("did not expect description when extractor is nil")
	}
}
