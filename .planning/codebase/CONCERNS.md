# Codebase Concerns

**Analysis Date:** 2026-01-30

## Security Concerns

**Hardcoded API Token in Proxy:**
- Issue: Frontend proxy route uses a hardcoded fallback token with placeholder value "my-secret-token"
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/app/api/proxy/[...path]/route.ts` (line 20)
- Impact: If `process.env.API_TOKEN` is not set, requests will fail authentication or reveal placeholder token in logs
- Current mitigation: Environment variable configuration required, but fallback is insecure
- Recommendations:
  - Remove fallback token entirely; require env var to be set
  - Add startup validation that API_TOKEN is present before server starts
  - Never log the actual token, only log auth source ("client", "HARDCODED", or "none")

**Exposed Slack Credentials in Dev Files:**
- Issue: `.env.dev` file contains real Slack bot tokens and app tokens
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/.env.dev`
- Impact: Credentials exposed in version control; if leaked, attacker can impersonate the bot
- Current mitigation: None detected (file appears to be committed)
- Recommendations:
  - Remove `.env.dev` from version control (add to `.gitignore`)
  - Document `.env.dev` example with placeholder values (`.env.dev.example`)
  - Use secrets management system (GitHub Secrets, Vault, etc.) for CI/CD

**Proxy Forwards Authorization Headers Directly:**
- Issue: Frontend proxy accepts Authorization headers from client and forwards them to backend
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/app/api/proxy/[...path]/route.ts` (lines 19-29)
- Impact: Client could potentially pass arbitrary auth headers; no validation of header format
- Current mitigation: Backend has `authMiddleware` that validates Bearer token format and compares against server-side token
- Recommendations:
  - Document that proxy strips/ignores client Authorization headers in favor of server-side token
  - Remove client Authorization header forwarding entirely; always use server-side token

**Image URLs Loaded from Untrusted Source:**
- Issue: Avatar URLs (`j.profile_image`) are loaded directly from Slack API without validation and rendered via Next.js Image
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/UsersList.tsx` (lines 108-110, 159-166)
- Impact: Malicious Slack workspace could modify avatar URLs; `unoptimized` flag bypasses Next.js image optimization
- Current mitigation: `unoptimized` flag used (line 164), image requires valid URL
- Recommendations:
  - Remove `unoptimized` flag to leverage Next.js optimization and caching
  - Add URL validation whitelist (only allow Slack CDN domains)
  - Consider using fallback avatar if URL fails to load

**Bearer Token in Logs:**
- Issue: Bearer token is logged in HTTP handler errors
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/http_handlers.go` (line 38)
- Impact: Logs may expose API token if printed in debug output
- Current mitigation: Line 38 logs headers which contain Authorization header with token
- Recommendations:
  - Sanitize logs to mask Authorization headers
  - Never log full header values in production; use structured logging with token masked

## Performance Bottlenecks

**Unoptimized Image Loading in UsersList:**
- Issue: All avatar images use `unoptimized=true` flag, preventing Next.js optimization
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/UsersList.tsx` (line 164)
- Impact: No image compression, caching, or responsive sizing; larger bandwidth usage
- Current mitigation: None; fallback to default img tag behavior
- Improvement path:
  - Remove `unoptimized` flag to enable Next.js optimization
  - Ensure `width` and `height` are always provided (currently provided as 32x32)
  - Add `priority={false}` for lazy loading of non-critical avatars

**N+1 Query Pattern in Frontend:**
- Issue: UsersList fetches stats for each user sequentially with concurrency limit of 5
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/UsersList.tsx` (lines 46-76)
- Impact: If 100+ users exist, makes 100 separate requests instead of batch endpoint
- Current mitigation: Concurrency limit of 5 prevents complete parallelization but still multiple requests
- Improvement path:
  - Create batch endpoint: `/api/proxy/given-batch?users=U1,U2,U3` returning multiple user stats
  - Implement pagination if users list is very large
  - Cache results in component state to prevent refetches

**Fetching User Names/Avatars Separately:**
- Issue: Two separate effect hooks fetch user names and avatars; total of 100+ requests per 100 users
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/UsersList.tsx` (lines 89-132)
- Impact: Duplicates HTTP requests and roundtrips for user profile data
- Current mitigation: Batched with concurrency limit of 5
- Improvement path:
  - Merge into single `/api/proxy/user-batch` endpoint returning both names and avatars
  - Consolidate into one useEffect hook
  - Cache results to prevent refetches on range changes

