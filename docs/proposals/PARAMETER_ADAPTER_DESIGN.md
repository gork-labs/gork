# Parameter Adapter Design - Runtime Registration with Instance-Based Registry

## Executive Summary

This design eliminates fragile AST parsing by introducing runtime route registration with instance-based registries. Routes self-register with metadata, enabling reliable OpenAPI generation and clean parameter extraction across all routers. Each router instance owns its registry, avoiding global state and enabling isolated testing.

## Problems with Current Design

### 1. AST Parsing Fragility
- Can't handle dynamic route registration
- Breaks with variable aliasing or method chaining
- Can't resolve cross-package imports
- No support for runtime configuration

### 2. Complex Adapter Architecture
- Union types for query parameters are over-engineered
- Echo adapter creates nested API instances
- Context propagation adds unnecessary complexity
- Factory pattern creates adapters per request

### 3. Missing Runtime Information
- Can't access actual registered routes
- No validation of handler signatures
- Can't determine actual parameter types
- No integration with router middleware

### 4. Global State Anti-Patterns
- Shared global registries prevent isolated testing
- Multiple API instances conflict
- Hidden dependencies between components

## Core Concepts

### 1. Registry Ownership

Instead of a global registry, each router wrapper owns its registry:

```go
// pkg/api/registry.go
package api

import (
    "reflect"
    "sync"
)

// RouteInfo contains complete route metadata
type RouteInfo struct {
    Method      string
    Path        string
    Handler     interface{}
    HandlerName string
    RequestType reflect.Type
    ResponseType reflect.Type
    Options     HandlerOptions
    Middleware  []string
}

// RouteRegistry stores all registered routes
type RouteRegistry struct {
    mu     sync.RWMutex
    routes []*RouteInfo
}

// NewRouteRegistry creates a new registry instance
func NewRouteRegistry() *RouteRegistry {
    return &RouteRegistry{
        routes: make([]*RouteInfo, 0),
    }
}

// Register adds a route to the registry
func (r *RouteRegistry) Register(info *RouteInfo) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.routes = append(r.routes, info)
}

// GetRoutes returns all registered routes
func (r *RouteRegistry) GetRoutes() []*RouteInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return append([]*RouteInfo{}, r.routes...)
}
```

### 2. Router Interface with Registry Access

```go
// pkg/api/router.go
package api

// Router is the common interface that all router adapters must implement
type Router interface {
    // HTTP method registrations
    Get(path string, handler interface{}, opts ...Option)
    Post(path string, handler interface{}, opts ...Option)
    Put(path string, handler interface{}, opts ...Option)
    Delete(path string, handler interface{}, opts ...Option)
    Patch(path string, handler interface{}, opts ...Option)
    
    // Group creates a sub-router with the given prefix
    Group(prefix string, fn ...interface{}) Router
    
    // GetRegistry returns this router's registry
    GetRegistry() *RouteRegistry
    
    // Unwrap returns the underlying router implementation
    Unwrap() interface{}
}
```

### 3. Updated TypedRouter Base

```go
// TypedRouter provides generic type-safe methods
type TypedRouter[T any] struct {
    underlying     T
    registry       *RouteRegistry  // Instance-specific registry
    adapterFactory AdapterFactory
    prefix         string
    middleware     []Option
    registerFn     func(method, path string, handler http.HandlerFunc, info *RouteInfo)
}

// GetRegistry returns this router's registry
func (r *TypedRouter[T]) GetRegistry() *RouteRegistry {
    return r.registry
}

// register is the internal method that handles route registration
func (r *TypedRouter[T]) register[Req any, Resp any](
    method, path string,
    handler func(context.Context, Req) (Resp, error),
    opts ...Option,
) {
    allOpts := append([]Option{}, r.middleware...)
    allOpts = append(allOpts, opts...)
    httpHandler, info := CreateHandler(r.adapterFactory, handler, allOpts...)
    
    info.Method = method
    info.Path = r.prefix + path
    
    // Register in this router's registry, not a global one
    r.registry.Register(info)
    
    r.registerFn(method, path, httpHandler, info)
}
```

### 4. Router Adapter Implementations

#### Standard Library Example

```go
// pkg/api/adapters/stdlib/router.go
package stdlib

import (
    "context"
    "net/http"
    "github.com/gork-labs/gork/pkg/api"
)

// Router wraps http.ServeMux with typed methods
type Router struct {
    api.TypedRouter[*http.ServeMux]
    mux      *http.ServeMux
    registry *api.RouteRegistry
}

// NewRouter creates a new stdlib router wrapper
func NewRouter(mux *http.ServeMux, opts ...api.Option) *Router {
    registry := api.NewRouteRegistry()
    r := &Router{
        mux:      mux,
        registry: registry,
    }
    
    r.TypedRouter = api.TypedRouter[*http.ServeMux]{
        underlying:     mux,
        registry:       registry,  // Pass the registry instance
        adapterFactory: NewAdapter,
        middleware:     opts,
        registerFn: func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
            pattern := method + " " + r.prefix + path
            mux.HandleFunc(pattern, handler)
        },
    }
    
    return r
}

// Group creates a sub-router with a path prefix
// Sub-routers share the same registry as their parent
func (r *Router) Group(prefix string) api.Router {
    return &Router{
        mux:      r.mux,
        registry: r.registry,  // Share parent's registry
        TypedRouter: api.TypedRouter[*http.ServeMux]{
            underlying:     r.mux,
            registry:       r.registry,  // Share parent's registry
            adapterFactory: r.adapterFactory,
            prefix:         r.prefix + prefix,
            middleware:     r.middleware,
            registerFn:     r.registerFn,
        },
    }
}
```

