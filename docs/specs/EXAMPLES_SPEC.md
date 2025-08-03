# OpenAPI Examples Specification

## Overview

This specification defines how to add examples to OpenAPI request/response structs in the Gork toolkit using a hybrid approach of struct tags and method-based examples.

## Goals

1. **Simple field-level examples** using struct tags for basic cases
2. **Complex instance examples** using typed methods for advanced scenarios  
3. **Type safety** with Go generics
4. **OpenAPI 3.1 compliance** mapping to standard examples structure
5. **Union type support** for discriminated unions
6. **Comment-driven documentation** following project conventions

## Design

### Core Types

```go
// Examples represents a collection of named examples for a type
type Examples[T any] map[string]T
```

### OpenAPI 3.1 Example Placement

Examples are supported in specific OpenAPI 3.1 locations:

1. **MediaType Objects** (primary) - Request/Response body examples
2. **Parameter Objects** (secondary) - Query/Path/Header parameter examples  
3. **Schema Objects** (limited) - Simple field-level examples only

### Convention Over Configuration Mapping

- **Body sections** → MediaType examples (main target)
- **Path/Query/Header sections** → Parameter examples
- **Individual fields** → Schema field examples (fallback)

### Usage Patterns

#### 1. Request/Response Body Examples (Primary)

For request/response body sections, providing complete examples:

```go
type CreateUserRequest struct {
    Body struct {
        Username string `gork:"username" validate:"required"`
        Email    string `gork:"email" validate:"email"`
    }
}

func (req CreateUserRequest) Examples() api.Examples[CreateUserRequest] {
    return api.Examples[CreateUserRequest]{
        // Basic user creation
        "basic": CreateUserRequest{
            Body: struct {
                Username string `gork:"username" validate:"required"`
                Email    string `gork:"email" validate:"email"`
            }{
                Username: "john_doe",
                Email:    "john@example.com",
            },
        },
        // Admin user creation
        "admin": CreateUserRequest{
            Body: struct {
                Username string `gork:"username" validate:"required"`
                Email    string `gork:"email" validate:"email"`
            }{
                Username: "admin_user", 
                Email:    "admin@company.com",
            },
        },
    }
}
```

#### 2. Struct Tag Examples (Fallback)

For simple types where method examples are not provided:

```go
type SimpleRequest struct {
    Name  string `gork:"name" validate:"required" example:"John Doe"`
    Email string `gork:"email" validate:"email" example:"john@example.com"`
    Age   int    `gork:"age" validate:"min=0,max=120" example:"30"`
}
```

#### 3. Union Type Examples

Critical for discriminated union types:

```go
func (p PaymentMethodRequest) Examples() api.Examples[PaymentMethodRequest] {
    return api.Examples[PaymentMethodRequest]{
        // Credit card payment
        "credit_card": PaymentMethodRequest{
            Path: struct{ UserID string `gork:"userId"` }{"user123"},
            Body: unions.Union2[BankPaymentMethod, CreditCardPaymentMethod]{
                B: &CreditCardPaymentMethod{
                    Type:       "credit_card",
                    CardNumber: "4111111111111111",
                },
            },
        },
        // Bank transfer payment
        "bank_transfer": PaymentMethodRequest{
            Path: struct{ UserID string `gork:"userId"` }{"user123"},
            Body: unions.Union2[BankPaymentMethod, CreditCardPaymentMethod]{
                A: &BankPaymentMethod{
                    Type:          "bank_account",
                    AccountNumber: "123456789",
                    RoutingNumber: "021000021",
                },
            },
        },
    }
}
```

## Implementation Requirements

### 1. Type Detection

The OpenAPI generator must detect:
- Methods with signature `func (T) Examples() api.Examples[T]`
- Struct fields with `example:"value"` tags
- Handle both cases gracefully with method examples taking precedence

### 2. OpenAPI Output Structure

#### Request Body Examples (MediaType Objects)

Method examples for request bodies map to MediaType examples:

```json
{
  "requestBody": {
    "required": true,
    "content": {
      "application/json": {
        "schema": {"$ref": "#/components/schemas/CreateUserRequestBody"},
        "examples": {
          "basic": {
            "summary": "basic",
            "value": {
              "username": "john_doe",
              "email": "john@example.com"
            }
          },
          "admin": {
            "summary": "admin",
            "value": {
              "username": "admin_user",
              "email": "admin@company.com"  
            }
          }
        }
      }
    }
  }
}
```

#### Response Body Examples

```json
{
  "responses": {
    "200": {
      "description": "Success",
      "content": {
        "application/json": {
          "schema": {"$ref": "#/components/schemas/UserResponse"},
          "examples": {
            "basic": {
              "summary": "basic",
              "value": {
                "userID": "usr_123abc",
                "username": "john_doe"
              }
            }
          }
        }
      }
    }
  }
}
```

