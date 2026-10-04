package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type streamDeadlineErrorWriter struct {
	*httptest.ResponseRecorder
}

func (streamDeadlineErrorWriter) SetWriteDeadline(time.Time) error {
	return errors.New("connection closed")
}

func serveTestStream(t *testing.T, w http.ResponseWriter, target string, handler interface{}) {
	t.Helper()
	httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, handler)
	httpHandler(w, httptest.NewRequest(http.MethodGet, target, nil))
}

func TestStreamHandlerWritesEvents(t *testing.T) {
	rec := httptest.NewRecorder()
	serveTestStream(t, rec, "/live?topic=feed", func(_ context.Context, req streamTestRequest, stream *Stream[streamTestEvents]) error {
		if err := stream.Send(streamTestEvents{Row: &streamTestRow{ID: req.Query.Topic}}); err != nil {
			return err
		}
		return stream.Send(streamTestEvents{Done: &struct{}{}})
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	wantHeaders := map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"X-Accel-Buffering": "no",
	}
	for name, want := range wantHeaders {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("expected header %s=%q, got %q", name, want, got)
		}
	}
	want := "event: row\ndata: {\"display_name\":\"\",\"id\":\"feed\"}\n\nevent: done\ndata: {}\n\n"
	if rec.Body.String() != want {
		t.Errorf("unexpected stream body:\n%q\nwant:\n%q", rec.Body.String(), want)
	}
}

func TestStreamHandlerRequestErrors(t *testing.T) {
	called := false
	handler := func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error {
		called = true
		return nil
	}

	t.Run("validation error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		serveTestStream(t, rec, "/live", handler)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected JSON error response, got Content-Type %q", ct)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		type badRequest struct {
			Query struct {
				Limit int `gork:"limit"`
			}
		}
		rec := httptest.NewRecorder()
		serveTestStream(t, rec, "/live?limit=abc", func(context.Context, badRequest, *Stream[streamTestEvents]) error {
			called = true
			return nil
		})

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
			t.Errorf("expected JSON error body, got %q", rec.Body.String())
		}
	})

	if called {
		t.Error("handler must not run when the request is invalid")
	}
}

func TestStreamHandlerWriteDeadlineError(t *testing.T) {
	rec := httptest.NewRecorder()
	called := false
	serveTestStream(t, streamDeadlineErrorWriter{rec}, "/live?topic=feed", func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error {
		called = true
		return nil
	})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
	if called {
		t.Error("handler must not run when the write deadline cannot be cleared")
	}
}

func TestStreamHandlerErrorEndsStream(t *testing.T) {
	rec := httptest.NewRecorder()
	serveTestStream(t, rec, "/live?topic=feed", func(_ context.Context, _ streamTestRequest, stream *Stream[streamTestEvents]) error {
		_ = stream.Send(streamTestEvents{Done: &struct{}{}})
		return errors.New("upstream failed")
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "event: done\ndata: {}\n\n" {
		t.Errorf("expected the stream to end after the sent event, got %q", rec.Body.String())
	}
}

func TestStreamHandlerKeepAlive(t *testing.T) {
	interval := keepAliveInterval
	keepAliveInterval = time.Millisecond
	t.Cleanup(func() { keepAliveInterval = interval })

	rec := httptest.NewRecorder()
	serveTestStream(t, rec, "/live?topic=feed", func(_ context.Context, _ streamTestRequest, stream *Stream[streamTestEvents]) error {
		for i := 0; i < 20; i++ {
			if err := stream.Send(streamTestEvents{Done: &struct{}{}}); err != nil {
				return err
			}
			time.Sleep(time.Millisecond)
		}
		return nil
	})

	body := rec.Body.String()
	if !strings.Contains(body, ": ping\n\n") {
		t.Errorf("expected keep-alive comments in %q", body)
	}
	for _, frame := range strings.Split(strings.TrimSuffix(body, "\n\n"), "\n\n") {
		if frame != ": ping" && frame != "event: done\ndata: {}" {
			t.Errorf("unexpected frame %q", frame)
		}
	}
}
