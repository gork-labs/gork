# Gorkson Codec Integration Specification

## Overview

This specification proposes moving the Type Codec system from the `pkg/api` package to the `pkg/gorkson` package, making gorkson the universal data conversion layer for all transport ↔ Go type conversions in the Gork framework.

## Current Architecture Problems

### 1. Duplicated Type Conversion Logic

**Two separate systems doing the same thing:**

```go
// pkg/gorkson/gorkson.go - JSON conversion
func (m *Marshaler) setFieldValue(field reflect.Value, value any) error {
    // Handles: string, int, float, bool, struct, pointer, slice conversion
}

// pkg/api/convention_parser.go - HTTP parameter conversion  
func (p *ConventionParser) setFieldValue(ctx context.Context, fieldValue reflect.Value, field reflect.StructField, value string) error {
    // First try codec registry
    // Then fallback to basic conversion
}
```

### 2. Inconsistent Type Support

- **gorkson**: Only supports basic types (string, int, float, bool) + JSON marshaling fallback
- **api codecs**: Supports custom type conversion with validation and OpenAPI schemas
- **Result**: JSON body parsing can't use the same rich type conversion as HTTP parameters

### 3. Architecture Mismatch

```
Current (WRONG):
HTTP Request → api.ConventionParser → codec conversion → Go struct
JSON Body    → gorkson.Marshaler   → basic conversion → Go struct
```

```
Should be (RIGHT):
HTTP Request → api.ConventionParser → gorkson (with codecs) → Go struct  
JSON Body    → gorkson (with codecs) → Go struct
Response     → gorkson (with codecs) → transport format
```

### 4. Limited Codec Usage

Codecs are only available for HTTP parameters, not for:
- JSON body fields with custom types
- Response formatting consistency
- Other transport formats

## Proposed Architecture

### 1. Gorkson as Universal Data Conversion Layer

**Move codec system to gorkson:**

```go
// pkg/gorkson/codec.go
type TypeCodec[T any] interface {
    Parse(ctx context.Context, value string) (*T, error)
    Format(ctx context.Context, value *T) (string, error)
    Schema() OpenAPISchema
}

// Global codec registry in gorkson
var globalCodecRegistry = NewCodecRegistry()

func RegisterCodec[T any](codec TypeCodec[T]) error {
    return globalCodecRegistry.Register(codec)
}
```

**Enhanced Marshaler with codec support:**

```go
// pkg/gorkson/gorkson.go
func (m *Marshaler) setFieldValue(field reflect.Value, value any) error {
    // 1. Try codec conversion (NEW)
    if codec, exists := globalCodecRegistry.GetCodec(field.Type()); exists {
        return m.setFieldWithCodec(field, value, codec)
    }
    
    // 2. Fall back to current basic conversion
    return m.setBasicFieldValue(field, value)
}
```

### 2. API Layer Delegation

**api.ConventionParser becomes a thin wrapper:**

```go
// pkg/api/convention_parser.go
func (p *ConventionParser) setFieldValue(ctx context.Context, fieldValue reflect.Value, field reflect.StructField, value string) error {
    // Delegate to gorkson with context
    return gorkson.SetFieldValueFromString(ctx, fieldValue, value)
}
```

### 3. Unified Type Conversion

**All conversion flows through gorkson:**

```go
// HTTP parameters
api.ConventionParser → gorkson.SetFieldValueFromString() → codec system → basic conversion

// JSON body parsing  
gorkson.Unmarshal() → codec system → gorkson.setFieldValue()

// Response formatting
gorkson.Marshal() → codec system → gorkson.convertToGorkSON()
```

## Benefits

### 1. Consistent Type Support Everywhere

```go
// Same Priority enum codec works for:
type GetTaskRequest struct {
    Path struct {
        Priority Priority `gork:"priority"`  // HTTP parameter
    }
    Body struct {
        NewPriority Priority `gork:"new_priority"`  // JSON field
    }
}
```

### 2. Reduced Code Duplication

- Single `setFieldValue` implementation
- One codec registry for all transport formats
- Unified error handling and validation

### 3. Enhanced JSON Processing

```go
// JSON can now use rich codecs
{
    "task_id": "123",           // TaskCodec resolves to full Task entity
    "created_at": "2023-12-25T10:30:00Z",  // TimeCodec handles RFC3339
    "priority": "high"          // PriorityCodec converts to enum
}
```

### 4. Future Transport Format Support

- GraphQL resolvers
- Protocol Buffers
- MessagePack
- Any format can use the same codec system

