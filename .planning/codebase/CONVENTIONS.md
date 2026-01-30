# Coding Conventions

**Analysis Date:** 2026-01-30

## Naming Patterns

**Files:**
- React components: PascalCase (e.g., `UsersList.tsx`, `ThemeProvider.tsx`, `DateRangePicker.tsx`)
- Utility/Library files: camelCase (e.g., `logger.ts`)
- Route handlers: Lowercase with brackets for dynamic segments (e.g., `route.ts`, `[...path]`)

**Functions:**
- Component functions: PascalCase (e.g., `export default function UsersList()`)
- Utility functions: camelCase (e.g., `withTiming()`, `shouldLog()`)
- Event handlers: camelCase with `on` prefix or descriptive name (e.g., `onClick()`)

**Variables:**
- State variables: camelCase (e.g., `mounted`, `lastArgsRef`, `range`)
- Type/interface names: PascalCase (e.g., `DateRange`, `UsersListProps`, `ProxyParams`)
- Constants: camelCase or UPPER_SNAKE_CASE depending on scope (e.g., `LEVELS`, `defaultLevel`, `concurrency`)

**Types:**
- React component props: PascalCase ending with `Props` (e.g., `UsersListProps`, `DateRangePickerProps`)
- Generic types: PascalCase (e.g., `DateRange`, `Level`)
- Union types: PascalCase (e.g., `'Givers' | 'Recipients' | string`)

## Code Style

**Formatting:**
- No Prettier config defined - ESLint is the primary linting tool
- Code uses consistent spacing and indentation from ESLint configuration
- Quote style: Double quotes for strings (observed throughout codebase)
- Semicolons: Used consistently at end of statements

**Linting:**
- Tool: ESLint 9.39.2
- Config: `eslint.config.mjs` using `eslint-config-next/core-web-vitals`
- Run command: `npm run lint` (lints entire directory)
- Key rules enforced: Next.js core web vitals best practices

**ESLint Configuration:**
```javascript
// eslint.config.mjs
import { defineConfig, globalIgnores } from 'eslint/config'
import nextVitals from 'eslint-config-next/core-web-vitals'

const eslintConfig = defineConfig([
  ...nextVitals,
  globalIgnores([
    '.next/**',
    'out/**',
    'build/**',
    'next-env.d.ts',
  ]),
])
export default eslintConfig
```

## Import Organization

**Order:**
1. Next.js/React framework imports
2. External library imports
3. Internal component imports
4. Internal utility imports
5. Type imports (`import type`)

**Examples from codebase:**
```typescript
// app/layout.tsx
import './globals.css'
import type { Metadata } from 'next'
import type { ReactNode } from 'react'
import { Inter } from 'next/font/google'
import { ThemeProvider } from '../components/ThemeProvider'
import ThemeToggle from '../components/ThemeToggle'
import Logo from '../components/Logo'
import Link from 'next/link'
```

```typescript
// components/UsersList.tsx
"use client"
import React, { useEffect, useState, useRef } from 'react'
import Image from 'next/image'
```

**Path Aliases:**
- `@/*` maps to `./src/*` (configured in `tsconfig.json`)
- Current codebase uses relative imports (e.g., `../components/`, `../lib/`)

## Error Handling

**Patterns:**
- Try-catch blocks used in async operations (fetch calls)
- Error logging with `console.error()` at catch boundaries
- Graceful degradation: Failed API calls default to zero counts or null values
- No custom error types defined; uses native Error objects

**Example from `UsersList.tsx`:**
```typescript
try {
  const resp = await fetch(`${path}?${q.toString()}`)
  if (!resp.ok) {
    out[u] = 0
    continue
  }
  const j = await resp.json()
  out[u] = title === 'Givers' ? j.given : j.received
} catch (err) {
  console.error(err);
  out[u] = 0
}
```

