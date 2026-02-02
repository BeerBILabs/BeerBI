# BeerBot Frontend

Next.js dashboard for BeerBot analytics and team appreciation insights.

## Tech Stack

- **Framework**: Next.js 15+ with App Router
- **Language**: TypeScript 5.9+
- **Styling**: Tailwind CSS 4.x
- **Runtime**: Bun (recommended) or Node.js 18+
- **Deployment**: Docker

## Quick Start

See [root README](../README.md) for full setup instructions.

## Environment Variables

| Variable                   | Required | Default                 | Description              |
|----------------------------|----------|-------------------------|--------------------------|
| `NEXT_PUBLIC_BACKEND_BASE` | ✅       | `http://localhost:8080` | Backend API base URL     |
| `NEXT_PUBLIC_API_TOKEN`    | ✅       | -                       | API authentication token |
