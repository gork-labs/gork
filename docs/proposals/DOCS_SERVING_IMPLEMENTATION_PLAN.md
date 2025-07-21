# Documentation Serving Implementation Plan

## Overview

This document outlines the step-by-step implementation plan for the documentation serving feature described in DOCS_SERVING_DESIGN.md.

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)

#### 1.1 Create Documentation Handler Package

**File**: `pkg/api/docs/handler.go`
- [ ] Define `UIType` enum (Stoplight, SwaggerUI, Redoc)
- [ ] Define `DocsConfig` struct with all configuration options
- [ ] Implement `DocsHandler` struct
- [ ] Create `NewDocsHandler` constructor
- [ ] Implement `ServeHTTP` method for serving documentation

**File**: `pkg/api/docs/options.go`
- [ ] Define `DocsOption` function type
- [ ] Implement option functions:
  - [ ] `WithUI(ui UIType) DocsOption`
  - [ ] `WithOpenAPIPath(path string) DocsOption`
  - [ ] `WithTitle(title string) DocsOption`
  - [ ] `WithTheme(theme string) DocsOption`
  - [ ] `WithCustomCSS(css string) DocsOption`
  - [ ] `WithCustomJS(js string) DocsOption`
  - [ ] `WithCDN(enabled bool) DocsOption`
  - [ ] `WithBasePath(path string) DocsOption`

**File**: `pkg/api/docs/config.go`
- [ ] Define `DefaultDocsConfig()` function with sensible defaults
- [ ] Add validation for configuration options

#### 1.2 Asset Management

**File**: `pkg/api/docs/assets/embed.go`
- [ ] Create embed directives for UI assets
- [ ] Implement asset serving logic
- [ ] Add caching headers support

**File**: `pkg/api/docs/assets/templates/index.html`
- [ ] Create HTML template for Stoplight Elements
- [ ] Add configuration injection points
- [ ] Implement responsive design

### Phase 2: UI Integrations (Week 2)

#### 2.1 Stoplight Elements Integration

**File**: `pkg/api/docs/ui/stoplight.go`
- [ ] Implement Stoplight Elements renderer
- [ ] Add CDN fallback support
- [ ] Configure theme support

#### 2.2 Swagger UI Integration

**File**: `pkg/api/docs/ui/swagger.go`
- [ ] Implement Swagger UI renderer
- [ ] Add configuration options
- [ ] Support custom themes

#### 2.3 Redoc Integration

**File**: `pkg/api/docs/ui/redoc.go`
- [ ] Implement Redoc renderer
- [ ] Add configuration mapping
- [ ] Support theme customization

### Phase 3: Router Adapter Integration (Week 3)

#### 3.1 Define Common Interface

**File**: `pkg/api/docs_router.go`
- [ ] Define `DocsRouter` interface
- [ ] Add to existing router interfaces

#### 3.2 Standard Library Adapter

**File**: `pkg/adapters/stdlib/docs.go`
- [ ] Implement `DocsRoute` method
- [ ] Handle path normalization
- [ ] Register OpenAPI endpoints

#### 3.3 Chi Adapter

**File**: `pkg/adapters/chi/docs.go`
- [ ] Implement `DocsRoute` method
- [ ] Use Chi's routing features
- [ ] Handle middleware integration

#### 3.4 Echo Adapter

**File**: `pkg/adapters/echo/docs.go`
- [ ] Implement `DocsRoute` method
- [ ] Integrate with Echo's context
- [ ] Support Echo middleware

#### 3.5 Gin Adapter

**File**: `pkg/adapters/gin/docs.go`
- [ ] Implement `DocsRoute` method
- [ ] Use Gin's routing groups
- [ ] Handle Gin-specific features

#### 3.6 Gorilla Adapter

**File**: `pkg/adapters/gorilla/docs.go`
- [ ] Implement `DocsRoute` method
- [ ] Use Gorilla's subrouter
- [ ] Support path variables

### Phase 4: OpenAPI Spec Serving (Week 4)

#### 4.1 Spec Generation Enhancement

**File**: `pkg/api/openapi_generator.go`
- [ ] Add documentation-specific metadata
- [ ] Support version negotiation
- [ ] Add server URL injection

#### 4.2 Content Negotiation

**File**: `pkg/api/docs/openapi_handler.go`
- [ ] Implement OpenAPI spec handler
- [ ] Add JSON/YAML content negotiation
- [ ] Support version downgrade (3.1 → 3.0)
- [ ] Add caching headers

### Phase 5: Testing (Week 5)