#### Echo Example with Shared Registry

```go
// pkg/api/adapters/echo/router.go
package echo

// Router wraps echo.Echo with typed methods
type Router struct {
    echo       *echo.Echo
    group      *echo.Group
    registry   *api.RouteRegistry
    prefix     string
    middleware []api.Option
}

// NewRouter creates a new Echo router wrapper with its own registry
func NewRouter(e *echo.Echo, opts ...api.Option) *Router {
    return &Router{
        echo:       e,
        registry:   api.NewRouteRegistry(),
        middleware: opts,
    }
}

// NewGroup creates a new Echo group wrapper that shares the registry
func NewGroup(parent *Router, g *echo.Group, opts ...api.Option) *Router {
    return &Router{
        echo:       parent.echo,
        group:      g,
        registry:   parent.registry,  // Share parent's registry
        prefix:     g.Prefix(),
        middleware: opts,
    }
}

// GetRegistry returns this router's registry
func (r *Router) GetRegistry() *api.RouteRegistry {
    return r.registry
}

// Group creates a sub-router with prefix
func (r *Router) Group(prefix string, middleware ...echo.MiddlewareFunc) *Router {
    var g *echo.Group
    if r.group != nil {
        g = r.group.Group(prefix, middleware...)
    } else {
        g = r.echo.Group(prefix, middleware...)
    }
    
    return NewGroup(r, g, r.middleware...)
}
```

### 5. Union Type Support

Union types remain a core feature, enabling type-safe handling of multiple possible types. The current implementation uses a field-based approach that allows direct type switching:

```go
// pkg/unions/unions.go
package unions

// Union2 represents a value that can be one of two types
type Union2[A, B any] struct {
    A *A
    B *B
}

// Union3 represents a value that can be one of three types
type Union3[A, B, C any] struct {
    A *A
    B *B
    C *C
}

// Union4 represents a value that can be one of four types
type Union4[A, B, C, D any] struct {
    A *A
    B *B
    C *C
    D *D
}

// Value returns the active value and its type index (0-based)
func (u Union2[A, B]) Value() (interface{}, int) {
    switch {
    case u.A != nil:
        return u.A, 0
    case u.B != nil:
        return u.B, 1
    default:
        return nil, -1
    }
}
```

Example usage in handlers:

```go
// PaymentMethod can be either CreditCard or BankTransfer
type PaymentRequest struct {
    Amount  float64                                       `json:"amount" validate:"required,min=0"`
    Method  unions.Union2[CreditCard, BankTransfer]     `json:"method" validate:"required"`
    UserID  string                                        `json:"user_id" validate:"required,uuid"`
}

// Handler usage with type switch
func ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
    value, _ := req.Method.Value()
    switch v := value.(type) {
    case *CreditCard:
        // Process credit card payment
        return processCreditCard(v)
    case *BankTransfer:
        // Process bank transfer
        return processBankTransfer(v)
    default:
        return nil, errors.New("invalid payment method")
    }
}

// LoginMethod supports multiple authentication types
type LoginRequest struct {
    Method unions.Union4[EmailLogin, PhoneLogin, OAuthLogin, BiometricLogin] `json:"method" validate:"required"`
}

// Response can vary based on request
type FlexibleResponse unions.Union3[UserData, ErrorDetails, RedirectInfo]
```

The unions use validation-based unmarshaling - they try each type in order and use the first one that successfully unmarshals AND passes validation. This ensures correct type detection and supports two patterns:

1. **Structural discrimination**: Types are distinguished by their structure and required fields
2. **Explicit discrimination**: Types include a discriminator field with validation constraints

Example with explicit discriminators:
```go
type PaymentType string

const (
    CreditCard   PaymentType = "cc"
    BankTransfer PaymentType = "bank-transfer"
)

type CreditCardPayment struct {
    Type   PaymentType `json:"type" openapi:"discriminator=cc" validate:"required,eq=cc"`
    Number string      `json:"number" validate:"required,creditcard"`
    CVV    string      `json:"cvv" validate:"required,len=3"`
    Expiry string      `json:"expiry" validate:"required"`
}

type BankTransferPayment struct {
    Type          PaymentType `json:"type" openapi:"discriminator=bank-transfer" validate:"required,eq=bank-transfer"`
    AccountNumber string      `json:"account_number" validate:"required"`
    RoutingNumber string      `json:"routing_number" validate:"required"`
    BankName      string      `json:"bank_name" validate:"required"`
}

// Direct usage as request type
type UpdatePaymentMethodReq unions.Union2[BankTransferPayment, CreditCardPayment]

// Handler receives the union directly
func UpdatePaymentMethod(ctx context.Context, req UpdatePaymentMethodReq) (*PaymentResponse, error) {
    value, _ := req.Value()
    switch v := value.(type) {
    case *CreditCardPayment:
        // v.Type == "cc"
        return processCreditCard(v)
    case *BankTransferPayment:
        // v.Type == "bank-transfer"
        return processBankTransfer(v)
    }
    return nil, errors.New("invalid payment method")
}
```

