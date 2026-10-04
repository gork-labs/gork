# Type Resolution Specification and Design

## Executive Summary

This document specifies and designs automatic type resolution for the Gork framework, enabling developers to automatically resolve complex types from path/query parameters without manual parsing. This feature leverages the existing Type Parser Registry infrastructure to provide seamless type resolution, supporting any kind of loader from simple parsers (duration, UUID) to complex entity resolution.

## Problem Statement

Currently, developers must manually:
1. Extract path parameters as primitive types (strings, ints)
2. Parse or resolve those parameters into complex types
3. Handle parsing/resolution errors
4. Repeat this pattern across many handlers

This leads to boilerplate code and inconsistent error handling patterns.

## Solution Overview

Implement a new type codec system to support bidirectional type codecs with OpenAPI schema generation, allowing developers to declare complex type fields that can be parsed from strings, formatted back to strings, and automatically documented:

```go
type User struct {
    ID   int    `gork:"id"`
    Name string `gork:"name"`
}

type Duration time.Duration

type GetUserRequest struct {
    Path struct {
        User     User     `gork:"userId"`    // Automatically parsed from "userId" parameter
        Timeout  Duration `gork:"timeout"`   // Automatically parsed from "timeout" parameter
    }
}

type GetUserResponse struct {
    Author  User     `gork:"author"`  // Automatically formatted to transportable string
    Elapsed Duration `gork:"elapsed"` // Automatically formatted to transportable string
}
```

## Current Architecture Analysis

### Existing Infrastructure

1. **Type Parser Registry** (`pkg/api/type_parser_registry.go`)
   - Supports custom type parsers with signature: `func(ctx context.Context, value string) (*T, error)`
   - Already integrated into `ConventionParser.setFieldValue`
   - Thread-safe with proper error handling

2. **Convention Parser** (`pkg/api/convention_parser.go`)
   - Orchestrates request parsing across all sections (Path, Query, Headers, etc.)
   - Calls type parsers before falling back to basic type conversion
   - Passes `context.Context` for dependency access

3. **Parameter Adapters** (`pkg/adapters/*/`)
   - Framework-specific parameter extraction
   - Support for path, query, header, and cookie parameters

### Integration Points

The type codec system integrates at the `setFieldValue` level in `ConventionParser` for parsing and adds a new `FormatFieldValue` method for formatting, leveraging the existing type parser infrastructure.

## Specification

### Type Codec Interface

```go
// TypeCodec handles bidirectional conversion and provides OpenAPI schema
type TypeCodec[T any] interface {
    // Parse converts transportable values (strings, numbers, booleans) to Go types
    Parse(ctx context.Context, value string) (*T, error)
    
    // Format converts Go types back to transportable format (strings, numbers, booleans, objects)
    Format(ctx context.Context, value *T) (string, error)
    
    // Schema provides OpenAPI schema information for this type
    Schema() OpenAPISchema
}

// OpenAPI type constants
const (
    OpenAPITypeString  = "string"
    OpenAPITypeInteger = "integer" 
    OpenAPITypeNumber  = "number"
    OpenAPITypeBoolean = "boolean"
    OpenAPITypeArray   = "array"
    OpenAPITypeObject  = "object"
)

// OpenAPI format constants  
const (
    OpenAPIFormatUUID     = "uuid"
    OpenAPIFormatDateTime = "date-time"
    OpenAPIFormatDuration = "duration"
    OpenAPIFormatEmail    = "email"
    OpenAPIFormatURI      = "uri"
    OpenAPIFormatByte     = "byte"
    OpenAPIFormatBinary   = "binary"
)

// OpenAPISchema represents OpenAPI schema information for a type codec
type OpenAPISchema struct {
    Type        string      `json:"type"`          // "string", "integer", "number", "boolean", "array", "object"
    Format      string      `json:"format,omitempty"`
    Pattern     string      `json:"pattern,omitempty"`
    Example     interface{} `json:"example,omitempty"`
    Description string      `json:"description,omitempty"`
    MinLength   *int        `json:"minLength,omitempty"`
    MaxLength   *int        `json:"maxLength,omitempty"`
    Minimum     *float64    `json:"minimum,omitempty"`      // For numeric types
    Maximum     *float64    `json:"maximum,omitempty"`      // For numeric types
    Enum        []interface{} `json:"enum,omitempty"`
    Properties  map[string]*OpenAPISchema `json:"properties,omitempty"` // For object types
    Items       *OpenAPISchema `json:"items,omitempty"`       // For array types
}

// TypeCodecEntry contains codec information with type-erased methods
type TypeCodecEntry struct {
    Parser    func(ctx context.Context, value string) (interface{}, error)
    Formatter func(ctx context.Context, value interface{}) (string, error)
    Schema    OpenAPISchema
}

// CodecRegistry manages type codecs for different types
type CodecRegistry interface {
    // Register registers a codec
    Register[T any](codec TypeCodec[T]) error
    
    // Parser returns the parser for the given type
    Parser(targetType reflect.Type) (func(ctx context.Context, value string) (interface{}, error), bool)
    
    // Formatter returns the formatter for the given type
    Formatter(targetType reflect.Type) (func(ctx context.Context, value interface{}) (string, error), bool)
    
    // Schema returns the schema for the given type
    Schema(targetType reflect.Type) (OpenAPISchema, bool)
    
    // HasParser returns true if a parser is registered for the given type
    HasParser(targetType reflect.Type) bool
}
```

