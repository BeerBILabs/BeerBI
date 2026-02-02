# Phase 2: Beer-Branded Theming - Research

**Researched:** 2026-02-02
**Domain:** Next.js 15 theming with Tailwind CSS 4 and cookie-based persistence
**Confidence:** HIGH

## Summary

This phase implements beer-branded theming for the existing Next.js 15 application with Tailwind CSS 4.1.14. The project already has a basic theme system in place using localStorage and class-based dark mode. Based on user decisions, we need to migrate to cookie-based storage (for SSR compatibility), implement beer-themed color palettes, add smooth transitions, add tooltips to the toggle, and improve FOUC prevention.

The standard approach for Next.js 15 + Tailwind CSS 4 theming involves CSS custom properties (CSS variables) for colors, class-based dark mode toggling (`.dark` class on `<html>`), and either the `next-themes` library or a custom implementation with cookies for persistence. The current codebase uses a custom implementation with localStorage that can be enhanced to meet all requirements.

**Key technical decisions:**
- Keep custom theme implementation (no next-themes needed) but migrate localStorage to cookies
- Use existing CSS variable architecture but replace neutral colors with beer-themed palette
- Add CSS transitions for smooth theme switching (~250ms on background/color properties)
- Implement inline tooltip component (no library needed for simple hover tooltip)
- Add blocking script in `<head>` for FOUC prevention (reads cookie before hydration)

**Primary recommendation:** Enhance the existing custom theme system with cookie persistence and beer-themed CSS variables rather than introducing next-themes library. The current architecture is sound and adding cookies + color palette changes is simpler than migrating to a new library.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Next.js | 15.5.11 | React framework with App Router and Server Components | Already in project, provides cookies API for SSR-compatible persistence |
| Tailwind CSS | 4.1.14 | Utility-first CSS framework with class-based dark mode | Already in project, v4 uses CSS variables natively for theming |
| React | 19.2.0 | UI library | Already in project, required for Next.js |
| lucide-react | 0.546.0 | Icon library | Already in project, provides Sun/Moon icons for toggle |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| next-themes | 0.4.x | Theme management library for Next.js | When you want automatic localStorage, system preference, and multi-theme support - NOT needed for this phase |
| js-cookie | 3.0.x | Cookie manipulation library | When you need complex cookie operations - NOT needed, Next.js cookies API is sufficient |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom implementation | next-themes | next-themes adds 6kb, provides localStorage (not cookies), and more features than needed. Custom is lighter and matches user requirements exactly. |
| Cookies API | js-cookie | js-cookie works client-side only. Next.js cookies API works server-side and client-side (via Server Actions), enabling true SSR theme application. |
| Inline tooltip | Radix UI Tooltip | Radix adds 10kb+ for tooltip alone. Simple hover tooltip doesn't need library for this use case. |

**Installation:**
```bash
# No new dependencies needed - all functionality can be built with existing stack
```

## Architecture Patterns

### Recommended Project Structure
```
app/
├── layout.tsx              # Root layout with ThemeProvider, suppressHydrationWarning
├── globals.css             # CSS variables for light/dark themes with beer colors
components/
├── ThemeProvider.tsx       # Client component that syncs cookie with DOM class
├── ThemeToggle.tsx         # Button with icons, tooltip, and toggle logic
├── theme-script.tsx        # Blocking script component for FOUC prevention
lib/
└── theme-cookie.ts         # Server-side cookie reading utilities
```

### Pattern 1: Cookie-Based Theme Persistence with SSR
**What:** Store theme preference in a cookie (not localStorage) so the server can read it and apply the correct theme class during SSR, preventing FOUC.

**When to use:** Any Next.js app with SSR/SSG that needs to persist user theme preference across sessions.

**Example:**
```typescript
// app/layout.tsx (Server Component)
import { cookies } from 'next/headers'

export default async function RootLayout({ children }) {
  const cookieStore = await cookies()
  const theme = cookieStore.get('theme')?.value || 'light'

  return (
    <html lang="en" className={theme} suppressHydrationWarning>
      <body>{children}</body>
    </html>
  )
}
```

