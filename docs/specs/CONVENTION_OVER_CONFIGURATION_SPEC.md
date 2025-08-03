# Convention Over Configuration Request/Response Structure Specification

## Overview

This specification defines a convention-based approach for structuring HTTP request and response types in Gork, replacing the current tag-based parameter mapping system with explicit embedded structs.

## Goals

1. **Explicit Structure**: Make it immediately clear where each parameter comes from
2. **Self-Documenting**: Request structs serve as API documentation
3. **Consistent**: Enforce standard naming across all Gork codebases
4. **Maintainable**: Simplify parsing logic and OpenAPI generation
5. **Validation-Friendly**: Enable namespaced validation error reporting

## Request Structure Convention

### Standard Sections

All request types MUST use these exact section names when present:

- `Query` - Query string parameters
- `Body` - Request body (JSON, form data, etc.)
- `Path` - Path parameters from URL routes
- `Headers` - HTTP headers
- `Cookies` - HTTP cookies

### Basic Structure

```go
type RequestName struct {
    Query   QueryStruct   // Optional
    Body    BodyStruct    // Optional  
    Path    PathStruct    // Optional
    Headers HeaderStruct  // Optional
    Cookies CookieStruct  // Optional
}
```

### Field Requirements

1. **Section names are case-sensitive** and must match exactly: `Query`, `Body`, `Path`, `Headers`, `Cookies`
2. **Sections are optional** - omit sections that are not needed
3. **Each section contains a struct** with the relevant fields
4. **Use `gork:"name"` tags** for field naming (not `json:"name"`)
5. **Use `validate:"..."` tags** for validation rules

## Examples

### Simple GET Request
```go
type GetUserRequest struct {
    Path struct {
        UserID string `gork:"user_id" validate:"required,uuid"`
    }
    Query struct {
        IncludeProfile bool `gork:"include_profile"`
        Fields         []string `gork:"fields"`
    }
}
```

### POST Request with Body
```go
type CreateUserRequest struct {
    Body struct {
        Name     string `gork:"name" validate:"required,min=1,max=100"`
        Email    string `gork:"email" validate:"required,email"`
        Age      int    `gork:"age" validate:"min=18,max=120"`
        Metadata map[string]interface{} `gork:"metadata"`
    }
    Headers struct {
        Authorization string `gork:"Authorization" validate:"required"`
        ContentType   string `gork:"Content-Type"`
    }
}
```

### Complex Request with All Sections
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
            Bio     string   `gork:"bio" validate:"max=500"`
            Tags    []string `gork:"tags" validate:"dive,min=1,max=20"`
        } `gork:"profile"`
    }
    Headers struct {
        Authorization string `gork:"Authorization" validate:"required"`
        IfMatch       string `gork:"If-Match"`
    }
    Cookies struct {
        SessionID string `gork:"session_id"`
        Preferences string `gork:"preferences"`
    }
}
```

### File Upload Request
```go
type UploadFileRequest struct {
    Path struct {
        ProjectID string `gork:"project_id" validate:"required,uuid"`
    }
    Body struct {
        File        multipart.File        `gork:"file" validate:"required"`
        Description string                `gork:"description" validate:"max=1000"`
        Tags        []string              `gork:"tags" validate:"dive,min=1,max=20"`
        Metadata    map[string]string     `gork:"metadata"`
    }
    Headers struct {
        ContentType string `gork:"Content-Type" validate:"required"`
    }
}
```

### Union Types in Requests

Union types work seamlessly within any section using the `unions.UnionN` pattern:

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

type OAuthLogin struct {
    Type        string `gork:"type,discriminator=oauth" validate:"required"`
    Provider    string `gork:"provider" validate:"required,oneof=google facebook github"`
    AccessToken string `gork:"access_token" validate:"required"`
}

type LoginRequest struct {
    Body struct {
        // Union type with discriminator
        LoginMethod unions.Union3[EmailLogin, PhoneLogin, OAuthLogin] `gork:"login_method" validate:"required"`
        RememberMe  bool `gork:"remember_me"`
    }
}
```

### Union Types in Query Parameters

```go
type BasicFilter struct {
    Type  string `gork:"type,discriminator=basic" validate:"required"`
    Field string `gork:"field" validate:"required"`
    Value string `gork:"value" validate:"required"`
}

type RangeFilter struct {
    Type  string `gork:"type,discriminator=range" validate:"required"`
    Field string `gork:"field" validate:"required"`
    Min   *int   `gork:"min"`
    Max   *int   `gork:"max"`
}

type SearchUsersRequest struct {
    Query struct {
        // Union types work in query parameters too
        Filter unions.Union2[BasicFilter, RangeFilter] `gork:"filter"`
        Limit  int `gork:"limit" validate:"min=1,max=100"`
        Offset int `gork:"offset" validate:"min=0"`
    }
}
```

### Nested Union Types

```go
type CreditCardPayment struct {
    Type   string `gork:"type,discriminator=credit_card" validate:"required"`
    Number string `gork:"number" validate:"required,creditcard"`
    CVV    string `gork:"cvv" validate:"required,len=3"`
}

type BankTransfer struct {
    Type          string `gork:"type,discriminator=bank_transfer" validate:"required"`
    AccountNumber string `gork:"account_number" validate:"required"`
    RoutingNumber string `gork:"routing_number" validate:"required"`
}

type PayPalPayment struct {
    Type  string `gork:"type,discriminator=paypal" validate:"required"`
    Email string `gork:"email" validate:"required,email"`
}

type ShippingAddress struct {
    Type    string `gork:"type,discriminator=shipping" validate:"required"`
    Street  string `gork:"street" validate:"required"`
    City    string `gork:"city" validate:"required"`
    Country string `gork:"country" validate:"required,iso3166_1_alpha2"`
}

type BillingAddress struct {
    Type    string `gork:"type,discriminator=billing" validate:"required"`
    Street  string `gork:"street" validate:"required"`
    City    string `gork:"city" validate:"required"`
    Country string `gork:"country" validate:"required,iso3166_1_alpha2"`
}

type CreateOrderRequest struct {
    Body struct {
        // Union for payment method
        Payment unions.Union3[CreditCardPayment, BankTransfer, PayPalPayment] `gork:"payment" validate:"required"`
        
        // Union for address type
        Address unions.Union2[ShippingAddress, BillingAddress] `gork:"address" validate:"required"`
        
        Items []struct {
            ProductID string `gork:"product_id" validate:"required,uuid"`
            Quantity  int    `gork:"quantity" validate:"required,min=1"`
        } `gork:"items" validate:"required,dive"`
        
        Total int64 `gork:"total" validate:"required,min=1"`
    }
}
```

