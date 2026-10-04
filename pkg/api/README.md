# pkg/api - Convention Over Configuration HTTP Handler

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg)](https://codecov.io/gh/gork-labs/gork/tree/main/pkg/api)

This package provides a Convention Over Configuration HTTP handler adapter that uses structured request types to eliminate the need for parameter location tags.

## Installation

```bash
go get github.com/gork-labs/gork@latest
```

## Convention Over Configuration

Instead of using tags to specify parameter locations, use structured request types with standard sections: `Query`, `Body`, `Path`, `Headers`, and `Cookies`.

### Basic Example

```go
package main

import (
    "context"
    "net/http"
    "github.com/gork-labs/gork/pkg/api"
)

// User represents the user data structure
type User struct {
    // ID is the unique identifier for the user
    ID    string `gork:"id"`
    // Name is the user's full name
    Name  string `gork:"name"`
    // Email is the user's email address
    Email string `gork:"email"`
}

// Convention Over Configuration request structure
type CreateUserRequest struct {
    Body struct {
        // Name is the user's full name
        Name  string `gork:"name" validate:"required,min=3"`
        // Email is the user's email address
        Email string `gork:"email" validate:"required,email"`
    }
}

type CreateUserResponse struct {
    Body User
}

// Implement your business logic
func CreateUser(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
    return &CreateUserResponse{
        Body: User{
            ID:    "user-123",
            Name:  req.Body.Name,
            Email: req.Body.Email,
        },
    }, nil
}

func main() {
    // Create convention handler
    factory := api.NewConventionHandlerFactory()
    adapter := &api.HTTPParameterAdapter{}
    handler, _ := factory.CreateHandler(adapter, CreateUser)
    
    http.HandleFunc("/users", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Error Handling

To send an HTTP error status, return an `*api.HTTPError`. Use `api.NewHTTPError(status, message)` to make one:

```go
func GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    user, err := db.GetUser(req.Path.ID)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, api.NewHTTPError(http.StatusNotFound, "User not found") // 404 {"error":"User not found"}
        }
        return nil, err // 500 {"error":"Internal Server Error"}
    }
    return &GetUserResponse{Body: user}, nil
}
```

Gork writes the response for the error that the handler returns:

| Error | Status | Body |
| --- | --- | --- |
| `*api.HTTPError` with a 4xx status | `Status` | `{"error": Message}` |
| `*api.HTTPError` with a 5xx status | `Status` | `{"error": "<status text>"}` |
| `*api.ValidationErrorResponse` | 400 | `{"error": Message, "details": Details}` |
| Other errors | 500 | `{"error": "Internal Server Error"}` |

- Gork finds an `*api.HTTPError` in a wrapped error with `errors.As`.
- For a 5xx status, Gork does not send the message to the client. Gork writes the message to the log.

### Error Responses in the OpenAPI Spec

Each operation has the responses 400, 422 and 500. To add other error responses, declare their statuses with `api.WithErrorResponses`:

```go
router.Post("/login", Login, api.WithErrorResponses(http.StatusUnauthorized, http.StatusTooManyRequests))
```

Each status gets a response with the `ErrorResponse` schema:

```json
"401": {
  "description": "Unauthorized",
  "content": {
    "application/json": {
      "schema": {"$ref": "#/components/schemas/ErrorResponse"}
    }
  }
}
```

The responses 400, 422 and 500 always use the standard responses. `WithErrorResponses` does not change them.

### Authentication

These options add a security requirement to the OpenAPI operation. They do not check the request. Check the credentials in a middleware or in the handler.

| Option | Security scheme |
| --- | --- |
| `api.WithBasicAuth()` | `BasicAuth`: `{"type": "http", "scheme": "basic"}` |
| `api.WithBearerTokenAuth()` | `BearerAuth`: `{"type": "http", "scheme": "bearer"}` |
| `api.WithAPIKeyAuth()` | `ApiKeyAuth`: `{"type": "apiKey", "in": "header", "name": "X-API-Key"}` |
| `api.WithCookieAuth(name)` | `<name>`: `{"type": "apiKey", "in": "cookie", "name": "<name>"}` |

For a session cookie, use `WithCookieAuth` with the cookie name. The name of the security scheme is the cookie name:

```go
router.Get("/me", GetMe, api.WithCookieAuth("session_id"), api.WithErrorResponses(http.StatusUnauthorized))
```

### Request Structure

Use structured sections to organize parameters by their HTTP location:

```go
type UpdateUserRequest struct {
    Path struct {
        // UserID is the unique identifier for the user to update
        UserID string `gork:"user_id" validate:"required,uuid"`
    }
    Query struct {
        // Notify determines if notifications should be sent
        Notify bool `gork:"notify"`
    }
    Headers struct {
        // Version specifies the API version for the request
        Version int `gork:"X-User-Version"`
    }
    Body struct {
        // Name is the updated user's full name
        Name  string `gork:"name" validate:"omitempty,min=3,max=100"`
        // Email is the updated user's email address
        Email string `gork:"email" validate:"omitempty,email"`
    }
}
```

### Mixed Parameter Example

All parameter types in one request:

```go
// POST /users/123?notify=true
// Headers: X-User-Version: 2
// Body: {"name": "John", "email": "john@example.com"}

