# Phase 3: Quarterly Rankings - Research

**Researched:** 2026-02-02
**Domain:** Next.js App Router routing, tab navigation, date/quarter calculations
**Confidence:** HIGH

## Summary

Phase 3 implements a quarterly rankings page at `/rankings` with year/quarter navigation. The existing codebase uses Next.js 15.5.11 App Router with React 19, Tailwind CSS, and a beer-themed design system with CSS variables. The backend already supports date-range queries via `start` and `end` parameters.

The primary technical challenges are:
1. Multi-segment dynamic routes (`/rankings/[year]/[quarter]`) with proper TypeScript typing
2. Accessible tab navigation between quarters and all-time view
3. Quarter date calculations (start/end dates, current/previous quarter logic)
4. Rank change indicators comparing current quarter to previous quarter

**Primary recommendation:** Use Next.js App Router dynamic routes with optional catch-all for `/rankings/[[...path]]`, calculate quarter dates client-side with native JavaScript, and build accessible tabs using ARIA patterns already established in the codebase.

## Standard Stack

The established libraries/tools for this domain:

### Core (Already in Project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Next.js | 15.5.11 | App Router, dynamic routes | Already installed, provides file-based routing |
| React | 19.2.x | UI components | Already installed |
| Tailwind CSS | 4.1.14 | Styling with CSS variables | Already configured with beer theme |
| lucide-react | 0.546.0 | Icons (arrows, chevrons) | Already installed, provides ChevronsUp/Down |

### Supporting (Already Available)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| next/navigation | (builtin) | useRouter, usePathname, redirect | Client navigation, URL state |
| next/link | (builtin) | Link component | Tab/button navigation |

### Not Needed
| Instead of | Don't Use | Reason |
|------------|-----------|--------|
| @headlessui/react | Custom tabs | Overkill for this use case, adds bundle weight |
| react-aria | Custom tabs | Project doesn't need full a11y library |
| date-fns/moment | Native JS | Quarter calculations are simple enough |

**No additional dependencies required.** All functionality can be built with existing stack.

## Architecture Patterns

### Recommended Route Structure
```
app/
├── rankings/
│   ├── page.tsx           # Redirect /rankings -> /rankings/2026/q1 (current quarter)
│   ├── all/
│   │   └── page.tsx       # All-time rankings at /rankings/all
│   └── [year]/
│       └── [quarter]/
│           └── page.tsx   # Quarterly rankings at /rankings/2026/q1
```

### Alternative: Optional Catch-All (Simpler)
```
app/
├── rankings/
│   └── [[...path]]/
│       └── page.tsx       # Handles: /rankings, /rankings/all, /rankings/2026/q1
```

The catch-all approach is simpler for handling redirects and 404s in one place.

### Pattern 1: Dynamic Route with Promise Params (Next.js 15)
**What:** Next.js 15 changed params to be a Promise
**When to use:** All dynamic route segments
**Example:**
```typescript
// Source: https://nextjs.org/docs/app/api-reference/file-conventions/dynamic-routes
// app/rankings/[year]/[quarter]/page.tsx
interface PageProps {
  params: Promise<{ year: string; quarter: string }>
}

export default async function QuarterlyRankingsPage({ params }: PageProps) {
  const { year, quarter } = await params
  // Validate: year is 4-digit number, quarter is q1|q2|q3|q4
  const yearNum = parseInt(year, 10)
  const quarterNum = parseInt(quarter.slice(1), 10)

  if (isNaN(yearNum) || yearNum < 2020 || yearNum > 2030) {
    notFound()
  }
  if (!/^q[1-4]$/.test(quarter)) {
    notFound()
  }

  return <QuarterlyRankingsClient year={yearNum} quarter={quarterNum} />
}
```

### Pattern 2: Redirect from Base Route
**What:** Redirect `/rankings` to current quarter
**When to use:** When user navigates to bare `/rankings` URL
**Example:**
```typescript
// Source: https://nextjs.org/docs/app/guides/redirecting
// app/rankings/page.tsx
import { redirect } from 'next/navigation'

export default function RankingsRedirect() {
  const now = new Date()
  const year = now.getFullYear()
  const quarter = Math.ceil((now.getMonth() + 1) / 3)
  redirect(`/rankings/${year}/q${quarter}`)
}
```

