# BeerBot - Common Commands

# Default recipe to display help
default:
    @just --list

# Development Environment
# =======================

# Start development environment (with hot reload)
dev:
    docker compose -f compose.dev.yaml up --build -d

# Stop development environment
dev-down:
    docker compose -f compose.dev.yaml down

# Restart development environment
dev-restart:
    docker compose -f compose.dev.yaml restart

# View development logs
dev-logs:
    docker compose -f compose.dev.yaml logs -f

# View backend development logs
dev-logs-backend:
    docker compose -f compose.dev.yaml logs -f backend-dev

# View frontend development logs
dev-logs-frontend:
    docker compose -f compose.dev.yaml logs -f frontend-dev

# Production Environment
# ======================

# Start production environment
prod:
    docker compose -f compose.yaml up --build -d

# Stop production environment
prod-down:
    docker compose -f compose.yaml down

# Restart production environment
prod-restart:
    docker compose -f compose.yaml restart

# View production logs
prod-logs:
    docker compose -f compose.yaml logs -f

# View backend production logs
prod-logs-backend:
    docker compose -f compose.yaml logs -f backend

# View frontend production logs
prod-logs-frontend:
    docker compose -f compose.yaml logs -f frontend

# Testing
# =======

# Run all tests: backend (gofmt, vet, race tests) then frontend (lint, tsc, vitest, build)
test: test-backend test-frontend

# Run backend checks only
test-backend:
    docker compose -f compose.test.yaml run --build --rm backend-test

# Run frontend checks only
test-frontend:
    docker compose -f compose.test.yaml run --build --rm frontend-test

# Run tests and remove containers and networks
test-clean: test
    docker compose -f compose.test.yaml down --remove-orphans

# Run backend query benchmarks against a seeded database (size via BEERBI_BENCH_ROWS)
bench rows="1000000":
    docker run --rm -e BEERBI_BENCH_ROWS={{rows}} -v "$PWD/backend/bot":/src -w /src golang:1.27-alpine sh -c 'apk add -q build-base && go test -run "^$" -bench . -benchmem ./...'

# Development data
# ================

# Fill backend/bot/data/bot.db with deterministic synthetic beers (fake user IDs)
seed rows="200000":
    docker run --rm -v "$PWD/backend/bot":/src -w /src golang:1.27-alpine sh -c 'apk add -q build-base && go run ./cmd/seed -db data/bot.db -rows {{rows}}'

# Cleanup
# =======

# Stop all environments
stop-all:
    docker compose -f compose.dev.yaml down
    docker compose -f compose.yaml down
    docker compose -f compose.test.yaml down

# Remove all containers, networks, and volumes
clean:
    docker compose -f compose.dev.yaml down -v
    docker compose -f compose.yaml down -v
    docker compose -f compose.test.yaml down -v

# Prune Docker system (removes unused containers, networks, images)
prune:
    docker system prune -f

# Deep clean (removes everything including volumes)
deep-clean:
    docker compose -f compose.dev.yaml down -v
    docker compose -f compose.yaml down -v
    docker compose -f compose.test.yaml down -v
    docker system prune -af --volumes

# Utilities
# =========

# Show status of all services
status:
    @echo "=== Development ==="
    @docker compose -f compose.dev.yaml ps
    @echo "\n=== Production ==="
    @docker compose -f compose.yaml ps

# Rebuild all images without cache
rebuild-dev:
    docker compose -f compose.dev.yaml build --no-cache

# Rebuild production images without cache
rebuild-prod:
    docker compose -f compose.yaml build --no-cache

# Execute shell in backend development container
shell-backend:
    docker compose -f compose.dev.yaml exec backend-dev sh

# Execute shell in frontend development container
shell-frontend:
    docker compose -f compose.dev.yaml exec frontend-dev sh
