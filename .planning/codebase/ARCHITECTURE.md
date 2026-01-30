# Architecture

**Analysis Date:** 2026-01-30

## Pattern Overview

**Overall:** Monorepo with decoupled frontend and backend services communicating via REST API. The backend is a Slack integration bot with persistence, while the frontend is a Next.js leaderboard UI.

**Key Characteristics:**
- Monorepo structure with independent deployment targets (frontend and backend)
- Backend operates as a Slack socket mode bot with HTTP REST API server
- Frontend uses Next.js server components with client-side state management
- API proxy pattern for secure token injection and backend communication
- SQLite for persistence with idempotent event processing

## Layers

**Frontend Presentation Layer:**
- Purpose: Render interactive leaderboard UI with date filtering
- Location: `frontend/project/app/page.tsx`, `frontend/project/components/`
- Contains: React Server Components, Client Components, UI components, theme management
- Depends on: Next.js runtime, React hooks, lucide-react icons, react-datepicker
- Used by: End users via browser

**Frontend API Integration Layer:**
- Purpose: Bridge frontend to backend via proxy server, inject authentication token server-side
- Location: `frontend/project/app/api/proxy/[...path]/route.ts`, `frontend/project/app/api/health/route.ts`
- Contains: Next.js API routes, token injection middleware, request forwarding logic
- Depends on: `process.env.API_TOKEN`, `process.env.NEXT_PUBLIC_BACKEND_BASE`
- Used by: Frontend client components via `/api/proxy/*` endpoints

**Backend Slack Event Layer:**
- Purpose: Connect to Slack, receive events via socket mode, parse beer emoji mentions
- Location: `backend/bot/main.go` (lines 31-155: SlackConnectionManager, lines 418-556: event handler)
- Contains: Socket mode connection management with exponential backoff reconnection, event deduplication
- Depends on: Slack socket mode API, environment tokens (BOT_TOKEN, APP_TOKEN)
- Used by: Main process to receive and process real-time events

**Backend Data Persistence Layer:**
- Purpose: Store beer transactions and event deduplication
- Location: `backend/bot/store.go`
- Contains: SQLite schema management, beer counting queries, event idempotency tracking
- Depends on: SQLite database (`database/sql`), `github.com/mattn/go-sqlite3`
- Used by: Main process for all data operations

**Backend REST API Layer:**
- Purpose: Expose beer counts, user lists, and health status to frontend
- Location: `backend/bot/main.go` (lines 276-404: HTTP handlers and routes)
- Contains: Handlers for `/api/given`, `/api/received`, `/api/user`, `/api/givers`, `/api/recipients`, `/api/health`
- Depends on: Auth middleware, SQLite store, Slack API for user profiles
- Used by: Frontend proxy to fetch leaderboard data

## Data Flow

**Beer Gift Flow:**

1. User posts Slack message with mention `<@USERID>` and emoji (default `:beer:`)
2. Slack socket mode delivers message event to `SlackConnectionManager.processEvents()`
3. Event handler parses mentions and emojis using regex (`main.go` lines 468-494)
4. Daily limit checked against `store.CountGivenOnDate()` (line 505-519)
5. Beer transaction inserted via `store.AddBeer()` with unique constraint on `(giver_id, recipient_id, ts)` (line 542)
6. Event ID marked processed via `store.TryMarkEventProcessed()` to prevent duplicates (line 458)

**Leaderboard Display Flow:**

1. Frontend loads: `UsersPage.tsx` initializes with current year date range (lines 80-85)
2. useEffect fetches `/api/proxy/givers` and `/api/proxy/recipients` (lines 91-92)
3. Backend `/api/givers` and `/api/recipients` query distinct users from beers table (lines 348-374)
4. Frontend `UsersList` component fetches stats for each user via `/api/proxy/given` and `/api/proxy/received` with concurrent workers (lines 48-76)
5. Backend handlers count beers in date range using `CountGivenInDateRange()` and `CountReceivedInDateRange()` (lines 283-322)
6. User details fetched via `/api/proxy/user` using Slack API `GetUserInfo()` (line 329)
7. Leaderboard sorted by count descending, top 100 shown with avatars (lines 135-139)

**State Management:**

**Frontend:**
- `UsersPage` manages: `range` (start/end dates), `givers` and `recipients` lists
- `UsersList` manages: `stats` (counts per user), `names` (real names), `avatars` (profile images)
- State persistence: None (resets on page load; theme persists via localStorage)

**Backend:**
- Slack connection state: Maintained in `SlackConnectionManager` with mutex protection
- Database state: SQLite tables for `beers`, `processed_events`, `emoji_counts`
- Event deduplication: `processed_events` table prevents double-processing via `TryMarkEventProcessed()`

## Key Abstractions