### Pattern 3: Accessible Tab Navigation (No Library)
**What:** ARIA-compliant tabs built with native elements
**When to use:** Quarter/All-Time tab bar
**Example:**
```typescript
// Follows WAI-ARIA tabs pattern
// Source: https://react-aria.adobe.com/Tabs
interface TabProps {
  tabs: Array<{ id: string; label: string; href: string }>
  activeTab: string
}

function TabBar({ tabs, activeTab }: TabProps) {
  return (
    <div role="tablist" aria-label="Rankings period">
      {tabs.map(tab => (
        <Link
          key={tab.id}
          href={tab.href}
          role="tab"
          aria-selected={tab.id === activeTab}
          className={tab.id === activeTab ? 'active-tab' : 'inactive-tab'}
        >
          {tab.label}
        </Link>
      ))}
    </div>
  )
}
```

### Pattern 4: Quarter Date Calculations
**What:** Calculate quarter start/end dates for API calls
**When to use:** When fetching quarterly data from backend
**Example:**
```typescript
// Source: https://bobbyhadz.com/blog/javascript-get-date-quarter
function getQuarterDates(year: number, quarter: number): { start: string; end: string } {
  const startMonth = (quarter - 1) * 3
  const startDate = new Date(year, startMonth, 1)
  const endDate = new Date(year, startMonth + 3, 0) // Last day of quarter

  return {
    start: startDate.toISOString().slice(0, 10),
    end: endDate.toISOString().slice(0, 10)
  }
}

// Get current quarter
function getCurrentQuarter(): { year: number; quarter: number } {
  const now = new Date()
  return {
    year: now.getFullYear(),
    quarter: Math.ceil((now.getMonth() + 1) / 3)
  }
}

// Get previous quarter (handles year wrap)
function getPreviousQuarter(year: number, quarter: number): { year: number; quarter: number } {
  if (quarter === 1) {
    return { year: year - 1, quarter: 4 }
  }
  return { year, quarter: quarter - 1 }
}
```

### Anti-Patterns to Avoid
- **Client-side routing state:** Don't use React state for quarter/year selection; use URL params for shareability
- **Fetching all quarters at once:** Fetch only the displayed quarter's data
- **Complex tab libraries:** Don't add @headlessui or react-aria; native elements with ARIA attrs suffice
- **Moment.js for dates:** Quarter calculations are 3 lines of vanilla JS

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Date range API queries | Custom date logic | Existing backend `start`/`end` params | Backend already supports date ranges |
| User info fetching | New endpoint | Existing `/api/proxy/user` | Already fetches names/avatars |
| Leaderboard display | New component | Existing `UsersList` component | Already handles ranking display, just needs rank change prop |
| Theme styling | Hardcoded colors | Existing CSS variables | `hsl(var(--primary))` etc. already defined |

**Key insight:** The backend already has all required endpoints. Phase 3 is purely frontend work: routing, tabs, and a small rank-change indicator addition to UsersList.

## Common Pitfalls

### Pitfall 1: Synchronous Params Access (Next.js 15)
**What goes wrong:** `params.year` instead of `await params`
**Why it happens:** Next.js 14 allowed synchronous access; 15 requires Promise
**How to avoid:** Always `const { year, quarter } = await params`
**Warning signs:** TypeScript error about Promise, or runtime undefined

### Pitfall 2: Future Quarter Visibility
**What goes wrong:** Showing Q3/Q4 tabs in January 2026
**Why it happens:** Generating all 4 quarter tabs without checking current date
**How to avoid:** Calculate which quarters are valid (completed + current); hide future ones
**Warning signs:** Clicking Q4 in Q1 shows empty leaderboard

### Pitfall 3: Year Wrap for Previous Quarter
**What goes wrong:** Q1 2026 - 1 = Q0 2026 (invalid)
**Why it happens:** Simple subtraction without year adjustment
**How to avoid:** If quarter === 1, return { year: year - 1, quarter: 4 }
**Warning signs:** "Last Quarter" button breaks in January

### Pitfall 4: Hydration Mismatch on Current Quarter
**What goes wrong:** Server renders Q1, client renders Q2 (clock difference)
**Why it happens:** Server and client have different system times
**How to avoid:** Determine quarter client-side after mount, or pass from server
**Warning signs:** React hydration warning in console

