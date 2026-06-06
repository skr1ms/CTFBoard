ifneq ($(wildcard ../.env.local),)
include ../.env.local
export
else ifneq ($(wildcard ../.env.local.example),)
include ../.env.local.example
export
endif

POSTGRES_DSN ?= postgres://user:password@localhost:5432/ctf_platform?sslmode=disable
BASE_STACK = docker compose -f ../deployment/docker/docker-compose.local.yml --env-file ../.env.local

# HELP =================================================================================================================
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# COMPOSE ==============================================================================================================

##@ Compose

compose-up: compose-infra

compose-infra: ## Start infrastructure only (db, redis, seaweedfs, monitoring)
	$(BASE_STACK) up --build -d postgres redis seaweedfs postgres-exporter redis-exporter prometheus loki promtail grafana cadvisor
	@echo "Infrastructure started"

compose-full: ## Start full stack (infra + backend + vault)
	$(BASE_STACK) up --build -d
	@echo "Full stack started"

compose-up-all: compose-full ## Alias for compose-full

compose-down: ## Stop and remove compose stack
	$(BASE_STACK) down --remove-orphans
	@echo "Stack stopped"

compose-logs: ## Follow compose logs
	$(BASE_STACK) logs -f

compose-ps: ## List compose services status
	$(BASE_STACK) ps

# APP ==================================================================================================================

OAPI_CODEGEN_VERSION ?= v2.6.0
SQLC_VERSION ?= v1.31.1
MOCKERY_VERSION ?= v3.5.5
WIRE_VERSION ?= v0.7.0
GOLANGCI_LINT_VERSION ?= v2.12.2
GOLANGCI_LINT_VERSION_NUM := $(patsubst v%,%,$(GOLANGCI_LINT_VERSION))
REDOCLY_VERSION ?= 1.26.0
PYTHON ?= python3
CODEGEN_PY_REQUIREMENTS ?= scripts/requirements-codegen.txt
CODEGEN_VENV ?= .venv-codegen
CODEGEN_PYTHON ?= $(CODEGEN_VENV)/bin/python
CODEGEN_PY_DEPS_STAMP ?= $(CODEGEN_VENV)/.deps.stamp
CODEGEN_GOFLAGS ?= -mod=readonly
GO_BIN_DIR := $(shell sh -c 'gobin=$$(go env GOBIN); if [ -n "$$gobin" ]; then printf "%s" "$$gobin"; else printf "%s/bin" "$$(go env GOPATH)"; fi')
OAPI_CODEGEN := $(shell sh -c 'if [ -x "$(GO_BIN_DIR)/oapi-codegen" ]; then printf "%s" "$(GO_BIN_DIR)/oapi-codegen"; else command -v oapi-codegen 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/oapi-codegen"; fi')
GOVULNCHECK := $(shell sh -c 'if [ -x "$(GO_BIN_DIR)/govulncheck" ]; then printf "%s" "$(GO_BIN_DIR)/govulncheck"; else command -v govulncheck 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/govulncheck"; fi')
GOLANGCI_LINT := $(shell sh -c 'for bin in "$(GO_BIN_DIR)/golangci-lint" "$$(command -v golangci-lint 2>/dev/null)"; do if [ -x "$$bin" ] && "$$bin" --version 2>/dev/null | grep -q "version $(GOLANGCI_LINT_VERSION_NUM)"; then printf "%s" "$$bin"; exit; fi; done; if [ -x "$(GO_BIN_DIR)/golangci-lint" ]; then printf "%s" "$(GO_BIN_DIR)/golangci-lint"; else command -v golangci-lint 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/golangci-lint"; fi')
SQLC := $(shell sh -c 'if [ -x "$(GO_BIN_DIR)/sqlc" ]; then printf "%s" "$(GO_BIN_DIR)/sqlc"; else command -v sqlc 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/sqlc"; fi')
GOOSE := $(shell sh -c 'if [ -x "$(GO_BIN_DIR)/goose" ]; then printf "%s" "$(GO_BIN_DIR)/goose"; else command -v goose 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/goose"; fi')
GOIMPORTS := $(shell sh -c 'if [ -x "$(GO_BIN_DIR)/goimports" ]; then printf "%s" "$(GO_BIN_DIR)/goimports"; else command -v goimports 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/goimports"; fi')
MOCKERY := $(shell sh -c 'if [ -x "$(GO_BIN_DIR)/mockery" ]; then printf "%s" "$(GO_BIN_DIR)/mockery"; else command -v mockery 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/mockery"; fi')
WIRE := $(shell sh -c 'if [ -x "$(GO_BIN_DIR)/wire" ]; then printf "%s" "$(GO_BIN_DIR)/wire"; else command -v wire 2>/dev/null || printf "%s" "$(GO_BIN_DIR)/wire"; fi')