The `openapi:"discriminator=value"` tag explicitly marks discriminator fields for OpenAPI generation, keeping OpenAPI concerns separate from validation logic.

With this pattern, the validation ensures only the correct type unmarshals successfully, providing explicit control over type discrimination.

### 6. OpenAPI Generation with Union Support

The OpenAPI generator properly handles union types as oneOf schemas:

```go
// pkg/api/openapi.go
package api

import (
    "reflect"
    "github.com/gork-labs/gork/pkg/unions"
)

// GenerateOpenAPI creates an OpenAPI spec from a specific registry
func GenerateOpenAPI(registry *RouteRegistry, options ...OpenAPIOption) *OpenAPISpec {
    routes := registry.GetRoutes()
    
    spec := &OpenAPISpec{
        OpenAPI: "3.1.0",
        Info: Info{
            Title:   "Generated API",
            Version: "1.0.0",
        },
        Paths: make(map[string]*PathItem),
        Components: &Components{
            Schemas:         make(map[string]*Schema),
            SecuritySchemes: make(map[string]*SecurityScheme),
        },
    }
    
    // Apply options
    for _, opt := range options {
        opt(spec)
    }
    
    // Union schema registry for tracking discriminators
    unionRegistry := NewUnionSchemaRegistry()
    
    // Generate from routes...
    for _, route := range routes {
        path := convertToOpenAPIPath(route.Path)
        
        if spec.Paths[path] == nil {
            spec.Paths[path] = &PathItem{}
        }
        
        operation := &Operation{
            OperationID: route.HandlerName,
            Tags:        route.Options.Tags,
            Security:    convertSecurity(route.Options.Security),
            Parameters:  extractParameters(route.RequestType),
        }
        
        // Add request body for POST/PUT/PATCH
        if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
            schema := generateSchema(route.RequestType, spec.Components.Schemas, unionRegistry)
            operation.RequestBody = &RequestBody{
                Required: true,
                Content: map[string]*MediaType{
                    "application/json": {Schema: schema},
                },
            }
        }
        
        // Add response with union support
        responseSchema := generateSchema(route.ResponseType, spec.Components.Schemas, unionRegistry)
        operation.Responses = map[string]*Response{
            "200": {
                Description: "Success",
                Content: map[string]*MediaType{
                    "application/json": {Schema: responseSchema},
                },
            },
        }
        
        // Set operation on path
        setOperation(spec.Paths[path], route.Method, operation)
    }
    
    return spec
}

// generateSchema creates schemas with proper union type handling
func generateSchema(t reflect.Type, schemas map[string]*Schema, unionRegistry *UnionSchemaRegistry) *Schema {
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    
    // Check if this is a union type
    if isUnionType(t) {
        return generateUnionSchema(t, schemas, unionRegistry)
    }
    
    // Regular schema generation...
    typeName := t.Name()
    if typeName != "" && schemas[typeName] != nil {
        return &Schema{Ref: "#/components/schemas/" + typeName}
    }
    
    schema := &Schema{
        Type:       "object",
        Properties: make(map[string]*Schema),
        Required:   []string{},
    }
    
    // Process fields
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if field.PkgPath != "" { // Skip unexported
            continue
        }
        
        fieldSchema := generateSchema(field.Type, schemas, unionRegistry)
        // ... field processing
        schema.Properties[jsonFieldName(field)] = fieldSchema
    }
    
    if typeName != "" {
        schemas[typeName] = schema
        return &Schema{Ref: "#/components/schemas/" + typeName}
    }
    
    return schema
}

// generateUnionSchema creates oneOf schemas for union types
func generateUnionSchema(t reflect.Type, schemas map[string]*Schema, registry *UnionSchemaRegistry) *Schema {
    unionInfo := extractUnionTypes(t)
    
    oneOfSchemas := make([]*Schema, len(unionInfo.Types))
    for i, variantType := range unionInfo.Types {
        oneOfSchemas[i] = generateSchema(variantType, schemas, registry)
    }
    
    schema := &Schema{
        OneOf: oneOfSchemas,
    }
    
    // Check if types have explicit discriminator fields
    discriminator := detectDiscriminator(unionInfo.Types)
    if discriminator != nil {
        schema.Discriminator = discriminator
    }
    
    return schema
}

// detectDiscriminator checks if union types have a common discriminator field
func detectDiscriminator(types []reflect.Type) *Discriminator {
    if len(types) < 2 {
        return nil
    }
    
    // Check for discriminator fields marked with openapi tag
    var discriminatorField string
    var mapping = make(map[string]string)
    
    for i, t := range types {
        if t.Kind() == reflect.Ptr {
            t = t.Elem()
        }
        
        for j := 0; j < t.NumField(); j++ {
            field := t.Field(j)
            openapiTag := field.Tag.Get("openapi")
            
            // Look for discriminator= in openapi tag
            if strings.HasPrefix(openapiTag, "discriminator=") {
                if i == 0 {
                    discriminatorField = field.Tag.Get("json")
                    if discriminatorField == "" {
                        discriminatorField = field.Name
                    }
                }
                
                // Extract the discriminator value
                parts := strings.Split(openapiTag, ",")
                for _, part := range parts {
                    if strings.HasPrefix(part, "discriminator=") {
                        value := strings.TrimPrefix(part, "discriminator=")
                        mapping[value] = "#/components/schemas/" + t.Name()
                        break
                    }
                }
                break
            }
        }
    }
    
    // Only return discriminator if all types have the same field with discriminator values
    if len(mapping) == len(types) && discriminatorField != "" {
        return &Discriminator{
            PropertyName: discriminatorField,
            Mapping:      mapping,
        }
    }
    
    return nil
}

// isUnionType checks if a type is Union2, Union3, or Union4
func isUnionType(t reflect.Type) bool {
    if t.PkgPath() != "github.com/gork-labs/gork/pkg/unions" {
        return false
    }
    return t.Name() == "Union2" || t.Name() == "Union3" || t.Name() == "Union4"
}

// UnionSchemaRegistry tracks union types for OpenAPI generation
type UnionSchemaRegistry struct {
    unions map[reflect.Type][]reflect.Type
}

func NewUnionSchemaRegistry() *UnionSchemaRegistry {
    return &UnionSchemaRegistry{
        unions: make(map[reflect.Type][]reflect.Type),
    }
}

func (r *UnionSchemaRegistry) RegisterUnion(t reflect.Type, variants []reflect.Type) {
    r.unions[t] = variants
}

// OpenAPIOption allows customizing the generated spec
type OpenAPIOption func(*OpenAPISpec)

// WithTitle sets the API title
func WithTitle(title string) OpenAPIOption {
    return func(spec *OpenAPISpec) {
        spec.Info.Title = title
    }
}

// WithVersion sets the API version
func WithVersion(version string) OpenAPIOption {
    return func(spec *OpenAPISpec) {
        spec.Info.Version = version
    }
}
```