type UpdateUserRequest struct {
    Path struct {
        // UserID comes from the URL path parameter
        UserID string `gork:"user_id"`
    }
    Query struct {
        // Notify comes from the query string
        Notify bool `gork:"notify"`
    }
    Headers struct {
        // Version comes from HTTP headers
        Version int `gork:"X-User-Version"`
    }
    Body struct {
        // Name comes from the JSON request body
        Name  string `gork:"name"`
        // Email comes from the JSON request body
        Email string `gork:"email"`
    }
}
```

### Array Body

A `Body` of a slice type is a JSON array, in a request and in a response:

```go
type CreateUsersRequest struct {
    Body []NewUser
}

type ListUsersResponse struct {
    Body []User
}
```

- The OpenAPI schema is `{"type": "array", "items": {"$ref": "#/components/schemas/User"}}`.
- Gork validates each item of a request array with its `validate` tags.
- A request `Body []byte` is the raw request body. It has no JSON schema.

### Response Cookies

A plain field in the response `Cookies` section sets a cookie with the name from the `gork` tag. Gork gives this cookie the `Secure`, `HttpOnly` and `SameSite=Lax` attributes. Gork does not send a cookie for an empty value.

To control the attributes, use a field of type `*http.Cookie` or `http.Cookie`. Gork sends this cookie unchanged with `http.SetCookie`. The cookie name comes from `Cookie.Name`, so the field does not need a `gork` tag. A nil `*http.Cookie` sends no cookie.

```go
type LoginResponse struct {
    Cookies struct {
        // Session gets Secure, HttpOnly and SameSite=Lax
        Session string `gork:"session_id"`
        // JavaScript in the browser can read Theme
        Theme *http.Cookie
    }
}

func Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
    resp := &LoginResponse{}
    resp.Cookies.Session = newSessionID()
    resp.Cookies.Theme = &http.Cookie{
        Name:     "theme",
        Value:    "dark",
        Path:     "/",
        MaxAge:   86400,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    }
    return resp, nil
}
```

The OpenAPI document does not show response cookies.

### Response Status and Redirects

Gork sends 200 for a response with a `Body` and 204 for a response without a `Body`. To send a different status, add the route option `api.WithStatus(status)`. The OpenAPI operation then shows the success response with this status and its status text as the description. A stream handler always sends 200.

For a redirect, use a response without a `Body` that sets the `Location` header in the `Headers` section, and add `api.WithStatus` with a 3xx status:

```go
type GitHubCallbackRequest struct {
    Query struct {
        Code  string `gork:"code" validate:"required"`
        State string `gork:"state" validate:"required"`
    }
}

type GitHubCallbackResponse struct {
    Headers struct {
        // Location is the page that the browser opens next
        Location string `gork:"Location"`
    }
}

func GitHubCallback(ctx context.Context, req GitHubCallbackRequest) (*GitHubCallbackResponse, error) {
    if !validState(req.Query.State) {
        return nil, api.NewHTTPError(http.StatusForbidden, "The state is not valid.") // 403 {"error":"The state is not valid."}
    }
    resp := &GitHubCallbackResponse{}
    resp.Headers.Location = "/github"
    return resp, nil // 303 See Other, Location: /github, no body
}