### Pitfall 5: Empty Rankings for New Leaderboard Users
**What goes wrong:** No rank change indicator for users without previous quarter data
**Why it happens:** User didn't exist in previous quarter's leaderboard
**How to avoid:** Check if user was in previous quarter; show "NEW" badge or no indicator
**Warning signs:** Undefined when calculating rank difference

### Pitfall 6: Non-Accessible Tab Navigation
**What goes wrong:** Tabs don't work with keyboard, no ARIA labels
**Why it happens:** Using div/button without proper roles
**How to avoid:** Use `role="tablist"`, `role="tab"`, `aria-selected`, `aria-label`
**Warning signs:** Accessibility audit failures, keyboard navigation broken

## Code Examples

### Rank Change Indicator Component
```typescript
// Source: Lucide icons for arrows (https://lucide.dev/icons/)
import { ChevronUp, ChevronDown, ChevronsUp, ChevronsDown, Minus } from 'lucide-react'

interface RankChangeProps {
  current: number
  previous: number | null // null = new to leaderboard
}

function RankChangeIndicator({ current, previous }: RankChangeProps) {
  if (previous === null) {
    return (
      <span
        className="text-xs font-medium px-1.5 py-0.5 rounded"
        style={{
          backgroundColor: 'hsl(var(--accent))',
          color: 'hsl(var(--accent-foreground))'
        }}
      >
        NEW
      </span>
    )
  }

  const change = previous - current // positive = moved up

  if (change === 0) {
    return <Minus size={14} style={{ color: 'hsl(var(--muted-foreground))' }} />
  }

  const isUp = change > 0
  const isBigMove = Math.abs(change) >= 3

  const Icon = isUp
    ? (isBigMove ? ChevronsUp : ChevronUp)
    : (isBigMove ? ChevronsDown : ChevronDown)

  return (
    <Icon
      size={16}
      style={{ color: isUp ? 'hsl(142 76% 36%)' : 'hsl(0 72% 51%)' }}
      aria-label={`${isUp ? 'Up' : 'Down'} ${Math.abs(change)} place${Math.abs(change) !== 1 ? 's' : ''}`}
    />
  )
}
```

### Year Selector Dropdown
```typescript
// Styled to match beer theme using existing CSS variables
interface YearSelectorProps {
  years: number[]
  selectedYear: number
  onChange: (year: number) => void
}

function YearSelector({ years, selectedYear, onChange }: YearSelectorProps) {
  return (
    <select
      value={selectedYear}
      onChange={(e) => onChange(parseInt(e.target.value, 10))}
      className="px-3 py-1.5 rounded-md text-sm font-medium border cursor-pointer"
      style={{
        backgroundColor: 'hsl(var(--card))',
        color: 'hsl(var(--card-foreground))',
        borderColor: 'hsl(var(--border))',
      }}
      aria-label="Select year"
    >
      {years.map(year => (
        <option key={year} value={year}>{year}</option>
      ))}
    </select>
  )
}
```