install-tools: ## Install development tools
	@echo "Installing tools..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	@go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)
	@echo "Tools installed"

install-tools-latest: ## Install latest tool versions to $(GO_BIN_DIR) for local use (not for CI/reproducible codegen)
	@echo "Installing latest tools to $(GO_BIN_DIR)..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/google/wire/cmd/wire@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/vektra/mockery/v3@latest
	@echo "Latest tools installed"

OPENAPI_SRC = internal/openapi/openapi.yml
OPENAPI_BUNDLE = internal/openapi/openapi.yaml
OPENAPI_MERGED_SCHEMAS = internal/openapi/components/schemas.yml

##@ Code generation (validate-openapi/openapi-bundle: Node.js/npx; merge-schemas: Python 3)

install-codegen-python-deps: $(CODEGEN_PY_DEPS_STAMP) ## Install Python dependencies required by OpenAPI schema merge

$(CODEGEN_PY_DEPS_STAMP): $(CODEGEN_PY_REQUIREMENTS)
	@echo "Installing codegen Python dependencies..."
	@$(PYTHON) -m venv $(CODEGEN_VENV)
	@$(CODEGEN_PYTHON) -m pip install --upgrade pip
	@$(CODEGEN_PYTHON) -m pip install -r $(CODEGEN_PY_REQUIREMENTS)
	@touch $(CODEGEN_PY_DEPS_STAMP)

merge-schemas: install-codegen-python-deps ## Merge all schema files into components/schemas.yml (temporary file)
	@echo "Merging schema files..."
	@$(CODEGEN_PYTHON) scripts/merge-schemas.py

openapi-bundle: merge-schemas ## Bundle split OpenAPI spec (openapi.yml + routes) into single openapi.yaml
	@echo "Bundling OpenAPI spec..."
	@mkdir -p internal/openapi
	@command -v npx >/dev/null 2>&1 || (echo "Error: npx (Node.js) required for openapi-bundle. Install Node.js or run: npm install -g @redocly/cli"; exit 1)
	npx --yes @redocly/cli@$(REDOCLY_VERSION) bundle $(OPENAPI_SRC) -o $(OPENAPI_BUNDLE) --ext yaml
	@echo "OpenAPI spec bundled to $(OPENAPI_BUNDLE)"

openapi: oapi-codegen ## Alias for OpenAPI bundle + codegen

oapi-codegen: openapi-bundle ## Generate OpenAPI code from spec (bundles openapi.yml then runs codegen)
	@echo "Generating OpenAPI code..."
	@$(OAPI_CODEGEN) -config codegen/oapi-codegen-types.yml $(OPENAPI_BUNDLE)
	@$(OAPI_CODEGEN) -config codegen/oapi-codegen-server.yml $(OPENAPI_BUNDLE)
	@$(OAPI_CODEGEN) -config codegen/oapi-codegen-spec.yml $(OPENAPI_BUNDLE)
	@$(OAPI_CODEGEN) -config codegen/oapi-codegen-client.yml $(OPENAPI_BUNDLE)
	@rm -f $(OPENAPI_BUNDLE) $(OPENAPI_MERGED_SCHEMAS)
	@echo "OpenAPI code generated"

sqlc: sqlc-verify ## Generate sqlc code from queries and schema
	@echo "Generating sqlc code..."
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(SQLC) -f codegen/sqlc.yaml generate
	@echo "sqlc code generated"

sqlc-verify: ## Verify sqlc schema and queries compile without generating code
	@echo "Verifying sqlc..."
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(SQLC) -f codegen/sqlc.yaml compile
	@echo "sqlc verification OK"

validate-openapi: openapi-bundle ## Validate OpenAPI spec (bundles then lints with minimal ruleset)
	@echo "Validating OpenAPI spec..."
	@command -v npx >/dev/null 2>&1 || (rm -f $(OPENAPI_BUNDLE) $(OPENAPI_MERGED_SCHEMAS); echo "Error: npx (Node.js) required. Install Node.js or run: npm install -g @redocly/cli"; exit 1)
	@trap 'rm -f $(OPENAPI_BUNDLE) $(OPENAPI_MERGED_SCHEMAS)' EXIT; npx --yes @redocly/cli@$(REDOCLY_VERSION) lint $(OPENAPI_BUNDLE)
	@echo "OpenAPI validation OK"

