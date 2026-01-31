# Architecture Patterns: BeerBot Theming and Quarterly Rankings

**Domain:** Beer leaderboard enhancement (theming + quarterly views)
**Researched:** 2026-01-31
**Confidence:** HIGH

## Executive Summary

This milestone adds theming and quarterly ranking views to the existing BeerBot application. The architecture integrates cleanly with existing components:
- **Theme system:** Custom implementation already exists; can optionally migrate to next-themes for consistency
- **Quarterly data:** Backend already supports date ranges; extend with quarter-aware endpoints
- **Routing:** Next.js 15 App Router supports dynamic quarterly pages using nested dynamic segments
- **Styling:** Tailwind class-based dark mode already configured; extend with beer-branded color palette

## Existing Architecture Analysis

### Frontend (Next.js 15 + React 19)
```
frontend/project/
├── app/
│   ├── layout.tsx              # Root layout with ThemeProvider
│   ├── page.tsx                # Home page (renders UsersPage)
│   ├── globals.css             # Tailwind + custom theme tokens
│   └── api/
│       └── proxy/[...path]/    # Backend proxy with auth injection
├── components/
│   ├── ThemeProvider.tsx       # Custom theme context (localStorage + classList)
│   ├── ThemeToggle.tsx         # Sun/Moon toggle button
│   └── UsersPage.tsx           # Main leaderboard with date range picker
└── tailwind.config.js          # darkMode: ['class'] with HSL tokens
```

**Current theme implementation:**
- Custom `ThemeProvider` using `localStorage` + `document.documentElement.classList`
- `useTheme` hook provides `setTheme()` and `toggleTheme()`
- Avoids hydration mismatch by checking `mounted` state
- HSL-based CSS custom properties in `globals.css` (`:root` for light, `.dark` for dark)

### Backend (Go + SQLite)
```
backend/bot/
├── main.go                     # HTTP server + Slack integration
├── store.go                    # SQLite queries
└── beers table schema:
    ├── giver_id, recipient_id
    ├── ts (Slack timestamp)
    ├── ts_rfc (RFC3339 datetime for queries)
    └── count
```

**Existing API endpoints:**
- `GET /api/given?user=X&start=YYYY-MM-DD&end=YYYY-MM-DD` - beers given in date range
- `GET /api/received?user=X&start=YYYY-MM-DD&end=YYYY-MM-DD` - beers received in date range
- `GET /api/givers` - all distinct givers
- `GET /api/recipients` - all distinct recipients
- `GET /api/user?user=X` - Slack user info (name, profile image)

**Key observations:**
- Date range queries already implemented using `substr(ts_rfc, 1, 10) BETWEEN ? AND ?`
- No quarterly aggregation yet
- No leaderboard endpoint (frontend fetches all users, then queries individually)

## Integration Points

### 1. Theming Integration

**Current state:** Custom theme implementation works but differs from common patterns.

**Integration options:**

#### Option A: Keep Custom Implementation (Recommended for this milestone)
**Pros:**
- Already working
- Zero dependencies
- Minimal changes needed
- Simpler migration path for beer-branded colors

**Cons:**
- Non-standard pattern (harder for new developers)
- No built-in system preference sync

**Changes needed:**
1. Extend HSL token palette in `globals.css` with beer-branded colors
2. Add theme toggle to header (already exists in `layout.tsx`)
3. Test dark mode across all components

#### Option B: Migrate to next-themes
**Pros:**
- Industry-standard library (14.4k+ GitHub stars)
- Prevents flash of unstyled content (FOUC)
- System preference sync built-in
- Better TypeScript support

**Cons:**
- Migration effort (replace ThemeProvider, test all components)
- Additional dependency

**Migration complexity:** LOW (same hook pattern, mostly drop-in replacement)

**Recommendation:** Keep custom implementation for this milestone. Consider next-themes migration in future if team prefers standard patterns.

### 2. Backend Quarterly Data Integration

**New endpoints needed:**

```
GET /api/leaderboard?start=YYYY-MM-DD&end=YYYY-MM-DD&type=givers|recipients&limit=50
Response: [
  { user_id: "U123", total: 42, real_name: "Alice", profile_image: "..." },
  ...
]
```

**Implementation approach:**

