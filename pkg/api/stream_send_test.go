package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type streamTestRow struct {
	ID   string `gork:"id"`
	Name string `gork:"display_name"`
}

type streamTestEvents struct {
	Row  *streamTestRow `gork:"row"`
	Done *struct{}      `gork:"done"`
}

type streamFailingPayload struct{}

func (streamFailingPayload) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failed")
}

type streamInvalidPayload struct{}

func (streamInvalidPayload) MarshalJSON() ([]byte, error) {
	return []byte("{not json"), nil
}

type streamIndentedPayload struct{}

func (streamIndentedPayload) MarshalJSON() ([]byte, error) {
	return []byte("{\n  \"text\": \"a\\nb\"\n}"), nil
}

type streamCustomEvents struct {
	Failing  *streamFailingPayload  `gork:"failing"`
	Invalid  *streamInvalidPayload  `gork:"invalid"`
	Indented *streamIndentedPayload `gork:"indented"`
}

type streamFailingWriter struct {
	http.ResponseWriter
}

func (streamFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type streamNoFlushWriter struct {
	header http.Header
	body   strings.Builder
}

func (w *streamNoFlushWriter) Header() http.Header         { return w.header }
func (w *streamNoFlushWriter) Write(b []byte) (int, error) { return w.body.Write(b) }
func (w *streamNoFlushWriter) WriteHeader(int)             {}

func newTestStream[E any](w http.ResponseWriter) *Stream[E] {
	return &Stream[E]{out: &eventWriter{w: w, rc: http.NewResponseController(w)}}
}

func TestStreamSendWritesEventFrame(t *testing.T) {
	rec := httptest.NewRecorder()
	stream := newTestStream[streamTestEvents](rec)

	if err := stream.Send(streamTestEvents{Row: &streamTestRow{ID: "1", Name: "Ann"}}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if err := stream.Send(streamTestEvents{Done: &struct{}{}}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	want := "event: row\ndata: {\"display_name\":\"Ann\",\"id\":\"1\"}\n\n" +
		"event: done\ndata: {}\n\n"
	if rec.Body.String() != want {
		t.Errorf("unexpected stream body:\n%q\nwant:\n%q", rec.Body.String(), want)
	}
	if !rec.Flushed {
		t.Error("expected Send to flush the response")
	}
}

func TestStreamSendRequiresExactlyOneField(t *testing.T) {
	stream := newTestStream[streamTestEvents](httptest.NewRecorder())

	tests := []struct {
		name  string
		event streamTestEvents
		want  string
	}{
		{"no field set", streamTestEvents{}, "stream event must set exactly one field, got 0"},
		{"two fields set", streamTestEvents{Row: &streamTestRow{}, Done: &struct{}{}}, "stream event must set exactly one field, got 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := stream.Send(tt.event)
			if err == nil || err.Error() != tt.want {
				t.Errorf("expected error %q, got %v", tt.want, err)
			}
		})
	}
}

func TestStreamSendCompactsMultiLineJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	stream := newTestStream[streamCustomEvents](rec)

	if err := stream.Send(streamCustomEvents{Indented: &streamIndentedPayload{}}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	want := "event: indented\ndata: {\"text\":\"a\\nb\"}\n\n"
	if rec.Body.String() != want {
		t.Errorf("unexpected stream body:\n%q\nwant:\n%q", rec.Body.String(), want)
	}
}

func TestStreamSendEncodingErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	stream := newTestStream[streamCustomEvents](rec)

	if err := stream.Send(streamCustomEvents{Failing: &streamFailingPayload{}}); err == nil {
		t.Error("expected marshal error")
	}
	if err := stream.Send(streamCustomEvents{Invalid: &streamInvalidPayload{}}); err == nil {
		t.Error("expected compact error for invalid JSON")
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected no output after encoding errors, got %q", rec.Body.String())
	}
}

func TestStreamSendWriteErrors(t *testing.T) {
	t.Run("write error", func(t *testing.T) {
		stream := newTestStream[streamTestEvents](streamFailingWriter{httptest.NewRecorder()})
		if err := stream.Send(streamTestEvents{Done: &struct{}{}}); err == nil || err.Error() != "write failed" {
			t.Errorf("expected write error, got %v", err)
		}
	})

	t.Run("flush not supported", func(t *testing.T) {
		stream := newTestStream[streamTestEvents](&streamNoFlushWriter{header: http.Header{}})
		if err := stream.Send(streamTestEvents{Done: &struct{}{}}); !errors.Is(err, http.ErrNotSupported) {
			t.Errorf("expected http.ErrNotSupported, got %v", err)
		}
	})
}
