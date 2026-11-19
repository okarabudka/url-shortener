# Makefile — local Docker workflow. Run `make help` for the list of targets.
# Only Docker and make are required; Go tooling runs in containers.

SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE := docker compose
GO_IMAGE := golang:1.27
LINT_IMAGE := golangci/golangci-lint:v2.15.0
# Named volumes keep the module and build caches between runs.
DOCKER_GO := docker run --rm -v "$(CURDIR)":/src -w /src \
	-v url-shortener-gomod:/go/pkg/mod -v url-shortener-gocache:/root/.cache

.PHONY: help env up down logs smoke test lint

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-6s\033[0m %s\n", $$1, $$2}'

env: ## Create .env from .env.example (skipped if present)
	@test -f .env || cp .env.example .env

up: env ## Build and start the service
	$(COMPOSE) up -d --build

down: ## Stop the service (the database volume is kept)
	$(COMPOSE) down

logs: ## Follow the service logs
	$(COMPOSE) logs -f

smoke: ## Create, follow and delete a link against the running service
	@set -a && source .env && set +a && sh scripts/smoke.sh

test: ## Run the tests with the race detector and coverage
	$(DOCKER_GO) $(GO_IMAGE) go test -race -cover ./...

lint: ## Run go vet and golangci-lint
	$(DOCKER_GO) $(GO_IMAGE) go vet ./...
	$(DOCKER_GO) $(LINT_IMAGE) golangci-lint run
