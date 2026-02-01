# Phase 1: Dependency Modernization - Research

**Researched:** 2026-02-01
**Domain:** Multi-stack dependency upgrade (Next.js, Go, Bun, Docker Alpine)
**Confidence:** HIGH (all target versions verified against official sources)

## Summary

Phase 1 covers four discrete dependency upgrades across the BeerBot stack. Each upgrade target was verified against official release channels. Two of the four specified target versions required clarification against actual availability:

- Next.js 15.5.11: Confirmed to exist and is the latest 15.5.x patch. However, it carries a **known, unresolvable `@next/swc` version mismatch warning** (GitHub Issue #89251). This warning is cosmetic/non-blocking -- Next.js 15.5.11 lists `@next/swc@15.5.7` as its own optional dependency, so the mismatch is baked into the package itself. The warning cannot be fixed on the 15.5.x line; the only resolution is upgrading to 16.x, which is out of scope for this phase.
- Go 1.25.6: Confirmed released 2026-01-15 as a security patch. The current `go.mod` already declares `go 1.25` with `toolchain go1.25.3`, so only the toolchain line needs updating.
- Bun 1.3.7: Confirmed released 2026-01-27. The `package.json` engine constraint is already `"bun": "^1.3.0"`, so this is a runtime upgrade only -- no config changes needed.
- Alpine 3.23: Confirmed available (current patch 3.23.3). The backend Dockerfile final stage currently uses `alpine:3.20` and must be updated. The Go builder stage (`golang:1.25-alpine`) already resolves to Alpine 3.23.3 as of 2026-01-28.

**Primary recommendation:** Upgrade each component independently in order (Go, Next.js, Bun, Docker), verifying compilation/build at each step before proceeding to the next. The `@next/swc` warning on 15.5.11 is expected and should be documented as a known issue, not treated as a failure.

## Standard Stack

The project uses these specific versions. No alternatives are in scope -- this is a patch-level upgrade within existing major versions.

### Core Targets
| Component | Current | Target | File(s) to Change |
|-----------|---------|--------|-------------------|
| Next.js | ^15.5.6 (resolves to 15.5.6 or higher per lockfile) | 15.5.11 | `frontend/project/package.json` |
| Go | 1.25 / toolchain go1.25.3 | 1.25.6 | `backend/bot/go.mod` |
| Bun | ^1.3.0 (runtime) | 1.3.7 | Runtime install (no config change needed) |
| Alpine (backend final stage) | 3.20 | 3.23 | `docker/backend/Dockerfile` |

### Supporting Context
| Component | Current | Notes |
|-----------|---------|-------|
| React | ^19.2.0 | No upgrade specified; stays as-is |
| Tailwind CSS | ^4.1.14 | No upgrade specified; stays as-is |
| Go builder image | golang:1.25-alpine | Already resolves to Go 1.25.6 + Alpine 3.23.3 as of Jan 28, 2026 |
| Bun Docker image | oven/bun:1 | Debian-based; no change needed for this phase |
| eslint-config-next | ^15.5.6 | Should be upgraded alongside next to stay in sync |

### Installation Commands
```bash
# Frontend: update Next.js and eslint-config-next to 15.5.11
cd frontend/project
bun add next@15.5.11 eslint-config-next@15.5.11

# Go backend: update toolchain directive
cd backend/bot
go get go@1.25.6
# This updates go.mod toolchain line; then verify with:
go mod tidy
go build ./...

# Bun runtime: no package.json change needed.
# The engines field is "^1.3.0" which already permits 1.3.7.
# If a specific CI/Docker pin is needed, see Docker section below.

# Docker: edit Dockerfile directly (see Code Examples)
```

## Architecture Patterns

### Upgrade Sequencing Pattern
**What:** Upgrade one component at a time, verify it builds/compiles, then proceed. Do not batch all four upgrades into a single commit.
**When to use:** Always. Each component has independent verification criteria and independent failure modes.
**Rationale:** If a batch upgrade fails, you cannot isolate which component caused the failure. Sequential upgrades make bisection trivial.

### Recommended Order
1. **Go backend** -- simplest upgrade (toolchain directive + `go mod tidy`). Verify with `go build ./...` and `go test ./...`.
2. **Next.js frontend** -- moderate complexity. Verify with `bun run build`. The `@next/swc` warning will appear; this is expected.
3. **Bun runtime** -- runtime-only upgrade if running locally. If pinning in Docker, handled in step 4.
4. **Docker images** -- update Alpine base in backend Dockerfile. Rebuild all containers and verify health checks pass.

### Docker Multi-Stage Build Pattern (Backend)
The backend Dockerfile uses a two-stage build:
- **Stage 1 (builder):** `golang:1.25-alpine` -- already resolves to Alpine 3.23.3 + Go 1.25.6. No change needed here.
- **Stage 2 (final):** `alpine:3.20` -- this is the line that must change to `alpine:3.23`.

The frontend Dockerfile uses `oven/bun:1` (Debian-based) for all stages. This image does NOT use Alpine. Do not attempt to switch the frontend to an Alpine-based Bun image -- the current Dockerfile installs `wget` and `curl` via `apt-get`, which only works on Debian-based images.

### Anti-Patterns to Avoid
- **Upgrading Next.js and React simultaneously without need:** React is already at ^19.2.0 and is not a target for this phase. Upgrading it alongside Next.js introduces unnecessary variables.
- **Trying to fix the @next/swc warning on 15.5.x:** There is no fix available on this version line. The warning is a known upstream issue (vercel/next.js#89251). Attempting workarounds (force reinstall, pinning @next/swc) will not resolve it.
- **Using `alpine:3.23.3` instead of `alpine:3.23`:** Pin to the minor version tag (`3.23`), not the patch tag. Docker will pull the latest patch automatically, which is the correct behavior for security updates.
- **Forgetting `go mod tidy` after updating the toolchain:** The `go get go@1.25.6` command updates the `go.mod` file, but `go mod tidy` is still needed to clean up any indirect dependency changes.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Next.js upgrade | Manual version resolution across all @next/* packages | `bun add next@15.5.11 eslint-config-next@15.5.11` | Bun resolves transitive @next/* deps automatically |
| Go toolchain update | Manual go.mod text editing | `go get go@1.25.6` then `go mod tidy` | The `go` tool manages toolchain directive semantics (see Go 1.25 behavior change below) |
| Alpine version verification | Custom health check scripts | Docker healthcheck in docker-compose.yml (already configured) | Health checks already exist for both services |
| Dependency compatibility checking | Custom scripts | `bun run build` (Next.js) and `go build ./...` (Go) | These are the canonical verification steps; anything else is redundant |

**Key insight:** All four upgrades are managed by their respective ecosystem tooling. The correct approach is to invoke the tool (`bun add`, `go get`, Docker image tag change) and then run the standard build verification. Do not manually edit lockfiles or version resolution logic.

## Common Pitfalls

### Pitfall 1: @next/swc Version Mismatch Warning
**What goes wrong:** After upgrading to next@15.5.11, the dev server and build output a warning: "Mismatching @next/swc version, detected: 15.5.7 while Next.js is on 15.5.11."
**Why it happens:** Next.js 15.5.8+ was released without corresponding @next/swc packages. The next@15.5.11 package itself declares @next/swc@15.5.7 as its optional dependency, making the mismatch inherent to the package.
**How to avoid:** You cannot avoid it on 15.5.x. Accept the warning. Do not treat it as a build failure. The build and runtime function correctly despite the warning.
**Warning signs:** If someone sees this warning and tries to "fix" it by upgrading to Next.js 16, that is scope creep for this phase.

### Pitfall 2: Go Toolchain Directive Auto-Management
**What goes wrong:** Running `go mod tidy` or `go get` on Go 1.25 no longer auto-injects a `toolchain` line (changed in Go 1.25). If you run `go get go@1.25.6` and the go line already says `go 1.25`, Go 1.25 will omit the toolchain line entirely (since toolchain == go version, it is implied).
**Why it happens:** Go 1.25 changed the behavior: when the toolchain matches the go directive exactly, the toolchain line is deleted as redundant.
**How to avoid:** Run `go get go@1.25.6` and then `go mod tidy`. Accept whatever go.mod state results. Do not manually add back a `toolchain` line if it was removed -- that is the correct behavior.
**Warning signs:** The go.mod file may end up with just `go 1.25.6` and no `toolchain` line. This is correct and expected.

### Pitfall 3: Alpine 3.23 apk v3
**What goes wrong:** Alpine 3.23 ships apk-tools v3, which is a major rewrite. In Docker contexts, this is transparent because you are pulling a fresh image, not upgrading in-place. However, if any custom scripts or CI pipelines interact with apk internals or libapk directly, they may break.
**Why it happens:** apk v3 is a new package manager version with a different internal API.
**How to avoid:** In Docker, simply change `FROM alpine:3.20` to `FROM alpine:3.23`. The `apk add --no-cache` syntax is unchanged. For this project, the backend Dockerfile only runs `apk add --no-cache ca-certificates wget curl`, which works identically on apk v3.
**Warning signs:** Any `apk` command that worked before should continue to work. If something fails, check whether it relies on libapk programmatic access (not the case for this project).

### Pitfall 4: Bun Lockfile Staleness
**What goes wrong:** The `bun.lock` file pins exact versions. If `package.json` specifies `next@15.5.11` but the lockfile was generated with an older Bun or different resolution, the installed version may differ.
**Why it happens:** Bun's lockfile format is binary and version-dependent.
**How to avoid:** After editing `package.json`, always run `bun install` (not just `bun add`) to regenerate the lockfile. Commit both `package.json` and `bun.lock` together.
**Warning signs:** `bun run build` succeeds but the installed next version (check with `bun list next`) differs from the target.

### Pitfall 5: docker-compose.dev.yml Dockerfile Path Mismatch
**What goes wrong:** The `docker-compose.dev.yml` references `../docker/frontend/Dockerfile.dev` (relative path with single `..`), while other compose files use `../../docker/frontend/Dockerfile`. This is intentional -- the dev compose file's `context` is `./frontend/project`, so the relative path to the docker directory differs.
**Why it happens:** Different build contexts require different relative paths to the Dockerfile.
**How to avoid:** Do not change any Dockerfile path references in docker-compose files. Only change the `FROM` line inside the Dockerfiles themselves.
**Warning signs:** `docker compose build` fails with "Dockerfile not found." This means a path was accidentally changed.

## Code Examples

Verified patterns from official sources:

### Go: Updating Toolchain to 1.25.6
```go
// backend/bot/go.mod -- before:
module github.com/DanielWeeber/beer-with-me/bot

go 1.25

toolchain go1.25.3

// After running: go get go@1.25.6 && go mod tidy
// Expected result (Go 1.25 behavior: toolchain line omitted when equal to go line):
module github.com/DanielWeeber/beer-with-me/bot

go 1.25.6

// Source: https://go.dev/doc/toolchain (Go 1.25 toolchain management)
```

### Next.js: Upgrading to 15.5.11
```bash
# Run from frontend/project/
bun add next@15.5.11 eslint-config-next@15.5.11

# Verify the upgrade worked:
bun list next
# Should show: next@15.5.11

# Build to verify functionality:
bun run build
# Expected: build succeeds, may show @next/swc mismatch warning (this is normal)

# Source: package.json engines and Next.js 15.5 release notes
# https://nextjs.org/blog/next-15-5
```

### Docker: Backend Dockerfile Alpine Update
```dockerfile
# docker/backend/Dockerfile

# Stage 1: Builder -- NO CHANGE NEEDED
# golang:1.25-alpine already resolves to Alpine 3.23.3 + Go 1.25.6
FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/bot ./

# Stage 2: Final image -- CHANGE THIS LINE ONLY
# FROM alpine:3.20        <-- old
FROM alpine:3.23          # <-- new
RUN apk add --no-cache ca-certificates wget curl
COPY --from=builder /bin/bot /bin/bot
VOLUME /data
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/bin/bot"]

# Source: https://hub.docker.com/_/alpine (official image tags)
# Alpine 3.23.0 release: https://www.alpinelinux.org/posts/Alpine-3.23.0-released.html
```

### Docker: Frontend Dockerfile -- No Changes Needed
```dockerfile
# docker/frontend/Dockerfile
# The oven/bun:1 image is Debian-based and does NOT need Alpine migration.
# Bun 1.3.7 will be picked up when the lockfile is regenerated with bun@1.3.7.
# No Dockerfile changes are required for the frontend for this phase.

# The Bun version in the Docker image is controlled by the oven/bun:1 tag.
# To pin Bun to exactly 1.3.7 in Docker, use: oven/bun:1.3.7
# However, since the requirement is "updated to 1.3.7" and oven/bun:1 will
# resolve to the latest 1.x (currently 1.3.8), pinning to 1.3.7 specifically
# requires changing FROM oven/bun:1 to FROM oven/bun:1.3.7 if exact version
# pinning is desired.
```

### Verification: Full Stack Build
```bash
# 1. Build and test Go backend
cd backend/bot
go build ./...
go test ./...

# 2. Build Next.js frontend
cd frontend/project
bun install
bun run build

# 3. Build Docker images
cd <repo root>
docker compose build

# 4. Run and verify health checks
docker compose up -d
# Wait for start_period (40s for backend)
curl http://localhost:8080/api/health   # backend
curl http://localhost:3000/api/health   # frontend proxy
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Go toolchain auto-injects `toolchain` line | Go 1.25 omits `toolchain` when it equals `go` directive | Go 1.25 (2025) | go.mod files are cleaner; do not manually add toolchain line back |
| Alpine apk v2 | Alpine 3.23 ships apk v3 | Alpine 3.23 (Dec 2025) | Transparent for `apk add` usage; breaking only for libapk programmatic users |
| Next.js 15.5 warns about deprecations for 16 | 15.5.x is the last stable 15.x line before 16 | Next.js 15.5 (2025) | 15.5.11 is a maintenance-only release; 16 is the active development line |

**Deprecated/outdated:**
- `next lint` command: Deprecated in Next.js 15.5, removed in 16. This project's `package.json` already uses `"lint": "eslint ."` -- no action needed.
- `alpine:3.20`: End of support approaching. Alpine 3.23 is the current stable release.

## Open Questions

1. **Should the frontend Dockerfile pin Bun to exactly 1.3.7?**
   - What we know: The requirement says "Bun runtime updated to 1.3.7." The current Dockerfile uses `oven/bun:1` which resolves to the latest 1.x (currently 1.3.8). The `package.json` engines field is `"bun": "^1.3.0"`.
   - What's unclear: Whether "updated to 1.3.7" means "at least 1.3.7" or "exactly 1.3.7."
   - Recommendation: Change `FROM oven/bun:1` to `FROM oven/bun:1.3.7` in both `Dockerfile` and `Dockerfile.dev` to satisfy the literal requirement. This is a safe pin -- 1.3.7 is a released, stable version.

2. **Should eslint-config-next be pinned to 15.5.11?**
   - What we know: eslint-config-next is currently `^15.5.6`. It should stay in sync with the `next` package version.
   - What's unclear: Whether ^15.5.6 already resolves to 15.5.11 via the lockfile.
   - Recommendation: Explicitly upgrade it alongside next: `bun add next@15.5.11 eslint-config-next@15.5.11`.

## Sources

### Primary (HIGH confidence)
- Go release history: https://go.dev/doc/devel/release -- confirmed go1.25.6 released 2026-01-15
- Go toolchain docs: https://go.dev/doc/toolchain -- confirmed Go 1.25 toolchain line behavior
- Next.js 15.5 blog: https://nextjs.org/blog/next-15-5 -- confirmed 15.5 features and deprecations
- Next.js CVE advisory: https://github.com/vercel/next.js/discussions/86939 -- confirmed 15.5.7+ is security-patched
- Alpine 3.23 release: https://www.alpinelinux.org/posts/Alpine-3.23.0-released.html -- confirmed 3.23.0 released Dec 3, 2025
- Golang Docker Hub: https://hub.docker.com/_/golang/ -- confirmed golang:1.25-alpine resolves to Go 1.25.6 + Alpine 3.23.3
- Bun GitHub releases: https://github.com/oven-sh/bun/releases -- confirmed v1.3.7 released Jan 27, 2026

### Secondary (MEDIUM confidence)
- Next.js @next/swc mismatch issue: https://github.com/vercel/next.js/issues/89251 -- confirmed warning is unresolvable on 15.5.x, affects 15.5.8-15.5.11
- Next.js 15.5.11 existence confirmed via WebSearch cross-referencing npm versions page and GitHub issue #89251 (both reference 15.5.11 as the latest 15.5.x)
- Bun @next/swc interaction: https://github.com/oven-sh/bun/issues/8726 -- known issue with Bun not updating @next/swc alongside next

### Tertiary (LOW confidence)
- Alpine 3.23 apk v3 breaking changes detail: https://wiki.alpinelinux.org/wiki/Release_Notes_for_Alpine_3.23.0 -- referenced but not fully fetched; the `apk add` syntax is confirmed unchanged

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all four target versions verified against official release channels
- Architecture: HIGH -- upgrade patterns verified against Go toolchain docs and Next.js release notes
- Pitfalls: HIGH for @next/swc warning (GitHub issue confirmed), HIGH for Go toolchain behavior (official docs), MEDIUM for Alpine apk v3 (release notes referenced, not fully fetched)

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (stable patch releases; unlikely to change within 30 days)
