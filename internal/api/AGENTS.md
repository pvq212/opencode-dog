# REST API Layer

## Overview

Per-resource handler files with shared `api.go` (struct, constructor, route registration, helpers). 73.3% test coverage via `api_test.go`.

## Files

| File | LOC | Handlers |
|------|-----|----------|
| `api.go` | 73 | `New()`, `RegisterRoutes()`, `requireRole()`, `writeJSON()`, `writeErr()` |
| `auth_handler.go` | 91 | `handleLogin`, `handleMe`, `handleChangePassword` |
| `project_handler.go` | 97 | `handleProjects`, `handleProjectDetail` |
| `ssh_key_handler.go` | 66 | `handleSSHKeys`, `handleSSHKeyDetail` |
| `provider_handler.go` | 64 | `handleProviders` |
| `keyword_handler.go` | 44 | `handleKeywords` |
| `task_handler.go` | 54 | `handleTasks`, `handleTaskDetail` |
| `setting_handler.go` | 76 | `handleSettings`, `handleSettingDetail` |
| `mcp_server_handler.go` | 121 | `handleMCPServers`, `handleMCPServerDetail` |
| `user_handler.go` | 116 | `handleUsers`, `handleUserDetail` |
| `api_test.go` | ~1080 | 55 tests covering all handlers, RBAC, validation |

## Route Structure

```
/api/auth/login          ← Public (no auth)
/api/**                  ← Protected via auth.Middleware (Bearer Token)
```

`RegisterRoutes(mux)` registers public routes directly on the mux; protected routes go on an internal `protected` mux wrapped with `auth.Middleware`.

## Handler Conventions

- Collection endpoints: `handleProjects`, `handleTasks` (dispatch by `r.Method` internally)
- Detail endpoints: `handleProjectDetail` (dispatch by `r.Method` — GET/PUT/DELETE)
- ID extraction: `strings.TrimPrefix(r.URL.Path, "/api/xxx/")`
- RBAC: inline `requireRole(w, r, db.RoleAdmin)` checks within each handler

## Response Format

- Success: `writeJSON(w, status, data)` — sets `Content-Type: application/json`
- Error: `writeErr(w, status, "message")` — returns `{"error": "message"}`

## Adding a New Endpoint

1. Create `{resource}_handler.go` with handler methods on `*API`
2. Register routes in `RegisterRoutes()` on the `protected` mux
3. Dispatch by `r.Method` (GET/POST/PUT/DELETE) within the handler
4. RBAC: call `a.requireRole(w, r, db.RoleAdmin, ...)` for write operations
5. Add tests to `api_test.go` using `newTestEnv()` + `doRequest()` helpers

## Notes

- No router parameter parser — path parsing via `strings.TrimPrefix`
- `/api/mcp-servers/{id}/install` is POST, triggers async install (`go a.mcpMgr.Install(...)`)
- Nested resources (providers, keywords) include projectId in path: `/api/providers/{projectId}`
- Tasks List API supports query string pagination (`limit`, `offset`)