**Hardcoded Top 100 Users Limit:**
- Issue: UsersList always renders only top 100 users by count
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/UsersList.tsx` (line 138)
- Impact: Leaderboard truncates if more than 100 users give/receive beers
- Current mitigation: Slice applied at sort level
- Improvement path:
  - Add pagination controls to show more users
  - Load users on demand with virtual scrolling for large lists
  - Make limit configurable

## Error Handling Gaps

**Silent Error Suppression in Frontend:**
- Issue: Network errors in UsersList are caught but only logged to console; UI shows no error state
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/UsersList.tsx` (lines 60-71, 101-115)
- Impact: User has no way to know if data failed to load; partial data displayed without indication
- Current mitigation: `console.error` called, but no UI feedback
- Recommendations:
  - Add error state to component
  - Show user-friendly error message in UI
  - Implement retry mechanism with exponential backoff

**Unhandled Promise Rejection in Proxy Load:**
- Issue: `load()` function in UsersPage.tsx returns Promise but isn't awaited; errors may be unhandled
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/UsersPage.tsx` (lines 95-112)
- Impact: Fetch failures silently fail without retry or user notification
- Current mitigation: Individual fetch responses checked with `.ok` property
- Recommendations:
  - Add `.catch()` handler to `load()` call
  - Implement error boundary or error state
  - Show loading state while fetching

**Missing Boundary Validation on Date Range:**
- Issue: Date range inputs accept any valid date; no validation that start <= end
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/DateRangePicker.tsx`
- Impact: Reverse date ranges could cause backend errors or unexpected results
- Current mitigation: None visible in code review
- Recommendations:
  - Validate that start date <= end date before sending request
  - Show error if user selects invalid range
  - Auto-swap dates if end < start

## Data Consistency Concerns

**Event Deduplication Depends on Slack Envelope ID:**
- Issue: Event deduplication uses `envelope_id` from Slack, but falls back to computed ID using timestamp
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/slack.go` (lines 274-281)
- Impact: If envelope_id is missing/empty, computed fallback using user/timestamp/channel may collide
- Current mitigation: Fallback to `msg|channel|user|timestamp` format
- Recommendations:
  - Always require envelope_id from Slack; log error if missing
  - Consider adding microsecond precision to timestamp fallback
  - Monitor for duplicate beers logged with same giver/recipient

**Database Migration Complexity:**
- Issue: Store migration logic attempts to add columns and recreate table with aggregation
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/store.go` (lines 22-159)
- Impact: Complex transaction logic could fail partway, leaving schema inconsistent
- Current mitigation: Transaction with rollback on error
- Recommendations:
  - Add pre-migration backup of database
  - Add migration version tracking to prevent re-running
  - Test migration path with large datasets
  - Consider using migration tool (golang-migrate, flyway) instead of inline logic

**Slack Timestamp Parsing Fragility:**
- Issue: `parseSlackTimestamp` pads fractional seconds but may lose precision
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/slack.go` (lines 399-429)
- Impact: Beers recorded at same second may have identical ts_rfc, violating UNIQUE constraint
- Current mitigation: UNIQUE on (giver_id, recipient_id, ts) uses raw timestamp, not ts_rfc
- Recommendations:
  - Document that ts_rfc is for queries only; UNIQUE constraint on original ts string
  - Add test for beers given in same second
  - Consider storing original ts as primary key component

## Fragile Areas

**Slack Connection Manager Reconnection Logic:**
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/slack.go` (lines 108-171)
- Why fragile: Exponential backoff restarts entire socket client on each failure; concurrent event processing continues during reconnect
- Safe modification:
  - Coordinate event processing to pause during reconnection
  - Add telemetry to track reconnection frequency
  - Test with intentional Slack disconnections
- Test coverage: No explicit tests for reconnection scenarios

**UsersList Message Parsing Regex:**
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/slack.go` (lines 293-310)
- Why fragile: Regex parsing for @mentions and emoji position is brittle; edge cases like escaped emoji or nested mentions not handled
- Safe modification:
  - Add whitespace handling between mentions and emoji
  - Test with various message formats
  - Consider using proper Slack message parser library
- Test coverage: `slack.go` has debug logging but no unit tests for message parsing

**Daily Limit Check Race Condition:**
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/slack.go` (lines 338-354)
- Why fragile: Check-then-act pattern (count beers, then insert) has race condition if multiple messages arrive simultaneously
- Safe modification:
  - Move limit check into database transaction before insert
  - Use database constraint to enforce limit
  - Test with concurrent message arrivals