### Shortcut Buttons (Current/Last Quarter)
```typescript
// Matches beer theme button styling from existing components
interface ShortcutButtonsProps {
  currentQuarter: { year: number; quarter: number }
  lastQuarter: { year: number; quarter: number }
  isCurrentActive: boolean
  isLastActive: boolean
}

function ShortcutButtons({
  currentQuarter,
  lastQuarter,
  isCurrentActive,
  isLastActive
}: ShortcutButtonsProps) {
  return (
    <div className="flex gap-2 mb-4">
      <Link
        href={`/rankings/${currentQuarter.year}/q${currentQuarter.quarter}`}
        className="px-4 py-2 rounded-md text-sm font-semibold transition-colors"
        style={{
          backgroundColor: isCurrentActive
            ? 'hsl(var(--primary))'
            : 'hsl(var(--secondary))',
          color: isCurrentActive
            ? 'hsl(var(--primary-foreground))'
            : 'hsl(var(--secondary-foreground))',
        }}
      >
        Current Quarter
      </Link>
      <Link
        href={`/rankings/${lastQuarter.year}/q${lastQuarter.quarter}`}
        className="px-4 py-2 rounded-md text-sm font-semibold transition-colors"
        style={{
          backgroundColor: isLastActive
            ? 'hsl(var(--primary))'
            : 'hsl(var(--secondary))',
          color: isLastActive
            ? 'hsl(var(--primary-foreground))'
            : 'hsl(var(--secondary-foreground))',
        }}
      >
        Last Quarter
      </Link>
    </div>
  )
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Pages Router `getStaticPaths` | App Router `generateStaticParams` | Next.js 13+ | Different API, Promise params |
| Synchronous params access | `await params` Promise | Next.js 15 | Must await or use React `use()` |
| `next/router` useRouter | `next/navigation` useRouter | Next.js 13+ | Different import path |
| Class-based components | Server Components + Client | React 18+ | Default server render, opt-in client |

**Current patterns in this codebase:**
- App Router with `page.tsx` convention
- Client Components marked with `"use client"`
- CSS variables with `hsl(var(--token))` pattern
- Lucide icons imported individually

## Open Questions

Things that couldn't be fully resolved:

1. **Years with Data Query**
   - What we know: Backend stores transactions with timestamps
   - What's unclear: No endpoint to get "all years with data"
   - Recommendation: Either query distinct years from backend (new endpoint), or scan a reasonable range (2020-current) and filter out empty ones client-side

2. **Rank Comparison Data Source**
   - What we know: Frontend fetches current quarter rankings via existing endpoints
   - What's unclear: Need previous quarter data simultaneously for comparison
   - Recommendation: Fetch both current and previous quarter in parallel, compute rank difference client-side

3. **Edge Case: Current Quarter Just Started**
   - What we know: Q1 starts Jan 1, Q2 starts Apr 1, etc.
   - What's unclear: If today is Jan 2, is "Current Quarter" useful (only 2 days of data)?
   - Recommendation: Always show current quarter shortcut; empty states are acceptable UX

## Claude's Discretion Recommendations

From CONTEXT.md, these areas are for Claude to decide:

### New Users Badge
**Recommendation:** Show "NEW" badge instead of no indicator.
**Rationale:** A "NEW" badge is more informative than an empty space and helps explain why there's no arrow. Use small accent-colored badge matching existing theme.

### Edge Case: Quarter Just Started
**Recommendation:** Always show both "Current Quarter" and "Last Quarter" shortcuts.
**Rationale:** Even with minimal data, users expect these buttons to work. The leaderboard will show fewer users with lower counts, which is acceptable. No special messaging needed.

### Year Dropdown Placement
**Recommendation:** Place year dropdown to the LEFT of quarter tabs, on the same horizontal line.
**Rationale:**
- Left-to-right reading order: year then quarter makes semantic sense (2026 Q1)
- Keeps navigation compact in one row
- Grayed out / hidden when "All Time" tab is active

### Loading and Error States
**Recommendation:**
- Loading: Skeleton loader matching UsersList card dimensions
- Error: Inline error message with retry button
- Empty: "No beers recorded for this quarter" with beer emoji
**Rationale:** Consistent with existing UX patterns; keep it simple.

## Sources

### Primary (HIGH confidence)
- [Next.js Dynamic Routes Docs](https://nextjs.org/docs/app/api-reference/file-conventions/dynamic-routes) - Params typing, catch-all segments
- [Next.js Redirecting Guide](https://nextjs.org/docs/app/guides/redirecting) - redirect() function patterns
- [Lucide Icons](https://lucide.dev/icons/) - ChevronsUp, ChevronsDown icon availability
- Existing codebase files: `layout.tsx`, `page.tsx`, `UsersPage.tsx`, `UsersList.tsx`, `ThemeToggle.tsx`, `handlers.go`, `store.go`

### Secondary (MEDIUM confidence)
- [React Aria Tabs](https://react-aria.adobe.com/Tabs) - ARIA tab patterns (not using library, but pattern reference)
- [bobbyhadz Quarter Calculation](https://bobbyhadz.com/blog/javascript-get-date-quarter) - JavaScript quarter date logic

### Tertiary (LOW confidence)
- None - all claims verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - verified against existing package.json and codebase
- Architecture: HIGH - verified against Next.js 15 docs and existing route structure
- Pitfalls: HIGH - based on documented Next.js 15 changes and common date logic issues

**Research date:** 2026-02-02
**Valid until:** 30 days (stable domain, Next.js 15 is current)
