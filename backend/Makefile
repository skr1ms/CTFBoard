ifneq ($(wildcard ../.env.local),)
include ../.env.local
export
else ifneq ($(wildcard ../.env.local.example),)
include ../.env.local.example
export
endif

POSTGRES_DSN ?= postgres://user:password@localhost:5432/astroctfb?sslmode=disable
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

OAPI_CODEGEN_VERSION ?= v2.5.1
SQLC_VERSION ?= v1.28.0
MOCKERY_VERSION ?= v2.49.0
WIRE_VERSION ?= v0.6.0
GOLANGCI_LINT_VERSION ?= v2.0.0
REDOCLY_VERSION ?= 1.25.0

install-tools: ## Install development tools (versions pinned for reproducible builds)
	@echo "Installing tools..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	@go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/vektra/mockery/v2/cmd/mockery@$(MOCKERY_VERSION)
	@echo "Tools installed"

OPENAPI_SRC = internal/openapi/openapi.yml
OPENAPI_BUNDLE = internal/openapi/openapi.yaml
OPENAPI_MERGED_SCHEMAS = internal/openapi/components/schemas.yml

##@ Code generation (validate-openapi/openapi-bundle: Node.js/npx; merge-schemas: Python 3)

merge-schemas: ## Merge all schema files into components/schemas.yml (temporary file)
	@echo "Merging schema files..."
	@python3 scripts/merge-schemas.py

openapi-bundle: merge-schemas ## Bundle split OpenAPI spec (openapi.yml + routes) into single openapi.yaml
	@echo "Bundling OpenAPI spec..."
	@mkdir -p internal/openapi
	@command -v npx >/dev/null 2>&1 || (echo "Error: npx (Node.js) required for openapi-bundle. Install Node.js or run: npm install -g @redocly/cli"; exit 1)
	npx --yes @redocly/cli@$(REDOCLY_VERSION) bundle $(OPENAPI_SRC) -o $(OPENAPI_BUNDLE) --ext yaml
	@echo "OpenAPI spec bundled to $(OPENAPI_BUNDLE)"

openapi: oapi-codegen ## Alias for OpenAPI bundle + codegen

oapi-codegen: openapi-bundle ## Generate OpenAPI code from spec (bundles openapi.yml then runs codegen)
	@echo "Generating OpenAPI code..."
	@oapi-codegen -config codegen/oapi-codegen-types.yml $(OPENAPI_BUNDLE)
	@oapi-codegen -config codegen/oapi-codegen-server.yml $(OPENAPI_BUNDLE)
	@oapi-codegen -config codegen/oapi-codegen-spec.yml $(OPENAPI_BUNDLE)
	@oapi-codegen -config codegen/oapi-codegen-client.yml $(OPENAPI_BUNDLE)
	@rm -f $(OPENAPI_BUNDLE) $(OPENAPI_MERGED_SCHEMAS)
	@echo "OpenAPI code generated"

sqlc: sqlc-verify ## Generate sqlc code from queries and schema
	@echo "Generating sqlc code..."
	@sqlc -f codegen/sqlc.yaml generate
	@echo "sqlc code generated"

sqlc-verify: ## Verify sqlc schema and queries compile without generating code
	@echo "Verifying sqlc..."
	@sqlc -f codegen/sqlc.yaml compile
	@echo "sqlc verification OK"

validate-openapi: openapi-bundle ## Validate OpenAPI spec (bundles then lints with minimal ruleset)
	@echo "Validating OpenAPI spec..."
	@command -v npx >/dev/null 2>&1 || (echo "Error: npx (Node.js) required. Install Node.js or run: npm install -g @redocly/cli"; exit 1)
	@npx --yes @redocly/cli@$(REDOCLY_VERSION) lint $(OPENAPI_BUNDLE)
	@echo "OpenAPI validation OK"

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@goose -dir migrations postgres "$(POSTGRES_DSN)" up
	@echo "Migrations applied"

migrate-down: ## Run database migrations down (last batch)
	@echo "Reverting migrations..."
	@goose -dir migrations postgres "$(POSTGRES_DSN)" down
	@echo "Migrations reverted"

migrate-status: ## Show migration status
	@goose -dir migrations postgres "$(POSTGRES_DSN)" status

migrate-create: ## Create new migration (usage: make migrate-create name=create_something)
	@goose -dir migrations create $(name) sql

wire: ## Generate Wire DI code
	@echo "Generating Wire code..."
	@cd internal/wire && go generate .
	@echo "Wire code generated"

