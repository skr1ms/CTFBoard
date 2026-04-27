.PHONY: help install dev build preview type-check lint lint-fix format codegen e2e e2e-down e2e-spec e2e-fast e2e-ui e2e-local e2e-ui-local test test-watch test-coverage check clean docker-build

# HELP ======================================================================
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# SETUP =====================================================================

##@ Setup

install: ## Install dependencies
	bun install

install-browsers: ## Install Playwright browsers (run separately if needed)
	bun x playwright install chromium

# DEV =======================================================================

##@ Development

dev: ## Start Vite dev server (hot reload)
	bun run dev

preview: ## Preview production build locally
	bun run preview

# BUILD =====================================================================

##@ Build

build: type-check ## Build production bundle (type-check first)
	bun run build

docker-build: ## Build Docker image for local testing
	docker build \
		--build-arg VITE_APP_NAME=$${VITE_APP_NAME:-CTF Platform} \
		--build-arg VITE_API_BASE_URL=$${VITE_API_BASE_URL:-http://localhost:8090/api/v1} \
		--build-arg VITE_WS_URL=$${VITE_WS_URL:-ws://localhost:8090/api/v1/ws} \
		--build-arg VITE_SSE_URL=$${VITE_SSE_URL:-http://localhost:8090/api/v1/sse} \
		-t ctf-platform-frontend:dev .

# CODE GENERATION ===========================================================

##@ Code Generation

codegen: ## Regenerate TypeScript API types from OpenAPI spec (requires backend running or spec file)
	bun run codegen

# QUALITY ===================================================================

##@ Quality

type-check: ## Run TypeScript compiler check (no emit)
	bun run type-check

lint: ## Run ESLint
	bun run lint

lint-fix: ## Run ESLint with auto-fix
	bun x eslint src --fix

format: ## Format source with Prettier
	bun run format

check: type-check lint ## Run type-check + lint (CI gate)

# TESTS =====================================================================

##@ Tests (unit)

test: ## Run unit tests (vitest)
	bun run test

test-watch: ## Run unit tests in watch mode
	bun run test:watch

test-coverage: ## Run unit tests with coverage report
	bun run test:coverage

##@ Tests (E2E)

CHROMIUM ?= $(shell which chromium 2>/dev/null || which google-chrome 2>/dev/null)
PW_ENV    = PLAYWRIGHT_BROWSERS_PATH=$(HOME)/.cache/ms-playwright PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=$(CHROMIUM) VITE_BACKEND_PORT=8090

e2e: ## Full e2e run: up -> build -> test -> open report
	docker compose -f ../../deployment/docker/docker-compose.e2e.yml up --build -d
	@echo "Waiting for backend..."
	@until curl -sf http://localhost:8090/api/v1/healthcheck > /dev/null; do sleep 2; done
	@echo "Backend ready."
	VITE_API_BASE_URL=http://localhost:8090/api/v1 \
	VITE_WS_URL=ws://localhost:8090/api/v1/ws \
	VITE_SSE_URL=http://localhost:8090/api/v1/sse \
	bun run build
	$(PW_ENV) bun x playwright test --reporter=html
	bun x playwright show-report

e2e-down: ## Tear down e2e stack and wipe volumes
	docker compose -f ../../deployment/docker/docker-compose.e2e.yml down -v

e2e-spec: ## Run one spec (stack must be up): make e2e-spec SPEC=auth.spec.ts
	$(PW_ENV) bun x playwright test $(SPEC)

e2e-fast: ## User-facing specs only, skip admin-* (stack must be up)
	$(PW_ENV) bun x playwright test --grep-invert 'admin-' --reporter=html
	bun x playwright show-report

e2e-ui: ## Interactive Playwright UI (e2e stack must be up, not local-compose)
	@if docker inspect backend --format '{{.Config.Image}}' 2>/dev/null | grep -q 'astroctfb-backend-local'; then \
	  echo ""; \
	  echo "ERROR: local-compose backend detected."; \
	  echo "  'make e2e-ui' requires the e2e stack (docker-compose.e2e.yml)."; \
	  echo "  Use 'make e2e-ui-local' to run Playwright UI against local-compose."; \
	  echo "  Or stop local-compose first and run 'make e2e' to bring up the e2e stack."; \
	  echo ""; \
	  exit 1; \
	fi
	$(PW_ENV) bun x playwright test --ui

e2e-local: ## Full e2e run against local-compose (.env.local creds, frontend on :8000)
	@echo "Seeding against local-compose (reading creds from .env.local)..."
	VITE_BACKEND_PORT=8090 bun e2e/seed.ts
	@echo "Building frontend bundle for local-compose..."
	VITE_API_BASE_URL=http://localhost:8000/api/v1 \
	VITE_WS_URL=ws://localhost:8000/api/v1/ws \
	VITE_SSE_URL=http://localhost:8000/api/v1/sse \
	bun run build
	$(PW_ENV) bun x playwright test --reporter=html
	bun x playwright show-report

e2e-ui-local: ## Interactive Playwright UI against local-compose (stack must be up)
	@echo "Starting Playwright UI against local-compose..."
	VITE_BACKEND_PORT=8090 bun e2e/seed.ts
	$(PW_ENV) bun x playwright test --ui

# CLEAN =====================================================================

##@ Cleanup

clean: ## Wipe dist/, node_modules/, bun.lock and reinstall everything from scratch
	rm -rf dist node_modules bun.lock
