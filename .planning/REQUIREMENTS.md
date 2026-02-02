# Requirements: BeerBot v1.0

**Defined:** 2026-01-31
**Core Value:** The leaderboard must accurately reflect beer transactions — if someone gives or receives a beer, it must be counted and displayed correctly.

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Dependencies

- [x] **DEP-01**: Next.js updated to 15.5.11 with verified functionality
- [x] **DEP-02**: Go backend updated to 1.25.6 with verified functionality
- [x] **DEP-03**: Bun runtime updated to 1.3.7 with verified functionality
- [x] **DEP-04**: Docker base images updated to Alpine 3.23

### Theming

- [x] **THEME-01**: Theme toggle control visible in UI
- [x] **THEME-02**: Light and dark mode switching works correctly
- [x] **THEME-03**: Theme preference persists in cookies across sessions
- [x] **THEME-04**: First visit defaults to light mode (user decision: no system preference)
- [x] **THEME-05**: No flash of unstyled content on page load (FOUC prevention)
- [x] **THEME-06**: Beer-branded color palette applied (golden/amber light, stout/porter dark)
- [x] **THEME-07**: All buttons consistently styled across both themes

### Quarterly Rankings

- [x] **QRANK-01**: Quarterly ranking pages accessible via navigation (Q1-Q4)
- [x] **QRANK-02**: Quarterly leaderboard displays givers with counts
- [x] **QRANK-03**: Quarterly leaderboard displays recipients with counts
- [x] **QRANK-04**: Year selector allows viewing historical quarters
- [x] **QRANK-05**: URL reflects current quarter selection (shareable links)
- [x] **QRANK-06**: "Current Quarter" shortcut available
- [x] **QRANK-07**: "Last Quarter" shortcut available

## Future Requirements

Deferred to later milestones. Tracked but not in current roadmap.

### Additional Themes

- **THEME-F01**: Additional beer-style themes (IPA green, amber, etc.)
- **THEME-F02**: Theme preview before applying

### Analytics

- **ANAL-F01**: Year-over-year quarterly comparison
- **ANAL-F02**: Trends visualization

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Next.js 16 upgrade | Breaking changes expand scope; defer to future milestone |
| Real-time theme sync across tabs | Over-engineering for single-user context |
| Custom date ranges on quarterly pages | Quarterly presets sufficient; main leaderboard has custom ranges |
| Mobile app | Web-first approach per PROJECT.md |
| Cloud LLM integration | Must be local per constraints |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DEP-01 | Phase 1 | Complete |
| DEP-02 | Phase 1 | Complete |
| DEP-03 | Phase 1 | Complete |
| DEP-04 | Phase 1 | Complete |
| THEME-01 | Phase 2 | Complete |
| THEME-02 | Phase 2 | Complete |
| THEME-03 | Phase 2 | Complete |
| THEME-04 | Phase 2 | Complete |
| THEME-05 | Phase 2 | Complete |
| THEME-06 | Phase 2 | Complete |
| THEME-07 | Phase 2 | Complete |
| QRANK-01 | Phase 3 | Complete |
| QRANK-02 | Phase 3 | Complete |
| QRANK-03 | Phase 3 | Complete |
| QRANK-04 | Phase 3 | Complete |
| QRANK-05 | Phase 3 | Complete |
| QRANK-06 | Phase 3 | Complete |
| QRANK-07 | Phase 3 | Complete |

**Coverage:**
- v1 requirements: 18 total
- Mapped to phases: 18 (100% coverage)
- Unmapped: 0

---
*Requirements defined: 2026-01-31*
*Last updated: 2026-02-02 after Phase 3 completion*