1. **Extend existing queries** (don't create new endpoints if possible)
2. **Add leaderboard endpoint** for efficient batch queries:
   ```sql
   SELECT recipient_id, SUM(count) as total
   FROM beers
   WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
   GROUP BY recipient_id
   ORDER BY total DESC
   LIMIT ?
   ```
3. **Optional: Add quarter helper endpoint** for quarter-to-date-range conversion:
   ```
   GET /api/quarters/2026/q1
   Response: { start: "2026-01-01", end: "2026-03-31", year: 2026, quarter: 1 }
   ```

**Date range calculation:**
- Frontend calculates quarter boundaries (Jan-Mar, Apr-Jun, Jul-Sep, Oct-Dec)
- Backend remains quarter-agnostic (just handles date ranges)
- Cleaner separation of concerns

**Performance considerations:**
- Current queries use `substr(ts_rfc, 1, 10)` which is non-indexed
- Add index: `CREATE INDEX idx_beers_ts_rfc_date ON beers (date(ts_rfc))`
- For quarterly aggregation at scale, consider adding `quarter` and `year` columns

### 3. Next.js Routing for Quarterly Pages

**Target URLs:**
```
/rankings/2026/q1
/rankings/2026/q2
/rankings/all-time
/rankings
```

**App Router structure:**
```
app/
├── rankings/
│   ├── page.tsx                    # Default view (current year or all-time)
│   ├── all-time/
│   │   └── page.tsx                # All-time rankings
│   └── [year]/
│       └── [quarter]/
│           └── page.tsx            # Quarterly rankings
```

**Dynamic segment implementation:**

```typescript
// app/rankings/[year]/[quarter]/page.tsx
interface PageProps {
  params: Promise<{ year: string; quarter: string }>
}

export default async function QuarterlyRankingsPage({ params }: PageProps) {
  const { year, quarter } = await params

  // Validate params
  if (!/^\d{4}$/.test(year)) notFound()
  if (!/^q[1-4]$/i.test(quarter)) notFound()

  // Calculate date range
  const quarterNum = parseInt(quarter.slice(1))
  const startMonth = (quarterNum - 1) * 3
  const start = new Date(parseInt(year), startMonth, 1)
  const end = new Date(parseInt(year), startMonth + 3, 0)

  // Fetch data
  const data = await fetchLeaderboard(start, end)

  return <RankingsView data={data} period={`Q${quarterNum} ${year}`} />
}
```

**Static generation for past quarters:**
```typescript
export async function generateStaticParams() {
  const currentYear = new Date().getFullYear()
  const years = [currentYear - 1, currentYear] // Last 2 years
  const quarters = ['q1', 'q2', 'q3', 'q4']

  return years.flatMap(year =>
    quarters.map(quarter => ({ year: year.toString(), quarter }))
  )
}
```

**Navigation component:**
```typescript
// components/QuarterSelector.tsx
function QuarterSelector() {
  const currentYear = new Date().getFullYear()
  const quarters = ['Q1', 'Q2', 'Q3', 'Q4']

  return (
    <nav>
      <Link href="/rankings/all-time">All Time</Link>
      {[currentYear - 1, currentYear].map(year =>
        quarters.map((q, i) =>
          <Link key={`${year}-${q}`} href={`/rankings/${year}/q${i+1}`}>
            {q} {year}
          </Link>
        )
      )}
    </nav>
  )
}
```

### 4. Tailwind Theme Extension

**Current setup:**
- `darkMode: ['class']` already configured
- HSL custom properties for all semantic colors
- `globals.css` defines `:root` (light) and `.dark` (dark) tokens

**Beer-branded palette extension:**

```css
/* globals.css - Add beer-branded colors */
@layer base {
  :root {
    /* Existing tokens... */
    --beer-gold: 45 93% 58%;        /* #f0c020 - Beer foam/highlight */
    --beer-amber: 30 90% 50%;       /* #e67300 - Amber ale */
    --beer-copper: 20 75% 40%;      /* #b35a00 - Copper/brown ale */
    --beer-dark: 25 50% 20%;        /* #4d2600 - Stout/porter */
    --beer-green: 160 85% 39%;      /* #00c896 - Existing accent (keep) */
  }
  .dark {
    /* Existing tokens... */
    --beer-gold: 45 85% 65%;        /* Lighter for dark bg */
    --beer-amber: 30 80% 55%;
    --beer-copper: 20 65% 50%;
    --beer-dark: 25 40% 85%;        /* Inverted for contrast */
    --beer-green: 160 75% 45%;
  }
}
```

```javascript
// tailwind.config.js - Extend theme
export const theme = {
  extend: {
    colors: {
      // ... existing colors
      beer: {
        gold: 'hsl(var(--beer-gold))',
        amber: 'hsl(var(--beer-amber))',
        copper: 'hsl(var(--beer-copper))',
        dark: 'hsl(var(--beer-dark))',
        green: 'hsl(var(--beer-green))',
      }
    }
  }
}
```

**Usage in components:**
```tsx
<h1 className="text-beer-gold dark:text-beer-gold">🍺 Quarterly Rankings</h1>
<div className="bg-beer-amber/10 border-beer-amber">...</div>
```

**Color strategy:**
- Primary CTA: `beer-green` (existing #00c896)
- Highlights: `beer-gold`
- Secondary: `beer-amber`
- Muted: `beer-copper`
- Dark elements: `beer-dark`

## Component Architecture

### New Components

```
components/
├── RankingsView.tsx            # Quarterly rankings display
├── QuarterSelector.tsx         # Quarter navigation
├── LeaderboardTable.tsx        # Table with givers/recipients
└── ThemePalettePicker.tsx      # [Optional] Multi-theme selector
```

### Modified Components

```
components/
├── ThemeProvider.tsx           # [No changes if keeping custom]
├── ThemeToggle.tsx             # [Update styling with beer colors]
└── UsersPage.tsx               # [Add link to quarterly view]
```

### Component Dependencies

```
RankingsView
├── uses: QuarterSelector, LeaderboardTable
├── fetches: /api/leaderboard
└── params: year, quarter (from route)

LeaderboardTable
├── uses: UserAvatar, ScoreDisplay
├── receives: data prop from RankingsView
└── handles: sorting, pagination

QuarterSelector
├── uses: Next.js Link
├── generates: /rankings/[year]/[quarter] URLs
└── highlights: current quarter
```

## Data Flow

### Quarterly Rankings Page Load

```
1. User navigates to /rankings/2026/q1
   ↓
2. Next.js calls QuarterlyRankingsPage with params
   ↓
3. Component validates year/quarter params
   ↓
4. Component calculates date range (2026-01-01 to 2026-03-31)
   ↓
5. Component fetches /api/leaderboard?start=2026-01-01&end=2026-03-31&type=recipients
   ↓
6. Backend queries SQLite with date range filter + GROUP BY
   ↓
7. Backend joins user metadata (name, image) from Slack API
   ↓
8. Frontend receives leaderboard array
   ↓
9. Component renders LeaderboardTable with data
   ↓
10. Theme applied via .dark class on <html> element
```

### Theme Toggle Flow

```
1. User clicks ThemeToggle button
   ↓
2. useTheme().toggleTheme() called
   ↓
3. Checks current theme: document.documentElement.classList.contains('dark')
   ↓
4. Toggles classList: .classList.toggle('dark')
   ↓
5. Updates localStorage: setItem('theme', 'dark' | 'light')
   ↓
6. CSS custom properties re-evaluated (automatic)
   ↓
7. All hsl(var(--*)) colors update instantly
   ↓
8. Button icon updates (Sun ↔ Moon)
```

## Build Order and Phases

### Phase 1: Backend Foundation
**Goal:** Support quarterly data queries efficiently

1. Add leaderboard endpoint to `main.go`
   - Aggregate query with GROUP BY
   - Join user metadata (or return user_id only, fetch metadata client-side)
   - Add limit parameter (default 50)

2. Optimize date queries in `store.go`
   - Add index on `date(ts_rfc)`
   - Test query performance with realistic data volumes

3. [Optional] Add quarter helper endpoint
   - `/api/quarters/:year/:quarter` → date range

**Validation:**
- Query `SELECT COUNT(*) FROM beers` completes in <100ms
- Leaderboard endpoint returns top 50 in <200ms
- Date range calculations produce correct boundaries

### Phase 2: Frontend Theme Enhancement
**Goal:** Beer-branded color palette with dark mode support

1. Extend `globals.css` with beer color tokens
   - Define `:root` values
   - Define `.dark` values
   - Test color contrast ratios (WCAG AA)

2. Update `tailwind.config.js` with beer color extensions

3. Update `ThemeToggle.tsx` styling
   - Use beer colors for button background
   - Improve hover/active states

4. [Optional] Add theme preview/picker component

**Validation:**
- Toggle between light/dark modes smoothly
- No FOUC (flash of unstyled content)
- Colors accessible (contrast ratio ≥4.5:1)

### Phase 3: Routing and Navigation
**Goal:** Dynamic quarterly pages with clean URLs

1. Create `app/rankings/[year]/[quarter]/page.tsx`
   - Parse and validate params
   - Calculate date range from quarter
   - Fetch leaderboard data
   - Handle loading and error states

2. Create `app/rankings/all-time/page.tsx`
   - Fetch all-time leaderboard (no date filter)

3. Create `app/rankings/page.tsx`
   - Default view (redirect to current quarter or all-time)

4. Create `components/QuarterSelector.tsx`
   - List available quarters (last 2 years)
   - Highlight active quarter
   - Link to all-time view

**Validation:**
- `/rankings/2026/q1` loads correctly
- Invalid routes (e.g., `/rankings/2026/q5`) show 404
- Navigation between quarters works
- Back button works correctly

### Phase 4: Leaderboard UI
**Goal:** Display quarterly rankings with user metadata

1. Create `components/LeaderboardTable.tsx`
   - Render user avatar, name, score
   - Sort by score (descending)
   - Handle empty state

2. Create `components/RankingsView.tsx`
   - Integrate QuarterSelector + LeaderboardTable
   - Show period label ("Q1 2026")
   - Add tabs for givers vs recipients

3. Update `components/UsersPage.tsx`
   - Add link to quarterly rankings
   - Consider merging with RankingsView (same data)

**Validation:**
- Leaderboard displays top 50 users
- User avatars load correctly
- Scores are accurate
- Empty state shows helpful message

### Phase 5: Integration and Polish
**Goal:** Smooth integration with existing app

1. Add quarterly rankings link to header navigation
2. Ensure theme consistency across all pages
3. Add loading states for data fetching
4. Add error handling (API failures, invalid quarters)
5. Test responsive design (mobile/tablet/desktop)
6. Update README with new features

**Validation:**
- All pages respect theme settings
- Navigation flows naturally
- Error states are user-friendly
- Mobile layout works well

## Patterns to Follow

### Pattern 1: HSL Token-Based Theming
**What:** Use CSS custom properties with HSL values for all colors

**When:** Defining any color (background, text, border, etc.)

**Example:**
```css
/* globals.css */
:root { --primary: 160 85% 39%; }
.dark { --primary: 160 75% 45%; }
```

```tsx
// Component
<div className="bg-primary text-primary-foreground">...</div>
```

**Why:** Single source of truth, automatic dark mode support, no JS required

### Pattern 2: Client Component for Theme State
**What:** Mark theme-dependent components with "use client"

**When:** Component reads theme state or toggles theme

**Example:**
```typescript
"use client"
import { useTheme } from './ThemeProvider'

export default function ThemeToggle() {
  const { toggleTheme } = useTheme()
  return <button onClick={toggleTheme}>...</button>
}
```

**Why:** Theme state is browser-side (localStorage), requires client rendering

### Pattern 3: Server Component for Data Fetching
**What:** Keep page components as async server components

**When:** Fetching data for initial render (quarterly leaderboards)

**Example:**
```typescript
// app/rankings/[year]/[quarter]/page.tsx (server component)
export default async function QuarterlyRankingsPage({ params }) {
  const { year, quarter } = await params
  const data = await fetchLeaderboard(year, quarter)
  return <ClientRankingsView data={data} />
}
```

**Why:** Better performance, SEO-friendly, less client-side JS

### Pattern 4: Dynamic Route Validation
**What:** Validate route params and return 404 for invalid values

**When:** Using dynamic segments like `[year]` or `[quarter]`

**Example:**
```typescript
import { notFound } from 'next/navigation'

if (!/^\d{4}$/.test(year) || !/^q[1-4]$/i.test(quarter)) {
  notFound()
}
```

**Why:** Prevents rendering errors, improves UX for typos/bad URLs

### Pattern 5: Date Range Abstraction
**What:** Centralize quarter-to-date conversion logic

**When:** Multiple components need to work with quarters

**Example:**
```typescript
// lib/quarters.ts
export function getQuarterRange(year: number, quarter: number) {
  const startMonth = (quarter - 1) * 3
  const start = new Date(year, startMonth, 1)
  const end = new Date(year, startMonth + 3, 0)
  return {
    start: start.toISOString().slice(0, 10),
    end: end.toISOString().slice(0, 10)
  }
}
```

**Why:** Single source of truth, easier testing, reusable

## Anti-Patterns to Avoid

### Anti-Pattern 1: Inline Color Values
**What:** Hardcoding colors like `bg-[#00c896]` instead of using tokens

**Why bad:** Breaks dark mode, no central control, harder to rebrand

**Instead:**
```tsx
// BAD
<div className="bg-[#00c896]">

// GOOD
<div className="bg-beer-green">
```

### Anti-Pattern 2: Client-Side Quarter Calculation on Every Render
**What:** Recalculating date ranges in component render functions

**Why bad:** Unnecessary computation, harder to test, inconsistent

**Instead:**
```typescript
// BAD
function MyComponent() {
  const start = new Date(2026, 0, 1) // Recalculated on every render
  ...
}

// GOOD
const QUARTER_RANGES = {
  '2026-q1': { start: '2026-01-01', end: '2026-03-31' }
}
```

### Anti-Pattern 3: N+1 User Metadata Queries
**What:** Fetching user info individually for each leaderboard entry

**Why bad:** Slow (50 requests for 50 users), hits Slack API rate limits

**Instead:**
```typescript
// BAD
leaderboard.map(async entry => {
  const user = await fetch(`/api/user?user=${entry.user_id}`)
  return { ...entry, ...user }
})

// GOOD
const leaderboardWithUsers = await fetch('/api/leaderboard?include=users')
```

### Anti-Pattern 4: Theme Flash on Page Load
**What:** Rendering light mode first, then switching to dark mode

**Why bad:** Poor UX, looks broken, causes layout shift

**Instead:**
```typescript
// ThemeProvider should run before first paint
useEffect(() => {
  const theme = localStorage.getItem('theme')
  document.documentElement.classList.toggle('dark', theme === 'dark')
  setMounted(true) // Only render toggle after theme applied
}, [])
```

### Anti-Pattern 5: Hardcoded Year/Quarter Lists
**What:** Manually updating quarter navigation every year

**Why bad:** Requires code changes annually, easy to forget

**Instead:**
```typescript
// Generate quarters dynamically
const currentYear = new Date().getFullYear()
const years = Array.from({ length: 3 }, (_, i) => currentYear - i)
const quarters = ['q1', 'q2', 'q3', 'q4']
```

## Performance Considerations

### Database Query Optimization

**Issue:** Quarterly aggregations scan entire beers table

**Solutions:**
1. Add index on `date(ts_rfc)` for WHERE clause
   ```sql
   CREATE INDEX idx_beers_ts_rfc_date ON beers (date(ts_rfc));
   ```

2. Add composite index for common query patterns
   ```sql
   CREATE INDEX idx_beers_recipient_date ON beers (recipient_id, date(ts_rfc));
   ```

3. [Optional] Denormalize with `year` and `quarter` columns
   ```sql
   ALTER TABLE beers ADD COLUMN year INTEGER;
   ALTER TABLE beers ADD COLUMN quarter INTEGER;
   CREATE INDEX idx_beers_year_quarter ON beers (year, quarter);
   ```

**Expected performance:**
- < 100ms for quarterly aggregation (with index)
- < 500ms for all-time aggregation (thousands of rows)

### Frontend Bundle Size

**Issue:** Adding next-themes increases bundle size

**Solutions:**
1. Keep custom implementation (zero bytes)
2. If using next-themes, ensure it's tree-shaken (only import what's needed)
3. Code-split theme toggle with dynamic import (optional)

**Current bundle:** ~120KB (Next.js + React + Tailwind)
**With next-themes:** ~125KB (+5KB, minimal impact)

### Static Generation for Past Quarters

**Issue:** Every quarterly page hit triggers server-side render

**Solutions:**
1. Use `generateStaticParams()` to pre-render past quarters
2. Set `revalidate = 86400` (24 hours) for current quarter
3. Current quarter uses ISR (Incremental Static Regeneration)

```typescript
export const revalidate = 86400 // 24 hours

export async function generateStaticParams() {
  // Pre-render last 8 quarters
  const params = []
  const now = new Date()
  for (let i = 0; i < 8; i++) {
    const date = new Date(now.getFullYear(), now.getMonth() - i * 3, 1)
    const year = date.getFullYear()
    const quarter = Math.floor(date.getMonth() / 3) + 1
    params.push({ year: year.toString(), quarter: `q${quarter}` })
  }
  return params
}
```

**Result:** Past quarters served as static HTML, current quarter updates daily

## Testing Strategy

### Unit Tests

```typescript
// lib/quarters.test.ts
describe('getQuarterRange', () => {
  test('Q1 2026 returns correct dates', () => {
    expect(getQuarterRange(2026, 1)).toEqual({
      start: '2026-01-01',
      end: '2026-03-31'
    })
  })

  test('Q4 2025 handles year boundaries', () => {
    expect(getQuarterRange(2025, 4)).toEqual({
      start: '2025-10-01',
      end: '2025-12-31'
    })
  })
})
```

### Integration Tests

```typescript
// app/rankings/[year]/[quarter]/page.test.tsx
describe('QuarterlyRankingsPage', () => {
  test('renders leaderboard for valid quarter', async () => {
    const page = await QuarterlyRankingsPage({
      params: Promise.resolve({ year: '2026', quarter: 'q1' })
    })
    expect(page).toContain('Q1 2026')
  })

  test('returns 404 for invalid quarter', async () => {
    expect(() =>
      QuarterlyRankingsPage({
        params: Promise.resolve({ year: '2026', quarter: 'q5' })
      })
    ).toThrow(NotFoundError)
  })
})
```

### E2E Tests

```typescript
// e2e/quarterly-rankings.spec.ts
test('navigating to quarterly rankings', async ({ page }) => {
  await page.goto('/rankings/2026/q1')
  await expect(page.locator('h1')).toContainText('Q1 2026')

  // Click next quarter
  await page.click('[href="/rankings/2026/q2"]')
  await expect(page).toHaveURL('/rankings/2026/q2')

  // Toggle theme
  await page.click('[aria-label*="theme"]')
  await expect(page.locator('html')).toHaveClass(/dark/)
})
```

## Sources and References

### Official Documentation
- [Tailwind CSS Dark Mode](https://tailwindcss.com/docs/dark-mode) - Official Tailwind CSS documentation for class-based dark mode
- [Next.js Dynamic Routes (App Router)](https://nextjs.org/docs/app/api-reference/file-conventions/dynamic-routes) - Official Next.js documentation for dynamic route segments
- [Next.js 15 Release Notes](https://nextjs.org/blog/next-15) - Changes to caching defaults and App Router

### Library Documentation
- [next-themes on npm](https://www.npmjs.com/package/next-themes) - Official next-themes package documentation
- [next-themes GitHub](https://github.com/pacocoursey/next-themes) - Source code and examples

### Best Practices Guides
- [Next.js Dynamic Route Segments in the App Router (2026 Guide)](https://thelinuxcode.com/nextjs-dynamic-route-segments-in-the-app-router-2026-guide/) - Comprehensive guide on dynamic routes
- [Theme colors with Tailwind CSS v4.0 and Next Themes](https://medium.com/@kevstrosky/theme-colors-with-tailwind-css-v4-0-and-next-themes-dark-light-custom-mode-36dca1e20419) - Custom theme implementation patterns
- [Go + SQLite Best Practices](https://jacob.gold/posts/go-sqlite-best-practices/) - Go SQLite optimization techniques
- [Best Practices for Using SQLite Date and Time Functions](https://www.slingacademy.com/article/best-practices-for-using-sqlite-date-and-time-functions/) - SQLite date query patterns

### API Design Patterns
- [Modern API Design Best Practices for 2026](https://www.xano.com/blog/modern-api-design-best-practices/) - REST API patterns including date ranges
- [How to GROUP BY Month and Year in SQLite](https://www.geeksforgeeks.org/sqlite/how-to-group-by-month-and-year-in-sqlite/) - Quarterly aggregation techniques

### Community Resources
- [Mastering Custom Dark Mode with Tailwind CSS](https://medium.com/@asyncme/mastering-custom-dark-mode-with-tailwind-css-from-class-to-selector-strategy-1d0e7d8888f3) - Advanced dark mode patterns
- [Creating Dynamic Routes in Next.js 15 with App Router](https://medium.com/@sehouli.hamza/creating-dynamic-routes-in-nextjs-15-with-app-router-e8cc15e401d0) - Practical dynamic route examples
