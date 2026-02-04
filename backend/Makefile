ifneq ($(wildcard ../.env.local),)
include ../.env.local
export
else ifneq ($(wildcard ../.env.local.example),)
include ../.env.local.example
export
endif

POSTGRES_DSN ?= postgres://user:password@localhost:5432/ctfboard?sslmode=disable
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

install-tools: ## Install development tools
	@echo "Installing tools..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/google/wire/cmd/wire@latest
	@echo "Tools installed"

oapi-codegen: ## Generate OpenAPI code from spec (types, server, spec, client)
	@echo "Generating OpenAPI code..."
	@mkdir -p internal/openapi
	@oapi-codegen -config codegen/oapi-codegen-types.yml internal/openapi/openapi.yaml
	@oapi-codegen -config codegen/oapi-codegen-server.yml internal/openapi/openapi.yaml
	@oapi-codegen -config codegen/oapi-codegen-spec.yml internal/openapi/openapi.yaml
	@oapi-codegen -config codegen/oapi-codegen-client.yml internal/openapi/openapi.yaml
	@echo "OpenAPI code generated"

sqlc: ## Generate sqlc code from queries and schema
	@echo "Generating sqlc code..."
	@sqlc -f codegen/sqlc.yaml generate
	@echo "sqlc code generated"

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@migrate -path migrations -database "$(POSTGRES_DSN)" up
	@echo "Migrations applied"

migrate-down: ## Run database migrations down (last batch)
	@echo "Reverting migrations..."
	@migrate -path migrations -database "$(POSTGRES_DSN)" down 1
	@echo "Migrations reverted"

migrate-create: ## Create new migration (usage: make migrate-create name=create_something)
	@migrate create -ext sql -dir migrations -seq $(name)

wire: ## Generate Wire DI code
	@echo "Generating Wire code..."
	@cd internal/wire && go generate .
	@echo "Wire code generated"

mockery: ## Generate all mocks for usecase and pkg packages
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

generate: oapi-codegen sqlc mockery wire ## Run all code generation (OpenAPI, sqlc, mocks)
	@echo "All code generation completed"

build: oapi-codegen ## Build the binary
	@echo "Building binary..."
	@go build -o bin/backend cmd/app/main.go
	@echo "Binary built: bin/backend"

build-cleanup: ## Build cleanup binary (for CI: GOOS=linux GOARCH=amd64)
	@echo "Building cleanup binary..."
	@GOOS=linux GOARCH=amd64 go build -o bin/ctfboard-cleanup ./cmd/cleanup
	@echo "Binary built: bin/ctfboard-cleanup"

run: oapi-codegen ## Run the application locally
	@echo "Running application..."
	@go run cmd/app/main.go

clean: ## Clean generated files
	@echo "Cleaning..."
	@rm -rf bin
	@rm -f coverage.out coverage.html
	@echo "Clean completed"

# TESTING ==============================================================================================================

##@ Tests

TEST_RACE_TIMEOUT ?= 15m

test: test-race

test-race: ## Run all tests with race detection (default)
	@echo "Running all tests (race)..."
	@go test -race -timeout $(TEST_RACE_TIMEOUT) ./...

test-no-race: ## Run all tests without race (faster)
	@echo "Running all tests (no race)..."
	@go test ./...

test-unit: ## Run unit tests (internal + pkg) without race
	@echo "Running unit tests..."
	@go test ./internal/... ./pkg/...

test-unit-race: ## Run unit tests with race detection
	@echo "Running unit tests (race)..."
	@go test -race ./internal/... ./pkg/...

test-integration: ## Run integration tests (no race)
	@echo "Running integration tests..."
	@go test -v ./integration-test/...

test-integration-race: ## Run integration tests with race detection
	@echo "Running integration tests (race)..."
	@go test -race -timeout $(TEST_RACE_TIMEOUT) -v ./integration-test/...

test-e2e: ## Run E2E tests (no race)
	@echo "Running E2E tests..."
	@go test -v ./e2e-test/...

test-e2e-race: ## Run E2E tests with race detection
	@echo "Running E2E tests (race)..."
	@go test -race -timeout $(TEST_RACE_TIMEOUT) -v ./e2e-test/...

