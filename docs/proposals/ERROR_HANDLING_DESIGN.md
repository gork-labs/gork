# Error Handling Design for GoAPI Framework

## Overview

This document proposes a simplified error handling design for the GoAPI framework that automatically handles validation errors and server errors using `go-playground/validator` in the handler factory.

## Current State

### What Works
- `NoContentResponse` successfully handles 204 No Content responses
- Basic error handling exists via `writeError` function
- OpenAPI generator already adds generic 400 and 500 error responses

### Gaps
1. No automatic validation of request structs
2. No distinction between parsing errors and validation errors
3. Generic error responses in OpenAPI

## Proposed Solution

### Core Principles
1. All validation happens automatically in `createHandlerFromAny`
2. Use `go-playground/validator` for struct validation
3. Three types of errors:
   - **422 Unprocessable Entity**: When request body/params can't be parsed
   - **400 Bad Request**: When parsed struct fails validation
   - **500 Internal Server Error**: When handler returns an error

### 1. Error Response Types

```go
// Package api

// ErrorResponse represents a generic error response
type ErrorResponse struct {
    Error   string                 `json:"error"`
    Details map[string]interface{} `json:"details,omitempty"`
}

// ValidationErrorResponse represents a 400 Bad Request due to validation failure
type ValidationErrorResponse struct {
    Error   string                   `json:"error"`
    Details map[string][]string      `json:"details,omitempty"` // field -> error messages
}
```

### 2. Integration with Handler Factory

Update `handler_factory.go` to automatically validate requests:

```go
import (
    "github.com/go-playground/validator/v10"
)

var validate = validator.New()

func createHandlerFromAny(adapter ParameterAdapter, handler interface{}, opts ...Option) (http.HandlerFunc, *RouteInfo) {
    // ... existing code ...
    
    return func(w http.ResponseWriter, r *http.Request) {
        // ... existing setup ...
        
        // Parse request body/params
        if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
            if err := json.NewDecoder(r.Body).Decode(reqPtr.Interface()); err != nil {
                // Can't parse = 422 Unprocessable Entity
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusUnprocessableEntity)
                json.NewEncoder(w).Encode(ErrorResponse{
                    Error: "Unable to parse request body",
                    Details: map[string]interface{}{
                        "parse_error": err.Error(),
                    },
                })
                return
            }
        }
        
        // ... parse query/path/header params ...
        
        // Validate the complete request struct
        if err := validate.Struct(reqPtr.Interface()); err != nil {
            // Validation failed = 400 Bad Request
            validationErrors := make(map[string][]string)
            for _, err := range err.(validator.ValidationErrors) {
                field := err.Field()
                if validationErrors[field] == nil {
                    validationErrors[field] = []string{}
                }
                validationErrors[field] = append(validationErrors[field], err.Tag())
            }
            
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(ValidationErrorResponse{
                Error:   "Validation failed",
                Details: validationErrors,
            })
            return
        }
        
        // Call handler
        results := handlerValue.Call([]reflect.Value{
            reflect.ValueOf(ctx),
            reqPtr,
        })
        
        responseVal := results[0]
        errVal := results[1]
        
        if !errVal.IsNil() {
            // Handler error = 500 Internal Server Error
            writeError(w, http.StatusInternalServerError, errVal.Interface().(error).Error())
            return
        }
        
        // ... handle successful response ...
    }
}
```

### 3. Request Struct Validation Tags

Handlers can use standard `validate` tags:

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"omitempty,min=18,max=120"`
}

type UpdateUserRequest struct {
    UserID string `param:"id" validate:"required,uuid"`
    Email  string `json:"email" validate:"omitempty,email"`
}
```

#### Automatic Required Fields

Some OpenAPI tags automatically make fields required:

1. **Discriminator fields** - Any field with `openapi:"discriminator=value"` is automatically required

```go
// Union type examples
type CreditCardPayment struct {
    Type   string `json:"type" openapi:"discriminator=cc"`  // Automatically required!
    Number string `json:"number" validate:"required"`
}

type BankTransferPayment struct {
    Type      string `json:"type" openapi:"discriminator=bank"`  // Automatically required!
    AccountNo string `json:"account" validate:"required"`
}
```

The framework will:
- Automatically add `required` validation to discriminator fields
- Include them in the OpenAPI schema's required array
- Validate them even without explicit `validate:"required"` tag

### 4. OpenAPI Generation Updates

The OpenAPI generator should:
1. Always include 422, 400, and 500 responses
2. Parse validation tags to enhance parameter/schema documentation
3. Continue to add discriminator fields to required array (already implemented)

