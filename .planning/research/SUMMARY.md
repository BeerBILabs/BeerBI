# Project Research Summary

**Project:** BeerBot v1.0 - Theming and Quarterly Rankings Enhancement
**Domain:** Slack beer recognition tracker with leaderboard frontend (enhancement milestone)
**Researched:** 2026-01-31
**Confidence:** HIGH

## Executive Summary

BeerBot v1.0 is an enhancement milestone for an existing, validated application built with Next.js 15.5.6 and Go 1.25. The milestone adds theming capabilities (light/dark mode with beer-branded color palettes) and quarterly ranking navigation to the existing leaderboard. Research focused on dependency modernization, theming implementation patterns, and integration approaches that minimize breaking changes.

**Recommended approach:** Stay on Next.js 15.x (upgrade to 15.5.11) and defer Next.js 16 upgrade to a future milestone. Next.js 16 introduces critical breaking changes (async-only request APIs, middleware rename, next lint removal) that would expand scope far beyond theming. Add next-themes 0.4.6 for robust theme switching with FOUC prevention, extend Tailwind CSS configuration with beer-branded HSL color tokens, and create quarterly navigation using Next.js App Router dynamic segments. Backend already supports date-range queries; extend with efficient leaderboard aggregation endpoint.

**Key risks and mitigation:** The application already has a custom theme implementation, creating integration complexity. Hydration mismatches are the primary risk when adding next-themes (mitigate with `suppressHydrationWarning` and mounted state checks). Tailwind v4 has breaking changes in theme configuration (verify current version is 4.1.14 and use CSS `@theme` syntax). Docker layer ordering needs correction to prevent cache invalidation on every code change. Go 1.25.6 and Bun 1.3.7 updates are low-risk patch releases.

## Key Findings

### Recommended Stack

BeerBot should modernize dependencies conservatively, focusing on security patches and stability rather than major version jumps. The existing Next.js 15 + React 19 foundation is solid and production-ready.

**Core technologies:**
- **Next.js 15.5.11** (update from 15.5.6) — Latest stable 15.x with backported bug fixes; avoid 16.x breaking changes that expand milestone scope
- **React 19.2.0** (already current) — Latest stable, fully compatible with Next.js 15.x
- **Go 1.25.6** (update from 1.25.3) — Latest with security fixes released Jan 28, 2026
- **Bun 1.3.7** (update from 1.3.0+) — Latest runtime with HMR improvements and dependency catalog features
- **Tailwind CSS 4.1.14** (already current) — Modern CSS framework with v4 @theme directive for custom palettes
- **next-themes 0.4.6** (new addition) — Industry standard for Next.js theme management (2,172+ projects), prevents FOUC, system preference support
- **Docker Alpine 3.23** (update from 3.20) — Latest Alpine with security patches (released Jan 27, 2026)

**Deferred upgrades:**
- Next.js 16.x — Breaking changes in async APIs, middleware architecture, and linting require separate milestone
- TypeScript 6/7 — Still in development, not production-ready

### Expected Features

Research identified clear feature priorities based on modern dashboard UX expectations and competitive differentiation opportunities.

**Must have (table stakes):**
- Light/dark theme toggle with system preference detection — Standard 2026 UX expectation
- Theme persistence across sessions — Users expect localStorage-based consistency
- No flash of unstyled content (FOUC) — Critical for perceived quality
- Accessible theme toggle (WCAG 2.2 Level AA) — Keyboard navigation, aria-pressed state, clear labels
- Quarterly time period navigation — Q1/Q2/Q3/Q4 selector with year controls
- Current quarter as default view — Users expect to see recent data first
- Visible active selection indicator — Must show which quarter/theme is active

**Should have (competitive):**
- Beer-branded color theme (golden/amber palette) — Creates product personality beyond generic light/dark
- Smooth theme transitions (200-300ms CSS) — Polished feel vs jarring switches
- URL state sync for shareable links — `/rankings/2026/q1` enables sharing specific quarters
- Year selector for historical data — Navigation across multiple years
- Quarterly color coding (seasonal) — Visual differentiation: Q1=winter blue, Q2=spring green, etc.
- Theme-aware charts/visualizations — Leaderboard bars adapt to active theme

