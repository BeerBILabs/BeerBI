.PHONY: build test run-dev logs

# Thin wrappers around the repo-level files; the justfile in the repo root is the main entry point.

build:
	docker build -f ../../docker/backend/Dockerfile -t beerbi-backend:local .

test:
	docker compose -f ../../compose.test.yaml run --build --rm backend-test

run-dev:
	docker compose -f ../../compose.dev.yaml up --build backend-dev cache

logs:
	docker compose -f ../../compose.dev.yaml logs -f backend-dev
