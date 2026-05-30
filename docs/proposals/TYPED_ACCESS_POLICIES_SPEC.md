# Typed Access Policies Specification

## Overview

This specification proposes typed access policies as Gork's primary authorization model. Access policies are explicit route options that run after request parsing and validation, but before handler execution.

The goal is to replace string-tag authorization rules with Go-native policy functions that are typed to the request shape, can use application dependencies, and remain visible at route registration.

## Problem Statement

Authorization checks usually need more than the request data:

1. Authenticated actor or service principal
2. Database-backed ownership and membership lookups
3. Billing, plan, feature flag, and publication state
4. Cached permission services
5. Auditing, tracing, and structured logging context

Putting authorization inside request structs creates poor dependency boundaries:

```go
func (r UpdateSiteRequest) Authorize(ctx context.Context) error {
    // Where do DB, billing, and membership services come from?
}
```

This pushes dependencies into `context.Context`, makes DTOs import application services, or hides important security behavior away from route registration.

String-based rule tags have a similar problem:

```go
type UpdateSiteRequest struct {
    Path struct {
        Site Site `gork:"siteId" rule:"owned_by($current_user)"`
    }
}
```

The tag is compact, but it loses Go refactoring support, hides dependencies, and makes complex authorization harder to read and test.

## Goals

1. **Explicit Route Security**: Authorization requirements are declared where routes are registered.
2. **Typed Request Access**: Policy functions receive the fully parsed request type.
3. **Explicit Dependencies**: Policy functions receive an application environment instead of pulling services from context.
4. **Composable Policies**: Common resource policies can be reused across routes.
5. **AOT-Friendly Execution**: Generated route wrappers can call policies directly without reflection.
6. **OpenAPI Integration**: Security and authorization metadata can be reflected in generated documentation.

## Non-Goals

1. Replace local request validation such as `validate:"required"` tags.
2. Encode all business invariants as generic framework policies.
3. Hide authorization inside handler bodies.
4. Require request DTOs to depend on application services.
5. Remove the rule engine immediately; typed policies should become the preferred authorization model, while rules can remain useful for validation-adjacent checks.

## Request Lifecycle

Typed access policies run after authentication, parsing, and validation:

```text
receive HTTP request
authenticate actor
parse request
validate request
authorize typed request
execute handler
encode response
```

This ordering guarantees policy functions can rely on a valid request object and a known actor.

## Core Types

### Actor

Gork should provide a minimal actor abstraction and allow applications to define their own richer principal type.

```go
type Actor interface {
    ActorID() string
}
```

Applications may use concrete actor types:

```go
type UserActor struct {
    ID    UserID
    OrgID OrgID
    Roles []Role
}

func (a UserActor) ActorID() string {
    return string(a.ID)
}
```

### Access Function

An access policy is typed to the application environment, actor, and request:

```go
type AccessFunc[Env any, Actor any, Req any] func(
    ctx context.Context,
    env Env,
    actor Actor,
    req Req,
) error
```

Returning `nil` allows the request to continue. Returning an authorization error stops the pipeline before the handler runs.

### Authorization Errors

Gork should define sentinel or structured errors that map cleanly to HTTP responses:

```go
var (
    ErrUnauthenticated = errors.New("unauthenticated")
    ErrForbidden       = errors.New("forbidden")
)
```

The default mapping is:

| Error | HTTP Status |
| --- | --- |
| `ErrUnauthenticated` | `401 Unauthorized` |
| `ErrForbidden` | `403 Forbidden` |
| any other error | `500 Internal Server Error` |

Applications may provide an error mapper to customize this behavior.

## Route Registration API

### Go 1.27 Generic Method API

Once Go supports generic methods, the preferred route API should be:

```go
router.Put[UpdateSiteRequest, UpdateSiteResponse](
    "/sites/{siteId}",
    UpdateSite,
    api.RequireAuthenticated(),
    api.Authorize(CanUpdateSite),
)
```

### Pre-Go 1.27 Function API

If Gork supports pre-1.27 Go versions, equivalent top-level generic functions can provide the same type safety:

```go
api.Put[UpdateSiteRequest, UpdateSiteResponse](
    router,
    "/sites/{siteId}",
    UpdateSite,
    api.RequireAuthenticated(),
    api.Authorize(CanUpdateSite),
)
```

This avoids `interface{}` at the route boundary without waiting for generic methods.

## Example

```go
type Env struct {
    Sites  SiteRepository
    Access AccessService
}

type UpdateSiteRequest struct {
    Path struct {
        SiteID SiteID `gork:"siteId" validate:"required"`
    }
    Body struct {
        Name string `gork:"name" validate:"required,min=1,max=120"`
    }
}

type UpdateSiteResponse struct {
    Body SiteView
}

func UpdateSite(ctx context.Context, req UpdateSiteRequest) (*UpdateSiteResponse, error) {
    // Business logic only. Authorization already ran.
    return nil, nil
}

func CanUpdateSite(
    ctx context.Context,
    env *Env,
    actor UserActor,
    req UpdateSiteRequest,
) error {
    allowed, err := env.Access.CanUpdateSite(ctx, actor.ID, req.Path.SiteID)
    if err != nil {
        return err
    }
    if !allowed {
        return api.ErrForbidden
    }
    return nil
}
```

Route registration:

```go
router := api.NewRouter[*Env, UserActor](env)

router.Put[UpdateSiteRequest, UpdateSiteResponse](
    "/sites/{siteId}",
    UpdateSite,
    api.RequireAuthenticated(),
    api.Authorize(CanUpdateSite),
)
```

## Structural Request Interfaces

Request methods are still useful when they expose pure structural information without dependencies:

```go
type HasSiteID interface {
    SiteID() SiteID
}

func (r UpdateSiteRequest) SiteID() SiteID {
    return r.Path.SiteID
}
```

This enables reusable generic policies:

```go
func RequireSiteRole[Req HasSiteID](role Role) api.AccessFunc[*Env, UserActor, Req] {
    return func(ctx context.Context, env *Env, actor UserActor, req Req) error {
        return env.Access.RequireSiteRole(ctx, actor.ID, req.SiteID(), role)
    }
}
```

Usage:

```go
router.Put[UpdateSiteRequest, UpdateSiteResponse](
    "/sites/{siteId}",
    UpdateSite,
    api.RequireAuthenticated(),
    api.Authorize(RequireSiteRole[UpdateSiteRequest](RoleEditor)),
)
```

Request interfaces should be limited to dependency-free accessors such as `SiteID()`, `OrgID()`, `CollectionID()`, or `OwnerID()`.

## Policy Composition

Gork should support multiple policies per route:

```go
router.Post[PublishSiteRequest, PublishSiteResponse](
    "/sites/{siteId}/publish",
    PublishSite,
    api.RequireAuthenticated(),
    api.Authorize(RequireSiteRole[PublishSiteRequest](RoleEditor)),
    api.Authorize(RequirePaidPlan[PublishSiteRequest]()),
    api.Authorize(RequirePublishableSite),
)
```

Policies run in declaration order. Execution stops on the first non-nil error.

Composite helpers may be added for readability:

```go
api.AuthorizeAll(
    RequireSiteRole[PublishSiteRequest](RoleEditor),
    RequirePaidPlan[PublishSiteRequest](),
)
```

The base route option model should still preserve the individual policy descriptors for documentation and tooling.

## Policy Metadata

Authorization functions can optionally expose metadata for OpenAPI and generated docs.

```go
type AccessPolicy interface {
    Name() string
    Description() string
    Scopes() []string
}
```

Function-based policies can be wrapped:

```go
api.NamedPolicy(
    "site.editor",
    "Requires editor access to the target site.",
    RequireSiteRole[UpdateSiteRequest](RoleEditor),
)
```

Generated OpenAPI may include vendor extensions:

```json
{
  "x-gork-access": [
    {
      "name": "site.editor",
      "description": "Requires editor access to the target site."
    }
  ]
}
```

## AOT Generation

Generated route wrappers should call policies directly:

```go
func registerUpdateSiteRoute(r *Router[*Env, UserActor]) {
    r.NativePut("/sites/{siteId}", func(w http.ResponseWriter, httpReq *http.Request) {
        actor, err := authenticateUserActor(httpReq)
        if err != nil {
            writeAuthError(w, err)
            return
        }

        req, err := decodeUpdateSiteRequest(httpReq)
        if err != nil {
            writeBadRequest(w, err)
            return
        }

        if err := validateUpdateSiteRequest(req); err != nil {
            writeValidationError(w, err)
            return
        }

        if err := CanUpdateSite(httpReq.Context(), r.Env(), actor, req); err != nil {
            writeAccessError(w, err)
            return
        }

        res, err := UpdateSite(httpReq.Context(), req)
        if err != nil {
            writeHandlerError(w, err)
            return
        }

        encodeUpdateSiteResponse(w, res)
    })
}
```

This removes reflective policy invocation from the hot path.

## Relationship to the Rule Engine

Typed access policies should become the preferred model for authorization.

The rule engine remains useful for validation-adjacent checks when:

1. The rule is local to parsed request data.
2. The rule does not need application dependencies.
3. The failure should be treated like a request validation problem.

Examples:

```go
type SearchRequest struct {
    Query struct {
        Sort string `gork:"sort" validate:"oneof=name created updated"`
    }
}
```

Authorization examples should move to typed policies:

```go
api.Authorize(RequireSiteRole[UpdateSiteRequest](RoleEditor))
api.Authorize(RequireOrgPermission[InviteUserRequest](PermissionInviteMembers))
api.Authorize(RequirePaidPlan[PublishSiteRequest]())
```

## Implementation Plan

### Phase 1: Runtime Policy Pipeline

1. Add `AccessFunc[Env, Actor, Req]`.
2. Add `Authorize(...)` route option.
3. Add `RequireAuthenticated()` route option.
4. Extend route execution to run policies after parsing and validation.
5. Add default error mapping for unauthenticated and forbidden errors.

### Phase 2: Metadata and Documentation

1. Add optional named policy wrappers.
2. Store policy metadata in route registry.
3. Add OpenAPI `x-gork-access` extensions.
4. Document examples for common resource policies.

### Phase 3: AOT Generation

1. Generate direct policy calls in route wrappers.
2. Generate typed authentication extraction.
3. Generate validation before policy execution.
4. Remove reflection from policy execution in generated mode.

### Phase 4: Rule Engine Repositioning

1. Update rule engine documentation to describe validation-adjacent use cases.
2. Move authorization examples from rule tags to typed policies.
3. Provide migration examples for `rule:"owned_by($current_user)"`.

## Open Questions

1. Should `Actor` be a router-level generic parameter or an authentication-option generic parameter?
2. Should anonymous/public routes use `Actor` as a pointer, an option type, or a separate route mode?
3. Should policies be allowed to enrich context for handlers, or should enrichment be modeled as typed middleware?
4. Should policy metadata be mandatory for generated public API documentation?
5. How should route groups compose inherited policies with route-local policies?

## Recommendation

Use typed access policies as the primary authorization model for Gork applications. Keep request DTOs structural and dependency-free. Keep rules for validation-adjacent checks, but do not use string tags as the main authorization surface.