### Context Requirements

Type codecs receive `context.Context` which may contain:
- Configuration values
- Request metadata
- External service connections
- Caching layers
- Logging/tracing context

### Struct Tag Convention

The codec system uses `gork:` struct tags consistently for all field naming:
- **Request parsing**: `gork:"created_after"` specifies HTTP parameter name
- **Response formatting**: `gork:"created_after"` specifies output field name
- **Convention compliance**: Aligns with Gork's structured request/response format

```go
type Request struct {
    Query struct {
        StartTime time.Time `gork:"start_ts"`  // HTTP: ?start_ts=1703500200
    }
}

type Response struct {
    StartTime time.Time `gork:"startTime"`  // Output: {"startTime": "1703500200"}
}
```

### Error Handling

Type codecs should return appropriate errors:
- `ErrInvalidFormat` - invalid input format (maps to 400)
- `ErrFormatFailed` - formatting failed (maps to 500)
- `ErrParseFailed` - parsing failed (maps to 400)
- `ErrExternalServiceError` - external service error (maps to 500)

### Registration API

```go
// Global codec registry for PoC
var globalCodecRegistry = NewCodecRegistry()

// RegisterCodec registers a codec for a type using the global registry
func RegisterCodec[T any](codec TypeCodec[T]) error {
    return globalCodecRegistry.Register(codec)
}
```

### Built-in Type Codecs

Gork provides built-in codecs for special types that need custom parsing beyond regular JSON serialization:

```go
// TimeCodec handles RFC3339 parsing/formatting with OpenAPI schema
type TimeCodec struct{}

func (c TimeCodec) Parse(ctx context.Context, value string) (*time.Time, error) {
    t, err := time.Parse(time.RFC3339, value)
    if err != nil {
        return nil, fmt.Errorf("invalid time format: %w", err)
    }
    return &t, nil
}

func (c TimeCodec) Format(ctx context.Context, value *time.Time) (string, error) {
    if value == nil {
        return "", nil
    }
    return value.Format(time.RFC3339), nil
}

func (c TimeCodec) Schema() OpenAPISchema {
    return OpenAPISchema{
        Type:        "string",
        Format:      "date-time",
        Pattern:     "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d{3})?Z?$",
        Example:     "2023-12-25T10:30:00Z",
        Description: "RFC3339 formatted timestamp",
    }
}

// Alternative: Unix timestamp codec for numeric time representation
type UnixTimeCodec struct{}

func (c UnixTimeCodec) Parse(ctx context.Context, value string) (*time.Time, error) {
    timestamp, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return nil, fmt.Errorf("invalid timestamp: %w", err)
    }
    t := time.Unix(timestamp, 0)
    return &t, nil
}

func (c UnixTimeCodec) Format(ctx context.Context, value *time.Time) (string, error) {
    if value == nil {
        return "", nil
    }
    return strconv.FormatInt(value.Unix(), 10), nil
}

func (c UnixTimeCodec) Schema() OpenAPISchema {
    return OpenAPISchema{
        Type:        "integer",
        Format:      "int64",
        Example:     1703500200,
        Description: "Unix timestamp in seconds",
    }
}

// Complex object codec (JSON serialization for objects)
type User struct {
    ID   int    `gork:"id"`
    Name string `gork:"name"`
}

type UserCodec struct{}

func (c UserCodec) Parse(ctx context.Context, value string) (*User, error) {
    var user User
    if err := json.Unmarshal([]byte(value), &user); err != nil {
        return nil, fmt.Errorf("invalid user JSON: %w", err)
    }
    return &user, nil
}

func (c UserCodec) Format(ctx context.Context, value *User) (string, error) {
    if value == nil {
        return "{}", nil
    }
    data, err := json.Marshal(value)
    if err != nil {
        return "", fmt.Errorf("failed to format user: %w", err)
    }
    return string(data), nil
}

func (c UserCodec) Schema() OpenAPISchema {
    return OpenAPISchema{
        Type:        "object",
        Example:     map[string]interface{}{"id": 123, "name": "John Doe"},
        Description: "User object as JSON",
        Properties: map[string]*OpenAPISchema{
            "id": {
                Type:        "integer",
                Description: "User ID",
            },
            "name": {
                Type:        "string",
                Description: "User name",
            },
        },
    }
}
```

