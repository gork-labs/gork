# Convention Over Configuration Implementation

This document describes the implementation of the Convention Over Configuration approach for Gork, as specified in [CONVENTION_OVER_CONFIGURATION_SPEC.md](specs/CONVENTION_OVER_CONFIGURATION_SPEC.md).

## Overview

The Convention Over Configuration implementation provides a structured approach to HTTP request/response handling with explicit sections, enhanced validation, and improved OpenAPI generation.

## Implementation Components

### 1. Core Parser (`pkg/api/convention_parser.go`)

The `ConventionParser` handles parsing HTTP requests into structured request types with explicit sections.

**Key Features:**
- Section-based parsing (Query, Body, Path, Headers, Cookies)
- Type parser registry for complex types
- `gork` tag support for field mapping
- Union type discriminator parsing

**Usage:**
```go
parser := NewConventionParser()

// Register type parsers for complex types
parser.RegisterTypeParser(func(ctx context.Context, id string) (*User, error) {
    return userService.GetByID(ctx, id)
})

// Parse request automatically detects sections
err := parser.ParseRequest(ctx, httpRequest, requestPtr, adapter)
```

### 2. Validation System (`pkg/api/convention_validation.go`)

The `ConventionValidator` provides comprehensive validation with proper error namespacing.

**Validation Levels:**
1. **Field-level validation**: Using `validate` tags
2. **Section-level validation**: Custom `Validate()` methods on sections
3. **Request-level validation**: Custom `Validate()` methods on requests

**Error Types:**
- `ValidationErrorResponse`: Client errors (HTTP 400)
- `RequestValidationError`: Request-level validation errors
- `SectionValidationError`: Section-specific errors (Query, Body, etc.)

**Usage:**
```go
validator := NewConventionValidator()
err := validator.ValidateRequest(requestPtr)

if IsValidationError(err) {
    // Handle as HTTP 400 Bad Request
} else {
    // Handle as HTTP 500 Internal Server Error
}
```

### 3. Type Parser Registry (`pkg/api/type_parser_registry.go`)

The `TypeParserRegistry` manages parsers for complex types, enabling automatic entity resolution.

**Parser Signature:**
```go
func(ctx context.Context, value string) (*T, error)
```

**Example:**
```go
registry := NewTypeParserRegistry()

// Register entity parser
registry.Register(func(ctx context.Context, id string) (*User, error) {
    return userRepo.GetByID(ctx, id)
})

// Register standard library parser
registry.Register(func(ctx context.Context, s string) (*time.Time, error) {
    t, err := time.Parse(time.RFC3339, s)
    return &t, err
})
```

### 4. Handler Factory (`pkg/api/convention_handler_factory.go`)

The `ConventionHandlerFactory` creates HTTP handlers using the Convention Over Configuration approach.

**Features:**
- Automatic detection of convention usage
- Response section processing (Body, Headers, Cookies)
- Proper error handling and status codes

**Integration:**
The factory is automatically used when request types use standard sections. Legacy handlers continue to work unchanged.

### 5. OpenAPI Generation (`pkg/api/convention_openapi_generator.go`)

Enhanced OpenAPI generation with full Convention Over Configuration support.

**Features:**
- Section-aware parameter generation
- Union type oneOf schemas with discriminators
- Response header documentation
- Automatic schema generation from `gork` tags

### 6. Enhanced Linter (`internal/lintgork/convention_analyzer.go`)

The linter validates Convention Over Configuration structures at build time.

**Checks:**
- Valid section names (Query, Body, Path, Headers, Cookies)
- Required `gork` tags on section fields
- Handler signature validation
- Mixed usage detection (convention + legacy)
- Discriminator validation for union types

## Request Structure

### Standard Sections

All request types can use these exact section names:

```go
type RequestName struct {
    Query   QueryStruct   // Optional - query parameters
    Body    BodyStruct    // Optional - request body  
    Path    PathStruct    // Optional - path parameters
    Headers HeaderStruct  // Optional - HTTP headers
    Cookies CookieStruct  // Optional - HTTP cookies
}
```

### Field Requirements

1. **Section names are case-sensitive**: `Query`, `Body`, `Path`, `Headers`, `Cookies`
2. **Sections are optional**: Only include sections you need
3. **Each section contains a struct**: With relevant fields using `gork` tags
4. **Use `gork:"name"` tags**: For field naming (not `json:"name"`)
5. **Use `validate:"..."` tags**: For validation rules

### Example Request

```go
type UpdateUserRequest struct {
    Path struct {
        UserID string `gork:"user_id" validate:"required,uuid"`
    }
    Query struct {
        Force   bool `gork:"force"`
        Notify  bool `gork:"notify"`
    }
    Body struct {
        Name    string  `gork:"name" validate:"min=1,max=100"`
        Email   *string `gork:"email" validate:"omitempty,email"`
        Profile struct {
            Bio  string   `gork:"bio" validate:"max=500"`
            Tags []string `gork:"tags" validate:"dive,min=1,max=20"`
        } `gork:"profile"`
    }
    Headers struct {
        Authorization string `gork:"Authorization" validate:"required"`
        IfMatch       string `gork:"If-Match"`
    }
    Cookies struct {
        SessionID   string `gork:"session_id"`
        Preferences string `gork:"preferences"`
    }
}
```

## Response Structure

### Standard Response Format

```go
type ResponseName struct {
    Body    BodyStruct    // Optional - response body
    Headers HeaderStruct  // Optional - response headers
    Cookies CookieStruct  // Optional - response cookies
}
```