**Defer (v2+):**
- Multiple branded themes (Stout, IPA, Wheat) — One beer theme sufficient for v1, avoid decision paralysis
- Auto-switching based on time of day — Breaks user trust in deliberate preference setting
- Year-over-year comparison views — High complexity, limited MVP value
- Custom date ranges beyond quarters — Quarterly presets cover 90% of use cases

### Architecture Approach

The enhancement integrates cleanly with existing architecture by extending current patterns rather than replacing them. Frontend uses Next.js 15 App Router with Tailwind class-based dark mode. Backend provides date-range query support; needs efficient aggregation endpoint for leaderboards.

**Major components:**

1. **Theme System (Frontend)** — Custom ThemeProvider already exists using localStorage + classList; can optionally migrate to next-themes for consistency. HSL-based CSS custom properties in globals.css enable automatic dark mode color adaptation. Beer-branded colors extend existing palette using Tailwind v4 @theme directive.

2. **Quarterly Navigation (Frontend)** — Next.js App Router dynamic segments (`/rankings/[year]/[quarter]/page.tsx`) handle quarterly URLs. QuarterSelector component generates navigation links. Date calculation logic converts quarter numbers to ISO date ranges. Static generation pre-renders past quarters; current quarter uses ISR.

3. **Leaderboard Aggregation (Backend)** — New `/api/leaderboard` endpoint performs efficient GROUP BY queries with date range filtering. Existing date-range support in store.go extends naturally. Database index on `date(ts_rfc)` ensures sub-100ms query performance. User metadata joins from Slack API or cached.

4. **Integration Points** — ThemeProvider wraps app layout with suppressHydrationWarning on html tag. QuarterSelector integrates into header navigation. Leaderboard components consume aggregated data. All components respect theme tokens via Tailwind utilities.

### Critical Pitfalls

Research identified five critical pitfalls that could derail the milestone if not addressed proactively.

1. **Hydration Mismatch with next-themes** — Theme state is undefined during SSR, causing server HTML to mismatch client HTML. React throws hydration errors and components flash. **Prevention:** Add `suppressHydrationWarning` to `<html>` tag, delay theme-dependent rendering with mounted state checks, use CSS-based approach for theme-aware content (dark:hidden / dark:block).

2. **Next.js 16 Async Breaking Changes** — If upgrading to 16.x, synchronous access to params/searchParams/cookies/headers fails at build time. All routes break simultaneously. **Prevention:** Stay on 15.5.11 for this milestone. If future upgrade needed, run codemod first: `npx @next/codemod@canary upgrade latest`.

3. **Tailwind v4 Theme Configuration** — Tailwind v4 moved from JavaScript config to CSS-first with @theme directive. Old tailwind.config.js patterns silently ignored. **Prevention:** Use `@theme { --color-primary: value; }` syntax in globals.css, not module.exports in config file. Run `npx @tailwindcss/upgrade@next` if migrating.

4. **Docker Layer Cache Invalidation** — Copying source before dependencies forces full reinstall on every code change. 5-10x slower builds. **Prevention:** Copy go.mod/go.sum first, run `go mod download`, then copy source. Multi-stage builds with minimal runtime image.

5. **Unnecessary Route Handlers in App Router** — Creating `/api/leaderboard/route.ts` when Server Component can fetch directly adds 100-300ms latency. **Prevention:** Fetch database directly in Server Components. Use Route Handlers only for external webhooks, mobile APIs, OAuth callbacks.

## Implications for Roadmap

Based on research, the work naturally divides into three sequential phases that build upon each other while minimizing risk.

### Phase 1: Dependency Modernization and Backend Foundation
**Rationale:** Establish stable foundation before adding features. Dependency updates carry hidden risks (Docker cache, Go version compatibility) that could block later work if discovered mid-feature development.

