# Domain Pitfalls: Next.js 16.x Upgrade + Theming

**Domain:** Enhancement milestone - dependency upgrades and theming
**Researched:** 2026-01-31
**Confidence:** HIGH (verified with official docs and recent 2026 sources)

## Executive Summary

This document catalogs pitfalls specific to upgrading BeerBot's stack (Next.js 16.1.6 → latest, Go backend, Docker containers) and adding theming with next-themes 0.4.6 and Tailwind CSS. Each pitfall includes warning signs, prevention strategies, and phase recommendations.

**Critical risk areas:**
1. Next.js 16 async breaking changes (CRITICAL - affects all routes)
2. Hydration mismatches with next-themes (HIGH - breaks SSR)
3. Docker cache invalidation (MEDIUM - slows CI/CD)
4. Tailwind v4 migration (MEDIUM - config system redesign)

---

## Critical Pitfalls

Mistakes that cause rewrites or major breakage.

### Pitfall 1: Synchronous Request API Usage (Next.js 16)

**What goes wrong:** Next.js 16 completely removed synchronous access to `params`, `searchParams`, `cookies()`, `headers()`, and `draftMode()`. Code using these synchronously will fail at build time.

**Why it happens:** Next.js 15 introduced async versions with temporary backward compatibility. Developers assume the old synchronous pattern still works in v16.

**Consequences:**
- Build failures with `TypeError: params is a Promise`
- Runtime errors if builds somehow succeed
- All existing routes break simultaneously
- Image generation functions fail
- Sitemap generation breaks

**Warning signs:**
```tsx
// RED FLAG - will break in Next.js 16
export default function Page({ params }) {
  const slug = params.slug  // Direct access
}

// RED FLAG - synchronous cookie/header access
export default function Component() {
  const cookieStore = cookies()  // Missing await
}
```

**Prevention:**
1. **Run codemod FIRST**: `npx @next/codemod@canary upgrade latest`
2. **Use type safety**: Run `npx next typegen` for automatic async parameter types
3. **Audit all route files**: Search for `params`, `searchParams`, `cookies()`, `headers()` usage
4. **Update pattern**:
```tsx
// CORRECT - Next.js 16
export default async function Page(props) {
  const params = await props.params
  const searchParams = await props.searchParams
}

// CORRECT - async utilities
export default async function Component() {
  const cookieStore = await cookies()
  const headersList = await headers()
}
```

**Detection:**
- Build-time: TypeScript errors if using `npx next typegen`
- Build-time: Next.js build fails with Promise-related errors
- Code search: `grep -r "{ params }" app/` finds synchronous destructuring
- Code search: `grep -r "cookies()" app/ | grep -v "await"` finds non-awaited calls

**Phase recommendation:** Address in Phase 1 (Dependencies Upgrade) - blocks all other work if not fixed.