### 7. Usage Examples

#### Single API Instance with Union Types

```go
// Types with union support
type PaymentMethod unions.Union2[CreditCard, BankTransfer]

type CreditCard struct {
    Number  string `json:"number" validate:"required,creditcard"`
    CVV     string `json:"cvv" validate:"required,len=3"`
    Expiry  string `json:"expiry" validate:"required"`
}

type BankTransfer struct {
    AccountNumber string `json:"account_number" validate:"required"`
    RoutingNumber string `json:"routing_number" validate:"required"`
    BankName      string `json:"bank_name" validate:"required"`
}

type ProcessPaymentRequest struct {
    Amount   float64        `json:"amount" validate:"required,min=0"`
    Method   PaymentMethod  `json:"payment_method" validate:"required"`
    Currency string         `json:"currency" validate:"required,oneof=USD EUR GBP"`
}

type PaymentResponse unions.Union3[PaymentSuccess, PaymentPending, PaymentError]

func ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (PaymentResponse, error) {
    // Implementation that returns one of the union variants
}

func main() {
    mux := http.NewServeMux()
    
    // Create router wrapper with its own registry
    router := stdlib.NewRouter(mux)
    
    // Register routes with union types
    router.Post("/payments", ProcessPayment)
    router.Get("/users/{id}", GetUser)
    router.Post("/users", CreateUser)
    
    // Generate OpenAPI from this router's registry
    // Union types will be properly represented as oneOf schemas
    mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
        spec := api.GenerateOpenAPI(
            router.GetRegistry(),
            api.WithTitle("Payment API"),
            api.WithVersion("1.0.0"),
        )
        json.NewEncoder(w).Encode(spec)
    })
    
    http.ListenAndServe(":8080", mux)
}
```

#### Multiple Independent APIs

```go
func main() {
    // First API
    userMux := http.NewServeMux()
    userRouter := stdlib.NewRouter(userMux)
    userRouter.Get("/users/{id}", GetUser)
    userRouter.Post("/users", CreateUser)
    
    // Second API
    productMux := http.NewServeMux()
    productRouter := stdlib.NewRouter(productMux)
    productRouter.Get("/products/{id}", GetProduct)
    productRouter.Post("/products", CreateProduct)
    
    // Each has its own OpenAPI spec
    userMux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
        spec := api.GenerateOpenAPI(userRouter.GetRegistry())
        json.NewEncoder(w).Encode(spec)
    })
    
    productMux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
        spec := api.GenerateOpenAPI(productRouter.GetRegistry())
        json.NewEncoder(w).Encode(spec)
    })
    
    // Run both APIs
    go http.ListenAndServe(":8080", userMux)
    http.ListenAndServe(":8081", productMux)
}
```

#### Testing with Isolated Registry and Union Types

