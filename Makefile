SHELL := /bin/bash
COMPOSE := docker compose

.PHONY: up down logs test test-db test-all lint fmt web-check hooks

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

reset:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f --tail=100

## Fast unit tests (no database).
test:
	cd backend && go test ./...

## Integration tests: starts the database and points the tests at it.
test-db:
	$(COMPOSE) up -d db
	cd backend && TEST_DATABASE_URL='postgres://sportsbook:sportsbook@127.0.0.1:5437/sportsbook?sslmode=disable' go test ./... -count=1

test-all: test-db
	cd frontend && npm run test

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint && npm run typecheck

fmt:
	cd backend && gofmt -w .

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-push
	@echo "pre-push hook enabled: backend + frontend tests"

web-check:
	cd frontend && npm run build

## Local frontend dev (API expected on :8080).
web-dev:
	cd frontend && npm run dev
