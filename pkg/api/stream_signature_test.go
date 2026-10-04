package api

import (
	"context"
	"reflect"
	"testing"
)

type streamTestRequest struct {
	Query struct {
		Topic string `gork:"topic" validate:"required"`
	}
}

type streamValueEvents struct {
	Row streamTestRow `gork:"row"`
}

type streamUntaggedEvents struct {
	Row *streamTestRow
}

type streamUnexportedEvents struct {
	row *streamTestRow `gork:"row"`
}

type streamEmptyEvents struct{}

func expectStreamPanic(t *testing.T, want string, handler interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r != want {
			t.Errorf("expected panic %q, got %v", want, r)
		}
	}()
	validateHandlerSignature(reflect.TypeOf(handler))
}

func TestValidateStreamHandlerSignature(t *testing.T) {
	t.Run("valid stream handler", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("valid stream handler should not panic: %v", r)
			}
		}()
		validateHandlerSignature(reflect.TypeOf(func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error { return nil }))
	})

	t.Run("third parameter is not a stream", func(t *testing.T) {
		expectStreamPanic(t, "third handler parameter must be *api.Stream[E]",
			func(context.Context, streamTestRequest, string) error { return nil })
	})

	t.Run("stream passed by value", func(t *testing.T) {
		expectStreamPanic(t, "third handler parameter must be *api.Stream[E]",
			func(context.Context, streamTestRequest, Stream[streamTestEvents]) error { return nil })
	})

	t.Run("response value returned", func(t *testing.T) {
		expectStreamPanic(t, "stream handler must return only error",
			func(context.Context, streamTestRequest, *Stream[streamTestEvents]) (*TestResponse, error) {
				return nil, nil
			})
	})

	t.Run("non-error returned", func(t *testing.T) {
		expectStreamPanic(t, "stream handler must return only error",
			func(context.Context, streamTestRequest, *Stream[streamTestEvents]) string { return "" })
	})

	t.Run("event type is not a struct", func(t *testing.T) {
		expectStreamPanic(t, "stream event type string must be a struct with at least one field",
			func(context.Context, streamTestRequest, *Stream[string]) error { return nil })
	})

	t.Run("event struct without fields", func(t *testing.T) {
		expectStreamPanic(t, "stream event type api.streamEmptyEvents must be a struct with at least one field",
			func(context.Context, streamTestRequest, *Stream[streamEmptyEvents]) error { return nil })
	})

	t.Run("event field is not a pointer", func(t *testing.T) {
		expectStreamPanic(t, "stream event field api.streamValueEvents.Row must be an exported pointer with a gork tag",
			func(context.Context, streamTestRequest, *Stream[streamValueEvents]) error { return nil })
	})

	t.Run("event field without gork tag", func(t *testing.T) {
		expectStreamPanic(t, "stream event field api.streamUntaggedEvents.Row must be an exported pointer with a gork tag",
			func(context.Context, streamTestRequest, *Stream[streamUntaggedEvents]) error { return nil })
	})

	t.Run("unexported event field", func(t *testing.T) {
		expectStreamPanic(t, "stream event field api.streamUnexportedEvents.row must be an exported pointer with a gork tag",
			func(context.Context, streamTestRequest, *Stream[streamUnexportedEvents]) error { return nil })
	})
}

func TestStreamHandlerRouteInfo(t *testing.T) {
	handler := func(context.Context, streamTestRequest, *Stream[streamTestEvents]) error { return nil }

	_, info := createHandlerFromAny(&HTTPParameterAdapter{}, handler)

	if info.StreamType != reflect.TypeOf(streamTestEvents{}) {
		t.Errorf("expected StreamType streamTestEvents, got %v", info.StreamType)
	}
	if info.RequestType != reflect.TypeOf(streamTestRequest{}) {
		t.Errorf("expected RequestType streamTestRequest, got %v", info.RequestType)
	}
	if info.ResponseType != nil {
		t.Errorf("expected nil ResponseType, got %v", info.ResponseType)
	}

	_, plain := createHandlerFromAny(&HTTPParameterAdapter{}, func(context.Context, streamTestRequest) error { return nil })
	if plain.StreamType != nil {
		t.Errorf("expected nil StreamType for a normal handler, got %v", plain.StreamType)
	}
}
