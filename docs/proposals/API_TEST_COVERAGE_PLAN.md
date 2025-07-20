# API Test Coverage Plan

## Goal
Achieve 100% test coverage for the `pkg/api` package by adding comprehensive tests for all untested code paths.

## Current Status
- Current coverage: 45.6%
- Tested: Basic `parseQueryParams` and `getFunctionName` functionality
- Major gaps: `HandlerFunc`, error handling, edge cases, and security options

## Implementation Plan

### Phase 1: Core Functionality Tests

#### 1.1 HandlerFunc Integration Tests
Create a new test function `TestHandlerFunc` with the following test cases:

**HTTP Method Tests:**
- [ ] Test GET request with query parameters
- [ ] Test POST request with JSON body
- [ ] Test PUT request with JSON body
- [ ] Test PATCH request with JSON body
- [ ] Test DELETE request with query parameters
- [ ] Test unsupported HTTP method (returns 405)

**Handler Execution Tests:**
- [ ] Test successful handler execution with valid response
- [ ] Test handler returning an error
- [ ] Test handler returning nil response and nil error

**Request Parsing Tests:**
- [ ] Test invalid JSON body (malformed JSON)
- [ ] Test valid JSON body parsing
- [ ] Test query parameter parsing for GET/DELETE

**Response Encoding Tests:**
- [ ] Test successful JSON response encoding
- [ ] Test response encoding failure (e.g., with circular reference)

#### 1.2 Option Functions Tests
Create `TestHandlerOptions` with the following:
- [ ] Test `WithTags` option
- [ ] Test `WithBasicAuth` option
- [ ] Test `WithBearerTokenAuth` option
- [ ] Test `WithAPIKeyAuth` option
- [ ] Test multiple options applied together

#### 1.3 Metadata Tests
Create `TestGetHandlerMetadata`:
- [ ] Test retrieving metadata for a registered handler
- [ ] Test retrieving metadata for non-existent handler
- [ ] Test metadata persistence across requests

### Phase 2: Error Handling Tests

#### 2.1 WriteError Function Tests
Create `TestWriteError`:
- [ ] Test 4xx error with custom message
- [ ] Test 5xx error with sanitized message
- [ ] Test JSON encoding of error response
- [ ] Test logger usage for 5xx errors

### Phase 3: Edge Cases and Comprehensive Coverage

#### 3.1 Enhanced parseQueryParams Tests
Add test cases to `TestParseQueryParams`:
- [ ] Fields with empty json tag (`json:""`)
- [ ] Fields with dash json tag (`json:"-"`)
- [ ] Empty parameter values
- [ ] Invalid numeric conversions:
  - [ ] Invalid int value
  - [ ] Invalid uint value
  - [ ] Invalid bool value
  - [ ] Invalid float value
- [ ] Float32 and Float64 field types
- [ ] Non-string slice types (should be ignored)
- [ ] Empty slice when no values provided

#### 3.2 Unsupported Field Types Tests
Create `TestSetFieldValueUnsupportedTypes`:
- [ ] Test all unsupported reflect types:
  - [ ] Invalid, Uintptr, Complex64, Complex128
  - [ ] Array, Chan, Func, Interface
  - [ ] Map, Ptr, Struct, UnsafePointer

#### 3.3 Enhanced getFunctionName Tests
Add test cases to `TestGetFunctionName`:
- [ ] Nil function pointer
- [ ] Function without package path
- [ ] Function name without dots

### Phase 4: Test Utilities and Helpers

#### 4.1 Create Test Helpers
```go
// Mock handler functions
func successHandler(ctx context.Context, req TestRequest) (*TestResponse, error)
func errorHandler(ctx context.Context, req TestRequest) (*TestResponse, error)
func nilResponseHandler(ctx context.Context, req TestRequest) (*TestResponse, error)

// Test request/response types with various field types
type ComplexTestRequest struct {
    IntField    int      `json:"int_field"`
    UintField   uint     `json:"uint_field"`
    FloatField  float64  `json:"float_field"`
    Float32Field float32 `json:"float32_field"`
    BoolField   bool     `json:"bool_field"`
    StringSlice []string `json:"string_slice"`
    IgnoredField string  `json:"-"`
    EmptyTag    string   `json:""`
}

// HTTP test helpers
func createTestRequest(method, url string, body io.Reader) *http.Request
func createTestResponseRecorder() *httptest.ResponseRecorder
```

### Phase 5: Implementation Strategy

1. **Start with Phase 1** - Core functionality is most critical
2. **Use table-driven tests** - Group related test cases for maintainability
3. **Mock dependencies** - Use httptest package for HTTP testing
4. **Test both success and failure paths** - Ensure all error cases are covered
5. **Use coverage tools** - Run `go test -coverprofile=coverage.out` frequently

### Verification Process

After implementing each phase:
1. Run `make test-api` to ensure tests pass
2. Generate coverage report: `go test ./pkg/api -coverprofile=coverage.out`
3. View detailed coverage: `go tool cover -html=coverage.out`
4. Identify remaining uncovered lines
5. Add specific test cases for any missed code paths

### Expected Outcome

Upon completion of all phases:
- 100% test coverage for pkg/api package
- Comprehensive test suite covering all edge cases
- Improved confidence in API adapter reliability
- Documentation through well-structured test cases

### Time Estimate

- Phase 1: 2-3 hours (core functionality is complex)
- Phase 2: 1 hour (error handling)
- Phase 3: 2 hours (edge cases require careful testing)
- Phase 4: 1 hour (test utilities)
- Total: 6-7 hours of focused development