# No-Content Response Implementation Plan

## Overview

This document outlines the implementation plan for supporting handler responses that should result in HTTP 204 No Content. This addresses two specific scenarios:
1. Handlers that return only an `error` (no response struct)
2. Handlers that return a response struct without a `Body` field

## Current Problem

The current implementation requires all response types to follow Convention Over Configuration with `Body`, `Headers`, and `Cookies` sections. This causes panics when:
- A handler returns `(*struct{}, error)` where the struct is empty
- A handler returns `(error)` only
- A handler returns a struct without the required convention fields

### Example Problematic Handler
```go
func DeleteUser(_ context.Context, req DeleteUserRequest) (*struct{}, error) {
    // Handle user deletion logic here
    return nil, nil
}
```

## Proposed Solution

### 1. Handler Signature Detection

Support the following handler patterns:
- `func(ctx context.Context, req RequestType) error` - Error-only return
- `func(ctx context.Context, req RequestType) (*ResponseType, error)` - Current pattern

### 2. Response Type Analysis

**Important**: All response types (when defined) MUST follow Convention Over Configuration with `Body`, `Headers`, `Cookies` sections.

For response handling, detect the following scenarios:
- **Error-only handler**: `func(...) error` → HTTP 204 No Content
- **Nil response value**: When handler returns `(nil, nil)` → HTTP 204 No Content
- **Non-nil conventional response**: Process all sections (`Body`, `Headers`, `Cookies`) normally

**No exceptions**: Response structs without conventional sections are not supported and should cause compilation/validation errors.

### 3. Clarification of HTTP 204 Scenarios

HTTP 204 No Content responses are generated in exactly these cases:
1. **Error-only handlers**: `func(ctx, req) error` → Always 204 on success
2. **Nil response values**: `func(ctx, req) (*ConventionalResponse, error)` returning `(nil, nil)` → 204

All other scenarios process the conventional response normally:
- **Absent Body with Headers/Cookies**: Response sent with headers/cookies, no body content
- **Empty Body with Headers/Cookies**: Response sent with headers/cookies and empty body (`{}`)
- **Non-nil conventional response**: All sections (Body, Headers, Cookies) processed normally

## Implementation Plan

### Phase 1: Handler Signature Validation Updates

**File**: `pkg/api/handler_factory.go` and related validation functions

1. **Update `validateHandlerSignature` function**:
   - Allow handlers with single `error` return
   - Update signature validation logic
   - Add tests for new signature patterns

2. **Update `buildRouteInfo` function**:
   - Handle cases where `ResponseType` is nil (error-only handlers)
   - Ensure proper routing registration

### Phase 2: Runtime Handler Execution Updates

**File**: `pkg/api/convention_handler_factory.go`

1. **Update `executeConventionHandler` function**:
   - Handle error-only handler calls
   - Detect single vs dual return values

2. **Update `processConventionResponse` function**:
   - Handle nil response values (return 204)
   - Add logic for empty response detection

3. **Update `processResponseSections` function**:
   - Handle nil response values (return 204)
   - Continue to require conventional sections for non-nil responses
   - Process Headers and Cookies sections even when Body is absent

### Phase 3: OpenAPI Generation Updates

**File**: `pkg/api/convention_openapi_generator.go`

1. **Update `buildConventionOperation` function**:
   - Skip response processing for error-only handlers
   - Handle nil response types

2. **Update `processResponseSections` function**:
   - Handle nil response types (generate 204 No Content response specification)
   - Keep panic for non-conventional response types (maintain strict convention)
   - Ensure Headers and Cookies are processed even when Body is absent

3. **Add helper functions**:
   ```go
   func (g *ConventionOpenAPIGenerator) generateNoContentResponse() *Response
   ```

### Phase 4: Route Registry Updates

**File**: `pkg/api/registry.go`

1. **Update `RouteInfo` struct**:
   - Make `ResponseType` optional (can be nil)
   - Add field to track response content expectation

