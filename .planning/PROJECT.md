# BeerBot Improvements

## What This Is

BeerBot is a Slack bot that tracks beer-related recognition between team members. When someone reacts with a beer emoji in a designated Slack channel, the bot records it and maintains leaderboards showing who's given and received the most. The Next.js frontend displays these leaderboards with date range filtering. This milestone modernizes the stack and adds new UI features.

## Core Value

The leaderboard must accurately reflect beer transactions — if someone gives or receives a beer, it must be counted and displayed correctly.

## Requirements

### Validated

- ✓ Beer giving/receiving tracked via Slack emoji reactions — existing
- ✓ Leaderboard displays givers and recipients with counts — existing
- ✓ Date range filtering (quick presets + custom) — existing
- ✓ User avatars and names fetched from Slack — existing
- ✓ API proxy handles authentication to backend — existing

### Active

- [ ] Dependencies updated to latest stable (Go, Docker, Next.js, Bun)
- [ ] UI buttons are theme-optimized and consistent
- [ ] Proper theme switch with beer-branded colors (light/dark)
- [ ] Quarterly ranking subpages (Q1-Q4 for current and last year)

### Out of Scope

- Cloud-hosted LLM — must be local for privacy/cost
- Mobile app — web-first
- Real-time updates — current polling model is sufficient
- Multi-workspace support — single Slack workspace

## Context

**Current stack:**
- Backend: Go with SQLite, Slack Socket Mode
- Frontend: Next.js 16.1.6, React 19.2.4, Tailwind CSS, Bun
- Infrastructure: Docker containers

**Existing theme support:**
- `next-themes` already installed (0.4.6)
- Tailwind dark mode configured via class
- Theme toggle needs to be added to UI

**Data availability:**
- Backend stores timestamps with each beer transaction
- Quarterly aggregation can use existing `beers` table with date range queries
- May need new backend endpoints for quarterly data

## Constraints

- **Local LLM**: AI features must use local inference (not cloud APIs)
- **Visual identity**: Beer/brand themed colors, not generic light/dark
- **Backward compatibility**: Existing leaderboard functionality must not break

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Update to latest stable versions | Stay current, security patches, new features | — Pending |
| Beer-themed color palette | Brand identity, more engaging than generic themes | — Pending |
| Quarterly rankings as subpages | Clean navigation, dedicated views per quarter | — Pending |
| Local LLM for v2 | Privacy, no API costs, offline capability | — Pending |

---
*Last updated: 2026-01-30 after initialization*
