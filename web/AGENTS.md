# React Admin Frontend

## Overview

TypeScript + React Admin 5.14 + Vite 7 + MUI 7. Build output goes to `../internal/webui/dist/`, embedded in Go binary via `go:embed`. Lint passes with 0 errors.

## Structure

```
web/src/
├── main.tsx              # React root mount
├── App.tsx               # React Admin config (Resources, CustomRoutes, permissions)
├── Layout.tsx            # Custom navigation menu (with Guides link)
├── Dashboard.tsx         # Dashboard overview
├── theme.ts              # Dark/light theme definitions
├── authProvider.ts       # JWT auth (localStorage)
├── dataProvider.ts       # Custom REST API client (not ra-data-simple-rest)
└── resources/            # Per-entity admin pages
    ├── projects.tsx       # Project CRUD
    ├── sshKeys.tsx        # SSH key management
    ├── providers.tsx      # Channel config
    ├── keywords.tsx       # Trigger keyword management (custom page, not standard Resource)
    ├── tasks.tsx          # Task list / detail
    ├── settings.tsx       # System settings (Monaco JSON editor)
    ├── mcpServers.tsx     # MCP server management
    ├── users.tsx          # User RBAC management
    └── Guides.tsx         # Channel integration guides
```

## Key Files

### `dataProvider.ts`
- Custom implementation (not `ra-data-simple-rest`)
- Most resources use client-side sorting + pagination
- `tasks` uses server-side pagination (query string `limit`/`offset`)
- Exports two non-standard methods: `installMcpServer(id)` and `saveKeywords(projectId, keywords)`
- `keywords` resource adds `id` client-side (backend doesn't return id, frontend uses index)
- Delete method uses `as DeleteResult` type assertion (React Admin generic constraint limitation)

### `authProvider.ts`
- Token stored in `localStorage.token`
- User object stored in `localStorage.user`
- Permissions derived from `user.role` (admin / editor / viewer)

### `App.tsx`
- Permission control: `viewer` cannot see create/edit buttons
- `admin` exclusive: SSH Keys, Settings, MCP Servers, Users
- `CustomRoutes`: `/guides` and `/keywords` (non-standard Resources)

## Build

```bash
npm run build    # tsc -b && vite build → outputs to ../internal/webui/dist/
npm run dev      # Dev mode (backend must be running, API proxied to localhost:8080)
npm run lint     # ESLint check (0 errors expected)
```

## Conventions

- TypeScript strict mode
- ESLint 9 flat config (react-hooks + react-refresh)
- No Prettier (ESLint only)
- No frontend tests
- Dark theme is default (`defaultTheme="dark"`)
- No `as any` or `@ts-ignore` — use proper TypeScript types

## Notes

- Vite `base: "/"` — SPA routing, Go server handles fallback
- `settings` uses `key` field as `id` (not UUID)
- Monaco Editor for JSON config editing (auth.json, .opencode.json, oh-my-opencode.json)
