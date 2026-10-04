package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type streamResumeRequest struct {
	Headers struct {
		LastEventID string `gork:"Last-Event-ID"`
	}
}

func resumeStream(_ context.Context, req streamResumeRequest, stream *Stream[streamTestEvents]) error {
	last, _ := strconv.Atoi(req.Headers.LastEventID)
	next := strconv.Itoa(last + 1)
	return stream.SendWithID(next, streamTestEvents{Row: &streamTestRow{ID: next}})
}

func TestStreamSendWithIDResumesAfterLastEventID(t *testing.T) {
	httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, resumeStream)
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	req.Header.Set("Last-Event-ID", "41")
	rec := httptest.NewRecorder()

	httpHandler(rec, req)

	if want := "id: 42\nevent: row\ndata: {\"display_name\":\"\",\"id\":\"42\"}\n\n"; rec.Body.String() != want {
		t.Errorf("body = %q, want %q", rec.Body.String(), want)
	}
}

func TestStreamSendWithIDRejectsLineBreaks(t *testing.T) {
	rec := httptest.NewRecorder()
	stream := newTestStream[streamTestEvents](rec)

	for _, id := range []string{"1\nevent: fake", "1\r", "1\x00"} {
		if err := stream.SendWithID(id, streamTestEvents{Done: &struct{}{}}); err == nil {
			t.Errorf("SendWithID(%q) gave no error", id)
		}
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want nothing written", rec.Body.String())
	}
}

func TestStreamResumeHeaderIsParameter(t *testing.T) {
	registry := newStreamTestRegistry(t, resumeStream)

	params := GenerateOpenAPI(registry).Paths["/routea"].Get.Parameters
	if len(params) != 1 || params[0].Name != "Last-Event-ID" || params[0].In != "header" {
		t.Errorf("parameters = %+v, want the Last-Event-ID header", params)
	}
}