## Response Structure Convention

### Standard Response Format
```go
type ResponseName struct {
    Body    BodyStruct    // Optional - response body
    Headers HeaderStruct  // Optional - response headers
    Cookies CookieStruct  // Optional - response cookies
}
```

### Response Examples

#### Simple JSON Response
```go
type GetUserResponse struct {
    Body struct {
        ID       string    `gork:"id"`
        Name     string    `gork:"name"`
        Email    string    `gork:"email"`
        Created  time.Time `gork:"created"`
        Profile  struct {
            Bio  string   `gork:"bio"`
            Tags []string `gork:"tags"`
        } `gork:"profile"`
    }
}
```

#### Response with Headers and Cookies
```go
type LoginResponse struct {
    Body struct {
        Token     string    `gork:"token"`
        ExpiresAt time.Time `gork:"expires_at"`
        User      struct {
            ID   string `gork:"id"`
            Name string `gork:"name"`
        } `gork:"user"`
    }
    Headers struct {
        Location      string `gork:"Location"`
        CacheControl  string `gork:"Cache-Control"`
    }
    Cookies struct {
        SessionToken string `gork:"session_token"`
        Preferences  string `gork:"preferences"`
    }
}
```

#### File Download Response
```go
type DownloadFileResponse struct {
    Body []byte // Raw file content
    Headers struct {
        ContentType        string `gork:"Content-Type"`
        ContentDisposition string `gork:"Content-Disposition"`
        ContentLength      int64  `gork:"Content-Length"`
        ETag               string `gork:"ETag"`
    }
}
```

#### Union Types in Responses

```go
type SuccessResult struct {
    Type    string      `gork:"type,discriminator=success" validate:"required"`
    Message string      `gork:"message"`
    Data    interface{} `gork:"data"`
}

type ErrorResult struct {
    Type    string `gork:"type,discriminator=error" validate:"required"`
    Code    int    `gork:"code"`
    Message string `gork:"message"`
    Details string `gork:"details"`
}

type WarningResult struct {
    Type    string      `gork:"type,discriminator=warning" validate:"required"`
    Warning string      `gork:"warning"`
    Data    interface{} `gork:"data"`
}

type ProcessDataResponse struct {
    Body struct {
        // Union type for different result types
        Result unions.Union3[SuccessResult, ErrorResult, WarningResult] `gork:"result"`
        
        Timestamp time.Time `gork:"timestamp"`
        RequestID string    `gork:"request_id"`
    }
}
```

#### Complex Union Response Example

```go
type UserProfile struct {
    Type     string   `gork:"type,discriminator=user" validate:"required"`
    Bio      string   `gork:"bio"`
    Website  string   `gork:"website"`
    Location string   `gork:"location"`
    Tags     []string `gork:"tags"`
}

type CompanyProfile struct {
    Type        string `gork:"type,discriminator=company" validate:"required"`
    Industry    string `gork:"industry"`
    Size        string `gork:"size"`
    Website     string `gork:"website"`
    Description string `gork:"description"`
}

type GetProfileResponse struct {
    Body struct {
        ID       string `gork:"id"`
        Name     string `gork:"name"`
        Email    string `gork:"email"`
        
        // Union type for different profile types
        Profile unions.Union2[UserProfile, CompanyProfile] `gork:"profile"`
        
        Settings struct {
            Theme       string `gork:"theme"`
            Notifications bool `gork:"notifications"`
        } `gork:"settings"`
    }
}
```

## Parsing Logic Specification

### Request Parsing Algorithm

1. **Reflect on the request struct** to identify sections
2. **For each standard section present**:
   - `Query`: Parse query string parameters using `gork` tags
   - `Body`: Parse request body (JSON, form data, multipart) using `gork` tags
   - `Path`: Extract path parameters from URL route using `gork` tags  
   - `Headers`: Extract HTTP headers using `gork` tags
   - `Cookies`: Extract HTTP cookies using `gork` tags
3. **Validate each section** using `validate` tags
4. **Perform custom validation**:
   - Check if each section implements `Validator` interface
   - Call `Validate()` method if implemented
   - Check if entire request struct implements `Validator` interface  
   - Call `Validate()` method on request struct if implemented
   - Collect all validation errors
5. **Handle validation results**:
   - Custom validation error types (`*ValidationError`) → HTTP 400 Bad Request
   - Any other error from `Validate()` methods → HTTP 500 Internal Server Error
   - Field-level validation errors → HTTP 400 Bad Request with section namespacing

### Validation Error Format

```go
type ValidationErrorResponse struct {
    Error   string                    `json:"error"`
    Details map[string][]string      `json:"details"`
}

// Example error response:
{
    "error": "Validation failed",
    "details": {
        "query.force": ["required"],
        "body.name": ["required", "min"],
        "body.email": ["email"],
        "body.login_method": ["discriminator"], // Union validation error
        "body.payment.number": ["creditcard"],  // Nested union validation
        "path.user_id": ["uuid"],
        "headers.authorization": ["required"]
    }
}
```

### Union Type Validation

Union types integrate with the existing validation system and support discriminator validation:

```go
type EmailLogin struct {
    Type     string `gork:"type,discriminator=email" validate:"required"`     // Discriminator field
    Email    string `gork:"email" validate:"required,email"`
    Password string `gork:"password" validate:"required"`
}

type PhoneLogin struct {
    Type  string `gork:"type,discriminator=phone" validate:"required"`      // Discriminator field
    Phone string `gork:"phone" validate:"required,e164"`
    Code  string `gork:"code" validate:"required,len=6"`
}

type LoginRequest struct {
    Body struct {
        // Union with discriminator validation
        LoginMethod unions.Union2[EmailLogin, PhoneLogin] `gork:"login_method" validate:"required"`
    }
}
```

**Union Validation Rules**:
- Each union variant MUST have a discriminator field (usually `type` or `kind`)
- Discriminator values are specified in `gork` tags: `gork:"type,discriminator=email"`
- Discriminator values MUST be unique across union variants
- Validation errors include the full path: `body.login_method.email` for nested validation
- Union discriminator errors show as: `body.login_method: ["discriminator"]`

### Section-Level Validation

Section-level validation allows for cross-field validation within sections or across the entire request after basic parsing and field validation is complete. Sections or request structs can implement the `Validator` interface to define custom validation logic.

#### Validator Interface

```go
type Validator interface {
    Validate() error
}
```

#### Section Validation Examples

