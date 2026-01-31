# Feature Landscape: Theming & Quarterly Navigation

**Domain:** Leaderboard dashboard with branded theming and time-period navigation
**Researched:** 2026-01-31
**Focus:** Enhancement milestone adding theme switching and quarterly rankings to existing BeerBot

## Table Stakes

Features users expect in modern dashboards with theming and time navigation. Missing these makes the product feel incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Light/Dark theme toggle** | Standard in 2026 UX; users expect system preference support | Low | Must respect `prefers-color-scheme` media query |
| **Theme persistence** | Users expect theme choice to persist across sessions | Low | localStorage is standard; 5-10MB available |
| **No flash on load (FOUC)** | White flash on dark mode destroys trust in implementation | Medium | Script must run before render; critical for perceived quality |
| **Accessible theme toggle** | WCAG 2.2 Level AA compliance required in 2026 | Low | `aria-pressed` state, keyboard navigation, clear visual feedback |
| **Preset time periods** | Users expect "this week," "last week," "month to date" shortcuts | Low | Manual date picking without presets feels tedious |
| **Current selection visibility** | Users must see what time period is active | Low | Visible even after scrolling; supports keyboard navigation |
| **Clear all filters** | Ability to reset selections is critical for multi-filter UIs | Low | Single action to return to default view |
| **Responsive theme on all viewports** | Theme must work on mobile, tablet, desktop | Medium | Colors, contrast, touch targets must scale |

## Differentiators

Features that would make BeerBot's theming stand out. Not expected, but create delight.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Beer-branded color themes** | Multiple branded palettes (not just light/dark) creates personality | Medium | Lager (golden), Stout (dark brown), IPA (hoppy green), etc. |
| **Quarterly color coding** | Visual differentiation: Q1=winter blue, Q2=spring green, Q3=summer yellow, Q4=fall orange | Low | Helps users mentally map quarters to seasons |
| **Theme preview on hover** | See theme before committing to switch | Medium | Reduces friction; users explore themes confidently |
| **Seasonal theme auto-switch** | Automatically switch to quarterly-appropriate theme | Low | Optional; respects user's manual selection if set |
| **Year-over-year comparison view** | Show current Q2 2026 vs Q2 2025 side-by-side | High | Powerful for competitive leaderboards |
| **Smooth theme transitions** | Animated color transitions (200-300ms) feel polished | Low | CSS transitions on custom properties; avoid jarring switches |
| **Theme-aware charts/visualizations** | Leaderboard bars, avatars adapt to theme colors | Medium | Ensures visual coherence across all components |
| **Quick quarter navigation** | Tab-style Q1/Q2/Q3/Q4 selector for rapid switching | Low | More efficient than date picker for quarterly views |

## Anti-Features

Features to explicitly NOT build. Common mistakes in this domain.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Auto-switching theme based on time of day** | Users set preferences deliberately; auto-switching breaks trust | Respect user choice; suggest theme on first visit only |
| **Too many theme variants (10+ options)** | Decision paralysis; maintenance nightmare | Limit to 3-5 well-designed branded themes |
| **Hourly/daily time granularity for quarters** | Quarterly view doesn't need day-level precision; creates clutter | Show aggregate quarterly stats; link to detailed view if needed |
| **Animated theme switcher icons** | Distracting; feels gimmicky in professional dashboard | Simple, clear toggle with immediate visual feedback |
| **Forcing users to select theme on first load** | Interrupts onboarding flow; feels intrusive | Default to system preference or sensible light mode |
| **Date ranges that don't align with quarters** | Confusion: "Q1" should always mean Jan-Mar, not custom ranges | Use calendar quarters (Q1: Jan-Mar, Q2: Apr-Jun, Q3: Jul-Sep, Q4: Oct-Dec) |
| **Excessive data refresh rates** | Creates cognitive strain; unnecessary for leaderboard data | Refresh on user action or reasonable intervals (5-10 min) |
| **Hidden filter state after selection** | Users lose context of what they're viewing | Always show active filters prominently; allow inline editing |
| **Toggle buttons for actions requiring confirmation** | Misuse of toggle pattern; bad UX | Toggles for binary states only (theme on/off); buttons for navigation |
| **Resetting filters when adding new filter** | Forces users to start over; creates frustration | Preserve existing selections; allow cumulative filtering |
| **Complex custom date pickers without presets** | Forces unnecessary clicks; slows down common tasks | Lead with presets; offer custom as secondary option |
| **Ignoring WCAG contrast ratios in custom themes** | Legal risk; excludes users with vision impairments | Test all theme colors for WCAG 2.2 Level AA compliance (4.5:1 minimum) |

