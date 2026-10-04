# Type Codec System Examples

This directory contains examples demonstrating the new Type Codec system in Gork, which enables automatic type resolution and conversion for path/query parameters and response formatting.

## Features

The Type Codec system provides:

- **Bidirectional conversion**: Parse strings from HTTP parameters to Go types, and format Go types back to strings
- **OpenAPI integration**: Automatic schema generation for properly documented APIs
- **Entity resolution**: Convert IDs to full entity objects automatically
- **Type safety**: Full compile-time type checking with generics
- **Error handling**: Consistent error types that map to appropriate HTTP status codes
- **Complete replacement**: Replaces the deprecated TypeParserRegistry system

## Basic Usage

### 1. Define a Codec

```go
type TimeCodec struct{}

func (c TimeCodec) Parse(ctx context.Context, value string) (*time.Time, error) {
    t, err := time.Parse(time.RFC3339, value)
    if err != nil {
        return nil, api.NewInvalidFormatError("time.Time", value, "expected RFC3339 format")
    }
    return &t, nil
}

func (c TimeCodec) Format(ctx context.Context, value *time.Time) (string, error) {
    if value == nil {
        return "", nil
    }
    return value.Format(time.RFC3339), nil
}

func (c TimeCodec) Schema() api.OpenAPISchema {
    return api.OpenAPISchema{
        Type:        "string",
        Format:      "date-time",
        Pattern:     `^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d{3})?Z?$`,
        Example:     "2023-12-25T10:30:00Z",
        Description: "RFC3339 formatted timestamp",
    }
}
```

### 2. Register the Codec

```go
func init() {
    gorkson.RegisterCodec[time.Time](TimeCodec{})
}
```

### 3. Use in Request/Response Structs

```go
type GetUserRequest struct {
    Path struct {
        UserID    int       `gork:"userId"`     // Automatically parsed
        CreatedAt time.Time `gork:"createdAt"`  // Uses TimeCodec
    }
    Query struct {
        Active bool `gork:"active"`  // Automatically parsed
        Limit  int  `gork:"limit"`   // Automatically parsed
    }
}

type GetUserResponse struct {
    User      User      `json:"user"`
    CreatedAt time.Time `json:"createdAt"`  // Automatically formatted
    Active    bool      `json:"active"`     // Automatically formatted
}
```

### 4. Handler Function

```go
func GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    // All fields are automatically parsed using their registered codecs
    // req.Path.UserID is an int
    // req.Path.CreatedAt is a time.Time
    // req.Query.Active is a bool
    // req.Query.Limit is an int
    
    return &GetUserResponse{
        User:      findUser(req.Path.UserID),
        CreatedAt: req.Path.CreatedAt, // Will be formatted back to RFC3339
        Active:    req.Query.Active,   // Will be formatted back to "true"/"false"
    }, nil
}
```

## Advanced Examples

### Entity Resolution

Convert entity IDs to full objects automatically:

```go
type UserCodec struct {
    userService UserService
}

func (c UserCodec) Parse(ctx context.Context, value string) (*User, error) {
    userID, err := strconv.Atoi(value)
    if err != nil {
        return nil, api.NewInvalidFormatError("User", value, "invalid user ID")
    }
    
    // Fetch full user from database
    return c.userService.GetByID(ctx, userID)
}

func (c UserCodec) Format(ctx context.Context, value *User) (string, error) {
    if value == nil {
        return "", nil
    }
    // Format back to just the ID for transport
    return strconv.Itoa(value.ID), nil
}
```

### Enum Validation with iota

```go
// Priority uses iota for internal representation but string for API transport
type Priority int

const (
    PriorityLow Priority = iota    // 0
    PriorityMedium                 // 1  
    PriorityHigh                   // 2
)

// String provides clean string representation
func (p Priority) String() string {
    switch p {
    case PriorityLow:
        return "low"
    case PriorityMedium:
        return "medium"
    case PriorityHigh:
        return "high"
    default:
        return "unknown"
    }
}

// IsValid enables easy validation
func (p Priority) IsValid() bool {
    return p >= PriorityLow && p <= PriorityHigh
}

type PriorityCodec struct{}

func (c PriorityCodec) Parse(ctx context.Context, value string) (*Priority, error) {
    switch value {
    case "low":
        priority := PriorityLow
        return &priority, nil
    case "medium":
        priority := PriorityMedium
        return &priority, nil
    case "high":
        priority := PriorityHigh
        return &priority, nil
    default:
        return nil, api.NewInvalidFormatError("Priority", value, "must be one of: low, medium, high")
    }
}

func (c PriorityCodec) Format(ctx context.Context, value *Priority) (string, error) {
    if value == nil {
        return "medium", nil // Default
    }
    return value.String(), nil
}

func (c PriorityCodec) Schema() api.OpenAPISchema {
    return api.OpenAPISchema{
        Type: "string",
        Enum: []interface{}{"low", "medium", "high"},
        Example: "medium",
        Description: "Task priority level (internal iota, external string)",
    }
}

// Benefits of iota-based enums:
// - Type-safe comparisons: if priority >= PriorityHigh
// - Efficient sorting: sort by numeric value
// - Memory efficient: int instead of string storage
// - Validation: priority.IsValid()
// - Clean API: still uses strings in JSON/HTTP
```

## Built-in Codecs

Gork provides built-in codecs for special types that need custom parsing:

- `TimeCodec` - RFC3339 time parsing/formatting
- `UnixTimeCodec` - Unix timestamp parsing/formatting

Basic types like `string`, `int`, and `bool` are handled automatically by the regular JSON serialization system (gorkson) and don't need custom codecs.

## Automatic Schema Validation

Gork automatically validates input values against schema constraints before calling your codec:

```go
type EmailCodec struct{}

func (c EmailCodec) Schema() api.OpenAPISchema {
    return api.OpenAPISchema{
        Type:      "string",
        Format:    "email",
        Pattern:   `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
        MinLength: ptr(5),
        MaxLength: ptr(254),
        Enum:      []interface{}{"admin@example.com", "user@example.com"},
    }
}

func (c EmailCodec) Parse(ctx context.Context, value string) (*Email, error) {
    // All validation already done automatically:
    // ✅ Pattern validation (email regex)
    // ✅ Length validation (5-254 characters) 
    // ✅ Enum validation (if specified)
    
    // Focus only on business logic
    return &Email{Value: value}, nil
}
```

**Validation Types:**
- **Pattern** - Regex validation
- **MinLength/MaxLength** - String length validation
- **Minimum/Maximum** - Numeric range validation  
- **Enum** - Allowed values validation

## Error Handling

The codec system provides structured error handling:

```go
// Create specific error types
api.NewInvalidFormatError("time.Time", "invalid-date", "expected RFC3339 format")
api.NewParseError("User", "123", originalError)
api.NewFormatError("User", "invalid-user")
```

Errors automatically map to appropriate HTTP status codes:
- `ErrInvalidFormat`, `ErrParseFailed` → 400 Bad Request
- `ErrFormatFailed`, `ErrExternalServiceError` → 500 Internal Server Error

## OpenAPI Integration

Codecs automatically generate OpenAPI schemas for better documentation:

```yaml
parameters:
  - name: createdAt
    in: path
    required: true
    schema:
      type: string
      format: date-time
      pattern: '^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d{3})?Z?$'
      example: '2023-12-25T10:30:00Z'
      description: 'RFC3339 formatted timestamp'
```

## Running the Example

```bash
cd examples/type_codecs
go run main.go
```

This will demonstrate:
1. Codec registration
2. Type parsing and formatting
3. Entity resolution
4. Schema generation
5. End-to-end request/response handling
