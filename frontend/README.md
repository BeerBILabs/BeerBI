# BeerBot Frontend

Next.js dashboard for BeerBot analytics and team appreciation insights.

## Tech Stack

- **Framework**: Next.js 16 with App Router (`output: 'standalone'`)
- **Language**: TypeScript (strict mode; type-checked with TypeScript 7)
- **Styling**: Tailwind CSS 4.x
- **Runtime**: Bun (recommended) or Node.js 18+
- **Deployment**: Docker

## Quick Start

See [root README](../README.md) for full setup instructions.

## Environment Variables

| Variable                   | Required | Default                 | Description              |
|----------------------------|----------|-------------------------|--------------------------|
| `BACKEND_BASE` | ✅ | `http://localhost:8080` | Backend URL, read at runtime by the Next.js server (proxy and `/api/leaderboard`); never sent to the browser |
| `API_TOKEN` | ✅ | - | Bearer token for the backend API; server-side only, the browser never sees it |

## Data flow

The browser only talks to Next.js route handlers, which call the backend at `BACKEND_BASE` with `API_TOKEN`:

- `/api/proxy/<path>` passes requests through to the backend `/api/<path>` (streams the body, forwards caching headers, never forwards a browser `Authorization` header).
- `/api/leaderboard?start=&end=[&prev_start=&prev_end=]` returns `{ current, previous }`: one request per leaderboard, the previous quarter (for rank changes) is fetched in parallel on the server. It passes on the backend caching as `Cache-Control: private, max-age=<shorter max-age>` plus an `ETag` built from the backend ETags (304 on a match).

User names are taken from the stats responses when the backend includes them (`real_name`/`profile_image` per top-user row, `giver_name`/`giver_image`/`recipient_name`/`recipient_image` per pair; a name without an image means no avatar); otherwise they are resolved through `/api/proxy/users?ids=` and cached in localStorage for 7 days.

## Font

Inter is self-hosted (`next/font/local` with the file from `@fontsource-variable/inter`), so the build does not download fonts.

## Bundle size

```bash
cd project
bun run analyze   # ANALYZE=true next build --webpack, reports in .next/analyze/*.html
```

The chart components (Recharts, about 400 KB) are loaded on demand on `/analytics`; the date range picker uses native date inputs.

## Theme

Light/dark mode is stored in the `theme` cookie (`light` or `dark`). It is set in the browser only (`components/ThemeProvider.tsx`), so the first visit causes no extra request; only the inline script in `app/layout.tsx` reads it (before the first paint) to avoid a flash of the wrong theme. The server never reads it, so `/` and `/analytics` are prerendered as static pages. Colours always come from the HSL variables in `app/globals.css`.

## Tests

```bash
cd project
bun install --frozen-lockfile
bun run test            # Vitest (TZ is forced to Europe/Berlin)
bun run test:coverage   # with coverage and threshold
bun run typecheck && bun run lint   # ESLint 10, warnings fail the run
```
