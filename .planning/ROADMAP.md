# Roadmap: BeerBot v1.0

## Overview

BeerBot v1.0 modernizes the stack and adds themed UI features. Starting with dependency updates to establish a stable foundation, adding a beer-branded theming system with light/dark mode, and finishing with quarterly ranking navigation that lets users explore leaderboards by quarter.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Dependency Modernization** - Update stack to latest stable versions
- [ ] **Phase 2: Beer-Branded Theming** - Add light/dark mode with branded color palette
- [ ] **Phase 3: Quarterly Rankings** - Enable quarter-by-quarter leaderboard navigation

## Phase Details

### Phase 1: Dependency Modernization
**Goal**: All dependencies updated to latest stable versions with verified functionality
**Depends on**: Nothing (first phase)
**Requirements**: DEP-01, DEP-02, DEP-03, DEP-04
**Success Criteria** (what must be TRUE):
  1. Next.js 15.5.11 installed and all existing features work correctly
  2. Go backend updated to 1.25.6 and compiles without errors
  3. Bun runtime updated to 1.3.7 and dev server starts successfully
  4. Docker images use Alpine 3.23 and containers build without errors
  5. All existing leaderboard functionality verified working (date ranges, user data)
**Plans**: TBD

Plans:
- [ ] TBD (pending plan-phase)

### Phase 2: Beer-Branded Theming
**Goal**: Users can toggle between light and dark themes with beer-branded colors
**Depends on**: Phase 1
**Requirements**: THEME-01, THEME-02, THEME-03, THEME-04, THEME-05, THEME-06, THEME-07
**Success Criteria** (what must be TRUE):
  1. User can see theme toggle control in UI and click to switch themes
  2. Theme switches immediately between light and dark modes without page reload
  3. User's theme preference persists when closing and reopening browser
  4. System preference (prefers-color-scheme) automatically applied on first visit
  5. No flash of wrong theme colors appears when page loads
  6. Light mode uses golden/amber beer colors and dark mode uses stout/porter colors
  7. All buttons have consistent styling in both light and dark themes
**Plans**: TBD

Plans:
- [ ] TBD (pending plan-phase)

### Phase 3: Quarterly Rankings
**Goal**: Users can navigate and view leaderboards for any quarter of any year
**Depends on**: Phase 2
**Requirements**: QRANK-01, QRANK-02, QRANK-03, QRANK-04, QRANK-05, QRANK-06, QRANK-07
**Success Criteria** (what must be TRUE):
  1. User can navigate to quarterly pages (Q1, Q2, Q3, Q4) from main UI
  2. Quarterly leaderboard shows top givers with beer counts for that quarter
  3. Quarterly leaderboard shows top recipients with beer counts for that quarter
  4. User can select different years to view historical quarters
  5. URL changes when selecting quarter (e.g., /rankings/2026/q1) and is shareable
  6. User can click "Current Quarter" shortcut to jump to current quarter
  7. User can click "Last Quarter" shortcut to jump to previous quarter
**Plans**: TBD

Plans:
- [ ] TBD (pending plan-phase)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Dependency Modernization | 0/TBD | Not started | - |
| 2. Beer-Branded Theming | 0/TBD | Not started | - |
| 3. Quarterly Rankings | 0/TBD | Not started | - |
