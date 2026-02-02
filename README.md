![BeerBot Logo](logo.svg)

# BeerBot

A Slack bot for tracking team appreciation through virtual beer giving.

When someone mentions a colleague with a beer emoji in Slack, BeerBot records it and maintains leaderboards showing who's given and received the most recognition.

## Overview

This monorepo contains:

| Component | Description | Tech Stack |
|-----------|-------------|------------|
| [backend/](./backend/) | Slack bot service with REST API | Go, SQLite, Slack Socket Mode |
| [frontend/](./frontend/) | Analytics dashboard | Next.js, React, Tailwind CSS, Bun |

## Development Requirements

### Prerequisites

- **Docker** & **Docker Compose** - For containerized development and deployment
- **Just** - Command runner for common development tasks (recommended)

### Installing Just

Just is a command runner that simplifies common development tasks. Install it using one of these methods:

**macOS:**

```bash
brew install just
```

**Linux:**

```bash
# Using cargo
cargo install just

# Or download binary from GitHub releases
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin
```

**Windows:**

```bash
# Using cargo
cargo install just

# Or using scoop
scoop install just
```

For other installation methods, see the [Just documentation](https://github.com/casey/just#installation).

### Development Workflow

This project includes a `justfile` with common commands:

```bash
# View all available commands
just

# Start development environment
just dev

# View logs
just dev-logs

# Stop development environment
just dev-down

# Run tests
just test

# Clean up all containers and volumes
just clean
```

See the `justfile` for a complete list of available commands.

## Quick Start

### Using Docker Hub Images

See [compose.yaml](compose.yaml).

```bash
docker compose up -d
```

Open <http://localhost:3000> to view the dashboard.

### Slack Setup

1. Create a Slack app at <https://api.slack.com/apps>
2. Enable **Socket Mode**
3. Add Bot Token Scopes: `channels:history`, `groups:history`, `im:history`, `mpim:history`, `users:read`, `chat:write`
4. Generate an App-Level Token with `connections:write` scope
5. Install the app to your workspace
6. Invite the bot to channels where you want to track beers

## How It Works

```txt
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
