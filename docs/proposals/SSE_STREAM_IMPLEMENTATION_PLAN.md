# Server-Sent Events (SSE) Stream Handlers — Implementation Plan

## Overview

Add a second handler form that sends Server-Sent Events. A stream handler uses the usual request conventions. A typed event struct defines the events. The OpenAPI document describes the stream with the OpenAPI 3.2 `itemSchema`.

## Goals

- **Type safety**: The handler sends typed events. Gork checks the event struct at registration.
- **Same request conventions**: Gork parses and validates the request like for a normal handler.
- **Runtime OpenAPI**: The spec shows each event name and the schema of its payload.
- **No new options**: The keep-alive interval and the headers are fixed.

## Handler Form

A handler with a third parameter `*api.Stream[E]` is a stream handler. It registers with the usual router methods.

```go
func Live(ctx context.Context, req LiveRequest, stream *api.Stream[LiveEvent]) error

r.Get("/live", Live)
```

- `validateHandlerSignature` accepts 2 or 3 parameters. A stream handler must return only `error`.
- `RouteInfo.StreamType` holds the event type `E`. It is nil for other handlers.

## Event Struct Convention

`E` is a struct. Each field is one event type, like the request sections convention.

```go
type LiveEvent struct {
    Feed        *FeedRow   `gork:"feed"`
    Agent       *AgentNode `gork:"agent"`
    Workstreams *struct{}  `gork:"workstreams"`
}
```

- Each field must be an exported pointer with a `gork` tag. The tag is the SSE `event:` name.
- An event without a payload uses `*struct{}`.
- Gork panics at registration when the struct breaks a rule.
- `Stream.Send(e E) error` returns an error when no field or more than one field is set.

## Runtime

1. Gork parses and validates the request. An error gives the usual JSON error response (400/422/500).
2. Gork clears the write deadline through `http.ResponseController`. Gork ignores `http.ErrNotSupported`. Another error gives a 500 response.
3. Gork writes status 200 and these headers before the handler runs:
   - `Content-Type: text/event-stream`
   - `Cache-Control: no-cache`
   - `X-Accel-Buffering: no`
4. Each `Send` writes `event: <tag>\ndata: <json>\n\n` and flushes. The data uses the same `gorkson` encoding as response bodies. `json.Compact` keeps the data on one line.
5. A goroutine writes `: ping\n\n` every 15 seconds while the handler runs. A mutex in the stream prevents a mix of a ping and an event.
6. When the client disconnects, `ctx` is done and the handler returns. An error from the handler only ends the stream.

Gork writes the headers at stream start, not at the first `Send`. This is the simpler option: the keep-alive writer does not have to start the response, and the client gets the headers immediately.

There is no `id:` or `Last-Event-ID` support.

## OpenAPI

The 200 response has `text/event-stream` with an `itemSchema`. The `oneOf` has one entry for each event field:

```yaml
itemSchema:
  oneOf:
    - type: object
      required: [event, data]
      properties:
        event: { const: agent }
        data:
          contentMediaType: application/json
          contentSchema: { $ref: '#/components/schemas/AgentNode' }
```

- Each payload type is a usual component schema.
- `MediaType.ItemSchema` and the schema fields `const`, `contentMediaType` and `contentSchema` are new.
- The spec has `openapi: 3.2.0` when it has a stream route. Other specs keep `3.1.0`.
- Stream routes keep the standard error responses (400/422/500).
- The Swagger validator does not accept OpenAPI 3.2 documents. The `gork` CLI and `scripts/validate-openapi.sh` do not send 3.2 documents to it. Redocly lint checks them.

## Adapters

- stdlib, chi, gorilla, gin and echo flush through `http.ResponseController`. A test for each adapter reads the first event before the handler returns.
- Fiber runs on fasthttp, which sends the response body only after the handler returns. The Fiber adapter panics when a stream route registers.

## Linter

`lintgork` finds handlers with a `*api.Stream[E]` parameter. It reports event fields that are not exported, not pointers, or without a `gork` tag.

## Tests

- `Send`: event frame, gork tag names, compact JSON, no field or two fields set, encode errors, write and flush errors.
- Signature: valid stream handler and each signature or event struct error.
- Runtime: headers, parse and validation errors, write deadline error, handler error, keep-alive.
- HTTP server: flush before the handler returns, `WriteTimeout` does not stop the stream, client disconnect.
- OpenAPI: `itemSchema`, component schemas, error responses, `3.2.0` only with a stream route.
- Adapters: flush test for each `net/http` adapter, registration panic for Fiber.