```go
type CreateUserBody struct {
    Name            string    `gork:"name" validate:"required,min=1,max=100"`
    Email           string    `gork:"email" validate:"required,email"`
    Password        string    `gork:"password" validate:"required,min=8"`
    ConfirmPassword string    `gork:"confirm_password" validate:"required"`
    BirthDate       time.Time `gork:"birth_date" validate:"required"`
    Terms           bool      `gork:"terms" validate:"required"`
}

// Validate implements cross-field validation
func (b *CreateUserBody) Validate() error {
    var errors []string
    
    // Password confirmation validation
    if b.Password != b.ConfirmPassword {
        errors = append(errors, "passwords do not match")
    }
    
    // Age validation (must be 18+)
    age := time.Now().Year() - b.BirthDate.Year()
    if age < 18 {
        errors = append(errors, "user must be at least 18 years old")
    }
    
    // Terms acceptance validation
    if !b.Terms {
        errors = append(errors, "terms and conditions must be accepted")
    }
    
    if len(errors) > 0 {
        return &BodyValidationError{
            Errors: errors,
        }
    }
    
    return nil
}

type CreateUserRequest struct {
    Body CreateUserBody
}

// Example with database validation that can fail with HTTP 500
func (b *CreateUserBody) Validate() error {
    var errors []string
    
    // Client validation errors (HTTP 400)
    if b.Password != b.ConfirmPassword {
        errors = append(errors, "passwords do not match")
    }
    
    age := time.Now().Year() - b.BirthDate.Year()
    if age < 18 {
        errors = append(errors, "user must be at least 18 years old")
    }
    
    if !b.Terms {
        errors = append(errors, "terms and conditions must be accepted")
    }
    
    // Database validation that can fail with HTTP 500
    exists, err := checkEmailExists(b.Email) // Database call
    if err != nil {
        // Return non-ValidationError → HTTP 500 Internal Server Error
        return fmt.Errorf("failed to check email uniqueness: %w", err)
    }
    
    if exists {
        errors = append(errors, "email address is already registered")
    }
    
    // Client validation errors → HTTP 400 Bad Request
    if len(errors) > 0 {
        return &BodyValidationError{
            Errors: errors,
        }
    }
    
    return nil
}
```

#### Request-Level Validation

```go
type TransferFundsRequest struct {
    Path struct {
        AccountID string `gork:"account_id" validate:"required,uuid"`
    }
    Body struct {
        ToAccount string  `gork:"to_account" validate:"required,uuid"`
        Amount    float64 `gork:"amount" validate:"required,min=0.01"`
        Currency  string  `gork:"currency" validate:"required,iso4217"`
        Reference string  `gork:"reference" validate:"max=100"`
    }
    Headers struct {
        Authorization string `gork:"Authorization" validate:"required"`
    }
}

// Validate implements cross-section validation on the entire request
func (r *TransferFundsRequest) Validate() error {
    var errors []string
    
    // Client validation errors (HTTP 400)
    if r.Path.AccountID == r.Body.ToAccount {
        errors = append(errors, "cannot transfer funds to the same account")
    }
    
    if r.Body.Currency == "USD" && r.Body.Amount > 10000.00 {
        errors = append(errors, "USD transfers cannot exceed $10,000")
    }
    
    if r.Body.Currency == "EUR" && r.Body.Amount > 8500.00 {
        errors = append(errors, "EUR transfers cannot exceed €8,500")
    }
    
    if r.Body.Amount > 1000.00 && r.Body.Reference == "" {
        errors = append(errors, "reference is required for transfers over 1,000")
    }
    
    // Database validation that can fail with HTTP 500
    fromAccount, err := getAccountBalance(r.Path.AccountID) // Database call
    if err != nil {
        // Return non-ValidationError → HTTP 500 Internal Server Error
        return fmt.Errorf("failed to check account balance: %w", err)
    }
    
    if fromAccount.Balance < r.Body.Amount {
        errors = append(errors, "insufficient funds")
    }
    
    // External service validation that can fail with HTTP 500
    valid, err := validateCurrency(r.Body.Currency) // External API call
    if err != nil {
        // Return non-ValidationError → HTTP 500 Internal Server Error
        return fmt.Errorf("failed to validate currency: %w", err)
    }
    
    if !valid {
        errors = append(errors, "unsupported currency")
    }
    
    // Client validation errors → HTTP 400 Bad Request
    if len(errors) > 0 {
        return &RequestValidationError{
            Errors: errors,
        }
    }
    
    return nil
}
```

#### Mixed Section and Request Validation

```go
type CreateOrderQuery struct {
    NotifyUser    bool   `gork:"notify_user"`
    Priority      string `gork:"priority" validate:"oneof=low normal high urgent"`
    ProcessAfter  string `gork:"process_after"` // ISO8601 timestamp
}

// Validate implements section-level validation
func (q *CreateOrderQuery) Validate() error {
    if q.Priority == "urgent" && !q.NotifyUser {
        return &QueryValidationError{
            Errors: []string{"user notification required for urgent orders"},
        }
    }
    
    if q.ProcessAfter != "" {
        if processTime, err := time.Parse(time.RFC3339, q.ProcessAfter); err != nil {
            return &QueryValidationError{
                Errors: []string{"process_after must be valid ISO8601 timestamp"},
            }
        } else if processTime.Before(time.Now()) {
            return &QueryValidationError{
                Errors: []string{"process_after cannot be in the past"},
            }
        }
    }
    
    return nil
}

type CreateOrderBody struct {
    Items    []OrderItem `gork:"items" validate:"required,dive"`
    Discount float64     `gork:"discount" validate:"min=0,max=100"`
    Coupon   string      `gork:"coupon"`
}

// Validate implements section-level validation
func (b *CreateOrderBody) Validate() error {
    // Coupon required for discounts over 10%
    if b.Discount > 10.0 && b.Coupon == "" {
        return &BodyValidationError{
            Errors: []string{"coupon code required for discounts over 10%"},
        }
    }
    
    return nil
}

type CreateOrderRequest struct {
    Query CreateOrderQuery
    Body  CreateOrderBody
}

// Validate implements request-level cross-section validation
func (r *CreateOrderRequest) Validate() error {
    var errors []string
    
    // High priority orders cannot have discounts (business rule)
    if r.Query.Priority == "urgent" && r.Body.Discount > 0 {
        errors = append(errors, "urgent orders cannot have discounts applied")
    }
    
    // Large orders require notification
    totalItems := len(r.Body.Items)
    if totalItems > 50 && !r.Query.NotifyUser {
        errors = append(errors, "user notification required for orders with more than 50 items")
    }
    
    if len(errors) > 0 {
        return &RequestValidationError{
            Errors: errors,
        }
    }
    
    return nil
}
```

