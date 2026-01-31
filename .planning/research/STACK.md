# Technology Stack - BeerBot v1.0

**Project:** BeerBot - Slack beer recognition tracker with leaderboard frontend
**Researched:** 2026-01-31
**Milestone:** v1.0 - Theming and Quarterly Rankings Enhancement

## Executive Summary

BeerBot v1.0 is an enhancement milestone for an existing, validated application. Current stack is Next.js 15.5.6 with React 19.2.0, Go 1.25 backend, and Docker deployment. Research focused on:

1. **Dependency modernization** - Identifying latest stable versions and upgrade paths
2. **Theming enhancement** - Adding next-themes for beer-branded light/dark mode
3. **Version compatibility** - Ensuring ecosystem harmony across frontend/backend

**Recommendation:** STAY on Next.js 15.x (15.5.11) for this milestone. Defer Next.js 16 upgrade to future milestone due to breaking changes in async APIs, middleware renaming, and linting removal that would expand scope beyond theming enhancements.

---

## Current Stack (Baseline)

| Technology | Current Version | Status |
|------------|----------------|--------|
| Next.js | 15.5.6 | Active, maintenance updates available |
| React | 19.2.0 | Latest stable |
| Go | 1.25 (toolchain 1.25.3) | Minor update available (1.25.6) |
| Bun | 1.3.0+ (engine requirement) | Major update available (1.3.7) |
| Tailwind CSS | 4.1.14 | Latest stable (v4.1.x) |
| TypeScript | 5.9.3 | Latest stable |
| lucide-react | 0.546.0 | Minor update available (0.563.1) |

---

## Recommended Stack for v1.0

### Core Framework

| Technology | Version | Purpose | Why | Upgrade Path |
|------------|---------|---------|-----|--------------|
| **Next.js** | **15.5.11** | React framework for frontend | Latest 15.x with bug fixes. AVOID 16.x due to breaking changes in async APIs, middleware→proxy rename, next lint removal | Minor update from 15.5.6 → 15.5.11 (backported bug fixes) |
| **React** | **19.2.0** | UI library | Already at latest stable. Fully compatible with Next.js 15.x | No change needed |
| **React DOM** | **19.2.0** | React renderer | Matches React version | No change needed |
| **Bun** | **1.3.7** | JavaScript runtime & package manager | Latest stable with HMR improvements, dependency catalogs for monorepos, interactive updates | Update from 1.3.0+ → 1.3.7 |
| **TypeScript** | **5.9.3** | Type safety | Latest stable. TypeScript 6/7 in development but not production-ready | No change needed |

**Rationale for staying on Next.js 15.x:**
- Next.js 16.1 introduces major breaking changes that would expand milestone scope
- Async-only request APIs require codebase audit and refactoring
- `next lint` removal requires new ESLint setup
- Middleware→Proxy rename affects architecture
- 15.5.11 is actively maintained with backported bug fixes
- Theming features work identically on 15.x and 16.x

### Backend

| Technology | Version | Purpose | Why | Upgrade Path |
|------------|---------|---------|-----|--------------|
| **Go** | **1.25.6** | Backend service language | Latest stable (released Jan 28, 2026) with security fixes and bug fixes | Minor update from 1.25.3 → 1.25.6 |
| **go-sqlite3** | **1.14.32** | SQLite driver | Already at latest | No change needed |
| **slack-go/slack** | **0.17.3** | Slack SDK for Socket Mode | Already current | No change needed |
| **zerolog** | **1.34.0** | Structured logging | Already current | No change needed |

### Styling & Theming

| Technology | Version | Purpose | Why | Notes |
|------------|---------|---------|-----|-------|
| **Tailwind CSS** | **4.1.14** | Utility-first CSS framework | Already at latest v4.x with modern CSS features (cascade layers, @property, color-mix()) | Already current |
| **@tailwindcss/postcss** | **4.1.14** | PostCSS integration | Matches Tailwind version | Already current |
| **next-themes** | **0.4.6** | Theme switching library | **NEW ADDITION** - Industry standard for Next.js theme management. 2,172+ projects use it. No flash, system preference support, perfect for light/dark beer themes | Add to dependencies |