validate-openapi-bundle: openapi-bundle ## Validate OpenAPI bundle without removing it (used by generate before oapi-codegen)
	@echo "Validating OpenAPI spec..."
	@command -v npx >/dev/null 2>&1 || (rm -f $(OPENAPI_BUNDLE) $(OPENAPI_MERGED_SCHEMAS); echo "Error: npx (Node.js) required. Install Node.js or run: npm install -g @redocly/cli"; exit 1)
	@npx --yes @redocly/cli@$(REDOCLY_VERSION) lint $(OPENAPI_BUNDLE)
	@echo "OpenAPI validation OK"

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@$(GOOSE) -dir migrations postgres "$(POSTGRES_DSN)" up
	@echo "Migrations applied"

migrate-down: ## Run database migrations down (last batch)
	@echo "Reverting migrations..."
	@$(GOOSE) -dir migrations postgres "$(POSTGRES_DSN)" down
	@echo "Migrations reverted"

migrate-status: ## Show migration status
	@$(GOOSE) -dir migrations postgres "$(POSTGRES_DSN)" status

migrate-create: ## Create new migration (usage: make migrate-create name=create_something)
	@$(GOOSE) -dir migrations create $(name) sql

wire: ## Generate Wire DI code
	@echo "Generating Wire code..."
	@cd internal/wire && GOFLAGS="$(CODEGEN_GOFLAGS)" $(WIRE)
	@$(GOIMPORTS) -w -local github.com/TakuyaYagam1/AstroCTFb internal/wire/wire_gen.go
	@echo "Wire code generated"

mockery: ## Generate all mocks (one config per usecase so same interface can output to different mocks dirs; single-file config would require one dir per interface and break tests)
	@echo "Generating mocks..."
	@$(MOCKERY) version | grep -Eq '^v3\.' || (echo "Error: mockery v3.x is required. Run 'make install-tools' or ensure PATH points to github.com/vektra/mockery/v3."; exit 1)
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_user.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_team.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_challenge.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_competition.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_email.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_settings.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_notification.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_page.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_pkg.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_seed.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_backup.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_middleware.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_cache.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_usecase.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_avatar.yml
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_config.yml
	@echo "All mocks generated"

generate-mocks-user: ## Generate mocks for user package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_user.yml

generate-mocks-team: ## Generate mocks for team package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_team.yml

generate-mocks-challenge: ## Generate mocks for challenge package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_challenge.yml

generate-mocks-competition: ## Generate mocks for competition package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_competition.yml

generate-mocks-email: ## Generate mocks for email package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_email.yml

generate-mocks-pkg: ## Generate mocks for pkg interfaces used in pkg tests (mailer)
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_pkg.yml

generate-mocks-settings: ## Generate mocks for settings package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_settings.yml

generate-mocks-notification: ## Generate mocks for notification package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_notification.yml

generate-mocks-page: ## Generate mocks for page package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_page.yml

generate-mocks-seed: ## Generate mocks for seed package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_seed.yml

generate-mocks-backup: ## Generate mocks for backup package
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_backup.yml

generate-mocks-middleware: ## Generate mocks for middleware interfaces (APITokenAuther, UserByIDGetter)
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_middleware.yml

generate-mocks-cache: ## Generate mocks for cache interfaces (TeamBracketIDGetter)
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_cache.yml

generate-mocks-config: ## Generate mocks for config interfaces (vaultSecretGetter)
	@GOFLAGS="$(CODEGEN_GOFLAGS)" $(MOCKERY) --config codegen/.mockery_config.yml

generate: validate-openapi-bundle oapi-codegen sqlc mockery wire ## Run all code generation (validate OpenAPI, then OpenAPI, sqlc, mocks)
	@echo "All code generation completed"

codegen-check: generate ## Verify generated files are committed and up to date
	@echo "Checking generated diff..."
	@git diff --exit-code -- . || (echo "Generated files are out of date. Run 'make generate' and commit the result."; exit 1)

build: ## Build the binary (run 'make generate' first if spec/sql changed)
	@echo "Building binary..."
	@go build -o bin/backend cmd/app/main.go
	@echo "Binary built: bin/backend"

