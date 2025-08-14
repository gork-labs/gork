# Webhook Support (Stripe as Reference) — Revised Implementation Plan

## Overview

Introduce first‑class, type‑safe webhook handling with runtime OpenAPI generation. Stripe is the reference provider; the core remains provider‑agnostic and extensible. Breaking changes are allowed to achieve a clean, clear API.

## Goals

- **Type safety**: Handlers receive strongly‑typed provider objects and validated user metadata.
- **Provider‑agnostic core**: Interfaces make it easy to add GitHub, SendGrid, etc.
- **Runtime OpenAPI**: Webhook routes produce meaningful request/response schemas and extensions.
- **Provider-grade verification**: Signature verification and timestamp tolerance.
- **Clarity**: One canonical Stripe implementation; avoid duplicates.

## Core API Design

All core webhook primitives live in `pkg/api/webhook.go`.

```go
// Marker interface implemented by provider-specific request types.
type WebhookRequest interface { WebhookRequest() }

// Parsed event returned by provider handlers after verification/parsing.
type WebhookEvent struct {
    Type           string          // provider event type, e.g. "payment_intent.succeeded"
    ProviderObject any            // concrete provider payload, e.g. *stripe.PaymentIntent
    UserMetaJSON   json.RawMessage // optional: provider-extracted metadata JSON
}

// Implemented by providers (Stripe, GitHub, ...).
type WebhookHandler[T WebhookRequest] interface {
    // Verifies signature, parses payload, returns typed provider object and optional user metadata.
    ParseRequest(req T) (WebhookEvent, error)

    // Response helpers (used by dispatcher and OpenAPI reflection).
    SuccessResponse() any
    ErrorResponse(err error) any

    // Advertises supported event types for validation and OpenAPI.
    GetValidEventTypes() []string
}

// Dispatcher configuration and options.
type WebhookHandlerOption struct {
    Tags                 []string
    EventHandlers        map[string]any
    StrictUserValidation bool // false: pass nil metadata on validation errors; true: return 400
}

type WebhookOption func(*WebhookHandlerOption)

// Dispatcher that binds provider handler + event handlers into an http.HandlerFunc.
func WebhookHandlerFunc[T WebhookRequest](h WebhookHandler[T], opts ...WebhookOption) http.HandlerFunc
```

### Typed handler registration

Replace the old `WithEventHandler` with a fully typed API:

```go
// Registers a type-safe handler for `eventType`.
// P: provider payload (e.g., *stripe.PaymentIntent)
// U: user metadata (validated via go-playground/validator)
func WithEventHandler[P any, U any](
    eventType string,
    handler func(ctx context.Context, provider *P, user *U) (resp any, err error),
) WebhookOption
```

Dispatcher behavior:

- Resolve `event.Type` to the registered handler.
- Assert `event.ProviderObject` to `*P`; mismatch → 500 via `ErrorResponse` with a clear message.
- If `event.UserMetaJSON` is present, unmarshal into `*U` and validate with `validator.v10`.
- If metadata validation fails:
  - When `StrictUserValidation=false` (default): pass `nil` for `*U` and continue.
  - When `StrictUserValidation=true`: return 400 with `ErrorResponse`.

Webhook routes are auto-detected when using `api.WebhookHandlerFunc`; no extra tagging option is required.

## Stripe Provider (Reference Implementation)

Location: `pkg/webhooks/stripe/`

### Request type

```go
// types.go
type Request struct {
    Headers struct {
        StripeSignature string `gork:"Stripe-Signature" validate:"required"`
        ContentType     string `gork:"Content-Type"`
    }
    Body []byte
}

func (Request) WebhookRequest() {}
```

### Handler

```go
// handler.go
type Handler struct {
    secret     string
    tolerance  time.Duration // e.g., 5m replay window
    typeMapper func(eventType string, ev stripe.Event) (provider any, userMeta json.RawMessage, err error)
}

func NewHandler(secret string, opts ...Option) *Handler

func (h *Handler) ParseRequest(req Request) (api.WebhookEvent, error) {
    // Verify signature via stripe-go webhook.ConstructEventWithOptions
    // Build provider object via curated prefix mapping
    // Extract metadata (when available) as json.RawMessage for user validation
}

func (h *Handler) SuccessResponse() any    // { "received": true }
func (h *Handler) ErrorResponse(err error) any
func (h *Handler) GetValidEventTypes() []string

type Option func(*Handler)
func WithTolerance(d time.Duration) Option
func WithTypeMapper(f func(string, stripe.Event) (any, json.RawMessage, error)) Option
```

### Type mapping and metadata extraction

