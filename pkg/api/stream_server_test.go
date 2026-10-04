package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func readStreamFrame(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	var frame strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		if line == "\n" {
			return frame.String()
		}
		frame.WriteString(line)
	}
}

func TestStreamOverHTTPServer(t *testing.T) {
	release := make(chan struct{})
	disconnected := make(chan struct{})
	handler := func(ctx context.Context, _ streamTestRequest, stream *Stream[streamTestEvents]) error {
		if err := stream.Send(streamTestEvents{Row: &streamTestRow{ID: "1"}}); err != nil {
			return err
		}
		<-release
		time.Sleep(100 * time.Millisecond)
		if err := stream.Send(streamTestEvents{Done: &struct{}{}}); err != nil {
			return err
		}
		<-ctx.Done()
		close(disconnected)
		return ctx.Err()
	}
	httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, handler)

	server := httptest.NewUnstartedServer(httpHandler)
	server.Config.WriteTimeout = 50 * time.Millisecond
	server.Start()
	defer server.Close()

	resp, err := http.Get(server.URL + "/live?topic=feed")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}
	reader := bufio.NewReader(resp.Body)

	if frame := readStreamFrame(t, reader); frame != "event: row\ndata: {\"display_name\":\"\",\"id\":\"1\"}\n" {
		t.Errorf("unexpected first frame %q", frame)
	}
	close(release)
	if frame := readStreamFrame(t, reader); frame != "event: done\ndata: {}\n" {
		t.Errorf("unexpected frame after the write timeout %q", frame)
	}

	_ = resp.Body.Close()
	select {
	case <-disconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("handler context was not canceled after the client disconnected")
	}
}