#### Complex Section Validation

```go
type UpdateOrderQuery struct {
    Status     string `gork:"status" validate:"oneof=pending processing shipped delivered cancelled"`
    NotifyUser bool   `gork:"notify_user"`
    Reason     string `gork:"reason"`
}

// Validate implements business rule validation
func (q *UpdateOrderQuery) Validate() error {
    // Reason is required when cancelling an order
    if q.Status == "cancelled" && q.Reason == "" {
        return &QueryValidationError{
            Errors: []string{"reason is required when cancelling an order"},
        }
    }
    
    // Notification only available for certain statuses
    if q.NotifyUser && q.Status == "pending" {
        return &QueryValidationError{
            Errors: []string{"user notification not available for pending orders"},
        }
    }
    
    return nil
}

type UpdateOrderRequest struct {
    Path struct {
        OrderID string `gork:"order_id" validate:"required,uuid"`
    }
    Query UpdateOrderQuery
}
```

#### Conditional Field Validation

```go
type PaymentBody struct {
    Method      string  `gork:"method" validate:"required,oneof=credit_card bank_transfer paypal"`
    Amount      float64 `gork:"amount" validate:"required,min=0.01"`
    Currency    string  `gork:"currency" validate:"required,iso4217"`
    
    // Credit card fields
    CardNumber  string `gork:"card_number"`
    ExpiryMonth int    `gork:"expiry_month"`
    ExpiryYear  int    `gork:"expiry_year"`
    CVV         string `gork:"cvv"`
    
    // Bank transfer fields
    AccountNumber string `gork:"account_number"`
    RoutingNumber string `gork:"routing_number"`
    
    // PayPal fields
    PayPalEmail string `gork:"paypal_email"`
}

func (p *PaymentBody) Validate() error {
    var errors []string
    
    switch p.Method {
    case "credit_card":
        if p.CardNumber == "" {
            errors = append(errors, "card_number is required for credit card payments")
        }
        if p.ExpiryMonth < 1 || p.ExpiryMonth > 12 {
            errors = append(errors, "expiry_month must be between 1 and 12")
        }
        if p.ExpiryYear < time.Now().Year() {
            errors = append(errors, "expiry_year cannot be in the past")
        }
        if len(p.CVV) < 3 || len(p.CVV) > 4 {
            errors = append(errors, "cvv must be 3 or 4 digits")
        }
        
    case "bank_transfer":
        if p.AccountNumber == "" {
            errors = append(errors, "account_number is required for bank transfers")
        }
        if p.RoutingNumber == "" {
            errors = append(errors, "routing_number is required for bank transfers")
        }
        
    case "paypal":
        if p.PayPalEmail == "" {
            errors = append(errors, "paypal_email is required for PayPal payments")
        }
        // Additional email format validation could be done here
    }
    
    if len(errors) > 0 {
        return &BodyValidationError{
            Errors: errors,
        }
    }
    
    return nil
}
```

#### Validation Error Types

```go
// Request-level validation error
type RequestValidationError struct {
    Errors []string `json:"errors"`
}

func (e *RequestValidationError) Error() string {
    return fmt.Sprintf("request validation failed: %s", strings.Join(e.Errors, ", "))
}

// Section-specific validation errors
type QueryValidationError struct {
    Errors []string `json:"errors"`
}

func (e *QueryValidationError) Error() string {
    return fmt.Sprintf("query validation failed: %s", strings.Join(e.Errors, ", "))
}

type BodyValidationError struct {
    Errors []string `json:"errors"`
}

func (e *BodyValidationError) Error() string {
    return fmt.Sprintf("body validation failed: %s", strings.Join(e.Errors, ", "))
}

type PathValidationError struct {
    Errors []string `json:"errors"`
}

func (e *PathValidationError) Error() string {
    return fmt.Sprintf("path validation failed: %s", strings.Join(e.Errors, ", "))
}

type HeadersValidationError struct {
    Errors []string `json:"errors"`
}

func (e *HeadersValidationError) Error() string {
    return fmt.Sprintf("headers validation failed: %s", strings.Join(e.Errors, ", "))
}

type CookiesValidationError struct {
    Errors []string `json:"errors"`
}

func (e *CookiesValidationError) Error() string {
    return fmt.Sprintf("cookies validation failed: %s", strings.Join(e.Errors, ", "))
}
```

#### Validation Error Response Format

```go
// HTTP 400 Bad Request - Validation failed (client error)
{
    "error": "Validation failed",
    "details": {
        // Field-level validation errors (from validate tags)
        "body.name": ["required", "min"],
        "body.email": ["email"],
        "query.limit": ["min"],
        "path.account_id": ["uuid"],
        
        // Section-level validation errors (from section Validate() methods returning *ValidationError)
        "body": ["passwords do not match", "coupon code required for discounts over 10%"],
        "query": ["user notification required for urgent orders"],
        
        // Request-level validation errors (from request Validate() method returning *ValidationError)
        "request": ["cannot transfer funds to the same account", "urgent orders cannot have discounts applied"]
    }
}

// HTTP 500 Internal Server Error - Infrastructure failure (server error)
{
    "error": "Internal server error",
    "message": "Request validation failed due to server error"
    // Note: Actual error details are logged but not exposed to client for security
}
```

### Response Generation Algorithm

1. **Check for standard sections** in response struct
2. **For each section present**:
   - `Body`: Marshal to JSON/other format using `gork` tags
   - `Headers`: Set HTTP headers using `gork` tag names
   - `Cookies`: Set HTTP cookies using `gork` tag names

## Tag Specifications

### `gork` Tag

**Purpose**: Define the wire format name for the field

**Format**: `gork:"field_name[,option=value,...]"`

**Rules**:
- MUST be present on all fields that are transmitted over the wire
- Name MUST match the expected parameter/field name in HTTP
- For headers, use the exact header name (e.g., `Authorization`, `Content-Type`)
- For cookies, use the exact cookie name
- Use snake_case for JSON fields by convention
- Use kebab-case for query parameters by convention

**Options**:
- `discriminator=value` - Specify discriminator value for union types

**Examples**:
```go
Name        string `gork:"name"`                              // JSON: "name"
UserID      string `gork:"user_id"`                           // JSON: "user_id"  
ContentType string `gork:"Content-Type"`                      // Header: "Content-Type"
APIKey      string `gork:"api-key"`                           // Query: "api-key"
Type        string `gork:"type,discriminator=email"`          // Discriminator field
```