router.Get("/github/callback", GitHubCallback, api.WithStatus(http.StatusSeeOther), api.WithErrorResponses(http.StatusForbidden))
```

An error from the handler gives the usual JSON error response. The OpenAPI operation shows the redirect without content:

```json
"303": {
  "description": "See Other",
  "headers": {
    "Location": {"description": "Response header", "schema": {"type": "string"}}
  }
}
```

### Context Usage

The adapter passes through the HTTP request context:

```go
func GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    // Access request-scoped values
    userID := ctx.Value("userID").(string)
    
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue processing
    }
    
    return fetchUser(ctx, req.Path.ID)
}
```

## Features

- **Convention Over Configuration**: No need for parameter location tags
- **Type Safety**: Compile-time type checking for requests and responses
- **Automatic Validation**: Built-in request validation using gork tags
- **Error Handling**: Consistent error responses with proper HTTP status codes
- **Structured Requests**: Clear separation of parameters by HTTP location
- **Context Propagation**: Full support for context cancellation and values
- **Framework Agnostic**: Works with any router that accepts `http.HandlerFunc`

## Handler Signature

Handlers must follow this signature:

```go
func HandlerName(ctx context.Context, req RequestType) (*ResponseType, error)
```

Where:
- `ctx` is the request context
- `req` is your request type with convention sections (validated automatically)
- `ResponseType` is your response type with convention sections (pointer)
- `error` is for error handling

A handler can also be a method value, for example `router.Get("/workstreams", h.ListWorkstreams)`. The function or method name is the `operationId`, and its doc comment is the operation description.

## Stream Handlers (Server-Sent Events)

A handler with a third parameter `*api.Stream[E]` sends Server-Sent Events. Register it with the usual router methods.

```go
// LiveEvents lists the events of the stream. Each field is one event type.
type LiveEvents struct {
    Feed  *FeedRow  `gork:"feed"`
    Reset *struct{} `gork:"reset"` // event without payload
}

func Live(ctx context.Context, req LiveRequest, stream *api.Stream[LiveEvents]) error {
    if err := stream.Send(LiveEvents{Reset: &struct{}{}}); err != nil {
        return err
    }
    for {
        select {
        case <-ctx.Done():
            return nil
        case row := <-feed:
            if err := stream.Send(LiveEvents{Feed: &row}); err != nil {
                return err
            }
        }
    }
}

router.Get("/live", Live)
```

Rules:
- Each field of the event struct must be an exported pointer with a `gork` tag. The tag is the SSE `event:` name. Gork checks the struct at registration and panics when it breaks a rule.
- `Send` writes `event: <tag>` and `data: <json>` and flushes. Set exactly one field for each call. Otherwise `Send` returns an error.
- `SendWithID(id, e)` also writes the line `id: <id>`. An id with CR, LF or NUL gives an error.
- Gork parses and validates the request before the stream starts. An invalid request gets the usual JSON error response.
- Then Gork sends status 200 with `Content-Type: text/event-stream`. An error from the handler only ends the stream.
- Gork sends a `: ping` comment every 15 seconds while the handler runs.
- Gork clears the write deadline of the connection, so `http.Server.WriteTimeout` does not stop the stream.
- When the client disconnects, `ctx` is done. Return from the handler.
- Call `Send` only from the handler goroutine.
- The OpenAPI spec shows the route as `text/event-stream` with an `itemSchema`. The spec has `openapi: 3.2.0` when it has a stream route.
- The `itemSchema` is a `$ref` to a component with the name of the event type, for example `LiveEvents`. The component is a `oneOf` with one entry for each event field. Thus a tool that does not read `itemSchema`, such as openapi-typescript, still generates a type for the events.
- Each `oneOf` entry describes one event as OpenAPI 3.2 shows it for `text/event-stream`. `event` is a string with the tag as `const`. `data` is a string with `contentMediaType: application/json` and a `contentSchema` for the payload. `id` is an optional string, because any handler can call `SendWithID`. The spec ignores comments, so the `: ping` lines are not in the schema.
- The Fiber adapter does not support stream handlers.

### Resume After a Reconnect

A browser `EventSource` keeps the id of the last event. When it connects again, it sends this id in the `Last-Event-ID` request header. Read the header with a field of the `Headers` section, and send each event with `SendWithID`:

```go
type LiveRequest struct {
    Headers struct {
        // LastEventID is the id of the last event that the client got
        LastEventID string `gork:"Last-Event-ID"`
    }
}

