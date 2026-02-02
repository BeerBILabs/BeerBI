# Phase 2: Beer-Branded Theming - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can toggle between light and dark themes with beer-branded colors. The toggle already exists in the header — this phase styles it and implements the theming system with proper persistence.

</domain>

<decisions>
## Implementation Decisions

### Toggle placement & style
- Keep existing toggle in header/navbar — no relocation needed
- Keep existing toggle form (icon button or switch) — just restyle it
- Add tooltip on hover showing "Switch to dark/light mode"
- Claude's discretion: toggle animation when switching (fade, rotate, etc.)

### Color palette
- Light mode: Subtle warmth — mostly neutral with warm amber accents, professional feel with hint of beer
- Dark mode: Deep brown/black — rich dark browns like a stout, warm dark mode feel
- Primary accent color: Amber/gold — beer-colored accents in both themes
- No specific brand reference — generic beer tones

### Theme transitions
- Smooth fade transition (~200-300ms) when switching themes
- No images or charts to handle — just text and UI elements
- Subtle pulse/glow effect on toggle when theme switch is triggered
- Claude's discretion: whether borders/shadows/dividers transition or swap instantly

### Persistence & first visit
- Default to light mode on first visit (ignore system preference initially)
- User's manual choice always overrides — no "auto" option, simple two-state toggle
- Store preference in cookie (enables server-side reading for SSR)
- FOUC prevention: nice to have but not critical — brief flash acceptable if it simplifies implementation

### Claude's Discretion
- Toggle icon animation style and timing
- Whether all visual elements (borders, shadows) fade together or only main colors
- Exact transition duration within the ~200-300ms range
- FOUC prevention approach — balance complexity vs benefit

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. Generic beer tones, not tied to any brand.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-beer-branded-theming*
*Context gathered: 2026-02-02*