**Delivers:**
- Updated Next.js 15.5.11, Go 1.25.6, Bun 1.3.7, Alpine 3.23
- Optimized Docker layer ordering for faster builds
- New `/api/leaderboard` aggregation endpoint
- Database index on `date(ts_rfc)` for query performance
- Verified compatibility matrix (all tests passing)

**Addresses:**
- Infrastructure stability (from PITFALLS.md: Docker cache invalidation, Docker CVE exposure)
- Backend query optimization (from ARCHITECTURE.md: performance considerations)

**Avoids:**
- Next.js 16 breaking changes (explicitly deferred)
- Go major version import path issues (staying in 1.25.x)
- React 19 third-party library incompatibilities (audit dependencies first)

**Research flag:** Standard dependency update patterns; skip phase-specific research.

### Phase 2: Theming Implementation
**Rationale:** Theme system must be solid before quarterly navigation depends on it. Hydration issues discovered late cause cascading rework. Theme tokens need to be defined before quarterly components use them.

**Delivers:**
- next-themes 0.4.6 integration with FOUC prevention
- Beer-branded HSL color palette (golden/amber theme)
- Extended globals.css with @theme directive
- Updated ThemeToggle component with accessibility
- suppressHydrationWarning on root layout
- Smooth CSS transitions between themes
- All existing components theme-aware

**Uses:**
- Tailwind CSS 4.1.14 with @theme syntax (from STACK.md)
- next-themes for industry-standard theme management (from STACK.md)
- HSL color tokens for automatic dark mode adaptation (from ARCHITECTURE.md)

**Implements:**
- Theme System component from ARCHITECTURE.md
- WCAG 2.2 Level AA accessibility requirements (from FEATURES.md)

