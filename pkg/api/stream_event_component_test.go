package api

import (
	"context"
	"testing"
)

type streamAnonymousEvents = struct {
	Row *streamTestRow `gork:"row"`
}

func TestStreamEventComponent(t *testing.T) {
	named := func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error { return nil }
	anonymous := func(context.Context, streamTestRequest, *Stream[streamAnonymousEvents]) error { return nil }
	spec := GenerateOpenAPI(newStreamTestRegistry(t, named, named, anonymous))

	for _, path := range []string{"/routea", "/routeb"} {
		if got := spec.Paths[path].Get.Responses["200"].Content["text/event-stream"].ItemSchema.Ref; got != "#/components/schemas/streamTestEvents" {
			t.Errorf("%s itemSchema ref = %q, want the streamTestEvents component", path, got)
		}
	}
	if events := spec.Components.Schemas["streamTestEvents"]; events == nil || len(events.OneOf) != 2 {
		t.Errorf("streamTestEvents = %+v, want a oneOf of 2 events", events)
	}
	if item := spec.Paths["/routec"].Get.Responses["200"].Content["text/event-stream"].ItemSchema; item.Ref != "" || len(item.OneOf) != 1 {
		t.Errorf("anonymous itemSchema = %+v, want an inline oneOf of 1 event", item)
	}
}
