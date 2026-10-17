# BeerBI Performance, Scalability and Test Audit

**Date:** 2026-10-17
**Scope:** whole repository at commit `bfbd6cf` (branch `main`, clean tree)
**Audience:** the implementation agents that will act on this audit. Every finding names the file and line, states the evidence, proposes a fix and says how to verify it.

Goals the audit was asked to cover:

1. Improve backend/API speed
2. Improve handling of large data
3. Optimize page loads
4. Improve overall software speed
5. Make sure all tests are present

---

## 0. Verified facts (what was actually run)

| Check | Command | Result |
|---|---|---|
| Backend tests | `cd backend/bot && go test -count=1 -cover ./...` | `ok`, **coverage 5.6 %**, 3 test functions |
| Backend vet | `go vet ./...` | clean |
| Backend gofmt | `gofmt -l .` | `store_test.go` is not formatted |
| Frontend types | `./node_modules/.bin/tsc --noEmit` | clean |
| Frontend lint | `bun run lint` | **1 error**: unused import `CachedUserInfo` in `frontend/project/components/Leaderboard.tsx:7` |
| Frontend build | `bun run build` (Next 16.1.6, Turbopack) | succeeds; **every route is ƒ (dynamic)**, nothing is prerendered |
| Client JS | `find .next/static -name '*.js'` | 1.56 MB raw; largest chunks: recharts 404 KB (111 KB gz), react-datepicker+date-fns 2×268 KB (68 KB gz each), react-dom 220 KB |
| `just test` | `docker compose -f compose.test.yaml build backend-test` | **fails**: `target stage "test" could not be found` |
| SQLite index creation on a fresh DB | temporary probe test (deleted afterwards) | after first `migrate()`: only `sqlite_autoindex_beers_1`; the three `idx_beers_*` indexes appear only after a **second** `migrate()` |
| SQLite query plans | `EXPLAIN QUERY PLAN` via probe | `substr(ts_rfc,1,10) BETWEEN ? AND ?` → **SCAN**; `ts_rfc >= ? AND ts_rfc < ?` → **SEARCH ... USING INDEX** |
| SQLite pragmas in use | probe | `journal_mode=delete`, `busy_timeout=5000` |
| Timeline semantics | probe over 50 rows | `given == received` for **every** point (see C1) |

Severity scale used below: **P0** blocks correctness or CI, **P1** large measurable speed/scalability win, **P2** worthwhile, **P3** nice to have.

---

## 1. Backend (Go, `backend/bot/`)

### B1 (P1) Date filters defeat every index

All analytics and count queries filter with `substr(ts_rfc, 1, 10) BETWEEN ? AND ?` or `CAST(strftime('%Y', substr(ts_rfc,1,10)) AS INTEGER) BETWEEN ? AND ?`. Wrapping the column in a function makes the predicate non-sargable, so SQLite scans the whole table (or whole index) for every request.

Occurrences in `backend/bot/store.go`: lines 258, 270 (per-user counts, also used by the daily-limit check on every Slack message), 405, 409, 414 (timeline, three times), 466 (quarterly), 514, 539 (top users), 580 (heatmap), 622 (pairs).

Evidence (probe, 50 rows, representative shape):

```
substr(...) BETWEEN      -> SCAN beers USING INDEX idx_beers_giver_id_ts_rfc
ts_rfc >= ? AND ts_rfc < ? -> SEARCH beers USING INDEX idx_beers_emoji_ts_rfc (ts_rfc>? AND ts_rfc<?)
```

Fix:

- Replace every date predicate with `ts_rfc >= ? AND ts_rfc < ?` where the bound values are plain `YYYY-MM-DD` strings and the upper bound is `end + 1 day`. Because `ts_rfc` is stored as an RFC3339 UTC string (`store.go:247`), a 10-character prefix compares correctly, and this form also works for rows written by the legacy migration path which stores `YYYY-MM-DD HH:MM:SS` without a `T` (`store.go:144`). Do **not** compare against `...T00:00:00Z`, it would wrongly exclude those legacy rows.
- For the quarterly query, compute the year bounds in Go and use the same range predicate.
- Add a helper `dateBounds(start, end time.Time) (lo, hi string)` and unit-test it (see T1).