#### Parameter Examples

For Path/Query/Header parameters:

```json
{
  "parameters": [
    {
      "name": "userId",
      "in": "path", 
      "schema": {"type": "string"},
      "examples": {
        "basic": {"value": "usr_123abc"},
        "admin": {"value": "usr_admin"}
      }
    }
  ]
}
```

#### Schema Field Examples (Fallback)

Struct tag examples map to schema field examples:

```json
{
  "properties": {
    "name": {
      "type": "string", 
      "example": "John Doe"
    },
    "email": {
      "type": "string",
      "example": "john@example.com"
    }
  }
}
```

### 3. Implementation Integration

Examples must be supported in Convention Over Configuration processing:

- **Body sections** → MediaType objects with examples (primary focus)
- **Path/Query/Header sections** → Parameter objects with examples  
- **Schema field properties** → Schema field examples (fallback only)

### 4. Error Handling

The implementation must gracefully handle:
- **Method panics** during example generation
- **Invalid example data** that doesn't match the type
- **Missing Examples() methods** (fallback to struct tags)
- **Reflection errors** during method detection

### 5. Precedence Rules

1. **Method examples** override struct tag examples when both exist
2. **Multiple method examples** are combined (all included)
3. **Struct tag examples** are used only when no method exists
4. **No examples** is valid (field remains empty)

## Integration Points

### Schema Generation

Modify `generateSchemaFromType()` in `convention_openapi_generator.go`:

```go
func (g *ConventionOpenAPIGenerator) generateSchemaFromType(fieldType reflect.Type, validateTag string, components *Components) *Schema {
    schema := reflectTypeToSchema(fieldType, components.Schemas)
    
    // 1. Check for Examples() method
    if examples := g.extractMethodExamples(fieldType); examples != nil {
        schema.Examples = examples
    }
    
    // 2. Fallback to struct tag examples
    if schema.Examples == nil {
        if example := g.extractStructTagExample(fieldType, validateTag); example != nil {
            schema.Example = example
        }
    }
    
    return schema
}
```

### MediaType Enhancement

Update MediaType to support examples:

```go
type MediaType struct {
    Schema   *Schema                   `json:"schema,omitempty"`
    Examples map[string]*ExampleValue `json:"examples,omitempty"`
}

type ExampleValue struct {
    Summary string      `json:"summary,omitempty"`
    Value   interface{} `json:"value"`
}
```

## Testing Strategy

### Unit Tests

1. **Method detection** - verify Examples() methods are found
2. **Type safety** - ensure Examples[T] type matching
3. **OpenAPI output** - verify correct JSON structure  
4. **Precedence** - method examples override struct tags
5. **Error handling** - graceful fallback on failures

### Integration Tests

1. **End-to-end** OpenAPI generation with examples
2. **Union types** examples in generated spec
3. **Complex nested** structures with examples
4. **Performance** impact on spec generation

### Example Test Cases

```go
// Test basic method examples
func TestMethodExamples(t *testing.T) {
    type TestType struct {
        Field string `gork:"field"`
    }
    
    func (t TestType) Examples() api.Examples[TestType] {
        return api.Examples[TestType]{
            "basic": TestType{Field: "value"},
        }
    }
    
    // Verify Examples() method is detected and processed
}

// Test struct tag fallback
func TestStructTagFallback(t *testing.T) {
    type TestType struct {
        Field string `gork:"field" example:"default_value"`
    }
    
    // Verify struct tag example is used when no Examples() method exists
}

// Test precedence
func TestExamplePrecedence(t *testing.T) {
    type TestType struct {
        Field string `gork:"field" example:"tag_value"`
    }
    
    func (t TestType) Examples() api.Examples[TestType] {
        return api.Examples[TestType]{
            "method": TestType{Field: "method_value"},
        }
    }
    
    // Verify method examples take precedence over struct tags
}
```

## Migration Path

### Phase 1: Core Implementation
- Add `Examples[T]` type to `openapi_types.go`
- Implement method detection in schema generation
- Add struct tag example support

### Phase 2: Integration
- Update MediaType to support examples
- Add Parameter example support  
- Integrate with union type generation

### Phase 3: Enhancement
- Comment parsing for better summaries
- Performance optimizations
- Advanced example validation

## Compatibility

This feature is **backward compatible**:
- Existing code without examples continues to work
- New example features are additive
- No breaking changes to existing APIs
- Optional feature activation via method presence

## Success Metrics

1. **Type Safety** - All examples compile and type-check
2. **OpenAPI Compliance** - Generated specs validate against OpenAPI 3.1
3. **Developer Experience** - Simple, intuitive API for adding examples
4. **Performance** - Minimal impact on spec generation time
5. **Coverage** - Support for all struct types including unions