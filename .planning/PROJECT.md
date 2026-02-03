# BeerBot Improvements

## What This Is

BeerBot is a Slack bot that tracks beer-related recognition between team members. When someone reacts with a beer emoji in a designated Slack channel, the bot records it and maintains leaderboards showing who's given and received the most. The Next.js frontend displays these leaderboards with date range filtering, beer-branded light/dark theming, and quarterly ranking navigation.

## Core Value

The leaderboard must accurately reflect beer transactions -- if someone gives or receives a beer, it must be counted and displayed correctly.

## Requirements

### Validated

- ✓ Beer giving/receiving tracked via Slack emoji reactions -- existing
- ✓ Leaderboard displays givers and recipients with counts -- existing
- ✓ Date range filtering (quick presets + custom) -- existing
- ✓ User avatars and names fetched from Slack -- existing
- ✓ API proxy handles authentication to backend -- existing
- ✓ Dependencies updated to latest stable (Go 1.25.6, Docker Alpine 3.23, Next.js 15.5.11, Bun 1.3.7) -- v1.0
- ✓ UI buttons are theme-optimized and consistent -- v1.0
- ✓ Proper theme switch with beer-branded colors (light/dark) -- v1.0
- ✓ Quarterly ranking subpages (Q1-Q4 for current and last year) -- v1.0

### Active

(None -- planning next milestone)

### Out of Scope

- Cloud-hosted LLM -- must be local for privacy/cost
- Mobile app -- web-first
- Real-time updates -- current polling model is sufficient
- Multi-workspace support -- single Slack workspace
- Real-time theme sync across tabs -- over-engineering for single-user context

## Context

**Shipped v1.0** with 5,003 LOC across TypeScript, Go, and CSS.

**Current stack:**
- Backend: Go 1.25.6 with SQLite, Slack Socket Mode
- Frontend: Next.js 15.5.11, React, Tailwind CSS, Bun 1.3.7
- Infrastructure: Docker containers (Alpine 3.23)
- Theming: next-themes with cookie-based persistence, SSR FOUC prevention
- Rankings: Quarterly navigation with year selector and rank change indicators

**Known tech debt:**
- N+1 API calls for user stats in leaderboard (consider batch endpoint)
- No error retry UI for individual user fetch failures
- User cache 7-day TTL with no manual invalidation
- @next/swc 15.5.7 mismatch warning (upstream issue)

## Constraints

- **Local LLM**: AI features must use local inference (not cloud APIs)
- **Visual identity**: Beer/brand themed colors, not generic light/dark
- **Backward compatibility**: Existing leaderboard functionality must not break

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Update to latest stable versions | Stay current, security patches, new features | ✓ Good -- Go 1.25.6, Next.js 15.5.11, Bun 1.3.7, Alpine 3.23 |
| Beer-themed color palette | Brand identity, more engaging than generic themes | ✓ Good -- golden amber light, stout dark |
| Quarterly rankings as subpages | Clean navigation, dedicated views per quarter | ✓ Good -- /rankings/YYYY/qN with year selector |
| Local LLM for v2 | Privacy, no API costs, offline capability | -- Pending |
| Cookie-based theme persistence | SSR support, FOUC prevention | ✓ Good -- Server Action + client dual write |
| Light mode default on first visit | Simpler than system preference detection | ✓ Good -- consistent first experience |
| Exact version pins for Next.js | Deterministic builds, security patches | ✓ Good -- avoided unexpected changes |
| Alpine minor tags, runtime exact pins | Balance auto-patch pulls with reproducibility | ✓ Good -- established base image pattern |
| ARIA tablist for quarter navigation | Keyboard accessibility, screen reader support | ✓ Good -- WCAG compliance |

---
*Last updated: 2026-02-03 after v1.0 milestone*