## Feature Dependencies

```
Theme System Dependencies:
- CSS Custom Properties (foundation)
  ├─> localStorage persistence (requires browser API)
  ├─> System preference detection (requires media query)
  ├─> FOUC prevention script (must run before React mount)
  └─> Theme toggle component (UI layer)

Quarterly Navigation Dependencies:
- Date calculation logic (foundation)
  ├─> Current quarter detection
  ├─> Year selection (needed for historical quarters)
  ├─> URL state sync (for shareable links)
  └─> Quarter selector component (UI layer)

Integration Dependencies:
- Theme-aware components (must consume theme context)
  ├─> Leaderboard bars/charts
  ├─> Avatar borders/backgrounds
  └─> Date filter UI
```

## Behavioral Patterns Users Expect

### Theme Switching Behavior

**On first visit:**
1. Detect system preference (`prefers-color-scheme`)
2. Apply matching theme without flash
3. Store choice in localStorage

**On subsequent visits:**
1. Read localStorage first (overrides system preference)
2. Apply stored theme before render
3. Show theme toggle in consistent location (usually top-right)

**On theme toggle click:**
1. Immediate visual change (no loading state needed)
2. Smooth CSS transition (200-300ms)
3. Update localStorage
4. Visual feedback on toggle button (pressed state)

**Expected toggle UI:**
- Radio buttons for 2 options (light/dark)
- Dropdown or segmented control for 3+ options
- Clear labels ("Light," "Dark," "Lager Theme")
- Keyboard accessible (Tab to focus, Enter/Space to activate)
- ARIA attributes (`aria-pressed` or `aria-current`)

### Quarterly Navigation Behavior

**Default view:**
- Show current quarter on page load
- Clearly indicate which quarter is active
- Provide context (e.g., "Q1 2026: Jan - Mar")

**Quarter selection:**
- Tab-style navigation for Q1/Q2/Q3/Q4
- Year selector for historical data
- Visual indicator for active quarter (highlight, underline)
- Update URL for shareable links (e.g., `/leaderboard?q=Q2&y=2026`)

**Expected navigation patterns:**
- Horizontal tabs: `[Q1] [Q2] [Q3] [Q4]`
- Year dropdown or arrows: `< 2025 | 2026 >`
- Breadcrumb trail: `Home > Leaderboard > 2026 > Q2`
- Preset shortcuts: "Current Quarter," "Last Quarter," "Same Quarter Last Year"

**Data loading behavior:**
- Show loading state when switching quarters
- Preserve filter selections across quarter changes (if sensible)
- Handle missing data gracefully (e.g., "No data for Q4 2025")

## Implementation Patterns

### CSS Custom Properties for Branded Themes

**Three-tier color architecture:**
1. **Palette level** - Raw colors (hex/rgb values)
2. **Functional level** - Semantic tokens (--primary, --background, --text)
3. **Component level** - Local scope (--button-bg, --card-border)

**Example structure:**
```css
/* Palette (Lager theme) */
:root[data-theme="lager"] {
  --palette-gold: #F4C430;
  --palette-amber: #FFBF00;
  --palette-cream: #FFFDD0;
  --palette-brown: #8B4513;
}

/* Functional (semantic tokens) */
:root[data-theme="lager"] {
  --primary-color: var(--palette-gold);
  --background-color: var(--palette-cream);
  --text-color: var(--palette-brown);
  --border-color: var(--palette-amber);
}

/* Component usage */
.leaderboard-card {
  background: var(--background-color);
  border: 2px solid var(--border-color);
  color: var(--text-color);
}
```

**Color variation technique:**
- Use `linear-gradient` overlays for darker variants
- Avoid creating separate variables for every shade
- Example: `background: linear-gradient(rgba(0,0,0,0.25), rgba(0,0,0,0.25)) var(--primary-color);`

