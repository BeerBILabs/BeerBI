# Docker Hub Deployment Setup

This repository is configured to automatically build and push Docker images to Docker Hub.

## 🚀 Automatic Deployment

### Triggers

- **Push to main**: Builds and pushes `latest` tag
- **Push tags**: Builds semantic version tags (e.g., `v1.0.0`, `v1.0`, `v1`)
- **Pull requests**: Builds images for testing (no push)
- **Manual trigger**: Via GitHub Actions UI

### Image Repository

- **Docker Hub**: `danielweeber/beerbot-frontend`
- **Registry**: `docker.io`

## ⚙️ Setup Requirements

### 1. Docker Hub Credentials

Add these secrets to your GitHub repository:

1. Go to Repository Settings → Secrets and variables → Actions
2. Add the following secrets:

| Secret Name | Description | Value |
|------------|-------------|--------|
| `DOCKER_USERNAME` | Docker Hub username | `danielweeber` |
| `DOCKER_PASSWORD` | Docker Hub password/token | Your Docker Hub password or access token |

### 2. Docker Hub Access Token (Recommended)

Instead of using your password, create an access token:

1. Log in to [Docker Hub](https://hub.docker.com/)
2. Go to Account Settings → Security → Access Tokens
3. Create a new token with Read/Write permissions
4. Use this token as `DOCKER_PASSWORD` secret

### 3. Environment Variables (Optional)

Configure repository variables for build-time environment:

| Variable Name | Description | Default |
|--------------|-------------|---------|
| `NEXT_PUBLIC_BACKEND_BASE` | Backend API base URL | `http://localhost:8080` |

## 🏗️ Build Configuration

### Multi-Platform Support

Images are built for:

- `linux/amd64` (Intel/AMD)
- `linux/arm64` (Apple Silicon/ARM)

### Tagging Strategy

```text
danielweeber/beerbot-frontend:latest      # Latest main branch
danielweeber/beerbot-frontend:main        # Main branch
danielweeber/beerbot-frontend:sha-main-abc123  # Commit SHA
danielweeber/beerbot-frontend:v1.0.0      # Semantic version
danielweeber/beerbot-frontend:v1.0        # Major.minor
danielweeber/beerbot-frontend:v1          # Major version
```

### Build Context

- **Context**: `.` (root directory)
- **Dockerfile**: `./Dockerfile`
- **Cache**: GitHub Actions cache for faster builds
- **Build Args**: Configurable environment variables

## 🐳 Usage

### Pull Latest Image

```bash
docker pull danielweeber/beerbot-frontend:latest
```

### Run Container

```bash
docker run -d \
  --name beerbot-frontend \
  -p 3000:3000 \
  -e NEXT_PUBLIC_BACKEND_BASE="http://your-backend:8080" \
  danielweeber/beerbot-frontend:latest
```

### With Docker Compose

```yaml
services:
  frontend:
    image: danielweeber/beerbot-frontend:latest
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_BACKEND_BASE=http://backend:8080
    depends_on:
      - backend
    restart: unless-stopped

  backend:
    image: danielweeber/beerbot-backend:latest
    ports:
      - "8080:8080"
    environment:
      - BOT_TOKEN=${BOT_TOKEN}
      - APP_TOKEN=${APP_TOKEN}
      - CHANNEL=${CHANNEL}
      - API_TOKEN=${API_TOKEN}
    volumes:
      - ./data:/data
    restart: unless-stopped
```

### Full Stack Deployment

```bash
# Create network
docker network create beerbot

# Run backend
docker run -d \
  --name beerbot-backend \
  --network beerbot \
  -p 8080:8080 \
  -e BOT_TOKEN="xoxb-your-token" \
  -e APP_TOKEN="xapp-your-token" \
  -e CHANNEL="C1234567890" \
  -e API_TOKEN="your-api-token" \
  danielweeber/beerbot-backend:latest

# Run frontend
docker run -d \
  --name beerbot-frontend \
  --network beerbot \
  -p 3000:3000 \
  -e NEXT_PUBLIC_BACKEND_BASE="http://beerbot-backend:8080" \
  danielweeber/beerbot-frontend:latest
```

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
- ✅ Only pushes on main branch (not PRs)
- ✅ Vulnerability scanning available on Docker Hub
- ✅ Environment variable configuration

### Secrets Management

- Never commit Docker Hub credentials to git
- Use access tokens instead of passwords
- Rotate tokens regularly
- Monitor access logs on Docker Hub
- Keep backend API tokens secure

## 🔗 Integration

### Backend Communication

The frontend automatically connects to the backend via:

- Environment variable: `NEXT_PUBLIC_BACKEND_BASE`
- API proxy routes: `/api/proxy/*`
- Default: `http://localhost:8080` for development

### Production Deployment

For production, ensure:

1. Backend is accessible from frontend container
2. Proper network configuration (Docker networks or external URLs)
3. HTTPS configuration for security
4. Environment variables match your setup
