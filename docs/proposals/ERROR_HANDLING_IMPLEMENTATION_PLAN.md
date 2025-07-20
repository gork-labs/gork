# Error Handling Implementation Plan

## Overview
This document provides a detailed implementation plan for adding automatic validation and error handling to the GoAPI framework using `go-playground/validator`.

## Goals
1. Automatic validation of all request structs using `go-playground/validator`
2. Three distinct error types:
   - **422 Unprocessable Entity**: When request can't be parsed
   - **400 Bad Request**: When validation fails
   - **500 Internal Server Error**: When handler returns error
3. No changes required to existing handlers
4. Accurate OpenAPI documentation

## Implementation Phases

### Phase 1: Core Setup and Dependencies
**Timeline: 1 day**

#### TODO List:
- [ ] Add `go-playground/validator` dependency
  ```bash
  go get github.com/go-playground/validator/v10
  ```

- [ ] Create `pkg/api/errors.go` with error response types
  - [ ] Define `ErrorResponse` struct
  - [ ] Define `ValidationErrorResponse` struct
  - [ ] Add JSON tags and documentation

- [ ] Create `pkg/api/validator.go` for validator setup
  - [ ] Initialize global validator instance
  - [ ] Configure JSON tag name usage
  - [ ] Add custom error formatters
  - [ ] Add discriminator field detection

### Phase 2: Handler Factory Integration
**Timeline: 2-3 days**

#### TODO List:
- [ ] Update `pkg/api/handler_factory.go`
  - [ ] Import validator package
  - [ ] Add validation after request parsing
  - [ ] Distinguish between parse errors (422) and validation errors (400)
  - [ ] Update error handling flow
  - [ ] Add discriminator field pre-processing

- [ ] Implement automatic required field detection
  - [ ] Check for `openapi:"discriminator=..."` tags
  - [ ] Automatically treat discriminator fields as required
  - [ ] Inject validation rules for discriminator fields
  - [ ] Ensure discriminator values match expected constants

- [ ] Update request parsing logic
  - [ ] Separate parsing from validation
  - [ ] Return appropriate error for each failure type
  - [ ] Ensure all parameter sources are validated

- [ ] Add validation tests
  - [ ] Test required field validation
  - [ ] Test format validation (email, uuid, etc.)
  - [ ] Test numeric range validation
  - [ ] Test parse error vs validation error distinction
  - [ ] Test discriminator fields are automatically required
  - [ ] Test discriminator validation without explicit `validate:"required"`

### Phase 3: OpenAPI Generator Updates
**Timeline: 2 days**

#### TODO List:
- [ ] Update `pkg/api/openapi_generator.go`
  - [ ] Always add 422, 400, and 500 responses to operations
  - [ ] Add error schema references
  - [ ] Ensure components include error schemas
  - [ ] Add discriminator fields to required array automatically

- [ ] Update `tools/openapi-gen/internal/generator/generator.go`
  - [ ] Replace generic error responses with proper schemas
  - [ ] Add 422 response for all endpoints
  - [ ] Update error descriptions

- [ ] Parse validation tags for schema enhancement
  - [ ] Extract validation rules from struct tags
  - [ ] Map validation tags to OpenAPI constraints
  - [ ] Add validation info to parameter descriptions
  - [ ] Ensure discriminator fields appear in required array
  - [ ] Add enum constraint for discriminator values

### Phase 4: Examples and Testing
**Timeline: 1-2 days**

#### TODO List:
- [ ] Update example handlers with validation tags
  - [ ] Add validation to `CreateUserRequest`
  - [ ] Add validation to `UpdateUserRequest`
  - [ ] Add validation to other request types

- [ ] Create validation examples
  - [ ] Show common validation patterns
  - [ ] Demonstrate all supported validators
  - [ ] Include edge cases

- [ ] Test with real requests
  - [ ] Test valid requests pass through
  - [ ] Test malformed JSON returns 422
  - [ ] Test invalid data returns 400
  - [ ] Test handler errors return 500

- [ ] Regenerate and validate OpenAPI
  - [ ] Run generator on examples
  - [ ] Verify error responses in spec
  - [ ] Validate with OpenAPI tools

### Phase 5: Documentation and Polish
**Timeline: 1 day**

#### TODO List:
- [ ] Update documentation
  - [ ] Add validation section to README
  - [ ] Document supported validation tags
  - [ ] Provide migration guide
  - [ ] Add troubleshooting section

- [ ] Performance optimization
  - [ ] Cache validator instances if needed
  - [ ] Benchmark validation overhead
  - [ ] Optimize hot paths

- [ ] Add advanced features
  - [ ] Custom validation functions
  - [ ] Validation groups
  - [ ] Conditional validation

## Implementation Details

### errors.go
```go
package api

// ErrorResponse represents a generic error response
type ErrorResponse struct {
    Error   string                 `json:"error"`
    Details map[string]interface{} `json:"details,omitempty"`
}

// ValidationErrorResponse represents validation errors
type ValidationErrorResponse struct {
    Error   string              `json:"error"`
    Details map[string][]string `json:"details,omitempty"`
}
```

