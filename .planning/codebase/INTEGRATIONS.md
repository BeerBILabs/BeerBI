# External Integrations

**Analysis Date:** 2026-01-30

## APIs & External Services

**Slack API:**
- SDK/Client: `github.com/slack-go/slack` v0.17.3
- Connection: Socket Mode (WebSocket) for real-time events
- Auth: Two tokens required
  - Bot token: `BOT_TOKEN` env var (format: `xoxb-...`)
  - App token: `APP_TOKEN` env var (format: `xapp-...`)
- Features used:
  - Socket Mode events (message events)
  - User info retrieval (`GetUserInfo`)
  - Message posting (`PostMessage`)
  - Events API for message handling
  - Auth testing (`AuthTestContext`)

**Backend API (Internal):**
- REST endpoints exposed on `:8080` by bot service
- Consumed by frontend via Next.js proxy route
- Endpoints:
  - `GET /api/given` - Beer count given by user (query: user, day/start/end)
  - `GET /api/received` - Beer count received by user (query: user, day/start/end)
  - `GET /api/user` - User info from Slack (query: user)
  - `GET /api/givers` - List of all givers
  - `GET /api/recipients` - List of all recipients
  - `GET /healthz` - Health check (no auth)
  - `GET /api/health` - Detailed health check (no auth)
  - `GET /metrics` - Prometheus metrics (no auth)
- Auth: Bearer token via `Authorization` header
  - Token value: `API_TOKEN` env var
  - Enforced via `authMiddleware` in `backend/bot/main.go`

## Data Storage

**Database:**
- Type: SQLite 3
- Location: File path configurable via `DB_PATH` env var
- Client: github.com/mattn/go-sqlite3 v1.14.32 (cgo wrapper)
- Implementation: `backend/bot/store.go` (SQLiteStore)
- Tables:
  - `beers` - Core beer transfer records
    - Columns: id, giver_id, recipient_id, ts (Slack ts), ts_rfc (parsed time), count
    - Indexes: giver_id, recipient_id, ts_rfc for date range queries
    - Unique constraint: (giver_id, recipient_id, ts)
  - `processed_events` - Deduplication tracking
    - Columns: id, event_id, ts
    - Unique constraint: event_id
  - `emoji_counts` - Auxiliary emoji tracking
    - Columns: id, user_id, emoji, count
    - Unique constraint: (user_id, emoji)

**File Storage:**
- Not used - all state in SQLite

**Caching:**
- No distributed caching layer
- Frontend uses in-memory React state and request deduplication

## Authentication & Identity

**Slack OAuth:**
- Provider: Slack OAuth 2.0
- Implementation: Custom Slack app configuration
- Token types: Bot token + App-level token for Socket Mode
- User identification: Slack user IDs (e.g., U12345678)
- No user authentication in frontend - trusts backend Slack validation

**API Authentication:**
- Provider: Custom Bearer token
- Implementation: `authMiddleware` in `backend/bot/main.go` (line 607-628)
- Token validation: Simple string comparison against `API_TOKEN` env var
- Scope: Protects `/api/given`, `/api/received`, `/api/user`, `/api/givers`, `/api/recipients`
- Public endpoints: `/healthz`, `/api/health`, `/metrics`

## Frontend Backend Communication

**Proxy Architecture:**
- Frontend route: `frontend/project/app/api/proxy/[...path]/route.ts`
- Purpose: Server-side proxy to forward requests to backend
- Token handling:
  - If client sends `Authorization` header, forwards it to backend
  - Otherwise uses server-side `API_TOKEN` env var as fallback
  - Logs auth source: 'client', 'HARDCODED', or 'none'
- Supports all HTTP methods: GET, POST, PUT, PATCH, DELETE, OPTIONS

**Endpoint Mapping:**
- Frontend: `/api/proxy/given` -> Backend: `http://localhost:8080/api/given`
- Frontend: `/api/proxy/received` -> Backend: `http://localhost:8080/api/received`
- Frontend: `/api/proxy/user` -> Backend: `http://localhost:8080/api/user`
- Frontend: `/api/proxy/givers` -> Backend: `http://localhost:8080/api/givers`
- Frontend: `/api/proxy/recipients` -> Backend: `http://localhost:8080/api/recipients`

