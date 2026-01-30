# Technology Stack

**Analysis Date:** 2026-01-30

## Languages

**Primary:**
- TypeScript 5.9.3 - Frontend (React/Next.js components and API routes)
- Go 1.25 - Backend bot service (Slack integration and REST API)

**Secondary:**
- JavaScript - Build and configuration files

## Runtime

**Frontend:**
- Node.js (via Bun package manager) - Development and build
- Browser runtime - Client-side React execution

**Backend:**
- Go 1.25 runtime
- Alpine Linux 3.20 - Docker runtime environment

**Package Managers:**
- Bun 1.3.0+ - Frontend package management
- Lockfile: `bun.lock` present
- Go modules - Backend dependency management
- Lockfile: `go.sum` present

## Frameworks

**Frontend Core:**
- Next.js 15.5.6 - React meta-framework with App Router and Server Components
- React 19.2.0 - UI component library
- React DOM 19.2.0 - React rendering

**Frontend Styling:**
- Tailwind CSS 4.1.14 - Utility-first CSS framework
- @tailwindcss/postcss 4.1.14 - PostCSS plugin for Tailwind
- PostCSS 8.4.24 - CSS transformation pipeline
- Autoprefixer 10.4.14 - Vendor prefix automation

**Frontend UI Components:**
- lucide-react 0.546.0 - Icon component library
- react-datepicker 5.1.0 - Date range picker component

**Backend Core:**
- Standard Go library (net/http) - HTTP server and routing
- github.com/slack-go/slack v0.17.3 - Slack API client and WebSocket support
- github.com/mattn/go-sqlite3 v1.14.32 - SQLite database driver

**Monitoring & Observability:**
- github.com/prometheus/client_golang v1.23.2 - Prometheus metrics collection and HTTP handler
- github.com/rs/zerolog v1.34.0 - Structured JSON logging

## Key Dependencies

**Frontend Critical:**
- Next.js - Server-side rendering, API routes, static optimization
- React - Component model and state management
- Tailwind CSS - Design system and responsive styling

**Backend Critical:**
- slack-go/slack - Slack API integration (events, messages, user info, socket mode)
- go-sqlite3 - Local persistent data storage
- Prometheus client - Metrics exposure for monitoring
- zerolog - Structured logging for debugging and observability

## Configuration

**Frontend:**
- TypeScript config: `frontend/project/tsconfig.json`
  - Target: ES2017
  - Strict null checks enabled
  - Path aliases: `@/*` -> `./src/*` (not used in current structure)
  - JSX: preserve (Next.js handles transformation)

- Next.js config: `frontend/project/next.config.js`
  - React strict mode enabled for development warnings

- Tailwind config: `frontend/project/tailwind.config.js`
  - Dark mode: class-based toggle
  - Content: `./app/**/*` and `./components/**/*`
  - Custom color theme with HSL variables (background, foreground, primary, secondary, accent, etc.)
  - Custom border radius scale

- PostCSS config: `frontend/project/postcss.config.mjs`
  - Plugins: @tailwindcss/postcss, autoprefixer

- ESLint config: `frontend/project/eslint.config.mjs`

**Backend:**
- Dockerfile: `backend/bot/Dockerfile`
  - Multi-stage build: Go 1.25 builder -> Alpine 3.20 runtime
  - Volume mount: `/data` for SQLite database persistence
  - Exposed port: 8080
  - Healthcheck: wget to `/api/health` endpoint

- Docker Compose: `backend/docker-compose.yml`
  - Service: beerbot_backend
  - Port mapping: 8080:8080
  - Resource limits: 256M memory, 0.5 CPU
  - Resource reservations: 128M memory, 0.25 CPU
  - Health checks enabled with 30s interval

## Environment Variables

**Frontend (Next.js):**
- `NEXT_PUBLIC_BACKEND_BASE` - Backend API base URL (default: `http://localhost:8080`)
- `API_TOKEN` - API authentication token (default: `my-secret-token`) - **exposed to frontend**

**Backend (Go):**
- `BOT_TOKEN` - Slack bot token (xoxb-...)
- `APP_TOKEN` - Slack app-level token (xapp-...) for Socket Mode
- `CHANNEL` - Slack channel ID to monitor
- `API_TOKEN` - API authentication token for REST endpoints
- `DB_PATH` - SQLite database file path
- `ADDR` - HTTP listen address (default: `:8080`)
- `MAX_PER_DAY` - Maximum beers a user may give per day (default: 10)
- `EMOJI` - Custom emoji to track (default: `:beer:`)

## Build & Development Commands

**Frontend:**
```bash
bun install              # Install dependencies
bun run dev              # Start development server with turbopack
bun run build            # Build for production
bun start                # Start production server
bun run lint             # Run ESLint
```

**Backend:**
```bash
go mod download          # Download dependencies
go build -o ./bot .      # Build binary
go test ./...            # Run tests
docker build ./bot       # Build Docker image
```

## Platform Requirements

**Development:**
- Frontend: Node.js compatible runtime (Bun 1.3.0+), modern web browser
- Backend: Go 1.25 toolchain

**Production:**
- Frontend: Next.js compatible hosting (Node.js runtime or edge runtime)
- Backend: Docker or standalone Alpine/Linux environment with SQLite support
- Slack workspace with custom app configured for Socket Mode

**Network Requirements:**
- Backend requires outbound HTTPS to Slack API (api.slack.com)
- Frontend makes requests to backend API
- Frontend health check: GET `/api/health` on backend

---

*Stack analysis: 2026-01-30*