### `validate` Tag

**Purpose**: Define validation rules using go-playground/validator syntax

**Format**: `validate:"rule1,rule2,rule3"`

**Examples**:
```go
Name   string `gork:"name" validate:"required,min=1,max=100"`
Email  string `gork:"email" validate:"required,email"`
Age    int    `gork:"age" validate:"min=18,max=120"`
Tags   []string `gork:"tags" validate:"dive,min=1,max=20"`
```

## OpenAPI Generation

### Automatic Schema Generation

The convention enables automatic OpenAPI 3.1.0 schema generation with full union type support:

```go
func generateOpenAPIOperation(requestType, responseType reflect.Type) *openapi.Operation {
    operation := &openapi.Operation{}
    
    // Process request sections
    if queryType := getSection(requestType, "Query"); queryType != nil {
        operation.Parameters = append(operation.Parameters, 
            generateQueryParameters(queryType)...)
    }
    
    if pathType := getSection(requestType, "Path"); pathType != nil {
        operation.Parameters = append(operation.Parameters,
            generatePathParameters(pathType)...)
    }
    
    if headerType := getSection(requestType, "Headers"); headerType != nil {
        operation.Parameters = append(operation.Parameters,
            generateHeaderParameters(headerType)...)
    }
    
    if bodyType := getSection(requestType, "Body"); bodyType != nil {
        operation.RequestBody = generateRequestBody(bodyType)
    }
    
    // Process response sections  
    if bodyType := getSection(responseType, "Body"); bodyType != nil {
        operation.Responses["200"] = generateResponse(responseType)
    }
    
    return operation
}
```

### Union Type Schema Generation

```go
func generateSchemaFromType(fieldType reflect.Type, validateTag string) *openapi.Schema {
    // Check if this is a union type
    if isUnionType(fieldType) {
        return generateUnionSchema(fieldType)
    }
    
    // Regular type generation
    return generateRegularSchema(fieldType, validateTag)
}

func generateUnionSchema(unionType reflect.Type) *openapi.Schema {
    // Extract generic type parameters from unions.UnionN[T1, T2, ...]
    typeArgs := extractUnionTypeArgs(unionType)
    
    oneOfSchemas := make([]*openapi.Schema, len(typeArgs))
    discriminatorMapping := make(map[string]string)
    
    for i, argType := range typeArgs {
        schema := generateSchemaFromType(argType, "")
        oneOfSchemas[i] = &openapi.Schema{
            Ref: fmt.Sprintf("#/components/schemas/%s", argType.Name()),
        }
        
        // Extract discriminator value from struct tags
        if discriminatorValue := getDiscriminatorValue(argType); discriminatorValue != "" {
            discriminatorMapping[discriminatorValue] = fmt.Sprintf("#/components/schemas/%s", argType.Name())
        }
    }
    
    return &openapi.Schema{
        OneOf: oneOfSchemas,
        Discriminator: &openapi.Discriminator{
            PropertyName: "type", // or extract from tags
            Mapping:      discriminatorMapping,
        },
    }
}

// Example generated OpenAPI schema for union types:
{
  "LoginRequest": {
    "type": "object",
    "properties": {
      "login_method": {
        "oneOf": [
          {"$ref": "#/components/schemas/EmailLogin"},
          {"$ref": "#/components/schemas/PhoneLogin"},
          {"$ref": "#/components/schemas/OAuthLogin"}
        ],
        "discriminator": {
          "propertyName": "type",
          "mapping": {
            "email": "#/components/schemas/EmailLogin",
            "phone": "#/components/schemas/PhoneLogin", 
            "oauth": "#/components/schemas/OAuthLogin"
          }
        }
      }
    }
  }
}
```

### Parameter Generation

```go
func generateQueryParameters(queryType reflect.Type) []*openapi.Parameter {
    var parameters []*openapi.Parameter
    
    for i := 0; i < queryType.NumField(); i++ {
        field := queryType.Field(i)
        gorkTag := field.Tag.Get("gork")
        validateTag := field.Tag.Get("validate")
        
        param := &openapi.Parameter{
            Name:     gorkTag,
            In:       "query", 
            Required: strings.Contains(validateTag, "required"),
            Schema:   generateSchemaFromType(field.Type, validateTag),
        }
        
        parameters = append(parameters, param)
    }
    
    return parameters
}
```

## Handler Signature

### Updated Handler Function Signature

Handlers maintain the same signature:

```go
func HandlerName(ctx context.Context, req RequestType) (*ResponseType, error)
```

### Example Handler

```go
func UpdateUser(ctx context.Context, req UpdateUserRequest) (*UpdateUserResponse, error) {
    // Access path parameters
    userID := req.Path.UserID
    
    // Access query parameters  
    force := req.Query.Force
    
    // Access request body
    newName := req.Body.Name
    newEmail := req.Body.Email
    
    // Access headers
    auth := req.Headers.Authorization
    
    // Business logic here...
    
    return &UpdateUserResponse{
        Body: struct {
            ID       string    `gork:"id"`
            Name     string    `gork:"name"`
            Updated  time.Time `gork:"updated"`
        }{
            ID:      userID,
            Name:    newName, 
            Updated: time.Now(),
        },
        Headers: struct {
            Location string `gork:"Location"`
        }{
            Location: fmt.Sprintf("/users/%s", userID),
        },
    }, nil
}
```

### Handler with Union Types