mockery: ## Generate all mocks (one config per usecase so same interface can output to different mocks dirs; single-file config would require one dir per interface and break tests)
	@echo "Generating mocks..."
	@mockery --config codegen/.mockery_user.yml
	@mockery --config codegen/.mockery_team.yml
	@mockery --config codegen/.mockery_challenge.yml
	@mockery --config codegen/.mockery_competition.yml
	@mockery --config codegen/.mockery_email.yml
	@mockery --config codegen/.mockery_settings.yml
	@mockery --config codegen/.mockery_notification.yml
	@mockery --config codegen/.mockery_page.yml
	@mockery --config codegen/.mockery_pkg.yml
	@mockery --config codegen/.mockery_seed.yml
	@mockery --config codegen/.mockery_backup.yml
	@mockery --config codegen/.mockery_middleware.yml
	@mockery --config codegen/.mockery_cache.yml
	@mockery --config codegen/.mockery_usecase.yml
	@mockery --config codegen/.mockery_config.yml
	@echo "All mocks generated"

generate-mocks-user: ## Generate mocks for user package
	@mockery --config codegen/.mockery_user.yml

generate-mocks-team: ## Generate mocks for team package
	@mockery --config codegen/.mockery_team.yml

generate-mocks-challenge: ## Generate mocks for challenge package
	@mockery --config codegen/.mockery_challenge.yml

generate-mocks-competition: ## Generate mocks for competition package
	@mockery --config codegen/.mockery_competition.yml

generate-mocks-email: ## Generate mocks for email package
	@mockery --config codegen/.mockery_email.yml

generate-mocks-pkg: ## Generate mocks for pkg interfaces used in pkg tests (mailer)
	@mockery --config codegen/.mockery_pkg.yml

generate-mocks-settings: ## Generate mocks for settings package
	@mockery --config codegen/.mockery_settings.yml

generate-mocks-notification: ## Generate mocks for notification package
	@mockery --config codegen/.mockery_notification.yml

generate-mocks-page: ## Generate mocks for page package
	@mockery --config codegen/.mockery_page.yml

generate-mocks-seed: ## Generate mocks for seed package
	@mockery --config codegen/.mockery_seed.yml

generate-mocks-backup: ## Generate mocks for backup package
	@mockery --config codegen/.mockery_backup.yml

generate-mocks-middleware: ## Generate mocks for middleware interfaces (APITokenAuther, UserByIDGetter)
	@mockery --config codegen/.mockery_middleware.yml

generate-mocks-cache: ## Generate mocks for cache interfaces (TeamBracketIDGetter)
	@mockery --config codegen/.mockery_cache.yml

generate-mocks-config: ## Generate mocks for config interfaces (vaultSecretGetter)
	@mockery --config codegen/.mockery_config.yml

generate: validate-openapi oapi-codegen sqlc mockery wire ## Run all code generation (validate OpenAPI, then OpenAPI, sqlc, mocks)
	@echo "All code generation completed"

build: ## Build the binary (run 'make generate' first if spec/sql changed)
	@echo "Building binary..."
	@go build -o bin/backend cmd/app/main.go
	@echo "Binary built: bin/backend"

build-cleanup: ## Build cleanup binary (for CI: GOOS=linux GOARCH=amd64)
	@echo "Building cleanup binary..."
	@GOOS=linux GOARCH=amd64 go build -o bin/astroctfb-cleanup ./cmd/cleanup
	@echo "Binary built: bin/astroctfb-cleanup"

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

test: test-race

TEST_PKGS = ./cmd/... ./config/... ./internal/... ./pkg/... ./integration-test/... ./e2e-test/...

test-race: ## Run all tests with race detection (default, excludes load-test)
	@echo "Running all tests (race)..."
	@go test -race -parallel 5 -timeout $(TEST_RACE_TIMEOUT) $(TEST_PKGS)

test-no-race: ## Run all tests without race (faster, excludes load-test)
	@echo "Running all tests (no race)..."
	@go test -parallel 5 $(TEST_PKGS)

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

COVERPKG_TESTABLE = ./internal/controller/...,./internal/usecase,./internal/usecase/challenge,./internal/usecase/competition,./internal/usecase/email,./internal/usecase/settings,./internal/usecase/notification,./internal/usecase/page,./internal/usecase/team,./internal/usecase/user,./internal/repo/persistent,./internal/storage,./config/...,./pkg/crypto,./pkg/mailer,./pkg/cache,./pkg/validator,./pkg/vault,./pkg/websocket

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
	@golangci-lint run -v --timeout $(LINT_TIMEOUT) ./...

lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running linter with auto-fix..."
	@golangci-lint run -v --fix --timeout $(LINT_TIMEOUT) ./...

deps-audit: ## Check dependencies for vulnerabilities
	@echo "Checking dependencies..."
	@govulncheck ./...

fmt: ## Format code with gofmt and goimports
	@echo "Formatting code..."
	@command -v goimports >/dev/null 2>&1 || (echo "goimports not found, run: make install-tools"; exit 1)
	@gofmt -w .
	@goimports -w -local github.com/TakuyaYagam1/AstroCTFb .
	@echo "Done"

mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	@go mod tidy