**Avoids:**
- Hydration mismatch (PITFALLS.md #3) via suppressHydrationWarning + mounted checks
- Tailwind v4 config breaking changes (PITFALLS.md #4) via CSS @theme directive
- Theme flash on page load via next-themes script injection

**Research flag:** Standard theme implementation patterns; consult next-themes docs if hydration issues arise.

### Phase 3: Quarterly Rankings Navigation
**Rationale:** Builds on theme foundation and backend API. Dynamic routing depends on stable data layer. This is the user-facing feature that delivers milestone value.

**Delivers:**
- `/rankings/[year]/[quarter]/page.tsx` dynamic routes
- `/rankings/all-time/page.tsx` for lifetime stats
- QuarterSelector navigation component
- LeaderboardTable component with theme-aware styling
- URL state sync for shareable links (e.g., `/rankings/2026/q1`)
- Static generation for past quarters, ISR for current quarter
- Date range calculation utilities

**Uses:**
- Next.js App Router dynamic segments (from ARCHITECTURE.md)
- Leaderboard aggregation endpoint (from Phase 1)
- Theme tokens and dark mode support (from Phase 2)

**Implements:**
- Quarterly Navigation component (from ARCHITECTURE.md)
- Table stakes features: quarterly selector, default to current quarter (from FEATURES.md)
- Should-have features: year selector, URL state sync (from FEATURES.md)

**Avoids:**
- Unnecessary Route Handlers (PITFALLS.md #8) by fetching in Server Components
- Overusing "use client" (PITFALLS.md #14) by keeping most components server-rendered
- N+1 user metadata queries by batching in leaderboard endpoint

**Research flag:** Standard App Router patterns; skip phase-specific research.

### Phase Ordering Rationale

- **Backend first prevents rework:** Discovering leaderboard API needs query restructuring during Phase 3 would force Phase 1 rework. Build data layer upfront.
- **Theme before UI prevents cascading changes:** Quarterly components depend on theme tokens. Adding theming late requires updating all quarterly UI components retroactively.
- **Dependency updates isolated:** Separating updates from features makes rollback easier if issues arise. Clear boundary between infrastructure and product work.
- **Each phase independently testable:** Phase 1 validates via API tests, Phase 2 via theme toggle/persistence, Phase 3 via quarterly navigation flows.
- **Matches pitfall severity:** Critical pitfalls (hydration, async params) addressed in early phases before they can block feature development.

### Research Flags

**Phases with standard patterns (skip research-phase):**
- **Phase 1:** Dependency updates follow documented upgrade paths. Docker optimization is well-documented. Go patch updates are low-risk.
- **Phase 2:** next-themes has extensive documentation and examples. Tailwind theming is well-covered in official docs. Hydration mitigation is standard pattern.
- **Phase 3:** Next.js App Router dynamic routes are thoroughly documented. Leaderboard UI patterns are common in dashboard applications.

**No phases need deeper research.** All patterns are industry-standard with high-confidence documentation. Proceed directly to requirements definition and phase planning.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All versions verified from official sources (GitHub releases, Docker Hub, npm) within past 4 days. Next.js 16 breaking changes documented in official upgrade guide. |
| Features | HIGH | Theme switching patterns verified across multiple authoritative sources (CSS-Tricks, Smashing Magazine, MDN). WCAG 2.2 accessibility requirements official. Quarterly navigation patterns from dashboard UX guides. |
| Architecture | HIGH | Current BeerBot codebase analyzed. Next.js 15 App Router patterns from official docs. Tailwind dark mode class-based approach documented. Go SQLite query patterns verified. |
| Pitfalls | HIGH | Next.js 16 breaking changes from official docs. next-themes hydration fixes from official library docs + verified community patterns. Docker multi-stage best practices from official Docker docs. |

**Overall confidence:** HIGH

All core recommendations based on official documentation (Next.js, React, Tailwind, Docker, Go) published in January 2026 or verified community patterns with multiple authoritative sources. next-themes version 0.4.6 published ~10 months ago (March 2025) is the only dependency not cutting-edge, but it's widely adopted (2,172+ projects) and stable.

### Gaps to Address

Minor gaps exist where implementation details need validation during development, but none affect architectural decisions.

- **Beer color palette specifics** — Research provides HSL token structure and color strategy, but exact amber/golden/copper values need design review. Recommendation: Use UI Colors palette generator to create 11-step palettes from chosen base colors. Verify WCAG AA contrast ratios (4.5:1 minimum) during Phase 2.

- **Leaderboard user metadata caching** — Backend currently fetches Slack user info on-demand. Research identified N+1 query anti-pattern but didn't specify caching strategy. Recommendation: Decide during Phase 1 whether to cache in SQLite, Redis, or fetch batch during leaderboard aggregation.

- **Current BeerBot theme implementation details** — Research identified custom ThemeProvider exists but didn't analyze full implementation. Recommendation: During Phase 2 kickoff, audit existing ThemeProvider to determine migration path (keep custom vs migrate to next-themes).

- **Quarterly data volume** — Performance recommendations assume "thousands of rows" but actual BeerBot data volume unknown. Recommendation: Query production database during Phase 1 to validate index strategy. If <10k rows, index is optional; if >100k rows, consider denormalizing year/quarter columns.

## Sources

### Primary (HIGH confidence)

**Official Documentation:**
- [Next.js Releases](https://github.com/vercel/next.js/releases) — Version numbers, release notes
- [Next.js 16 Upgrade Guide](https://nextjs.org/docs/app/guides/upgrading/version-16) — Breaking changes (async params, Turbopack, next lint removal)
- [Next.js App Router Dynamic Routes](https://nextjs.org/docs/app/api-reference/file-conventions/dynamic-routes) — Dynamic segment patterns
- [Tailwind CSS v4.0](https://tailwindcss.com/blog/tailwindcss-v4) — v4 theme syntax
- [Tailwind CSS Theme Documentation](https://tailwindcss.com/docs/theme) — @theme directive usage
- [Tailwind CSS Dark Mode](https://tailwindcss.com/docs/dark-mode) — Class-based dark mode
- [Go Downloads](https://go.dev/dl/) — Version 1.25.6 released Jan 28, 2026
- [Go Modules Major Version Handling](https://go.dev/doc/modules/major-version) — Import path versioning
- [Docker Build Best Practices](https://docs.docker.com/build/building/best-practices/) — Layer ordering, cache optimization
- [Docker Hub: golang](https://hub.docker.com/_/golang) — golang:1.25.6-alpine3.23 image
- [Docker Hub: alpine](https://hub.docker.com/_/alpine) — alpine:3.23 image (Jan 27, 2026)
- [Docker Hub: oven/bun](https://hub.docker.com/r/oven/bun/tags) — bun:1.3.7 image (Jan 27, 2026)
- [WCAG 2.2 Guidelines](https://www.w3.org/TR/WCAG21/) — Accessibility standards

**Library Documentation:**
- [next-themes on npm](https://www.npmjs.com/package/next-themes) — Version 0.4.6 usage
- [next-themes GitHub](https://github.com/pacocoursey/next-themes) — Hydration prevention patterns
- [Bun Releases](https://github.com/oven-sh/bun/releases) — Bun 1.3.7 changelog
- [lucide-react npm](https://www.npmjs.com/package/lucide-react) — Version 0.563.1

### Secondary (MEDIUM confidence)

**Best Practices Guides:**
- [Theme Colors with Tailwind v4 and Next Themes](https://medium.com/@kevstrosky/theme-colors-with-tailwind-css-v4-0-and-next-themes-dark-light-custom-mode-36dca1e20419) — Custom theme implementation
- [Next.js 15 to 16 Migration Guide](https://medium.com/@mernstackdevbykevin/next-js-15-to-16-your-complete-migration-playbook-6a7631e6cc3d) — Community migration experiences
- [Tailwind CSS v4 Complete Migration Guide](https://medium.com/@mernstackdevbykevin/tailwind-css-v4-0-complete-migration-guide-breaking-changes-you-need-to-know-7f99944a9f95) — Breaking changes walkthrough
- [Fixing Hydration Mismatch in next-themes](https://medium.com/@pavan1419/fixing-hydration-mismatch-in-next-js-next-themes-issue-8017c43dfef9) — Verified community solution
- [Common Mistakes with Next.js App Router - Vercel](https://vercel.com/blog/common-mistakes-with-the-next-js-app-router-and-how-to-fix-them) — Route Handler anti-patterns
- [Go + SQLite Best Practices](https://jacob.gold/posts/go-sqlite-best-practices/) — Query optimization
- [Docker Multi-Stage for Go (Jan 2026)](https://oneuptime.com/blog/post/2026-01-07-go-docker-multi-stage/view) — Recent tutorial
- [Creating Dynamic Routes in Next.js 15 with App Router](https://medium.com/@sehouli.hamza/creating-dynamic-routes-in-nextjs-15-with-app-router-e8cc15e401d0) — Practical examples

**UX/Design Resources:**
- [From Data To Decisions: UX Strategies For Real-Time Dashboards](https://www.smashingmagazine.com/2025/09/ux-strategies-real-time-dashboards/) — Dashboard time filtering patterns
- [Filter UX Design Patterns & Best Practices](https://www.pencilandpaper.io/articles/ux-pattern-analysis-enterprise-filtering) — Time period selection
- [Best Practices for Designing Leaderboards](https://www.sportfitnessapps.com/blog/best-practices-for-designing-leaderboards) — Leaderboard time filtering
- [UI Colors Palette Generator](https://uicolors.app/generate) — Tool for beer color palettes
- [Tints.dev](https://www.tints.dev/) — Tailwind 11-color palette generator

### Tertiary (LOW confidence)

**Community Reports:**
- [React 19 Compatibility Issues (Jan 2026)](https://medium.com/@quicksilversel/i-upgraded-three-apps-to-react-19-heres-what-broke-648087c7217b) — Real-world upgrade experience
- [Bun Workspace Issues](https://github.com/oven-sh/bun/issues/25014) — Known monorepo limitations
- [Is Bun Production-Ready in 2026?](https://dev.to/last9/is-bun-production-ready-in-2026-a-practical-assessment-181h) — Community assessment

---
*Research completed: 2026-01-31*
*Ready for roadmap: yes*
