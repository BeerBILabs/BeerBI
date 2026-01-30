# Codebase Concerns

**Analysis Date:** 2026-01-30

## Tech Debt

**Hardcoded API Token in Frontend Proxy:**
- Issue: API token falls back to hardcoded default value when env var is missing
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/app/api/proxy/[...path]/route.ts` (line 20)
- Impact: Security vulnerability - the string `'my-secret-token'` appears in code. If this is deployed as-is, anyone can authenticate to the API.
- Fix approach: Remove hardcoded fallback entirely. Require `API_TOKEN` environment variable to be set. Log a fatal error at startup if missing rather than using a default.

**TypeScript Strict Mode Disabled:**
- Issue: `tsconfig.json` has `"strict": false` but `"strictNullChecks": true` is set independently, creating inconsistent type safety
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/tsconfig.json` (lines 11, 28)
- Impact: Reduces type safety. Some files may have implicit `any` types. React components use `any` types (see line 93 in `UsersList.tsx`)
- Fix approach: Enable full strict mode (`"strict": true`) and fix type errors. Use proper TypeScript types instead of `any`.

**Database Migration Logic is Complex:**
- Issue: Store migration in `store.go` handles table recreation with aggregation, but the logic has fragile error handling
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/store.go` (lines 22-159)
- Impact: If migration partially fails (e.g., transaction commit fails), data could be silently lost or corrupted. The rename operation at line 150 logs but doesn't prevent issues.
- Fix approach: Add rollback safety checks. Log more granularly. Add dry-run mode for testing migrations. Consider using a dedicated migration framework.

**String-based JSON Responses in Backend API:**
- Issue: HTTP handlers manually construct JSON strings instead of using `json.Marshal`
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 300-301, 320-321)
- Impact: JSON injection vulnerability if user IDs contain special characters. Not properly escaping values.
- Fix approach: Replace manual string formatting with `json.NewEncoder(w).Encode()` (already done for other handlers). Use struct types for responses.

## Known Bugs

**Race Condition in Frontend Stats Loading:**
- Symptoms: Multiple concurrent requests to `/api/proxy/given` and `/api/proxy/received` for same users can cause duplicate or lost data in state
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/components/UsersList.tsx` (lines 25-87)
- Trigger: Rapidly changing date range while stats are loading, or multiple quick filter button clicks
- Workaround: Already partially mitigated with `cancelled` flag and 200ms debounce, but not fully safe. Race condition can still occur between fetch completion and state update.

**Missing Error Propagation in Frontend:**
- Symptoms: When API requests fail silently, users see "No data" instead of an error message
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/components/UsersPage.tsx` (lines 89-96), `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/components/UsersList.tsx` (lines 61-71)
- Trigger: Backend is unreachable, returns 5xx, or proxy endpoint returns 401
- Workaround: None - users must check browser console to diagnose

## Security Considerations

**Bearer Token Authentication is Weak:**
- Risk: Token is transmitted in plain HTTP headers. If TLS is not enforced, it can be intercepted.
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 607-627)
- Current mitigation: None documented. Assumes TLS at deployment layer.
- Recommendations: Document TLS requirement explicitly in deployment guides. Consider adding request signing or mutual TLS. Add token rotation strategy.

**API Token Exposed in Browser Proxy:**
- Risk: The frontend proxy passes API token in Authorization header. If frontend is compromised or logs are exposed, token is leaked.
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/app/api/proxy/[...path]/route.ts` (lines 18-29)
- Current mitigation: Token is server-side only (good). But hardcoded default breaks this (see Tech Debt above).
- Recommendations: Remove hardcoded token. Add token rotation. Consider moving proxy logic to dedicated API gateway.

**No Input Validation on User IDs:**
- Risk: User IDs from Slack could contain SQL injection vectors (though parameterized queries protect)
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 284-287, 304-307)
- Current mitigation: Parameterized SQL queries prevent injection. User ID comes from Slack API (trusted source).
- Recommendations: Add validation that user IDs match Slack format (`U[A-Z0-9]+`). Log suspicious patterns.

**Regex DoS Potential:**
- Risk: Emoji detection regex could be vulnerable to ReDoS if emoji format is malicious
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (line 469)
- Current mitigation: Emoji regex is simple and safe. User mention regex is pre-compiled.
- Recommendations: Add timeout to regex matching. Consider pre-compiling `emojiRe`.

## Performance Bottlenecks

**N+1 Query Pattern in Frontend User List:**
- Problem: For each user in list, frontend makes separate API call to `/api/proxy/user` to get name/avatar
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/components/UsersList.tsx` (lines 89-132)
- Cause: API design doesn't support batch user lookups. With 100 users, this is 100+ HTTP requests.
- Improvement path: Add batch endpoint `/api/users?user_ids=U1,U2,U3` to backend. Frontend batches requests.

**Unbounded List Sorting and Slicing:**
- Problem: `GetAllGivers()` and `GetAllRecipients()` load entire dataset into memory, no pagination
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/store.go` (lines 295-328)
- Cause: No pagination parameters in API design. With thousands of users, memory usage grows unbounded.
- Improvement path: Add `LIMIT` and `OFFSET` parameters to queries. Return page metadata (total count, has_next).

**Synchronous Event Processing:**
- Problem: Main Slack event handler blocks on database writes and API calls (daily limit check calls `PostMessage`)
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 419-556)
- Cause: Beer recording must complete before next message event can be processed. Slack timeout is 3 seconds.
- Improvement path: Use worker pool pattern. Queue events, process asynchronously, ack socket mode immediately.

## Fragile Areas

