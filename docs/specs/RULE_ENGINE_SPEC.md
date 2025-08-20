# Rule Engine Specification

## Overview

The Rule Engine provides a declarative way to define business domain rules that can access any part of a parsed request. Rules are reusable business logic functions that work with multiple entity types using explicit type switches.

## Goals

1. **Business Domain Focus**: Define business logic rules, not request format validation
2. **Cross-Section Access**: Rules can reference entities from any request section (Path, Query, Body, Headers, Cookies)
3. **Type Safety**: Explicit type switches for each supported entity type
4. **Reusability**: Rules work across different request types and entity combinations
5. **Performance**: No reflection overhead during rule execution

## Core Concepts

### Rule Definition

Rules are functions registered globally that validate entities based on their type and optional arguments from other parts of the request.

```go
// Single entity rule signature
type SingleEntityRule func(ctx context.Context, entity any) error

// Multi-argument rule signature (variadic)
type MultiArgumentRule func(ctx context.Context, entity any, args ...any) error
```

Rules must use explicit type switches inside their body and return validation errors mapped to HTTP 400 (see Error Handling). Internal/server errors should be returned as `error` to produce HTTP 500.

### Naming and Registry

- Rule names are process‑global and must be unique. Registering the same name twice panics with a clear error.
- Registration is intended to happen at init/startup time. The registry is read‑mostly and goroutine‑safe for lookups during request handling.
- Recommended naming convention uses dotted namespaces to avoid collisions, e.g., `auth.admin`, `billing.owned_by`.

### Rule Registration

Rules are registered globally using explicit type switches:

```go
// Single entity rules
api.RegisterRule("admin", func(ctx context.Context, entity interface{}) error {
    switch e := entity.(type) {
    case *User:
        if !e.IsAdmin {
            return &PathValidationError{Errors: []string{"user must be admin"}}
        }
    case *Role:
        if !e.IsAdminRole {
            return &PathValidationError{Errors: []string{"role must be admin"}}
        }
    case *Account:
        if e.AccountType != "admin" {
            return &PathValidationError{Errors: []string{"account must be admin"}}
        }
    default:
        return fmt.Errorf("admin rule not supported for type %T", entity)
    }
    return nil
})

// Cross-entity rules
api.RegisterRule("owned_by", func(ctx context.Context, entity interface{}, owner interface{}) error {
    switch e := entity.(type) {
    case *Item:
        switch o := owner.(type) {
        case *User:
            if e.OwnerID != o.ID {
                return &PathValidationError{Errors: []string{"item not owned by this user"}}
            }
        case *Organization:
            if e.OrgID != o.ID {
                return &PathValidationError{Errors: []string{"item not owned by this organization"}}
            }
        }
    case *Project:
        switch o := owner.(type) {
        case *User:
            if e.UserID != o.ID {  // Different field name, no problem!
                return &PathValidationError{Errors: []string{"project not owned by this user"}}
            }
        case *Organization:
            if e.OrgID != o.ID {
                return &PathValidationError{Errors: []string{"project not owned by this organization"}}
            }
        }
    default:
        return fmt.Errorf("owned_by rule not supported for entity type %T", entity)
    }
    return nil
})

// Cross-section rules
api.RegisterRule("category_matches", func(ctx context.Context, entity interface{}, category interface{}) error {
    switch e := entity.(type) {
    case *Item:
        categoryStr := category.(string)  // From Body.Category
        if e.Category != categoryStr {
            return &PathValidationError{
                Errors: []string{"item category does not match request category"},
            }
        }
    case *Product:
        categoryStr := category.(string)
        if e.ProductCategory != categoryStr {
            return &PathValidationError{
                Errors: []string{"product category does not match request category"},
            }
        }
    }
    return nil
})
```

### Rule Usage in Request Structures

Rules are applied using the `rule` tag with optional arguments referencing other parts of the request:

```go
type UpdateItemRequest struct {
  Path struct {
    User User `gork:"user_id" rule:"admin"`
    Item Item `gork:"item_id" rule:"owned_by($.Path.User) && category_matches($.Body.Category)"`
  }
  Query struct {
    Force bool `gork:"force"`
  }
  Body struct {
    Name     string `gork:"name"`
    Category string `gork:"category"`
  }
  Headers struct {
    Authorization string `gork:"Authorization"`
  }
}
```

## Rule Reference Syntax

### Argument Grammar

Arguments in `rule:"..."` use a simple, explicit grammar:

- Field reference (two forms only):
  - Absolute: `$.X.Y[.Z...]` — traverse from the request root. Example: `$.Path.User`, `$.Headers.Authorization`.
  - Relative: `.X.Y[.Z...]` — resolve from the same parent struct as the annotated field (typically the same section). Example (on `Path.Item`): `.User` resolves to `Path.User`.
  - Nested struct traversal is supported. Slice/map indexing is NOT supported in v1.
- String literal: `'text'` or `"text"` (quotes required when passing strings directly).
- Number literal: `123`, `45.67` (parsed as float64; rule code can convert as needed).
- Boolean literal: `true`, `false`.
- Null literal: `null` (passed as nil).
- Context variable: `$varName` — pulls a value from per-request context variables, set via `rules.WithContextVars(ctx, rules.ContextVars{...})`. Useful for current user, permissions, etc. Example: `owned_by($current_user)`.

No inline operators or expressions are supported in v1 (e.g., `Type=admin`). Pass values explicitly as separate arguments and implement comparisons inside the rule.

### Multiple Arguments

Rules can accept multiple arguments separated by commas:

```go
Item `gork:"item_id" rule:"owned_by($.Path.User) && category_matches($.Body.Category) && within_limit($.Query.MaxPrice)"`
```

### Chained Rules

Multiple rules can be applied to a single field using boolean expressions with && (AND) and || (OR) operators:

```go
User `gork:"user_id" rule:"admin() && active() && verified()"`
```

## Comprehensive Examples

### Simple Entity Validation

```go
type GetUserRequest struct {
  Path struct {
    User User `gork:"user_id" rule:"active() && verified()"`
  }
}

api.RegisterRule("active", func(ctx context.Context, entity interface{}) error {
    switch e := entity.(type) {
    case *User:
        if !e.IsActive {
            return &PathValidationError{Errors: []string{"user must be active"}}
        }
    case *Account:
        if e.Status != "active" {
            return &PathValidationError{Errors: []string{"account must be active"}}
        }
    }
    return nil
})

api.RegisterRule("verified", func(ctx context.Context, entity interface{}) error {
    switch e := entity.(type) {
    case *User:
        if !e.EmailVerified {
            return &PathValidationError{Errors: []string{"user email must be verified"}}
        }
    }
    return nil
})
```

### Cross-Entity Validation

```go
type TransferFundsRequest struct {
  Path struct {
    FromAccount Account `gork:"from_account_id" rule:"owned_by($.Headers.User) && sufficient_balance($.Body.Amount)"`
    ToAccount   Account `gork:"to_account_id" rule:"accepts_currency($.Body.Currency) && not_same_as($.Path.FromAccount)"`
  }
  Body struct {
    Amount   float64 `gork:"amount"`
    Currency string  `gork:"currency"`
  }
  Headers struct {
    User User `gork:"X-User-ID"`  // Parsed using registered User type parser
  }
}

api.RegisterRule("sufficient_balance", func(ctx context.Context, entity interface{}, amount interface{}) error {
    switch e := entity.(type) {
    case *Account:
        amountVal := amount.(float64)
        if e.Balance < amountVal {
            return &PathValidationError{
                Errors: []string{"insufficient account balance"},
            }
        }
    }
    return nil
})

api.RegisterRule("accepts_currency", func(ctx context.Context, entity interface{}, currency interface{}) error {
    switch e := entity.(type) {
    case *Account:
        currencyStr := currency.(string)
        for _, accepted := range e.AcceptedCurrencies {
            if accepted == currencyStr {
                return nil
            }
        }
        return &PathValidationError{
            Errors: []string{fmt.Sprintf("account does not accept currency %s", currencyStr)},
        }
    }
    return nil
})

api.RegisterRule("not_same_as", func(ctx context.Context, entity interface{}, other interface{}) error {
    switch e := entity.(type) {
    case *Account:
        switch o := other.(type) {
        case *Account:
            if e.ID == o.ID {
                return &RequestValidationError{
                    Errors: []string{"cannot transfer to the same account"},
                }
            }
        }
    }
    return nil
})
```

### Authorization-Based Rules

