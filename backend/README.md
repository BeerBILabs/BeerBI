# BeerBot Backend

Go-based Slack bot service with REST API for tracking beer emoji interactions.

## Tech Stack

- **Language**: Go 1.21+
- **Database**: SQLite
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
- `GET /api/givers`
- `GET /api/recipients`

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