**Slack Connection Reconnection Logic:**
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 31-155, 562-583)
- Why fragile: Exponential backoff can reach 5 minutes. If backend crashes during reconnect loop, messages are lost. Multiple goroutines access `socketClient` without full synchronization.
- Safe modification: Lock all socket client access. Add max reconnect attempts before degraded mode. Log every state transition.
- Test coverage: No tests for reconnection logic, connection monitor, or health check endpoint.

**Database Schema Migration Path:**
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/store.go` (lines 22-159)
- Why fragile: Rewrites entire `beers` table on first run if schema doesn't have UNIQUE constraint. Complex SQL with multiple transaction boundaries. No rollback on partial failure.
- Safe modification: Write separate pre-migration and schema-upgrade functions. Test against realistic data sizes. Add data validation after migration.
- Test coverage: Basic `store_test.go` exists but doesn't test migration failure cases or data consistency after upgrade.

**Date Parsing is Inconsistent:**
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 178-200, 631-660)
- Why fragile: `parseSlackTimestamp` and `parseDateRangeFromParams` handle edge cases differently. Slack timestamps with fractions are parsed one way, query params another way.
- Safe modification: Create a single `TimeParser` type with test cases for edge cases (missing fraction, zero values, year boundaries).
- Test coverage: No explicit tests for timestamp parsing near year/month boundaries.

**Frontend State Synchronization:**
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/components/UsersList.tsx` (entire file)
- Why fragile: Multiple `useEffect` hooks that can be triggered in any order. State updates check `mounted.current` but not all callbacks do. Type assertions with `as string` are unsafe.
- Safe modification: Consider using a state machine or reducer pattern. Make all async operations respect the same cancellation mechanism.
- Test coverage: No tests for component behavior - only manual testing possible.

## Scaling Limits

**SQLite File I/O Bottleneck:**
- Current capacity: Single-file SQLite database with no connection pooling can handle ~100 concurrent queries before lock contention
- Limit: Cannot scale beyond single machine. Writes become serialized with high concurrency.
- Scaling path: Migrate to PostgreSQL. Add query caching with Redis. Implement read replicas for analytics queries.

**Frontend API Proxy is a Bottleneck:**
- Current capacity: Next.js server single instance can proxy ~1000 req/sec
- Limit: No horizontal scaling documented. Single proxy point of failure.
- Scaling path: Put proxy behind load balancer. Cache responses. Consider moving auth to dedicated service.

**Slack Socket Mode Single Connection:**
- Current capacity: One socket mode connection per bot instance can deliver ~100 messages/sec
- Limit: High-traffic workspace will lose events during spikes. Only one connection allowed per app token.
- Scaling path: Multiple bot instances with shared database. Use Slack's message history API as fallback.

## Dependencies at Risk

**Deprecated SQLite CGO Binding:**
- Risk: `github.com/mattn/go-sqlite3` requires C compilation. Adds build complexity and security surface.
- Impact: Docker build must include gcc. SQLite version is tied to system version.
- Migration plan: Consider pure Go SQLite (e.g., `modernc.org/sqlite`) for easier deployment and security patching.

**Old Slack SDK Version:**
- Risk: Code imports `github.com/slack-go/slack` without version pinning visible
- Impact: Breaking changes in SDK could break event handling. Socket mode implementation may be outdated.
- Migration plan: Pin to specific version in go.mod. Test against latest before upgrading.

**Next.js Turbopack in Development:**
- Risk: `"dev": "next dev --turbopack"` uses unstable bundler
- Impact: Different behavior between dev and production. Build may fail in CI.
- Migration plan: Remove `--turbopack` flag or use stable bundler in production. Test CI builds separately.

## Missing Critical Features

**No Event Retry Logic:**
- Problem: If beer recording fails, no automatic retry. Message from Slack is silently lost.
- Blocks: Reliability guarantees for beer tracking. Can't detect lost events.

**No Data Validation on Beer Messages:**
- Problem: Any user mention + emoji is recorded. No validation that mention is valid or user exists.
- Blocks: Data quality audits. Detecting spoofing attempts.

**No Audit Log:**
- Problem: No record of who gave/received beers, just totals by date.
- Blocks: Debugging, compliance, dispute resolution.

**No Admin Interface:**
- Problem: Can't manually correct beer counts or delete erroneous entries.
- Blocks: Operational support. Can only fix database directly.

## Test Coverage Gaps

**Backend Slack Event Processing:**
- What's not tested: Main event handler, emoji/mention parsing logic, daily limit enforcement
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 419-556)
- Risk: Regex changes could silently break mention detection. Daily limit logic untested.
- Priority: High - this is core business logic

**Frontend Component Rendering:**
- What's not tested: User list component, date picker, error states, loading states
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/components/UsersList.tsx`, `/home/tdolfen/Projects/github.com/BeerBILabs/frontend/project/components/UsersPage.tsx`
- Risk: UI bugs not caught. Type errors from `any` types in `namesOut`/`avatarsOut` not validated.
- Priority: Medium - detected by visual testing but slow

**API Authorization:**
- What's not tested: Bearer token validation, missing token rejection, invalid token rejection
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 607-627)
- Risk: Accidental removal of auth checks goes undetected. Public endpoints may be exposed.
- Priority: High - security critical

**Date Range Queries:**
- What's not tested: Boundary conditions (year end, leap years), timezone handling, invalid dates
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/store.go` (lines 245-272)
- Risk: Off-by-one errors in date ranges. Timezone confusion between UTC and local.
- Priority: Medium - affects correctness of analytics

**Database Connection Errors:**
- What's not tested: Database unreachable, corrupt database, quota exceeded, permission errors
- Files: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/main.go` (lines 245-254)
- Risk: Server starts successfully but fails on first query. No degraded mode fallback.
- Priority: High - impacts reliability

---

*Concerns audit: 2026-01-30*