**API Route Error Handling (from `route.ts`):**
```typescript
try {
  const resp = await fetch(url, { ... })
  // handle response
} catch (err) {
  console.error('[proxy] error', JSON.stringify({ ... }))
  return NextResponse.json({ error: (err as Error).message }, { status: 500 })
}
```

## Logging

**Framework:** `console` methods (debug, info, warn, error) wrapped in custom `logger` utility

**Custom Logger Module (`lib/logger.ts`):**
- Provides: `logger.debug()`, `logger.info()`, `logger.warn()`, `logger.error()`
- All methods accept message string and optional metadata object
- Formatting: ISO timestamp, level uppercase, JSON metadata
- Level filtering: Production defaults to 'info', development defaults to 'debug'
- Exported function: `withTiming<T>()` for measuring async operation performance

**Logging Patterns:**
```typescript
// info level with metadata
logger.info('range_quick', { label: q.label, start: next.start, end: next.end })
logger.warn('fetch_givers: non_ok', { status: gResp.status })

// Performance timing
const result = await withTiming('fetch_givers', () => fetch('/api/proxy/givers'))
```

**Unstructured Logging:**
- API routes and components may use direct `console.error()` for error reporting
- Server-side API routes use structured logging with JSON metadata

## Comments

**When to Comment:**
- JSDoc/TSDoc not used consistently in this codebase
- Inline comments present only for non-obvious logic (e.g., state synchronization patterns)
- Comments explain "why" not "what" (e.g., "avoid synchronous setState inside effect")

**Example from `UsersPage.tsx`:**
```typescript
// Initialize empty, then set on mount to avoid SSR/CSR time drift
const [range, setRange] = useState<DateRange>({ start: '', end: '' })

// Set default range using a macrotask to avoid synchronous setState inside effect
const id = setTimeout(() => {
  // ...
}, 0)
```

**Type Annotations:**
- Explicit return types on functions: `function Home(): ReactElement`
- Component prop types declared separately as interfaces/types
- React.FC not used; props passed as function parameter with type

## Function Design

**Size:** Functions generally kept concise, with async operations delegated to nested functions where needed

**Parameters:**
- Destructured from props for React components (e.g., `{ title, users, range }`)
- Named parameters preferred for utility functions
- Generic type parameters used in utility functions (e.g., `withTiming<T>()`)

**Return Values:**
- Component functions return JSX.Element or ReactElement
- API routes return NextResponse or NextResponse.json()
- Utility functions return Promise<T> for async operations
- Functions handling fetch errors return default values (0, null, etc.) rather than throwing

## Module Design

**Exports:**
- Default exports used for React components
- Named exports used for utilities and custom hooks
- Multiple exports from a single file (e.g., `ThemeProvider.tsx` exports both component and hook)

**Barrel Files:**
- Not used in this codebase
- All imports are direct relative paths to specific files

**Component Structure:**
- Client components marked with `"use client"` directive at top (required by Next.js)
- Server components (like layout, page) use server-side features (metadata, etc.)
- Props typed inline or with separate type definitions above component

## Data Fetching Patterns

**Client-Side Fetch:**
- Wrapped in `useEffect` hooks
- Concurrency control via worker pattern (e.g., 5 concurrent requests in `UsersList.tsx`)
- Cancellation flags prevent state updates on unmounted components
- Debouncing with `setTimeout` to avoid duplicate requests (200ms delay in stats fetching)

**Server-Side Proxy:**
- API route at `/api/proxy/[...path]` forwards requests to backend
- Injects Authorization header from server-side `API_TOKEN` env var
- Returns response as-is with status and content-type preserved

## TypeScript Configuration

**Key Settings (from `tsconfig.json`):**
- `strict: false` (not fully strict mode)
- `strictNullChecks: true` (explicitly enabled despite loose mode)
- `jsx: "react-jsx"` (new JSX transform)
- `target: ES2017`
- `moduleResolution: node`
- `forceConsistentCasingInFileNames: true`

---

*Convention analysis: 2026-01-30*