build-cleanup: ## Build cleanup binary (for CI: GOOS=linux GOARCH=amd64)
	@echo "Building cleanup binary..."
	@GOOS=linux GOARCH=amd64 go build -o bin/ctf-platform-cleanup ./cmd/cleanup
	@echo "Binary built: bin/ctf-platform-cleanup"

run: generate ## Run the application locally (depends on generate so codegen is up to date)
	@echo "Running application..."
	@go run cmd/app/main.go

clean: ## Clean generated files
	@echo "Cleaning..."
	@rm -rf bin
	@rm -f coverage.out coverage.html
	@rm -f $(OPENAPI_BUNDLE) $(OPENAPI_MERGED_SCHEMAS)
	@echo "Clean completed"

# TESTING ==============================================================================================================

##@ Tests

TEST_RACE_TIMEOUT ?= 15m
BENCH_PKGS = ./internal/scoring/... ./pkg/crypto/... ./internal/websocket/... ./internal/usecase/competition/...
BENCH_TIME  ?= 3s
BENCH_COUNT ?= 1

# TEST_PKGS covers all packages except load-test (requires -tags load) and load-test itself.
TEST_PKGS = ./cmd/... ./config/... ./internal/... ./pkg/... ./integration-test/... ./e2e-test/...
TEST_FAST_PKGS = ./cmd/... ./config/... ./internal/... ./pkg/...

test: ## Run all tests without race (excludes load-test)
	@echo "Running all tests..."
	@go test -parallel 5 -timeout $(TEST_RACE_TIMEOUT) $(TEST_PKGS)

test-fast: ## Run fast local tests (cmd + config + internal + pkg, no Docker/e2e/load)
	@echo "Running fast local tests..."
	@go test -parallel 5 $(TEST_FAST_PKGS)

test-race: ## Run all tests with race detection (excludes load-test)
	@echo "Running all tests (race)..."
	@go test -race -parallel 5 -timeout $(TEST_RACE_TIMEOUT) $(TEST_PKGS)

test-fast-race: ## Run fast local tests with race detection
	@echo "Running fast local tests (race)..."
	@go test -race -parallel 5 -timeout $(TEST_RACE_TIMEOUT) $(TEST_FAST_PKGS)

test-no-race: test ## Alias for test (no race)

bench: ## Run benchmark tests (scoring, crypto, websocket, competition)
	@echo "Running benchmarks..."
	@go test -run='^$$' -bench=. -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) -benchmem $(BENCH_PKGS)

bench-scoring: ## Benchmark dynamic scoring algorithms
	@go test -run='^$$' -bench=. -benchtime=$(BENCH_TIME) -benchmem ./internal/scoring/...

bench-crypto: ## Benchmark AES-256-GCM encrypt/decrypt
	@go test -run='^$$' -bench=. -benchtime=$(BENCH_TIME) -benchmem ./pkg/crypto/...

bench-websocket: ## Benchmark WebSocket broadcaster
	@go test -run='^$$' -bench=. -benchtime=$(BENCH_TIME) -benchmem ./internal/websocket/...

bench-competition: ## Benchmark competition usecases
	@go test -run='^$$' -bench=. -benchtime=$(BENCH_TIME) -benchmem ./internal/usecase/competition/...

test-unit: ## Run unit tests (internal + pkg) without race
	@echo "Running unit tests..."
	@go test -parallel 5 ./internal/... ./pkg/...

test-unit-race: ## Run unit tests with race detection
	@echo "Running unit tests (race)..."
	@go test -race -parallel 5 ./internal/... ./pkg/...

test-integration: ## Run integration tests (no race)
	@echo "Running integration tests..."
	@go test -parallel 5 -v ./integration-test/...

test-integration-race: ## Run integration tests with race detection
	@echo "Running integration tests (race)..."
	@go test -race -parallel 5 -timeout $(TEST_RACE_TIMEOUT) -v ./integration-test/...

test-e2e: ## Run E2E tests (no race)
	@echo "Running E2E tests..."
	@go test -parallel 5 -v ./e2e-test/...

test-e2e-race: ## Run E2E tests with race detection
	@echo "Running E2E tests (race)..."
	@go test -race -parallel 5 -timeout $(TEST_RACE_TIMEOUT) -v ./e2e-test/...

