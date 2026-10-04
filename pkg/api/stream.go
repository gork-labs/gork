package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/gork-labs/gork/pkg/gorkson"
)

// keepAliveInterval is the time between keep-alive comments on an open stream.
var keepAliveInterval = 15 * time.Second

// Stream sends Server-Sent Events from a stream handler:
// func(ctx context.Context, req Request, stream *api.Stream[E]) error.
//
// E is a struct with one pointer field for each event type. The gork tag of
// a field is the SSE event name. An event without a payload uses *struct{}.
type Stream[E any] struct {
	out *eventWriter
}

// Send writes one event to the client and flushes it. Exactly one field of e
// must be set. Call Send only from the handler goroutine.
func (s *Stream[E]) Send(e E) error {
	name, payload, err := streamEvent(reflect.ValueOf(e))
	if err != nil {
		return err
	}
	data, err := gorkson.Marshal(payload)
	if err != nil {
		return err
	}
	var line bytes.Buffer
	if err := json.Compact(&line, data); err != nil {
		return err
	}
	return s.out.write("event: " + name + "\ndata: " + line.String() + "\n\n")
}

func (s *Stream[E]) eventType() reflect.Type {
	return reflect.TypeOf((*E)(nil)).Elem()
}

func (s *Stream[E]) attach(out *eventWriter) {
	s.out = out
}

// streamParam has unexported methods, so only *Stream[E] implements it.
type streamParam interface {
	eventType() reflect.Type
	attach(out *eventWriter)
}

var streamParamType = reflect.TypeOf((*streamParam)(nil)).Elem()

// streamEventType returns the event type E of a stream handler, or nil for other handlers.
func streamEventType(handlerType reflect.Type) reflect.Type {
	if handlerType.NumIn() != 3 {
		return nil
	}
	return reflect.Zero(handlerType.In(2)).Interface().(streamParam).eventType()
}

// validateStreamHandlerSignature validates func(context.Context, Request, *Stream[E]) error.
func validateStreamHandlerSignature(t reflect.Type) {
	if !t.In(2).Implements(streamParamType) {
		panic("third handler parameter must be *api.Stream[E]")
	}
	if t.NumOut() != 1 || !t.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("stream handler must return only error")
	}
	validateStreamEventType(streamEventType(t))
}

// validateStreamEventType validates that each field of the event struct is an
// exported pointer with a gork tag.
func validateStreamEventType(eventType reflect.Type) {
	if eventType.Kind() != reflect.Struct || eventType.NumField() == 0 {
		panic(fmt.Sprintf("stream event type %s must be a struct with at least one field", eventType))
	}
	for i := 0; i < eventType.NumField(); i++ {
		field := eventType.Field(i)
		if !field.IsExported() || field.Type.Kind() != reflect.Ptr || parseGorkTag(field.Tag.Get("gork")).Name == "" {
			panic(fmt.Sprintf("stream event field %s.%s must be an exported pointer with a gork tag", eventType, field.Name))
		}
	}
}

// streamEvent returns the event name and payload of the only set field.
func streamEvent(e reflect.Value) (string, any, error) {
	var name string
	var payload any
	set := 0
	for i := 0; i < e.NumField(); i++ {
		if e.Field(i).IsNil() {
			continue
		}
		set++
		name = parseGorkTag(e.Type().Field(i).Tag.Get("gork")).Name
		payload = e.Field(i).Interface()
	}
	if set != 1 {
		return "", nil, fmt.Errorf("stream event must set exactly one field, got %d", set)
	}
	return name, payload, nil
}

// eventWriter serializes the writes of Send and of the keep-alive loop.
type eventWriter struct {
	mu sync.Mutex
	w  http.ResponseWriter
	rc *http.ResponseController
}

func (o *eventWriter) write(s string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, err := io.WriteString(o.w, s); err != nil {
		return err
	}
	return o.rc.Flush()
}

func (o *eventWriter) keepAlive(done <-chan struct{}) {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			_ = o.write(": ping\n\n")
		}
	}
}

// serveStream opens the event stream and runs a stream handler. The stream
// opens before the handler runs, so an error from the handler only ends it.
func serveStream(w http.ResponseWriter, r *http.Request, handlerValue reflect.Value, reqPtr reflect.Value) {
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_ = rc.Flush()

	out := &eventWriter{w: w, rc: rc}
	stream := reflect.New(handlerValue.Type().In(2).Elem())
	stream.Interface().(streamParam).attach(out)

	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		out.keepAlive(done)
		close(stopped)
	}()
	handlerValue.Call([]reflect.Value{reflect.ValueOf(r.Context()), reqPtr.Elem(), stream})
	close(done)
	<-stopped
}
