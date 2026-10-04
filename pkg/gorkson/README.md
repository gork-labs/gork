# pkg/gorkson - JSON Encoding and Type Codecs

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg)](https://codecov.io/gh/gork-labs/gork/tree/main/pkg/gorkson)

This package is the type conversion layer of Gork. It encodes and decodes JSON with `gork` tags, and it keeps the global registry of type codecs. `pkg/api` uses it for parameters, request bodies, responses and stream events.

## Installation

```bash
go get github.com/gork-labs/gork@latest
```

## JSON with gork Tags

```go
type User struct {
    ID        string    `gork:"id"`
    CreatedAt time.Time `gork:"created_at"`
}

data, err := gorkson.Marshal(User{ID: "u1", CreatedAt: time.Now()})
// {"created_at":"2024-01-02T03:04:05Z","id":"u1"}

var u User
err = gorkson.Unmarshal(data, &u)
```

The field name comes from the `gork` tag. When a field has no `gork` tag, the `json` tag gives the name. When a field has no tag, the Go field name is the name, as in `encoding/json`. An unexported field or a field with the name `-` is not encoded. An exported embedded struct without a tag gives its fields to the outer object, as in `encoding/json`.

A value with no codec that implements `json.Marshaler` is written with its `MarshalJSON` method, also inside a struct, a slice or a pointer. A union from `pkg/unions` in a field thus writes only its set member.

## Type Codecs

A codec converts a Go type to a text value and back, and it gives the OpenAPI schema of the type:

```go
type TypeCodec[T any] interface {
    Parse(ctx context.Context, value string) (*T, error)
    Format(ctx context.Context, value *T) (string, error)
    Schema() OpenAPISchema
}
```

Register a codec once, for example in `init`:

```go
if err := gorkson.RegisterCodec[Priority](PriorityCodec{}); err != nil {
    panic(err)
}
```

- The codec applies to fields of type `T` and `*T` and to slice items.
- A second registration for the same type replaces the first.
- `RegisterCodec` returns an error when `T` is an interface type.

### Built-in Codecs

| Codec | Text form | Schema |
|-------|-----------|--------|
| `TimeCodec` | RFC3339 (`2024-01-02T03:04:05Z`) | `{"type": "string", "format": "date-time"}` |
| `UnixTimeCodec` | Unix seconds (`1704164645`) | `{"type": "integer", "format": "int64", "minimum": 0}` |

`TimeCodec` is the registered codec for `time.Time`. To use Unix seconds for all `time.Time` values, register `UnixTimeCodec`:

```go
gorkson.RegisterCodec[time.Time](gorkson.UnixTimeCodec{})
```

### Codecs in JSON

- When the schema type is `string` (or empty), the codec text is the content of a JSON string.
- For other schema types (`integer`, `number`, `boolean`, `object`, `array`), the codec text is the JSON value itself. A codec with an `integer` schema that formats `"42"` writes `42`.
- `Unmarshal` gives the content of a JSON string to `Parse`. For other JSON values, it gives their JSON text.
- `Marshal` writes `null` for a nil pointer. `Unmarshal` does not change a field for a `null` value.
- `Marshal` and `Unmarshal` give `context.Background()` to the codec.

### Schema Validation

Before `Parse`, gorkson checks the value against these constraints of the codec schema:

1. `Pattern`
2. `MinLength` and `MaxLength` (for the `string` type)
3. `Minimum` and `Maximum` (for the `integer` and `number` types)
4. `Enum`

The check applies to parameters and to JSON bodies. Thus `Parse` does not have to repeat these checks.

### Errors

- `NewParseError(typeName, value, err)` and `NewParseErrorWithReason(typeName, value, reason)` give a `*ParseError`.
- `NewFormatError(typeName, reason)` and `NewFormatErrorWithBase(typeName, reason, err)` give a `*FormatError`.
- `Marshal` and `Unmarshal` return codec errors. They do not ignore them.

## String Values

`pkg/api` uses these functions for parameters, response headers and response cookies:

- `SetFieldValueFromString(ctx, field, value)` sets a `reflect.Value` from a string. A registered codec has priority. Other supported types are strings, integers, floats, booleans, comma-separated string slices, and pointers to these types.
- `FormatFieldValueToString(ctx, value)` gives the string form of a value. A registered codec has priority. Basic kinds use their plain text form, `nil` gives an empty string, and other values give their JSON encoding.

## Registry

`GetCodecRegistry()` returns the global `CodecRegistry`. The OpenAPI generator of `pkg/api` reads the codec schemas from it. `NewCodecRegistry()` makes a separate registry.

## License

MIT License - see the root [LICENSE](../../LICENSE) file for details.
