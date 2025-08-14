package api

import (
	"net/http"
	"reflect"
	"testing"
)

// dummyHTTPHandler used to create a stable function pointer
func dummyHTTPHandler(w http.ResponseWriter, r *http.Request) {}

func TestGetOriginalWebhookHandler_DefaultBranch(t *testing.T) {
	// Prepare: manually insert non-registry entry for this handler pointer
	h := http.HandlerFunc(dummyHTTPHandler)
	ptr := reflect.ValueOf(h).Pointer()
	webhookHandlerRegistry[ptr] = "raw-entry"

	got := GetOriginalWebhookHandler(h)
	if got != "raw-entry" {
		t.Fatalf("expected raw-entry, got %#v", got)
	}
}

func TestGetWebhookRouteMetadata_NonRegistryEntry(t *testing.T) {
	h := http.HandlerFunc(dummyHTTPHandler)
	// Ensure default-branch mapping exists
	ptr := reflect.ValueOf(h).Pointer()
	webhookHandlerRegistry[ptr] = "raw-entry"

	pinfo, events := GetWebhookRouteMetadata(h)
	if pinfo != nil || len(events) != 0 {
		t.Fatalf("expected nil provider info and no events, got %#v, %#v", pinfo, events)
	}
}