2. **Update route registration logic**:
   - Handle nil response types
   - Maintain backward compatibility

### Phase 5: Test Updates

1. **Add test cases for**:
   - Error-only handlers
   - Nil response values (handler returns `(nil, nil)`)
   - Conventional responses with absent Body but Headers/Cookies
   - Conventional responses with empty Body (`{}`) and Headers/Cookies
   - Mixed scenarios in route registry

2. **Update existing tests**:
   - Fix broken tests that expect panics
   - Update OpenAPI generation tests
   - Add integration tests

## Detailed Code Changes

### 1. Handler Signature Validation

```go
// In pkg/api/handler_factory.go
func validateHandlerSignature(t reflect.Type) {
    if t.Kind() != reflect.Func {
        panic("handler must be a function")
    }
    
    if t.NumIn() != 2 {
        panic("handler must have exactly 2 parameters: (context.Context, RequestType)")
    }
    
    // Validate first parameter is context.Context
    if !t.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
        panic("first parameter must be context.Context")
    }
    
    // Allow either (ResponseType, error) or (error) returns
    numOut := t.NumOut()
    if numOut != 1 && numOut != 2 {
        panic("handler must return either (error) or (*ResponseType, error)")
    }
    
    // Last return must be error
    lastOut := t.Out(numOut - 1)
    if !lastOut.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
        panic("last return value must be error")
    }
    
    // If two returns, first must be pointer to struct
    if numOut == 2 {
        firstOut := t.Out(0)
        if firstOut.Kind() != reflect.Ptr || firstOut.Elem().Kind() != reflect.Struct {
            panic("response type must be pointer to struct")
        }
    }
}
```

### 2. OpenAPI Generator Updates

```go
// In pkg/api/convention_openapi_generator.go
func (g *ConventionOpenAPIGenerator) processResponseSections(respType reflect.Type, operation *Operation, components *Components, route *RouteInfo) {
    // Handle nil response type (error-only handlers)
    if respType == nil {
        operation.Responses["204"] = g.generateNoContentResponse()
        return
    }
    
    if respType.Kind() == reflect.Ptr {
        respType = respType.Elem()
    }
    
    // Handle non-struct responses
    if respType.Kind() != reflect.Struct {
        operation.Responses["204"] = g.generateNoContentResponse()
        return
    }
    
    // All non-nil response types MUST follow Convention Over Configuration
    if !g.usesConventionSections(respType) {
        panic(fmt.Sprintf("response type must use Convention Over Configuration sections (Body, Headers, Cookies)\nHandler: %s %s -> %s\nResponse type: %s\nFound fields: %s", 
            route.Method, route.Path, route.HandlerName, respType.Name(), g.getFieldNames(respType)))
    }
    
    // Original convention processing logic...
}

func (g *ConventionOpenAPIGenerator) generateNoContentResponse() *Response {
    return &Response{
        Description: "No Content",
    }
}
```

### 3. Runtime Handler Updates

```go
// In pkg/api/convention_handler_factory.go
func (f *ConventionHandlerFactory) processConventionResponse(w http.ResponseWriter, r *http.Request, handlerValue reflect.Value, reqPtr reflect.Value) {
    // Call the handler via reflection
    results := handlerValue.Call([]reflect.Value{
        reflect.ValueOf(r.Context()),
        reqPtr.Elem(),
    })
    
    // Handle different return patterns
    if len(results) == 1 {
        // Error-only handler
        errInterface := results[0].Interface()
        if errInterface != nil {
            if errVal, ok := errInterface.(error); ok {
                writeError(w, http.StatusInternalServerError, errVal.Error())
                return
            }
        }
        // Success with no content
        w.WriteHeader(http.StatusNoContent)
        return
    }
    
    // Dual return handler (response, error)
    respVal := results[0]
    errInterface := results[1].Interface()
    
    if errInterface != nil {
        if errVal, ok := errInterface.(error); ok {
            writeError(w, http.StatusInternalServerError, errVal.Error())
            return
        }
    }
    
    // Process response sections if the response follows Convention Over Configuration
    f.processResponseSections(w, respVal)
}
```

