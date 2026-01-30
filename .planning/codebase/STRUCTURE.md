# Codebase Structure

**Analysis Date:** 2026-01-30

## Directory Layout

```
/home/tdolfen/Projects/github.com/BeerBILabs/
├── backend/                           # Slack bot backend service
│   ├── bot/                           # Go application code
│   │   ├── main.go                    # Entry point, Slack event handler, HTTP API server
│   │   ├── store.go                   # SQLite persistence layer
│   │   ├── store_test.go              # Unit tests for store
│   │   ├── store_givers_test.go       # Unit tests for givers functionality
│   │   └── go.mod                     # Go dependencies
│   └── .github/
│       └── workflows/                 # CI/CD workflows (if any)
├── frontend/                          # Next.js frontend service
│   ├── project/                       # Next.js application root
│   │   ├── app/                       # Next.js app directory (routing)
│   │   │   ├── page.tsx               # Root page (homepage)
│   │   │   ├── layout.tsx             # Root layout with header/footer
│   │   │   ├── not-found.tsx          # 404 page
│   │   │   ├── api/                   # Next.js API routes
│   │   │   │   ├── proxy/             # API proxy routes
│   │   │   │   │   └── [...path]/     # Catch-all route handler
│   │   │   │   │       └── route.ts   # Proxy to backend API
│   │   │   │   └── health/            # Health check route
│   │   │   │       └── route.ts       # Health endpoint
│   │   │   └── globals.css            # Global styles
│   │   ├── components/                # Reusable React components
│   │   │   ├── UsersPage.tsx          # Main page component with date filtering
│   │   │   ├── UsersList.tsx          # List of users with stats (Givers/Recipients)
│   │   │   ├── DateRangePicker.tsx    # Date range selection component
│   │   │   ├── ThemeProvider.tsx      # Theme context and localStorage management
│   │   │   ├── ThemeToggle.tsx        # Dark/light mode toggle button
│   │   │   ├── Logo.tsx               # BeerBot logo component
│   │   ├── public/                    # Static assets
│   │   │   └── favicon.svg            # Favicon
│   │   ├── scripts/                   # Build/utility scripts
│   │   ├── package.json               # Frontend dependencies
│   │   ├── tsconfig.json              # TypeScript configuration
│   │   ├── tailwind.config.js         # Tailwind CSS theming
│   │   ├── next.config.js             # Next.js configuration
│   │   ├── postcss.config.mjs          # PostCSS configuration
│   │   ├── eslint.config.mjs          # ESLint configuration
│   │   └── bun.lock                   # Bun package manager lockfile
│   ├── docker/                        # Docker configuration (if any)
│   └── .github/
│       └── workflows/                 # CI/CD workflows (if any)
├── .planning/
│   └── codebase/                      # Codebase analysis documents (this directory)
├── .git/                              # Git repository
└── README.md                          # Project overview
```

## Directory Purposes

**`backend/bot/`:**
- Purpose: Slack bot service code for handling beer events and providing REST API
- Contains: Go application logic, SQLite store, tests
- Key files: `main.go` (660 lines: Slack integration, HTTP server, event loop), `store.go` (330 lines: database operations)

**`frontend/project/app/`:**
- Purpose: Next.js App Router pages and API routes
- Contains: Server components (`page.tsx`, `layout.tsx`), client route handlers (`api/`), page-level structure
- Key files: `page.tsx` (imports UsersPage), `layout.tsx` (header/footer), `api/proxy/[...path]/route.ts` (backend proxy)

**`frontend/project/components/`:**
- Purpose: Reusable React components for the leaderboard UI
- Contains: Client components with hooks, styled UI elements
- Key files: `UsersPage.tsx` (state management and layout), `UsersList.tsx` (data fetching and rendering)

**`frontend/project/public/`:**
- Purpose: Static files served as-is by Next.js
- Contains: Images, icons, fonts
- Key files: `favicon.svg` (site icon)

**`frontend/project/scripts/`:**
- Purpose: Development or build-time utility scripts
- Contains: Helper scripts (currently empty or minimal)

## Key File Locations

**Entry Points:**

- `frontend/project/app/page.tsx`: Frontend homepage, imports `UsersPage` component (React Server Component wrapper)
- `frontend/project/components/UsersPage.tsx`: Main leaderboard UI with date range selection and state
- `backend/bot/main.go`: Go application entry point, contains `func main()` at line 202

**Configuration:**

- `backend/bot/go.mod`: Go module definition with dependencies (Slack, SQLite, Prometheus, zerolog)
- `frontend/project/package.json`: Node.js dependencies (Next.js, React, Tailwind CSS, ESLint)
- `frontend/project/tsconfig.json`: TypeScript compiler options, path alias `@/*` → `./src/*` (currently unused)
- `frontend/project/tailwind.config.js`: Tailwind CSS theme colors (dark mode support)
- `frontend/project/next.config.js`: Next.js build configuration

**Core Logic:**

