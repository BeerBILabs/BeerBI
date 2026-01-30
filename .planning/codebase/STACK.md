# Technology Stack

**Analysis Date:** 2026-01-30

## Languages

**Primary:**
- TypeScript 5.9.3 - All application code
- JavaScript (JSX/TSX) - Component and configuration files

**Secondary:**
- CSS - Styling via Tailwind CSS

## Runtime

**Environment:**
- Node.js (via Bun runtime)

**Package Manager:**
- Bun 1.3.0+
- Lockfile: `bun.lock` (present)

## Frameworks

**Core:**
- Next.js 16.1.6 - React meta-framework with server-side rendering and API routes
- React 19.2.4 - UI library and component framework
- React DOM 19.2.4 - DOM rendering

**UI & Styling:**
- Tailwind CSS 4.1.18 - Utility-first CSS framework
- @tailwindcss/postcss 4.1.18 - PostCSS plugin for Tailwind
- Tailwind Animate CSS 1.4.0 - Animation utilities
- Lucide React 0.563.0 - Icon library

**Utilities:**
- next-themes 0.4.6 - Dark mode and theme management
- react-datepicker 9.1.0 - Date range picker component
- Next.js Font - Built-in Google Fonts loader (Inter font)

**Build/Dev:**
- TypeScript 5.9.3 - Type checking
- ESLint 9.39.2 - Code linting
- eslint-config-next 16.1.6 - Next.js ESLint configuration
- PostCSS 8.5.6 - CSS processing
- Autoprefixer 10.4.23 - Vendor prefixes for CSS

## Key Dependencies

**Critical:**
- next 16.1.6 - Framework core, routing, API handling, Turbopack dev server
- react 19.2.4 - Component framework and hooks
- typescript 5.9.3 - Type safety and development experience

**Infrastructure:**
- @types/react 19.2.10 - TypeScript definitions for React
- @types/node 25.1.0 - TypeScript definitions for Node.js

## Configuration

**Environment:**
- Environment variables configured via:
  - `NEXT_PUBLIC_BACKEND_BASE` - Backend API base URL (default: `http://localhost:8080`)
  - `API_TOKEN` - Authentication token for backend communication
  - `NODE_ENV` - Runtime environment (development/production)
- Secrets stored in `.env` and `.env.local` (not committed per `.gitignore`)

**Build:**
- `next.config.js` - Next.js configuration (React Strict Mode enabled)
- `tsconfig.json` - TypeScript compiler options
  - Target: ES2017
  - Module: ESNext
  - Strict mode: Partial (strictNullChecks enabled, strict: false)
  - Path alias: `@/*` maps to `./src/*`
- `tailwind.config.js` - Tailwind CSS theme and content configuration (dark mode via class)
- `postcss.config.mjs` - PostCSS plugins (Tailwind, Autoprefixer)
- `eslint.config.mjs` - ESLint configuration with Next.js core web vitals

## Platform Requirements

**Development:**
- Bun 1.3.0+ as runtime and package manager
- Node.js compatible environment
- Turbopack-enabled Next.js dev server

**Production:**
- Deployment target: Any Node.js/Bun compatible host
- Build output: Next.js standalone output (`.next/` directory)
- Static and dynamic rendering support

---

*Stack analysis: 2026-01-30*