test-usecase: ## Run all usecase tests with race
	@echo "Running usecase tests (race)..."
	@$(MAKE) test-usecase-user-race
	@$(MAKE) test-usecase-team-race
	@$(MAKE) test-usecase-challenge-race
	@$(MAKE) test-usecase-competition-race
	@$(MAKE) test-usecase-email-race

test-usecase-no-race: ## Run all usecase tests without race
	@echo "Running usecase tests (no race)..."
	@go test -v ./internal/usecase/user/... ./internal/usecase/team/... ./internal/usecase/challenge/... ./internal/usecase/competition/... ./internal/usecase/email/...

test-usecase-user: test-usecase-user-race
test-usecase-user-race: ## Run user usecase tests with race
	@go test -race -v ./internal/usecase/user/...

test-usecase-user-no-race: ## Run user usecase tests without race
	@go test -v ./internal/usecase/user/...

test-usecase-team: test-usecase-team-race
test-usecase-team-race: ## Run team usecase tests with race
	@go test -race -v ./internal/usecase/team/...

test-usecase-team-no-race: ## Run team usecase tests without race
	@go test -v ./internal/usecase/team/...

test-usecase-challenge: test-usecase-challenge-race
test-usecase-challenge-race: ## Run challenge usecase tests with race
	@go test -race -v ./internal/usecase/challenge/...

test-usecase-challenge-no-race: ## Run challenge usecase tests without race
	@go test -v ./internal/usecase/challenge/...

test-usecase-competition: test-usecase-competition-race
test-usecase-competition-race: ## Run competition usecase tests with race
	@go test -race -v ./internal/usecase/competition/...

test-usecase-competition-no-race: ## Run competition usecase tests without race
	@go test -v ./internal/usecase/competition/...

test-usecase-email: test-usecase-email-race
test-usecase-email-race: ## Run email usecase tests with race
	@go test -race -v ./internal/usecase/email/...

test-usecase-email-no-race: ## Run email usecase tests without race
	@go test -v ./internal/usecase/email/...

COVERPKG = ./cmd/...,./config/...,./internal/...,./pkg/...

COVERPKG_TESTABLE = ./internal/controller/...,./internal/usecase,./internal/usecase/challenge,./internal/usecase/competition,./internal/usecase/email,./internal/usecase/settings,./internal/usecase/notification,./internal/usecase/page,./internal/usecase/team,./internal/usecase/user,./internal/repo/persistent,./internal/storage,./config/...,./pkg/crypto,./pkg/httputil,./pkg/jwt,./pkg/mailer,./pkg/redis,./pkg/validator,./pkg/vault,./pkg/websocket

test-coverage: ## Generate coverage report (unit + integration + e2e, -coverpkg to include production code)
	@echo "Running all tests with coverage (requires Docker for e2e)..."
	@go test -coverprofile=coverage.out -coverpkg=$(COVERPKG) ./...
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

test-coverage-unit: ## Coverage from unit tests only (fast, no Docker)
	@echo "Running unit tests with coverage..."
	@go test -coverprofile=coverage_unit.out -coverpkg=$(COVERPKG) ./internal/... ./pkg/... ./config/...
	@go tool cover -func=coverage_unit.out | tail -1
	@go tool cover -html=coverage_unit.out -o coverage.html
	@echo "Coverage report saved to coverage.html (unit only)"

test-coverage-testable: ## Coverage only for testable packages (excludes cmd, wire, openapi, entity, mocks, sqlc, logger, migrator, postgres, seed). Goal: 80%%
	@echo "Running all tests with coverage (testable packages only, requires Docker for e2e)..."
	@go test -coverprofile=coverage_testable.out -coverpkg=$(COVERPKG_TESTABLE) ./...
	@echo "--- Coverage (testable packages) ---"
	@go tool cover -func=coverage_testable.out
	@go tool cover -html=coverage_testable.out -o coverage_testable.html
	@echo "Report: coverage_testable.html"

# QUALITY ==============================================================================================================

LINT_TIMEOUT ?= 5m

lint: ## Run golangci-lint (LINT_TIMEOUT for CI, e.g. 10m)
	@echo "Running linter..."
	@golangci-lint run -v --timeout $(LINT_TIMEOUT) ./...

deps-audit: ## Check dependencies for vulnerabilities
	@echo "Checking dependencies..."
	@govulncheck ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	@go mod tidy