- `backend/bot/main.go`: Slack event handler (lines 419-556), HTTP API handlers (lines 283-404), connection manager (lines 31-155)
- `backend/bot/store.go`: SQLite schema migrations (lines 22-160), query methods for counting beers (lines 245-329)
- `frontend/project/components/UsersList.tsx`: Data fetching logic with concurrency control (lines 24-132), UI rendering (lines 141-184)
- `frontend/project/app/api/proxy/[...path]/route.ts`: Request forwarding logic, token injection (lines 5-63)

**Testing:**

- `backend/bot/store_test.go`: Unit tests for emoji counts and beer transactions
- `backend/bot/store_givers_test.go`: Additional store tests (location: `backend/bot/`)

## Naming Conventions

**Files:**

- **Backend:** PascalCase for types (none in main.go, conventions used in comments), snake_case for tables (e.g., `giver_id`, `ts_rfc`)
- **Frontend:** PascalCase for components (e.g., `UsersList.tsx`, `DateRangePicker.tsx`), camelCase for utilities
- **Tests:** `{subject}_test.go` pattern (e.g., `store_test.go`), `TestFunction` test case names

**Directories:**

- **Backend:** Lowercase (`bot/`)
- **Frontend:** Lowercase (`app/`, `components/`, `public/`), nested by feature/route (e.g., `app/api/proxy/`)

**Functions/Types:**

- **Backend (Go):** PascalCase for exported (e.g., `NewSQLiteStore`, `AddBeer`, `CountGivenInDateRange`), camelCase for unexported (e.g., `authMiddleware`, `parseSlackTimestamp`)
- **Frontend (TSX):** PascalCase for components (e.g., `UsersPage`, `UsersList`), camelCase for hooks and helpers (e.g., `useTheme`)

## Where to Add New Code

**New Feature - Feature Branch:**
1. Add feature-specific components to `frontend/project/components/` (e.g., `BeerStats.tsx`)
2. If feature requires new API endpoint, add handler to `backend/bot/main.go` (pattern: register route in mux)
3. If feature requires database schema changes, add migration to `backend/bot/store.go` (pattern: add to migrate() function)
4. Add tests to `backend/bot/store_test.go` or create `backend/bot/{feature}_test.go`

**New Component (Frontend):**
- Location: `frontend/project/components/{ComponentName}.tsx`
- Pattern: PascalCase filename, default export, typed props
- Example: See `DateRangePicker.tsx` (lines 1-45)

**New API Endpoint (Backend):**
- Location: Add handler in `backend/bot/main.go` (around line 283-404 with other handlers)
- Pattern: Create `*http.HandlerFunc`, wrap with `authMiddleware()` if protected, register with `mux.Handle()`
- Example: See `givenHandler` (lines 283-302) or `receivedHandler` (lines 303-322)

**Utilities/Helpers (Backend):**
- Location: `backend/bot/main.go` or new file `backend/bot/{utility}.go`
- Pattern: Exported functions for shared logic, unexported for internal use
- Example: `parseDateRangeFromParams()` (lines 178-200), `parseSlackTimestamp()` (lines 631-660)

**Utilities/Helpers (Frontend):**
- Location: `frontend/project/components/` or new file `frontend/project/lib/{utility}.ts`
- Pattern: Export as named functions or constants
- Example: Theme management in `ThemeProvider.tsx` (lines 5-36)

## Special Directories

**`backend/bot/testdata/`:**
- Purpose: Test database files created at runtime
- Generated: Yes (created by tests in `store_test.go`)
- Committed: No (in `.gitignore`)

**`frontend/project/.next/`:**
- Purpose: Next.js build output and cache
- Generated: Yes (by `next build`)
- Committed: No (in `.gitignore`)

**`frontend/project/node_modules/`:**
- Purpose: Installed npm dependencies
- Generated: Yes (by `bun install`)
- Committed: No (in `.gitignore`)

**`frontend/project/public/`:**
- Purpose: Static assets served directly by Next.js (no processing)
- Generated: No
- Committed: Yes

## Project Root Conventions

**Monorepo Structure:**
- Backend and frontend are independent services in same repo
- Each has its own package manager (Go modules, npm/bun)
- Each has its own CI/CD workflows in `.github/workflows/`

**Environment Configuration:**
- **Backend:** Read from CLI flags or env vars (see `main.go` lines 203-226: flag.String calls)
  - Required: `BOT_TOKEN`, `APP_TOKEN`, `CHANNEL`
  - Optional: `DB_PATH`, `API_TOKEN`, `ADDR`, `MAX_PER_DAY`, `EMOJI`

- **Frontend:** Next.js env vars must be prefixed `NEXT_PUBLIC_` to be available client-side
  - `NEXT_PUBLIC_BACKEND_BASE`: Backend URL (default: `http://localhost:8080`)
  - `API_TOKEN`: Injected server-side in proxy route (never exposed to browser)

---

*Structure analysis: 2026-01-30*