test-load: ## Run load tests (spins up testcontainers)
	@echo "Running load tests..."
	@go test -tags load -v -timeout 30m -count=1 ./load-test/...

test-load-short: ## Run load tests excluding the soak test
	@echo "Running load tests (short)..."
	@go test -tags load -v -short -timeout 10m -count=1 ./load-test/...

test-load-race: ## Run load tests with race detection (slower)
	@echo "Running load tests (race)..."
	@go test -tags load -race -v -timeout 60m -count=1 ./load-test/...

test-load-report: ## Run load tests (short mode) and write JSON reports to load-test/reports/
	@echo "Running load tests with JSON reports (short)..."
	@mkdir -p load-test/reports
	@LOAD_TEST_REPORT_DIR=load-test/reports go test -tags load -v -short -timeout 15m -count=1 ./load-test/...

test-load-report-full: ## Run full load tests including soak and write JSON reports
	@echo "Running full load tests with JSON reports..."
	@mkdir -p load-test/reports
	@LOAD_TEST_REPORT_DIR=load-test/reports go test -tags load -v -timeout 45m -count=1 ./load-test/...

test-usecase: ## Run all usecase tests with race
	@echo "Running usecase tests (race)..."
	@$(MAKE) test-usecase-user-race
	@$(MAKE) test-usecase-team-race
	@$(MAKE) test-usecase-challenge-race
	@$(MAKE) test-usecase-competition-race
	@$(MAKE) test-usecase-email-race

test-usecase-no-race: ## Run all usecase tests without race
	@echo "Running usecase tests (no race)..."
	@go test -parallel 5 -v ./internal/usecase/user/... ./internal/usecase/team/... ./internal/usecase/challenge/... ./internal/usecase/competition/... ./internal/usecase/email/...

test-usecase-user: test-usecase-user-race
test-usecase-user-race: ## Run user usecase tests with race
	@go test -race -parallel 5 -v ./internal/usecase/user/...

test-usecase-user-no-race: ## Run user usecase tests without race
	@go test -parallel 5 -v ./internal/usecase/user/...

test-usecase-team: test-usecase-team-race
test-usecase-team-race: ## Run team usecase tests with race
	@go test -race -parallel 5 -v ./internal/usecase/team/...

test-usecase-team-no-race: ## Run team usecase tests without race
	@go test -parallel 5 -v ./internal/usecase/team/...

test-usecase-challenge: test-usecase-challenge-race
test-usecase-challenge-race: ## Run challenge usecase tests with race
	@go test -race -parallel 5 -v ./internal/usecase/challenge/...

test-usecase-challenge-no-race: ## Run challenge usecase tests without race
	@go test -parallel 5 -v ./internal/usecase/challenge/...

test-usecase-competition: test-usecase-competition-race
test-usecase-competition-race: ## Run competition usecase tests with race
	@go test -race -parallel 5 -v ./internal/usecase/competition/...

test-usecase-competition-no-race: ## Run competition usecase tests without race
	@go test -parallel 5 -v ./internal/usecase/competition/...

test-usecase-email: test-usecase-email-race
test-usecase-email-race: ## Run email usecase tests with race
	@go test -race -parallel 5 -v ./internal/usecase/email/...

test-usecase-email-no-race: ## Run email usecase tests without race
	@go test -parallel 5 -v ./internal/usecase/email/...

COVERPKG = ./cmd/...,./config/...,./internal/...,./pkg/...

COVERPKG_TESTABLE = ./internal/controller/...,./internal/usecase,./internal/usecase/avatar,./internal/usecase/backup,./internal/usecase/cacheutil,./internal/usecase/challenge,./internal/usecase/competition,./internal/usecase/computil,./internal/usecase/email,./internal/usecase/guard,./internal/usecase/notification,./internal/usecase/page,./internal/usecase/settings,./internal/usecase/setup,./internal/usecase/storageadmin,./internal/usecase/team,./internal/usecase/user,./internal/repo,./internal/repo/persistent,./internal/repo/webapi,./internal/storage,./internal/cache,./internal/websocket,./config/...,./pkg/crypto,./pkg/mailer,./pkg/slug,./pkg/sse,./pkg/validator,./pkg/vault

test-coverage: ## Generate coverage report (unit + integration + e2e, -coverpkg to include production code)
	@echo "Running all tests with coverage (requires Docker for e2e)..."
	@go test -parallel 5 -coverprofile=coverage.out -coverpkg=$(COVERPKG) $(TEST_PKGS)
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