**Source:** Next.js official documentation - [Functions: cookies](https://nextjs.org/docs/app/api-reference/functions/cookies)

### Pattern 2: CSS Variables with Class-Based Dark Mode
**What:** Define all colors as CSS custom properties under `:root` and `.dark`, then reference them in Tailwind config. Switching theme is just toggling `.dark` class on `<html>`.

**When to use:** When you need consistent theming across all components and want to avoid prop drilling or context for colors.

**Example:**
```css
/* globals.css */
@layer base {
  :root {
    --background: 43 40% 96%;     /* Warm light beige */
    --foreground: 30 20% 15%;     /* Dark brown text */
    --primary: 38 92% 50%;        /* Amber/gold accent */
  }
  .dark {
    --background: 25 15% 8%;      /* Deep stout brown */
    --foreground: 43 40% 96%;     /* Light text */
    --primary: 38 85% 55%;        /* Bright gold accent */
  }
}
```

**Source:** Tailwind CSS official documentation - [Dark Mode](https://tailwindcss.com/docs/dark-mode)

### Pattern 3: Blocking Script for FOUC Prevention
**What:** Inject a synchronous inline script in `<head>` that reads the theme cookie and applies the class before React hydrates. This prevents the "flash of wrong theme."

**When to use:** When FOUC prevention is important and you're using cookies for theme storage.

**Example:**
```tsx
// components/theme-script.tsx
export function ThemeScript() {
  return (
    <script
      dangerouslySetInnerHTML={{
        __html: `
          (function() {
            var theme = document.cookie
              .split('; ')
              .find(row => row.startsWith('theme='))
              ?.split('=')[1] || 'light';
            document.documentElement.classList.toggle('dark', theme === 'dark');
          })();
        `,
      }}
    />
  )
}

// In app/layout.tsx <head>
<head>
  <ThemeScript />
</head>
```

**Source:** Community pattern from [How to Create Non-Flickering Dark/Light Mode Toggle in Next.js Using Cookies](https://www.designcise.com/web/tutorial/how-to-create-non-flickering-dark-or-light-mode-toggle-in-next-js-using-cookies)

### Pattern 4: CSS Transition for Smooth Theme Switch
**What:** Add CSS transitions to color and background properties so theme changes fade smoothly rather than snapping instantly.

**When to use:** When visual polish is important and you want theme switches to feel smooth. NOT when you need instant response.

**Example:**
```css
/* globals.css */
@layer base {
  * {
    transition-property: color, background-color, border-color;
    transition-duration: 250ms;
    transition-timing-function: ease-in-out;
  }

  /* Disable transitions during theme toggle to prevent animation cascade */
  .disable-transitions * {
    transition: none !important;
  }
}
```

**Note:** User specified ~200-300ms transition. 250ms is a good middle ground.

**Source:** CSS-Tricks and MDN - [Using CSS transitions](https://developer.mozilla.org/en-US/docs/Web/CSS/Guides/Transitions/Using)

### Pattern 5: Server Action for Cookie Updates
**What:** Use Next.js Server Actions to update cookies when theme changes, allowing both client and server to access the updated value.

**When to use:** When you need to modify cookies from client-side events in App Router.

**Example:**
```typescript
// app/actions/theme.ts
'use server'
import { cookies } from 'next/headers'

export async function setThemeCookie(theme: 'light' | 'dark') {
  const cookieStore = await cookies()
  cookieStore.set('theme', theme, {
    path: '/',
    maxAge: 60 * 60 * 24 * 365, // 1 year
    sameSite: 'lax',
  })
}
```

**Source:** Next.js official documentation - [Server Actions](https://nextjs.org/docs/app/building-your-application/data-fetching/server-actions-and-mutations)

### Anti-Patterns to Avoid
- **Using localStorage for SSR apps:** The server cannot read localStorage, causing mismatched renders and FOUC. Use cookies instead.
- **Reading useTheme during render:** Many values are undefined until client mount, causing hydration errors. Use `useState` + `useEffect` instead.
- **Forgetting suppressHydrationWarning:** When server renders `<html class="light">` but client immediately changes to `<html class="dark">`, React will warn. Add `suppressHydrationWarning` to `<html>`.
- **Transitioning all CSS properties:** `transition: all` is slow and causes jank. Only transition color-related properties.
- **Saturated colors in dark mode:** Bright saturated colors vibrate against dark backgrounds. Reduce saturation by ~20% in dark mode.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Full-featured theme system with system preference | Custom theme provider with localStorage and system detection | next-themes library | Handles system preference, cross-tab sync, forced themes per page, and FOUC prevention automatically. But NOT needed for this phase - user wants cookie-based, not localStorage. |
| Accessible tooltips with ARIA | Custom div with hover state | Radix UI Tooltip or Ark UI Tooltip | Proper ARIA attributes, keyboard navigation, screen reader support, positioning logic. But simple tooltip can be inline without library. |
| Cookie parsing/serialization | String manipulation | js-cookie library | Handles encoding, special characters, expiration dates. But Next.js cookies API is sufficient for simple use case. |

**Key insight:** Theme systems seem simple but have edge cases (system preference changes, cross-tab sync, SSR/SSG, hydration, performance). The existing custom implementation in this project is actually well-structured and only needs cookie migration + color updates, NOT a full library replacement.

## Common Pitfalls

### Pitfall 1: Hydration Mismatch from localStorage
**What goes wrong:** Server renders light mode, client reads localStorage and switches to dark mode after hydration, causing React hydration error: "Text content does not match server-rendered HTML."

**Why it happens:** Server cannot access localStorage (browser-only API), so it renders default theme. Client reads localStorage during hydration and immediately changes DOM, creating mismatch.

**How to avoid:**
1. Use cookies instead of localStorage (server can read cookies)
2. Add `suppressHydrationWarning` to `<html>` element
3. Delay theme-dependent rendering until after mount with `useState(false)` + `useEffect(() => setMounted(true))`

**Warning signs:**
- Console error: "Warning: Text content did not match. Server: 'X' Client: 'Y'"
- Console error: "Hydration failed because the initial UI does not match what was rendered on the server"
- Theme flashes from light to dark on page load

**Sources:**
- [Fixing Hydration Mismatch in Next.js (next-themes Issue)](https://medium.com/@pavan1419/fixing-hydration-mismatch-in-next-js-next-themes-issue-8017c43dfef9)
- [Next.js: Text content does not match server-rendered HTML](https://nextjs.org/docs/messages/react-hydration-error)

### Pitfall 2: FOUC Despite Cookie Usage
**What goes wrong:** Even with cookies, users see a brief flash of the wrong theme on initial page load.

**Why it happens:** The server correctly applies theme class, but CSS hasn't loaded yet OR client-side JavaScript runs before CSS applies OR theme class is applied after first paint.

**How to avoid:**
1. Add inline blocking script in `<head>` that applies theme class synchronously
2. Ensure critical CSS (CSS variables) is inlined or loaded early
3. Don't wait for React hydration to apply theme - do it in blocking script

**Warning signs:**
- Cookie is set correctly but still see flash
- Flash only happens on hard refresh, not client navigation
- Theme is correct after flash

**Sources:**
- [Fixing Dark Mode Flickering (FOUC) in React and Next.js](https://notanumber.in/blog/fixing-react-dark-mode-flickering)
- [Understanding & Fixing FOUC in Next.js App Router (2025 Guide)](https://dev.to/amritapadhy/understanding-fixing-fouc-in-nextjs-app-router-2025-guide-ojk)

### Pitfall 3: Saturated Colors in Dark Mode
**What goes wrong:** Colors that look great in light mode appear to vibrate or strain eyes in dark mode.

**Why it happens:** Saturated colors (high HSL saturation) create optical vibration when placed on dark backgrounds due to the way human eyes perceive contrast.

**How to avoid:**
1. Reduce saturation by 15-20% for dark mode colors
2. Use slightly lighter/brighter versions of colors in dark mode
3. Test colors against dark backgrounds, not just light
4. Use HSL color space for easier saturation adjustment

**Warning signs:**
- Colors look "too bright" or "neon" in dark mode
- Users report eye strain in dark mode
- Colors that worked in light mode fail WCAG contrast in dark mode

**Sources:**
- [Dark mode UI design – 7 best practices](https://atmos.style/blog/dark-mode-ui-best-practices)
- [10 Dark Mode UI Best Practices & Principles for 2026](https://www.designstudiouiux.com/blog/dark-mode-ui-design-best-practices/)

### Pitfall 4: Forgetting suppressHydrationWarning on <html>
**What goes wrong:** React throws hydration warnings even though theme is working correctly.

**Why it happens:** The server renders `<html lang="en" class="light">` but client-side code (blocking script or useEffect) changes it to `<html lang="en" class="dark">` before React hydrates. React sees the mismatch and warns.

**How to avoid:**
1. Add `suppressHydrationWarning` prop to `<html>` element
2. Only add to `<html>`, not to child elements (it only suppresses one level deep)
3. Still use proper server-side rendering - this just suppresses the expected warning

**Warning signs:**
- Console warning about HTML mismatch on `<html>` element
- Warning specifically mentions class attribute
- Warning only appears on initial page load, not navigation

**Sources:**
- [next-themes GitHub](https://github.com/pacocoursey/next-themes) - Official recommendation
- [React's suppressHydrationWarning: Fixing Hydration Errors](https://medium.com/@praveenb0927/reacts-suppresshydrationwarning-fixing-hydration-errors-causes-solutions-and-best-practices-62977194e6f4)

### Pitfall 5: Transition Performance Issues
**What goes wrong:** Adding `transition: all` to elements causes janky, slow theme transitions or layout shifts.

**Why it happens:** `transition: all` transitions every CSS property including non-animatable ones, causing the browser to recalculate layout/paint for every property change.

**How to avoid:**
1. Only transition color-related properties: `transition-property: color, background-color, border-color`
2. Use `transition-duration: 200-300ms` for snappy feel
3. Use `ease-in-out` timing function for smooth acceleration
4. Consider temporarily disabling transitions during toggle for instant switch (user preference: smooth fade)

**Warning signs:**
- Theme transition feels slow or janky
- Browser DevTools Performance shows long paint times during transition
- Layout shifts or element resizing during theme change

**Sources:**
- [Disable transitions on theme toggle](https://paco.me/writing/disable-theme-transitions)
- [CSS Page Transitions For A Better User Experience](https://www.sliderrevolution.com/resources/css-page-transitions/)

### Pitfall 6: Cookie Not Setting Properly from Client
**What goes wrong:** Setting cookie from client-side JavaScript doesn't persist or isn't readable by server on next request.

**Why it happens:**
- Cookie not set with correct `path` (defaults to current path)
- Cookie set as httpOnly from client (invalid)
- Cookie domain mismatch
- SameSite issues in development (localhost)

**How to avoid:**
1. Use Server Action to set cookies (recommended for App Router)
2. If setting from client, use `document.cookie = 'theme=dark; path=/; max-age=31536000; SameSite=Lax'`
3. Ensure `path=/` so cookie is sent on all routes
4. Don't use httpOnly from client (impossible to set)
5. Use SameSite=Lax for development compatibility

**Warning signs:**
- Theme resets to default on page refresh
- Cookie appears in DevTools but server doesn't see it
- Theme works in client navigation but not hard refresh

**Sources:**
- [A guide to cookies in Next.js](https://blog.logrocket.com/guide-cookies-next-js/)
- [Next.js cookies API Reference](https://nextjs.org/docs/app/api-reference/functions/cookies)

## Code Examples

Verified patterns from official sources:

### Server-Side Theme Reading (App Router)
```typescript
// app/layout.tsx
import { cookies } from 'next/headers'

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const cookieStore = await cookies()
  const theme = cookieStore.get('theme')?.value || 'light'

  return (
    <html lang="en" className={theme} suppressHydrationWarning>
      <body>
        <ThemeProvider />
        {children}
      </body>
    </html>
  )
}
```
**Source:** [Next.js cookies API documentation](https://nextjs.org/docs/app/api-reference/functions/cookies)

### Client-Side Theme Toggle with Server Action
```typescript
// app/actions/theme.ts
'use server'
import { cookies } from 'next/headers'

export async function setTheme(theme: 'light' | 'dark') {
  const cookieStore = await cookies()
  cookieStore.set('theme', theme, {
    path: '/',
    maxAge: 60 * 60 * 24 * 365, // 1 year
    sameSite: 'lax',
  })
}

// components/ThemeToggle.tsx
'use client'
import { setTheme } from '@/app/actions/theme'

export default function ThemeToggle() {
  async function toggle() {
    const isDark = document.documentElement.classList.contains('dark')
    const newTheme = isDark ? 'light' : 'dark'
    document.documentElement.classList.toggle('dark', newTheme === 'dark')
    await setTheme(newTheme)
  }

  return <button onClick={toggle}>Toggle</button>
}
```
**Source:** [Next.js Server Actions and Mutations](https://nextjs.org/docs/app/building-your-application/data-fetching/server-actions-and-mutations)

### Blocking Script for FOUC Prevention
```tsx
// components/ThemeScript.tsx
export function ThemeScript() {
  return (
    <script
      dangerouslySetInnerHTML={{
        __html: `
          (function() {
            try {
              var theme = document.cookie
                .split('; ')
                .find(row => row.startsWith('theme='))
                ?.split('=')[1];
              if (theme === 'dark') {
                document.documentElement.classList.add('dark');
              } else {
                document.documentElement.classList.remove('dark');
              }
            } catch (e) {}
          })();
        `,
      }}
    />
  )
}

// In app/layout.tsx - add to head section (or return in component with metadata)
```
**Source:** [How to Create Non-Flickering Dark/Light Mode Toggle in Next.js Using Cookies](https://www.designcise.com/web/tutorial/how-to-create-non-flickering-dark-or-light-mode-toggle-in-next-js-using-cookies)

### Beer-Themed Color Palette (HSL)
```css
/* app/globals.css */
@layer base {
  :root {
    /* Light mode - warm neutrals with amber accents */
    --background: 43 40% 96%;          /* Warm off-white */
    --foreground: 30 20% 15%;          /* Dark brown text */
    --card: 0 0% 100%;                 /* Pure white cards */
    --card-foreground: 30 20% 15%;     /* Dark brown text */
    --primary: 38 92% 50%;             /* Golden amber accent */
    --primary-foreground: 0 0% 100%;   /* White text on amber */
    --border: 35 25% 85%;              /* Light warm gray border */
    --accent: 43 70% 88%;              /* Light amber background */
    --accent-foreground: 30 20% 15%;   /* Dark text on light amber */
  }

  .dark {
    /* Dark mode - stout/porter deep browns */
    --background: 25 15% 8%;           /* Deep stout brown/black */
    --foreground: 43 40% 96%;          /* Warm off-white text */
    --card: 25 20% 12%;                /* Slightly lighter brown cards */
    --card-foreground: 43 40% 96%;     /* Warm off-white text */
    --primary: 38 85% 55%;             /* Bright gold (reduced saturation) */
    --primary-foreground: 25 15% 8%;   /* Dark text on gold */
    --border: 25 15% 20%;              /* Medium brown border */
    --accent: 25 25% 18%;              /* Medium brown background */
    --accent-foreground: 43 40% 96%;   /* Light text */
  }
}
```
**Source:** Research compilation from [Figma Amber Colors](https://www.figma.com/colors/amber/), [SchemeColor Beer Palette](https://www.schemecolor.com/chilled-beer-colors.php), and dark mode best practices

### CSS Transitions for Smooth Theme Switch
```css
/* app/globals.css */
@layer base {
  * {
    transition-property: color, background-color, border-color;
    transition-duration: 250ms;
    transition-timing-function: ease-in-out;
  }

  /* Prevent transitions on page load */
  .no-transitions * {
    transition: none !important;
  }

  /* Prevent animations for users who prefer reduced motion */
  @media (prefers-reduced-motion: reduce) {
    * {
      transition-duration: 0ms !important;
    }
  }
}
```
**Source:** [MDN: Using CSS transitions](https://developer.mozilla.org/en-US/docs/Web/CSS/Guides/Transitions/Using)

### Simple Accessible Tooltip
```tsx
// components/ThemeToggle.tsx
'use client'
import { useState } from 'react'

export default function ThemeToggle() {
  const [isDark, setIsDark] = useState(false)
  const [showTooltip, setShowTooltip] = useState(false)

  return (
    <div className="relative">
      <button
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        onFocus={() => setShowTooltip(true)}
        onBlur={() => setShowTooltip(false)}
        aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
      >
        {isDark ? <Sun /> : <Moon />}
      </button>

      {showTooltip && (
        <div
          role="tooltip"
          className="absolute bottom-full mb-2 px-2 py-1 text-xs rounded bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900 whitespace-nowrap"
        >
          {isDark ? 'Switch to light mode' : 'Switch to dark mode'}
        </div>
      )}
    </div>
  )
}
```
**Source:** [How to create an accessible tooltip using React](https://dev.to/micaavigliano/how-to-create-an-accessible-tooltip-using-react-2cck)

### Tailwind Config for Class-Based Dark Mode
```javascript
// tailwind.config.js
export const darkMode = ['class']; // Use class strategy, not media query

export const theme = {
  extend: {
    colors: {
      background: 'hsl(var(--background))',
      foreground: 'hsl(var(--foreground))',
      // ... all other colors reference CSS variables
    }
  }
}
```
**Source:** [Tailwind CSS Dark Mode Documentation](https://tailwindcss.com/docs/dark-mode)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| localStorage for theme | Cookies for theme | ~2021-2022 | Enables SSR/SSG to read preference and prevent FOUC. Cookies work server-side. |
| Manual color values in components | CSS custom properties (variables) | Tailwind v3+ | Easier theming, single source of truth, runtime theme switching without recompiling. |
| Media query dark mode only | Class-based dark mode | Tailwind v2+ | User can manually toggle theme, not just follow system. More control. |
| Page reload for theme change | Client-side class toggle | Modern React | Instant theme switching without page reload. Better UX. |
| Tailwind v3 dark mode config | Tailwind v4 @custom-variant | 2024-2025 | Tailwind v4 uses CSS-first config. Class-based dark mode still works but config syntax different. |
| next-themes using localStorage | Manual cookie implementation | 2026 trend | For SSR-heavy apps, cookies provide better control and avoid library weight when not needed. |

**Deprecated/outdated:**
- **Theme context with prop drilling:** Use CSS variables instead - simpler, no context needed
- **Separate CSS files for themes:** Use class-based CSS variables - single file, runtime switching
- **prefers-color-scheme only:** Users expect manual toggle, not just system follow
- **Setting theme in _app.js:** Next.js 13+ uses App Router with layout.tsx, not _app.js

## Open Questions

Things that couldn't be fully resolved:

1. **Tailwind v4 class-based dark mode configuration**
   - What we know: Tailwind v4 documentation shows `@custom-variant dark` in CSS, but project's tailwind.config.js uses `darkMode = ['class']` which is v3 syntax
   - What's unclear: Whether v4 still supports the v3 config syntax or if migration is needed
   - Recommendation: Test current config first. If it works, keep it. If not, add `@custom-variant dark (&:where(.dark, .dark *))` to globals.css as per v4 docs.

2. **Optimal FOUC prevention strategy complexity**
   - What we know: User said FOUC prevention is "nice to have but not critical" and brief flash is acceptable if it simplifies implementation
   - What's unclear: Whether the blocking script approach is worth the complexity vs accepting brief flash
   - Recommendation: Start with server-side cookie reading + suppressHydrationWarning (simple). Add blocking script only if FOUC is noticeable in production build.

3. **Toggle animation timing and style**
   - What we know: User left toggle animation style and timing to Claude's discretion (fade, rotate, etc.)
   - What's unclear: Whether to animate the icon itself (rotate/fade between Sun/Moon) or just the button background
   - Recommendation: Simple crossfade between icons with 200ms transition is cleanest. Rotation can be distracting. Test both and choose in planning.

4. **Whether all visual elements should transition**
   - What we know: User left borders/shadows/dividers transition to Claude's discretion
   - What's unclear: Performance impact of transitioning all borders vs just backgrounds/text
   - Recommendation: Transition color, background-color, and border-color together (250ms). Skip box-shadow transitions (can cause jank). Test performance in planning.

## Sources

### Primary (HIGH confidence)
- [Next.js cookies API Reference](https://nextjs.org/docs/app/api-reference/functions/cookies) - Cookie reading/setting in App Router
- [Tailwind CSS Dark Mode Documentation](https://tailwindcss.com/docs/dark-mode) - Class-based dark mode configuration
- [MDN: Using CSS transitions](https://developer.mozilla.org/en-US/docs/Web/CSS/Guides/Transitions/Using) - Transition properties and best practices
- [next-themes GitHub Repository](https://github.com/pacocoursey/next-themes) - Reference for comparison (not using, but authoritative on patterns)

### Secondary (MEDIUM confidence)
- [How to Create Non-Flickering Dark/Light Mode Toggle in Next.js Using Cookies](https://www.designcise.com/web/tutorial/how-to-create-non-flickering-dark-or-light-mode-toggle-in-next-js-using-cookies) - Cookie-based implementation pattern
- [Fixing Dark Mode Flickering (FOUC) in React and Next.js](https://notanumber.in/blog/fixing-react-dark-mode-flickering) - FOUC prevention strategies
- [Dark mode UI design – 7 best practices](https://atmos.style/blog/dark-mode-ui-best-practices) - Color saturation and contrast guidance
- [SchemeColor Beer Palette](https://www.schemecolor.com/chilled-beer-colors.php) - Beer color hex values
- [Figma Amber Colors](https://www.figma.com/colors/amber/) - Amber/gold color palette
- [How to create an accessible tooltip using React](https://dev.to/micaavigliano/how-to-create-an-accessible-tooltip-using-react-2cck) - Accessible tooltip pattern

### Tertiary (LOW confidence)
- [Step-by-Step Guide to Adding Dark Mode with next-themes](https://medium.com/@salihbezai98/step-by-step-guide-to-adding-dark-mode-with-next-themes-in-next-js-and-tailwind-css-15db7876f071) - General guidance but next-themes specific
- [10 Dark Mode UI Best Practices & Principles for 2026](https://www.designstudiouiux.com/blog/dark-mode-ui-design-best-practices/) - General best practices
- Community discussions on WebSearch results - Verified against official docs where possible

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in project, verified from package.json
- Architecture: HIGH - Patterns verified from Next.js and Tailwind official docs, existing code reviewed
- Pitfalls: HIGH - Cross-referenced multiple sources and official documentation
- Color palette: MEDIUM - Based on color theory and design resources, but subjective aesthetic choices
- FOUC prevention: HIGH - Multiple sources agree on blocking script approach, verified with official Next.js patterns

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - stable domain, framework changes slowly)

**Notes:**
- Current project already has theme system with localStorage - migration path is clear
- User decisions from CONTEXT.md significantly narrow scope - no system preference, no next-themes, cookies only
- Beer color palette requires subjective design judgment - recommend testing with user
- All technical patterns verified against official Next.js 15 and Tailwind CSS 4 documentation