### Theme Persistence Pattern

**localStorage key structure:**
```javascript
// Simple string value
localStorage.setItem('beerbot-theme', 'lager');

// On page load (before React mount)
const savedTheme = localStorage.getItem('beerbot-theme') || 'light';
document.documentElement.setAttribute('data-theme', savedTheme);
```

**React implementation (Context API + localStorage):**
1. Create `ThemeContext` with provider
2. Custom hook `useTheme()` returns current theme and setter
3. `useEffect` syncs to localStorage on theme change
4. Script in `<head>` prevents FOUC

### Quarterly Time Filter Pattern

**Preset-first approach:**
```
[Current Quarter] [Last Quarter] [Same Quarter Last Year] | Custom Range
```

**Tab navigation for quarters:**
```
Year: [< 2025] [2026] [>]
     [Q1] [Q2] [Q3] [Q4]
```

**URL state sync:**
- `/leaderboard?period=Q2-2026`
- Enables shareable links
- Preserves state on refresh
- Supports browser back/forward

## Accessibility Requirements

### Theme Toggle Accessibility

**WCAG 2.2 Level AA compliance:**
- Color contrast: 4.5:1 for text, 3:1 for UI components
- Keyboard navigation: Tab to focus, Enter/Space to activate
- Screen reader support: `aria-pressed="true"` for toggle state
- Focus indicators: Clear visual outline on keyboard focus
- No color-only indicators: Use text labels + icons

**Recommended ARIA structure:**
```html
<button
  aria-pressed="false"
  aria-label="Switch to dark theme"
  class="theme-toggle"
>
  <span aria-hidden="true">🌙</span>
  Dark
</button>
```

### Time Filter Accessibility

**Keyboard navigation:**
- Arrow keys to navigate quarters
- Tab to move between year/quarter controls
- Enter/Space to select
- Escape to close custom date picker

**Screen reader announcements:**
- Announce current selection: "Q2 2026 selected, April through June"
- Announce changes: "Now viewing Q1 2025"
- Clear labels: "Select quarter" not just icon buttons

## MVP Recommendation

For MVP theming and quarterly navigation, prioritize:

### Phase 1: Table Stakes (Must Have)
1. Light/dark theme toggle with localStorage persistence
2. FOUC prevention script
3. System preference detection
4. Quarterly tab navigation (Q1/Q2/Q3/Q4 for current year)
5. Current quarter as default view
6. Accessible theme toggle (WCAG 2.2 AA)

### Phase 2: Polish (Should Have)
7. One branded theme (e.g., "Lager" golden theme)
8. Year selector for historical quarters
9. Smooth theme transitions (CSS)
10. URL state sync for shareable links
11. "Current Quarter" / "Last Quarter" preset shortcuts

### Phase 3: Differentiators (Nice to Have)
12. Additional branded themes (Stout, IPA)
13. Quarterly color coding (seasonal colors)
14. Theme preview on hover
15. Year-over-year comparison view
16. Theme-aware charts/visualizations

## Defer to Post-MVP

**These can wait:**
- Seasonal theme auto-switching (low value, adds complexity)
- More than 3 branded themes (maintenance burden)
- Hourly/daily granularity (not needed for quarterly view)
- Advanced comparison features (side-by-side multi-year)
- Theme customization UI (let users pick custom colors)

**Rationale for deferral:**
- Branded theming is nice-to-have; light/dark is expected
- Quarterly navigation is simpler than full date range filtering
- Get core functionality working before adding polish
- Can iterate based on user feedback post-launch

## Confidence Assessment

| Category | Confidence | Source Quality |
|----------|-----------|----------------|
| Theme switching patterns | HIGH | Multiple authoritative sources (CSS-Tricks, Smashing Magazine, MDN) |
| Branded color implementation | HIGH | CSS custom properties well-documented |
| Theme persistence | HIGH | localStorage patterns standard in React |
| WCAG accessibility | HIGH | Official WCAG 2.2 documentation verified |
| Quarterly navigation UX | MEDIUM | Dashboard patterns verified; quarterly-specific examples limited |
| Time filter anti-patterns | MEDIUM | General dashboard guidance; need to adapt for quarterly context |
| Year-over-year comparison | LOW | Limited sources on implementation patterns; may need phase-specific research |

