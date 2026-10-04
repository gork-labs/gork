package gorilla

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	muxpkg "github.com/gorilla/mux"
	"github.com/gork-labs/gork/pkg/api"
)

type streamTestRequest struct {
	Path struct {
		Topic string `gork:"topic" validate:"required"`
	}
}

type streamTestTopic struct {
	Name string `gork:"name"`
}

type streamTestEvents struct {
	Topic *streamTestTopic `gork:"topic"`
}

func TestStreamRouteFlushesEvents(t *testing.T) {
	mux := muxpkg.NewRouter()
	router := NewRouter(mux)
	handler := func(ctx context.Context, req streamTestRequest, stream *api.Stream[streamTestEvents]) error {
		if err := stream.Send(streamTestEvents{Topic: &streamTestTopic{Name: req.Path.Topic}}); err != nil {
			return err
		}
		<-ctx.Done()
		return nil
	}
	router.Get("/live/{topic}", handler)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/live/feed")
	if err != nil {
		t.Fatalf("GET /live/feed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}

	want := "event: topic\ndata: {\"name\":\"feed\"}\n\n"
	frame := make([]byte, len(want))
	if _, err := io.ReadFull(resp.Body, frame); err != nil {
		t.Fatalf("read event before the handler returns: %v", err)
	}
	if string(frame) != want {
		t.Errorf("unexpected event %q", frame)
	}
}