func Live(ctx context.Context, req LiveRequest, stream *api.Stream[LiveEvents]) error {
    for _, row := range rowsAfter(req.Headers.LastEventID) {
        if err := stream.SendWithID(row.ID, LiveEvents{Feed: &row}); err != nil {
            return err
        }
    }
    for {
        select {
        case <-ctx.Done():
            return nil
        case row := <-feed:
            if err := stream.SendWithID(row.ID, LiveEvents{Feed: &row}); err != nil {
                return err
            }
        }
    }
}
```

The OpenAPI spec shows `Last-Event-ID` as a header parameter of the route.

## Type Conversion

The `gorkson` package converts all values: path, query, header and cookie parameters, the JSON body, the JSON response, response headers and cookies, and stream event payloads. A type with a registered codec uses its codec. The codec also gives the OpenAPI schema of the type, and the parser checks each value against the schema constraints before it calls the codec.

`time.Time` and `*time.Time` use RFC3339 with fractional seconds and the OpenAPI schema `{"type": "string", "format": "date-time"}` with no registration.

```go
type GetTaskRequest struct {
    Path struct {
        Task        Task      `gork:"taskId"`      // codec registered with gorkson.RegisterCodec[Task]
        CompletedAt time.Time `gork:"completedAt"` // built-in RFC3339 codec
    }
}
```

Parameters of other types use the basic conversion: strings, integers, floats, booleans, comma-separated string slices, and pointers to these types. See the [root README](../../README.md#-type-codec-system) and the [gorkson README](../gorkson/README.md) for the codec API.

## OpenAPI Integration

This adapter automatically generates OpenAPI specifications from convention-based request/response structures using the gork CLI tool.

### Required Fields

The `required` list of a schema follows how Gork reads a request and writes a response:

- **Request**: A request body field or a parameter is required only when it has `validate:"required"`. A client can omit other fields. Then the field gets its zero value.
- **Response**: Gork writes the response body and the stream event data with `gorkson`. `gorkson` writes each exported field, and it does not use `omitempty`. Thus each of these fields is required.
- A pointer field of a response is required and nullable. A nil pointer or interface gives `null`, so the key is always there.
- A nil slice gives `[]` and a nil map gives `{}`. Thus a slice or map field is not nullable.
- The property name comes from the `gork` tag, else from the `json` tag, else it is the Go field name. A field with the name `-` is not in the schema.
- An exported embedded struct without a tag gives its properties to the outer schema, because `gorkson` writes its fields in the outer object.
- A named struct type gives one component. When a request and a response both use the type, the component gets the response rule.
- A webhook response uses `encoding/json`, so it keeps the `validate:"required"` rule.

### Component Schemas

- A named struct type gives one component. A field of a recursive type refers to this component with `$ref`.
- Only the union types of `pkg/unions` give a `oneOf`. To give a union a name, use a type alias: `type PaymentMethod = unions.Union2[Card, Bank]`.
- A defined type such as `type PaymentMethod unions.Union2[Card, Bank]` does not keep the `MarshalJSON` method. Thus `gorkson` writes it as an object with the fields `A` and `B`, and the schema shows this object.

### Component Names

A component of a named type has the name of the Go type. The name does not include the package.

A component of a generic type instance has the name of the generic type and one part for each type argument. The parts are joined by `_`. Each type argument gives its part with these rules:

| Type argument | Part | Example instance | Component name |
| --- | --- | --- | --- |
| Named type `T` | `T` without the package | `Envelope[models.Item]` | `Envelope_Item` |
| Built-in type | The type name | `Envelope[string]` | `Envelope_string` |
| Slice `[]T` | `Array_` and the part of `T` | `Envelope[[]Item]` | `Envelope_Array_Item` |
| Array `[N]T` | `ArrayN_` and the part of `T` | `Envelope[[3]Item]` | `Envelope_Array3_Item` |
| Pointer `*T` | `Nullable_` and the part of `T` | `Envelope[*Item]` | `Envelope_Nullable_Item` |
| Map `map[K]V` | `Map_`, the part of `K`, `_`, and the part of `V` | `Envelope[map[string]Item]` | `Envelope_Map_string_Item` |
| Generic instance | The name of the instance with these rules | `Envelope[Page[Item]]` | `Envelope_Page_Item` |

A type with more than one type argument gives one part for each argument. For example, `Pair[Item, User]` gives `Pair_Item_User`.

A pointer gets its own part because the schema of a pointer is nullable. Thus `Envelope[*Item]` and `Envelope[Item]` have different schemas.

Two different Go types must not get the same component name. For example, `models.Item` and `billing.Item` both get the name `Item`. If this occurs, the OpenAPI generator stops with a panic. The panic message names the two Go types and the component name. To fix the problem, rename one of the types.

## Examples

See the [examples](../../examples/) directory for complete working examples with different web frameworks.

## License

MIT License - see the root [LICENSE](../../LICENSE) file for details.