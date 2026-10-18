# BeerBot Backend

Go-based Slack bot service with REST API for tracking beer emoji interactions.

## Tech Stack

- **Language**: Go 1.27+
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGO), WAL mode
- **Integration**: Slack Socket Mode
- **Deployment**: Docker

## Quick Start

See [root README](../README.md) for full setup instructions.

## API Endpoints

All `/api/*` endpoints except `/api/health` require `Authorization: Bearer <API_TOKEN>`. Without a configured `API_TOKEN` they all answer 401.

### Beer Statistics

- `GET /api/given?user={user_id}&start={date}&end={date}`
- `GET /api/received?user={user_id}&start={date}&end={date}`

### Users

- `GET /api/user?user={user_id}` → `{"real_name": "...", "profile_image": "..."}` (404 if unknown; 503 with `Retry-After` while the Slack lookup is still running or failed)
- `GET /api/users?ids={id1},{id2}` (max. 100) → `{"<id>": {"real_name": "...", "profile_image": "..."}}`

Names are resolved Redis → SQLite `user_cache` → Slack `users.info` (10 parallel workers) and preloaded from `users.list` at start and daily (scope `users:read`).

### Analytics

- `GET /api/stats/combined?start={date}&end={date}[&granularity=&limit=&pairs_limit=&start_year=&end_year=]` (timeline, top users, heatmap, pairs and quarterly series in one response; the queries run concurrently)
- `GET /api/stats/timeline?start=&end=&granularity=day|week|month` → `[{"date": "...", "count": N}]` (one series since 2.0; the former `given`/`received` fields were always identical)
- `GET /api/stats/quarterly`, `/api/stats/top`, `/api/stats/heatmap`, `/api/stats/pairs`
- Top-user entries carry `real_name` and `profile_image`, pairs carry `giver_name`, `giver_image`, `recipient_name`, `recipient_image`. The fields are omitted when a name is unknown.

Responses carry `ETag` and `Cache-Control` (1 hour for ranges that ended before today, 30 s otherwise; the combined response counts as current while its quarterly series includes the current year).

### Removed in 2.0

`GET /api/givers` and `GET /api/recipients` (unauthenticated, unbounded lists) are gone and answer 404; use `/api/stats/top`.

### Health and metrics (no auth)

- `GET /api/health`, `GET /healthz`
- `GET /metrics` (Prometheus)

## Environment Variables

| Variable      | Required | Default  | Description                    |
|---------------|----------|----------|--------------------------------|
| `BOT_TOKEN`   | ✅       | -        | Slack Bot User OAuth Token     |
| `APP_TOKEN`   | ✅       | -        | Slack App-Level Token          |
| `CHANNEL`     | ✅       | -        | Channel ID to monitor (the bot exits at startup without it) |
| `API_TOKEN`   | ✅       | -        | Bearer token for REST API      |
| `DB_PATH`     | ❌       | `/data/bot.db` (image) | SQLite database path |
| `ADDR`        | ❌       | `:8080`  | HTTP listen address            |
| `REDIS_ADDR`  | ❌       | `cache:6379` | Redis address; without Redis the bot runs on SQLite alone |
| `EMOJI`       | ❌       | `:beer:` | Emoji to track                 |
| `MAX_PER_DAY` | ❌       | `10`     | Maximum beers per user per day |
| `LOG_LEVEL`   | ❌       | `info`   | zerolog level (`debug`, `info`, `warn`, `error`) |
| `PROCESSED_EVENTS_RETENTION_DAYS` | ❌ | `30` | Days to keep Slack event deduplication records (`processed_events`); whole days from 1 to 3650, pruned daily |
| `EVENT_WORKERS` | ❌     | `4`      | Parallel workers for beer messages (`0` = inline) |

## Tests

The SQLite driver is pure Go, so no C toolchain is needed:

```bash
cd backend/bot
CGO_ENABLED=0 go test ./...
go test -race -cover ./...          # on Linux the race detector needs CGO (gcc)
go test -run '^$' -bench . ./...    # query benchmarks on a seeded DB (BEERBI_BENCH_ROWS, default 1,000,000)
```

`just test-backend` runs gofmt, `go vet` and the race tests in Docker.

## Database and upgrades

SQLite runs in WAL mode: `bot.db-wal` and `bot.db-shm` next to `bot.db` belong to the database. Back up all three with the bot stopped (or use `sqlite3 bot.db ".backup copy.db"`), and always before an upgrade: migrations run at startup (2.0 rewrites `beers.ts_rfc` to RFC3339 UTC once). The bot needs up to 15 s to shut down cleanly, so give the container `stop_grace_period: 20s`. See [DOCKER.md](../DOCKER.md).
