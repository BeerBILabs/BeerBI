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
| `NEXT_PUBLIC_BACKEND_BASE` | ✅       | `http://localhost:8080` | Backend API base URL     |
| `NEXT_PUBLIC_API_TOKEN`    | ✅       | -                       | API authentication token |

## Bundle size

```bash
cd project
bun run analyze   # ANALYZE=true next build --webpack, reports in .next/analyze/*.html
```

The chart components (Recharts, about 400 KB) are loaded on demand on `/analytics`; the date range picker uses native date inputs.

## Theme

Light/dark mode is stored in the `theme` cookie (`light` or `dark`). It is set in the browser only (`components/ThemeProvider.tsx`), so the first visit causes no extra request; `app/layout.tsx` and its inline script read it to avoid a flash of the wrong theme. Colours always come from the HSL variables in `app/globals.css`.

## Tests

```bash
cd project
bun install --frozen-lockfile
bun run test            # Vitest (TZ is forced to Europe/Berlin)
bun run test:coverage   # with coverage and threshold
bun run typecheck && bun run lint
```