```go
func Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
    // Access the union type using generated accessor methods
    loginMethod := req.Body.LoginMethod
    
    // Handle different login methods using type switches or accessor methods
    switch {
    case loginMethod.IsEmailLogin():
        emailLogin := loginMethod.EmailLogin()
        return handleEmailLogin(ctx, emailLogin)
        
    case loginMethod.IsPhoneLogin():
        phoneLogin := loginMethod.PhoneLogin()
        return handlePhoneLogin(ctx, phoneLogin)
        
    case loginMethod.IsOAuthLogin():
        oauthLogin := loginMethod.OAuthLogin()
        return handleOAuthLogin(ctx, oauthLogin)
        
    default:
        return nil, errors.New("unknown login method")
    }
}

func CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
    // Access union types in request body
    payment := req.Body.Payment
    address := req.Body.Address
    
    // Process payment based on type
    var paymentResult string
    switch {
    case payment.IsCreditCardPayment():
        cc := payment.CreditCardPayment()
        paymentResult = processCreditCard(cc.Number, cc.CVV)
        
    case payment.IsBankTransfer():
        bank := payment.BankTransfer()
        paymentResult = processBankTransfer(bank.AccountNumber, bank.RoutingNumber)
        
    case payment.IsPayPalPayment():
        paypal := payment.PayPalPayment()
        paymentResult = processPayPal(paypal.Email)
    }
    
    // Process address based on type
    var shippingAddress string
    switch {
    case address.IsShippingAddress():
        shipping := address.ShippingAddress()
        shippingAddress = fmt.Sprintf("%s, %s, %s", shipping.Street, shipping.City, shipping.Country)
        
    case address.IsBillingAddress():
        billing := address.BillingAddress()
        shippingAddress = fmt.Sprintf("%s, %s, %s", billing.Street, billing.City, billing.Country)
    }
    
    // Return response with union type
    return &CreateOrderResponse{
        Body: struct {
            OrderID string `gork:"order_id"`
            // Union type in response
            Result unions.Union3[SuccessResult, ErrorResult, WarningResult] `gork:"result"`
        }{
            OrderID: generateOrderID(),
            Result: unions.NewUnion3[SuccessResult, ErrorResult, WarningResult](SuccessResult{
                Message: "Order created successfully",
                Data:    paymentResult,
            }),
        },
    }, nil
}
```

## Migration Guide

### Breaking Changes

**ALL existing request/response structs must be migrated** to the new convention.

### Migration Steps

1. **Identify current parameter sources** from `openapi:"in=..."` tags
2. **Group fields by parameter source** (query, body, path, headers, cookies)  
3. **Create embedded structs** for each section
4. **Convert `json` tags to `gork` tags**
5. **Update handlers** to access nested fields
6. **Test validation error responses** for proper namespacing

### Migration Example

**Before**:
```go
type UpdateUserRequest struct {
    UserID string `json:"user_id" openapi:"in=path" validate:"required,uuid"`
    Force  bool   `json:"force" openapi:"in=query"`  
    Name   string `json:"name" validate:"required,min=1,max=100"`
    Email  string `json:"email" validate:"email"`
    Auth   string `json:"authorization" openapi:"in=header" validate:"required"`
}
```

**After**:
```go
type UpdateUserRequest struct {
    Path struct {
        UserID string `gork:"user_id" validate:"required,uuid"`
    }
    Query struct {
        Force bool `gork:"force"`
    }
    Body struct {
        Name  string `gork:"name" validate:"required,min=1,max=100"`
        Email string `gork:"email" validate:"email"`
    }
    Headers struct {
        Authorization string `gork:"Authorization" validate:"required"`
    }
}
```

**Handler Update**:
```go
// Before
func UpdateUser(ctx context.Context, req UpdateUserRequest) (*UpdateUserResponse, error) {
    userID := req.UserID
    name := req.Name
}

// After  
func UpdateUser(ctx context.Context, req UpdateUserRequest) (*UpdateUserResponse, error) {
    userID := req.Path.UserID
    name := req.Body.Name
}
```

## Implementation Notes

### Parsing Order

1. Parse `Path` parameters first (from URL route)
2. Parse `Query` parameters second (from query string)
3. Parse `Headers` and `Cookies` third (from HTTP headers)
4. Parse `Body` last (from request body)

### Error Handling

- **Missing required sections**: Validation error with section name
- **Invalid field values**: Validation error with `section.field` format
- **Malformed JSON body**: HTTP 400 with parse error details
- **Unknown sections**: Ignored (forward compatibility)

### Content Type Handling

- **JSON**: Default for `Body` sections
- **Form data**: Supported for `Body` sections with appropriate Content-Type
- **Multipart**: Supported for file uploads in `Body` sections
- **Query encoding**: Standard URL query parameter encoding

### Complex Type Parsing

The framework supports automatic parsing of complex types from string-based request parameters (path, query, headers, cookies). This enables automatic entity resolution and custom type conversion.

#### Type Parser Registration

Register type parsers using a clean API that infers types from function signatures:

```go
// Register entity loaders
api.RegisterTypeParser(func(ctx context.Context, id string) (*User, error) {
    return userService.GetByID(ctx, id)
})

api.RegisterTypeParser(func(ctx context.Context, id string) (*Company, error) {
    return companyService.GetByID(ctx, id)
})

// Register standard library type parsers
api.RegisterTypeParser(func(ctx context.Context, s string) (*time.Time, error) {
    t, err := time.Parse(time.RFC3339, s)
    return &t, err
})

api.RegisterTypeParser(func(ctx context.Context, s string) (*uuid.UUID, error) {
    id, err := uuid.Parse(s)
    return &id, err
})

api.RegisterTypeParser(func(ctx context.Context, s string) (*url.URL, error) {
    return url.Parse(s)
})
```

#### Parser Function Requirements

Type parser functions must have the exact signature:
```go
func(ctx context.Context, value string) (*T, error)
```

Where:
- `ctx` - Request context for timeouts, tracing, cancellation
- `value` - String value from the request (path param, query param, header, cookie)
- `*T` - Pointer to the target type
- `error` - Parsing/loading error

#### Usage in Request Structures

Once registered, complex types work automatically in any section:

```go
type GetUserRequest struct {
    Path struct {
        User    User      `gork:"user_id"`    // Auto-resolves using User parser
        Company Company   `gork:"company_id"` // Auto-resolves using Company parser
    }
    Query struct {
        Since   time.Time `gork:"since"`      // Auto-resolves using time.Time parser
        TraceID uuid.UUID `gork:"trace_id"`   // Auto-resolves using uuid.UUID parser
    }
    Headers struct {
        Referer url.URL   `gork:"Referer"`    // Auto-resolves using url.URL parser
    }
}

type UpdateUserRequest struct {
    Path struct {
        User User `gork:"user_id"`  // Fully loaded User entity, not just ID
    }
    Body struct {
        Name  string `gork:"name" validate:"required"`
        Email string `gork:"email" validate:"required,email"`
    }
}
```

#### Error Handling for Complex Types

Type parsing errors are handled based on the error type returned:

```go
func(ctx context.Context, id string) (*User, error) {
    // Validate ID format first
    if !isValidUUID(id) {
        return nil, &PathValidationError{
            Errors: []string{"user_id must be a valid UUID"},
        }
    }
    
    // Load from database
    user, err := userRepo.GetByID(ctx, id)
    if err == sql.ErrNoRows {
        return nil, &PathValidationError{
            Errors: []string{"user not found"},
        }
    }
    if err != nil {
        // Database error → HTTP 500
        return nil, fmt.Errorf("failed to load user: %w", err)
    }
    
    return user, nil
}
```