**Why next-themes:**
- Zero-configuration theme switching
- No flash of incorrect theme (critical for UX)
- System preference detection
- Works with Tailwind's dark mode variants
- Supports custom themes beyond light/dark (future beer style themes)
- Small bundle size
- Compatible with both Next.js 15 and 16

### UI & Icons

| Technology | Version | Purpose | Why | Upgrade Path |
|------------|---------|---------|-----|--------------|
| **lucide-react** | **0.563.1** | Icon library | Latest with accessibility improvements (aria-hidden attributes added) | Minor update from 0.546.0 → 0.563.1 |
| **react-datepicker** | **5.1.0** | Date selection for quarterly views | Already current | No change needed |

### Development Tools

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **@types/react** | **19.2.2** | React TypeScript definitions | Matches React 19.2 |
| **@types/node** | **24.8.1** | Node.js TypeScript definitions | Current for Bun runtime |
| **ESLint** | **9.x** | Linting (now standalone) | Required since next lint deprecated |
| **eslint-config-next** | **15.5.6** | Next.js ESLint rules | Matches Next.js version |
| **tw-animate-css** | **1.4.0** | Tailwind animation utilities | Already current |

---

## Infrastructure & Deployment

### Docker Base Images

| Component | Current Image | Recommended Image | Reasoning |
|-----------|--------------|-------------------|-----------|
| **Backend Builder** | golang:1.25-alpine | **golang:1.25.6-alpine3.23** | Latest Go 1.25 with latest Alpine (3.23.3 released Jan 27, 2026) for security patches |
| **Backend Runtime** | alpine:3.20 | **alpine:3.23** | Latest Alpine (3.23.3) with recent security updates |
| **Frontend** | oven/bun:1 | **oven/bun:1.3.7** | Latest Bun 1.3.7 (released Jan 27, 2026) with performance improvements |

**Alpine version progression:**
- Alpine 3.20.9, 3.21.6, 3.22.3, and 3.23.3 all released Jan 27, 2026
- 3.23 is newest stable series
- Minimal size (~5 MB) maintained

---

## New Dependencies to Install

### Frontend

```bash
# Add theming support
bun add next-themes@0.4.6

# Update existing packages
bun update lucide-react@0.563.1
bun update next@15.5.11
```

### Backend

```bash
# Update Go version in go.mod
go 1.25.6

# Update dependencies
go get -u github.com/mattn/go-sqlite3
go get -u github.com/slack-go/slack
go get -u github.com/rs/zerolog
go get -u github.com/prometheus/client_golang
```

---

## Beer-Themed Color Customization

### Tailwind CSS v4 Custom Theme Approach

Tailwind v4 uses the new `@theme` directive for custom colors. For beer-branded themes:

```css
/* In your main CSS file */
@theme {
  --color-amber: oklch(0.75 0.15 65);        /* Amber ale */
  --color-golden: oklch(0.85 0.12 85);       /* Golden lager */
  --color-stout: oklch(0.25 0.05 30);        /* Dark stout */
  --color-porter: oklch(0.35 0.08 35);       /* Porter brown */
  --color-ipa: oklch(0.65 0.18 55);          /* IPA copper */
  --color-wheat: oklch(0.90 0.08 90);        /* Wheat beer */
}
```

