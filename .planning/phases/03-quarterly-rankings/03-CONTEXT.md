# Phase 3: Quarterly Rankings - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can navigate and view leaderboards for any quarter of any year. Includes quarter tabs, year selection, rank change indicators, and shortcut buttons for current/last quarter. Main page remains focused on current month — quarterly rankings live on separate `/rankings` route.

</domain>

<decisions>
## Implementation Decisions

### Navigation pattern
- Separate page at `/rankings` (not tabs on main page)
- Entry points: header link + contextual button near main leaderboard
- Main page stays focused on current month only
- `/rankings` page has horizontal tabs: **All Time | Q1 | Q2 | Q3 | Q4**
- Year selector dropdown above/alongside tabs (applies to Q1-Q4 only)
- All Time tab ignores year selector (shows lifetime totals)
- URL structure: `/rankings/2026/q1` for quarters, `/rankings/all` for all-time
- `/rankings` (bare) redirects to current quarter

### Quarter display
- Same two-column layout as main leaderboard: Top Givers + Top Recipients
- Rank change indicator compared to previous quarter:
  - Single arrow (↑/↓) for 1-2 places moved
  - Double arrow (⇑/⇓) for 3+ places moved
- Previous quarter comparison: Q2→Q1, Q1→Q4 of prior year

### Year selection
- Dropdown menu for year selection
- Shows all years with transaction data (no arbitrary limit)
- Future quarters are hidden from tabs (only show completed + current)
- Direct URL to future quarter returns 404

### Shortcuts behavior
- "Current Quarter" and "Last Quarter" buttons above the tab bar
- Styled as buttons matching beer theme
- Show active state when viewing that quarter

### Claude's Discretion
- How to handle users new to the leaderboard this quarter (badge vs no indicator)
- Edge case handling when current quarter just started (shortcut button logic)
- Exact placement of year dropdown relative to tabs
- Loading and error states

</decisions>

<specifics>
## Specific Ideas

- Main page = current month focus; rankings page = all quarterly/yearly exploration
- Rank change arrows should make big movers visually obvious (double arrow for 3+ places)
- Keep same visual language as existing leaderboard for familiarity

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-quarterly-rankings*
*Context gathered: 2026-02-02*
