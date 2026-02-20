# Project Knowledge Base

## Overview

Multi-channel AI code analysis bot. Receives webhooks from GitLab / Slack / Telegram, triggers analysis via OpenCode Server HTTP API, and replies to the originating channel. Go backend + React Admin frontend, frontend compiled and embedded into a single binary via `go:embed`.

## Architecture Flow

```
Webhook Request → Provider Layer (validate + parse) → Analyzer (HTTP call to OpenCode Server) → Reply to channel
                                                                  ↕                ↕
                                                          PostgreSQL (all config)  OpenCode Server (Docker)
                                                                  ↑
                                                       React Admin WebUI (embedded in Go binary)
```

## Project Structure

```
opencode-dog/
├── cmd/server/main.go              # Entry point (config → server → auto migration → start)
├── internal/
│   ├── config/                     # Environment variable loading (infrastructure only)
│   ├── auth/                       # HMAC token auth + RBAC middleware (admin/editor/viewer)
│   │   ├── auth.go                 #   Login, token generation, middleware, password hashing
│   │   └── hmac.go                 #   HMAC encode/decode primitives
│   ├── db/                         # PostgreSQL CRUD (pgx v5 connection pool)
│   │   ├── store.go                #   Store interface (95 methods) — central abstraction
│   │   ├── models.go               #   9 model structs + enum constants
│   │   ├── db.go                   #   DB struct, New(), Close(), RunMigrations()
│   │   ├── dbmock/store.go         #   In-memory mock Store for unit testing
│   │   └── {entity}.go             #   Per-entity CRUD (project, user, task, ssh_key, etc.)
│   ├── provider/                   # Channel abstraction layer (interface + implementations)
│   │   ├── types.go                #   Provider interface + IncomingMessage + TriggerMode
│   │   ├── registry.go             #   Thread-safe provider registry
│   │   ├── gitlab.go               #   GitLab Note Event webhook handler
│   │   ├── slack.go                #   Slack Event API webhook handler
│   │   └── telegram.go             #   Telegram Bot API webhook handler
│   ├── analyzer/                   # OpenCode Server HTTP client
│   │   ├── analyzer.go             #   Keyword matching + analysis orchestration
│   │   └── opencode_client.go      #   Session management + synchronous message sending
│   ├── api/                        # REST API endpoints (split into per-resource handlers)
│   │   ├── api.go                  #   API struct, New(), RegisterRoutes(), helpers
│   │   └── {resource}_handler.go   #   Per-resource handlers (auth, project, user, etc.)
│   ├── mcp/                        # MCP Protocol server (mcp-go, exposes 5 tools)
│   ├── mcpmgr/                     # MCP npm package install/uninstall management
│   ├── server/                     # HTTP server assembly + webhook route registration + graceful shutdown
│   │   ├── server.go               #   Server composition root, wires all packages
│   │   └── mcp_handler.go          #   MCP SSE/HTTP handler adapter
│   └── webui/                      # go:embed frontend static files
│       └── embed.go                #   SPA file server with fallback routing
├── web/                            # React Admin frontend (TypeScript + Vite + MUI)
│   └── src/
│       ├── App.tsx                 #   React Admin config (Resources, permissions)
│       ├── Layout.tsx              #   Custom navigation menu
│       ├── Dashboard.tsx           #   Dashboard overview
│       ├── authProvider.ts         #   JWT auth (localStorage)
│       ├── dataProvider.ts         #   Custom REST API client
│       └── resources/              #   Per-entity admin pages
├── migrations/                     # PostgreSQL schema (auto-executed on startup)
│   ├── 001_init.sql                #   9 tables + 2 enums
│   └── 002_opencode_server.sql     #   OpenCode Server config fields
├── Dockerfile                      # Multi-stage build (Go builder → Node.js 22 runtime)
└── docker-compose.yml              # PostgreSQL 16 + OpenCode Server + App
```

## Quick Reference

| Task | Location | Notes |
|------|----------|-------|
| Add new channel (e.g. Discord) | `internal/provider/` | Implement `Provider` interface, register in `server.go` |
| Add/modify API endpoint | `internal/api/` | Create `{resource}_handler.go`, register in `api.go` RegisterRoutes() |
| Add database entity | `internal/db/` | Add to `models.go` + `store.go` interface + create `{entity}.go` + update `dbmock/store.go` |
| Modify trigger logic | `internal/analyzer/analyzer.go` | `matchKeyword()` + OpenCode HTTP client call |
| OpenCode HTTP client | `internal/analyzer/opencode_client.go` | Session management + synchronous message sending |
| Add MCP tool | `internal/mcp/server.go` | `registerTools()` method |
| Frontend pages | `web/src/resources/` | Each `.tsx` corresponds to a React Admin resource |
| Auth logic | `internal/auth/auth.go` | Custom HMAC token (not a standard JWT library) |
| Environment variables | `internal/config/config.go` | `getEnv()` pattern, defaults in code |
| Database schema | `migrations/*.sql` | Auto-executed on startup, idempotent design |

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24, net/http (stdlib), pgx v5 |
| Frontend | React 19, React Admin 5.14, Vite 7, MUI 7, Monaco Editor |
| Database | PostgreSQL 16 |
| Auth | Custom HMAC token, bcrypt password hashing |
| External SDK | go-gitlab v0.115, mcp-go v0.44 |
| AI Engine | OpenCode Server (Docker container, HTTP API) |
| Container | Docker multi-stage build (Go builder → Node.js 22 runtime) |