Verification: a Go test that runs `EXPLAIN QUERY PLAN` on each store query and asserts the plan contains `SEARCH` and not `SCAN`; a benchmark on a seeded DB (see T5).

### B2 (P0) Indexes are never created on a fresh database

`migrate()` creates the indexes in `store.go:51-65` only if the `beers` table already exists, then creates the table at `store.go:83-88` and returns. A brand-new deployment therefore runs without any secondary index until the process is restarted. Verified by the probe (see section 0).

Fix: move index creation to after the table exists (end of `migrate()`), and add the composite indexes the analytics queries actually need so they become index-only:

```sql
CREATE INDEX IF NOT EXISTS idx_beers_ts_giver     ON beers (ts_rfc, giver_id, count);
CREATE INDEX IF NOT EXISTS idx_beers_ts_recipient ON beers (ts_rfc, recipient_id, count);
```

Keep the existing `(giver_id, ts_rfc)` and `(recipient_id, ts_rfc)` indexes for the per-user count paths. Rename `idx_beers_emoji_ts_rfc` (it has nothing to do with emoji). Run `ANALYZE` after migration.

Verification: test that after `NewSQLiteStore` on an empty file `sqlite_master` lists all expected indexes.

### B3 (P1) SQLite connection is not configured for concurrent reads and writes

`main.go:67` opens the DB with `sql.Open("sqlite3", dbFile)` and no DSN options or pool settings. Verified pragmas: `journal_mode=delete`. In rollback-journal mode the Slack writer blocks every API reader and vice versa, which is exactly the pattern this service has (bot writes while dashboards read).

Fix:

- DSN: `file:<path>?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_foreign_keys=1&_cache_size=-32000`
- `db.SetMaxOpenConns(…)`: either a single writer connection plus a separate read pool, or at minimum `SetMaxOpenConns(4)` and `SetConnMaxIdleTime`.
- Run `PRAGMA optimize` on shutdown.
- Consider switching to `modernc.org/sqlite` (pure Go). It removes CGO, makes the Docker multi-arch build (`linux/arm64` under QEMU with gcc, `.github/workflows/docker-backend.yml`) several times faster, and allows `go test -race` without a C toolchain. Measure before committing to it: `modernc` is roughly 1.5 to 2× slower per query than `mattn` in micro-benchmarks, which is irrelevant at this data size but should be stated.

Verification: store test opens the DB through the production constructor and asserts `PRAGMA journal_mode` returns `wal`.

### B4 (P1) Timeline query scans the table three times and returns two identical series

`GetTimelineStats` (`store.go:402-422`) builds three CTEs over the same rows with the same WHERE clause. `given_counts` and `received_counts` are the same query (both `SUM(count)` over all beers grouped by period), so `given == received` on every point (verified). The frontend draws two overlapping areas.

Fix: single `SELECT period, SUM(count) ... GROUP BY period`. Decide the product meaning: either drop `received` from the API and chart, or make the timeline per-user (then given and received differ). Update `frontend/project/components/charts/ActivityChart.tsx` accordingly.

### B5 (P2) Unconditional `fmt.Printf` debugging in the hot path

`store.go` contains 21 `fmt.Printf` calls that run on every analytics query, including printing the full SQL text of the timeline query (`store.go:424`). These are unbuffered writes to stdout that bypass the zerolog level and cost real time under load, and they double the log volume in Docker.

Fix: delete them or route through `zerolog` at Debug level. Also lower the per-request `Info` logs in `handlers.go` ("request completed", "redis cache hit") to Debug; keep one structured access log line per request if desired.

### B6 (P1) User-name resolution hits the Slack API sequentially and ignores the SQLite cache