**Note:** Basic types like `string`, `int`, and `bool` are handled automatically by the regular JSON serialization system (gorkson) and don't require custom codecs.

### Caching Strategy

Type codecs can implement caching if needed for expensive operations.

## Design Details

### Core Components

#### 1. Codec Registry

```go
// pkg/api/codec_registry.go
package api

import (
    "fmt"
    "reflect"
    "regexp"
    "sync"
)

// codecRegistry manages type codecs for different types
type codecRegistry struct {
    mu      sync.RWMutex
    entries map[reflect.Type]TypeCodecEntry
}

// NewCodecRegistry creates a new codec registry
func NewCodecRegistry() CodecRegistry {
    return &codecRegistry{
        entries: make(map[reflect.Type]TypeCodecEntry),
    }
}

// Register registers a parser with required schema and optional formatter
func (r *codecRegistry) Register(targetType reflect.Type, parser TypeParser, schema SchemaProvider, opts ...CodecOption) error {
    if targetType == nil {
        return fmt.Errorf("target type cannot be nil")
    }
    if parser == nil {
        return fmt.Errorf("parser cannot be nil")
    }
    if schema == nil {
        return fmt.Errorf("schema provider cannot be nil")
    }
    
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.entries[targetType]; exists {
        return fmt.Errorf("type codec for type %s already registered", targetType)
    }
    
    entry := TypeCodecEntry{
        Parser: parser,
        Schema: schema,
    }
    
    // Apply options
    for _, opt := range opts {
        opt(&entry)
    }
    
    r.entries[targetType] = entry
    return nil
}

// Parser returns the parser for the given type
func (r *codecRegistry) Parser(targetType reflect.Type) (func(ctx context.Context, value string) (interface{}, error), bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    entry, exists := r.entries[targetType]
    if !exists {
        return nil, false
    }
    return entry.Parser, true
}

// Formatter returns the formatter for the given type (if codec implements TypeFormatter)
func (r *codecRegistry) Formatter(targetType reflect.Type) (func(ctx context.Context, value interface{}) (string, error), bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    entry, exists := r.entries[targetType]
    if !exists || entry.Formatter == nil {
        return nil, false
    }
    return entry.Formatter, true
}

// Schema returns the schema provider for the given type
func (r *codecRegistry) Schema(targetType reflect.Type) (SchemaProvider, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    entry, exists := r.entries[targetType]
    if !exists {
        return nil, false
    }
    return entry.Schema, true
}

// HasParser returns true if a parser is registered for the given type
func (r *codecRegistry) HasParser(targetType reflect.Type) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    _, exists := r.entries[targetType]
    return exists
}
```

#### 2. Integration with ConventionParser

