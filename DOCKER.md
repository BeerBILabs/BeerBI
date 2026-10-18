# Docker Deployment Guide

This guide covers deploying BeerBot using Docker and Docker Hub images.

## 🚀 Quick Start

### Using Docker Compose

Please check the [justfile](justfile) for the latest deployment instructions.

## 📦 Docker Hub Images

### Backend

- **Repository**: `beerbilabs/beerbi-backend`
- **Registry**: `docker.io`

### Frontend

- **Repository**: `beerbilabs/beerbi-frontend`
- **Registry**: `docker.io`

## ⚙️ Setup Requirements

### Docker Hub Credentials (For CI/CD)

Add these secrets to your GitHub repository:

1. Go to Repository Settings → Secrets and variables → Actions
2. Add the following secrets:

| Secret Name       | Description               |
|-------------------|---------------------------|
| `DOCKER_USERNAME` | Docker Hub username       |
| `DOCKER_PASSWORD` | Docker Hub token          |

### Creating Access Token

Instead of using your password, create an access token:

1. Log in to [Docker Hub](https://hub.docker.com/)
2. Go to Account Settings → Security → Access Tokens
3. Create a new token with Read/Write permissions
4. Use this token as `DOCKER_PASSWORD` secret

## 🏗️ Build Configuration

### Multi-Platform Support

Images are built for:

- `linux/amd64` (Intel/AMD)
- `linux/arm64` (Apple Silicon/ARM)

The backend is pure Go (`CGO_ENABLED=0`, SQLite via `modernc.org/sqlite`): its builder stage runs on the build platform and cross-compiles for each target, so the Go build needs neither QEMU nor a C toolchain. Only the small runtime stage (`apk add` of ca-certificates, wget, curl) still runs under QEMU for the non-native platform.

### Database files

The backend runs SQLite in WAL mode. Next to `/data/bot.db` you will see `bot.db-wal` and `bot.db-shm`; they belong to the database. Back up the whole `/data` volume with the container stopped, or take an online copy with the `sqlite3` CLI (not included in the image), e.g. `sqlite3 bot.db ".backup backup.db"` in the volume directory. Copying only `bot.db` while the bot runs can miss the latest commits.

### Tagging Strategy

**Backend:**

```bash
beerbilabs/beerbi-backend:v1.0.0
```

**Frontend:**

```bash
beerbilabs/beerbi-frontend:v1.0.0
```

The project uses semantic versioning.

## 🐳 Usage

### Backend

**Pull and run:**

See [justfile](justfile) for the latest deployment instructions.

**Environment Variables:**

| Variable      | Required | Default  | Description                    |
|---------------|----------|----------|--------------------------------|
| `BOT_TOKEN`   | ✅       | -        | Slack Bot User OAuth Token     |
| `APP_TOKEN`   | ✅       | -        | Slack App-Level Token          |
| `API_TOKEN`   | ✅       | -        | Bearer token for REST API      |
| `CHANNEL`     | ❌       | -        | Specific channel ID to monitor |
| `EMOJI`       | ❌       | `:beer:` | Emoji to track                 |
| `MAX_PER_DAY` | ❌       | `10`     | Maximum beers per user per day |
| `PROCESSED_EVENTS_RETENTION_DAYS` | ❌ | `30` | Days to keep Slack event deduplication records (`processed_events`); whole days from 1 to 3650, pruned daily |

### Frontend

See [justfile](justfile) for the latest deployment instructions.

**Environment Variables:**

| Variable                   | Required | Default                 | Description                    |
|----------------------------|----------|-------------------------|--------------------------------|
| `NEXT_PUBLIC_BACKEND_BASE` | ✅       | `http://localhost:8080` | Backend API base URL           |
| `NEXT_PUBLIC_API_TOKEN`    | ✅       | -                       | API authentication token       |

### Full Stack Deployment

See [justfile](justfile) for the latest deployment instructions.

## 🔄 Automatic Deployment

### Triggers

- **Push tags**: Builds semantic version tags (e.g., `v1.0.0`, `v1.0`, `v1`)

## 🔍 Monitoring

### GitHub Actions

- View build status in the Actions tab
- Check build summaries for deployment details
- Monitor build times and cache effectiveness

### Docker Hub

- View image layers and vulnerabilities
- Check pull statistics
- Manage image tags and retention

## 🛡️ Security

### Best Practices

- ✅ Uses official Docker Hub registry
- ✅ Multi-platform builds for compatibility
- ✅ Secure credential management via GitHub Secrets
- ✅ Build cache optimization
- ✅ Only pushes on tags
- ✅ Vulnerability scanning available on Docker Hub

### Secrets Management

- Never commit Docker Hub credentials to git
- Use access tokens instead of passwords
- Rotate tokens regularly
- Monitor access logs on Docker Hub
- Keep backend API tokens secure