## Open Questions for Phase-Specific Research

When implementing quarterly navigation:
- What's the best UX for "Same Quarter Last Year" comparison?
- Should we allow custom date ranges in addition to quarterly presets?
- How to handle edge cases (current quarter not finished, missing historical data)?
- What loading states feel right when switching between quarters?

When implementing multiple branded themes:
- How many themes before it becomes decision paralysis? (Research suggests 3-5 max)
- Should theme names be beer-related or descriptive? ("Lager" vs "Golden")
- Do users want to set different themes per page or site-wide only?

## Sources

### Theme Switching & CSS Custom Properties
- [Theming and Theme Switching with React and styled-components | CSS-Tricks](https://css-tricks.com/theming-and-theme-switching-with-react-and-styled-components/)
- [React & CSS in 2026: Best Styling Approaches Compared | Medium](https://medium.com/@imranmsa93/react-css-in-2026-best-styling-approaches-compared-d5e99a771753)
- [CSS Custom Properties and Theming | CSS-Tricks](https://css-tricks.com/css-custom-properties-theming/)
- [How To Configure Application Color Schemes With CSS Custom Properties | Smashing Magazine](https://www.smashingmagazine.com/2020/08/application-color-schemes-css-custom-properties/)
- [Easy Dark Mode (and Multiple Color Themes!) in React | CSS-Tricks](https://css-tricks.com/easy-dark-mode-and-multiple-color-themes-in-react/)

### Theme Persistence & React Implementation
- [Mastering State Persistence with Local Storage in React | Medium](https://medium.com/@roman_j/mastering-state-persistence-with-local-storage-in-react-a-complete-guide-1cf3f56ab15c)
- [How to Use React for State Persistence | UXPin](https://www.uxpin.com/studio/blog/how-to-use-react-for-state-persistence/)
- [Learn Local Storage in React: Create a Light and Dark Theme Switcher Application](https://selftaughttxg.com/2023/05-23/learn-local-storage-in-react-create-a-light-and-dark-theme-switcher-application/)

### Dashboard Time Filtering & Navigation
- [From Data To Decisions: UX Strategies For Real-Time Dashboards | Smashing Magazine](https://www.smashingmagazine.com/2025/09/ux-strategies-real-time-dashboards/)
- [Filter UX Design Patterns & Best Practices - Pencil & Paper](https://www.pencilandpaper.io/articles/ux-pattern-analysis-enterprise-filtering)
- [React Date Range Picker component - MUI X](https://mui.com/x/react-date-pickers/date-range-picker/)
- [Date picker UI design best practices | Cieden](https://cieden.com/book/atoms/date-picker/date-picker-ui)

### Accessibility & WCAG Guidelines
- [A Practical Guide to Developing an Accessible Toggle Button Component in React | Medium](https://medium.com/@natalia.sokolova.ca/a-practical-guide-to-developing-an-accessible-toggle-button-component-in-react-3d3638c2f135)
- [2026 WCAG & ADA Website Compliance Requirements & Standards](https://www.accessibility.works/blog/wcag-ada-website-compliance-standards-requirements/)
- [Web Content Accessibility Guidelines (WCAG) 2.1 | W3C](https://www.w3.org/TR/WCAG21/)

### UX Design Trends & Anti-Patterns
- [12 UI/UX Design Trends That Will Dominate 2026](https://www.index.dev/blog/ui-ux-design-trends)
- [Toggle UX: Tips on Getting it Right | Eleken](https://www.eleken.co/blog-posts/toggle-ux)
- [Dashboard Design UX Patterns Best Practices - Pencil & Paper](https://www.pencilandpaper.io/articles/ux-pattern-analysis-data-dashboards)

### Leaderboard Time Filtering
- [Best Practices for Designing Leaderboards - 2V Modules](https://www.sportfitnessapps.com/blog/best-practices-for-designing-leaderboards)
- [Leaderboards Best Practices - Heroic Labs Documentation](https://heroiclabs.com/docs/nakama/concepts/leaderboards/best-practices/)
- [System Design of a Time Based Leaderboard | Medium](https://medium.com/@adenababanla/system-design-of-a-time-based-leaderboard-39410ed1264e)
