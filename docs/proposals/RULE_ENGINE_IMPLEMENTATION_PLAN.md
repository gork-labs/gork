# Rule Engine Implementation Plan

## Scope

Implement the Rule Engine as specified in docs/specs/RULE_ENGINE_SPEC.md with:
- Global rule registry (unique names, thread-safe lookups)
- Rule tag parsing with argument grammar (field refs, literals, nested fields)
- Argument resolution against parsed request structs
- Execution pipeline integrated into Convention Over Configuration flow
- Error aggregation and mapping to existing validation error types
- Caching for performance
- Linter checks (registration and usage)

Non-goals (v1): inline expressions/operators, slice/map indexing in field refs, rule composition/templates (documented as future enhancements).

## Deliverables

- Production code under `pkg/rules` implementing registry, tag parsing, resolution, and execution; `pkg/api` integrates by invoking the engine (no cyclic deps)
- Linter updates under `internal/lintgork` for rule registration/usage checks
- Tests with high coverage for all new components and integration paths
- Documentation updates (examples + README references if needed)

## High-level Design

1. Registry (pkg/rules): `name -> descriptor{ fn reflect.Value, arity int, doc string }` with sync.RWMutex
2. Tag parser (pkg/rules): parse `rule:"..."` into ordered list of `Invocation{name, []ArgToken}`
3. Arg tokens (pkg/rules): FieldRef supports absolute `$.X.Y[...]` and relative `.X.Y[...]`; plus `Literal{String|Bool|Number|Null}`
4. Resolver (pkg/rules): cache of compiled accessors for field refs (per request type), navigate via reflection; needs resolution context for relative refs (parent value + inferred section when applicable)
5. Executor (pkg/rules): for each field’s rules, call fn(ctx, entity, args...) and aggregate validation errors, returning `[]error`
6. Integration (pkg/api): call rules engine from `ConventionHandlerFactory.executeConventionHandler` after validation, before handler invocation, and map returned errors to API validation errors

## Detailed Tasks

### 1) Registry and Public API
- Add `pkg/rules/registry.go`
  - `func RegisterRule(name string, fn interface{}) error` (panic on duplicate)
  - `func getRule(name string) (reflect.Value, bool)`
  - Validate fn is `func(context.Context, any, ...any) error` (variadic allowed, also accept fixed arity ≥2 for ergonomics)
  - Store metadata (arity expectation, package path/name via `runtime.FuncForPC`) for linter/doc
- Concurrency: protect map with `sync.RWMutex`

### 2) Tag Parsing
- Add `pkg/rules/tag.go` (standalone); API layer will call into it when processing request structs
  - Parse struct field tag `rule:"..."`
- Grammar: `rule = invocation *("," invocation)`; `invocation = ident ["(" [args] ")"]`
- Args: split on commas honoring quotes; produce tokens: FieldRef (absolute `$.…` or relative `.…` only), Literal(String|Bool|Number|Null)
  - Return `[]Invocation` maintaining order
- Unit tests for edge cases (spacing, quotes, empty args, multiple rules)

### 3) Argument Resolution
- Add `pkg/rules/resolver.go`
  - API: `ResolveArgs(reqPtr any, tokens []ArgToken) ([]any, *RequestValidationError)`
- FieldRef resolution:
    - Absolute: start at request root, traverse `X.Y[...]`
    - Relative: start at the parent struct of the annotated field (same level), traverse `X.Y[...]`
    - If resolving into `Body` when Body is `[]byte`, return `RequestValidationError`
  - Literal values: coerce as specified (numbers as float64)
- Tests: nested structs, missing sections/fields, Body as []byte, caching hit

### 4) Execution Pipeline Integration
- Add `pkg/rules/engine.go`
  - For a given request struct value, iterate all fields across sections in parse order
- For each field with `rule` tag:
    - Gather entity value (addressable as pointer if needed)
    - For each invocation in order:
      - Lookup rule function
      - Resolve args with context {parent value, inferred section name if parent is Path/Query/Body/Headers/Cookies}
      - Call fn via reflection: `fn(ctx, entity, args...)`
      - If returns a ValidationError: collect; continue
      - If returns non-nil non-ValidationError: return 500 early
  - Return aggregated ValidationError(s) if any
- Wire into `ConventionHandlerFactory.executeConventionHandler` (pkg/api):
  - After `ValidateRequest`, call rule engine; on validation errors, use existing error handling path (HTTP 400)

### 5) Error Types and Aggregation
- In pkg/rules: return `[]error` from engine without constructing API error types to avoid cyclic deps
- In pkg/api: map and aggregate into existing `*PathValidationError`, `*QueryValidationError`, `*BodyValidationError`, `*HeadersValidationError`, `*CookiesValidationError`, or a single `*RequestValidationError` when mixing types
- Tests for aggregation combinations in pkg/api integration layer

### 6) Performance & Caching
- Arg accessor cache: `map[key]compiledAccessor` where key = requestType + section + path
- Rule lookup cache uses registry map (read lock)
- Benchmarks (optional) around resolver to ensure no regressions

### 7) Linter Integration (internal/lintgork)
- Add checks:
  - rule-registration: validate signature, duplication, documentation presence (Go comment above registration site)
  - rule-usage: ensure rule exists, arg count matches (allow variadic), field refs valid for the request type (best-effort static check), basic literal validation
- Config flags wired as in SPEC `.lintgork.yml`

### 8) OpenAPI/Docs (optional)
- Expose rule metadata for routes in OpenAPI extensions (e.g., `x-rules`) via `RouteInfo`
- Extract comments for rules using doc scanner (defer if out of scope)

### 9) Testing Strategy
- Unit tests:
  - Registry: signature validation, duplicate name panic, concurrent reads
  - Tag parser: grammar coverage, literals, nested refs
  - Resolver: happy/edge paths, caching
  - Executor: ordering, aggregation, server error short-circuit
- Integration tests:
  - End-to-end request using `ConventionHandlerFactory` exercising rules across sections
  - Body as `[]byte` case returning resolution error
- Race tests: run parallel execution through rule engine

### 10) Documentation & Examples
- Update examples to demonstrate rule usage with field refs and literals
- Add README snippet referencing rules and their purpose (brief)

## Milestones

1. Registry + Parser (PR1)
2. Resolver + Caching (PR2)
3. Execution + Factory integration (PR3)
4. Tests + Benchmarks (PR4)
5. Linter checks (PR5)
6. Docs & Examples (PR6)

## Acceptance Criteria

- All tasks implemented with tests ≥ 95% coverage for new packages
- No API breaking changes outside the new Rule Engine APIs
- Linters pass; `make test` green across repo
- Documentation updated; examples compile
