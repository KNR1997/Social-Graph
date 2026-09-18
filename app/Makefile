# Local development targets. `make help` lists them.
#
# .env is optional: it is loaded when present so local runs pick up
# DATABASE_URL, but nothing here requires it to exist.
ifneq (,$(wildcard .env))
include .env
export
endif

GO           ?= go
BIN_DIR      := bin
INTEGRATION  := -tags integration

.DEFAULT_GOAL := help

.PHONY: help
help: ## List the available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## --- environment ------------------------------------------------------------

.PHONY: up
up: ## Start Postgres 18 and wait for it to be ready
	docker compose up -d --wait database

.PHONY: down
down: ## Stop the database, keeping its volume
	docker compose down

.PHONY: clean-db
clean-db: ## Stop the database and delete its volume
	docker compose down --volumes

.PHONY: logs
logs: ## Tail the database logs
	docker compose logs -f database

## --- run and build ----------------------------------------------------------

.PHONY: run
run: ## Run the API against the local database
	$(GO) run ./cmd/api

.PHONY: build
build: ## Build both binaries into ./bin
	$(GO) build -trimpath -o $(BIN_DIR)/api ./cmd/api
	$(GO) build -trimpath -o $(BIN_DIR)/migrate ./cmd/migrate

.PHONY: dev
dev: up migrate-up run ## Start the database, migrate, then run the API

## --- test -------------------------------------------------------------------

.PHONY: test
test: ## Run the fast tests (no Docker required)
	$(GO) test -race ./...

.PHONY: test-integration
test-integration: up ## Run the integration tests against a real Postgres
	$(GO) test $(INTEGRATION) -race -count=1 ./...

.PHONY: test-all
test-all: test test-integration ## Run every test

.PHONY: cover
cover: ## Fast tests with a coverage report at coverage.html
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "wrote coverage.html"

.PHONY: test-one
test-one: ## Run a single test: make test-one PKG=./internal/application RUN=TestDeposit
	$(GO) test -race -run '$(RUN)' -v $(PKG)

## --- database ---------------------------------------------------------------

.PHONY: migrate-up
migrate-up: ## Apply all pending migrations
	$(GO) run ./cmd/migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration
	$(GO) run ./cmd/migrate down

.PHONY: migrate-status
migrate-status: ## Show which migrations have been applied
	$(GO) run ./cmd/migrate status

.PHONY: migrate-create
migrate-create: ## Create a migration: make migrate-create NAME=add_widgets
	@test -n "$(NAME)" || (echo "NAME is required, e.g. make migrate-create NAME=add_widgets"; exit 1)
	$(GO) run ./cmd/migrate create $(NAME) sql

## --- codegen ----------------------------------------------------------------

.PHONY: sqlc
sqlc: ## Regenerate the type-safe query layer from db/queries
	sqlc generate

.PHONY: sqlc-vet
sqlc-vet: ## Check the queries against the schema without generating
	sqlc vet

.PHONY: wire
wire: ## Regenerate the dependency-injection graph
	wire ./...

.PHONY: generate
generate: sqlc wire ## Run every generator

## --- quality ----------------------------------------------------------------

.PHONY: fmt
fmt: ## Format and tidy imports
	golangci-lint fmt ./...

.PHONY: lint
lint: ## Run the linters, including on the integration-tagged files
	golangci-lint run ./...
	golangci-lint run --build-tags integration ./...

.PHONY: tidy
tidy: ## Tidy go.mod
	$(GO) mod tidy

.PHONY: check
check: fmt lint test ## What CI runs on every push

.PHONY: tools
tools: ## Install the pinned developer tools
	$(GO) install github.com/google/wire/cmd/wire@latest
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo "golangci-lint: https://golangci-lint.run/welcome/install/"
