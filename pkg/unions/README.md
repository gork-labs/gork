# pkg/unions - Type-Safe Union Types for Go

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg?flag=pkg%2Funions)](https://codecov.io/gh/gork-labs/gork/tree/main/pkg/unions)

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
    // Create with email login
    req := LoginRequest{
        A: &EmailLogin{
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
    switch {
    case decoded.A != nil:
        fmt.Printf("Email login: %s\n", decoded.A.Email)
    case decoded.B != nil:
        fmt.Printf("Phone login: %s\n", decoded.B.Phone)
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

For OpenAPI generation, you can add discriminator information using gork tags:

```go
type PaymentMethod struct {
    Type string `gork:"type,discriminator=payment"`
    Data unions.Union3[CreditCard, BankAccount, PayPal] `gork:"data"`
}
```

## API Reference

### Methods

All union types provide:

- `MarshalJSON() ([]byte, error)` - JSON marshaling
- `UnmarshalJSON([]byte) error` - JSON unmarshaling  
- `A`, `B` (and `C`, `D` for larger unions) - Pointer fields for each variant
- `Validate(*validator.Validate) error` - Built-in validation

### Type Checking

Use nil checks to determine which variant is active:

```go
union := Union2[string, int]{
    A: &"hello",  // Set first variant
}

// Check which variant is active
switch {
case union.A != nil:
    fmt.Println("String:", *union.A)
case union.B != nil:
    fmt.Println("Int:", *union.B)
}

// Direct access (with nil check)
if union.A != nil {
    fmt.Println("It's a string:", *union.A)
}
```

## Examples

See the [examples](../../examples/) directory for complete working examples.

## License

MIT License - see the root [LICENSE](../../LICENSE) file for details.