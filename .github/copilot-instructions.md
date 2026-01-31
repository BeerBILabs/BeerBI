<!-- Short, actionable Copilot instructions for the BeerBI monorepo -->
# Copilot / AI Assistant Notes — BeerBI

Purpose
-------
Provide immediate, practical context so an AI coding assistant can be productive in this monorepo (quick setup, important files, conventions, and common debugging patterns).

Big picture
-----------
- Monorepo with two main components:
  - `backend/` — Go Slack bot service (Socket Mode) that records "beer" events into a SQLite DB. Key folder: `backend/bot/`.
  - `frontend/` — Next.js (App Router) dashboard in `frontend/project/`, built with Bun/Node and Tailwind.
- Persistent data: SQLite file used by the bot (default path: `bot/data/data/bot.db`).
- Communication: Frontend calls backend REST APIs; backend listens to Slack via Socket Mode and writes to SQLite.

Quick dev commands (examples)
-----------------------------
- Start backend (Docker dev variant, source-mounted):
  - `cd bot && docker-compose -f docker-compose.dev.yml up --build`
- Start backend from source (host Go):
  - `cd bot && go run . -addr=:8080 -db=/tmp/beerbot.db`
- Backend tests:
  - `cd bot && go test ./...`
- Start frontend (recommended, inside `frontend/project`):
  - `cd frontend/project && bun install && bun run dev` (or `npm install && npm run dev`)
- Run full stack via root docker compose (images):
  - `docker compose up -d`

Key files to inspect first
------------------------
- `backend/bot/main.go` — Slack event entry points and HTTP server initialization.
- `backend/bot/store.go` — DB migrations, queries, and dedupe logic (look for `processed_events` and `INSERT OR IGNORE`).
- `backend/bot/store_test.go`, `backend/bot/*_test.go` — existing unit tests and examples of expected behavior.
- `frontend/project/app/api/proxy/[...path]/route.ts` — client proxy to backend; shows how frontend injects `NEXT_PUBLIC_API_TOKEN`.
- Compose files: `bot/docker-compose*.yml`, `frontend/docker-compose*.yml`, and root `docker-compose.yml` for multi-service examples.

Project-specific conventions & important patterns
-----------------------------------------------
- Event deduplication is essential: backend uses a `processed_events` table and performs `INSERT OR IGNORE` to avoid duplicate processing of Slack events. When changing event handling, preserve atomic marking-before-processing semantics.
- Slack event filtering: code filters on `ev.SubType`, `ev.BotID`, and a configured `CHANNEL`. When modifying event parsing, check `main.go` for existing filters to avoid double-counting.
- DB migrations: `store.go` performs migrations on startup. Changes to schema should be accompanied by migration logic and tests.
- Emoji detection / mention parsing is implemented in the backend; adding support for additional emoji or formats requires touching the detection helpers and tests.

Integration points & external dependencies
----------------------------------------
- Slack Socket Mode: requires `BOT_TOKEN` and `APP_TOKEN`. App-level token must have `connections:write`.
- REST API protected by `API_TOKEN` (Bearer). Frontend uses `NEXT_PUBLIC_API_TOKEN` to proxy requests.
- Data is stored in SQLite — path is configured via env (see `DB_PATH`/`DB_PATH`-like env in `backend/README.md`). Use `bot/data/data/bot.db` in dev compose.

What to check when changing event handling
-----------------------------------------
1. Confirm Slack App scopes and enabled events (see `backend/README.md` and `bot/docker-compose.override.yml` examples).
2. Ensure new event paths mark the event as processed BEFORE doing side-effects (use `processed_events` and an atomic `INSERT OR IGNORE`).
3. Add unit tests in `backend/bot/*_test.go` that simulate duplicate deliveries and assert only one DB row is created.

Examples (use these exact files/commands)
---------------------------------------
- Inspect dedupe code: `backend/bot/store.go`
- Run backend tests in a container (if host lacks Go):
  - `docker run --rm -v $(pwd)/bot:/src -w /src golang:1.23-alpine sh -c "apk add --no-cache git build-base && go test ./..."`
- Reset local DB and restart bot (safe sequence):
  - `mv bot/data/data/bot.db bot/data/data/bot.db.bak`
  - `cd bot && docker-compose -f docker-compose.yml up -d --build`

Notes for AI agents
-------------------
- Prefer small, focused PRs: modify event handling in one change, add tests, and update migrations if required.
- Preserve existing behavior where tests expect it — run `cd bot && go test ./...` before proposing behavior changes.
- When suggesting fixes for duplicate processing, point to `processed_events`/`store.go` and `main.go` filters rather than broad strokes.

If something is unclear
-----------------------
- Ask which component to focus on (backend event logic vs frontend UI). Provide the file path and a short reproduction (commands to run). I'll refine the instructions.

References
----------
- `backend/copilot-instructions.md` — component-specific, German-language guidance (contains debug recipes and DB maintenance commands).
- `backend/README.md`, `frontend/README.md`, root `README.md` — setup, env vars, and architecture notes.