**Sources:**
- [Next.js 16 Upgrade Guide](https://nextjs.org/docs/app/guides/upgrading/version-16)
- Confidence: HIGH (official documentation)

---

### Pitfall 2: Turbopack Default Breaking Webpack Configs

**What goes wrong:** Next.js 16 uses Turbopack as the default bundler. Custom webpack configurations cause builds to fail with "webpack config detected" errors.

**Why it happens:** Teams assume backward compatibility with webpack configs from previous versions.

**Consequences:**
- `next build` fails immediately
- Custom loaders don't work
- Build process completely broken
- CI/CD pipeline failures

**Warning signs:**
- `webpack:` configuration exists in `next.config.js`
- Custom loaders or plugins defined
- Build output shows "webpack config detected, build failed"

**Prevention:**
1. **Check for webpack config**: Review `next.config.js` for `webpack:` key
2. **Choose migration path**:
   - **Option A**: Migrate to Turbopack equivalents (preferred)
   - **Option B**: Opt-out with `--webpack` flag in package.json
3. **Update scripts** if staying on webpack:
```json
{
  "scripts": {
    "dev": "next dev --webpack",
    "build": "next build --webpack"
  }
}
```
4. **Migrate config location**: Move Turbopack config from `experimental.turbopack` to top-level `turbopack` in next.config.js

**Detection:**
- Pre-upgrade: Search for `webpack:` in next.config.js
- Build-time: Build fails with explicit webpack error
- Warning in dev console about deprecated config location

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - decide before upgrading Next.js version.

**Sources:**
- [Next.js 16 Upgrade Guide](https://nextjs.org/docs/app/guides/upgrading/version-16)
- Confidence: HIGH (official documentation)

---

### Pitfall 3: Hydration Mismatch with next-themes

**What goes wrong:** Theme state is undefined during SSR, causing server HTML to mismatch client HTML. React throws hydration errors and components flash/re-render.

**Why it happens:**
- Theme preference stored in localStorage (client-only)
- Server renders with no theme knowledge
- Client mounts and applies theme, creating HTML mismatch

**Consequences:**
- Console errors: "Hydration failed because the initial UI does not match..."
- Flash of unstyled content (FOUC)
- Flash of wrong theme (light theme briefly visible before dark mode applies)
- Poor user experience, especially on slow connections
- React disables concurrent features when hydration errors occur

**Warning signs:**
```tsx
// RED FLAG - theme usage without mounting check
export default function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  return <button onClick={() => setTheme('dark')}>{theme}</button>
  // 'theme' is undefined on server
}

// RED FLAG - ThemeProvider at root without suppressHydrationWarning
export default function RootLayout({ children }) {
  return (
    <html lang="en">  {/* Missing suppressHydrationWarning */}
      <body>
        <ThemeProvider attribute="class">
          {children}
        </ThemeProvider>
      </body>
    </html>
  )
}
```

**Prevention:**
1. **Add suppressHydrationWarning to html tag**:
```tsx
<html lang="en" suppressHydrationWarning>
```
This is necessary because next-themes updates the `<html>` element with the theme class.

2. **Delay theme-dependent rendering until mounted**:
```tsx
'use client'
import { useEffect, useState } from 'react'
import { useTheme } from 'next-themes'

export default function ThemeToggle() {
  const [mounted, setMounted] = useState(false)
  const { theme, setTheme } = useTheme()

  useEffect(() => setMounted(true), [])

  if (!mounted) {
    return <div className="w-10 h-10" /> // Placeholder
  }

  return <button onClick={() => setTheme('dark')}>{theme}</button>
}
```

3. **Use CSS-based approach for theme-aware content**:
```tsx
// Show both versions, hide with CSS
<div className="dark:hidden">Light mode content</div>
<div className="hidden dark:block">Dark mode content</div>
```

4. **Configure ThemeProvider correctly**:
```tsx
<ThemeProvider
  attribute="class"  // Use class-based theming for Tailwind
  defaultTheme="system"  // Respect system preference
  enableSystem  // Enable system theme detection
>
```

**Detection:**
- Dev console: "Warning: Prop `className` did not match"
- Dev console: "Hydration failed..."
- Visual: Flash of wrong theme on page load
- React DevTools: Hydration error markers

**Phase recommendation:** Phase 2 (Theming Implementation) - fundamental to dark mode implementation.

**Sources:**
- [next-themes Hydration Fix](https://medium.com/@pavan1419/fixing-hydration-mismatch-in-next-js-next-themes-issue-8017c43dfef9)
- [next-themes GitHub](https://github.com/pacocoursey/next-themes)
- Confidence: HIGH (official library docs + verified community patterns)

---

### Pitfall 4: Tailwind v4 Theme Configuration Breaking Changes

**What goes wrong:** Tailwind CSS v4 moved from JavaScript configuration (`tailwind.config.js`) to CSS-first configuration with `@theme` directive. Existing color palettes and theme variables don't work.

**Why it happens:**
- Developers upgrade Tailwind without reading migration guide
- Assumption that config system is backward compatible
- Copy-paste patterns from v3 tutorials

**Consequences:**
- Custom colors don't generate utility classes
- Theme variables undefined
- `tailwind.config.js` ignored (silently in many cases)
- Build succeeds but custom styles missing
- Hard-to-debug "why isn't my color working?" issues

**Warning signs:**
```js
// RED FLAG - v3 pattern doesn't work in v4
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        primary: '#3b82f6',
        secondary: '#8b5cf6',
      }
    }
  }
}
```

**Prevention:**

1. **Use @theme directive in CSS**:
```css
/* app/globals.css */
@import 'tailwindcss';

@theme {
  --color-primary: #3b82f6;
  --color-secondary: #8b5cf6;
  --font-sans: Inter, sans-serif;
}
```

2. **Reset namespaces when overriding completely**:
```css
@theme {
  --color-*: initial;  /* Remove all default colors */
  --color-white: #fff;
  --color-black: #000;
  --color-brand-blue: #0066ff;
}
```

3. **Use inline for variable references**:
```css
/* WRONG - cascade issues */
@theme {
  --font-sans: var(--font-inter);
}

/* CORRECT */
@theme inline {
  --font-sans: var(--font-inter);
}
```

4. **Use OKLCH color format** for better perceptual uniformity:
```css
@theme {
  --color-brand-blue: oklch(0.55 0.25 250);  /* Modern, perceptual */
  --color-legacy-red: #ff0000;  /* Still works but less ideal */
}
```

5. **Run official migration tool**:
```bash
npx @tailwindcss/upgrade@next
```

**Detection:**
- Build-time: No errors (configuration silently ignored)
- Runtime: Custom color utilities don't exist (e.g., `bg-primary` doesn't work)
- Dev tools: Computed styles show default values instead of custom
- Search: `grep -r "tailwind.config" .` shows old config files

**Phase recommendation:** Phase 2 (Theming Implementation) - must migrate before adding custom palette.

**Note:** If staying on Tailwind v3, this pitfall doesn't apply. Verify Tailwind version before upgrading.

**Sources:**
- [Tailwind CSS v4 Migration Guide](https://medium.com/@mernstackdevbykevin/tailwind-css-v4-0-complete-migration-guide-breaking-changes-you-need-to-know-7f99944a9f95)
- [Tailwind CSS Theme Documentation](https://tailwindcss.com/docs/theme)
- Confidence: HIGH (official documentation verified)

---

## Moderate Pitfalls

Mistakes that cause delays or technical debt.

### Pitfall 5: React 19 Third-Party Library Incompatibility

**What goes wrong:** Next.js 16 uses React 19.2, which has breaking changes affecting third-party libraries. Libraries like Recoil, older MUI versions, and next-mdx-remote break completely.

**Why it happens:**
- JSX namespace moved in React 19
- Server Components conflict with Pages Router-era libraries
- Libraries haven't updated for React 19

**Consequences:**
- Build failures with cryptic TypeScript errors
- Runtime crashes in components using incompatible libraries
- MDX rendering breaks with App Router
- State management libraries stop working

**Warning signs:**
```tsx
// Libraries known to have issues:
import { atom } from 'recoil'  // Recoil needs updates
import { renderToString } from 'next-mdx-remote'  // Pages Router only
import { Button } from '@mui/material'  // Need v5.18.0+ for React 19
```

**Prevention:**

1. **Audit dependencies before upgrading**:
```bash
# Check for known incompatible libraries
npm list recoil next-mdx-remote @mui/material
```

2. **Update TypeScript types**:
```bash
npm install --save-dev @types/react@latest @types/react-dom@latest
```

3. **MUI minimum version**:
```bash
npm install @mui/material@^5.18.0  # Minimum for React 19
```

4. **MDX migration**: Replace `next-mdx-remote` with:
   - `@next/mdx` for App Router
   - Server Components-compatible alternatives

5. **Recoil alternatives**:
   - Migrate to Zustand (React 19 compatible)
   - Or use React Context + useReducer
   - Or wait for Recoil React 19 support

6. **Test incrementally**:
```bash
# Upgrade React first in isolation
npm install react@19 react-dom@19
npm run build  # See what breaks before Next.js upgrade
```

**Detection:**
- Build-time: TypeScript errors about JSX namespace
- Build-time: "Module not found" or version conflict errors
- Runtime: Component crashes with "Cannot read property" errors
- Test failures in components using problematic libraries

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - audit and update before Next.js upgrade.

**Sources:**
- [React 19 Compatibility Real-World Issues](https://medium.com/@quicksilversel/i-upgraded-three-apps-to-react-19-heres-what-broke-648087c7217b)
- [Next.js 16 Release Notes](https://nextjs.org/blog/next-16)
- Confidence: HIGH (verified real-world reports from Jan 2026)

---

### Pitfall 6: Docker Layer Cache Invalidation on Dependency Changes

**What goes wrong:** Copying source code before installing dependencies invalidates Docker cache on every code change, forcing full dependency reinstall even when go.mod/go.sum unchanged.

**Why it happens:** Incorrect layer ordering in Dockerfile - copying all files before running `go mod download`.

**Consequences:**
- 5-10x slower builds during development
- CI/CD pipeline timeouts
- Expensive re-downloads of all Go modules
- Developer frustration with slow iteration

**Warning signs:**
```dockerfile
# RED FLAG - wrong order
FROM golang:1.25-alpine
WORKDIR /app
COPY . .              # Copies everything including source
RUN go mod download   # Cache invalidated on ANY file change
RUN go build -o server
```

**Prevention:**

1. **Correct layer ordering**:
```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files FIRST
COPY go.mod go.sum ./
RUN go mod download   # Cached unless go.mod/go.sum changes

# THEN copy source code
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server
```

2. **Use multi-stage builds** for smaller final images:
```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server

# Runtime stage
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /server /server
ENTRYPOINT ["/server"]
```

3. **Add .dockerignore**:
```
# .dockerignore
.git
.env
*.md
node_modules
.next
```

4. **Use cache mounts** for even better performance (BuildKit):
```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
```

**Detection:**
- Slow builds: Check Docker build output for "downloading go modules" on every build
- CI logs: Look for repeated dependency downloads
- Timing: Compare build time for code-only changes vs dependency changes

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - fix when updating Go version in Dockerfile.

**Sources:**
- [Docker Build Cache Documentation](https://docs.docker.com/build/cache/)
- [Docker Multi-Stage Builds for Go](https://oneuptime.com/blog/post/2026-01-07-go-docker-multi-stage/view)
- Confidence: HIGH (official Docker docs + recent 2026 best practices)

---

### Pitfall 7: Go Major Version Upgrades Requiring Import Path Changes

**What goes wrong:** Upgrading a Go module dependency to v2+ requires changing import paths to include `/v2`, `/v3`, etc. Build fails with "module not found" if import paths not updated.

**Why it happens:** Go's semantic import versioning requires different import paths for major versions. Developers run `go get -u` expecting automatic upgrades.

**Consequences:**
- Build failures: "module example.com/pkg not found"
- Confusion about why dependency update didn't work
- Mixed versions in dependency tree (v1 and v2 both present)
- Subtle bugs from using wrong version

**Warning signs:**
```bash
# RED FLAG - upgrading to v2+ without import path changes
go get -u github.com/example/pkg  # Gets v1.x.x only

# Imports still reference v1
import "github.com/example/pkg"  // Should be /v2
```

**Prevention:**

1. **Check for major version updates before upgrading**:
```bash
# See available versions
go list -m -versions github.com/example/pkg

# Output: github.com/example/pkg v1.0.0 v1.1.0 v2.0.0 v2.1.0
```

2. **Update imports for v2+**:
```go
// Before (v1)
import "github.com/example/pkg"

// After (v2)
import "github.com/example/pkg/v2"
```

3. **Update go.mod explicitly**:
```bash
go get github.com/example/pkg/v2@latest
```

4. **Use go mod tidy to clean up**:
```bash
go mod tidy  # Removes old v1 entry if not used
```

5. **Search and replace imports**:
```bash
# Find all imports of package
grep -r "github.com/example/pkg" --include="*.go"

# Update with sed or manual find-replace
```

**Detection:**
- Build-time: "module not found" errors despite dependency in go.mod
- go.mod: Shows both v1 and v2 versions of same package
- Code: Import statements don't match go.mod version

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - audit go.mod for major version changes.

**Sources:**
- [Go Modules Major Version Handling](https://go.dev/doc/modules/major-version)
- [Go Module Version Numbers](https://go.dev/doc/modules/version-numbers)
- Confidence: HIGH (official Go documentation)

---

### Pitfall 8: Unnecessary Route Handlers in App Router

**What goes wrong:** Creating API Route Handlers (route.ts) when logic can execute directly in Server Components. This adds unnecessary network overhead and complexity.

**Why it happens:** Developers apply Pages Router patterns (API routes for server logic) to App Router without understanding Server Components.

**Consequences:**
- Extra network round-trip (component → route handler → database)
- Increased latency (100-300ms additional)
- More code to maintain
- Harder to understand data flow
- Defeats purpose of Server Components

**Warning signs:**
```tsx
// RED FLAG - Server Component fetching from own Route Handler
// app/posts/page.tsx
export default async function PostsPage() {
  const response = await fetch('/api/posts')  // Unnecessary hop
  const posts = await response.json()
  return <div>{posts.map(...)}</div>
}

// app/api/posts/route.ts
export async function GET() {
  const posts = await db.posts.findMany()
  return Response.json(posts)
}
```

**Prevention:**

1. **Fetch directly in Server Components**:
```tsx
// CORRECT - no Route Handler needed
// app/posts/page.tsx
export default async function PostsPage() {
  const posts = await db.posts.findMany()  // Direct database access
  return <div>{posts.map(...)}</div>
}
```

2. **Use Route Handlers only for**:
   - External webhooks
   - API endpoints for mobile apps
   - Third-party integrations
   - OAuth callbacks

3. **Use Server Actions for mutations**:
```tsx
// CORRECT - Server Action instead of Route Handler
// app/actions.ts
'use server'
export async function createPost(formData: FormData) {
  const title = formData.get('title')
  await db.posts.create({ data: { title } })
  revalidatePath('/posts')
}

// app/posts/new/page.tsx
import { createPost } from '../actions'
export default function NewPost() {
  return <form action={createPost}>...</form>
}
```

**Detection:**
- Code review: Route Handlers only called by same-app Server Components
- Performance: Network tab shows internal API calls during SSR
- Architecture smell: Every page has corresponding route handler

**Phase recommendation:** Phase 3 (New Routes/Subpages) - when adding new functionality, use correct patterns.

**Sources:**
- [Common Mistakes with Next.js App Router - Vercel](https://vercel.com/blog/common-mistakes-with-the-next-js-app-router-and-how-to-fix-them)
- Confidence: HIGH (official Vercel blog)

---

## Minor Pitfalls

Mistakes that cause annoyance but are fixable.

### Pitfall 9: Docker Base Image CVE Exposure

**What goes wrong:** Using mutable Docker tags (`golang:1.25-alpine`) without regular rebuilds leaves images vulnerable to CVEs fixed in patch releases.

**Why it happens:**
- Tags like `1.25-alpine` point to latest patch (e.g., 1.25.4 → 1.25.5)
- Docker caches resolved image locally
- Rebuilds use cached base image, missing security updates

**Consequences:**
- Production images run Go 1.25.4 with CVE-2025-61729 (High severity)
- Security scanners flag vulnerabilities
- Compliance failures
- Potential exploits

**Warning signs:**
```dockerfile
# MODERATE RISK - mutable tag, no update strategy
FROM golang:1.25-alpine
```

**Prevention:**

1. **Use --pull flag for fresh base images**:
```bash
docker build --pull -t myapp .
```

2. **Pin to specific digests** for supply chain integrity:
```dockerfile
FROM golang:1.25-alpine@sha256:abc123...
```
Note: Requires manual updates or automation (Docker Scout)

3. **Use specific patch versions**:
```dockerfile
FROM golang:1.25.5-alpine3.21  # Explicit patch version
```

4. **Automate base image updates**:
   - Renovate Bot / Dependabot for Dockerfile dependencies
   - Docker Scout for automated remediation PRs

5. **Regular rebuild schedule**:
```yaml
# .github/workflows/rebuild-base.yml
on:
  schedule:
    - cron: '0 2 * * 1'  # Weekly rebuilds
```

**Detection:**
- Security scans: Trivy, Snyk, or Docker Scout show base image CVEs
- Image inspection: `docker image inspect` shows old creation date
- Comparison: Local image differs from registry version

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - address when updating Go version.

**Sources:**
- [Docker Build Best Practices](https://docs.docker.com/build/building/best-practices/)
- [Go Docker Image CVE Fixes](https://github.com/golang-migrate/migrate/issues/1351)
- Confidence: HIGH (official Docker docs + recent CVE examples)

---

### Pitfall 10: Missing Runtime Dependencies in Multi-Stage Builds

**What goes wrong:** Using minimal base images (Alpine, distroless, scratch) for production stage without including required runtime dependencies. TLS connections fail, timezone operations break.

**Why it happens:** Developer focuses on image size, copies only binary to minimal base, doesn't test TLS or timezone features.

**Consequences:**
- TLS certificate verification fails (missing ca-certificates)
- `time.Now()` operations fail or return wrong timezone
- HTTP requests to HTTPS endpoints fail
- Subtle production bugs that don't appear in full images

**Warning signs:**
```dockerfile
# RED FLAG - minimal image without runtime deps
FROM scratch
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
# Missing: ca-certificates, tzdata
```

**Prevention:**

1. **Use Alpine with essential packages**:
```dockerfile
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /server /server
```

2. **Or use distroless with static binary**:
```dockerfile
# Build with static linking
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server

# Use distroless with ca-certificates
FROM gcr.io/distroless/static-debian12
COPY --from=builder /server /server
```

3. **Test TLS and timezone operations**:
```go
// Add to health check endpoint
func healthCheck() {
  // Test HTTPS
  _, err := http.Get("https://example.com")

  // Test timezone
  location, err := time.LoadLocation("America/New_York")
}
```

**Detection:**
- Runtime: "x509: certificate signed by unknown authority"
- Runtime: Timezone operations return UTC instead of expected timezone
- Health checks fail in production but work in development

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - verify when updating Docker configuration.

**Sources:**
- [Docker Multi-Stage Go Tutorial](https://oneuptime.com/blog/post/2026-01-07-go-docker-multi-stage/view)
- Confidence: MEDIUM (verified community best practices, not official docs)

---

### Pitfall 11: Parallel Routes Without default.js

**What goes wrong:** Next.js 16 requires explicit `default.js` files for all parallel route slots. Missing files cause build failures or unexpected `notFound()` behavior.

**Why it happens:** Next.js 15 was more permissive. Developers migrate without reading breaking changes.

**Consequences:**
- Build failures
- Routes show 404 unexpectedly
- Slots render blank instead of intended content

**Warning signs:**
```
app/
  @modal/
    login/
      page.tsx
  # Missing: @modal/default.tsx
```

**Prevention:**

1. **Add default.js to all slots**:
```tsx
// app/@modal/default.tsx
export default function Default() {
  return null  // or notFound()
}
```

2. **Audit parallel routes during upgrade**:
```bash
# Find all @ directories
find app -type d -name "@*"

# Verify each has default.tsx/js
```

**Detection:**
- Build-time: Next.js build error about missing default.js
- Runtime: Slots show 404 when not explicitly matched

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - if using parallel routes.

**Sources:**
- [Next.js 16 Upgrade Guide](https://nextjs.org/docs/app/guides/upgrading/version-16)
- Confidence: HIGH (official documentation)

---

### Pitfall 12: Local Images with Query Strings

**What goes wrong:** Next.js 16 requires explicit configuration for local images with query strings. Previously working `<Image src="/photo?v=1" />` fails.

**Why it happens:** Security tightening around local file access patterns.

**Consequences:**
- Images don't render (404)
- Build warnings
- Broken UI

**Warning signs:**
```tsx
// RED FLAG - query strings on local images
<Image src="/assets/logo.png?v=2" alt="Logo" />
```

**Prevention:**

1. **Configure localPatterns**:
```js
// next.config.js
module.exports = {
  images: {
    localPatterns: [
      {
        pathname: '/assets/**',
        search: '?v=*',
      },
    ],
  },
}
```

2. **Alternative: Remove query strings**, use Next.js cache busting:
```tsx
// Import images directly (automatic cache busting)
import logo from '@/public/assets/logo.png'
<Image src={logo} alt="Logo" />
```

**Detection:**
- Runtime: 404 errors for images with query params
- Build: Warning about unconfigured local image patterns

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - fix during Next.js upgrade if using query strings.

**Sources:**
- [Next.js 16 Upgrade Guide](https://nextjs.org/docs/app/guides/upgrading/version-16)
- Confidence: HIGH (official documentation)

---

### Pitfall 13: Bun Runtime Monorepo/Workspace Issues

**What goes wrong:** When using Bun runtime in monorepo/workspace setups, Next.js fails with "typescript package not installed" errors despite package being present.

**Why it happens:** Bun's workspace resolution differs from pnpm/npm. Type package hoisting works differently.

**Consequences:**
- Dev server fails to start
- Build failures
- Forces fallback to Node.js runtime (loses Bun performance benefits)

**Warning signs:**
```bash
# RED FLAG - monorepo with Bun runtime
packages/
  app/
    package.json  # Next.js app
  shared/
    package.json
# Using Bun workspaces

# Error: "Cannot find module 'typescript'"
```

**Prevention:**

1. **Add explicit dependencies in workspace packages**:
```json
{
  "devDependencies": {
    "@types/node": "latest",
    "typescript": "latest"
  }
}
```

2. **Test on Bun before deploying**:
```bash
bun run dev
bun run build
```

3. **Document workarounds**:
   - Non-monorepo projects work fine
   - Specific dependency installation may be needed

4. **Consider Vercel deployment**: Bun runtime supported natively, may handle workspace resolution better

**Detection:**
- Dev server: "Cannot find module 'typescript'" in monorepo
- Works with pnpm, fails with Bun
- Single-repo setup works, monorepo fails

**Phase recommendation:** Phase 1 (Dependencies Upgrade) - test Bun compatibility if using runtime optimization.

**Sources:**
- [Bun Workspace Issues with Next.js](https://github.com/oven-sh/bun/issues/25014)
- [Bun Runtime on Vercel](https://vercel.com/blog/bun-runtime-on-vercel-functions)
- Confidence: MEDIUM (GitHub issues, not fully resolved as of Jan 2026)

---

### Pitfall 14: Overusing "use client" Directive

**What goes wrong:** Adding `"use client"` to components that don't need interactivity. Defeats Server Components optimization, bloats client bundle.

**Why it happens:**
- Developers treat it as "fix for any error" magic directive
- Don't understand Server/Client component boundaries
- Apply Client patterns to all components by habit

**Consequences:**
- Larger JavaScript bundles
- Slower page loads
- Loss of automatic code splitting benefits
- Hydration overhead for non-interactive content

**Warning signs:**
```tsx
// RED FLAG - no interactivity, doesn't need "use client"
'use client'
export default function BlogPost({ title, content }) {
  return (
    <article>
      <h1>{title}</h1>
      <div>{content}</div>
    </article>
  )
}
```

**Prevention:**

1. **Only use "use client" when needed**:
   - Event handlers (onClick, onChange)
   - Browser APIs (localStorage, window)
   - React hooks (useState, useEffect)
   - Third-party interactive libraries

2. **Keep Client Components small**:
```tsx
// CORRECT - minimal Client Component
'use client'
function LikeButton({ postId }) {
  const [liked, setLiked] = useState(false)
  return <button onClick={() => setLiked(!liked)}>Like</button>
}

// Server Component with embedded Client Component
export default function BlogPost({ post }) {
  return (
    <article>
      <h1>{post.title}</h1>
      <div>{post.content}</div>
      <LikeButton postId={post.id} />  {/* Only button is client */}
    </article>
  )
}
```

3. **Composition pattern**: Pass Server Components as children to Client Components:
```tsx
'use client'
function ClientWrapper({ children }) {
  const [open, setOpen] = useState(false)
  return <div onClick={() => setOpen(!open)}>{children}</div>
}

// Server Component children
<ClientWrapper>
  <ServerRenderedContent />  {/* Stays on server */}
</ClientWrapper>
```

**Detection:**
- Bundle analysis: Large client bundles with mostly static content
- Dev tools: Many components hydrating that don't have interactivity
- Code review: "use client" on components without hooks or event handlers

**Phase recommendation:** Phase 3 (New Routes/Subpages) - establish pattern when adding new pages.

**Sources:**
- [Common Mistakes with App Router - Vercel](https://vercel.com/blog/common-mistakes-with-the-next-js-app-router-and-how-to-fix-them)
- Confidence: HIGH (official Vercel blog)

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1: Dependencies Upgrade | Async params breaking all routes | Run codemod first, use type generation |
| Phase 1: Dependencies Upgrade | Turbopack breaking webpack configs | Decide webpack opt-out or migrate to Turbopack |
| Phase 1: Dependencies Upgrade | React 19 library incompatibilities | Audit dependencies, update MUI/others |
| Phase 1: Dependencies Upgrade | Docker cache invalidation | Fix layer ordering before Go upgrade |
| Phase 1: Dependencies Upgrade | Go major version import paths | Check version jumps, update imports |
| Phase 2: Theming Implementation | next-themes hydration mismatch | Add suppressHydrationWarning, mount checks |
| Phase 2: Theming Implementation | Tailwind v4 config not working | Migrate to @theme CSS syntax |
| Phase 2: Theming Implementation | Theme flash on page load | Use CSS-based approach or loading states |
| Phase 3: New Routes/Subpages | Unnecessary Route Handlers | Fetch directly in Server Components |
| Phase 3: New Routes/Subpages | Overusing "use client" | Only add for interactivity |
| Phase 3: New Routes/Subpages | Not revalidating after mutations | Use revalidatePath/revalidateTag |

---

## Research Confidence Assessment

| Area | Confidence | Rationale |
|------|------------|-----------|
| Next.js 16 Breaking Changes | HIGH | Official Next.js documentation, verified migration guide |
| React 19 Compatibility | HIGH | Official React docs + real-world 2026 migration reports |
| next-themes Hydration | HIGH | Official library docs + verified community solutions |
| Tailwind v4 Migration | HIGH | Official Tailwind docs (if upgrading to v4) |
| Docker Multi-Stage | HIGH | Official Docker docs + recent 2026 tutorials |
| Go Module Versioning | HIGH | Official Go documentation |
| App Router Patterns | HIGH | Official Vercel blog posts |
| Bun Runtime Issues | MEDIUM | GitHub issues, not fully resolved yet |

---

## Sources

### Official Documentation (Highest Confidence)
- [Next.js 16 Upgrade Guide](https://nextjs.org/docs/app/guides/upgrading/version-16)
- [Next.js App Router Mistakes - Vercel](https://vercel.com/blog/common-mistakes-with-the-next-js-app-router-and-how-to-fix-them)
- [Tailwind CSS Theme Documentation](https://tailwindcss.com/docs/theme)
- [Docker Build Best Practices](https://docs.docker.com/build/building/best-practices/)
- [Go Modules Major Version Handling](https://go.dev/doc/modules/major-version)

### Verified Community Resources (High Confidence)
- [next-themes GitHub Repository](https://github.com/pacocoursey/next-themes)
- [Fixing Hydration Mismatch - Medium](https://medium.com/@pavan1419/fixing-hydration-mismatch-in-next-js-next-themes-issue-8017c43dfef9)
- [Docker Multi-Stage for Go (Jan 2026)](https://oneuptime.com/blog/post/2026-01-07-go-docker-multi-stage/view)
- [Tailwind v4 Migration Guide](https://medium.com/@mernstackdevbykevin/tailwind-css-v4-0-complete-migration-guide-breaking-changes-you-need-to-know-7f99944a9f95)

### Real-World Reports (Medium-High Confidence)
- [React 19 Upgrade Report (Jan 2026)](https://medium.com/@quicksilversel/i-upgraded-three-apps-to-react-19-heres-what-broke-648087c7217b)
- [Bun Workspace Issues](https://github.com/oven-sh/bun/issues/25014)
- [Docker Cache Strategies](https://docs.docker.com/build/cache/)

---

## Key Takeaways for Roadmap Planning

1. **Phase 1 is highest risk**: Next.js 16 async changes affect entire codebase. Must run codemod and test thoroughly before proceeding.

2. **Theming requires SSR awareness**: Hydration mismatches are guaranteed without proper mounting checks and suppressHydrationWarning.

3. **Tailwind v4 is opt-in breaking change**: If staying on v3, most theming pitfalls don't apply. Verify version before planning migration.

4. **Docker optimizations save CI/CD costs**: Layer ordering fix is low-effort, high-impact for iteration speed.

5. **App Router patterns differ from Pages Router**: Route Handlers, "use client", and Server Components need new mental models. Training/documentation phase recommended.

6. **Go dependency upgrades need import audits**: Major version bumps require code changes, not just `go get -u`.

7. **Test Bun runtime compatibility early**: Monorepo issues may force fallback to Node.js, affecting performance assumptions.