```go
// In openapi_generator.go
func generateOperation(handler interface{}, info *RouteInfo) *Operation {
    // ... existing code ...
    
    // Always add standard error responses
    operation.Responses["422"] = &Response{
        Description: "Unprocessable Entity - Request body could not be parsed",
        Content: map[string]*MediaType{
            "application/json": {
                Schema: &Schema{
                    Ref: "#/components/schemas/ErrorResponse",
                },
            },
        },
    }
    
    operation.Responses["400"] = &Response{
        Description: "Bad Request - Validation failed",
        Content: map[string]*MediaType{
            "application/json": {
                Schema: &Schema{
                    Ref: "#/components/schemas/ValidationErrorResponse",
                },
            },
        },
    }
    
    operation.Responses["500"] = &Response{
        Description: "Internal Server Error",
        Content: map[string]*MediaType{
            "application/json": {
                Schema: &Schema{
                    Ref: "#/components/schemas/ErrorResponse",
                },
            },
        },
    }
    
    return operation
}

func generateSchemaFromType(t reflect.Type) *Schema {
    // ... existing code ...
    
    // Check for discriminator fields
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if _, ok := parseDiscriminator(field.Tag.Get("openapi")); ok {
            // Discriminator fields are always required
            if !contains(schema.Required, jsonFieldName) {
                schema.Required = append(schema.Required, jsonFieldName)
            }
        }
    }
    
    return schema
}
```

## Implementation Details

### Custom Validation Messages

```go
// Configure validator with custom messages
func init() {
    validate = validator.New()
    
    // Use JSON tag names in errors instead of Go field names
    validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
        if name == "-" {
            return ""
        }
        return name
    })
}
```

### Enhanced Error Messages

```go
// Helper to create user-friendly validation messages
func formatValidationError(err validator.FieldError) string {
    switch err.Tag() {
    case "required":
        return fmt.Sprintf("%s is required", err.Field())
    case "email":
        return fmt.Sprintf("%s must be a valid email address", err.Field())
    case "min":
        return fmt.Sprintf("%s must be at least %s", err.Field(), err.Param())
    case "max":
        return fmt.Sprintf("%s must be at most %s", err.Field(), err.Param())
    default:
        return fmt.Sprintf("%s failed %s validation", err.Field(), err.Tag())
    }
}
```

### Validation Integration

Update the validator setup to handle implicit requirements:

```go
// In validator.go
func validateStruct(v interface{}) error {
    // First, ensure discriminator fields are marked as required
    ensureDiscriminatorValidation(v)
    
    // Then run standard validation
    return validate.Struct(v)
}

func ensureDiscriminatorValidation(v interface{}) {
    rv := reflect.ValueOf(v)
    if rv.Kind() == reflect.Ptr {
        rv = rv.Elem()
    }
    
    if rv.Kind() != reflect.Struct {
        return
    }
    
    rt := rv.Type()
    for i := 0; i < rt.NumField(); i++ {
        field := rt.Field(i)
        
        // Check if field has discriminator tag
        if _, ok := parseDiscriminator(field.Tag.Get("openapi")); ok {
            // Ensure the field value is not empty
            fv := rv.Field(i)
            if fv.Kind() == reflect.String && fv.String() == "" {
                // This will be caught by the validator
                // We could also inject a custom validation here
            }
        }
    }
}
```

## Benefits

1. **Automatic Validation**: No need to manually validate in handlers
2. **Consistency**: All APIs use the same validation and error format
3. **Simplicity**: Handlers just focus on business logic
4. **Standards**: Uses widely-adopted `go-playground/validator`
5. **Clear Errors**: Distinct status codes for different error types

## Example Usage

```go
// Handler just focuses on business logic
func CreateUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error) {
    // Validation already done by framework!
    
    // Business logic
    user, err := db.CreateUser(req)
    if err != nil {
        // This will automatically become a 500 error
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    return &UserResponse{
        ID:    user.ID,
        Email: user.Email,
    }, nil
}

// Request with validation tags
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8,max=50"`
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Age      *int   `json:"age" validate:"omitempty,min=13,max=120"`
}
```

## Error Response Examples

### 422 - Can't Parse Request
```json
{
    "error": "Unable to parse request body",
    "details": {
        "parse_error": "invalid character 'x' looking for beginning of value"
    }
}
```

### 400 - Validation Failed
```json
{
    "error": "Validation failed",
    "details": {
        "email": ["required"],
        "password": ["min"],
        "age": ["max"]
    }
}
```

### 500 - Internal Error
```json
{
    "error": "Internal server error"
}
```

## Future Enhancements

1. Custom validation functions
2. Localized error messages
3. Field-level error details with actual vs expected values
4. Request ID tracking for error correlation

## Conclusion

This simplified design provides automatic validation and error handling without requiring handlers to implement special interfaces or return typed errors. It maintains clean separation of concerns while providing consistent, well-documented error responses. 