```go
// ConventionParser uses global type codec registry
type ConventionParser struct {
    // typeRegistry removed - replaced by global codec registry
    validator     *validator.Validate
}

func NewConventionParser() *ConventionParser {
    return &ConventionParser{
        // typeRegistry removed - replaced by global codec registry
        validator:     validator.New(),
    }
}



// Modify setFieldValue to check for type codecs
func (p *ConventionParser) setFieldValue(ctx context.Context, fieldValue reflect.Value, field reflect.StructField, value string) error {
    // First try global codec registry parsers
    if parser, exists := globalCodecRegistry.Parser(field.Type); exists {
        result, err := parser(ctx, value)
        if err != nil {
            return fmt.Errorf("codec parse error: %w", err)
        }
        fieldValue.Set(reflect.ValueOf(result).Elem())
        return nil
    }
    
    // Then try complex type parsing (for backward compatibility)
    if parser := p.typeRegistry.GetParser(field.Type); parser != nil {
        result, err := parser(ctx, value)
        if err != nil {
            return err
        }
        fieldValue.Set(reflect.ValueOf(result).Elem())
        return nil
    }
    
    // Fall back to basic type conversion
    return p.setBasicFieldValue(fieldValue, field, value)
}

// FormatFieldValue converts Go types back to strings for responses
func (p *ConventionParser) FormatFieldValue(ctx context.Context, value interface{}) (string, error) {
    valueType := reflect.TypeOf(value)
    if formatter, exists := globalCodecRegistry.Formatter(valueType); exists {
        return formatter(ctx, value)
    }
    
    // Fall back to string conversion
    return fmt.Sprintf("%v", value), nil
}
```

#### 3. Codec Registration Helpers

```go
// pkg/api/codec_helpers.go
package api

import (
    "reflect"
)

// RegisterCodec provides a type-safe way to register a codec globally
func RegisterCodec[T any](codec TypeCodec[T]) error {
    return globalCodecRegistry.Register(codec)
}
```

#### 4. Standard Error Types

```go
// pkg/api/codec_errors.go
package api

import (
    "errors"
    "fmt"
    "net/http"
)

var (
    // ErrInvalidFormat indicates invalid input format
    ErrInvalidFormat = errors.New("invalid format")
    
    // ErrParseFailed indicates parsing failed
    ErrParseFailed = errors.New("parse failed")
    
    // ErrFormatFailed indicates formatting failed
    ErrFormatFailed = errors.New("format failed")
    
    // ErrExternalServiceError indicates external service error
    ErrExternalServiceError = errors.New("external service error")
)



func (e *CodecError) Unwrap() error {
    return e.Cause
}

// HTTPStatusCode returns the appropriate HTTP status code for the error
func (e *CodecError) HTTPStatusCode() int {
    if errors.Is(e.Cause, ErrInvalidFormat) || errors.Is(e.Cause, ErrParseFailed) {
        return http.StatusBadRequest
    }
    return http.StatusInternalServerError
}

// NewInvalidFormatError creates a new invalid format error
func NewInvalidFormatError(typeName, value string, details ...string) *CodecError {
    err := &CodecError{
        Type:      typeName,
        Value:     value,
        Operation: "parse",
        Cause:     ErrInvalidFormat,
    }
    
    if len(details) > 0 {
        err.Details = details[0]
    }
    
    return err
}

// CodecError wraps type codec errors with additional context
type CodecError struct {
    Type      string
    Value     string
    Operation string
    Details   string
    Cause     error
}

func (e *CodecError) Error() string {
    if e.Details != "" {
        return fmt.Sprintf("failed to %s %s from/to value '%s': %s (%v)", e.Operation, e.Type, e.Value, e.Details, e.Cause)
    }
    return fmt.Sprintf("failed to %s %s from/to value '%s': %v", e.Operation, e.Type, e.Value, e.Cause)
}

// NewFormatError creates a new formatting error
func NewFormatError(typeName, value string) *CodecError {
    return &CodecError{
        Type:      typeName,
        Value:     value,
        Operation: "format",
        Cause:     ErrFormatFailed,
    }
}
```

### Integration with Existing Systems

#### 1. Router Integration

```go
// Each router adapter can set up context with required dependencies
func (r *Router) setupResolverContext(ctx context.Context, req *http.Request) context.Context {
    // Add any required dependencies to context for type resolvers
    ctx = context.WithValue(ctx, "config", r.config)
    ctx = context.WithValue(ctx, "services", r.services)
    if userID := r.extractUserID(req); userID != "" {
        ctx = context.WithValue(ctx, "userID", userID)
    }
    return ctx
}
```

#### 2. OpenAPI Generation

Type codec fields generate OpenAPI parameter definitions using registered schema information:

