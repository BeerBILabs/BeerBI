# Coding Conventions

**Analysis Date:** 2026-01-30

## Naming Patterns

**Files:**
- Frontend React/Next components: PascalCase with `.tsx` extension (e.g., `UsersList.tsx`, `ThemeToggle.tsx`, `DateRangePicker.tsx`)
- Next.js API routes: snake_case for dynamic segments and lowercase for files (e.g., `app/api/proxy/[...path]/route.ts`, `app/api/health/route.ts`)
- Backend Go files: snake_case separating concerns (e.g., `store.go`, `store_test.go`, `store_givers_test.go`)

**Functions:**
- Frontend (React/TypeScript): camelCase for utility functions, PascalCase for React components
  - Example: `getSystemPrefersDark()` (utility), `UsersList()` (component)
  - Async functions use async/await syntax: `async function loadStats()`, `async function fetchNamesAvatars()`
- Backend (Go): camelCase for unexported functions, PascalCase for exported functions
  - Example: `NewSQLiteStore()` (exported constructor), `migrate()` (unexported), `parseDate()` (unexported)
- Type definitions used as constructors: `NewSlackConnectionManager(botToken, appToken)` pattern

**Variables:**
- Frontend: camelCase for all variables
  - State variables: `const [isDark, setIsDark] = useState(false)`
  - Refs: `const lastArgsRef = useRef<string | null>(null)`
  - Boolean flags: `mounted.current`, `cancelled`
- Backend (Go): camelCase for local variables, PascalCase for exported struct fields
  - Example in `SlackConnectionManager`: exported fields like `isConnected`, `lastPing`, `reconnectCount`
  - Date parsing: `layout := "2006-01-02"`, `startStr`, `endStr`

**Types:**
- Frontend TypeScript: PascalCase for all type definitions
  - Interface example: `type DateRangePickerProps = { start?: string | null; end?: string | null; onChange?: (start: string, end: string) => void }`
  - Type aliases: `type DateRange = { start?: string; end?: string }`
  - Union types: `type DateRange = { start: string; end: string }`
- Backend Go: PascalCase for exported struct types
  - Example: `type SlackConnectionManager struct`, `type ProxyParams struct`

## Code Style

**Formatting:**
- TypeScript/JavaScript: No explicit formatter configured; eslint manages style via configuration
- Go: Standard Go formatting conventions (implicit via language)
- HTML/CSS in components: Inline Tailwind classNames for styling, no separate CSS files for components

**Linting:**
- Frontend: ESLint with Next.js configuration (`eslint.config.mjs`)
  - Config extends: `next/core-web-vitals`, `next/typescript`
  - Disabled rules:
    - `react/no-unescaped-entities`: 0
    - `@typescript-eslint/no-explicit-any`: 0
  - Ignored paths: `node_modules/**`, `**/.next/**`, `out/**`, `build/**`, `**/next-env.d.ts`
  - ESLint version: ^9

**TypeScript Configuration:**
- Target: ES2017
- Strict mode: disabled (`"strict": false`), but `strictNullChecks: true` enabled
- Module resolution: node
- JSX: preserve (for Next.js)
- Path aliases: `@/*` maps to `./src/*` in `tsconfig.json`
- `noEmit: true` - TypeScript compiles but doesn't emit files

## Import Organization

**Order:**
1. React/Next imports: `import { useEffect, useState } from 'react'`
2. Next.js imports: `import Image from 'next/image'`, `import Link from 'next/link'`
3. Third-party packages: `import { Sun, Moon } from 'lucide-react'`, `import DatePicker from 'react-datepicker'`
4. Local relative imports: `import { useTheme } from './ThemeProvider'`, `import UsersList from './UsersList'`
5. Type imports: `import type { ReactElement } from 'react'`, `import type { Metadata } from 'next'`

**Path Aliases:**
- `@/*` resolves to `./src/*` (configured in `tsconfig.json`)
- Used for absolute imports instead of relative paths

**Go Import Organization:**
- Standard library imports first (no blank line after): `import ("context", "database/sql", "encoding/json", ...)`
- Third-party packages after: `"github.com/slack-go/slack"`, `"github.com/prometheus/client_golang/prometheus"`
- Blank imports for side effects: `_ "github.com/mattn/go-sqlite3"`

## Error Handling

**Frontend Patterns:**
- Try-catch blocks in async functions: Uses `try { ... } catch (err) { ... }` pattern
- Error coercion to Error type: `(err as Error)?.message ?? 'unknown error'`
- Graceful error recovery: Falls back to default values
  - Example: `out[u] = 0` when fetch fails in concurrent worker
  - Example: Sets avatar to null when image fetch fails
- Error logging: Uses `console.error(err)` for debugging (minimal, only in critical paths)

