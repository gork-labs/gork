# pkg/unions - Type-Safe Union Types for Go

This package provides type-safe union types (discriminated unions) for Go with full JSON marshaling support.

## Installation

```bash
go get github.com/gork-labs/gork/pkg/unions
```

## Usage

### Basic Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/gork-labs/gork/pkg/unions"
)

// Define your types
type EmailLogin struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type PhoneLogin struct {
    Phone string `json:"phone"`
    Code  string `json:"code"`
}

// Create a union type
type LoginRequest unions.Union2[EmailLogin, PhoneLogin]

func main() {
    // Create with email
    req := LoginRequest{
        Value: EmailLogin{
            Email:    "user@example.com",
            Password: "secret",
        },
    }
    
    // Marshal to JSON
    data, _ := json.Marshal(req)
    fmt.Println(string(data))
    // Output: {"email":"user@example.com","password":"secret"}
    
    // Unmarshal from JSON
    var decoded LoginRequest
    json.Unmarshal(data, &decoded)
    
    // Type-safe access
    switch v := decoded.Value.(type) {
    case EmailLogin:
        fmt.Printf("Email login: %s\n", v.Email)
    case PhoneLogin:
        fmt.Printf("Phone login: %s\n", v.Phone)
    }
}
```

### Available Union Types

- `Union2[A, B]` - Union of 2 types
- `Union3[A, B, C]` - Union of 3 types  
- `Union4[A, B, C, D]` - Union of 4 types

### JSON Marshaling

The union types automatically marshal to and from JSON:

```go
// Union2[string, int] examples:
{"value": "hello"}     // When containing a string
{"value": 42}          // When containing an int

// Union2[EmailLogin, PhoneLogin] examples:
{"email": "user@example.com", "password": "secret"}  // EmailLogin
{"phone": "+1234567890", "code": "1234"}            // PhoneLogin
```

### Validation

Union types support validation using struct tags:

```go
type ValidatedRequest unions.Union2[
    struct {
        Email string `json:"email" validate:"required,email"`
    },
    struct {
        Phone string `json:"phone" validate:"required,e164"`
    },
]
```

### Discriminators

For OpenAPI generation, you can add discriminator information:

```go
type PaymentMethod struct {
    Type string `json:"type" openapi:"discriminator"`
    Data unions.Union3[CreditCard, BankAccount, PayPal] `json:"data"`
}
```

## API Reference

### Methods

All union types provide:

- `MarshalJSON() ([]byte, error)` - JSON marshaling
- `UnmarshalJSON([]byte) error` - JSON unmarshaling
- `Value` field - Contains the actual value

### Type Checking

Use type switches or type assertions to work with union values:

```go
union := Union2[string, int]{Value: "hello"}

// Type switch
switch v := union.Value.(type) {
case string:
    fmt.Println("String:", v)
case int:
    fmt.Println("Int:", v)
}

// Type assertion
if str, ok := union.Value.(string); ok {
    fmt.Println("It's a string:", str)
}
```

## Examples

See the [examples](../../examples/) directory for complete working examples.

## License

MIT License - see the root [LICENSE](../../LICENSE) file for details.