#### 5.1 Unit Tests

**File**: `pkg/api/docs/handler_test.go`
- [ ] Test handler creation
- [ ] Test configuration options
- [ ] Test asset serving
- [ ] Test error cases

**File**: `pkg/api/docs/options_test.go`
- [ ] Test each option function
- [ ] Test option combinations
- [ ] Test validation

#### 5.2 Integration Tests

**Files**: `pkg/adapters/*/docs_test.go`
- [ ] Test DocsRoute implementation for each adapter
- [ ] Test route registration
- [ ] Test middleware integration
- [ ] Test OpenAPI endpoint

#### 5.3 E2E Tests

**File**: `pkg/api/docs/e2e_test.go`
- [ ] Test full documentation serving
- [ ] Test UI rendering
- [ ] Test API interaction from UI
- [ ] Test different configurations

### Phase 6: Documentation & Examples (Week 6)

#### 6.1 API Documentation

**File**: `pkg/api/docs/doc.go`
- [ ] Package documentation
- [ ] Usage examples
- [ ] Configuration guide

#### 6.2 Example Applications

**File**: `examples/docs/basic/main.go`
- [ ] Basic documentation setup
- [ ] Show minimal configuration

**File**: `examples/docs/advanced/main.go`
- [ ] Advanced configuration
- [ ] Custom themes and branding
- [ ] Authentication example

#### 6.3 README Updates

**File**: `README.md`
- [ ] Add documentation serving section
- [ ] Include quick start example
- [ ] Link to detailed documentation

### Phase 7: Performance & Security (Week 7)

#### 7.1 Performance Optimization

- [ ] Implement asset compression
- [ ] Add HTTP/2 push support
- [ ] Optimize bundle sizes
- [ ] Add performance benchmarks

#### 7.2 Security Hardening

- [ ] Implement CSP headers
- [ ] Add CORS configuration
- [ ] Support authentication middleware
- [ ] Add rate limiting hooks

### Phase 8: Polish & Release (Week 8)

#### 8.1 Final Testing

- [ ] Cross-browser testing
- [ ] Mobile responsiveness testing
- [ ] Load testing
- [ ] Security scanning

#### 8.2 Documentation Review

- [ ] Review all documentation
- [ ] Add migration guide
- [ ] Create troubleshooting guide
- [ ] Update changelog

#### 8.3 Release Preparation

- [ ] Version bump
- [ ] Release notes
- [ ] Demo video/GIF
- [ ] Blog post draft

## Technical Decisions

### 1. Asset Embedding Strategy

```go
//go:embed assets/stoplight/*
var stoplightAssets embed.FS

//go:embed assets/swagger/*
var swaggerAssets embed.FS

//go:embed assets/redoc/*
var redocAssets embed.FS
```

### 2. Path Handling

- Always normalize paths to end with `/*` for catch-all routing
- Strip base path before serving assets
- Handle SPA routing for client-side navigation

### 3. Configuration Precedence

1. Explicit options passed to `DocsRoute`
2. Environment variables (e.g., `GOAPI_DOCS_UI`)
3. Default configuration

### 4. Error Handling

- Return errors from `DocsRoute` for invalid configuration
- Log warnings for non-critical issues
- Graceful fallbacks for missing assets

## Dependencies

### Required

- Go 1.16+ (for embed support)
- No external runtime dependencies (all assets embedded)

### Build-time

- Stoplight Elements (via npm/yarn for updates)
- Swagger UI (via npm/yarn for updates)
- Redoc (via npm/yarn for updates)

## Success Metrics

1. **Ease of Use**: Setup time < 1 minute
2. **Performance**: Page load time < 2 seconds
3. **Bundle Size**: < 5MB for embedded assets
4. **Test Coverage**: > 90%
5. **Documentation**: All public APIs documented

## Risk Mitigation

1. **Large Bundle Sizes**
   - Mitigation: Offer CDN option, lazy loading

2. **Browser Compatibility**
   - Mitigation: Test on major browsers, provide polyfills

3. **Security Vulnerabilities**
   - Mitigation: Regular dependency updates, security scanning

4. **Performance Issues**
   - Mitigation: Caching, compression, CDN option

## Timeline

- **Total Duration**: 8 weeks
- **Buffer**: 2 weeks for unexpected issues
- **Review Checkpoints**: End of weeks 2, 4, 6

## Team Resources

- **Lead Developer**: Full-time for implementation
- **UI/UX Review**: Week 2 and 6
- **Security Review**: Week 7
- **Documentation Review**: Week 6 and 8 