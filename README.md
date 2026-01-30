# BeerBot

A Slack bot for tracking team appreciation through virtual beer giving.

When someone mentions a colleague with a beer emoji in Slack, BeerBot records it and maintains leaderboards showing who's given and received the most recognition.

## Overview

This monorepo contains:

| Component | Description | Tech Stack |
|-----------|-------------|------------|
| [backend/](./backend/) | Slack bot service with REST API | Go, SQLite, Slack Socket Mode |
| [frontend/](./frontend/) | Analytics dashboard | Next.js, React, Tailwind CSS, Bun |

## Quick Start

### Using Docker Hub Images

```yaml
# docker-compose.yml
services:
  backend:
    image: danielweeber/beerbot-backend:latest
    ports:
      - "8080:8080"
    environment:
      BOT_TOKEN: "xoxb-your-bot-token"
      APP_TOKEN: "xapp-your-app-token"
      API_TOKEN: "your-secure-api-token"
      CHANNEL: "C1234567890"
      EMOJI: ":beer:"
    volumes:
      - beerbot-data:/data

  frontend:
    image: danielweeber/beerbot-frontend:latest
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_BACKEND_BASE: "http://backend:8080"
      NEXT_PUBLIC_API_TOKEN: "your-secure-api-token"
    depends_on:
      - backend

volumes:
  beerbot-data:
```

```bash
docker compose up -d
```

Open http://localhost:3000 to view the dashboard.

### Slack Setup

1. Create a Slack app at https://api.slack.com/apps
2. Enable **Socket Mode**
3. Add Bot Token Scopes: `channels:history`, `groups:history`, `im:history`, `mpim:history`, `users:read`, `chat:write`
4. Generate an App-Level Token with `connections:write` scope
5. Install the app to your workspace
6. Invite the bot to channels where you want to track beers

## How It Works

```
@sarah great work on the release! :beer:
```

The bot detects beer emojis and tracks:
- Who gave the beer (message author)
- Who received it (mentioned users)
- Timestamp for date range queries

The frontend displays:
- Leaderboards for top givers and receivers
- Date filtering (presets + custom ranges)
- User profiles with Slack avatars

## Documentation

- **Backend**: See [backend/README.md](./backend/README.md) for API documentation, configuration options, and deployment details
- **Frontend**: See [frontend/README.md](./frontend/README.md) for UI features, customization, and build instructions

## License

MIT