**SlackConnectionManager:**
- Purpose: Abstract away Slack socket mode lifecycle and reconnection logic
- Examples: `backend/bot/main.go` lines 31-155
- Pattern: Connection pool with exponential backoff retry (max 5 minutes), health monitoring via `TestConnection()`

**SQLiteStore:**
- Purpose: Abstract database operations and schema migrations
- Examples: `backend/bot/store.go` lines 10-160 (schema), 162-329 (query methods)
- Pattern: Idempotent migrations with ALTER TABLE support, parameterized queries for safety

**Proxy API Pattern:**
- Purpose: Inject server-side secrets (API_TOKEN) into backend requests without exposing to client
- Examples: `frontend/project/app/api/proxy/[...path]/route.ts` lines 5-63
- Pattern: Catch-all route handler with request/response forwarding

**User Interface Components:**
- Purpose: Reusable, typed React components with clear responsibilities
- Examples: `UsersList` (data fetching + rendering), `DateRangePicker` (date selection), `ThemeProvider` (theme management)
- Pattern: Separation of "use client" components from server components, prop-based composition

## Entry Points

**Frontend Entry Point:**
- Location: `frontend/project/app/page.tsx`
- Triggers: Browser navigation to `/`
- Responsibilities: Render root `UsersPage` component (server component wrapper)

**Frontend API Proxy Entry Point:**
- Location: `frontend/project/app/api/proxy/[...path]/route.ts`
- Triggers: Client-side fetch to `/api/proxy/*`
- Responsibilities: Forward request to backend, inject authorization header, transform response

**Backend Entry Point:**
- Location: `backend/bot/main.go` (function main, line 202)
- Triggers: Container/process startup
- Responsibilities: Parse config, initialize DB and Slack manager, start HTTP server and event loop

**Slack Event Processing Entry Point:**
- Location: `backend/bot/main.go` (eventHandler function, lines 419-556)
- Triggers: Slack socket mode event delivery
- Responsibilities: Parse message event, extract mentions and emojis, validate daily limits, persist beer transaction

## Error Handling

**Strategy:** Explicit error propagation with graceful degradation. Database errors logged and returned to API caller. Slack connection errors trigger exponential backoff reconnection.

**Patterns:**

- **API Errors:** HTTP status codes (400 for bad request, 401 for unauthorized, 500 for server error) with JSON error body
  - Example: `frontend/project/app/api/proxy/[...path]/route.ts` lines 44-62 (try-catch with 500 response)

- **Slack Connection Errors:** Logged and queued for retry with backoff
  - Example: `backend/bot/main.go` lines 123-147 (TestConnection failures increment reconnectCount)

- **Database Errors:** Wrapped with context, returned to caller
  - Example: `backend/bot/store.go` lines 245-258 (CountGivenInDateRange with error return)

- **Event Processing Errors:** Logged via zerolog, event not marked processed (will retry on redelivery)
  - Example: `backend/bot/main.go` lines 459, 508, 516 (zlog.Error() calls)

- **Frontend Fetch Errors:** Caught, user data defaults to username, avatars default to null
  - Example: `frontend/project/components/UsersList.tsx` lines 68-71, 111-115 (catch blocks set defaults)

## Cross-Cutting Concerns

**Logging:**

- **Backend:** Structured logging via `github.com/rs/zerolog` with console writer; log level INFO for beer events, ERROR for failures
  - Example: `backend/bot/main.go` lines 264-266 (zerolog setup), 545 (beer given log)

- **Frontend:** Console logging for errors only via `console.error()`
  - Example: `frontend/project/components/UsersList.tsx` lines 69, 112 (error logging)

**Validation:**

- **Backend:** Date range parsing from query params with error messages
  - Example: `backend/bot/main.go` lines 178-200 (parseDateRangeFromParams validates format)

- **Frontend:** Optional date range with fallback to current year, URL search param validation in proxy route

**Authentication:**

- **Backend:** Bearer token via Authorization header, middleware enforces token match against env var
  - Example: `backend/bot/main.go` lines 607-627 (authMiddleware checks Bearer token)

- **Frontend:** Token injected server-side in proxy route, never exposed to browser
  - Example: `frontend/project/app/api/proxy/[...path]/route.ts` lines 20, 27 (API_TOKEN from env)

**Rate Limiting:**

- **Backend:** Daily limit per user enforced via `CountGivenOnDate()` check before beer transaction (line 505-519)
  - Default: 10 beers/day (configurable via MAX_PER_DAY env var)

**Concurrency:**

- **Frontend:** Worker pool pattern for fetching stats/names in parallel (5 concurrent workers)
  - Example: `frontend/project/components/UsersList.tsx` lines 46-76 (worker function pattern)

- **Backend:** Goroutine-based async Slack event processing with mutex-protected connection state
  - Example: `backend/bot/main.go` lines 159-174 (processEvents loop), lines 57-74 (mutex for IsConnected)

---

*Architecture analysis: 2026-01-30*
