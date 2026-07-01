SHELL := /bin/bash
COMPOSE := docker compose -f deploy/docker-compose.yml --env-file deploy/.env

TEST_DB  ?= postgres://hk:hk_dev_password@localhost:55432/hkgroup?sslmode=disable
TEST_RDB ?= redis://localhost:56379/0

.PHONY: help up down logs sqlc migrate test test-invariants vet fmt seed test-infra test-infra-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

up: ## Start the whole stack (postgres, redis, nats, api, web) + run migrations
	cp -n deploy/.env.example deploy/.env || true
	$(COMPOSE) up --build

down: ## Stop the stack
	$(COMPOSE) down

logs: ## Tail stack logs
	$(COMPOSE) logs -f

sqlc: ## Regenerate sqlc code from db/queries
	sqlc generate

migrate: ## Apply migrations to TEST_DB
	cd backend && DATABASE_URL="$(TEST_DB)" MIGRATIONS_DIR=$(PWD)/db/migrations go run ./cmd/migrate up

vet: ## go vet
	cd backend && go vet ./...

fmt: ## gofmt
	cd backend && gofmt -w .

test: ## Run all backend tests (needs test-infra)
	cd backend && TEST_DATABASE_URL="$(TEST_DB)" TEST_REDIS_URL="$(TEST_RDB)" go test ./... -count=1

test-invariants: ## Run only the 8 invariant tests (needs test-infra)
	cd backend && TEST_DATABASE_URL="$(TEST_DB)" TEST_REDIS_URL="$(TEST_RDB)" \
		go test ./internal/service/... -run 'Invariant|HappyPath|Idempotent' -count=1 -v

test-infra: ## Start throwaway postgres+redis+nats for tests, then migrate
	docker run -d --name hk-pg    -e POSTGRES_USER=hk -e POSTGRES_PASSWORD=hk_dev_password -e POSTGRES_DB=hkgroup -p 55432:5432 postgres:16-alpine
	docker run -d --name hk-redis -p 56379:6379 redis:7-alpine
	docker run -d --name hk-nats  -p 54222:4222 nats:2.10-alpine -js
	sleep 4
	$(MAKE) migrate

test-infra-down: ## Remove throwaway test infra
	-docker rm -f hk-pg hk-redis hk-nats