```go
type DeleteProjectRequest struct {
  Path struct {
    User    User    `gork:"user_id"`
    Project Project `gork:"project_id" rule:"owned_by($.Path.User) && deletable_by($.Path.User)"`
  }
}

api.RegisterRule("authorized_for", func(ctx context.Context, entity interface{}, authHeader interface{}) error {
    switch e := entity.(type) {
    case *User:
        authStr := authHeader.(string)
        token := strings.TrimPrefix(authStr, "Bearer ")
        
        claims, err := jwt.ValidateToken(token)
        if err != nil {
            return &HeadersValidationError{
                Errors: []string{"invalid authorization token"},
            }
        }
        
        if claims.UserID != e.ID {
            return &HeadersValidationError{
                Errors: []string{"token does not match user"},
            }
        }
    }
    return nil
})

api.RegisterRule("deletable_by", func(ctx context.Context, entity interface{}, user interface{}) error {
    switch e := entity.(type) {
    case *Project:
        switch u := user.(type) {
        case *User:
            // Only owners or admins can delete projects
            if e.OwnerID != u.ID && !u.IsAdmin {
                return &PathValidationError{
                    Errors: []string{"insufficient permissions to delete project"},
                }
            }
        }
    }
    return nil
})
```

### Conditional Rules Based on Request Data

```go
type UpdateUserRequest struct {
  Path struct {
    User User `gork:"user_id" rule:"admin_or_self($.Headers.CurrentUser) && changeable_if($.Query.Force)"`
  }
  Query struct {
    Force bool `gork:"force"`
  }
  Body struct {
    Role string `gork:"role"`
  }
  Headers struct {
    CurrentUser User `gork:"X-Current-User-ID"`  // Parsed using registered User type parser
  }
}

api.RegisterRule("admin_or_self", func(ctx context.Context, entity interface{}, currentUser interface{}) error {
    switch e := entity.(type) {
    case *User:
        switch cu := currentUser.(type) {
        case *User:
            // Must be admin or editing own profile
            if !cu.IsAdmin && e.ID != cu.ID {
                return &PathValidationError{
                    Errors: []string{"can only edit own profile unless admin"},
                }
            }
        }
    }
    return nil
})

api.RegisterRule("changeable_if", func(ctx context.Context, entity interface{}, force interface{}) error {
    switch e := entity.(type) {
    case *User:
        forceFlag := force.(bool)
        
        // Some users require force flag to be modified
        if e.IsProtected && !forceFlag {
            return &QueryValidationError{
                Errors: []string{"force flag required to modify protected user"},
            }
        }
    }
    return nil
})
```

### Complex Business Rules

```go
type PlaceOrderRequest struct {
  Path struct {
    Customer Customer `gork:"customer_id" rule:"active() && verified() && credit_limit($.Body.TotalAmount)"`
    Product  Product  `gork:"product_id" rule:"available() && in_region($.Path.Customer.Region) && category_allowed($.Path.Customer.Type)"`
  }
  Body struct {
    Quantity    int     `gork:"quantity"`
    TotalAmount float64 `gork:"total_amount"`
  }
}

api.RegisterRule("credit_limit", func(ctx context.Context, entity interface{}, amount interface{}) error {
    switch e := entity.(type) {
    case *Customer:
        amountVal := amount.(float64)
        if e.CreditLimit < amountVal {
            return &PathValidationError{
                Errors: []string{"order exceeds customer credit limit"},
            }
        }
    }
    return nil
})

api.RegisterRule("in_region", func(ctx context.Context, entity interface{}, customer interface{}) error {
    switch e := entity.(type) {
    case *Product:
        switch c := customer.(type) {
        case *Customer:
            for _, region := range e.AvailableRegions {
                if region == c.Region {
                    return nil
                }
            }
            return &PathValidationError{
                Errors: []string{"product not available in customer region"},
            }
        }
    }
    return nil
})

api.RegisterRule("category_allowed", func(ctx context.Context, entity interface{}, customer interface{}) error {
    switch e := entity.(type) {
    case *Product:
        switch c := customer.(type) {
        case *Customer:
            // Business customers can buy restricted products
            if e.Category == "restricted" && c.Type != "business" {
                return &PathValidationError{
                    Errors: []string{"product category restricted to business customers"},
                }
            }
        }
    }
    return nil
})
```

## Implementation Details

### Rule Execution Order

1. **Parse all sections** - Extract and parse Path, Query, Body, Headers, Cookies
2. **Load complex types** - Execute type parsers for entities
3. **Resolve rule arguments** - Extract referenced values from parsed sections
4. **Execute rules** - Call rule functions with entity and resolved arguments
5. **Collect errors** - Aggregate rule validation errors with other validation errors