```go
func TestPaymentAPI(t *testing.T) {
    // Create isolated router for testing
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux)
    
    // Register test routes with union types
    router.Post("/payments", ProcessPayment)
    router.Get("/payment-methods", ListPaymentMethods)
    
    // Test OpenAPI generation with union schemas
    spec := api.GenerateOpenAPI(router.GetRegistry())
    assert.Equal(t, 2, len(spec.Paths))
    
    // Verify union schema generation
    paymentSchema := spec.Components.Schemas["ProcessPaymentRequest"]
    methodSchema := paymentSchema.Properties["payment_method"]
    assert.NotNil(t, methodSchema.OneOf)
    assert.Equal(t, 2, len(methodSchema.OneOf)) // CreditCard and BankTransfer
    
    // Registry is isolated - won't affect other tests
}

func TestAuthAPI(t *testing.T) {
    // Completely separate registry
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux)
    
    // LoginMethod is Union4[EmailLogin, PhoneLogin, OAuthLogin, BiometricLogin]
    router.Post("/auth/login", Login)
    
    // This test has its own isolated registry
    spec := api.GenerateOpenAPI(router.GetRegistry())
    assert.Equal(t, 1, len(spec.Paths))
    
    // Verify Union4 handling
    loginSchema := spec.Components.Schemas["LoginRequest"]
    methodSchema := loginSchema.Properties["method"]
    assert.Equal(t, 4, len(methodSchema.OneOf))
}
```

### 8. Documentation Extraction Integration

The documentation extractor can be passed as a dependency:

```go
// OpenAPIGenerator with documentation support
type OpenAPIGenerator struct {
    docs *docs.TypeDocs
}

// NewOpenAPIGenerator creates a generator with documentation
func NewOpenAPIGenerator(packages []string) (*OpenAPIGenerator, error) {
    typeDocs, err := docs.ExtractDocs(packages)
    if err != nil {
        return nil, err
    }
    
    return &OpenAPIGenerator{docs: typeDocs}, nil
}

// GenerateOpenAPI creates an OpenAPI spec from a registry with docs
func (g *OpenAPIGenerator) GenerateOpenAPI(registry *RouteRegistry) *OpenAPISpec {
    routes := registry.GetRoutes()
    // ... generation with documentation
}
```

Usage:

```go
func main() {
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux)
    
    // Register routes
    router.Post("/users", CreateUser)
    router.Get("/users/{id}", GetUser)
    
    // Create generator with docs
    generator, _ := api.NewOpenAPIGenerator([]string{
        "./",
        "./models",
    })
    
    // Serve OpenAPI with documentation
    mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
        spec := generator.GenerateOpenAPI(router.GetRegistry())
        spec.Info.Title = "User Management API"
        json.NewEncoder(w).Encode(spec)
    })
    
    http.ListenAndServe(":8080", mux)
}
```

## Benefits of This Approach

### 1. **No Global State**
- Each router has its own registry
- Multiple independent APIs can coexist
- No hidden dependencies

### 2. **Testability**
- Tests run in complete isolation
- No need to reset global state between tests
- Can create lightweight test routers

### 3. **Flexibility**
- Can have multiple registries for different purposes
- Easy to create specialized OpenAPI specs
- Registries can be composed or filtered

### 4. **Clear Ownership**
- Registry lifecycle tied to router lifecycle
- No ambiguity about where routes are registered
- Sub-routers naturally share parent's registry

### 5. **Dependency Injection**
- Registry passed explicitly where needed
- Easy to mock or substitute for testing
- Clear data flow

## Example Generated OpenAPI Output

With union types and documentation extraction, the generated OpenAPI spec includes rich schemas:

```yaml
openapi: "3.1.0"
info:
  title: "Payment API"
  version: "1.0.0"

paths:
  /payments:
    post:
      operationId: "ProcessPayment"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ProcessPaymentRequest"
      responses:
        "200":
          description: "Success"
          content:
            application/json:
              schema:
                oneOf:
                  - $ref: "#/components/schemas/PaymentSuccess"
                  - $ref: "#/components/schemas/PaymentPending"
                  - $ref: "#/components/schemas/PaymentError"

components:
  schemas:
    ProcessPaymentRequest:
      type: "object"
      required: ["amount", "payment_method", "currency"]
      properties:
        amount:
          type: "number"
          minimum: 0
        payment_method:
          oneOf:
            - $ref: "#/components/schemas/CreditCard"
            - $ref: "#/components/schemas/BankTransfer"
          # No discriminator - union uses validation-based unmarshaling
        currency:
          type: "string"
          enum: ["USD", "EUR", "GBP"]
    
    CreditCard:
      type: "object"
      required: ["number", "cvv", "expiry"]
      properties:
        number:
          type: "string"
          format: "creditcard"
        cvv:
          type: "string"
          minLength: 3
          maxLength: 3
        expiry:
          type: "string"
    
    BankTransfer:
      type: "object"
      required: ["account_number", "routing_number", "bank_name"]
      properties:
        account_number:
          type: "string"
        routing_number:
          type: "string"
        bank_name:
          type: "string"
```

## Migration Strategy

### Phase 1: Add Runtime Registration
1. Add `RouteRegistry` to existing codebase
2. Update router wrappers to own registries
3. Keep existing AST-based generation working

### Phase 2: Implement Documentation Extraction
1. Create `docs` package for AST parsing of comments
2. Extract type and field documentation from Go source
3. Test documentation extraction independently

