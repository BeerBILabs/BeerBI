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

- [ ] **THEME-01**: Theme toggle control visible in UI
- [ ] **THEME-02**: Light and dark mode switching works correctly
- [ ] **THEME-03**: Theme preference persists in localStorage across sessions
- [ ] **THEME-04**: System preference detected on first visit (respects prefers-color-scheme)
- [ ] **THEME-05**: No flash of unstyled content on page load (FOUC prevention)
- [ ] **THEME-06**: Beer-branded color palette applied (golden/amber light, stout/porter dark)
- [ ] **THEME-07**: All buttons consistently styled across both themes

### Quarterly Rankings

- [ ] **QRANK-01**: Quarterly ranking pages accessible via navigation (Q1-Q4)
- [ ] **QRANK-02**: Quarterly leaderboard displays givers with counts
- [ ] **QRANK-03**: Quarterly leaderboard displays recipients with counts
- [ ] **QRANK-04**: Year selector allows viewing historical quarters
- [ ] **QRANK-05**: URL reflects current quarter selection (shareable links)
- [ ] **QRANK-06**: "Current Quarter" shortcut available
- [ ] **QRANK-07**: "Last Quarter" shortcut available

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
| THEME-01 | Phase 2 | Pending |
| THEME-02 | Phase 2 | Pending |
| THEME-03 | Phase 2 | Pending |
| THEME-04 | Phase 2 | Pending |
| THEME-05 | Phase 2 | Pending |
| THEME-06 | Phase 2 | Pending |
| THEME-07 | Phase 2 | Pending |
| QRANK-01 | Phase 3 | Pending |
| QRANK-02 | Phase 3 | Pending |
| QRANK-03 | Phase 3 | Pending |
| QRANK-04 | Phase 3 | Pending |
| QRANK-05 | Phase 3 | Pending |
| QRANK-06 | Phase 3 | Pending |
| QRANK-07 | Phase 3 | Pending |

**Coverage:**
- v1 requirements: 18 total
- Mapped to phases: 18 (100% coverage)
- Unmapped: 0

---
*Requirements defined: 2026-01-31*
*Last updated: 2026-02-01 after Phase 1 completion*
