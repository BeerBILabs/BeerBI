# External Integrations

**Analysis Date:** 2026-01-30

## APIs & External Services

**Backend API:**
- BeerBot Backend Service - Main data and statistics API
  - SDK/Client: Native `fetch` via Next.js proxy
  - Base URL: `NEXT_PUBLIC_BACKEND_BASE` environment variable (default: `http://localhost:8080`)
  - Endpoints called:
    - `GET /api/givers` - Fetch list of beer givers
    - `GET /api/recipients` - Fetch list of beer recipients
    - `GET /api/given?user={user}&[day|start]={date}&[end={date}]` - Get beer count given by user
    - `GET /api/received?user={user}&[day|start]={date}&[end={date}]` - Get beer count received by user
    - `GET /api/user?user={user}` - Fetch user profile (real_name, profile_image)
    - `GET /healthz` - Health check endpoint
  - Auth: Bearer token via `Authorization` header
  - Implementation: `app/api/proxy/[...path]/route.ts` (proxies all requests)

**Health Check:**
- Backend health endpoint at `/healthz`
- Used by: `app/api/health/route.ts`
- Response validation: HTTP 2xx indicates healthy, non-2xx or timeout indicates unhealthy

## Data Storage

**Databases:**
- Not detected - All data comes from backend API

**File Storage:**
- Local filesystem - Profile images (user avatars)
  - Served via: Next.js Image component with `unoptimized` flag
  - Source: `profile_image` field from user API endpoint
  - Implementation: `components/UsersList.tsx` (lines 159-166)

**Caching:**
- Not detected - Browser-based state management only

## Authentication & Identity

**Auth Provider:**
- Custom - Bearer token-based authentication
  - Implementation: API proxy in `app/api/proxy/[...path]/route.ts`
  - Header: `Authorization: Bearer {API_TOKEN}`
  - Token source: `API_TOKEN` environment variable (fallback: `my-secret-token`)
  - Fallback behavior: Uses hardcoded token if no header from client
  - Client propagation: Accepts Authorization header from frontend, forwards to backend

## Monitoring & Observability

**Error Tracking:**
- Not detected - No external error tracking service

**Logs:**
- Console-based logging via custom logger in `lib/logger.ts`
- Levels: debug, info, warn, error
- Environment-aware: debug in development, info in production
- Structured logging with JSON metadata
- Timing instrumentation via `withTiming()` wrapper function
- API proxy logging: Requests, responses, and errors logged to console
- Health check logging: Success, failure, and timeout scenarios

## CI/CD & Deployment

**Hosting:**
- Not detected - Configuration supports any Node.js/Bun compatible host

**CI Pipeline:**
- Not detected - No CI configuration files present

## Environment Configuration

**Required env vars:**
- `NEXT_PUBLIC_BACKEND_BASE` - Backend API URL (exposed to frontend, default: `http://localhost:8080`)
- `API_TOKEN` - Server-side authentication token (kept secret)
- `NODE_ENV` - Runtime environment indicator (development/production)

**Secrets location:**
- `.env` and `.env.local` files (not committed, per `.gitignore`)
- Server-side secrets: `API_TOKEN` kept in `.env`, never exposed to client
- Public config: `NEXT_PUBLIC_*` variables safe for browser

## Webhooks & Callbacks

**Incoming:**
- Not detected

**Outgoing:**
- Not detected - One-way API communication only

---

*Integration audit: 2026-01-30*