### Phase 3: Implement Adapters
1. Create simplified `ParameterAdapter` interface
2. Implement adapters for each router
3. Refactor parameter extraction to use adapters

### Phase 4: Switch OpenAPI Generation
1. Implement runtime-based OpenAPI generation with union support
2. Integrate documentation extraction with route metadata
3. Add `/openapi.json` endpoint to examples
4. Deprecate AST-based route detection

### Phase 5: Clean Up
1. Remove AST-based route detection code
2. Remove complex union types for queries
3. Simplify adapter creation
4. Keep only documentation-focused AST parsing

## Key Improvements

### 1. **Router Wrappers Eliminate Repetition**
Instead of passing method and path to both the router and api.HandlerFunc:
```go
// Old - Repetitive
mux.HandleFunc("GET /users/{id}", api.HandlerFunc("GET", "/users/{id}", GetUser))

// New - Clean
router.Get("/users/{id}", GetUser)
```

### 2. **Operation ID from Function Names**
The `getFunctionName` utility extracts operation IDs at runtime using reflection:
```go
handlerName := getFunctionName(handler) // Returns "GetUser"
```
No AST parsing or magic comments needed. The function name IS the operation ID.

### 3. **Type-Safe Router Methods**
Each router wrapper provides typed HTTP methods that:
- Register routes with the router
- Store metadata in the instance's registry
- Extract the operation ID from the handler function
- Support router-specific features naturally

### 4. **Direct Access to Underlying Routers**
The `Unwrap()` method provides escape hatches for router-specific features:
```go
// Access underlying router for custom configuration
router.Unwrap().Use(customMiddleware)

// Echo-specific methods for different underlying types
if group, ok := router.AsGroup(); ok {
    group.Use(groupMiddleware)
}
```

### 5. **Union Types as First-Class Citizens**
Union types (Union2/Union3/Union4) are fully supported throughout:
- Field-based storage allows direct type switching via `Value()` method
- Validation-based unmarshaling without requiring explicit discriminators
- OpenAPI generation creates oneOf schemas without discriminators
- Type safety maintained with compile-time guarantees
- Simple usage pattern: `switch v := union.Value().(type) { case *TypeA: ... }`

## Comparison with Current Design

| Feature | Current Design | New Design |
|---------|---------------|------------|
| Route Detection | AST parsing | Runtime registration |
| Parameter Extraction | Complex adapters with unions | Simple string getters |
| OpenAPI Generation | Parse source files | Use registered metadata |
| Router Support | Requires AST patterns | Just implement adapter |
| Type Safety | Lost in AST parsing | Preserved via generics |
| Dynamic Routes | Not supported | Fully supported |
| Middleware | Not integrated | First-class support |
| Performance | AST parsing overhead | Minimal runtime overhead |
| Router Access | Indirect only | Direct via Unwrap() |
| Global State | Shared registries | Instance-based registries |
| Union Types | Limited support | Full support with validation-based unmarshaling |

## Unified API Across All Routers

The beauty of this design is that all routers follow the same pattern:

```go
// Standard Library
router := stdlib.NewRouter(mux)
router.Get("/users/{id}", GetUser)
router.Post("/users", CreateUser)

// Chi
router := chirouter.NewRouter(r)
router.Get("/users/{id}", GetUser)
router.Post("/users", CreateUser)

// Echo
router := echorouter.NewRouter(e)
router.Get("/users/:id", GetUser)
router.Post("/users", CreateUser)

// Gin
router := ginrouter.NewRouter(g)
router.Get("/users/:id", GetUser)
router.Post("/users", CreateUser)

// Gorilla
router := gorillarouter.NewRouter(r)
router.Get("/users/{id}", GetUser)
router.Post("/users", CreateUser)

// All routers support:
// - Direct access: router.Unwrap()
// - Registry access: router.GetRegistry()
// - Groups with shared registry
// - Union types in requests/responses
```

## Implementation TODO

Since we're not maintaining backward compatibility, we can make a clean break and implement this design from scratch:

### 1. Core API Package (`pkg/api`)
- [ ] Create `Router` interface with standard HTTP methods and `Group()`
- [ ] Create `RouteRegistry` struct with thread-safe route storage
- [ ] Implement `RouteInfo` struct with all metadata fields
- [ ] Create `ParameterAdapter` interface with Query, Path, Header, Cookie methods
- [ ] Implement `TypedRouter[T]` generic base type with shared logic
- [ ] Implement `createHandler` function for HTTP handler generation
- [ ] Add parameter extraction logic using reflection and struct tags
- [ ] Create handler options (WithTags, WithBearerAuth, WithAPIKey, etc.)
- [ ] Add `NoContentResponse` type for 204 responses
- [ ] Implement `getFunctionName` utility using runtime reflection
- [ ] Create context key management for router-specific contexts

### 2. Union Type Support (`pkg/unions`)
- [x] Implement `Union2[A, B]` generic type with field-based storage
- [x] Implement `Union3[A, B, C]` generic type with field-based storage
- [x] Implement `Union4[A, B, C, D]` generic type with field-based storage
- [x] Add JSON marshaling/unmarshaling with validation-based discrimination
- [x] Implement Value() accessor method returning (interface{}, int)
- [x] Add validation support using go-playground/validator
- [ ] Add helper methods for checking active variant (IsA(), IsB(), etc.)
- [ ] Create comprehensive unit tests for edge cases
- [ ] Document the validation-based unmarshaling approach