**Error Classification:**
- `*ValidationError` types → HTTP 400 Bad Request (client error)
- Any other error → HTTP 500 Internal Server Error (server error)

#### Advanced Examples

**Entity with Validation:**
```go
type TransferRequest struct {
    Path struct {
        FromAccount Account `gork:"from_account_id"`
        ToAccount   Account `gork:"to_account_id"`
    }
    Body struct {
        Amount   float64 `gork:"amount" validate:"required,min=0.01"`
        Currency string  `gork:"currency" validate:"required,iso4217"`
    }
}

// Validate implements cross-entity validation
func (r *TransferRequest) Validate() error {
    // Entities are already loaded by type parsers
    if r.Path.FromAccount.ID == r.Path.ToAccount.ID {
        return &RequestValidationError{
            Errors: []string{"cannot transfer to the same account"},
        }
    }
    
    if r.Path.FromAccount.Balance < r.Body.Amount {
        return &RequestValidationError{
            Errors: []string{"insufficient funds"},
        }
    }
    
    return nil
}
```

**Multiple Complex Types:**
```go
type CreateOrderRequest struct {
    Path struct {
        Customer Customer `gork:"customer_id"`
    }
    Query struct {
        DeliveryDate time.Time `gork:"delivery_date"`
        Priority     string    `gork:"priority" validate:"oneof=low normal high"`
    }
    Body struct {
        Items []OrderItem `gork:"items" validate:"required,dive"`
        Notes string      `gork:"notes" validate:"max=500"`
    }
}
```

**Conditional Type Parsing:**
```go
// Parser can return different results based on context
api.RegisterTypeParser(func(ctx context.Context, id string) (*User, error) {
    // Check user permissions from context
    currentUser := auth.GetUser(ctx)
    if !currentUser.CanAccessUser(id) {
        return nil, &PathValidationError{
            Errors: []string{"access denied"},
        }
    }
    
    return userService.GetByID(ctx, id)
})
```

#### Implementation Notes

**Type Registry:**
- Framework uses reflection on parser function signatures to build internal type registry
- Registry maps `reflect.Type` → parser function
- Validation ensures parser signature matches requirements

**Parsing Order:**
- Complex type parsing happens after basic field extraction but before validation
- Failed parsing results in immediate error response
- Successful parsing allows normal validation to proceed

**Performance Considerations:**
- Parser functions should implement appropriate caching
- Database connections should use connection pooling
- Context cancellation should be respected for timeouts

**Security Considerations:**
- Always validate input format before database queries
- Implement proper authorization checks within parsers
- Use parameterized queries to prevent SQL injection
- Rate limit expensive parsing operations

### Performance Considerations

- **Reflection caching**: Cache struct analysis results
- **Lazy parsing**: Only parse sections that are present in the struct
- **Memory efficiency**: Use field-level parsing for large requests
- **Type parser caching**: Cache type parser registry and reflection analysis

## Future Extensions

### Potential Additions

1. **Custom section names**: Allow user-defined section names with validation
2. **Alternative encodings**: Support for XML, YAML, Protocol Buffers
3. **Conditional sections**: Sections that depend on other field values
4. **Nested sections**: Support for subsections within main sections
5. **Cross-section validation**: Validation that spans multiple sections
6. **Async section validation**: Support for validation that requires external calls

### Backwards Compatibility

This specification represents a **breaking change** with **no backwards compatibility** for the existing tag-based approach. All codebases must migrate to the new convention.

## Enhanced Linter Support

The convention over configuration approach enables enhanced static analysis through linter improvements that validate handler structure compliance.

### Linter Enhancements

#### Handler Structure Validation

The linter should validate that all registered handlers conform to the convention over configuration structure:

```go
// lintgork/handler_structure.go

type HandlerStructureChecker struct{}

func NewHandlerStructureChecker() *HandlerStructureChecker {
    return &HandlerStructureChecker{}
}

// Standard section names that are allowed by the framework
var allowedSections = map[string]bool{
    "Query":   true,
    "Body":    true,
    "Path":    true,
    "Headers": true,
    "Cookies": true,
}

func (h *HandlerStructureChecker) CheckHandler(handlerFunc *ast.FuncDecl) []LintError {
    var errors []LintError
    
    // Validate handler signature
    if err := h.validateHandlerSignature(handlerFunc); err != nil {
        errors = append(errors, err...)
    }
    
    // Extract request type from second parameter
    requestType := h.extractRequestType(handlerFunc)
    if requestType == nil {
        return errors
    }
    
    // Validate request structure
    if err := h.validateRequestStructure(requestType); err != nil {
        errors = append(errors, err...)
    }
    
    return errors
}

func (h *HandlerStructureChecker) validateRequestStructure(structType *ast.StructType) []LintError {
    var errors []LintError
    
    for _, field := range structType.Fields.List {
        if len(field.Names) == 0 {
            continue // Anonymous field
        }
        
        fieldName := field.Names[0].Name
        
        // Check if field name is a standard section
        if !allowedSections[fieldName] {
            errors = append(errors, LintError{
                Message: fmt.Sprintf("invalid section name '%s', must be one of: Query, Body, Path, Headers, Cookies", fieldName),
                Node:    field,
            })
            continue
        }
        
        // Validate section structure
        if err := h.validateSectionStructure(fieldName, field.Type); err != nil {
            errors = append(errors, err...)
        }
    }
    
    return errors
}
```

#### Section Structure Validation

```go
func (h *HandlerStructureChecker) validateSectionStructure(sectionName string, sectionType ast.Expr) []LintError {
    var errors []LintError
    
    // Section must be a struct type
    structType, ok := sectionType.(*ast.StructType)
    if !ok {
        errors = append(errors, LintError{
            Message: fmt.Sprintf("section '%s' must be a struct type", sectionName),
            Node:    sectionType,
        })
        return errors
    }
    
    // Validate fields within section
    for _, field := range structType.Fields.List {
        if err := h.validateSectionField(sectionName, field); err != nil {
            errors = append(errors, err...)
        }
    }
    
    return errors
}

func (h *HandlerStructureChecker) validateSectionField(sectionName string, field *ast.Field) []LintError {
    var errors []LintError
    
    if len(field.Names) == 0 {
        return errors // Anonymous field
    }
    
    fieldName := field.Names[0].Name
    
    // Check for gork tag
    if field.Tag == nil {
        errors = append(errors, LintError{
            Message: fmt.Sprintf("field '%s.%s' missing gork tag", sectionName, fieldName),
            Node:    field,
        })
        return errors
    }
    
    // Parse and validate gork tag
    gorkTag := h.extractGorkTag(field.Tag.Value)
    if gorkTag == "" {
        errors = append(errors, LintError{
            Message: fmt.Sprintf("field '%s.%s' missing gork tag", sectionName, fieldName),
            Node:    field,
        })
    }
    
    // Validate gork tag format
    if err := h.validateGorkTag(sectionName, fieldName, gorkTag); err != nil {
        errors = append(errors, err...)
    }
    
    return errors
}
```