### Rule Argument Resolution

The framework must resolve rule arguments by:

1. **Parsing argument strings** - `"Path.User"` → section: "Path", field: "User"
2. **Looking up parsed values** - Find the actual value in the parsed request structure
3. **Type conversion** - Pass resolved values to the rule function; the rule must perform explicit type switches/assertions.
4. **Error handling** - If a field reference cannot be resolved (missing section/field), return a `*RequestValidationError` with a clear message (HTTP 400). Literal parsing errors are also reported as `*RequestValidationError`.

### Execution Semantics

- Ordering: For a given field, rules execute in declaration order (left to right in the tag string).
- Aggregation: All validation errors from rules on a field are aggregated and returned alongside other validation errors. Execution continues after individual rule failures unless a rule returns a non‑validation `error` (treated as server error and short‑circuits processing).
- Scope: Rules execute only after successful request parsing and standard validation.
- Error typing: For absolute refs beginning with `$.Path/$.Query/$.Body/$.Headers/$.Cookies`, map errors to the corresponding validation type. For relative refs `.X...`, inherit the parent’s section if it is one of the five; otherwise use request‑level validation error.

### Thread Safety and Lifecycle

- Rule registration occurs during init/startup and panics on duplicate names.
- The registry is safe for concurrent reads during request handling.
- Dynamic registration after the first request is discouraged; if used, it must acquire internal locks (implementation detail) and still respects uniqueness.

### Error Handling

Rules return the same validation error types as other validation:

- `*PathValidationError` - Invalid path parameter
- `*QueryValidationError` - Invalid query parameter  
- `*BodyValidationError` - Invalid body field
- `*HeadersValidationError` - Invalid header
- `*CookiesValidationError` - Invalid cookie
- `*RequestValidationError` - Cross-section validation error
- `error` - Internal server error (HTTP 500)

### Performance Considerations

- **Rule registry caching** - Cache rule function lookups
- **Argument resolution caching** - Cache field path parsing and resolved accessors for nested field traversal
- **Type switch optimization** - Minimize reflection overhead
- **Lazy evaluation** - Only execute rules for successfully parsed entities

### Limitations (v1)

- No inline expressions/operators in rule arguments; pass values explicitly.
- Nested struct traversal is supported; slice/map indexing is not.
- Field references to `Body` require a structured Body. When a raw `[]byte` Body is used (e.g., some webhooks), Body field references cannot be resolved.

## Advanced Features

### Context-Aware Rules

Rules can access request context for user information, tracing, timeouts:

```go
api.RegisterRule("permission", func(ctx context.Context, entity interface{}) error {
    currentUser := auth.GetUser(ctx)
    
    switch e := entity.(type) {
    case *Document:
        if !hasPermission(currentUser, e, "read") {
            return &PathValidationError{
                Errors: []string{"insufficient permissions"},
            }
        }
    }
    return nil
})
```

### Database-Backed Rules

Rules can make database calls for complex validation:

```go
api.RegisterRule("unique_email", func(ctx context.Context, entity interface{}) error {
    switch e := entity.(type) {
    case *User:
        exists, err := userRepo.EmailExists(ctx, e.Email)
        if err != nil {
            // Database error → HTTP 500
            return fmt.Errorf("failed to check email uniqueness: %w", err)
        }
        if exists {
            return &BodyValidationError{
                Errors: []string{"email address already in use"},
            }
        }
    }
    return nil
})
```

### Conditional Rule Application

Rules can be applied conditionally based on other fields:

```go
type CreateUserRequest struct {
  Body struct {
    Type     string `gork:"type" validate:"oneof=admin user guest"`
    User     User   `gork:"user" rule:"admin_required_if(.Type, 'admin')"`
    Password string `gork:"password" rule:"strong_if(.Type, 'admin')"`
  }
}
```

## Rule Engine vs Validation

The rule engine is **purely business-domain oriented** and separate from request validation:

- **`validate` tags** - Request format validation (required, email, min/max, etc.)
- **`rule` tags** - Business domain validation (ownership, permissions, business constraints)

### Clear Separation of Concerns

**Request Validation:**
```go
type UpdateUserRequest struct {
  Path struct {
    UserID string `gork:"user_id" validate:"required,uuid"`  // Format validation
  }
  Body struct {
    Email string `gork:"email" validate:"required,email"`   // Format validation
    Age   int    `gork:"age" validate:"min=18,max=120"`     // Range validation
  }
}
```

