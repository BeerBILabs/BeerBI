# Codebase Structure

**Analysis Date:** 2026-01-30

## Directory Layout

```
/home/tdolfen/Projects/github.com/DanielWeeber/
├── BeerBot-backend/                    # Go backend service
│   ├── bot/                            # Main backend package
│   │   ├── main.go                     # Entry point, config, startup
│   │   ├── slack.go                    # Slack Socket Mode connection management
│   │   ├── http_handlers.go            # HTTP endpoint handlers and middleware
│   │   ├── store.go                    # SQLite store implementation
│   │   ├── metrics.go                  # Prometheus metrics setup
│   │   ├── data/                       # SQLite database files (gitignored)
│   │   └── *_test.go                   # Unit tests for store
│   ├── go.mod                          # Go module definition
│   ├── go.sum                          # Go dependency lockfile
│   ├── Dockerfile                      # Container build configuration
│   └── docker-compose*.yml             # Local dev/test orchestration
│
└── BeerBot-frontend/                   # Next.js frontend
    ├── project/                        # Next.js project root
    │   ├── app/                        # Next.js App Router
    │   │   ├── page.tsx                # Root page (/)
    │   │   ├── layout.tsx              # Root layout wrapper
    │   │   ├── not-found.tsx           # 404 handler
    │   │   └── api/
    │   │       ├── health/
    │   │       │   └── route.ts        # Health check endpoint
    │   │       └── proxy/
    │   │           └── [...path]/
    │   │               └── route.ts    # Backend proxy handler (catch-all)
    │   │
    │   ├── components/                 # React components (all client-side)
    │   │   ├── UsersPage.tsx           # Main page container with date range logic
    │   │   ├── UsersList.tsx           # Leaderboard list with concurrent fetching
    │   │   ├── DateRangePicker.tsx     # Date range picker UI
    │   │   ├── ThemeProvider.tsx       # Next-themes wrapper
    │   │   ├── ThemeToggle.tsx         # Light/dark mode toggle
    │   │   └── Logo.tsx                # App logo component
    │   │
    │   ├── lib/                        # Utilities and helpers
    │   │   └── logger.ts               # Frontend logging utility
    │   │
    │   ├── public/                     # Static assets (favicon, logo)
    │   ├── styles/                     # CSS (Tailwind imports)
    │   ├── package.json                # Node.js dependencies
    │   ├── tsconfig.json               # TypeScript configuration
    │   ├── next.config.js              # Next.js build configuration
    │   ├── tailwind.config.js          # Tailwind CSS configuration
    │   ├── postcss.config.mjs          # PostCSS pipeline
    │   └── eslint.config.mjs           # ESLint rules
    │
    └── docker-compose*.yml             # Container orchestration
```

## Directory Purposes

**BeerBot-backend/bot/:**
- Purpose: Core business logic - Slack event listening and HTTP API server
- Contains: Go source files, tests, database migrations (implicit in store.go)
- Key files: `main.go`, `slack.go`, `http_handlers.go`, `store.go`
- Database: SQLite file in `data/` directory (created at runtime via `DB_PATH` env var)

**BeerBot-frontend/project/app/:**
- Purpose: Next.js App Router entry points and API routes
- Contains: Page components, layout wrapper, API route handlers
- Key files: `page.tsx` (home), `layout.tsx` (wrapper), proxy and health routes

**BeerBot-frontend/project/components/:**
- Purpose: Reusable React components (all marked as client-side with "use client")
- Contains: UI components, containers, providers
- Key files: `UsersPage.tsx` (state container), `UsersList.tsx` (list rendering)

**BeerBot-frontend/project/lib/:**
- Purpose: Shared utilities and helpers
- Contains: Logging, formatting, helpers (non-component code)
- Key files: `logger.ts` (leveled logging with metadata)

## Key File Locations

**Entry Points:**

- `BeerBot-backend/bot/main.go`: Go backend starts here - parses flags, initializes database, starts HTTP server and Slack event loop
- `BeerBot-frontend/project/app/page.tsx`: Next.js frontend entry - renders UsersPage component
- `BeerBot-frontend/project/app/layout.tsx`: HTML wrapper - sets up theme provider, header, footer

**Configuration:**

- `BeerBot-backend/bot/main.go`: Flag parsing and environment variable reading (BOT_TOKEN, APP_TOKEN, CHANNEL, API_TOKEN, DB_PATH, ADDR, LOG_LEVEL)
- `BeerBot-frontend/project/package.json`: Node.js build and runtime config
- `BeerBot-frontend/project/tsconfig.json`: TypeScript compiler options, path aliases
- `BeerBot-frontend/project/next.config.js`: Next.js build behavior (currently minimal)

**Core Logic:**

