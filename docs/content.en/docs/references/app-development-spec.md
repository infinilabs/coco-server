# Application Development Specification

> **Goal**: Standardize secure access, unify API and data management, enable centralized SSO-based authentication, and rapidly build scenario-based applications with a fully pluggable architecture.

This document describes the standard application development framework built on **INFINI Framework**. It covers YAML configuration (especially SSO/managed auth), backend API security, frontend permission control, and pluggable module design. Use this as a reference when building a new project from scratch.

---

## Table of Contents

- [1. YAML Configuration](#1-yaml-configuration)
  - [1.1 Configuration Structure Overview](#11-configuration-structure-overview)
  - [1.2 Security & Authentication Configuration](#12-security--authentication-configuration)
  - [1.3 SSO / OAuth Configuration (Managed Mode)](#13-sso--oauth-configuration-managed-mode)
  - [1.4 Server Identity & Provider Configuration](#14-server-identity--provider-configuration)
  - [1.5 RBAC Principal Provider Configuration](#15-rbac-principal-provider-configuration)
  - [1.6 Elasticsearch & API Configuration](#16-elasticsearch--api-configuration)
  - [1.7 Configuration Precedence & Protection](#17-configuration-precedence--protection)
- [2. Golang Backend Development Framework](#2-golang-backend-development-framework)
  - [2.1 Application Bootstrap](#21-application-bootstrap)
  - [2.2 API Permission Registration](#22-api-permission-registration)
  - [2.3 API Permission Usage](#23-api-permission-usage)
  - [2.4 Standardized User Login](#24-standardized-user-login)
  - [2.5 Standardized User Authorization](#25-standardized-user-authorization)
  - [2.6 Standardized Data Authorization](#26-standardized-data-authorization)
  - [2.7 Standardized API Audit](#27-standardized-api-audit)
  - [2.8 Standardized Data Audit](#28-standardized-data-audit)
- [3. React UI Development Framework](#3-react-ui-development-framework)
  - [3.1 Standardized Main Menu Permission Control](#31-standardized-main-menu-permission-control)
  - [3.2 Standardized Sub-Feature Permission Control](#32-standardized-sub-feature-permission-control)
  - [3.3 Standardized User Management](#33-standardized-user-management)
  - [3.4 Standardized Authorization Management](#34-standardized-authorization-management)
  - [3.5 Standardized Application Authorization](#35-standardized-application-authorization)
  - [3.6 Standardized Audit Viewing](#36-standardized-audit-viewing)
  - [3.7 Standardized Login, Logout, User Profile & Permissions](#37-standardized-login-logout-user-profile--permissions)
- [4. Pluggable Module Architecture](#4-pluggable-module-architecture)
  - [4.1 Architecture Overview](#41-architecture-overview)
  - [4.2 Backend Plugin Specification](#42-backend-plugin-specification)
  - [4.3 Frontend Plugin Specification](#43-frontend-plugin-specification)
  - [4.4 Module Manifest & Feature Discovery](#44-module-manifest--feature-discovery)
  - [4.5 Extension Points](#45-extension-points)
  - [4.6 Plugin Lifecycle](#46-plugin-lifecycle)
  - [4.7 End-to-End Plugin Example](#47-end-to-end-plugin-example)
- [5. Microservice Integration Standards](#5-microservice-integration-standards)

---

## 1. YAML Configuration

The INFINI Framework uses a single YAML file (e.g., `app.yml`) as the primary configuration. Understanding this file is essential — especially the security and SSO sections — because it determines whether the application runs in **standalone mode** (local user database) or **managed mode** (centralized SSO authentication).

### 1.1 Configuration Structure Overview

```yaml
# app.yml — top-level structure
myapp:                        # Application-specific namespace
  server:                     # Server identity & provider info
    ...
web:                          # Framework web server config
  security:                   # Authentication & authorization settings
    ...
enterprise:                   # Enterprise features (RBAC, multi-tenancy)
  security:
    rbac:
      ...
elastic:                      # Elasticsearch connection (or)
elasticsearch:                # Elasticsearch node list
  ...
api:                          # API server binding
  ...
```

**Mapping to Go structs**:

```
app.yml                       Go Access Path
─────────────────────────────────────────────────────
myapp:                  →     env.ParseConfig("myapp", &config)
web:                    →     global.Env().SystemConfig.WebAppConfig
web.security:           →     global.Env().SystemConfig.WebAppConfig.Security
web.security.managed:   →     global.Env().SystemConfig.WebAppConfig.Security.Managed
```

### 1.2 Security & Authentication Configuration

The `web.security` block controls all authentication behavior:

```yaml
web:
  security:
    enabled: true              # Master switch for authentication
    managed: false             # false = standalone, true = centralized SSO
    authentication:
      native:
        enabled: true          # Enable local username/password login
```

#### Standalone Mode (`managed: false`)

In standalone mode, the application manages its own user database:

- Local login (`POST /account/login`) with email + password
- Local profile (`GET /account/profile`)
- Local password change (`PUT /account/password`)
- Users, roles, and permissions stored locally
- Initial admin user created during `POST /setup/_initialize`

```yaml
# Standalone mode — minimal config
web:
  security:
    enabled: true
    managed: false
    authentication:
      native:
        enabled: true
```

#### Managed Mode (`managed: true`)

In managed mode, authentication is delegated to an **external identity provider** via OAuth/SSO. The application does **not** manage user credentials locally:

- Local login/password endpoints are **disabled** (return panic if called)
- Authentication flows through OAuth authorize → callback → JWT session
- User profile is fetched from the external provider and cached locally
- Roles and permissions can be managed by an external RBAC provider

```yaml
# Managed mode — centralized auth
web:
  security:
    enabled: true
    managed: true
    authentication:
      native:
        enabled: true
      oauth:
        cloud:                              # OAuth provider name (arbitrary)
          enabled: true
          client_secret: "your-secret"
          authorize_url: "https://idp.example.com/oauth/authorize"
          token_url: "https://idp.example.com/oauth/token"
          profile_url: "https://idp.example.com/account/profile"
          redirect_url: "https://myapp.example.com/sso/callback/cloud"
          success_page: "/#/home"           # Redirect after successful login
          failed_page: "/#/login"           # Redirect after failed login
          scopes:
            - openid
            - email
            - profile
          bootstrap_admin_users:
            - "admin-user-id-from-idp"      # Auto-grant admin role on first login
```

### 1.3 SSO / OAuth Configuration (Managed Mode)

This is the most critical section for centralized authentication.

#### OAuth Flow

```
User → App Login Page → Redirect to IdP (authorize_url)
                                  ↓
                         User authenticates at IdP
                                  ↓
IdP → Redirect to App (redirect_url) with auth code
                                  ↓
App Backend → Exchange code for token (token_url)
           → Fetch user profile (profile_url)
           → Create JWT session
           → Redirect to success_page
```

#### OAuth Callback Handling

In managed mode, the framework registers an OAuth callback handler at the `redirect_url` path. When the IdP redirects back:

1. **Exchange auth code** for an access token (`token_url`)
2. **Fetch user profile** from the IdP (`profile_url`)
3. **Extract user identity** (tenant ID, user ID, email, name, avatar)
4. **Cache profile** in local KV store (key: `tenantID:userID`)
5. **Create JWT session** via `AddUserAccessTokenToSession()`
6. **Redirect** to `success_page` (e.g., `/#/home`)

#### Managed Mode Impact

When `managed: true`, the following behavior changes:

| Component | Standalone (`managed: false`) | Managed (`managed: true`) |
|-----------|------|---------|
| `POST /account/login` | Active — local password auth | **Disabled** — panics if called |
| `GET /account/profile` | Returns profile from local user DB | Returns profile from KV cache (synced from IdP) |
| `PUT /account/password` | Active — local password change | **Disabled** — panics if called |
| `POST /setup/_initialize` | Creates local admin user | Skips local user creation |
| `/provider/_info` response | `managed: false` | `managed: true` |
| Roles & Permissions | Managed locally | Can be synced from external RBAC provider |
| UI Security page | Shows User tab (local user CRUD) | Shows Auth tab (authorization mapping) |

### 1.4 Server Identity & Provider Configuration

```yaml
myapp:
  server:
    public: false                    # Whether server is publicly accessible
    name: "My App Server"           # Human-readable server name
    endpoint: "https://myapp.example.com/"  # Server base URL
    encode_icon_to_base64: false    # Convert icons to data URIs

    # Minimum client version requirement
    minimal_client_version:
      number: "1.0"

    # Provider/organization info (shown in UI & client apps)
    provider:
      name: "My Company"
      description: "Enterprise AI Platform"
      icon: "https://myapp.example.com/favicon.ico"
      website: "https://myapp.example.com/"
      eula: "https://myapp.example.com/terms"
      privacy_policy: "https://myapp.example.com/privacy"
      banner: "https://myapp.example.com/banner.svg"

    # Remote store configuration (optional)
    store:
      endpoint: "https://store.example.com"
      local: false
```

The `GET /provider/_info` endpoint returns this configuration (minus sensitive fields) as the **feature discovery** mechanism for all clients:

```json
{
  "name": "My App Server",
  "endpoint": "https://myapp.example.com/",
  "managed": true,
  "provider": { "name": "My Company" },
  "health": { "status": "green" },
  "setup_required": false
}
```

### 1.5 RBAC Principal Provider Configuration

For managed mode, roles and principals can be synced from an external identity management service:

```yaml
enterprise:
  security:
    rbac:
      principal_provider:
        endpoint: https://idp.example.com     # External RBAC service
        token: "service-token-here"           # API token for authentication
        auto_refresh: true                     # Auto-refresh user/team data
        refresh_interval: 10s                  # Refresh frequency
```

This enables the application to:
- Fetch team/group membership from the external provider
- Map external roles to application permissions
- Auto-sync when users or teams change in the IdP

### 1.6 Elasticsearch & API Configuration

```yaml
# Elasticsearch connection
elastic:
  elasticsearch: "default"      # Default connection name

elasticsearch:
  - name: default
    enabled: true
    endpoints:
      - "https://elasticsearch.example.com:9200"
    basic_auth:
      username: admin
      password: "password"

# API server binding
api:
  enabled: true
  network:
    binding: "0.0.0.0:9000"
  tls:                          # Optional HTTPS
    enabled: false
```

### 1.7 Configuration Precedence & Protection

In managed mode, certain configuration fields are **protected** — they cannot be overridden by runtime KV-store updates. The YAML file is the sole source of truth for:

| Protected Field | Reason |
|----------------|--------|
| `managed` | Security mode must be set at deployment |
| `oauth` providers | SSO URLs are infrastructure-level config |
| `provider` | Organization identity is fixed |
| `endpoint` | Server address is environment-specific |
| `public` | Access mode is infrastructure-level |
| `version` | Version is build-time |

Non-protected fields (like server name, app settings) can be updated at runtime via `PUT /settings`.

---

## 2. Golang Backend Development Framework

The backend is built on **INFINI Framework** (`infini.sh/framework`), which provides a unified HTTP API server, ORM, security primitives, and plugin lifecycle management. Each business module registers its own API routes and permissions in a Go `init()` function.

### 2.1 Application Bootstrap

#### Module System

The framework uses a two-tier module registration system:

```go
func main() {
    app := framework.NewApp("myapp", "My Application", ...)
    app.Init(nil)
    defer app.Shutdown()

    app.Setup(func() {
        // System modules — framework infrastructure
        module.RegisterSystemModule(&web.WebModule{})
        module.RegisterSystemModule(&security.Module{})
        module.RegisterSystemModule(&api.APIModule{})
        module.RegisterSystemModule(&elastic.ElasticModule{})
        module.RegisterSystemModule(&metrics.MetricsModule{})

        // User plugins — business modules
        module.RegisterUserPlugin(&task.TaskModule{})
        module.RegisterUserPlugin(&pipeline.PipeModule{})
        module.RegisterUserPlugin(&modules.MyApp{})

        module.Start()
    }, func() {}, nil)

    app.Run()
}
```

#### Module Interface

Every module implements:

```go
type Module interface {
    Name()  string
    Setup()           // Register ORM schemas, load config
    Start() error     // Initialize resources, start background tasks
    Stop()  error     // Cleanup on shutdown
}
```

#### Sub-Module Loading

Sub-modules self-register via Go `init()` functions, triggered by blank imports:

```go
// modules/myapp.go
package modules

import (
    _ "myapp/modules/users"       // triggers users.init()
    _ "myapp/modules/projects"    // triggers projects.init()
    _ "myapp/modules/billing"     // triggers billing.init()
)

type MyApp struct{}

func (m *MyApp) Name() string { return "myapp" }

func (m *MyApp) Setup() {
    // Register ORM schemas
    orm.MustRegisterSchemaWithIndexName(core.User{}, "user")
    orm.MustRegisterSchemaWithIndexName(core.Project{}, "project")
}

func (m *MyApp) Start() error { return nil }
func (m *MyApp) Stop() error  { return nil }
```

Plugins are auto-discovered and imported via a generated file:

```go
// plugins/generated_plugins.go — auto-generated, do not edit
package plugins

import (
    _ "myapp/plugins/security/filter"
    _ "myapp/plugins/processors/enrichment"
    _ "myapp/plugins/enterprise/managed/security"
    // ... more plugin imports
)
```

### 2.2 API Permission Registration

#### Permission Model

Permissions follow a three-part key: **Category # Resource / Action**.

| Component | Description | Example |
|-----------|-------------|---------|
| **Category** | Application namespace | `myapp`, `generic` |
| **Resource** | Business entity or module | `project`, `user`, `report` |
| **Action** | CRUD operation or custom action | `create`, `read`, `update`, `delete`, `search` |

The resulting permission key looks like: `myapp#project/create`, `generic#security:role/search`.

#### Step 1: Define Permission Objects

```go
const Category = "myapp"
const Resource = "project"

func init() {
    createPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
    readPermission   := security.GetSimplePermission(Category, Resource, string(security.Read))
    updatePermission := security.GetSimplePermission(Category, Resource, string(security.Update))
    deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
    searchPermission := security.GetSimplePermission(Category, Resource, string(security.Search))
}
```

Standard actions: `security.Create`, `security.Update`, `security.Read`, `security.Delete`, `security.Search`.

Custom actions:
```go
publishPermission := security.GetSimplePermission(Category, Resource, "publish")
approvePermission := security.GetSimplePermission(Category, Resource, "approve")
```

#### Step 2: Register Permissions

```go
// Register multiple permissions at once
security.GetOrInitPermissionKeys(
    createPermission, readPermission, updatePermission,
    deletePermission, searchPermission, publishPermission,
)

// Or register a single permission
security.GetOrInitPermissionKey(readPermission)
```

#### Step 3: Assign Default Roles

```go
// Assign one permission to a role
security.AssignPermissionsToRoles(searchPermission, "widget")

// Assign multiple permissions to a role
security.RegisterPermissionsToRole("widget",
    readPermission, searchPermission,
)
```

#### Complete Module Example

```go
package project

import (
    "infini.sh/framework/core/api"
    "infini.sh/framework/core/security"
)

type APIHandler struct {
    api.Handler
}

const Category = "myapp"
const Resource = "project"

func init() {
    create := security.GetSimplePermission(Category, Resource, string(security.Create))
    read   := security.GetSimplePermission(Category, Resource, string(security.Read))
    update := security.GetSimplePermission(Category, Resource, string(security.Update))
    del    := security.GetSimplePermission(Category, Resource, string(security.Delete))
    search := security.GetSimplePermission(Category, Resource, string(security.Search))

    security.GetOrInitPermissionKeys(create, read, update, del, search)

    handler := APIHandler{}

    api.HandleUIMethod(api.POST,   "/project/",        handler.create, api.RequirePermission(create))
    api.HandleUIMethod(api.GET,    "/project/:id",     handler.get,    api.RequirePermission(read))
    api.HandleUIMethod(api.PUT,    "/project/:id",     handler.update, api.RequirePermission(update))
    api.HandleUIMethod(api.DELETE, "/project/:id",     handler.delete, api.RequirePermission(del))
    api.HandleUIMethod(api.GET,    "/project/_search", handler.search, api.RequirePermission(search))
}
```

### 2.3 API Permission Usage

#### Route-Level Access Control

| Option | Purpose | Example |
|--------|---------|---------|
| `api.RequirePermission(perm)` | Requires the specified permission | All authenticated endpoints |
| `api.RequireLogin()` | Requires login (checked before permission) | Sensitive management APIs |
| `api.OptionLogin()` | Login optional; works with or without auth | Public-facing with optional personalization |
| `api.AllowPublicAccess()` | No authentication required | `/provider/_info`, setup endpoints |

```go
// Require specific permission
api.HandleUIMethod(api.POST, "/project/", handler.create, api.RequirePermission(createPerm))

// Require login + permission (double layer)
api.HandleUIMethod(api.POST, "/admin/setting", handler.update,
    api.RequireLogin(), api.RequirePermission(updatePerm))

// Optional login
api.HandleUIMethod(api.GET, "/public/info", handler.info, api.OptionLogin())

// Public access (no auth)
api.HandleUIMethod(api.GET, "/provider/_info", handler.providerInfo, api.AllowPublicAccess())
```

#### Feature Flags

Feature annotations modify response behavior:

| Feature Flag | Purpose |
|---|---|
| `FeatureCORS` | Enable CORS headers |
| `FeatureByPassCORSCheck` | Skip CORS origin validation (embeddable widgets) |
| `FeatureMaskSensitiveField` | Mask sensitive fields (e.g., `"***"`) in response |
| `FeatureRemoveSensitiveField` | Remove sensitive fields entirely from response |
| `FeatureFingerprintThrottle` | Rate-limit requests by client fingerprint |

```go
// Search endpoint with CORS + sensitive field removal
var secretKeys = map[string]bool{"config": true, "api_key": true}

api.HandleUIMethod(api.GET, "/provider/_search", handler.search,
    api.RequirePermission(searchPerm),
    api.Feature(FeatureCORS),
    api.Feature(FeatureRemoveSensitiveField),
    api.Label(SensitiveFields, secretKeys),
)

// Rate-limited write endpoint
api.HandleUIMethod(api.POST, "/resource/", handler.create,
    api.RequirePermission(createPerm),
    api.Feature(FeatureFingerprintThrottle),
)
```

#### CORS Preflight Handling

For cross-origin endpoints, register a matching `OPTIONS` route:

```go
api.HandleUIMethod(api.OPTIONS, "/api/resource", handler.get,
    api.RequirePermission(perm), api.Feature(FeatureCORS))
api.HandleUIMethod(api.GET,     "/api/resource", handler.get,
    api.RequirePermission(perm), api.Feature(FeatureCORS))
```

#### Dynamic CORS Origin Registration

Modules can register custom CORS origin validators:

```go
security.RegisterAllowOriginFunc("my_module", myOriginValidator)
```

#### Auth Filter Pipeline

The framework uses a filter pipeline. The `AuthFilter` runs before every route handler:

```
Request → AuthFilter (priority 200) → [PermissionCheck] → Handler → Response
```

1. `AuthFilter` validates the session via `ValidateLogin()`.
2. User permissions are loaded via `security.GetUserPermissions(claims)`.
3. User context is injected via `security.AddUserToContext(r.Context(), claims)`.
4. Missing session → 401. No permissions → 403.
5. Handler accesses user via `security.GetUserFromContext(r.Context())`.

Custom filters can be added:

```go
func init() {
    api.RegisterUIFilter(&MyFilter{})
}

type MyFilter struct { api.Handler }
func (f *MyFilter) GetPriority() int { return 300 } // After auth (200)
```

### 2.4 Standardized User Login

#### Authentication Chain

The system supports four authentication methods, tried in order:

| Priority | Method | Header/Mechanism | Use Case |
|----------|--------|-----------------|----------|
| 1 | **Session Cookie (JWT)** | Cookie `user_session_access_token` | Web UI login |
| 2 | **Bearer Token** | `Authorization: Bearer <jwt>` | API client access |
| 3 | **API Token** | `X-API-TOKEN: <token>` | Programmatic (long-lived) |
| 4 | **Integration Header** | `APP-INTEGRATION-ID: <id>` | Widget/guest access |

```go
func ValidateLogin(w http.ResponseWriter, r *http.Request) (*security.UserSessionInfo, error) {
    claims, err := ValidateLoginByAccessTokenSession(w, r)      // 1. Session JWT
    if claims == nil || !claims.UserSessionInfo.IsValid() {
        claims, err = ValidateLoginByAuthorizationHeader(w, r)   // 2. Bearer token
    }
    if claims == nil || !claims.UserSessionInfo.IsValid() {
        claims, err = ValidateLoginByAPITokenHeader(w, r)        // 3. X-API-TOKEN
    }
    if claims == nil || !claims.UserSessionInfo.IsValid() {
        claims, err = ValidateLoginByIntegrationHeader(w, r)     // 4. Integration
    }
    return claims.UserSessionInfo, nil
}
```

#### Standalone Login

```
POST /account/login
Content-Type: application/json

{ "email": "user@example.com", "password": "password123" }
```

Response:
```json
{ "status": "ok", "access_token": "<jwt>", "expire_in": 1710460800 }
```

The handler: validates credentials via bcrypt → creates `UserSessionInfo` with roles → generates JWT (HMAC-SHA256, 24h expiry) → sets session cookie → returns token.

#### Managed Login (SSO/OAuth)

In managed mode, the login flow is:

1. Frontend reads `/provider/_info` → gets `managed: true`.
2. Frontend redirects user to the OAuth login endpoint (e.g., `/sso/login/cloud`).
3. Backend redirects to the IdP's `authorize_url` (configured in `web.security.authentication.oauth`).
4. User authenticates at the external IdP.
5. IdP redirects back to `redirect_url` (e.g., `/sso/callback/cloud`) with auth code.
6. Backend exchanges code for token → fetches profile → creates JWT session.
7. Backend redirects to `success_page`.

No local password is stored or checked in managed mode.

### 2.5 Standardized User Authorization

#### Role-Based Access Control (RBAC)

```
User → Roles → Permissions → API Access
```

- **Users** have one or more **roles**.
- **Roles** contain a set of **permission keys**.
- **API routes** require specific **permission keys**.

Permission resolution:
1. Auth filter extracts user session.
2. Permissions resolved from roles: `security.MustGetPermissionKeysByRole(user.Roles)`.
3. Result: flat list of permission keys: `["myapp#project/create", "myapp#project/search", ...]`.
4. Route checks if required permission is in the user's set.

#### API Token Authorization

API tokens have **scoped permissions** — always a subset of the creating user's permissions:

```go
// Creating a token — requested permissions must be a subset of user's own
if !util.IsSuperset(userPermissions, requestedPermissions) {
    panic("invalid permissions")
}

// Validating a token — effective = intersection of token + user's current permissions
effectivePermissions := security.IntersectSetsFast(tokenPermissions, userPermissions)
```

This ensures: tokens never exceed the creator's permissions; if a user's role is reduced, their tokens are automatically restricted.

#### Guest/Integration Authorization

Integrations can define a guest user auto-authenticating via the `APP-INTEGRATION-ID` header. The guest runs with the permissions of the configured `RunAs` user.

### 2.6 Standardized Data Authorization

#### ORM Context-Based Data Scoping

Data access is controlled through the ORM context system:

```go
// API handler — inherits user context, applies ownership filtering
ctx := orm.NewContextWithParent(req.Context())

// Internal/system operation — no user scoping
ctx := orm.NewContext()
ctx.DirectAccess()
ctx.PermissionScope(security.PermissionScopePlatform)
```

| Context Method | Purpose |
|---|---|
| `orm.NewContextWithParent(req.Context())` | Inherits authenticated user; ORM applies ownership filtering |
| `orm.NewContext()` | Bare context; no user/ownership info |
| `ctx.DirectAccess()` | Bypass owner-based read/write filtering |
| `ctx.DirectReadAccess()` | Bypass owner-based read filtering only |
| `ctx.PermissionScope(scope)` | Set data visibility scope (`Platform`, `Public`) |

#### Ownership-Based Filtering

Data records have a `_system.owner_id` field. When using `orm.NewContextWithParent(req.Context())`, the ORM automatically filters to records owned by the authenticated user unless `DirectAccess()` is called.

#### Multi-Level Data Access Control

For complex data access, combine ownership with sharing rules:

```
1. Extract user from context → security.MustGetUserFromContext(ctx)
2. Get user's own resources → query where owner_id == userID
3. Get shared resources → sharing service rules
4. Intersect with query-specified scope
5. Apply document-level allow/deny lists
6. Build final query with must/should/must_not clauses
```

### 2.7 Standardized API Audit

#### Recommended Implementation

Implement an API audit filter that hooks into the filter pipeline:

```go
type AuditFilter struct {
    api.Handler
}

func (f *AuditFilter) GetPriority() int {
    return 300 // After auth filter (200)
}

func (f *AuditFilter) ApplyFilter(
    method, pattern string,
    options *api.HandlerOptions,
    next httprouter.Handle,
) httprouter.Handle {
    return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
        startTime := time.Now()
        user, _ := security.GetUserFromContext(r.Context())

        next(w, r, ps) // Execute handler

        auditLog := AuditRecord{
            Timestamp: startTime,
            Duration:  time.Since(startTime),
            Method:    method,
            Path:      r.URL.Path,
            UserID:    user.MustGetUserID(),
            ClientIP:  r.RemoteAddr,
        }
        orm.Create(ctx, &auditLog)
    }
}

func init() {
    api.RegisterUIFilter(&AuditFilter{})
}
```

### 2.8 Standardized Data Audit

#### Recommended Implementation

Use ORM hooks or wrapper functions to track data changes:

```go
type DataAuditRecord struct {
    Timestamp  time.Time
    UserID     string
    Action     string   // "create", "update", "delete"
    Resource   string   // "project", "user", etc.
    RecordID   string
    Before     string   // JSON snapshot before change (for update/delete)
    After      string   // JSON snapshot after change (for create/update)
}
```

---

## 3. React UI Development Framework

The frontend is built with **React** + **Redux Toolkit** + **Ant Design** + **React Router**, using `elegant-router` for file-based routing.

### 3.1 Standardized Main Menu Permission Control

#### Route-Based Menu Generation

Menus are **automatically derived** from the route tree. Routes are filtered by user permissions before building the menu:

```
App Startup → fetchGetUserInfo() → get user.permissions[]
  → filterAuthRoutesByPermissions(allRoutes, permissions)
  → Register filtered routes in router
  → Build menus from filtered routes
```

#### Route Permission Declaration

Each route declares required permissions in `meta.permissions`:

```typescript
{
  name: 'projects',
  path: '/projects',
  component: 'layout.base',
  meta: {
    i18nKey: 'route.projects',
    title: 'projects',
    localIcon: 'project',
    order: 2,
    permissions: ['myapp#project/search']
  }
}
```

For routes accessible with **any one** of multiple permissions:

```typescript
{
  name: 'security',
  path: '/security',
  meta: {
    permissions: [
      'generic#security:authorization/search',
      'generic#security:user/search',
      'generic#security:role/search'
    ],
    permissionLogic: 'or'  // Any one permission is sufficient
  }
}
```

#### Route Filtering Logic

```typescript
function filterAuthRoutesByPermissions(routes, permissions) {
  // - If meta.permissions is empty → allowed
  // - If permissionLogic === 'or' → allowed if user has ANY listed permission
  // - Otherwise → allowed only if user has ALL listed permissions
  // - Recursively filters children
}
```

**Constant routes** (`meta.constant: true`) are always accessible: `login`, `403`, `404`, `500`.

Menu items auto-hide when all children are filtered out. Routes with `meta.hideInMenu: true` are excluded from the menu but remain accessible by URL.

### 3.2 Standardized Sub-Feature Permission Control

#### The `useAuth` Hook

All component-level permission checks use the `useAuth()` hook:

```typescript
export function useAuth() {
  const userInfo = useAppSelector(selectUserInfo);
  const isLogin = useAppSelector(getIsLogin);

  function hasAuth(codes: string | string[]) {
    if (!isLogin) return false;
    if (typeof codes === 'string')
      return userInfo?.permissions?.includes(codes);
    return codes.every(code => userInfo?.permissions?.includes(code));
  }

  return { hasAuth, permissions: userInfo?.permissions };
}
```

#### Usage in Components

```tsx
const MyPage = () => {
  const { hasAuth } = useAuth();

  const permissions = {
    create: hasAuth('myapp#project/create'),
    update: hasAuth('myapp#project/update'),
    delete: hasAuth('myapp#project/delete'),
  };

  return (
    <div>
      {permissions.create && <Button type="primary">New Project</Button>}
      <Table columns={[
        // ... data columns
        {
          title: 'Actions',
          hidden: !permissions.update && !permissions.delete,
          render: (_, record) => (
            <Dropdown menu={{
              items: [
                permissions.update && { key: 'edit', label: 'Edit' },
                permissions.delete && { key: 'delete', label: 'Delete' },
              ].filter(Boolean)
            }}>
              <EllipsisOutlined />
            </Dropdown>
          )
        }
      ]} />
    </div>
  );
};
```

#### Permission Key Format Convention

| Category | Format | Example |
|----------|--------|---------|
| Business modules | `<app>#<resource>/<action>` | `myapp#project/create` |
| Security modules | `generic#security:<resource>/<action>` | `generic#security:role/search` |
| API tokens | `generic#security:auth:api-token/<action>` | `generic#security:auth:api-token/create` |

### 3.3 Standardized User Management

User management is available in **non-managed mode** (standalone):

```tsx
const permissions = {
  read:   hasAuth('generic#security:user/read'),
  create: hasAuth('generic#security:user/create'),
  delete: hasAuth('generic#security:user/delete'),
  update: hasAuth('generic#security:user/update'),
};
```

Standard service APIs:
```typescript
fetchUsers(params)           // GET  /security/user/_search
fetchUser(id)                // GET  /security/user/:id
createUser(data)             // POST /security/user/
updateUser(id, data)         // PUT  /security/user/:id
deleteUser(id)               // DELETE /security/user/:id
```

### 3.4 Standardized Authorization Management

In **managed mode**, the Auth tab manages principal-to-role mappings:

```tsx
const permissions = {
  read:   hasAuth('generic#security:authorization/read'),
  create: hasAuth('generic#security:authorization/create'),
  delete: hasAuth('generic#security:authorization/delete'),
  update: hasAuth('generic#security:authorization/update'),
};
```

Tabs are conditionally rendered based on mode:

```tsx
const providerInfo = useAppSelector(getProviderInfo);

if (permissions.viewAuth && providerInfo?.managed) {
  tabs.push({ key: 'auth', component: Auth });
}
if (permissions.viewUser && !providerInfo?.managed) {
  tabs.push({ key: 'user', component: User });
}
if (permissions.viewRole) {
  tabs.push({ key: 'role', component: Role });
}
```

### 3.5 Standardized Application Authorization

API Token management for programmatic access:

```typescript
POST   /auth/access_token            // Create token (with scoped permissions)
GET    /auth/access_token/_search    // List tokens
PUT    /auth/access_token/:id        // Update token
DELETE /auth/access_token/:id        // Delete token
```

### 3.6 Standardized Audit Viewing

Recommended approach:

1. Create a `/audit` route with permission `generic#security:audit/search`.
2. Table view displaying API audit records and data audit records.
3. Filter by user, time range, action type, resource.
4. Follow the standard `useQueryParams` + `Table` pattern.

### 3.7 Standardized Login, Logout, User Profile & Permissions

#### Login Flow

**Standalone mode**:
```
1. User visits protected route
2. Route guard → redirect to /login
3. User submits email + password
4. POST /account/login → { access_token }
5. GET /account/profile → { id, name, permissions: [...] }
6. Store token + userInfo in localStorage + Redux
7. Filter routes by permissions → build menus
8. Redirect to original route
```

**Managed mode (SSO)**:
```
1. User visits protected route
2. Route guard → redirect to /login
3. Login page detects managed=true from providerInfo
4. Redirect to OAuth login endpoint (e.g., /sso/login/cloud)
5. Backend redirects to IdP's authorize_url
6. User authenticates at external IdP
7. IdP redirects back → /sso/callback → JWT session created
7. Redirect to success_page → GET /account/profile
8. Continue from step 6 above
```

```typescript
// store/slice/auth/index.ts
login: create.asyncThunk(async (params) => {
  const { data, error } = await fetchLogin(params);
  if (!error) {
    const { data: userInfo } = await fetchGetUserInfo();
    localStg.set('userInfo', userInfo);
    return { token: data.access_token, userInfo };
  }
  return false;
})
```

#### Logout Flow

```
1. POST /account/logout → destroy server session
2. clearAuthStorage() → remove localStorage
3. resetAuth() → reset Redux state
4. resetRouteStore() → clear dynamic routes
5. Redirect to /login
```

#### User Profile

```
GET /account/profile → {
  id: "user_id",
  name: "User Name",
  email: "user@example.com",
  permissions: ["myapp#project/search", "myapp#project/create", ...]
}
```

The `permissions` array (resolved from user's roles) is the single source of truth for both route filtering and component-level permission checks.

#### Request Authentication

All API requests include the auth token via interceptor:

```typescript
export function getAuthorization() {
  const token = localStg.get('token');
  return token ? `Bearer ${token}` : null;
}
```

#### Route Guard

Runs on every navigation:

| Condition | Action |
|-----------|--------|
| Logged in → navigating to `/login` | Redirect to `/` |
| Route is `constant` (public) | Allow |
| Not logged in → auth route | Redirect to `/login?redirect=...` |
| Logged in + has permission | Allow |
| Logged in + no permission | Redirect to `/403` |

Super role users (configurable via `VITE_STATIC_SUPER_ROLE`) bypass all permission checks.

---

## 4. Pluggable Module Architecture

This section describes the design for making every feature a **fully self-contained, pluggable module** — covering both backend API and frontend UI.

### 4.1 Architecture Overview

#### Target Architecture

```
┌────────────────────────────────────────────────────────────┐
│  main.go                                                   │
│  ├── framework modules (web, security, api, elastic...)    │
│  └── feature modules (self-contained, independently built) │
│      ├── modules/users/    ──┐                             │
│      ├── modules/projects/   │  Each module:               │
│      ├── modules/billing/    ├─ • Manifest (metadata)      │
│      ├── modules/reports/    │  • ORM schemas              │
│      └── plugins/custom/   ──┘  • API routes               │
│                                  • Permissions              │
│                                  • Config schema            │
│                                  • UI metadata              │
│  ┌──────────────────────────┐                              │
│  │  GET /provider/_info      │ → module registry + caps    │
│  └──────────────────────────┘                              │
└────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────┐
│  web/ (React SPA — runtime composition)                    │
│  ├── Core shell (layout, auth, routing engine)             │
│  ├── Module registry (backend-driven routes/menus)         │
│  ├── Extension points (NavSlot, PageSlot, SettingsSlot)    │
│  └── Dynamic page loading (lazy import or micro-frontend)  │
└────────────────────────────────────────────────────────────┘
```

Key principles:
- Each module self-registers (ORM schemas, routes, permissions, manifest) — nothing in a central file.
- Enable/disable a feature by adding/removing a single Go import.
- Backend advertises installed modules via `/provider/_info`.
- Frontend builds routes, menus, and extension slots dynamically from the backend manifest.

### 4.2 Backend Plugin Specification

#### Module Interface

```go
type PluggableModule interface {
    Name()       string
    Setup()
    Start()      error
    Stop()       error
    ModuleInfo() ModuleManifest  // Metadata for discovery
}
```

#### Module Manifest

```go
type ModuleManifest struct {
    Name         string            `json:"name"`
    Version      string            `json:"version"`
    Category     string            `json:"category"`
    Description  string            `json:"description"`
    Dependencies []string          `json:"dependencies"`
    Schemas      []SchemaInfo      `json:"schemas"`
    Permissions  []PermissionInfo  `json:"permissions"`
    UIExtensions []UIExtension     `json:"ui_extensions"`
    ConfigSchema map[string]any    `json:"config_schema"`
}

type UIExtension struct {
    Type            string   `json:"type"`             // "route", "menu", "slot", "settings_tab"
    Name            string   `json:"name"`
    Path            string   `json:"path"`
    Component       string   `json:"component"`
    Icon            string   `json:"icon"`
    Order           int      `json:"order"`
    ParentMenu      string   `json:"parent_menu"`
    Permissions     []string `json:"permissions"`
    PermissionLogic string   `json:"permission_logic"` // "and" or "or"
    I18nKey         string   `json:"i18n_key"`
}
```

#### Self-Contained Module

```go
package billing

type Module struct{}

func (m *Module) Name() string { return "billing" }

func (m *Module) ModuleInfo() module.ModuleManifest {
    return module.ModuleManifest{
        Name:         "billing",
        Version:      "1.0.0",
        Category:     "myapp",
        Description:  "Billing and subscription management",
        Dependencies: []string{"users"},
        UIExtensions: []module.UIExtension{
            {Type: "route", Name: "billing", Path: "/billing", Icon: "dollar",
             Permissions: []string{"myapp#billing/search"}, Order: 5},
            {Type: "settings_tab", Name: "billing-settings",
             Permissions: []string{"myapp#billing/update"}},
        },
    }
}

func (m *Module) Setup() {
    orm.MustRegisterSchemaWithIndexName(Invoice{}, "invoice")

    create := security.GetSimplePermission("myapp", "billing", string(security.Create))
    search := security.GetSimplePermission("myapp", "billing", string(security.Search))
    // ... register permissions, register routes
}

func (m *Module) Start() error { return nil }
func (m *Module) Stop() error  { return nil }

func init() {
    module.RegisterFeatureModule(&Module{})
}
```

#### Module Registry

```go
var featureModules []PluggableModule

func RegisterFeatureModule(m PluggableModule) {
    featureModules = append(featureModules, m)
}

func GetAllModuleManifests() []ModuleManifest {
    var manifests []ModuleManifest
    for _, m := range featureModules {
        manifests = append(manifests, m.ModuleInfo())
    }
    return manifests
}

func ValidateDependencies() error {
    registered := map[string]bool{}
    for _, m := range featureModules {
        registered[m.Name()] = true
    }
    for _, m := range featureModules {
        for _, dep := range m.ModuleInfo().Dependencies {
            if !registered[dep] {
                return fmt.Errorf("module %q requires %q which is not registered", m.Name(), dep)
            }
        }
    }
    return nil
}
```

#### Enhanced Provider Info

`/provider/_info` advertises installed modules:

```json
{
  "provider": { "name": "My Company" },
  "managed": true,
  "modules": [
    {
      "name": "billing",
      "version": "1.0.0",
      "ui_extensions": [
        {"type": "route", "name": "billing", "path": "/billing", "icon": "dollar",
         "permissions": ["myapp#billing/search"], "order": 5}
      ]
    }
  ]
}
```

### 4.3 Frontend Plugin Specification

#### Dynamic Route Registration

Build routes at runtime from module manifests:

```typescript
const componentRegistry: Record<string, () => Promise<any>> = {
  'home':          () => import('@/pages/home/index.tsx'),
  'login':         () => import('@/pages/login/index.tsx'),
  'billing_list':  () => import('@/pages/billing/list.tsx'),
  'billing_edit':  () => import('@/pages/billing/edit/[id].tsx'),
};

export function registerPageComponent(name: string, loader: () => Promise<any>) {
  componentRegistry[name] = loader;
}

export function buildRoutesFromManifest(modules, userPermissions): RouteObject[] {
  const routes = [];
  for (const mod of modules) {
    for (const ext of mod.ui_extensions.filter(e => e.type === 'route')) {
      if (!checkPermission(ext.permissions, ext.permission_logic, userPermissions)) continue;
      const loader = componentRegistry[ext.component];
      if (!loader) continue;
      routes.push({ path: ext.path, lazy: loader, handle: { ...ext } });
    }
  }
  return routes.sort((a, b) => (a.handle?.order ?? 99) - (b.handle?.order ?? 99));
}
```

#### Extension Points (Slots)

Named slots in the layout where plugins inject UI:

```tsx
export const ExtensionSlot: React.FC<{ name: string; context?: any }> = ({ name, context }) => {
  const extensions = useExtensions(name);
  const { hasAuth } = useAuth();

  return (
    <>
      {extensions
        .filter(ext => !ext.permissions?.length || hasAuth(ext.permissions))
        .sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
        .map(ext => <ext.component key={ext.name} {...context} />)}
    </>
  );
};
```

Standard slot names:

| Slot | Location | Purpose |
|------|----------|---------|
| `sidebar-bottom` | Bottom of sidebar | Extra nav items |
| `content-top` | Top of content area | Banners, alerts |
| `settings-tabs` | Settings page | Plugin config tabs |
| `user-menu` | User dropdown | Extra user actions |
| `dashboard-widgets` | Home/dashboard | Plugin dashboard cards |

#### Plugin Registration

```typescript
interface PluginDefinition {
  name: string;
  version: string;
  routes?: Array<{ path: string; component: () => Promise<any>; meta: RouteMeta }>;
  extensions?: Array<{ slot: string; component: React.ComponentType; order?: number; permissions?: string[] }>;
  i18n?: Record<string, Record<string, string>>;
  onActivate?: () => void;
}

class PluginRegistry {
  private plugins = new Map<string, PluginDefinition>();
  private slotExtensions = new Map<string, SlotExtension[]>();

  register(plugin: PluginDefinition) {
    this.plugins.set(plugin.name, plugin);
    for (const ext of plugin.extensions ?? []) {
      const arr = this.slotExtensions.get(ext.slot) ?? [];
      arr.push({ ...ext, pluginName: plugin.name });
      this.slotExtensions.set(ext.slot, arr);
    }
    for (const route of plugin.routes ?? []) {
      registerPageComponent(`${plugin.name}:${route.path}`, route.component);
    }
    if (plugin.i18n) mergeI18nResources(plugin.i18n);
    plugin.onActivate?.();
  }

  getSlotExtensions(slot: string) { return this.slotExtensions.get(slot) ?? []; }
}

export const pluginRegistry = new PluginRegistry();
```

### 4.4 Module Manifest & Feature Discovery

#### Frontend Initialization Sequence

```typescript
async function init() {
  // 1. Get server info + module manifests
  const { data: providerInfo } = await fetchServer();
  store.dispatch(setProviderInfo(providerInfo));

  // 2. Load remote plugins (if any)
  await initRemotePlugins(providerInfo.modules ?? []);

  // 3. Get user session
  const { data: userInfo } = await fetchGetUserInfo();

  // 4. Build dynamic routes from manifests + local plugins
  const backendRoutes = buildRoutesFromManifest(providerInfo.modules, userInfo.permissions);
  const localRoutes = pluginRegistry.getAllRoutes();
  const allRoutes = mergeAndDedup([...backendRoutes, ...localRoutes]);

  // 5. Filter by permissions → register routes → build menus
  const allowedRoutes = filterRoutesByPermissions(allRoutes, userInfo.permissions);
  router.addRoutes(allowedRoutes);
  store.dispatch(setMenus(buildMenusFromRoutes(allowedRoutes)));
}
```

#### Permissions API

Dedicated endpoint listing all registered permissions (for role management UI):

```go
// GET /security/permission/_manifest
func (h *APIHandler) permissionManifest(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
    manifests := module.GetAllModuleManifests()
    permissions := []util.MapStr{}
    for _, m := range manifests {
        for _, p := range m.Permissions {
            permissions = append(permissions, util.MapStr{
                "module":   m.Name,
                "category": p.Category,
                "resource": p.Resource,
                "action":   p.Action,
                "key":      fmt.Sprintf("%s#%s/%s", p.Category, p.Resource, p.Action),
            })
        }
    }
    h.WriteJSON(w, util.MapStr{"permissions": permissions}, 200)
}
```

### 4.5 Extension Points

#### Backend

| Extension Point | Registration API | Use Case |
|---|---|---|
| Feature Module | `module.RegisterFeatureModule(m)` | Full module with routes, schemas, permissions |
| Pipeline Processor | `pipeline.RegisterProcessorPlugin(name, factory)` | Data processing plugins |
| Auth Filter | `api.RegisterUIFilter(filter)` | Request middleware (auth, audit, rate-limit) |
| CORS Origin | `security.RegisterAllowOriginFunc(name, fn)` | Dynamic CORS origin |
| Deferred Setup | `global.RegisterFuncBeforeSetup(fn)` | Pre-setup initialization |
| Post-Setup | `global.RegisterFuncAfterSetup(fn)` | Post-setup config loading |

#### Frontend

| Extension Point | Mechanism | Use Case |
|---|---|---|
| Route | `pluginRegistry.register({ routes })` | Add pages |
| Menu | Auto-derived from routes | Sidebar navigation |
| Slot Extension | `pluginRegistry.register({ extensions: [{ slot }] })` | Inject UI into existing pages |
| i18n | `pluginRegistry.register({ i18n })` | Add translations |
| Remote Plugin | `loadRemotePlugin(url)` | Runtime-loaded external plugins |

### 4.6 Plugin Lifecycle

#### Backend

```
Go init() → RegisterFeatureModule()
    ↓
ValidateDependencies() — check required modules exist
    ↓
module.Setup() — register ORM schemas, permissions, routes
    ↓
module.Start() — init resources, start background tasks
    ↓
Running — /provider/_info returns manifests
    ↓
module.Stop() — cleanup on shutdown
```

#### Frontend

```
Built-in plugins register (pluginRegistry.register())
    ↓
GET /provider/_info — receive module manifests
    ↓
Load remote plugins (if any)
    ↓
GET /account/profile — receive user permissions
    ↓
Build routes + menus (merge backend + local, filter by permissions)
    ↓
Running — ExtensionSlots render, lazy-load pages on demand
```

### 4.7 End-to-End Plugin Example

A complete **"Reports"** feature as a pluggable module:

#### Backend: `modules/reports/init.go`

```go
package reports

import (
    "infini.sh/framework/core/module"
    "infini.sh/framework/core/orm"
    "infini.sh/framework/core/api"
    "infini.sh/framework/core/security"
)

type Module struct{}

func (m *Module) Name() string { return "reports" }

func (m *Module) ModuleInfo() module.ModuleManifest {
    return module.ModuleManifest{
        Name: "reports", Version: "1.0.0", Category: "myapp",
        UIExtensions: []module.UIExtension{
            {Type: "route", Name: "reports", Path: "/reports", Icon: "chart",
             Permissions: []string{"myapp#report/search"}, Order: 7},
            {Type: "slot", Name: "dashboard-report-widget",
             Permissions: []string{"myapp#report/read"}},
        },
    }
}

func (m *Module) Setup() {
    orm.MustRegisterSchemaWithIndexName(Report{}, "report")

    create := security.GetSimplePermission("myapp", "report", string(security.Create))
    read   := security.GetSimplePermission("myapp", "report", string(security.Read))
    search := security.GetSimplePermission("myapp", "report", string(security.Search))
    security.GetOrInitPermissionKeys(create, read, search)

    h := APIHandler{}
    api.HandleUIMethod(api.POST, "/report/",        h.create, api.RequirePermission(create))
    api.HandleUIMethod(api.GET,  "/report/:id",     h.get,    api.RequirePermission(read))
    api.HandleUIMethod(api.GET,  "/report/_search", h.search, api.RequirePermission(search))
}

func (m *Module) Start() error { return nil }
func (m *Module) Stop() error  { return nil }

func init() { module.RegisterFeatureModule(&Module{}) }
```

#### Frontend: `src/pages/reports/plugin.ts`

```typescript
import { pluginRegistry } from '@/lib/pluginRegistry';

pluginRegistry.register({
  name: 'reports',
  version: '1.0.0',
  routes: [
    { path: '/reports', component: () => import('./list.tsx'),
      meta: { permissions: ['myapp#report/search'], icon: 'chart', order: 7 } },
    { path: '/reports/:id', component: () => import('./detail/[id].tsx'),
      meta: { permissions: ['myapp#report/read'], hideInMenu: true } },
  ],
  extensions: [
    { slot: 'dashboard-widgets',
      component: lazy(() => import('./components/DashboardWidget')),
      permissions: ['myapp#report/read'], order: 10 },
  ],
  i18n: {
    'en-US': { 'route.reports': 'Reports' },
    'zh-CN': { 'route.reports': '报表' },
  },
});
```

#### Enable / Disable

```go
// To enable: add import
import _ "myapp/modules/reports"

// To disable: remove the import line. No other files need modification.
```

---

## 5. Microservice Integration Standards

### Authentication Methods for External Services

| Method | Header | Best For |
|--------|--------|----------|
| **API Token** | `X-API-TOKEN: <token>` | Service-to-service (long-lived) |
| **Bearer JWT** | `Authorization: Bearer <jwt>` | Short-lived authenticated requests |
| **Integration ID** | `APP-INTEGRATION-ID: <id>` | Widget/embedded access with guest permissions |

### Creating a Service Token

```bash
curl -X POST https://myapp.example.com/auth/access_token \
  -H "Authorization: Bearer <admin_jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-service-token",
    "permissions": [
      {"category": "myapp", "resource": "report", "action": "search"},
      {"category": "myapp", "resource": "report", "action": "read"}
    ]
  }'
```

### New Module Checklist

#### Backend

1. Create package under `modules/<name>/`.
2. Implement `PluggableModule` interface (`Name`, `Setup`, `Start`, `Stop`, `ModuleInfo`).
3. In `Setup()`: register ORM schemas, define permissions, register API routes.
4. Call `module.RegisterFeatureModule(&Module{})` in `init()`.
5. Add blank import in main module file.

#### Frontend

1. Create `src/pages/<name>/plugin.ts` — register routes, slots, i18n.
2. Create page components (`list.tsx`, `new.tsx`, `edit/[id].tsx`).
3. Add service API in `src/service/api/<name>.ts`.
4. Use `useAuth()` for per-component permission control.

#### Configuration

1. Define YAML config section under application namespace.
2. Add JSON Schema in `ModuleManifest.ConfigSchema` for validation.
3. Document required and optional settings.

---

## Appendix A: Security Configuration Quick Reference

### Standalone Mode (Local Auth)

```yaml
myapp:
  server:
    endpoint: "https://myapp.example.com/"

web:
  security:
    enabled: true
    managed: false
    authentication:
      native:
        enabled: true
```

### Managed Mode (Centralized SSO)

```yaml
myapp:
  server:
    endpoint: "https://myapp.example.com/"
    provider:
      name: "My Company"

web:
  security:
    enabled: true
    managed: true
    authentication:
      native:
        enabled: true
      oauth:
        cloud:
          enabled: true
          client_secret: "your-client-secret"
          authorize_url: "https://idp.example.com/oauth/authorize"
          token_url: "https://idp.example.com/oauth/token"
          profile_url: "https://idp.example.com/account/profile"
          redirect_url: "https://myapp.example.com/sso/callback/cloud"
          success_page: "/#/home"
          failed_page: "/#/login"
          scopes: ["openid", "email", "profile"]
          bootstrap_admin_users: ["admin-user-id"]

enterprise:
  security:
    rbac:
      principal_provider:
        endpoint: https://idp.example.com
        token: "service-api-token"
        auto_refresh: true
        refresh_interval: 10s
```

## Appendix B: Permission Key Reference

### Application Permissions (Category: `<app>`)

| Resource | Standard Actions | Custom Actions (examples) |
|----------|-----------------|--------------------------|
| Any resource | `create`, `read`, `update`, `delete`, `search` | `publish`, `approve`, `export` |

### Security Permissions (Category: `generic`)

| Resource | Actions |
|----------|---------|
| `security:user` | `create`, `read`, `update`, `delete`, `search` |
| `security:role` | `create`, `read`, `update`, `delete`, `search` |
| `security:authorization` | `create`, `read`, `update`, `delete`, `search` |
| `security:auth:api-token` | `create`, `update`, `delete`, `search` |

## Appendix C: Design Decisions

| Decision | Rationale |
|----------|-----------|
| YAML-driven security mode | Deployment-level concern; not changeable at runtime |
| Protected config fields in managed mode | Prevents runtime override of security-critical settings |
| Go `init()` self-registration | Compile-time safety; add/remove feature by import |
| Backend-driven UI extensions | Single source of truth; frontend can't show features backend doesn't have |
| `ExtensionSlot` pattern | Decoupled composition; plugins don't need to know about each other |
| Permission-gated everything | Consistent security: routes, APIs, UI slots, menus |
| Lazy imports for all plugin components | Plugins don't impact initial bundle size |
| OAuth `bootstrap_admin_users` | First-login bootstrap; avoids chicken-and-egg for managed mode admin setup |