```go
// In ConventionOpenAPIGenerator
func (g *ConventionOpenAPIGenerator) processPathSection(sectionType reflect.Type, operation *Operation, components *Components) {
    for i := 0; i < sectionType.NumField(); i++ {
        field := sectionType.Field(i)
        gorkTag := field.Tag.Get("gork")
        
        if gorkTag == "" {
            continue
        }
        
        tagInfo := parseGorkTag(gorkTag)
        
        // Check if this is a type codec field
        if globalCodecRegistry.HasParser(field.Type) {
            // Get the schema directly from the codec
            schema, _ := globalCodecRegistry.Schema(field.Type)
            
            param := Parameter{
                Name:        tagInfo.Name,
                In:          "path",
                Required:    true,
                Schema:      convertToOpenAPISchema(schema),
            }
            operation.Parameters = append(operation.Parameters, param)
            continue
        }
        
        // Regular parameter processing...
    }
}

// generateExampleValue creates example values using codec formatter if available
func (g *ConventionOpenAPIGenerator) generateExampleValue(fieldType reflect.Type, schema OpenAPISchema) interface{} {
    // Try to generate an example using the codec formatter
    if formatter, exists := globalCodecRegistry.Formatter(fieldType); exists {
        // Create a zero value of the type and try to format it
        zeroValue := reflect.Zero(fieldType).Interface()
        if exampleStr, err := formatter(context.Background(), zeroValue); err == nil {
            return exampleStr
        }
    }
    
    // Fall back to schema example or type-based defaults
    if schema.Example != nil {
        return schema.Example
    }
    return getDefaultExample(schema.Type, schema.Format)
}

// convertToOpenAPISchema converts internal OpenAPISchema to OpenAPI spec Schema
func convertToOpenAPISchema(schema OpenAPISchema) *Schema {
    result := &Schema{
        Type:        schema.Type,
        Format:      schema.Format,
        Pattern:     schema.Pattern,
        MinLength:   schema.MinLength,
        MaxLength:   schema.MaxLength,
        Example:     schema.Example,
        Description: schema.Description,
        Enum:        schema.Enum,
    }
    
    return result
}
```

## Usage Examples

### Comprehensive Usage Example

```go
// SearchRequest demonstrates various transportable types
type SearchRequest struct {
    Query struct {
        Query       string    `gork:"q"`           // String parameter (handled by gorkson)
        Limit       int       `gork:"limit"`       // Integer parameter (handled by gorkson)
        Active      bool      `gork:"active"`      // Boolean parameter (handled by gorkson)
        CreatedAfter time.Time `gork:"created_after"` // Time parameter (uses TimeCodec)
        User        User      `gork:"user"`        // Complex object parameter (uses UserCodec)
    }
}

type SearchResponse struct {
    Results []string  `gork:"results"`     // Array of strings
    Count   int       `gork:"count"`       // Integer
    HasMore bool      `gork:"has_more"`    // Boolean
    Query   SearchQuery `gork:"query"`     // Complex object
}

type SearchQuery struct {
    Term   string `gork:"term"`
    Limit  int    `gork:"limit"`
    Active bool   `gork:"active"`
}

func SearchAPI(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
    // Parameters are parsed by their respective systems:
    // - req.Query.Query: standard string parsing (gorkson)
    // - req.Query.Limit: standard integer parsing (gorkson)
    // - req.Query.Active: standard boolean parsing (gorkson)
    // - req.Query.CreatedAfter: TimeCodec for RFC3339 parsing
    // - req.Query.User: UserCodec for complex object parsing
    
    results := performSearch(req.Query.Query, req.Query.Limit, req.Query.Active)
    
    return &SearchResponse{
        Results: results,
        Count:   len(results),
        HasMore: len(results) >= req.Query.Limit,
        Query: SearchQuery{
            Term:   req.Query.Query,
            Limit:  req.Query.Limit,
            Active: req.Query.Active,
        },
    }, nil
}

// Registration of custom codecs only
func init() {
    RegisterCodec[time.Time](TimeCodec{})    // Time codec (RFC3339)
    RegisterCodec[User](UserCodec{})         // Complex object codec
}
```

**HTTP Request Examples:**
```
# String format
GET /search?q=golang&limit=10&active=true&created_after=2023-12-25T10:30:00Z&user={"id":123,"name":"John"}

# Numeric timestamp format  
GET /search?q=golang&limit=10&active=1&created_after=1703500200&user={"id":123,"name":"John"}
```