### 3. Documentation Extraction (`pkg/api/docs`)
- [ ] Create AST parser for Go source files
- [ ] Extract type-level documentation comments
- [ ] Extract field-level documentation comments
- [ ] Extract handler function documentation
- [ ] Handle nested types and embedded structs
- [ ] Support multi-file packages
- [ ] Cache parsed documentation for performance
- [ ] Handle generic types in documentation
- [ ] Extract enum values from const blocks
- [ ] Support example extraction from comments

### 4. Router Adapters (`pkg/api/adapters/*`)

#### Standard Library (`pkg/api/adapters/stdlib`)
- [ ] Implement `Adapter` for parameter extraction
- [ ] Support Go 1.22+ path parameters via `PathValue()`
- [ ] Create `Router` wrapper with registry ownership
- [ ] Implement `NewRouter` constructor
- [ ] Implement all HTTP methods (Get, Post, Put, Delete, Patch)
- [ ] Implement `Group()` method with prefix concatenation
- [ ] Add `GetRegistry()` method
- [ ] Add `Unwrap()` method for underlying access
- [ ] Handle method+path pattern for Go 1.22+
- [ ] Write comprehensive tests

#### Chi (`pkg/api/adapters/chi`)
- [ ] Implement `Adapter` with Chi context support
- [ ] Extract path params via `chi.URLParamFromCtx()`
- [ ] Create `Router` wrapper using `TypedRouter`
- [ ] Implement `NewRouter` constructor
- [ ] Implement `Group()` using Chi's `Route()` method
- [ ] Handle Chi's middleware integration
- [ ] Store Chi context in request context
- [ ] No path conversion needed (Chi uses `{param}` natively)
- [ ] Add support for Chi's regex patterns
- [ ] Write comprehensive tests

#### Echo (`pkg/api/adapters/echo`)
- [ ] Implement `Adapter` for Echo context
- [ ] Create `Router` wrapper with dual-type support (Echo/Group)
- [ ] Implement `NewRouter` for Echo instances
- [ ] Implement `NewGroup` for Echo groups
- [ ] Add `UnwrapEcho()`, `UnwrapGroup()` methods
- [ ] Add `AsEcho()`, `AsGroup()` helper methods
- [ ] Implement `Group()` for both Echo and Group types
- [ ] Implement `toNativePath()` to convert `{param}` → `:param`
- [ ] Store Echo context properly in request
- [ ] Handle Echo's middleware at group level
- [ ] Write comprehensive tests

#### Gin (`pkg/api/adapters/gin`)
- [ ] Implement `Adapter` for Gin context
- [ ] Create `Router` wrapper with dual-type support (Engine/RouterGroup)
- [ ] Implement `NewRouter` for Gin engine
- [ ] Implement `NewGroup` for Gin router groups
- [ ] Add `UnwrapGin()`, `UnwrapGroup()` methods
- [ ] Add `AsGin()`, `AsGroup()` helper methods
- [ ] Implement `Group()` for both Engine and RouterGroup
- [ ] Implement `toNativePath()` to convert `{param}` → `:param`
- [ ] Store Gin context in request context
- [ ] Handle Gin's middleware chains
- [ ] Write comprehensive tests

#### Gorilla Mux (`pkg/api/adapters/gorilla`)
- [ ] Implement `Adapter` with `mux.Vars()` support
- [ ] Create `Router` wrapper using `TypedRouter`
- [ ] Implement `NewRouter` constructor
- [ ] Implement `Group()` using `PathPrefix().Subrouter()`
- [ ] No path conversion needed (Gorilla uses `{param}` natively)
- [ ] Handle Gorilla's regex constraints
- [ ] Support Gorilla's host and scheme matching
- [ ] Add query parameter constraints support
- [ ] Write comprehensive tests

### 5. OpenAPI Generation (`pkg/api/openapi`)
- [ ] Create `OpenAPIGenerator` struct with docs support
- [ ] Implement `GenerateOpenAPI()` using runtime registry
- [ ] Create `UnionSchemaRegistry` for discriminator tracking
- [ ] Implement `generateSchema` with recursive type handling
- [ ] Add `generateUnionSchema` for oneOf generation
- [ ] Implement `isUnionType` detection
- [ ] Extract parameters from struct tags (openapi, json, validate)
- [ ] Map validator tags to OpenAPI constraints
  - [ ] required → required field
  - [ ] email → format: email
  - [ ] min/max → minLength/maxLength or minimum/maximum
  - [ ] oneof → enum values
  - [ ] uuid → format: uuid
  - [ ] url → format: uri
  - [ ] datetime → format: date-time
- [ ] Handle union types with proper discriminators
- [ ] Generate security schemes from route options
- [ ] Add support for custom formats and patterns
- [ ] Handle nullable fields and optional parameters
- [ ] Support array and object validation
- [ ] Generate example values from struct tags
- [ ] Add response status code customization
- [ ] Support OpenAPI extensions (x-* fields)

