# AI Assistant Instructions — BeerBot

## Overview

Monorepo for BeerBot — a Slack bot that tracks team appreciation through virtual beer giving.

| Component | Path | Tech Stack |
|-----------|------|------------|
| Backend | `backend/bot/` | Go, SQLite, Slack Socket Mode |
| Frontend | `frontend/project/` | Next.js (App Router), React, Tailwind, Bun |

## Quick Commands (using Just)

All commands run from the repo root. Install [just](https://github.com/casey/just) first.

```bash
just              # List all available commands
just dev          # Start dev environment (hot reload)
just dev-logs     # View all logs (tailing!)
just dev-down     # Stop dev environment
just test         # Run tests
just clean        # Remove containers and volumes
```

**Other useful commands:**
- `just dev-logs-backend` / `just dev-logs-frontend` — component-specific logs
- `just shell-backend` / `just shell-frontend` — shell into containers
- `just prod` — start production environment
- `just status` — show running services

## Key Files

**Backend (`backend/bot/`):**
- `main.go` — Entry point, HTTP server, Slack event routing
- `store.go` — SQLite migrations, queries, `user_cache` table, deduplication via `processed_events`
- `events.go` — Slack event processing, beer detection, mention parsing
- `handlers.go` — REST API handlers (`/api/givers`, `/api/recipients`, `/api/user`, etc.)
- `middleware.go` — Auth middleware (Bearer token validation)

**Frontend (`frontend/project/`):**
- `app/api/proxy/[...path]/route.ts` — Proxy to backend, injects `API_TOKEN`
- `components/UsersList.tsx` — User display with localStorage caching
- `components/UsersPage.tsx` — Main leaderboard page
- `app/globals.css` — Theme variables and styles

**Config:**
- `compose.dev.yaml` — Development Docker Compose
- `compose.yaml` — Production Docker Compose
- `.env` — Environment variables (Slack tokens, etc.)

## Environment Variables

**Backend (via `.env`):**
- `BOT_TOKEN` — Slack Bot User OAuth Token
- `APP_TOKEN` — Slack App-Level Token (needs `connections:write`)
- `API_TOKEN` — Bearer token for REST API auth
- `DB_PATH` — SQLite database path (default: `/data/bot.db`)
- `PORT` — HTTP server port (default: `8080`)

**Frontend (via compose):**
- `NEXT_PUBLIC_BACKEND_BASE` — Backend URL (e.g., `http://backend-dev:8080`)
- `API_TOKEN` — Token for backend API calls (server-side only, no `NEXT_PUBLIC_` prefix)

## Important Patterns

### Event Deduplication
Backend uses `processed_events` table with `INSERT OR IGNORE`. Always mark events as processed BEFORE side-effects:
```go
ok, err := store.TryMarkEventProcessed(eventID, ts)
if !ok { return } // Already processed
// Now safe to process
```

### User Caching
- **Backend:** `user_cache` table stores Slack user ID → name/avatar mapping
- **Frontend:** localStorage cache with 7-day TTL
- Purpose: Show names for deactivated Slack users

### Database Migrations
`store.go` runs migrations on startup. Schema changes need migration logic in `migrate()`.

## Testing

```bash
just test                    # Run all tests via Docker
cd backend/bot && go test ./...  # Run backend tests locally
```

## Debugging

1. Check logs: `just dev-logs`
2. Backend shell: `just shell-backend`
3. Frontend shell: `just shell-frontend`
4. Reset DB: `rm backend/bot/data/bot.db && just dev-restart`

## Slack Setup

1. Create app at https://api.slack.com/apps
2. Enable Socket Mode
3. Bot Token Scopes: `channels:history`, `groups:history`, `im:history`, `mpim:history`, `users:read`, `chat:write`
4. App-Level Token: `connections:write` scope
5. Invite bot to channels to track

## Guidelines for AI Assistants

- Run `just test` before proposing behavior changes
- Preserve deduplication semantics when modifying event handling
- Add tests for new functionality in `backend/bot/*_test.go`
- Frontend uses HSL CSS variables for theming — maintain consistency
- Keep PRs focused: one feature/fix per change