test-coverage-unit: ## Coverage from unit tests only (fast, no Docker)
	@echo "Running unit tests with coverage..."
	@go test -parallel 5 -coverprofile=coverage_unit.out -coverpkg=$(COVERPKG) ./internal/... ./pkg/... ./config/...
	@go tool cover -func=coverage_unit.out | tail -1
	@go tool cover -html=coverage_unit.out -o coverage.html
	@echo "Coverage report saved to coverage.html (unit only)"

test-coverage-testable: ## Coverage only for testable packages (excludes cmd, wire, openapi, entity, mocks, sqlc, logger, migrator, postgres, seed). Goal: 80%%
	@echo "Running all tests with coverage (testable packages only, requires Docker for e2e)..."
	@go test -parallel 5 -coverprofile=coverage_testable.out -coverpkg=$(COVERPKG_TESTABLE) $(TEST_PKGS)
	@echo "--- Coverage (testable packages) ---"
	@go tool cover -func=coverage_testable.out
	@go tool cover -html=coverage_testable.out -o coverage_testable.html
	@echo "Report: coverage_testable.html"

# QUALITY ==============================================================================================================

LINT_TIMEOUT ?= 5m

lint: ## Run golangci-lint (LINT_TIMEOUT for CI, e.g. 10m)
	@echo "Running linter..."
	@$(GOLANGCI_LINT) --version | grep -q "version $(GOLANGCI_LINT_VERSION_NUM)" || (echo "Error: golangci-lint $(GOLANGCI_LINT_VERSION_NUM) is required. Run 'make install-tools' or ensure PATH points to the pinned version."; exit 1)
	@$(GOLANGCI_LINT) run -v --timeout $(LINT_TIMEOUT) ./...
	@$(MAKE) audit-architecture-strict

lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running linter with auto-fix..."
	@$(GOLANGCI_LINT) --version | grep -q "version $(GOLANGCI_LINT_VERSION_NUM)" || (echo "Error: golangci-lint $(GOLANGCI_LINT_VERSION_NUM) is required. Run 'make install-tools' or ensure PATH points to the pinned version."; exit 1)
	@$(GOLANGCI_LINT) run -v --fix --timeout $(LINT_TIMEOUT) ./...

deps-audit: ## Check dependencies for vulnerabilities
	@echo "Checking production dependencies..."
	@pkgs="$$(go list ./... | grep -Ev '(^|/)(e2e-test|integration-test|load-test)(/|$$)|/pkg/testutil(/|$$)|/mock(/|$$)')" ; \
	if [ -z "$$pkgs" ]; then echo "No production packages found"; exit 1; fi; \
	$(GOVULNCHECK) $$pkgs

deps-audit-all: ## Check all dependencies, including e2e/testutil/load-test packages
	@echo "Checking all dependencies..."
	@$(GOVULNCHECK) ./...

audit-architecture: ## Report Clean Architecture drift signals without failing
	@echo "== Forbidden ordinary handler imports =="
	@files="$$(find internal/controller/restapi/v1 -maxdepth 1 -name '*.go' ! -name 'router*.go' ! -name 'setup.go' ! -name '*_test.go' -print)" ; \
	if [ -n "$$files" ]; then rg -n '"github.com/TakuyaYagam1/AstroCTFb/(pkg/|internal/(apperr|domain|repo/|storage/|cache/|usecase/|repo/persistent/sqlc))|"github.com/go-chi/render"|"github.com/jackc/pgx|"github.com/redis/go-redis|"github.com/minio/minio-go|"github.com/hashicorp/vault|"github.com/wahrwelt-kit/go-jwtkit' $$files || true; fi
	@echo "== Raw ResponseWriter writes in ordinary handlers =="
	@files="$$(find internal/controller/restapi/v1 -maxdepth 1 -name '*.go' ! -name 'router*.go' ! -name 'setup.go' ! -name '*_test.go' -print)" ; \
	if [ -n "$$files" ]; then rg -n '(w\.WriteHeader|w\.Write\(|http\.Error\(|json\.NewEncoder\(w\)|io\.Copy\(w\))' $$files || true; fi
	@echo "== Raw SQL outside generated/sqlc and allowed dynamic repository helpers =="
	@rg -n --glob '*.go' --glob '!*_test.go' -e '\.Exec\(ctx, ".*(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP)' -e '\.Query\(ctx, ".*(SELECT|INSERT|UPDATE|DELETE)' -e '\.QueryRow\(ctx, ".*(SELECT|INSERT|UPDATE|DELETE)' internal pkg config | rg -v 'internal/repo/persistent/(sqlc/|squirrel.go|tx_manager.go|base.go|backup_postgres.go|backup_postgres_import.go)' || true
	@echo "== Largest non-generated Go files =="
	@find . -path './internal/openapi' -prune -o -path './internal/repo/persistent/sqlc' -prune -o -path './e2e-test' -prune -o -path './integration-test' -prune -o -path './load-test' -prune -o -path '*/mock' -prune -o -name '*.go' ! -name '*_test.go' ! -name 'wire_gen.go' -print | xargs wc -l | sort -nr | head -30