**Backend Patterns (Go):**
- Error wrapping: Uses `fmt.Errorf("message: %w", err)` for context preservation
- Exported functions return error as last return value: `NewSQLiteStore(db *sql.DB) (*SQLiteStore, error)`
- Error checking: Standard Go pattern `if err != nil { return ... }`
- Panics: Avoided in favor of returning errors; using `log.Fatalf()` only in main() for fatal initialization errors
- Error context in logs: Includes operation context in log messages: `log.Printf("Socket mode client error, will reconnect: %v", err)`

## Logging

**Frontend Framework:** console (standard browser API)

**Patterns:**
- `console.error()` used only for exceptions in async operations
- `console.log()` used for operation tracking in API proxy: `console.log('proxy (app router) authSource=', authSource, 'path=', pathStr)`
- No structured logging; output is minimal and direct

**Backend Framework:** log (standard Go library) and zerolog imported but mostly log usage

**Patterns:**
- `log.Printf()` for formatted output with context: `log.Printf("Slack connection established (reconnect count: %d)", scm.reconnectCount)`
- `log.Println()` for simple messages: `log.Println("Starting Slack socket mode client...")`
- Connection state tracking: Logs connection changes, reconnect attempts, and status
- `log.Fatalf()` used only in main() for initialization failures that prevent startup

## Comments

**When to Comment:**
- Function-level documentation for exported functions (Go pattern)
  - Example: `// NewSlackConnectionManager creates a new connection manager`
  - Example: `// IsConnected returns the current connection status`
- Inline comments for non-obvious logic or workarounds
  - Example: `// Fetch stats for each user` before complex useEffect
  - Example: `// Calculate exponential backoff delay` before calculation logic
  - Example: `// Avoid sending cookies/credentials to different origin by default`
- Explain the "why" not the "what": `// non-visual helper; ensures class applied on mount`

**JSDoc/TSDoc:**
- Minimal usage; function signatures are preferred for clarity
- Type annotations in TypeScript serve as documentation
- React components use destructured props with inline type definitions

## Function Design

**Size:**
- Frontend: Keep to 50-100 lines; separate complex logic into helper functions within useEffect hooks
  - Example: `loadStats()` async function defined inside useEffect in `UsersList.tsx`
  - Example: `worker()` function defined inside async function for concurrent operations
- Backend: Methods can be longer when dealing with SQL operations; ~30-50 lines typical
  - Example: `migrate()` handles all table creation and schema validation in one place

**Parameters:**
- Frontend React: Props destructured directly in function signature with inline type definition
  - Example: `function ThemeToggle()` (no props), `function UsersList({ title, users, range }: UsersListProps)`
- Backend Go: Context as first parameter for cancellation: `(scm *SlackConnectionManager) TestConnection(ctx context.Context)`

**Return Values:**
- Frontend: React components return JSX element (ReactElement) or null
  - Example: `export default function Home(): ReactElement { return ... }`
  - Example: `ThemeProvider()` returns `mounted ? null : null` (no visual output)
- Backend Go: Error as final return value (idiomatic)
  - Example: `func (s *SQLiteStore) GetCount(user string, emoji string) (int, error)`
  - Example: `func (scm *SlackConnectionManager) TestConnection(ctx context.Context) error`

## Module Design

**Exports:**
- Frontend: Default exports for page/route components, named exports for utilities
  - Example: `export default function UsersList()`
  - Example: `export function useTheme()` for custom hooks
- Backend Go: Uppercase names for public API, lowercase for internal
  - Example: `func NewSQLiteStore()` exported constructor
  - Example: `func (s *SQLiteStore) migrate()` unexported helper

**Barrel Files:**
- Not used in this codebase; imports are direct from source files

## Async Patterns (Frontend)

**useEffect with Cleanup:**
- Cleanup function returns cancellation logic
- Mutable refs track mounted state to prevent state updates after unmount
  - Example in `UsersList.tsx`: `let cancelled = false` flag with cleanup `return () => { cancelled = true }`
- Debouncing implemented with setTimeout: `const timer = setTimeout(() => { loadStats() }, 200)`

**Concurrent Operations:**
- Worker pattern for limiting concurrency
  - Example: `const concurrency = 5` with shared `idx` counter
  - Each worker runs in parallel with `Promise.all(workers)`
  - Increment counter atomically: `const i = idx++`

**Error Recovery in Async:**
- Failed operations default to safe values: `out[u] = 0` (no data means zero count)
- Check both `cancelled` and `mounted.current` before setState
- Comparison checks prevent redundant updates: `if (prev !== next && mounted.current) setStats(out)`

## State Management

**Frontend:**
- useState for component-level state (counters, booleans, objects)
- useRef for mutable state that doesn't trigger re-renders (refs, flags)
- Custom hook pattern for cross-component logic: `useTheme()` returns object with methods

**Backend:**
- Struct fields for instance state: `SlackConnectionManager` holds client references and counters
- sync.RWMutex for concurrent access: `scm.mu.RLock()` / `scm.mu.RUnlock()` pattern

---

*Convention analysis: 2026-01-30*
