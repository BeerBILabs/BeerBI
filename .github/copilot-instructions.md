# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Monorepo for BeerBot — a Slack bot that tracks team appreciation through virtual beer giving.

| Component | Path | Tech Stack |
|-----------|------|------------|
| Backend | `backend/bot/` | Go 1.27, SQLite, Slack Socket Mode, zerolog |
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
just seed             # Fill backend/bot/data/bot.db with synthetic data (ROWS=200000)
just bench            # Backend query benchmarks (BEERBI_BENCH_ROWS=1000000)
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

`ActivityHeatmap` is split so hovering never re-renders the grid: `heatmapModel.ts` builds the cells once (`Map` keyed by week and weekday, O(days)), `HeatmapGrid` is a `React.memo` component fed only the model and stable callbacks, and `HeatmapTooltip` owns the tooltip state and is driven through a ref.

Bundle: the Recharts chart components are loaded with `next/dynamic` (`ssr: false`, `ChartSkeleton` as placeholder) in `app/analytics/page.tsx`; do not import `components/charts/index.ts` from other pages. `DateRangePicker` uses two native `<input type="date">` (no date library). The production build uses `output: 'standalone'` (the release image copies `.next/standalone`; `bun run start` still works but prints a warning, the image runs `server.js`). Run `bun run analyze` in `frontend/project` for a bundle report (`@next/bundle-analyzer`, `ANALYZE=true`, webpack build, output in `.next/analyze/`).

### Theme System

`app/globals.css` defines HSL CSS custom properties for light/dark modes. Beer-themed golden accent (`#d4a84b`). Theme is stored in the `theme` cookie (`light`/`dark`, one year, `SameSite=Lax`). It is written client-side only (`ThemeProvider` on the first visit, `useTheme().setTheme` on toggle; there is no server action) and read by the blocking `ThemeScript` and, for now, server-side in `layout.tsx` to prevent FOUC. All color references should use the CSS variables (e.g., `hsl(var(--primary))`), not hardcoded values.

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
`store.go` → `migrate()` runs on startup. Schema changes must be additive/non-destructive and added to the migration function. Every step is idempotent: create/upgrade tables (a legacy `beers` table without `UNIQUE` is rebuilt in one transaction), drop obsolete indexes, run versioned data steps gated by `PRAGMA user_version` (v1: normalise `beers.ts_rfc` to RFC3339 UTC, logs `rows_changed`), create the indexes listed in `storeIndexes`, then `ANALYZE` if indexes were created or statistics are missing.

### Date Filters
All date filters on `beers` are half-open ranges `ts_rfc >= ? AND ts_rfc < ?` built by `dateBounds(start, end)` (inclusive calendar days; the upper bound is the bare date of the day after `end`) or `yearBounds`. Never wrap `ts_rfc` in a function in a `WHERE` clause: it disables the indexes. `store_plan_test.go` fails if a date-bounded query does not `SEARCH` an index. New date-bounded queries go into the query constants in `store.go` and into `planQueries()` in that test.

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

| Where | Command | What runs |
|---|---|---|
| Everything | `just test` | `compose.test.yaml`: backend then frontend, in containers |
| Backend | `just test-backend` | `gofmt -l` (must be empty), `go vet`, `go test -race -cover` (Dockerfile stage `test`) |
| Frontend | `just test-frontend` | `bun run lint`, `bun run typecheck`, `bun run test` (Vitest), `bun run build` (Dockerfile stage `test`) |
| CI | `.github/workflows/ci.yml` | Same checks plus `golangci-lint` and coverage thresholds; runs on push to `main`, pull requests and (via `workflow_call`) before the tag-triggered Docker builds |

- Backend tests use `t.TempDir()` for SQLite files. Test files are `backend/bot/*_test.go`. Lint config: `backend/bot/.golangci.yml`.
- Frontend tests live next to the code in `__tests__/` folders (`lib/__tests__`, `components/__tests__`) and use Vitest, jsdom and Testing Library. Config: `frontend/project/vitest.config.ts`. `TZ=Europe/Berlin` is forced for every run (`test/global-setup.ts`) so timezone bugs also fail on UTC runners; never format dates with `toISOString()`, use `formatLocalDate` from `lib/dateUtils.ts`.
- `tsconfig.json` has `"strict": true`; keep it that way (no `any` to silence errors, no `@ts-ignore`).
- ESLint 10 (flat config `frontend/project/eslint.config.mjs`) runs with `--max-warnings 0`: typescript-eslint recommended, `@next/eslint-plugin-next` core-web-vitals and `eslint-plugin-react-hooks` recommended (rules-of-hooks, exhaustive-deps and the React Compiler rules such as `set-state-in-effect`). Fix findings instead of disabling them; a disable needs a reason (`// TODO(owner): ...`). `eslint-config-next` is deliberately not used.
- TypeScript runs in two versions, as recommended for TS 7: `bun run typecheck` uses TypeScript 7 (`@typescript/native`, the native `tsc`), while the `typescript` package is aliased to `@typescript/typescript6` because Next's build-time type check and typescript-eslint need the TS 6 JavaScript API. Re-resolving the lockfile needs Bun >= 1.4 (older Bun mis-resolves the nested `npm:` alias). Drop the alias once TS 7.1 ships its API and the tools support it.
- Benchmarks: `bench_test.go` runs the analytics queries against a seeded database (`internal/seed`, 500 fake users, 6 years). `go test -run '^$' -bench . -benchmem ./...`; `BEERBI_BENCH_ROWS` sets the size (default 1,000,000, 20,000 with `-short`). Record before/after numbers in the PR when touching queries, indexes or the driver.
- Seed a dev database: `just seed` (or `go run ./cmd/seed -db data/bot.db -rows 200000` in `backend/bot`). It only creates the `beers` table; the bot adds indexes on its next start. All user IDs are fake (`U000000` …).
- Coverage minimums are the env vars `BACKEND_COVERAGE_THRESHOLD` and `FRONTEND_COVERAGE_THRESHOLD` at the top of `ci.yml` (the frontend value is also the default in `vitest.config.ts`). Raise them when tests are added.
- Do not use `docker compose -f compose.test.yaml up --abort-on-container-exit`: the first service to finish stops the other. Use `just test`.
- Without `just`: `docker compose -f compose.test.yaml run --build --rm backend-test` and `... frontend-test`. Host Go cannot link CGO (sqlite3) on the maintainer's Mac, so run Go inside Docker.

## Guidelines

- Run `just test` before proposing behavior changes
- Preserve deduplication semantics when modifying event handling
- Add tests for new backend functionality in `backend/bot/*_test.go`
- Use HSL CSS variables for all colors — never hardcode color values
- Keep PRs focused: one feature/fix per change