- Maintain a curated prefix map for major event families (e.g., `payment_intent.`, `customer.subscription.`, `invoice.`, `charge.`) → concrete Stripe types.
- For unmatched events, return `*stripe.Event` or `event.Data.Raw` so users can still handle them.
- Metadata extraction:
  - If the provider object exposes a `Metadata` map (e.g., `PaymentIntent.Metadata`), marshal it into `json.RawMessage` for user validation as `*U`.
  - Otherwise, `UserMetaJSON=nil`.

### Security

- Signature verification via Stripe SDK; enforce timestamp tolerance with `ConstructEventWithOptions`.
- Library behavior: if no handler is registered for an incoming event type, respond with provider `SuccessResponse()`.

## Dispatcher Details

Request flow:

1. Parse `T` via Gork conventions (headers/body); validate basic headers (e.g., `Stripe-Signature`).
2. Call `ParseRequest` on the provider handler to verify signature and produce `WebhookEvent`.
3. Resolve the event handler by `WebhookEvent.Type`. If not found: log and return `SuccessResponse()`.
4. Assert provider payload type `*P` and prepare validated `*U` per `StrictUserValidation` rules.
5. Call user handler and write either the response (200) or `ErrorResponse(err)` (500 or 400 in strict mode).

## OpenAPI Generation (Runtime-Based)

Use the existing runtime registry; do not add AST analysis.

- Webhook routes are detected via the dispatcher created by `api.WebhookHandlerFunc`.
- Operation fields:
  - Tags: include `webhooks`, plus any user tags.
  - Extensions: set `x-webhook-provider` and `x-webhook-events` in the in‑memory model. Decide whether to emit these in JSON/YAML or keep internal until extension serialization is supported.
  - Request schema: generated from the webhook request type (headers + raw body), not the provider event payload.
  - Responses: generated by reflecting `SuccessResponse()` and `ErrorResponse()` on the provider handler.
- Future (optional): for Stripe, add `oneOf` for a curated subset of `data.object` shapes with a discriminator.

## Examples

### http.ServeMux

```go
mux.Handle("/webhooks/stripe", api.WebhookHandlerFunc(
  stripe.NewHandler(os.Getenv("STRIPE_WEBHOOK_SECRET"),
    stripe.WithTolerance(5*time.Minute),
  ),
  api.WithEventHandler[*stripe.PaymentIntent, PaymentMetadata](
    "payment_intent.succeeded", handlePaymentIntentSucceeded,
  ),
  api.WithEventHandler[*stripe.Subscription, SubMetadata](
    "customer.subscription.deleted", handleSubscriptionDeleted,
  ),
))
```

```go
func handlePaymentIntentSucceeded(ctx context.Context, pi *stripe.PaymentIntent, meta *PaymentMetadata) (any, error) {
  // meta may be nil when validation fails and StrictUserValidation=false
  return stripepkg.SuccessResponse(), nil
}
```

## Testing Strategy

- Unit tests
  - Typed registration + dispatch with correct provider payloads.
  - Signature verification (success/failure) with tolerance.
  - Metadata extraction from provider objects; validation pass/fail; strict vs non‑strict behavior.
  - Event type validation at registration using `GetValidEventTypes()`.
- Integration tests
  - Valid Stripe payload + signature → 200 with success response.
  - Invalid signature → 401 with error response.
  - Unknown event type → 200 success, logged.
- OpenAPI tests
  - Webhook operation has correct tags, extensions, request schema (headers/body), and response shapes.

## Phases

- Phase 1 — Core
  - Introduce `WebhookRequest`, `WebhookEvent`.
  - Implement typed `WithEventHandler[P,U]` and `WebhookHandlerFunc` with validation.
  - Add `StrictUserValidation` behavior.
- Phase 2 — Stripe
  - Implement `pkg/webhooks/stripe` with SDK verification, tolerance, curated type mapping, metadata extraction.
  - Remove any duplicate Stripe handler paths and examples that do not verify signatures.
- Phase 3 — OpenAPI
  - Runtime webhook operation generation with provider/event extensions and request/response schemas.
  - Optional: add Stripe `oneOf` discriminator for a curated subset of events.
- Phase 4 — Docs/Examples
  - Update examples to pass the secret directly to `stripe.NewHandler`.
  - Document typed registration, metadata semantics, strict mode, and OpenAPI expectations.

## Acceptance Criteria

- Compile‑time typed provider payloads in handlers; clear runtime error on mismatches.
- Working Stripe signature verification with configured tolerance.
- Validated user metadata delivered as `*U` or rejected per strict mode.
- OpenAPI includes webhook routes with tags, provider/events extensions, request schema, and response schemas.
- Examples run end‑to‑end against Stripe test payloads.