## Key Design Patterns

### Dependency Injection via `db.Store` Interface

All packages depend on the `db.Store` interface (defined in `internal/db/store.go`) instead of the concrete `*db.DB` type. This enables unit testing with `dbmock.Store` (an in-memory mock in `internal/db/dbmock/store.go`) without a live database.

```
db.Store interface (95 methods)
    ├── *db.DB          — production PostgreSQL implementation
    └── *dbmock.Store   — in-memory mock for testing (exported fields for state injection)
```

Consumers: auth, api, analyzer, provider (slack/telegram), mcp, mcpmgr, server — all accept `db.Store`.

### API Handler Pattern

Handlers are split into per-resource files (`{resource}_handler.go`) with a shared `api.go` containing the struct, constructor, route registration, and helpers (`writeJSON`, `writeErr`, `requireRole`).

Routes use standard `http.ServeMux`:
- Public routes (login) registered directly on the mux
- Protected routes wrapped with `auth.Middleware()` for Bearer token validation
- RBAC checked inline via `requireRole()` within each handler

### Provider Registry Pattern

New channels implement the `Provider` interface (4 methods: `Type`, `ValidateConfig`, `BuildHandler`, `SendReply`) and register with the thread-safe `Registry`. The server loads provider configs from the database on startup and wires webhook routes accordingly.

## Conventions

- **Stdlib HTTP only**: No gin/echo/chi — use `http.ServeMux` + `HandleFunc`
- **No ORM**: Hand-written SQL (pgx v5), models in `db/models.go`
- **Structured logging**: `log/slog` throughout (JSON handler) — never `fmt.Println` or `log`
- **Config separation**: Environment variables for infrastructure only (DB, JWT); business config in PostgreSQL
- **Frontend embedding**: Vite outputs to `internal/webui/dist/`, Go embeds via `go:embed`
- **Interface-driven testing**: All DB consumers depend on `db.Store` interface, tested with `dbmock.Store`
- **Standard library testing**: No testify/gomock — stdlib `testing` + `net/http/httptest` only
- **No CI/CD**: No GitHub Actions / GitLab CI — Docker Compose deployment only

## Testing

11 test files, 120+ tests across 6 packages:

| Package | Coverage | Key test areas |
|---------|----------|----------------|
| `auth` | 96.1% | Login, token round-trip, middleware, RBAC, password hashing, admin seeding |
| `config` | 100% | Environment variable loading with defaults |
| `mcp` | 95.1% | All 5 MCP tools, error handling, JSON serialization |
| `api` | 73.3% | All 10 REST handlers, RBAC enforcement, validation, error paths |
| `provider` | 67.8% | GitLab/Slack/Telegram webhook parsing, signature verification, registry |
| `analyzer` | 65.2% | Keyword matching, analysis orchestration, HTTP client mocking |

```bash
go test ./internal/... -count=1           # Run all tests
go test ./internal/... -coverprofile=c.out # With coverage
go tool cover -func=c.out                 # Coverage report
```

## Prohibited

- **Do not use `.env` for business config** — business config goes in PostgreSQL, managed via WebUI
- **Do not add third-party HTTP frameworks** — keep stdlib `net/http`
- **Do not introduce ORM** — maintain hand-written SQL + pgx v5
- **Do not use `log` or `fmt.Println`** — always `slog.Logger`
- **Do not modify `internal/webui/dist/`** — that's a build artifact; edit `web/src/` and rebuild
- **Do not use `as any` or `@ts-ignore` in frontend** — use proper TypeScript types
- **Do not delete failing tests** — fix the code, not the tests

## Security Notes

- `JWT_SECRET` generates a random key if unset; all tokens invalidate on restart
- Default account `admin/admin` — must change password on first login
- Webhook verification is per-channel (GitLab token, Slack signing_secret, Telegram secret_token)
- SSH private keys stored in database (`ssh_keys` table) — redacted from API list responses

## Commands

```bash
# Local development
cd web && npm install && npm run build && cd ..
export DB_PASSWORD=xxx JWT_SECRET=xxx OPENCODE_CONFIG_DIR=/tmp/oc
go run ./cmd/server

# Docker deployment
cp .env.example .env   # Edit DB_PASSWORD + JWT_SECRET
docker compose up -d   # http://localhost:8080

# Frontend development (hot reload)
cd web && npm run dev   # Backend must be running simultaneously

# Testing
go test ./internal/... -count=1 -v     # All backend tests
cd web && npm run lint                  # Frontend lint (0 errors expected)
cd web && npm run build                 # Frontend build

# Verification
go vet ./...                            # Static analysis
go build ./...                          # Build check
```

## Runtime Notes

- `analyzer.go` calls OpenCode Server via HTTP API with a 5-minute timeout
- OpenCode Server runs in a separate Docker container (port 4096)
- Config files (auth.json etc.) sync to shared volume; OpenCode Server restart needed after changes
- MCP package install has a 3-minute timeout
- Webhook routes load from DB at startup; new provider configs need restart (or use fallback `/hook/` route)
- Frontend `dataProvider.ts` uses client-side sorting/pagination for most resources; `tasks` uses server-side pagination
- Auth is custom HMAC (`auth/hmac.go`), not a standard JWT library