audit-architecture-strict: ## Fail on Clean Architecture drift signals checked by audit-architecture
	@echo "Checking Clean Architecture boundaries..."
	@status=0; \
	files="$$(find internal/controller/restapi/v1 -maxdepth 1 -name '*.go' ! -name 'router*.go' ! -name 'setup.go' ! -name '*_test.go' -print)" ; \
	if [ -n "$$files" ] && rg -n '"github.com/TakuyaYagam1/AstroCTFb/(pkg/|internal/(apperr|domain|repo/|storage/|cache/|usecase/|repo/persistent/sqlc))|"github.com/go-chi/render"|"github.com/jackc/pgx|"github.com/redis/go-redis|"github.com/minio/minio-go|"github.com/hashicorp/vault|"github.com/wahrwelt-kit/go-jwtkit' $$files; then status=1; fi; \
	if [ -n "$$files" ] && rg -n '(w\.WriteHeader|w\.Write\(|http\.Error\(|json\.NewEncoder\(w\)|io\.Copy\(w\))' $$files; then status=1; fi; \
	if rg -n --glob '*.go' --glob '!*_test.go' -e '\.Exec\(ctx, ".*(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP)' -e '\.Query\(ctx, ".*(SELECT|INSERT|UPDATE|DELETE)' -e '\.QueryRow\(ctx, ".*(SELECT|INSERT|UPDATE|DELETE)' internal pkg config | rg -v 'internal/repo/persistent/(sqlc/|squirrel.go|tx_manager.go|base.go|backup_postgres.go|backup_postgres_import.go)'; then status=1; fi; \
	if rg -n --glob '*.go' --glob '!*_test.go' --glob '!**/mock/**' '"github.com/TakuyaYagam1/AstroCTFb/internal/(controller/|openapi|repo/persistent)|"github.com/wahrwelt-kit/go-httpkit/|"github.com/go-chi/|"github.com/jackc/pgx|"github.com/Masterminds/squirrel"|"github.com/redis/go-redis"|"github.com/minio/minio-go"|"github.com/hashicorp/vault"|"golang.org/x/oauth2' internal/usecase; then status=1; fi; \
	if [ "$$status" -ne 0 ]; then echo "Architecture audit failed"; exit "$$status"; fi; \
	echo "Architecture audit OK"

fmt: ## Format code with gofmt and goimports
	@echo "Formatting code..."
	@test -x "$(GOIMPORTS)" || (echo "goimports not found at $(GOIMPORTS), run: make install-tools"; exit 1)
	@files="$$(find . -path './internal/openapi' -prune -o -path './internal/repo/persistent/sqlc' -prune -o -path '*/mock' -prune -o -name '*.go' ! -name 'wire_gen.go' -print)" ; \
	if [ -n "$$files" ]; then gofmt -w $$files && $(GOIMPORTS) -w -local github.com/TakuyaYagam1/AstroCTFb $$files; fi
	@echo "Done"

fmt-generated: ## Format generated Go files explicitly
	@echo "Formatting generated code..."
	@test -x "$(GOIMPORTS)" || (echo "goimports not found at $(GOIMPORTS), run: make install-tools"; exit 1)
	@files="$$(find ./internal/openapi ./internal/repo/persistent/sqlc -name '*.go' -print 2>/dev/null; find . -path '*/mock/*.go' -print; find . -name wire_gen.go -print)" ; \
	if [ -n "$$files" ]; then gofmt -w $$files && $(GOIMPORTS) -w -local github.com/TakuyaYagam1/AstroCTFb $$files; fi
	@echo "Done"

mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	@go mod tidy