### 6. Parameter Extraction (`pkg/api/params`)
- [ ] Create parameter extractor using reflection
- [ ] Support json tags for field naming
- [ ] Support openapi tags for parameter location (path, query, header, cookie)
- [ ] Handle array parameters in query strings
- [ ] Support custom parameter parsers
- [ ] Add validation using validate tags
- [ ] Handle nested structs for body parameters
- [ ] Support file upload parameters
- [ ] Add comprehensive error messages
- [ ] Cache reflection data for performance

### 7. Utilities (`pkg/api/utils`)
- [ ] Path parser that identifies parameter segments
  - [ ] Parse {param} and :param formats
  - [ ] Extract parameter names and positions
  - [ ] Handle escaped special characters
  - [ ] Support wildcard segments
- [ ] Path converter between router formats
  - [ ] Convert {param} ↔ :param
  - [ ] Handle regex constraints
  - [ ] Preserve path structure
- [ ] Validator tag parser
  - [ ] Parse comma-separated constraints
  - [ ] Extract constraint parameters
  - [ ] Handle custom validators
- [ ] OpenAPI tag parser
  - [ ] Parse name and location
  - [ ] Support additional attributes
- [ ] Type name extraction utilities
- [ ] JSON field name resolver
- [ ] String case converters (camelCase, snake_case)

### 8. Testing
- [ ] Unit tests for RouteRegistry operations
- [ ] Unit tests for each adapter implementation
- [ ] Integration tests for all router wrappers
- [ ] Test registry isolation between instances
- [ ] Test group/sub-router registry sharing
- [ ] Documentation extraction tests with edge cases
- [ ] OpenAPI generation tests for all field types
- [ ] Union type serialization/deserialization tests
- [ ] Test discriminator generation for unions
- [ ] Parameter extraction tests for all adapters
- [ ] End-to-end tests with example APIs
- [ ] Benchmark tests for performance validation
- [ ] Concurrent access tests for thread safety
- [ ] Test middleware integration for each router
- [ ] Test error handling and validation

### 9. Examples (`examples/`)
- [ ] Create base types for all examples
  - [ ] User management types
  - [ ] Payment types with unions
  - [ ] Authentication types with Union4
  - [ ] Product catalog types
- [ ] Example API using stdlib router with unions
- [ ] Example API using Chi with groups
- [ ] Example API using Echo with middleware
- [ ] Example API using Gin with groups
- [ ] Example API using Gorilla with regex
- [ ] Complex example with nested unions
- [ ] Example with custom validators
- [ ] Example with file uploads
- [ ] Middleware integration examples
- [ ] Group/subrouter usage examples
- [ ] Multi-version API example
- [ ] Swagger UI integration
- [ ] ReDoc integration
- [ ] Example with API key auth
- [ ] Example with JWT bearer auth

### 10. Documentation
- [ ] Write comprehensive README.md
  - [ ] Quick start guide
  - [ ] Supported routers
  - [ ] Union type usage
  - [ ] OpenAPI generation
- [ ] Document Router interface contract
- [ ] Create adapter implementation guide
- [ ] Document parameter extraction
- [ ] Document union type patterns
- [ ] Create troubleshooting guide
- [ ] Document struct tag reference
- [ ] Add performance considerations
- [ ] Create API reference documentation
- [ ] Document testing strategies
- [ ] Add FAQ section
- [ ] Create video tutorials
- [ ] Write blog post about the design

### 11. CLI Updates (`cmd/openapi-gen`)
- [ ] Remove all AST-based route detection code
- [ ] Add runtime server mode for route discovery
- [ ] Update to use new generator with docs
- [ ] Add flag for documentation package paths
- [ ] Support multiple output formats (JSON, YAML)
- [ ] Add validation mode to check specs
- [ ] Support OpenAPI 3.0 and 3.1 output
- [ ] Add watch mode for development
- [ ] Support config file for complex setups
- [ ] Add init command to bootstrap projects
- [ ] Remove all backward compatibility code

### 12. Release Preparation
- [ ] Create migration guide from current version
- [ ] Document breaking changes
- [ ] Create compatibility matrix for routers
- [ ] Set up GitHub Actions for CI/CD
- [ ] Configure code coverage reporting
- [ ] Set up automated releases
- [ ] Create changelog
- [ ] Update all dependencies
- [ ] Security audit of dependencies
- [ ] Performance profiling
- [ ] Create benchmarks vs old version
- [ ] Plan deprecation timeline

### 13. Future Enhancements (Post-Release)
- [ ] Support for more routers (Fiber, Buffalo, etc.)
- [ ] GraphQL schema generation
- [ ] gRPC gateway integration
- [ ] Postman collection export
- [ ] Insomnia workspace export
- [ ] Client SDK generation
- [ ] Mock server generation
- [ ] Contract testing support
- [ ] Webhook documentation
- [ ] WebSocket documentation
- [ ] Server-sent events support
- [ ] OpenAPI 3.2 support when released

## Conclusion

This design eliminates the fragility of AST parsing while providing a cleaner, more maintainable architecture. The combination of runtime registration, instance-based registries, and first-class union support creates a powerful system for type-safe API development with automatic OpenAPI generation.