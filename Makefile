.PHONY: help up down logs db-migrate build run-auth run-gateway run-risk run-policy run-mcp test clean stop-local stop-all

COMPOSE := docker-compose -f infra/docker/docker-compose.yml
ENV_FILE := .env

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Infrastructure ---

up: ## Start all infrastructure (Postgres, Redis) and services
	$(COMPOSE) up -d

up-infra: ## Start only Postgres and Redis
	$(COMPOSE) up -d postgres redis

down: ## Stop all services
	$(COMPOSE) down

down-clean: ## Stop all services and remove volumes
	$(COMPOSE) down -v

stop-local: ## Kill locally running Go/Python/Node services on ports 8080-8084
	@for port in 8080 8081 8082 8083 8084; do \
		pid=$$(lsof -ti :$$port 2>/dev/null); \
		if [ -n "$$pid" ]; then \
			kill -9 $$pid && echo "Killed process on :$$port (PID $$pid)"; \
		fi; \
	done

stop-all: down stop-local ## Stop Docker containers and local processes

logs: ## Tail logs for all services
	$(COMPOSE) logs -f

logs-auth: ## Tail auth-service logs
	$(COMPOSE) logs -f auth-service

logs-gateway: ## Tail api-gateway logs
	$(COMPOSE) logs -f api-gateway

# --- Database ---

db-migrate: ## Run database migrations
	@echo "Migrations are auto-applied via docker-entrypoint-initdb.d"
	@echo "For manual migration, connect to Postgres and run the SQL files in infra/migrations/"

db-shell: ## Open psql shell
	docker exec -it authvora-postgres psql -U authvora -d authvora

db-reset: ## Reset database (drop and recreate)
	$(COMPOSE) down -v postgres
	$(COMPOSE) up -d postgres

# --- Development (local, outside Docker) ---

run-auth: ## Run auth-service locally
	cd services/auth-service && go run ./cmd/server

run-gateway: ## Run api-gateway locally
	cd services/api-gateway && go run ./cmd/server

run-risk: ## Run risk-engine locally
	cd services/risk-engine && python -m uvicorn app.main:app --host 0.0.0.0 --port 8082 --reload

run-policy: ## Run policy-engine locally
	cd services/policy-engine && go run ./cmd/server

run-mcp: ## Run mcp-server locally
	cd services/mcp-server && npm run dev

# --- Build ---

build: ## Build all service Docker images
	$(COMPOSE) build

build-auth: ## Build auth-service image
	$(COMPOSE) build auth-service

build-gateway: ## Build api-gateway image
	$(COMPOSE) build api-gateway

build-risk: ## Build risk-engine image
	$(COMPOSE) build risk-engine

# --- Test ---

test: test-auth test-risk ## Run all tests

test-auth: ## Run auth-service tests
	cd services/auth-service && go test ./...

test-risk: ## Run risk-engine tests
	cd services/risk-engine && python -m pytest

# --- Clean ---

clean: ## Remove build artifacts
	rm -rf services/auth-service/tmp
	rm -rf services/api-gateway/tmp
	rm -rf services/risk-engine/__pycache__
	rm -rf services/mcp-server/dist
	rm -rf sdk/node-sdk/dist
