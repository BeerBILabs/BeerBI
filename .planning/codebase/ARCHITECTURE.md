# Architecture

**Analysis Date:** 2026-01-30

## Pattern Overview

**Overall:** Distributed service architecture with React-based frontend and Go-based backend, connected via HTTP API proxy pattern.

**Key Characteristics:**
- Frontend proxies backend API requests to handle cross-origin and centralized authentication
- Backend uses Slack Socket Mode to listen for real-time events
- Separation of concerns: frontend for UI/UX, backend for event processing and data persistence
- State stored in SQLite with query optimization via indexes
- Metrics collection at HTTP handler level (Prometheus)

## Layers

**Frontend (Next.js):**
- Purpose: Render leaderboard UI, handle user interactions, proxy API requests
- Location: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project`
- Contains: React components, pages, API routes, utilities
- Depends on: Backend API at `process.env.NEXT_PUBLIC_BACKEND_BASE`
- Used by: Web browsers accessing the application

**Backend (Go):**
- Purpose: Slack event listening, beer count aggregation, HTTP API server
- Location: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot`
- Contains: HTTP handlers, event handlers, database layer, Slack connection management
- Depends on: SQLite database, Slack API (Socket Mode), internal metrics
- Used by: Frontend via `/api/*` routes, metrics scrapers

**Data Layer (SQLite):**
- Purpose: Persistent storage of beer transactions
- Location: Accessed via Store interface in `store.go`
- Contains: `beers` table (giver, recipient, timestamp, count), `processed_events` table (idempotency), `emoji_counts` table (stats)
- Depends on: `github.com/mattn/go-sqlite3`
- Used by: Backend handlers for queries and mutations

## Data Flow

**Event Ingestion Flow:**

1. Slack event occurs (reaction added to message)
2. Slack Socket Mode client receives event via WebSocket (`slack.go`)
3. `buildEventHandler()` in `slack.go` processes emoji reaction events
4. Event handler validates: correct emoji, correct channel, no duplicate processing
5. `TryMarkEventProcessed()` atomically checks and marks event as processed (prevents duplicates)
6. `AddBeer()` inserts/updates beer transaction in SQLite (`store.go`)
7. Metrics updated (`metrics.go`)

**Query Flow (Leaderboard):**

1. Frontend loads `UsersPage` component
2. User selects date range (quick preset or custom)
3. Component calls `/api/proxy/givers` and `/api/proxy/recipients` to fetch user lists
4. Next.js proxy route (`app/api/proxy/[...path]/route.ts`) intercepts request
5. Proxy adds API token from server environment (`process.env.API_TOKEN`)
6. Proxy forwards to backend HTTP server
7. Backend `newGiversHandler()` or `newRecipientsHandler()` queries store
8. SQLite returns distinct user IDs
9. Frontend then calls `/api/proxy/given?user=X&start=Y&end=Z` for each user
10. Backend `newGivenHandler()` queries `CountGivenInDateRange()`
11. Store queries indexed `beers` table with giver_id and ts_rfc date range
12. Results returned to frontend, rendered in sorted leaderboard

**State Management:**

- Frontend: React component state (useState for stats, names, avatars)
- Backend: SQLite (single source of truth), in-memory Slack connection state
- Shared state: None (unidirectional flow from backend to frontend)

## Key Abstractions

**Store Interface (Backend):**
- Purpose: Abstract database operations from HTTP handlers
- Examples: `store.go` defines `Store` interface; `SQLiteStore` implements it
- Pattern: Go interface for dependency injection; allows testing with mock stores

**Event Handler Factory (Backend):**
- Purpose: Encapsulate Slack event processing logic
- Examples: `buildEventHandler()` in `slack.go` returns closure capturing store, client, channel ID
- Pattern: Factory function returns event handler with dependencies bound

**SlackConnectionManager (Backend):**
- Purpose: Manage Socket Mode connection lifecycle with automatic reconnection
- Examples: `slack.go`, synchronized with RWMutex for concurrent access
- Pattern: Manager struct with exported GetClient/GetSocketClient, private state mutations

**HTTP Proxy Handler (Frontend):**
- Purpose: Centralize backend routing and add server-side authentication
- Examples: `app/api/proxy/[...path]/route.ts` uses catch-all dynamic routing
- Pattern: Catch-all Next.js route handler forwarding to backend with injected auth

**Concurrent Worker Pool (Frontend):**
- Purpose: Limit concurrent requests while fetching user data
- Examples: `UsersList.tsx` creates 5 worker tasks from queue
- Pattern: Shared index variable with loop counter, cancellation flag

## Entry Points

**Frontend Web Entry:**
- Location: `app/page.tsx`
- Triggers: Browser navigates to root path
- Responsibilities: Renders UsersPage component (wrapper)

**Frontend API Proxy:**
- Location: `app/api/proxy/[...path]/route.ts`
- Triggers: Any request to `/api/*` path
- Responsibilities: Forward request to backend with injected API token, log request/response

**Frontend Health Check:**
- Location: `app/api/health/route.ts`
- Triggers: `/api/health` requests
- Responsibilities: Return health status

**Backend Main:**
- Location: `bot/main.go`
- Triggers: Container startup
- Responsibilities: Parse config, initialize database, start HTTP server, connect Slack Socket Mode, handle graceful shutdown

**Backend HTTP Handlers:**
- Locations: `bot/http_handlers.go` (all handler registrations in `newMux()`)
- Handlers:
  - `/healthz` - Quick liveness check
  - `/metrics` - Prometheus metrics
  - `/api/given` - Count beers given by user in date range (auth required)
  - `/api/received` - Count beers received by user in date range (auth required)
  - `/api/user` - Get user info from Slack (auth required)
  - `/api/givers` - List all users who gave beer (no auth)
  - `/api/recipients` - List all users who received beer (no auth)
  - `/api/health` - Slack connection health

## Error Handling

**Strategy:** Fail-fast with context; log then return HTTP error status.

**Patterns:**

- Frontend: Try-catch async/await with `console.error()` fallback, returns count of 0 on error
- Backend: Return HTTP status codes (400 for validation, 401 for auth, 500 for internal errors)
- Backend: Structured logging with `zerolog` at appropriate levels (info, warn, error)
- Database: Transaction-like safety via unique constraints and atomic flag checks (`TryMarkEventProcessed`)

## Cross-Cutting Concerns

**Logging:**
- Frontend: `lib/logger.ts` provides leveled logger (debug/info/warn/error) with JSON metadata
- Backend: `zerolog` with console writer to stderr, RFC3339 timestamps, fields via context
- Proxy handler logs request details: method, path, query params, auth source, duration

**Validation:**
- Frontend: Type checking via TypeScript, required query params checked in handlers
- Backend: Query params parsed and validated (e.g., `parseDateRangeFromParams()` checks date format)
- Backend: Bearer token validation in `authMiddleware()`

**Authentication:**
- Frontend: Uses hardcoded `API_TOKEN` injected from server environment
- Backend: Bearer token validation; token passed via Authorization header by proxy
- No user authentication (assumes all requests in trusted Slack workspace)

**Metrics:**
- Backend: Prometheus metrics for HTTP request count/duration by endpoint and method
- Backend: Slack connection status tracked (connected/disconnected) and exposed in `/api/health`
- Timing: Captured via `time.Now()` in handlers, logged to console

---

*Architecture analysis: 2026-01-30*