## Migration Strategy

### For Existing Code
1. **Backward Compatibility**: All existing handlers continue to work unchanged
2. **Gradual Migration**: New handlers can use simplified patterns
3. **Documentation**: Update examples and guides

### For New Features
1. **Error-only handlers**: Suitable for DELETE, some PUT operations
2. **Empty response handlers**: For operations that don't return data
3. **Mixed patterns**: Can coexist in same application

## Testing Strategy

### Unit Tests
- Handler signature validation with new patterns
- OpenAPI generation for no-content responses
- Runtime execution of simplified handlers

### Integration Tests
- Full request/response cycle with no-content handlers
- OpenAPI spec generation with mixed handler types
- Route registration with different response patterns

### Example Test Cases
```go
func TestErrorOnlyHandler(t *testing.T) {
    handler := func(ctx context.Context, req TestRequest) error {
        return nil
    }
    
    // Test handler registration and execution - should generate 204
}

func TestNilConventionalResponse(t *testing.T) {
    handler := func(ctx context.Context, req TestRequest) (*TestConventionalResponse, error) {
        return nil, nil // This should generate 204
    }
    
    // Test 204 response generation for nil response
}

func TestConventionalResponseWithHeadersOnly(t *testing.T) {
    type ResponseWithHeaders struct {
        Headers struct {
            CustomHeader string `gork:"X-Custom-Header"`
        }
        // Note: Body is absent, not empty
    }
    
    handler := func(ctx context.Context, req TestRequest) (*ResponseWithHeaders, error) {
        return &ResponseWithHeaders{
            Headers: struct {
                CustomHeader string `gork:"X-Custom-Header"`
            }{CustomHeader: "test-value"},
        }, nil
    }
    
    // Test that headers are sent even when Body is absent
}

func TestConventionalResponseWithEmptyBody(t *testing.T) {
    type ResponseWithEmptyBody struct {
        Body struct{} // Empty body - will send {}
        Headers struct {
            CustomHeader string `gork:"X-Custom-Header"`
        }
    }
    
    handler := func(ctx context.Context, req TestRequest) (*ResponseWithEmptyBody, error) {
        return &ResponseWithEmptyBody{
            Headers: struct {
                CustomHeader string `gork:"X-Custom-Header"`
            }{CustomHeader: "test-value"},
        }, nil
    }
    
    // Test that headers are sent with empty body content ({})
}
```

## Documentation Updates

1. **Convention Over Configuration Spec**: Update to reflect new response patterns
2. **Handler Examples**: Add examples of simplified handlers
3. **OpenAPI Generation**: Document no-content response generation
4. **Migration Guide**: Help developers transition existing code

## Implementation Timeline

1. **Week 1**: Phase 1 - Handler signature validation
2. **Week 2**: Phase 2 - Runtime execution updates  
3. **Week 3**: Phase 3 - OpenAPI generation updates
4. **Week 4**: Phase 4 - Route registry updates
5. **Week 5**: Phase 5 - Test updates and documentation

## Success Criteria

1. ✅ Error-only handlers work at runtime and generate HTTP 204
2. ✅ Nil response values generate HTTP 204
3. ✅ Non-conventional response structs still cause compilation errors (strict validation maintained)
4. ✅ Conventional responses with Headers/Cookies but absent Body work correctly
5. ✅ OpenAPI specs correctly show 204 responses for error-only handlers
6. ✅ All existing functionality remains intact
7. ✅ Comprehensive test coverage
8. ✅ Updated documentation and examples

## Risk Mitigation

1. **Breaking Changes**: Extensive testing to ensure backward compatibility
2. **Performance**: Minimal impact due to reflection optimizations
3. **Edge Cases**: Comprehensive test suite covering all scenarios
4. **Documentation**: Clear migration path and examples

This implementation will provide a more flexible and developer-friendly API while maintaining the existing Convention Over Configuration benefits.