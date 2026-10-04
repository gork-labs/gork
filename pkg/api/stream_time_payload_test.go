package api

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

type streamTimeTick struct {
	At     time.Time  `gork:"at"`
	PrevAt *time.Time `gork:"prev_at"`
}

type streamTimeEvents struct {
	Tick *streamTimeTick `gork:"tick"`
}

func TestStreamSendEncodesTimeAsRFC3339(t *testing.T) {
	rec := httptest.NewRecorder()
	stream := newTestStream[streamTimeEvents](rec)
	at := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	if err := stream.Send(streamTimeEvents{Tick: &streamTimeTick{At: at}}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	want := "event: tick\ndata: {\"at\":\"2024-01-02T03:04:05Z\",\"prev_at\":null}\n\n"
	if rec.Body.String() != want {
		t.Errorf("unexpected stream body:\n%q\nwant:\n%q", rec.Body.String(), want)
	}
}

func TestStreamOpenAPITimePayloadSchema(t *testing.T) {
	registry := newStreamTestRegistry(t, func(context.Context, streamTestRequest, *Stream[streamTimeEvents]) error { return nil })

	spec := GenerateOpenAPI(registry)

	data := spec.Paths["/routea"].Get.Responses["200"].Content["text/event-stream"].ItemSchema.OneOf[0].Properties["data"]
	if data.ContentSchema.Ref != "#/components/schemas/streamTimeTick" {
		t.Fatalf("contentSchema = %+v, want a reference to streamTimeTick", data.ContentSchema)
	}
	tick := spec.Components.Schemas["streamTimeTick"]
	if got := schemaJSON(t, tick.Properties["at"]); got != `{"type":"string","format":"date-time"}` {
		t.Errorf("at schema = %s", got)
	}
	if got := schemaJSON(t, tick.Properties["prev_at"]); got != `{"type":["string","null"],"format":"date-time"}` {
		t.Errorf("prev_at schema = %s", got)
	}
}