- Test coverage: No tests for concurrent gift-giving

## Test Coverage Gaps

**Frontend Components Have No Tests:**
- What's not tested: UsersList, UsersPage, DateRangePicker, API routes all lack unit/integration tests
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/components/`, `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-frontend/project/app/`
- Risk: Rendering bugs, data fetch failures, and UI logic errors go undetected
- Priority: High - UI is user-facing and critical for functionality

**Backend Message Parsing Not Tested:**
- What's not tested: Regex parsing for mentions and emoji, message filtering logic, beer counting
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/slack.go` (lines 293-336)
- Risk: Edge cases in message format could cause beers to fail silently
- Priority: High - core business logic

**Database Migration Not Tested:**
- What's not tested: Table recreation path, aggregation during migration, constraint enforcement
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/store.go` (lines 22-159)
- Risk: Production migration could corrupt data or create constraint violations
- Priority: High - critical for data integrity

**HTTP Handler Authorization Not Tested:**
- What's not tested: Bearer token validation, missing header handling, invalid format rejection
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/http_handlers.go` (lines 172-193)
- Risk: Authorization bypass if middleware logic changed
- Priority: Medium - security-critical but unlikely to change

**Concurrency Edge Cases:**
- What's not tested: Race conditions on concurrent gift-giving, simultaneous Slack reconnections, database transaction collisions
- Files: `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/slack.go`, `/home/tdolfen/Projects/github.com/DanielWeeber/BeerBot-backend/bot/store.go`
- Risk: Data corruption or lost updates under concurrent load
- Priority: Medium - unlikely in small deployments but critical at scale

## Dependencies at Risk

**Slack SDK Version:**
- Package: `github.com/slack-go/slack v0.17.3`
- Risk: Community-maintained library with moderate update frequency
- Impact: Security fixes may lag; socket mode API changes could break reconnection
- Migration plan: Monitor for updates; test socket mode changes thoroughly

**SQLite as Persistent Store:**
- Package: `github.com/mattn/go-sqlite3 v1.14.32`
- Risk: Single file database; no built-in replication or backup
- Impact: Database file corruption loses all data; no HA story
- Migration plan: Consider PostgreSQL for production; implement database backups

**React Datepicker:**
- Package: `react-datepicker ^9.1.0`
- Risk: Minor version bumps could introduce breaking changes
- Impact: Date format or behavior changes could break date range UI
- Migration plan: Pin to specific patch version; test date picker thoroughly on upgrades

## Scaling Limits

**Single SQLite Database:**
- Current capacity: Tested up to thousands of beer records; limited by single-file locking
- Limit: SQLite becomes slow with >10GB database or >100 concurrent writes
- Scaling path:
  - Migrate to PostgreSQL for horizontal scaling
  - Implement database connection pooling
  - Add read replicas for leaderboard queries

**Frontend N+1 Requests:**
- Current capacity: Leaderboard with 100+ users is performant but makes 200+ requests
- Limit: 1000+ users causes noticeable slowdown; mobile users see degraded experience
- Scaling path:
  - Implement batch API endpoints
  - Cache user profile data server-side
  - Use pagination or virtual scrolling

**Hardcoded Top 100 Users:**
- Current capacity: Leaderboard shows top 100 givers and 100 recipients
- Limit: If more than 100 users participate, remainder are invisible
- Scaling path:
  - Add pagination to leaderboard
  - Implement search/filter for specific users
  - Cache top-N users separately for performance

## Missing Critical Features

**No Undo/Correction Mechanism:**
- Problem: Once beers are given, they cannot be retracted or corrected
- Blocks: Users cannot fix accidental beer gifts or typos
- Recommendation: Add admin endpoint to remove/adjust beer counts with audit log

**No Rate Limiting:**
- Problem: No rate limits on API endpoints; backend only has daily per-user limit
- Blocks: DOS attacks could overwhelm backend; no per-IP or per-session limits
- Recommendation: Add rate limiting middleware (e.g., 100 req/min per IP)

**No Audit Log:**
- Problem: No record of who added beers, when, and why (original message not stored)
- Blocks: Cannot investigate disputes or verify authenticity
- Recommendation: Store original Slack message text and metadata with each beer record

**No User Authentication:**
- Problem: API endpoints have single token; no user identity for audit trail
- Blocks: Cannot attribute API changes to specific users
- Recommendation: Integrate with Slack OAuth for user-level authentication

---

*Concerns audit: 2026-01-30*