### validator.go
```go
package api

import (
    "reflect"
    "strings"
    "github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
    validate = validator.New()
    
    // Use JSON tag names in validation errors
    validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
        if name == "-" {
            return ""
        }
        return name
    })
    
    // Register custom validation for discriminator fields
    validate.RegisterStructValidation(validateDiscriminatorFields, 
        CreditCardPayment{}, 
        BankTransferPayment{},
        // ... other types with discriminators
    )
}

// validateDiscriminatorFields ensures discriminator fields are set correctly
func validateDiscriminatorFields(sl validator.StructLevel) {
    rv := sl.Current()
    rt := rv.Type()
    
    for i := 0; i < rt.NumField(); i++ {
        field := rt.Field(i)
        if discValue, ok := parseDiscriminator(field.Tag.Get("openapi")); ok {
            fieldValue := rv.Field(i)
            
            // Check if field is empty (required validation)
            if fieldValue.Kind() == reflect.String && fieldValue.String() == "" {
                sl.ReportError(fieldValue.Interface(), field.Name, 
                    field.Tag.Get("json"), "required", "")
                continue
            }
            
            // Check if field matches expected discriminator value
            if fieldValue.Kind() == reflect.String && fieldValue.String() != discValue {
                sl.ReportError(fieldValue.Interface(), field.Name,
                    field.Tag.Get("json"), "discriminator", 
                    fmt.Sprintf("must be '%s'", discValue))
            }
        }
    }
}
```

### Updated handler_factory.go snippet
```go
// In createHandlerFromAny
return func(w http.ResponseWriter, r *http.Request) {
    // ... existing setup ...
    
    // Parse request
    if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
        if err := json.NewDecoder(r.Body).Decode(reqPtr.Interface()); err != nil {
            // Parse error = 422
            writeJSONError(w, http.StatusUnprocessableEntity, ErrorResponse{
                Error: "Unable to parse request body",
                Details: map[string]interface{}{
                    "parse_error": err.Error(),
                },
            })
            return
        }
    }
    
    // ... parse other parameters ...
    
    // Pre-process discriminator fields before validation
    preprocessDiscriminatorFields(reqPtr.Interface())
    
    // Validate
    if err := validate.Struct(reqPtr.Interface()); err != nil {
        // Validation error = 400
        validationErrors := make(map[string][]string)
        for _, err := range err.(validator.ValidationErrors) {
            field := err.Field()
            
            // Special handling for discriminator errors
            if err.Tag() == "discriminator" {
                validationErrors[field] = append(validationErrors[field], 
                    fmt.Sprintf("invalid_discriminator: %s", err.Param()))
            } else {
                validationErrors[field] = append(validationErrors[field], err.Tag())
            }
        }
        
        writeJSONError(w, http.StatusBadRequest, ValidationErrorResponse{
            Error:   "Validation failed",
            Details: validationErrors,
        })
        return
    }
    
    // ... call handler and handle response ...
}
```

## Testing Strategy

### Unit Tests
- [ ] Validator configuration tests
- [ ] Error response serialization
- [ ] Parse vs validation error distinction
- [ ] All supported validation tags

### Integration Tests
- [ ] End-to-end request validation
- [ ] Multiple parameter sources
- [ ] Complex nested structs
- [ ] Custom validation rules

### Example Validation
- [ ] All examples compile
- [ ] Validation works as expected
- [ ] OpenAPI spec is accurate

## Success Criteria

1. ✓ Zero changes needed to existing handlers
2. ✓ Automatic validation on all requests
3. ✓ Clear distinction between error types (422/400/500)
4. ✓ Accurate OpenAPI documentation
5. ✓ Minimal performance impact
6. ✓ Easy to add validation rules

## Common Validation Tags Reference

```go
// Required
Field string `validate:"required"`

// String validation
Email string `validate:"required,email"`
URL   string `validate:"required,url"`
UUID  string `validate:"required,uuid"`

// Discriminator fields (automatically required)
Type string `json:"type" openapi:"discriminator=credit_card"`
Kind string `json:"kind" openapi:"discriminator=user"`

// Numeric validation
Age    int `validate:"required,min=18,max=120"`
Score  int `validate:"required,gte=0,lte=100"`

// String length
Name     string `validate:"required,min=2,max=50"`
Password string `validate:"required,min=8"`

// Optional with validation
Phone *string `validate:"omitempty,e164"`

// One of values
Status string `validate:"required,oneof=active inactive pending"`

// Custom validation
Custom string `validate:"required,customValidator"`
```

## Migration Guide

### For Existing Handlers
1. No code changes required
2. Add validation tags to request structs
3. Remove manual validation code
4. Test error responses

### Example Migration
```go
// Before
func CreateUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error) {
    if req.Email == "" {
        return nil, errors.New("email is required")
    }
    if !isValidEmail(req.Email) {
        return nil, errors.New("invalid email format")
    }
    // ... more validation ...
}

// After
type CreateUserRequest struct {
    Email string `json:"email" validate:"required,email"`
    // validation happens automatically!
}

func CreateUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error) {
    // Just business logic, no validation needed
}
```

## Next Steps

1. Review and approve simplified plan
2. Set up development branch
3. Implement Phase 1 (core setup)
4. Test with example handlers
5. Iterate based on feedback 