- `UserHandler` (`handlers.go:109-171`): Redis → Slack API → SQLite only on Slack failure. Without Redis (or on a Redis miss) **every** call is a Slack round-trip even though `user_cache` has the data.
- `BatchUsersHandler` (`handlers.go:544-599`): for every ID missing from Redis it calls `h.slackClient.GetUserInfo` **sequentially** (`handlers.go:550`) with no context, no timeout, no concurrency. 100 misses ≈ 100 × 100–300 ms ≈ 10–30 s per request. The frontend `fetch` has no timeout, so the leaderboard skeleton just hangs. Slack's `users.info` is Tier 4 (≈100 req/min), so a cold start with 100 users also trips rate limits.

Fix:

1. Lookup order: Redis → SQLite `user_cache` (treat rows with `updated_at` older than 7 days as stale but still usable while refreshing) → Slack.
2. Fetch Slack misses concurrently with a bounded worker pool (e.g. `errgroup` with limit 5) and a per-request context deadline (e.g. 8 s); return what you have and refresh the rest in the background.
3. Warm the cache: call `users.list` (paginated) once at startup and on a daily ticker, writing through to SQLite and Redis. After that the batch endpoint almost never touches Slack.
4. Return names and avatars inline from `/api/stats/top`, `/api/stats/pairs` and `/api/stats/combined` by LEFT JOINing `user_cache`. This removes the second round-trip the frontend currently does for every chart and leaderboard (see F3).

Verification: handler tests with a fake Slack client (interface) asserting cache order and concurrency; a test that 100 IDs with a 100 ms fake Slack latency complete in well under 2 s.

### B7 (P2) Combined analytics runs its queries one after another

`CombinedAnalyticsHandler` (`handlers.go:723-780`) executes timeline, top, heatmap and pairs sequentially. With B3 (WAL) in place they can run concurrently via `errgroup`. Also fold the quarterly series into the combined response; the frontend currently issues a fifth request for it (`frontend/project/components/charts/QuarterlyChart.tsx:36-56`).

### B8 (P2) No HTTP server hardening, compression or caching headers

- `main.go:145` creates `http.Server{Addr, Handler}` with no `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` or `MaxHeaderBytes`.
- Responses carry no `Cache-Control`/`ETag`. Stats for closed periods (last quarter, last year) never change and can be cached for hours; current-period stats can be cached for 30–60 s. The Next proxy currently strips all headers except `Content-Type` (`route.ts:55-58`), so fix both ends together (see F8).
- No gzip on the backend. The proxy re-serves bodies through Next which compresses, so this matters only for direct consumers; low priority.

### B9 (P2) Redis sync worker does 14 aggregate scans every 5 minutes

`StartSyncWorker` (`redis.go:523-547`) calls `PopulateFromDB` every 5 minutes, which runs `GetTopUsers` for 7 ranges (`redis.go:470-475, 486`), i.e. 14 GROUP BY scans. With B1/B2 fixed this is cheap; until then it is the single largest background load. Consider making the interval configurable and only refreshing ranges whose data can change (today, last 7 days, current month/quarter, all-time) on the short interval and closed ranges once a day.

### B10 (P2) Slack event processing is fully serial

`processEvents` (`slack.go:194-196`) calls the handler inline for each socket event. `handleMessageEvent` (`events.go:86-230`) then performs up to three `PostMessage` round-trips (`events.go:171, 181, 224`) and several DB writes before the next event is read. Ack is sent first (`events.go:55`), so Slack will not retry, but throughput is bounded by Slack API latency. For a single-team bot this is acceptable; if "software speed" includes bot responsiveness, hand each callback event to a bounded worker pool **after** `TryMarkEventProcessed` and keep the dedup guarantee documented in `CLAUDE.md`.

### B11 (P3) `processed_events` grows forever

`store.go:32-36`. Slack redelivery happens within minutes. Add a daily prune (`DELETE FROM processed_events WHERE ts < ?` for rows older than 30 days) so the UNIQUE index stays small.

### B12 (P3) Unauthenticated unbounded endpoints that nothing uses

`/api/givers` and `/api/recipients` (`handlers.go:189-230`, registered without auth in `main.go:133-134`) return every distinct ID with no limit. The frontend no longer calls them (grep of `api/proxy/` shows no usage). Remove them or put them behind auth with a limit.

### B13 (P2) Docker build settings

