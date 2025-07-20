package api

import (
	"reflect"
	"testing"
)

type Req struct{}

type Resp struct{}

func TestGenerateOpenAPIWithDocs(t *testing.T) {
	registry := NewRouteRegistry()
	info := &RouteInfo{
		Method:       "GET",
		Path:         "/foo",
		HandlerName:  "GetFoo",
		RequestType:  reflect.TypeOf(Req{}),
		ResponseType: reflect.TypeOf(Resp{}),
	}
	registry.Register(info)

	extractor := NewDocExtractor()
	// fake doc entries
	extractor.docs["GetFoo"] = Documentation{Description: "Returns foo."}

	spec := GenerateOpenAPIWithDocs(registry, extractor)
	p := spec.Paths["/foo"]
	if p == nil || p.Get == nil {
		t.Fatalf("missing get path")
	}
	if p.Get.Description != "Returns foo." {
		t.Errorf("description not propagated: %q", p.Get.Description)
	}
}
