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

All endpoints require Bearer token authentication.

### Beer Statistics

- `GET /api/given?user={user_id}&start={date}&end={date}`
- `GET /api/received?user={user_id}&start={date}&end={date}`

### User Management

- `GET /api/user?user={user_id}`

### Analytics

- `GET /api/stats/combined?start={date}&end={date}` (timeline, top users, heatmap, pairs and quarterly series in one response)
- `GET /api/stats/timeline`, `/api/stats/quarterly`, `/api/stats/top`, `/api/stats/heatmap`, `/api/stats/pairs`

Responses carry `ETag` and `Cache-Control` (1 hour for ranges that ended before today, 30 s otherwise; the combined response counts as current while its quarterly series includes the current year).

### Health

- `GET /api/health`

## Environment Variables

| Variable      | Required | Default  | Description                    |
|---------------|----------|----------|--------------------------------|
| `BOT_TOKEN`   | ✅       | -        | Slack Bot User OAuth Token     |
| `APP_TOKEN`   | ✅       | -        | Slack App-Level Token          |
| `API_TOKEN`   | ✅       | -        | Bearer token for REST API      |
| `CHANNEL`     | ❌       | -        | Specific channel ID to monitor |
| `EMOJI`       | ❌       | `:beer:` | Emoji to track                 |
| `MAX_PER_DAY` | ❌       | `10`     | Maximum beers per user per day |
| `PROCESSED_EVENTS_RETENTION_DAYS` | ❌ | `30` | Days to keep Slack event deduplication records (`processed_events`); whole days from 1 to 3650, pruned daily |
| `EVENT_WORKERS` | ❌     | `4`      | Parallel workers for beer messages (`0` = inline) |