```go
// SearchUsersReq demonstrates codec usage for time.Time fields
type SearchUsersReq struct {
    Query struct {
        CreatedAfter time.Time `gork:"created_after"`
    }
}

type SearchUsersResp struct {
    Users []User `gork:"users"`
    Query struct {
        CreatedAfter time.Time `gork:"created_after"` // Formatted back to transportable string by codec
    } `gork:"query"`
}

func SearchUsers(ctx context.Context, req SearchUsersReq) (*SearchUsersResp, error) {
    // req.Query.CreatedAfter is automatically parsed from "2023-12-25T10:30:00Z"
    users := findUsersCreatedAfter(req.Query.CreatedAfter)
    
    return &SearchUsersResp{
        Users: users,
        Query: struct {
            CreatedAfter time.Time `gork:"created_after"`
        }{
            CreatedAfter: req.Query.CreatedAfter, // Will be formatted back to transportable string
        },
    }, nil
}

// Setup
func init() {
    RegisterCodec[time.Time](TimeCodec{})
}


```

### Entity ID Resolution Example

This example demonstrates how to parse entity IDs from path parameters to full structs (fetched from database) and format them back to IDs for responses:

```go
// User entity 
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Email string `json:"email"`
}

// UserService interface for dependency injection
type UserService interface {
    GetByID(ctx context.Context, id int) (*User, error)
}

// UserCodec handles parsing user IDs to User structs and formatting back to IDs
type UserCodec struct {
    userService UserService
}

func NewUserCodec(userService UserService) UserCodec {
    return UserCodec{userService: userService}
}

func (c UserCodec) Parse(ctx context.Context, value string) (*User, error) {
    // Parse the ID from string
    userID, err := strconv.Atoi(value)
    if err != nil {
        return nil, fmt.Errorf("invalid user ID format: %w", err)
    }
    
    // Fetch the full user from database
    user, err := c.userService.GetByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch user: %w", err)
    }
    
    return user, nil
}

func (c UserCodec) Format(ctx context.Context, value *User) (string, error) {
    if value == nil {
        return "", fmt.Errorf("cannot format nil user")
    }
    
    // Format the user back to just its ID for transport
    return strconv.Itoa(value.ID), nil
}

func (c UserCodec) Schema() OpenAPISchema {
    return OpenAPISchema{
        Type:        "integer",
        Format:      "int32",
        Example:     123,
        Description: "User ID that will be resolved to full user entity",
        Minimum:     ptr(1.0),
    }
}

// Usage in request/response structs
type GetUserPostsRequest struct {
    Path struct {
        User User `gork:"userId"`  // "123" -> fetches User{ID:123, Name:"...", Email:"..."}
    }
}

type GetUserPostsResponse struct {
    Posts []Post `json:"posts"`
    Author User  `json:"author"`  // User{ID:123, Name:"...", Email:"..."} -> "123" in path params
}

func GetUserPosts(ctx context.Context, req GetUserPostsRequest) (*GetUserPostsResponse, error) {
    // req.Path.User is already a full User struct fetched from DB!
    posts := findPostsByUser(req.Path.User.ID)
    
    return &GetUserPostsResponse{
        Posts:  posts,
        Author: req.Path.User,  // Will be formatted back to ID string when needed
    }, nil
}

// Registration with dependency injection
func setupEntityCodecs(userService UserService) {
    RegisterCodec[User](NewUserCodec(userService))
}
```

**Example HTTP Request:**
```
GET /users/123/posts
# Path parameter "123" is automatically:
# 1. Parsed as integer 123
# 2. Used to fetch User{ID: 123, Name: "John Doe", Email: "john@example.com"} from DB
# 3. Made available as req.Path.User in handler
```

**Generated OpenAPI:**
```yaml
paths:
  /users/{userId}/posts:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
            format: int32
            minimum: 1
            example: 123
            description: "User ID that will be resolved to full user entity"
```

This pattern is especially useful for:
- **Entity resolution**: Converting IDs to full entities automatically
- **Dependency injection**: Codecs can access services through context or constructor injection  
- **Caching**: Entity codecs can implement caching for frequently accessed entities
- **Authorization**: Codecs can perform access checks during entity resolution
- **Validation**: Ensure entities exist and user has access before handler execution

### Generated OpenAPI Example

With the above type codec registrations, the generated OpenAPI specification would include properly typed parameters:

```yaml
# For UUID type codec
paths:
  /tasks/{taskId}:
    get:
      parameters:
        - name: taskId
          in: path
          required: true
          schema:
            type: string
            format: uuid
            pattern: '^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
            example: '550e8400-e29b-41d4-a716-446655440000'
            description: 'Task unique identifier'

# For Duration type codec
  /tasks/{taskId}/timeout/{timeout}:
    post:
      parameters:
        - name: timeout
          in: path
          required: true
          schema:
            type: string
            format: duration
            pattern: '^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$'
            example: '5m30s'
            description: 'Task timeout duration'

# For Priority enum type codec
  /issues/{issueId}/priority/{priority}:
    put:
      parameters:
        - name: priority
          in: path
          required: true
          schema:
            type: string
            enum: ['low', 'medium', 'high']
            example: 'medium'
            description: 'Issue priority level'
```

This provides:
- **Proper validation** - API clients get immediate validation feedback
- **Better documentation** - Clear examples and descriptions
- **Code generation** - Tools can generate properly typed client code
- **IDE support** - Better autocomplete and validation in API clients

## Implementation Plan (PoC)

### Phase 1: Core Codec Infrastructure

1. **Create TypeCodec[T] interface** with Parse, Format, and Schema methods
2. **Create global CodecRegistry** (`pkg/api/codec_registry.go`) with type-erased storage
3. **Extend ConventionParser** to use codec registry for parsing
4. **Add FormatFieldValue method** for response formatting
5. **Write comprehensive tests** for parse/format functionality

### Phase 2: time.Time Codec Implementation

1. **Implement TimeCodec** with RFC3339 parsing/formatting
2. **Add OpenAPI schema generation** for time.Time fields
3. **Update ConventionOpenAPIGenerator** to use codec schemas
4. **Test end-to-end** with time.Time in request/response structs
5. **Validate OpenAPI output** includes proper date-time schema

### Phase 3: Additional Common Types (Optional)

1. **Add Duration codec** for time.Duration parsing
2. **Add UUID codec** for uuid.UUID types
3. **Add enum validation** pattern for string-based enums
4. **Test complex scenarios** with multiple codec types

## Automatic Schema Validation

The framework provides automatic validation for all schema constraints defined in codec schemas. When a codec provides constraints in its `OpenAPISchema`, the system automatically validates input values against these constraints before calling the codec's `Parse` method.

### Supported Automatic Validations

```go
func (c EmailCodec) Schema() api.OpenAPISchema {
    return api.OpenAPISchema{
        Type:      api.OpenAPITypeString,
        Format:    api.OpenAPIFormatEmail,
        Pattern:   "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", // Regex validation
        MinLength: ptr(5),   // Minimum string length
        MaxLength: ptr(254), // Maximum string length (RFC 5321)
        Enum:      []interface{}{"admin@example.com", "user@example.com"}, // Enum validation
    }
}

func (c NumericCodec) Schema() api.OpenAPISchema {
    return api.OpenAPISchema{
        Type:    api.OpenAPITypeInteger,
        Minimum: ptr(1.0),   // Minimum numeric value
        Maximum: ptr(100.0), // Maximum numeric value
    }
}

func (c EmailCodec) Parse(ctx context.Context, value string) (Email, error) {
    // All schema validation already done automatically:
    // - Pattern validation (email regex)
    // - Length validation (5-254 characters)
    // - Enum validation (if specified)
    // Focus only on business logic here
    return Email(value), nil
}
```

### Validation Order

Validations are applied in this order:
1. **Pattern validation** - Regex pattern matching
2. **Length validation** - MinLength/MaxLength for strings
3. **Range validation** - Minimum/Maximum for numbers
4. **Enum validation** - Value must be in allowed enum list
5. **Codec Parse** - Custom parsing logic

### Benefits

- **DRY Principle** - No duplicate validation logic in codecs
- **Consistent errors** - Standardized validation error messages  
- **Performance** - Regex compiled once, efficient validation
- **OpenAPI integration** - Same constraints used for docs and validation
- **Automatic** - Works for any codec with schema constraints
- **Layered** - Schema validation + custom codec logic

## Input Validation

Type codecs should implement proper input validation in both directions:

```go
type ValidatedEmailCodec struct{}

func (c ValidatedEmailCodec) Parse(ctx context.Context, value string) (Email, error) {
    // Pattern validation is automatically handled by the registry
    // Additional validation logic can be added here if needed
    return Email(value), nil
}

func (c ValidatedEmailCodec) Format(ctx context.Context, value Email) (string, error) {
    if value == "" {
        return "", api.NewFormatError("Email", "empty")
    }
    return string(value), nil
}

func (c ValidatedEmailCodec) Schema() api.OpenAPISchema {
    return api.OpenAPISchema{
        Type:    api.OpenAPITypeString,
        Format:  api.OpenAPIFormatEmail,
        Pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", // Automatic validation
        MaxLength: ptr(254), // RFC 5321 limit
    }
}

func createEmailCodec() TypeCodec[Email] {
    return ValidatedEmailCodec{}
}
```

## Testing Strategy

### Unit Tests

- Test type codec registration and retrieval (both marshal and unmarshal)
- Test error handling for various failure scenarios in both directions
- Test helper functions and schema validation
- Test bidirectional conversion consistency (parse then format should be identity)

### Integration Tests

- Test end-to-end type codec usage in request/response handlers
- Test OpenAPI generation for type codec fields with proper schemas
- Test response formatting using formatters

## Migration Guide

### From Manual Parsing to Codec Registry

**Before:**
```go
type GetTaskRequest struct {
    Path struct {
        TaskID  string `gork:"taskId" validate:"required,uuid"`
        Timeout string `gork:"timeout" validate:"required"`
    }
}

func GetTask(ctx context.Context, req GetTaskRequest) (*TaskResponse, error) {
    // Manual UUID parsing
    taskUUID, err := uuid.Parse(req.Path.TaskID)
    if err != nil {
        return nil, fmt.Errorf("invalid task ID: %w", err)
    }
    
    // Manual duration parsing
    timeout, err := time.ParseDuration(req.Path.Timeout)
    if err != nil {
        return nil, fmt.Errorf("invalid timeout: %w", err)
    }
    
    // Manual formatting for response
    return &TaskResponse{
        TaskID:  req.Path.TaskID,   // String format
        Timeout: timeout.String(),  // Manual conversion
    }, nil
}
```

**After:**
```go
type GetTaskRequest struct {
    Path struct {
        TaskID  UUID     `gork:"taskId"`   // Automatically unmarshaled
        Timeout Duration `gork:"timeout"`  // Automatically unmarshaled
    }
}

type TaskResponse struct {
    TaskID  UUID     `json:"taskId"`   // Can be formatted if formatter registered
    Timeout Duration `json:"timeout"`  // Can be formatted if formatter registered
}

func GetTask(ctx context.Context, req GetTaskRequest) (*TaskResponse, error) {
    return &TaskResponse{
        TaskID:  req.Path.TaskID,   // Already parsed!
        Timeout: req.Path.Timeout,  // Already parsed!
    }, nil
}

// Setup is now simple with global registration
func setupServer() {
    // Register codecs globally
    api.RegisterCodec[UUID](UUIDCodec{})
    api.RegisterCodec[Duration](DurationCodec{})
    
    parser := api.NewConventionParser()
    // Use parser...
}
```

## Conclusion

This simplified design provides a practical codec system for Gork's PoC version, focusing on the core need to convert fields in conventional request/response structs to transportable format.

**Key benefits for PoC:**
- **Single interface** - TypeCodec[T] combines parsing, formatting, and schema in one place
- **Full type support** - handles strings, numbers, booleans, and complex objects
- **Global registry** - simple registration and access pattern for PoC
- **OpenAPI integration** - automatic schema generation with examples and validation
- **Complete replacement** - replaces the deprecated TypeParserRegistry system
- **Type safe** - full compile-time type checking with generics
- **Minimal boilerplate** - eliminates manual parsing code for all types
- **Consistent tagging** - uses `gork:` tags throughout for field naming
- **Format flexibility** - supports multiple representations (strings, numbers, JSON objects)

**Design principles:**
- **Parse** - convert strings from request parameters to Go types
- **Format** - convert Go types back to transportable strings
- **Schema** - provide OpenAPI documentation for the conversion
- **Convention over configuration** - automatic detection and processing
- **Global registry** - simple access pattern suitable for PoC

The implementation provides a solid foundation for automatic type conversion while maintaining the simplicity needed for a proof of concept. The design can be extended later with more sophisticated features as needed.