- `BeerBot-backend/bot/slack.go`: Slack Socket Mode connection manager, event handler factory, reconnection logic
- `BeerBot-backend/bot/http_handlers.go`: All HTTP endpoint implementations and middleware
- `BeerBot-backend/bot/store.go`: SQLite schema migration and query implementation
- `BeerBot-frontend/project/components/UsersPage.tsx`: Date range state, API calls for lists
- `BeerBot-frontend/project/components/UsersList.tsx`: Per-user stats fetching with concurrency, rendering

**Testing:**

- `BeerBot-backend/bot/store_test.go`: Tests for Store interface
- `BeerBot-backend/bot/store_givers_test.go`: Tests for giver-specific queries
- Frontend: No test files present; testing would go in `*.test.tsx` or `*.spec.tsx` files (not currently used)

## Naming Conventions

**Files:**

- Go backend: `*.go` for source, `*_test.go` for tests, `Dockerfile`, docker-compose files
- Frontend: `.tsx` for React components, `.ts` for utilities, `.mjs` for ESM config files
- All lowercase with underscores for multi-word files (e.g., `http_handlers.go`, `store_givers_test.go`)

**Directories:**

- Backend uses flat structure (all files in `bot/` package, no subdirectories)
- Frontend uses Next.js conventions: `app/` for routes, `components/` for UI, `lib/` for utilities, `public/` for static

**Functions/Exports:**

- Go: PascalCase for exported functions (e.g., `NewSQLiteStore`, `AddBeer`, `GetAllGivers`)
- Go: camelCase for unexported functions (e.g., `parseLogLevel`, `startHTTPServer`)
- TypeScript: camelCase for exports (e.g., `logger`, `withTiming`)
- React components: PascalCase (e.g., `UsersPage`, `UsersList`, `ThemeProvider`)

**Types:**

- Go: Exported struct names PascalCase (e.g., `SQLiteStore`, `SlackConnectionManager`)
- TypeScript: Type aliases camelCase for informal, PascalCase for formal (e.g., `type DateRange`, `type UsersListProps`)

## Where to Add New Code

**New Feature (e.g., new API endpoint):**
- Primary code: `BeerBot-backend/bot/http_handlers.go` (add handler function) + `main.go` (register in mux)
- Backend logic: `BeerBot-backend/bot/slack.go` (if event-driven) or `store.go` (if data query)
- Frontend: `BeerBot-frontend/project/components/UsersPage.tsx` or new component in `components/`
- Tests: `BeerBot-backend/bot/*_test.go` for new store methods

**New Component/Module:**
- UI component: Create file in `BeerBot-frontend/project/components/ComponentName.tsx`
- Utility: Add to `BeerBot-frontend/project/lib/` (or new subdirectory if many utilities)
- Backend service: Create new file in `BeerBot-backend/bot/` (e.g., `cache.go`, `notifications.go`)

**Styling:**
- Tailwind classes in JSX attributes (no separate CSS files)
- Global styles in `app/globals.css` (imported in layout.tsx)
- Dark mode: next-themes with `class` attribute (see ThemeProvider.tsx)

**Shared Utilities:**
- Frontend: `BeerBot-frontend/project/lib/`
- Backend: Flat in `bot/` package; consider extracting to separate packages if significant

## Special Directories

**BeerBot-backend/bot/data/:**
- Purpose: Runtime data storage (SQLite database files)
- Generated: Yes (created at runtime by backend startup via `ensureDBDir()`)
- Committed: No (gitignored via `.gitignore`)

**BeerBot-frontend/project/node_modules/:**
- Purpose: Installed npm dependencies
- Generated: Yes (created by npm/bun install)
- Committed: No (gitignored)

**BeerBot-frontend/project/.next/:**
- Purpose: Next.js build artifacts and cached modules
- Generated: Yes (created by `next build` or `next dev`)
- Committed: No (gitignored)

**BeerBot-frontend/project/public/:**
- Purpose: Static assets served as-is (favicon, logo SVG)
- Generated: No (checked in)
- Committed: Yes

## Configuration Files

**Backend Configuration:**

- `BeerBot-backend/bot/main.go`: Lines 53-76 - Flag definitions with environment variable fallbacks
- Required env vars: `BOT_TOKEN`, `APP_TOKEN`, `CHANNEL`
- Optional env vars: `EMOJI`, `DB_PATH`, `ADDR`, `API_TOKEN`, `LOG_LEVEL`, `MAX_PER_DAY`

**Frontend Configuration:**

- `BeerBot-frontend/project/next.config.js`: Next.js build configuration
- `BeerBot-frontend/project/tsconfig.json`: TypeScript compiler options (target ES2017, strict mode off but nullChecks on)
- `BeerBot-frontend/project/tailwind.config.js`: Tailwind CSS theme customization
- Environment: `NEXT_PUBLIC_BACKEND_BASE` (proxy destination, defaults to `http://localhost:8080`)

---

*Structure analysis: 2026-01-30*