### Example Response

```go
type UpdateUserResponse struct {
    Body struct {
        ID       string    `gork:"id"`
        Name     string    `gork:"name"`
        Updated  time.Time `gork:"updated"`
    }
    Headers struct {
        Location string `gork:"Location"`
        ETag     string `gork:"ETag"`
    }
    Cookies struct {
        SessionToken string `gork:"session_token"`
    }
}
```

## Union Types

Union types work seamlessly with discriminator support:

```go
type EmailLogin struct {
    Type     string `gork:"type,discriminator=email" validate:"required"`
    Email    string `gork:"email" validate:"required,email"`
    Password string `gork:"password" validate:"required"`
}

type PhoneLogin struct {
    Type  string `gork:"type,discriminator=phone" validate:"required"`
    Phone string `gork:"phone" validate:"required,e164"`
    Code  string `gork:"code" validate:"required,len=6"`
}

type LoginRequest struct {
    Body struct {
        LoginMethod unions.Union2[EmailLogin, PhoneLogin] `gork:"login_method" validate:"required"`
        RememberMe  bool `gork:"remember_me"`
    }
}
```

## Custom Validation

### Section-Level Validation

```go
type CreateUserBody struct {
    Name            string `gork:"name" validate:"required"`
    Password        string `gork:"password" validate:"required"`
    ConfirmPassword string `gork:"confirm_password" validate:"required"`
}

func (b *CreateUserBody) Validate() error {
    if b.Password != b.ConfirmPassword {
        return &api.BodyValidationError{
            Errors: []string{"passwords do not match"},
        }
    }
    return nil
}
```

### Request-Level Validation

```go
func (r *TransferFundsRequest) Validate() error {
    if r.Path.FromAccount == r.Body.ToAccount {
        return &api.RequestValidationError{
            Errors: []string{"cannot transfer to the same account"},
        }
    }
    return nil
}
```

## Complex Type Parsing

Register parsers for automatic entity resolution:

```go
// Register in your application initialization
factory := api.NewConventionHandlerFactory()

factory.RegisterTypeParser(func(ctx context.Context, id string) (*User, error) {
    return userService.GetByID(ctx, id)
})

factory.RegisterTypeParser(func(ctx context.Context, s string) (*time.Time, error) {
    t, err := time.Parse(time.RFC3339, s)
    return &t, err
})
```

Then use in requests:

```go
type GetUserRequest struct {
    Path struct {
        User    User      `gork:"user_id"`    // Auto-resolved
        Since   time.Time `gork:"since"`      // Auto-parsed
    }
}
```

## Error Handling

### Validation Errors (HTTP 400)

```json
{
    "error": "Validation failed",
    "details": {
        "query.limit": ["min"],
        "body.name": ["required"],
        "body.email": ["email"],
        "body": ["passwords do not match"],
        "request": ["cannot transfer to same account"]
    }
}
```

### Server Errors (HTTP 500)

Non-ValidationError types from `Validate()` methods result in HTTP 500:

```go
func (b *CreateUserBody) Validate() error {
    // This will cause HTTP 500
    return fmt.Errorf("database connection failed")
}
```

## Migration Guide

### From Legacy to Convention

1. **Identify parameter sources** from `openapi:"in=..."` tags
2. **Group fields by source** (query, body, path, headers, cookies)
3. **Create section structs** for each parameter source
4. **Convert tags**: `json:"name"` → `gork:"name"`
5. **Update handlers**: `req.UserID` → `req.Path.UserID`
6. **Add custom validation** if needed
7. **Test error responses** for proper namespacing

### Example Migration

**Before:**
```go
type UpdateUserRequest struct {
    UserID string `json:"user_id" openapi:"user_id,in=path" validate:"required,uuid"`
    Force  bool   `json:"force" openapi:"force,in=query"`  
    Name   string `json:"name" validate:"required"`
    Auth   string `json:"authorization" openapi:"Authorization,in=header" validate:"required"`
}
```

**After:**
```go
type UpdateUserRequest struct {
    Path struct {
        UserID string `gork:"user_id" validate:"required,uuid"`
    }
    Query struct {
        Force bool `gork:"force"`
    }
    Body struct {
        Name string `gork:"name" validate:"required"`
    }
    Headers struct {
        Authorization string `gork:"Authorization" validate:"required"`
    }
}
```

## Testing

The implementation includes comprehensive tests:

- `convention_parser_test.go`: Parser functionality
- `convention_validation_test.go`: Validation system
- `type_parser_registry_test.go`: Type parser registry

Run tests:
```bash
go test ./pkg/api/...
```

## Linting

Use the enhanced linter to validate structures:

```bash
go run ./cmd/lintgork ./examples/handlers/...
```

## Backward Compatibility

The implementation maintains backward compatibility:

- Legacy handlers using `openapi:"in=..."` tags continue to work
- Convention Over Configuration is automatically detected
- Mixed usage is detected and reported by the linter

## Performance Considerations

- **Reflection caching**: Struct analysis is cached
- **Lazy parsing**: Only sections present in structs are parsed
- **Type parser caching**: Parser registry uses efficient type lookups
- **Memory efficiency**: Field-level parsing minimizes allocations

## Examples

See complete examples in:
- `examples/handlers/users_convention.go`: Convention examples
- `examples/migration_example.go`: Migration patterns
- `examples/handlers/users.go`: Legacy examples

## Future Enhancements

Planned improvements:
- Union accessor method generation
- Enhanced discriminator validation
- Additional type parsers for standard library types
- Performance optimizations
- IDE integration improvements