**Business Rules:**
```go
type UpdateUserRequest struct {
  Path struct {
    User User `gork:"user_id" rule:"admin_or_self(Headers.CurrentUser)"` // Business logic
  }
  Body struct {
    Email string `gork:"email" validate:"required,email"`
  }
  Headers struct {
    CurrentUser User `gork:"X-Current-User-ID"`
  }
}
```

### Processing Order

1. **Parse sections** - Extract Path, Query, Body, Headers, Cookies
2. **Request validation** - Run `validate` tags (format, required, range)
3. **Type parsing** - Execute complex type parsers
4. **Business rules** - Execute `rule` tags (domain logic)

Business rules only run after successful request validation and type parsing.

## Future Enhancements

### Rule Composition

```go
User `gork:"user_id" rule:"active_admin"` // Composed rule

api.RegisterComposedRule("active_admin", []string{"active", "admin"})
```

### Rule Templates

```go
api.RegisterRuleTemplate("owned_by_current_user", "owned_by(Headers.Authorization)")
```

### Rule Documentation

Documentation is extracted from standard Go comments above rule registration:

```go
// admin validates that the entity represents an admin user or role.
// Supported types: User, Role, Account
api.RegisterRule("admin", func(ctx context.Context, entity interface{}) error {
    switch e := entity.(type) {
    case *User:
        if !e.IsAdmin {
            return &PathValidationError{Errors: []string{"user must be admin"}}
        }
    case *Role:
        if !e.IsAdminRole {
            return &PathValidationError{Errors: []string{"role must be admin"}}
        }
    default:
        return fmt.Errorf("admin rule not supported for type %T", entity)
    }
    return nil
})

// owned_by validates that the entity is owned by the specified owner.
// Supports various owner/entity combinations based on business domain.
api.RegisterRule("owned_by", func(ctx context.Context, entity interface{}, owner interface{}) error {
    // Rule implementation
})
```

The framework can automatically extract rule documentation for OpenAPI generation and CLI help.

### Linter Integration

The linter validates rule registrations and usage:

#### Rule Registration Validation

```bash
# Check rule registrations
lintgork -checks=rule-registration ./rules/...

# Example violations:
rules/auth.go:15:1: rule function must have signature func(context.Context, interface{}, ...interface{}) error
rules/auth.go:23:1: rule 'admin' missing documentation comment
rules/business.go:45:1: rule function must start with lowercase name matching rule name
```

#### Rule Usage Validation

```bash
# Check rule usage in request structs
lintgork -checks=rule-usage ./handlers/...

# Example violations:
handlers/user.go:12:5: rule 'admin' not registered
handlers/user.go:15:5: rule argument 'Path.NonExistentField' not found in request struct
handlers/user.go:18:5: rule 'owned_by' expects 2 arguments, got 1
handlers/user.go:22:5: rule argument 'Headers.User' type User not compatible with rule parameter type Account
```

#### Linter Configuration

```yaml
# .lintgork.yml
checks:
  rule-registration:
    enabled: true
    require-documentation: true
    validate-function-signature: true
    
  rule-usage:
    enabled: true
    validate-rule-exists: true
    validate-argument-count: true
    validate-argument-types: true
    validate-field-references: true
```

#### Linter Rules

**Rule Registration Rules:**
- Rule function must have correct signature: `func(context.Context, interface{}, ...interface{}) error`
- Rule name must match function documentation comment
- Rule must have documentation comment above registration
- Rule function should handle `default` case in type switches
- Rule should return appropriate error types (`*ValidationError` vs `error`)

**Rule Usage Rules:**
- Referenced rules must be registered
- Rule argument count must match registration
- Field references must exist in request struct (`Path.User`, `Body.Category`)
- Argument types must be compatible with rule parameter types
- Cross-section references must be valid

### Rule Testing Framework

```go
func TestAdminRule(t *testing.T) {
    rule := api.GetRule("admin")
    
    user := &User{IsAdmin: true}
    err := rule(context.Background(), user)
    assert.NoError(t, err)
    
    user.IsAdmin = false
    err = rule(context.Background(), user)
    assert.Error(t, err)
}
```

## Conclusion

The Rule Engine provides a powerful, declarative way to define complex validation logic that spans multiple parts of HTTP requests. By using explicit type switches and cross-section references, rules become reusable building blocks for business logic validation while maintaining type safety and performance.