#### Tag Validation

```go
func (h *HandlerStructureChecker) validateGorkTag(sectionName, fieldName, gorkTag string) []LintError {
    var errors []LintError
    
    // Parse gork tag: "field_name[,option=value,...]"
    parts := strings.Split(gorkTag, ",")
    if len(parts) == 0 {
        errors = append(errors, LintError{
            Message: fmt.Sprintf("field '%s.%s' has empty gork tag", sectionName, fieldName),
        })
        return errors
    }
    
    wireFormat := strings.TrimSpace(parts[0])
    if wireFormat == "" {
        errors = append(errors, LintError{
            Message: fmt.Sprintf("field '%s.%s' gork tag missing wire format name", sectionName, fieldName),
        })
    }
    
    // Validate options (discriminator, etc.)
    for i := 1; i < len(parts); i++ {
        option := strings.TrimSpace(parts[i])
        if err := h.validateGorkOption(sectionName, fieldName, option); err != nil {
            errors = append(errors, err...)
        }
    }
    
    return errors
}

```

#### Union Type Validation

```go
func (h *HandlerStructureChecker) validateUnionType(field *ast.Field) []LintError {
    var errors []LintError
    
    // Check if field type is unions.UnionN
    if !h.isUnionType(field.Type) {
        return errors
    }
    
    // Extract union type arguments
    unionArgs := h.extractUnionTypeArgs(field.Type)
    
    // Validate each union variant has discriminator
    for i, arg := range unionArgs {
        if err := h.validateUnionVariant(arg, i); err != nil {
            errors = append(errors, err...)
        }
    }
    
    return errors
}

func (h *HandlerStructureChecker) validateUnionVariant(variantType ast.Expr, index int) []LintError {
    var errors []LintError
    
    structType, ok := variantType.(*ast.StructType)
    if !ok {
        errors = append(errors, LintError{
            Message: fmt.Sprintf("union variant %d must be a struct type", index),
            Node:    variantType,
        })
        return errors
    }
    
    // Check for discriminator field
    hasDiscriminator := false
    discriminatorValues := make(map[string]bool)
    
    for _, field := range structType.Fields.List {
        if field.Tag == nil {
            continue
        }
        
        gorkTag := h.extractGorkTag(field.Tag.Value)
        if strings.Contains(gorkTag, "discriminator=") {
            hasDiscriminator = true
            
            // Extract discriminator value
            discValue := h.extractDiscriminatorValue(gorkTag)
            if discriminatorValues[discValue] {
                errors = append(errors, LintError{
                    Message: fmt.Sprintf("duplicate discriminator value '%s' in union", discValue),
                    Node:    field,
                })
            }
            discriminatorValues[discValue] = true
        }
    }
    
    if !hasDiscriminator {
        errors = append(errors, LintError{
            Message: fmt.Sprintf("union variant %d missing discriminator field", index),
            Node:    structType,
        })
    }
    
    return errors
}
```

#### Response Structure Validation

```go
func (h *HandlerStructureChecker) validateResponseStructure(responseType ast.Expr) []LintError {
    var errors []LintError
    
    // Response should be a pointer to a struct
    starExpr, ok := responseType.(*ast.StarExpr)
    if !ok {
        errors = append(errors, LintError{
            Message: "handler response type must be a pointer to struct",
            Node:    responseType,
        })
        return errors
    }
    
    structType, ok := starExpr.X.(*ast.StructType)
    if !ok {
        errors = append(errors, LintError{
            Message: "handler response type must be a pointer to struct",
            Node:    starExpr.X,
        })
        return errors
    }
    
    // Validate response sections
    allowedResponseSections := map[string]bool{
        "Body":    true,
        "Headers": true,
        "Cookies": true,
    }
    
    for _, field := range structType.Fields.List {
        if len(field.Names) == 0 {
            continue
        }
        
        fieldName := field.Names[0].Name
        if !allowedResponseSections[fieldName] {
            errors = append(errors, LintError{
                Message: fmt.Sprintf("invalid response section '%s', must be one of: Body, Headers, Cookies", fieldName),
                Node:    field,
            })
        }
    }
    
    return errors
}
```

### Linter Integration

#### Command Line Usage

```bash
# Check handler structure compliance
lintgork -checks=handler-structure ./handlers/...

# Check with specific rules including naming conventions
lintgork -checks=handler-structure,naming-conventions,union-discriminators ./...

# Generate report
lintgork -checks=handler-structure -format=json -output=report.json ./...
```

#### Configuration

```yaml
# .lintgork.yml
checks:
  handler-structure:
    enabled: true
    require-gork-tags: true
    validate-union-discriminators: true
    
  section-structure:
    enabled: true
    
  response-structure:
    enabled: true
    
  naming-conventions:
    enabled: false  # Optional lint-time enforcement
    query-params: "kebab-case"  # or "snake_case"
    body-fields: "snake_case"
    path-params: "snake_case"
    headers: "http-header-case"
    cookies: "valid-cookie-name"
```

### Example Linter Output

```
handlers/user.go:15:1: invalid section name 'Params', must be one of: Query, Body, Path, Headers, Cookies
handlers/user.go:18:2: field 'Query.user_id' missing gork tag
handlers/user.go:22:2: body field 'userName' should use snake_case (naming-conventions)
handlers/user.go:35:1: union variant 0 missing discriminator field
handlers/payment.go:42:2: duplicate discriminator value 'card' in union
```

### Integration with IDE

```go
// VS Code extension support
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "--enable=lintgork",
        "--lintgork.handler-structure=true"
    ]
}
```

This enhanced linter support ensures that all handler structures conform to the convention over configuration approach, catching structural issues at development time rather than runtime.

## Conclusion

This convention over configuration approach provides:

✅ **Explicit, self-documenting request structures**  
✅ **Consistent naming across all Gork projects**  
✅ **Simplified parsing and OpenAPI generation logic**  
✅ **Better validation error reporting with section namespacing**  
✅ **Enhanced developer experience with clear parameter grouping**  
✅ **Static analysis support through enhanced linting**  

The specification enforces a strict but intuitive structure that makes HTTP APIs more maintainable and easier to understand.