## Implementation Plan

### Phase 1: Move Codec System to Gorkson

1. **Create new files in pkg/gorkson:**
   - `codec.go` - Core codec interfaces and types
   - `codec_registry.go` - Registry implementation
   - `builtin_codecs.go` - Time, UUID, etc.
   - `codec_errors.go` - Error types

2. **Enhance gorkson.Marshaler:**
   - Add codec support to `setFieldValue`
   - Add codec support to `convertToGorkSON` (for formatting)
   - Add `SetFieldValueFromString` public API

### Phase 2: Update API Layer

1. **Simplify api.ConventionParser:**
   - Remove codec registry field
   - Delegate `setFieldValue` to gorkson
   - Remove `FormatFieldValue` (use gorkson)

2. **Update OpenAPI generator:**
   - Get schemas from gorkson codec registry
   - Remove duplicate schema logic

### Phase 3: Testing and Documentation

1. **Update examples and documentation**
2. **Comprehensive testing of all transport formats**

## API Changes

### New Public APIs (gorkson)

```go
// pkg/gorkson/codec.go
func RegisterCodec[T any](codec TypeCodec[T]) error
func SetFieldValueFromString(ctx context.Context, field reflect.Value, value string) error
func FormatFieldValueToString(ctx context.Context, value interface{}) (string, error)
func GetCodecRegistry() CodecRegistry
```



### Enhanced JSON Processing

```go
// JSON unmarshaling now supports codecs automatically
var req struct {
    CreatedAt time.Time `gork:"created_at"`  // Uses TimeCodec
    Task      Task      `gork:"task"`        // Uses TaskCodec  
}

err := gorkson.Unmarshal(jsonData, &req)
// CreatedAt parsed with RFC3339, Task resolved from ID
```

## Implementation Strategy

### Direct Implementation

Since this is a new project with no existing users:

1. **Move codec system directly to gorkson**
2. **Update all existing code to use gorkson APIs**  
3. **Remove duplicate implementations from api package**
4. **Update all examples and tests**

No backward compatibility needed - just implement the correct architecture from the start.

## Testing Strategy

### 1. Comprehensive Integration Tests

```go
func TestUniversalTypeConversion(t *testing.T) {
    gorkson.RegisterCodec[Priority](PriorityCodec{})
    
    // Test HTTP parameter parsing
    testHTTPParameterConversion(t)
    
    // Test JSON body parsing  
    testJSONBodyConversion(t)
    
    // Test response formatting
    testResponseFormatting(t)
}
```



## File Structure Changes

### New Files

```
pkg/gorkson/
├── codec.go              # Core codec interfaces
├── codec_registry.go     # Registry implementation  
├── builtin_codecs.go     # Time, UUID, etc.
├── codec_errors.go       # Error types
├── codec_integration_test.go
└── gorkson.go            # Enhanced with codec support
```

### Files to Update

```
pkg/api/
├── convention_parser.go  # Simplified, delegates to gorkson
└── openapi_generator.go  # Uses gorkson codec registry
```

### Files to Move

```
pkg/api/ → pkg/gorkson/
├── codec_registry.go     
├── codec_helpers.go        
├── builtin_codecs.go     
├── codec_errors.go       
├── type_codec.go         
└── *codec*test.go        # All codec-related tests
```

## Benefits Summary

1. **Architectural Clarity**: gorkson becomes the single data conversion layer
2. **Code Reuse**: One codec system for all transport formats  
3. **Consistency**: Same type conversion everywhere
4. **Extensibility**: Easy to add new transport formats
5. **Performance**: Eliminate duplicate conversion logic
6. **Maintainability**: Single source of truth for type conversion

## Risks and Mitigation

### Risk: Increased gorkson Complexity  
**Mitigation**: Clear separation of concerns, good documentation

## Success Criteria

1. ✅ JSON body parsing supports codecs  
2. ✅ Single codec registration API via gorkson
3. ✅ Comprehensive test coverage
4. ✅ Updated documentation and examples

## Conclusion

The codec system belongs in gorkson because **gorkson is the universal data conversion layer**, not just a JSON marshaler. By implementing codecs at the data conversion level, we create a unified, powerful system where:

- All transport formats benefit from rich type conversion
- Code is cleaner and more maintainable  
- The architecture matches the actual data flow
- Future extensibility is greatly enhanced

This refactoring transforms gorkson from a simple JSON marshaler into the universal type conversion engine that powers all of Gork's transport layer capabilities.