**Tools for palette generation:**
- [UI Colors](https://uicolors.app/generate) - Generate full 11-step palettes from custom colors
- [Tints.dev](https://www.tints.dev/) - Tailwind 11-color palette generator
- CSS `color-mix()` function (built into Tailwind v4)

### next-themes Integration

```typescript
// app/providers.tsx
import { ThemeProvider } from 'next-themes'

export function Providers({ children }) {
  return (
    <ThemeProvider
      attribute="data-theme"
      defaultTheme="system"
      themes={['light', 'dark', 'amber', 'stout']}
      enableSystem
    >
      {children}
    </ThemeProvider>
  )
}
```

**Custom themes beyond light/dark:**
- Amber theme: Light background with amber/golden accents
- Stout theme: Dark background with porter/stout browns
- Future: Per-beer-style themes (IPA, Wheat, etc.)

---

## Upgrade Considerations

### What to Update Immediately

1. **Go 1.25.3 → 1.25.6** - Security and bug fixes (Jan 28, 2026 release)
2. **Next.js 15.5.6 → 15.5.11** - Backported bug fixes, LRU cache fix
3. **Bun 1.3.0+ → 1.3.7** - Performance and stability improvements
4. **lucide-react 0.546.0 → 0.563.1** - Accessibility improvements (aria-hidden)
5. **Docker base images** - Alpine 3.20 → 3.23, golang:1.25 → 1.25.6-alpine3.23

### What to Defer

1. **Next.js 16.x** - Breaking changes requiring:
   - Async-only APIs for params/cookies/headers/searchParams
   - Middleware → Proxy renaming (edge runtime removed from proxy)
   - next lint removal (ESLint reconfiguration needed)
   - Turbopack config migration
   - Image metadata async function signatures
   - PPR behavioral changes

   **Defer to:** Future milestone after v1.0 feature work complete

2. **TypeScript 6.x/7.x** - Still in development, not production-ready

---

## Installation Steps

### 1. Update Frontend Dependencies

```bash
cd frontend/project

# Add new theming library
bun add next-themes@0.4.6

# Update to latest stable versions
bun add next@15.5.11
bun add lucide-react@0.563.1

# Verify all dependencies updated
bun update
```

### 2. Update Backend Dependencies

```bash
cd backend/bot

# Update Go version in go.mod
# Change: go 1.25
# To: go 1.25.6

# Update toolchain
# Change: toolchain go1.25.3
# To: toolchain go1.25.6

# Update dependencies
go get -u=patch ./...
go mod tidy
```

### 3. Update Docker Base Images

**Backend Dockerfile:**
```dockerfile
# Step 1: Build the application
FROM golang:1.25.6-alpine3.23 AS builder
# ... rest unchanged

# Step 2: Create the final image
FROM alpine:3.23
# ... rest unchanged
```

**Frontend Dockerfile:**
```dockerfile
# use the official Bun image
FROM oven/bun:1.3.7 AS base
# ... rest unchanged
```

---

## Version Compatibility Matrix

| Frontend | Backend | Runtime | Docker | Status |
|----------|---------|---------|--------|--------|
| Next.js 15.5.11 | Go 1.25.6 | Bun 1.3.7 | Alpine 3.23 | Recommended ✓ |
| React 19.2.0 | SQLite | Node.js (Bun compatible) | golang:1.25.6-alpine3.23 | Compatible ✓ |
| Tailwind 4.1.14 | Slack SDK 0.17.3 | - | oven/bun:1.3.7 | Compatible ✓ |
| next-themes 0.4.6 | - | - | - | Compatible ✓ |

---

## What NOT to Update (And Why)

| Technology | Current | Latest | Why NOT to Update |
|------------|---------|--------|-------------------|
| Next.js | 15.5.11 | 16.1.2 | Breaking changes in async APIs, middleware→proxy rename, next lint removal. Scope creep for theming milestone. |
| TypeScript | 5.9.3 | 6.0/7.0 dev | TypeScript 6/7 still in development, not production-ready |
| React | 19.2.0 | - | Already at latest |

---

## Risk Assessment

| Technology | Update Risk | Mitigation |
|------------|------------|------------|
| Next.js 15.5.6 → 15.5.11 | LOW | Patch release with bug fixes only |
| Go 1.25.3 → 1.25.6 | LOW | Patch release with security fixes |
| Bun 1.3.0+ → 1.3.7 | LOW | Patch release in same major version |
| Docker Alpine 3.20 → 3.23 | LOW | Alpine maintains backwards compatibility |
| next-themes (new) | LOW | Widely adopted (2,172+ projects), stable API |
| lucide-react 0.546.0 → 0.563.1 | VERY LOW | Patch updates with accessibility additions |

---

## Sources

### Next.js
- [Next.js Releases](https://github.com/vercel/next.js/releases)
- [Next.js 16](https://nextjs.org/blog/next-16)
- [Next.js 16.1](https://nextjs.org/blog/next-16-1)
- [Upgrading to Version 16](https://nextjs.org/docs/app/guides/upgrading/version-16)
- [Next.js 15 to 16 Migration Guide](https://medium.com/@mernstackdevbykevin/next-js-15-to-16-your-complete-migration-playbook-6a7631e6cc3d)

### Go
- [Go Downloads](https://go.dev/dl/)
- [Go Release History](https://go.dev/doc/devel/release)
- [Go 1.25 Features](https://versionlog.com/golang/1.25/)

### Bun
- [Bun Releases](https://github.com/oven-sh/bun/releases)
- [Bun v3.1 Release](https://www.infoq.com/news/2026/01/bun-v3-1-release/)
- [Is Bun Production-Ready in 2026?](https://dev.to/last9/is-bun-production-ready-in-2026-a-practical-assessment-181h)

### Theming & Styling
- [next-themes on npm](https://www.npmjs.com/package/next-themes)
- [next-themes GitHub](https://github.com/pacocoursey/next-themes)
- [Theme Colors with Tailwind v4 and Next Themes](https://medium.com/@kevstrosky/theme-colors-with-tailwind-css-v4-0-and-next-themes-dark-light-custom-mode-36dca1e20419)
- [Tailwind CSS v4.0](https://tailwindcss.com/blog/tailwindcss-v4)
- [Tailwind CSS Theme Variables](https://tailwindcss.com/docs/theme)
- [Customizing Colors - Tailwind CSS](https://tailwindcss.com/docs/customizing-colors)
- [UI Colors Palette Generator](https://uicolors.app/generate)
- [Tints.dev Palette Generator](https://www.tints.dev/)

### Docker & Infrastructure
- [Golang Docker Images](https://hub.docker.com/_/golang)
- [Alpine Docker Images](https://hub.docker.com/_/alpine)
- [Bun Docker Images](https://hub.docker.com/r/oven/bun/tags)

### Icons & UI
- [Lucide React Releases](https://github.com/lucide-icons/lucide/releases)
- [Lucide React npm](https://www.npmjs.com/package/lucide-react)

### TypeScript
- [TypeScript 5.9](https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-9.html)
- [TypeScript Releases](https://github.com/microsoft/typescript/releases)
- [Progress on TypeScript 7](https://devblogs.microsoft.com/typescript/progress-on-typescript-7-december-2025/)

---

## Confidence Assessment

| Category | Confidence | Reasoning |
|----------|-----------|-----------|
| Version Numbers | HIGH | All versions verified from official sources (GitHub releases, Docker Hub, npm) within past 4 days |
| Compatibility | HIGH | Next.js 15.x + React 19.2 is documented stable pairing. Bun 1.3 is production-ready per multiple sources |
| Upgrade Paths | HIGH | Minor version updates with documented changelogs |
| Next.js 16 Breaking Changes | HIGH | Official Next.js upgrade docs and community migration guides |
| next-themes Integration | MEDIUM | Widely adopted (2,172+ projects), but version 0.4.6 published ~10 months ago (March 2025) |
| Docker Images | HIGH | Official Docker Hub with recent publication dates (Jan 27-28, 2026) |
| Beer Theming Approach | MEDIUM | Tailwind v4 custom theming documented, but specific beer color palettes are custom design work |

---

## Summary

For BeerBot v1.0 milestone (theming and quarterly rankings):

**Update:**
- Go 1.25.3 → 1.25.6 (security fixes)
- Next.js 15.5.6 → 15.5.11 (bug fixes)
- Bun 1.3.0+ → 1.3.7 (performance)
- lucide-react 0.546.0 → 0.563.1 (accessibility)
- Docker base images (Alpine 3.20 → 3.23, golang/bun latest)

**Add:**
- next-themes 0.4.6 (theme switching)

**Defer:**
- Next.js 16 (breaking changes expand scope)
- TypeScript 6/7 (not production-ready)

This keeps the milestone focused on theming enhancements while modernizing dependencies safely.