- `docker/backend/Dockerfile:16` builds with plain `go build`; add `-trimpath -ldflags="-s -w"` and `CGO_ENABLED=1` explicitly (or `0` after B3's driver switch).
- There is no `test` stage, which breaks `just test` (T4).
- `compose.yaml` gives the backend 0.5 CPU / 256 MB but Redis has no memory limit and no volume; a Redis restart wipes the leaderboards until the next 5-minute sync (see C2). Add `depends_on: cache` and a `maxmemory` policy or a volume.

---

## 2. Frontend (Next.js 16, `frontend/project/`)

### F1 (P1) Reading cookies in the root layout makes every page dynamic

`app/layout.tsx:47-48` calls `cookies()` to add the `dark` class server-side. That opts the whole tree out of static rendering: the build output shows all five routes as ƒ (server-rendered on demand). The same file already ships a blocking inline `<script>` (`layout.tsx:19-44`) that applies the class before first paint, so the server read is redundant.

Fix: remove the `cookies()` call and the server-derived class; keep the inline script. `/` and `/analytics` then prerender as static shells and the server only handles `/api/*`. If server-side theming is wanted later, isolate it in a small `Suspense` boundary rather than the root layout.

Verification: `bun run build` shows `○ (Static)` for `/`, `/_not-found` and `/analytics`.

### F2 (P1) Three-hop client waterfall before any data is visible

Both pages are 100 % client components; the first HTML is a skeleton. The home page then does, per `Leaderboard` instance (`components/Leaderboard.tsx`):

1. `fetch /api/proxy/stats/top?limit=100` (line 235)
2. **then** (quarter mode) `fetch` previous quarter top (line 260)
3. **then** `userDataManager.getUsers(...)` (line 299)

Three sequential round-trips, each through the Next proxy, which itself adds a hop to the backend.

Fix (in order of impact):

- Return names/avatars inline from the backend (B6.4) so step 3 disappears.
- Run step 1 and step 2 with `Promise.all`.
- Move the first-paint fetch into a Server Component that calls the backend directly with `API_TOKEN` (no proxy hop) and streams with `Suspense` + `app/loading.tsx`; keep the client component for interactive range changes. This also makes the data appear in the initial HTML.
- Add `app/loading.tsx` and `app/error.tsx` segments.

### F3 (P1) Duplicate requests on first paint, no batch de-duplication

- `app/page.tsx:10,13` mounts two `Leaderboard` instances. First paint of `/` issues **three** `/stats/top?limit=100` requests (current range, current quarter, previous quarter) plus **two** overlapping `/users?ids=…` batches.
- `/analytics` mounts `LeaderboardChart` ×2 and `NetworkChart`, each calling `userDataManager.getUsers` (`LeaderboardChart.tsx:76`, `NetworkChart.tsx:59`) at the same time → up to three overlapping `/users` batches with mostly the same IDs. `UserDataManager.getUsers` (`lib/userCache.ts:122-194`) has **no in-flight de-dup**; only the single-user path does (`userCache.ts:101-116`). Each duplicate batch becomes duplicate Slack calls on the backend (B6).

Fix: keep a module-level `Map<userId, Promise<CachedUserInfo|null>>` for batch fetches too, and resolve each ID from the shared pending promise. Better: with B6.4 the frontend stops resolving names at all.

### F4 (P2) localStorage user cache is re-parsed on every lookup

`getCachedUser` (`lib/userCache.ts:29-35`) calls `getUserCache()` (`lib/userCache.ts:10-18`) which `JSON.parse`s the entire blob. `getUsers` loops over every ID, so resolving N users parses the whole cache N times; `setCachedUser` in the retry path (`userCache.ts:219`) parses and stringifies per user. There is no size cap on the blob.

Fix: parse once into an in-memory `Map` at module load, read from memory, write through to localStorage with a debounced/`requestIdleCallback` flush, cap entries (e.g. LRU 500), wrap all storage access in try/catch (already done).

### F5 (P1) Heatmap is O(days²) and re-renders the whole grid on every hover

`components/charts/ActivityHeatmap.tsx`:

- line 192: `grid.find(...)` inside `weeks.map × 7 days` → O(weeks × 7 × days) ≈ O(days²/7). "This Year" (≈290 days, 42 weeks) ≈ 85 k scans per render; an all-time range from 2020 (≈2 500 days) ≈ 6 M.
- lines 46-70, 146: tooltip position/visibility is `useState` in the same component, so **every `mouseenter`** re-renders the entire grid and re-runs the search above.

Fix: in the existing `useMemo` build `Map<`${weekKey}:${dayOfWeek}`, cell>` and look up in O(1); move the tooltip into a sibling component driven by a ref or `useSyncExternalStore`, or render the grid in a `React.memo` child that receives only `grid`, `weeks`, `maxCount`; memoize `getColor`.

Verification: React Profiler shows no grid re-render on hover; a unit test that builds a 2 000-day grid in < 50 ms.

### F6 (P2) Analytics page unmounts all charts on every range change

`app/analytics/page.tsx:131-137, 152`: while `loading` is true the `{analyticsData && (...)}` block is removed, so each date change destroys and re-creates five Recharts instances, re-runs name resolution (F3) and flashes the layout.

Fix: keep the previous `analyticsData` visible while the next request is in flight (store `data` and `isFetching` separately in `useAnalyticsData`, use `useTransition`), show an inline loading indicator, and pass `key`-stable data to the charts. Also add an `AbortController` to `useAnalyticsData` so superseded requests are cancelled rather than just ignored.

### F7 (P2) Bundle and code splitting

Measured client JS: 1.56 MB raw. Recharts chunk 404 KB (111 KB gzip), react-datepicker + date-fns 2 × 268 KB (68 KB gzip each, the date-fns locale/format surface is pulled in twice), react-dom 220 KB.

- All five chart components are statically imported by `app/analytics/page.tsx:7-13` and the datepicker by both pages. Use `next/dynamic` with `ssr: false` and the existing skeletons for the Recharts components so `/` does not pay for Recharts at all.
- `react-datepicker/dist/react-datepicker.css` is imported twice (`app/globals.css:3` and `components/DateRangePicker.tsx:4`). Keep one.
- Evaluate replacing `react-datepicker` (brings date-fns) with two native `<input type="date">` elements styled with the existing tokens; it would remove ≈136 KB gzip from both pages.
- Add `@next/bundle-analyzer` behind an env flag so the next agents can measure. Next 16 no longer prints per-route sizes in the build log.

### F8 (P2) Proxy route handler

`app/api/proxy/[...path]/route.ts`:

- line 35: `console.log` on every request (synchronous stdout write in the request path).
- line 53: `await resp.text()` buffers the whole body, then re-sends it. Pass `resp.body` through as a stream.
- lines 55-58: only `Content-Type` is forwarded, so backend `Cache-Control`/`ETag` (B8) would be dropped. Forward those headers.
- lines 22-24: a browser-supplied `Authorization` header is forwarded verbatim. The proxy exists so the browser never needs a token; drop client auth pass-through and always use `API_TOKEN`.
- `NEXT_PUBLIC_BACKEND_BASE` is only used server-side. The `NEXT_PUBLIC_` prefix inlines it at build time when present and exposes it to the client bundle. Rename to `BACKEND_BASE` (server only) and update `compose*.yaml`, `scripts/wait-for-backend.sh`, `.github/workflows/docker-frontend.yml` and the READMEs (frontend README still documents a `NEXT_PUBLIC_API_TOKEN` that does not exist).

### F9 (P3) Configuration hygiene

- `next.config.js` has no `output: 'standalone'`; the Dockerfile relies on the undocumented `NEXT_PRIVATE_STANDALONE=true`. Set `output: 'standalone'` and remove the env hack.
- `tsconfig.json:11` `"strict": false`. Enable strict; `tsc --noEmit` is currently clean, so the migration cost is bounded.
- `components/ThemeProvider.tsx:17-20` fires a server action (`setThemeCookie('light')`) on every first visit without a cookie: an extra POST on first page load. Set the cookie client-side only.
- `Leaderboard` requests `limit=100` and renders two 100-row columns with 200 avatar images; acceptable for a team, but add a visible "show more" or virtualization before raising the cap.

---

## 3. Tests

### T1 (P0) Backend coverage is 5.6 % with three tests, none on business logic

Existing: `store_test.go` (`TestSQLiteStore_IncGet`, `TestSQLiteStore_Beers`), `store_givers_test.go` (`TestGetAllGiversAndRecipients`). `store_test.go` is not gofmt-formatted and writes to a relative `./testdata` path instead of `t.TempDir()`.

Required new tests (file → cases):

- `store_migrate_test.go`: fresh DB has all indexes (catches B2); migrate is idempotent; legacy table without UNIQUE is rebuilt and counts are preserved; mixed `ts_rfc` formats are filtered correctly after B1.
- `store_stats_test.go`: `GetTimelineStats` day/week/month, range boundaries inclusive on both ends, year boundary, leap day; `GetQuarterlyStats`; `GetTopUsers` ordering and limit; `GetHeatmapStats`; `GetPairStats`; `CountGivenInDateRange` / `CountReceivedInDateRange`; `TryMarkEventProcessed` returns true once then false; `IsEventProcessed`.
- `store_plan_test.go`: `EXPLAIN QUERY PLAN` for each query contains `SEARCH` and no `SCAN` (guards B1 permanently).
- `events_test.go`: `handleMessageEvent` with a fake store and fake Slack poster: mention + emoji counting and "last mention wins" rule; multiple recipients; self-gift ignored; subtype messages ignored; wrong channel ignored; daily limit reached / would be exceeded; duplicate envelope IDs processed once; `parseSlackTimestamp` with and without fraction.
- `utils_test.go`: `parseDateRangeFromParams` (day, start/end, missing, invalid, start > end), `getQuarterNumber`, `getQuarterStartMonth`.
- `handlers_test.go` (`net/http/httptest`): every endpoint for 200/400/401/404/500, JSON shape, `limit` capping, `granularity` validation; `BatchUsersHandler` cache order and cap at 100; `matchDateRangeToCache` for each range key around month/quarter/year boundaries using an injectable clock.
- `middleware_test.go`: missing header, malformed header, wrong token, correct token, constant-time path.
- `redis_test.go` using `github.com/alicebob/miniredis/v2`: get/set/TTL, pipeline batch, `GetTopGivers` on missing key, circuit breaker open/close, `calculateActiveRanges` by date, `PopulateFromDB`.
- `slack_test.go`: reconnect backoff caps at 5 min, `reconnectCount` resets on hello.

To make the handler and event tests possible without network, introduce two small interfaces (`SlackUserFetcher` with `GetUserInfo`, `MessagePoster` with `PostMessage`) and inject them; `*slack.Client` satisfies both.

Target: ≥ 70 % statement coverage on the package, enforced in CI.

### T2 (P0) Frontend has no test runner and no tests

`compose.test.yaml` only runs `bun run build` for the frontend.

Add Vitest + `@testing-library/react` + `jsdom` (Bun runs Vitest fine) and cover:

- `lib/dateUtils.ts`: `formatLocalDate`, `parseLocalDate`, `generateDateRange` incl. DST and month ends.
- `lib/quarters.ts`: `getQuarterDates` (this will fail today, see C6), `getPreviousQuarter`, `isValidQuarter`.
- `lib/userCache.ts`: TTL expiry, batch split at 100, in-flight de-dup (after F3), retry on partial failure, localStorage disabled.
- `lib/chartTheme.ts`: `formatNumber`, `formatDate` for each granularity.
- `lib/useAnalyticsData.ts`: request params, cancellation on prop change, error path.
- Components with mocked `fetch`: `Leaderboard` (loading, error + retry, empty, rank-change indicators), `DateRangePicker` (emits `YYYY-MM-DD`), `QuarterlyRankingsPanel` (tab logic, year change clamps quarter), `ActivityHeatmap` (grid shape, O(1) lookup after F5).
- Route handlers: `app/api/proxy/[...path]/route.ts` injects `Authorization`, forwards status and headers, streams body; `app/api/health/route.ts` 200/502.
- Optional: one Playwright smoke test (`/` renders a leaderboard against a seeded backend) run in `compose.test.yaml`.

### T3 (P0) No CI runs tests, lint or vet

`.github/workflows/` contains only tag-triggered Docker build/push jobs. Add `ci.yml` on `push` and `pull_request`:

- backend: `gofmt -l` (fail if non-empty), `go vet ./...`, `golangci-lint run`, `go test -race -coverprofile=cover.out ./...`, coverage threshold.
- frontend: `bun install --frozen-lockfile`, `tsc --noEmit`, `bun run lint`, `bunx vitest run --coverage`, `bun run build`.
- Make the Docker build workflows depend on CI passing (or trigger on tags only after CI).

### T4 (P0) `just test` is broken

`compose.test.yaml:6` requests `target: test`, but `docker/backend/Dockerfile` has only the `builder` and final stages. Add:

```dockerfile
FROM builder AS test
RUN go vet ./... && go test -race -cover ./...
```

and make the frontend test service run `bun run lint && bunx tsc --noEmit && bunx vitest run && bun run build`.

### T5 (P1) Performance tests and a realistic data seed

Nothing in the repo measures speed. Add:

- `backend/bot/bench_test.go`: `BenchmarkGetTopUsers`, `BenchmarkGetTimelineStats`, `BenchmarkCountGivenInDateRange` against a seeded temp DB (e.g. 500 users, 1 M rows over 6 years). Record before/after numbers for B1–B3 in the PR description.
- `backend/bot/cmd/seed` (or a `-seed N` flag guarded by a build tag): populate a dev DB so `just dev` can exercise the dashboards with large data.
- A `k6` or `hey` script under `scripts/` for `/api/stats/combined` and `/api/users?ids=…` with 100 IDs.

---

## 4. Correctness issues found on the way

These are not performance items but the implementing agents should know about them; several affect what "correct" output looks like after the performance changes.

- **C1** Timeline `given == received` always (B4). The chart currently shows two identical series.
- **C2** `GetTopGivers`/`GetTopRecipients` return a non-nil empty slice for a missing key (`redis.go:354-360`); the handlers treat non-nil as a cache hit (`handlers.go:382`, `handlers.go:737`). After a Redis restart (no volume in `compose.yaml`) every "common range" leaderboard is **empty** for up to 5 minutes. Treat an empty result for a key that does not exist (`EXISTS`) as a miss, or fall through to SQLite when the slice is empty.
- **C3** `calculateActiveRanges` (`redis.go:397-421`) increments `last-month` for beers given in the first 7 days of the **current** month and `last-quarter` in the first 7 days of the current quarter, which attributes current-period beers to the previous period until the next 5-minute sync overwrites them. The "today" and "last-7-days" keys also never roll over at midnight except through that sync.
- **C4** `GivenHandler`/`ReceivedHandler` build JSON with `fmt.Sprintf` (`handlers.go:61, 90`), no escaping. Use `json.NewEncoder` with a struct like every other handler.
- **C5** Legacy migration path (`store.go:142-158`): `datetime(..., 'unixepoch')` stores `YYYY-MM-DD HH:MM:SS` (no `T`, no `Z`), creating mixed `ts_rfc` formats; the `ALTER TABLE beers RENAME TO beers_old` after `DROP TABLE beers` is dead code. Normalise `ts_rfc` to RFC3339 in a one-time migration step so B1 can use strict comparisons later.
- **C6** `getQuarterDates` (`lib/quarters.ts:23-24`) formats local midnight with `toISOString()`. In any UTC-positive timezone (Europe) `new Date(2026, 0, 1)` becomes `2025-12-31`, so quarter rankings start one day early and end one day early (`Mar 31` → `2026-03-30`). Use `formatLocalDate` from `lib/dateUtils.ts`.
- **C7** `Leaderboard.tsx:229` uses `toISOString()` for "today" in the all-time path; same timezone issue.
- **C8** Frontend README documents `NEXT_PUBLIC_API_TOKEN`; the code uses `API_TOKEN`.

---

## 5. Suggested work breakdown for the implementation agents

Each workstream lists the files it owns so parallel agents do not collide. Finish WS-A and WS-F first; the others depend on their infrastructure.

| WS | Title | Items | Owns | Depends on |
|---|---|---|---|---|
| A | SQLite query layer | B1, B2, B3, B4, B5, B11, C5, T1 (store tests), T5 (bench) | `backend/bot/store.go`, `backend/bot/main.go` (DB open only), new `store_*_test.go`, `bench_test.go` | none |
| B | User identity resolution | B6, F3 (server side), C2 | `backend/bot/handlers.go` (user handlers + stats JOIN), `backend/bot/redis.go`, new interfaces + `handlers_test.go` (user parts), `redis_test.go` | A (WAL for concurrent reads) |
| C | HTTP layer and bot loop | B7, B8, B10, B12, C3, C4, `main.go` server config, `middleware_test.go`, `events_test.go`, `utils_test.go`, remaining `handlers_test.go` | `backend/bot/handlers.go` (stats handlers), `events.go`, `slack.go`, `middleware.go`, `utils.go`, `main.go` (server config) | A |
| D | Frontend data flow and first paint | F1, F2, F3 (client de-dup), F6, F8, C6, C7, C8 | `app/layout.tsx`, `app/page.tsx`, `app/analytics/page.tsx`, `app/api/proxy/**`, `components/Leaderboard.tsx`, `components/UsersPage.tsx`, `components/QuarterlyRankingsPanel.tsx`, `lib/useAnalyticsData.ts`, `lib/userCache.ts`, `lib/quarters.ts`, `compose*.yaml` (env rename), READMEs | B (inline names) for the full win; can start with Promise.all + de-dup |
| E | Frontend rendering and bundle | F4, F5, F7, F9 | `components/charts/**`, `components/DateRangePicker.tsx`, `components/ThemeProvider.tsx`, `next.config.js`, `tsconfig.json`, `app/globals.css`, `package.json` (analyzer, strict) | none |
| F | Test infrastructure and CI | T2 (runner + lib tests), T3, T4, gofmt fix, ESLint fix | `.github/workflows/ci.yml`, `docker/backend/Dockerfile` (test stage), `compose.test.yaml`, `vitest.config.ts`, `frontend/project/package.json` (scripts/devDeps), `store_test.go` formatting, `Leaderboard.tsx:7` | none; other WS add tests into the harness F creates |

Acceptance criteria for the whole effort:

- `just test` passes and runs backend tests, frontend lint, type-check, unit tests and build.
- CI workflow green on PRs; backend coverage ≥ 70 %, frontend lib coverage ≥ 80 %.
- `EXPLAIN QUERY PLAN` guard test proves no table scans on `beers` for date-bounded queries.
- `bun run build` prints `○ (Static)` for `/` and `/analytics`.
- First paint of `/` issues at most two backend requests (current and previous period) and zero `/users` requests.
- `/api/users?ids=` with 100 cold IDs returns in < 2 s with a 100 ms fake Slack latency.
- Heatmap hover causes no re-render of the grid (React Profiler).
- Benchmarks in `bench_test.go` recorded before and after in the PR.

---

## 6. How the measurements were taken (reproduce)

```bash
# backend
cd backend/bot && go vet ./... && gofmt -l . && go test -count=1 -cover ./...

# frontend
cd frontend/project && bun install --frozen-lockfile \
  && ./node_modules/.bin/tsc --noEmit && bun run lint && bun run build
find .next/static -name '*.js' -exec du -ck {} + | tail -1
ls -S .next/static/chunks/*.js | head -5 | xargs du -k

# docker test target
docker compose -f compose.test.yaml build backend-test   # fails: target stage "test" not found
```

The SQLite probe (index creation, query plans, pragma values, timeline equality) was a temporary `_test.go` file in `backend/bot` that opened a fresh temp DB through `NewSQLiteStore`, listed `sqlite_master` before and after a second `migrate()`, ran `EXPLAIN QUERY PLAN` on the `substr` and range forms of the top-givers and per-user count queries, and compared `Given` with `Received` over 50 seeded rows. It was deleted after the run; re-creating it as `store_plan_test.go` is part of T1.
