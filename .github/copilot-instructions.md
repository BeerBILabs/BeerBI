# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Monorepo for BeerBot — a Slack bot that tracks team appreciation through virtual beer giving.

| Component | Path | Tech Stack |
|-----------|------|------------|
| Backend | `backend/bot/` | Go 1.25, SQLite, Slack Socket Mode, zerolog |
| Frontend | `frontend/project/` | Next.js 16 (App Router), React 19, Tailwind 4, Recharts, Bun |

## Commands

All commands use [just](https://github.com/casey/just) from the repo root:

```bash
just dev              # Start dev environment (Docker, hot reload)
just dev-down         # Stop dev environment
just dev-restart      # Restart services
just test             # Run all tests via Docker (compose.test.yaml)
just dev-logs         # Tail all logs
just dev-logs-backend # Backend logs only
just dev-logs-frontend # Frontend logs only
just shell-backend    # Shell into backend container
just shell-frontend   # Shell into frontend container
just status           # Show running services
just clean            # Remove containers and volumes
```

Run backend tests locally (outside Docker):
```bash
cd backend/bot && go test ./...
```

## Architecture

### Data Flow

```
Slack message → Socket Mode → EventProcessor (events.go)
  → Detect beer emoji → Parse @mentions → Check daily limit
  → TryMarkEventProcessed() for deduplication
  → Store in SQLite (beers table)

Frontend page → /api/proxy/[...path] (Next.js server-side)
  → Injects Authorization: Bearer {API_TOKEN}
  → Backend REST handler → SQLite query → JSON response
```

The frontend never talks to the backend directly from the browser. All requests route through the Next.js proxy at `app/api/proxy/[...path]/route.ts`, which injects the `API_TOKEN` server-side.

### Backend (`backend/bot/`)

Single Go binary with these layers:
- **main.go** — HTTP server setup, Slack Socket Mode connection, route registration, graceful shutdown
- **events.go** — `EventProcessor`: beer emoji detection, mention parsing (`<@([A-Z0-9]+)>`), daily limit enforcement (default 10/user/day), self-gift prevention
- **store.go** — SQLite database: schema migrations in `migrate()`, all queries. Tables: `beers`, `processed_events`, `user_cache`, `emoji_counts`
- **handlers.go** — REST API endpoints (see below)
- **slack.go** — `SlackConnectionManager`: auto-reconnect with exponential backoff, health monitoring every 30s
- **middleware.go** — Bearer token auth (constant-time comparison)

### API Endpoints

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `GET /api/given?user=&start=&end=` | Yes | Beers given by user in date range |
| `GET /api/received?user=&start=&end=` | Yes | Beers received by user in date range |
| `GET /api/user?user=` | Yes | Slack user info (with cache fallback) |
| `GET /api/givers` | No | All giver IDs |
| `GET /api/recipients` | No | All recipient IDs |
| `GET /api/stats/timeline?start=&end=&granularity=` | Yes | Activity timeline (day/week/month) |
| `GET /api/stats/quarterly?start_year=&end_year=` | Yes | Quarterly aggregation |
| `GET /api/stats/top?start=&end=&limit=` | Yes | Top givers/recipients |
| `GET /api/stats/heatmap?start=&end=` | Yes | Calendar heatmap data |
| `GET /api/stats/pairs?start=&end=&limit=` | Yes | Giver→recipient network data |
| `GET /api/health` | No | Health check |
| `GET /healthz` | No | Docker healthcheck |
| `GET /metrics` | No | Prometheus metrics |

### Frontend (`frontend/project/`)

Three main pages:
- `/` — Leaderboard (`UsersPage.tsx` → `Leaderboard.tsx`) with date range filters
- `/analytics` — Dashboard with timeline, top users, heatmap, network graph, quarterly charts
- `/rankings` / `/rankings/all` / `/rankings/[year]/[quarter]` — Quarterly rankings with rank change indicators

Chart components live in `components/charts/`. Shared utilities in `lib/` (userCache, quarters, chartTheme).

### Theme System

`app/globals.css` defines HSL CSS custom properties for light/dark modes. Beer-themed golden accent (`#d4a84b`). Theme is stored in a cookie (read server-side in layout.tsx to prevent FOUC). All color references should use the CSS variables (e.g., `hsl(var(--primary))`), not hardcoded values.

## Important Patterns

### Event Deduplication
Always mark events as processed BEFORE side-effects:
```go
ok, err := store.TryMarkEventProcessed(eventID, ts)
if !ok { return } // Already processed
// Now safe to process
```

### User Caching (two layers)
- **Backend:** `user_cache` SQLite table — survives restarts, allows showing deactivated Slack users
- **Frontend:** localStorage with 7-day TTL (`lib/userCache.ts`) — reduces API calls, enables offline viewing of previously loaded data

### Database Migrations
`store.go` → `migrate()` runs on startup. Schema changes must be additive/non-destructive and added to the migration function.

### Frontend Data Fetching
Components fetch via `/api/proxy/{endpoint}`. The `Leaderboard.tsx` component uses a 5-concurrent-worker pattern for parallel user stat requests.

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

## Testing

Backend tests use temporary SQLite files (cleaned up with `defer os.Remove()`). Test files are in `backend/bot/*_test.go`. The frontend is validated via build in `compose.test.yaml`.

## Guidelines

- Run `just test` before proposing behavior changes
- Preserve deduplication semantics when modifying event handling
- Add tests for new backend functionality in `backend/bot/*_test.go`
- Use HSL CSS variables for all colors — never hardcode color values
- Keep PRs focused: one feature/fix per change