**Backend URL Configuration:**
- Env var: `NEXT_PUBLIC_BACKEND_BASE` (frontend-side config)
- Default: `http://localhost:8080`
- Can point to external backend for production deployments

## Monitoring & Observability

**Metrics Collection:**
- Framework: Prometheus
- Package: github.com/prometheus/client_golang v1.23.2
- Endpoint: `GET /metrics` on backend
- Metrics:
  - `bwm_messages_processed_total` - Counter of processed messages by channel
- Prometheus HTTP handler: `promhttp.Handler()` mounted at `/metrics`

**Logging:**
- Framework: zerolog (JSON structured logging)
- Package: github.com/rs/zerolog v1.34.0
- Config: `backend/bot/main.go` (lines 263-266)
  - Time format: RFC3339
  - Output: stderr with console writer
  - Structured fields: timestamp, level, message
- Usage: Error, info, warn levels for:
  - Connection status changes
  - Beer transactions
  - Event processing
  - API requests (via console.log in proxy route)

**Health Checks:**
- Public endpoint: `GET /healthz` - Simple 200 OK response
- Extended health: `GET /api/health` - JSON response with:
  - status: "healthy" or "degraded"
  - service: "beerbot-backend"
  - slack_connected: boolean
  - timestamp: RFC3339 format
  - slack_connection_error: included if check_slack=true query param
- Docker health check: wget to `/api/health` every 30s

## Event Processing

**Slack Events:**
- Transport: Socket Mode (WebSocket)
- Event type: Events API -> Message events
- Scope: Only messages in configured `CHANNEL`
- Filtering:
  - Only user-generated messages (ev.User != "")
  - Ignores message subtypes (edits, bot messages)
- Deduplication: `processed_events` table with envelope_id or stable message key

**Message Parsing:**
- Regex for user mentions: `<@([A-Z0-9]+)>` - extracts recipient IDs
- Regex for emoji: configurable (default `:beer:`)
- Logic: Associates each emoji with last preceding mention
- Rate limiting: Daily limit per giver via `MAX_PER_DAY` env var

## Webhooks & Callbacks

**Incoming:**
- Slack Event API via Socket Mode (not traditional webhooks)
- No other incoming webhook endpoints

**Outgoing:**
- None - backend is event consumer only

## Environment Configuration

**Required env vars for Backend:**
- `BOT_TOKEN` - Slack bot token
- `APP_TOKEN` - Slack app-level token
- `CHANNEL` - Slack channel ID to monitor

**Optional env vars for Backend:**
- `DB_PATH` - SQLite DB path (default: memory or current dir)
- `API_TOKEN` - API auth token (default: none, endpoints unprotected if not set)
- `ADDR` - HTTP listen address (default: `:8080`)
- `MAX_PER_DAY` - Daily beer limit (default: 10)
- `EMOJI` - Custom emoji to track (default: `:beer:`)

**Required env vars for Frontend:**
- None - all have safe defaults

**Optional env vars for Frontend:**
- `NEXT_PUBLIC_BACKEND_BASE` - Backend URL (default: `http://localhost:8080`)
- `API_TOKEN` - Backend API token for proxy fallback

**Secrets location:**
- Backend: Environment variables (typically .env file or container secrets)
- Frontend: `API_TOKEN` should NOT be committed; use server-side only
- Development: Docker Compose or .env files
- Production: Container orchestration platform (Kubernetes secrets, etc.)

## Deployment Integration

**Docker:**
- Backend Dockerfile: `backend/bot/Dockerfile`
- Docker Compose files:
  - `backend/docker-compose.yml` - Production setup
  - `backend/docker-compose.dev.yml` - Development setup
  - `backend/docker-compose.test.yml` - Test setup

**CI/CD:**
- GitHub Actions workflows in `.github/workflows/` (not analyzed)

**Health Monitoring:**
- Docker health check: interval 30s, timeout 10s, 3 retries, 40s start period
- Endpoint: `http://localhost:8080/api/health`
- Frontend health route: `frontend/project/app/api/health/route.ts`

---

*Integration audit: 2026-01-30*
