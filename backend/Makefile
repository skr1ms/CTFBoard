ifneq ($(wildcard ../.env.local),)
include ../.env.local
export
else
$(warning WARNING: ../.env.local file not found!)
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
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

generate-openapi: ## Generate OpenAPI code from spec
	@echo "Generating OpenAPI code..."
	@mkdir -p internal/openapi
	@oapi-codegen -config oapi-codegen.yml internal/openapi/openapi.yaml
	@echo "OpenAPI code generated"

generate-mocks: ## Generate all mocks for usecase packages
	@echo "Generating mocks..."
	@mockery --config .mockery/.mockery_user.yml
	@mockery --config .mockery/.mockery_team.yml
	@mockery --config .mockery/.mockery_challenge.yml
	@mockery --config .mockery/.mockery_competition.yml
	@mockery --config .mockery/.mockery_email.yml
	@echo "All mocks generated"

generate-mocks-user: ## Generate mocks for user package
	@mockery --config .mockery/.mockery_user.yml

generate-mocks-team: ## Generate mocks for team package
	@mockery --config .mockery/.mockery_team.yml

generate-mocks-challenge: ## Generate mocks for challenge package
	@mockery --config .mockery/.mockery_challenge.yml

generate-mocks-competition: ## Generate mocks for competition package
	@mockery --config .mockery/.mockery_competition.yml

generate-mocks-email: ## Generate mocks for email package
	@mockery --config .mockery/.mockery_email.yml


build: generate-openapi ## Build the binary
	@echo "Building binary..."
	@go build -o bin/backend cmd/app/main.go
	@echo "Binary built: bin/backend"

run: generate-openapi ## Run the application locally
	@echo "Running application..."
	@go run cmd/app/main.go

clean: ## Clean generated files
	@echo "Cleaning..."
	@rm -rf bin
	@rm -f coverage.out coverage.html
	@echo "Clean completed"

# TESTING ==============================================================================================================

##@ Tests

test: test-race

test-race: ## Run all tests with race detection
	@echo "Running all tests (race)..."
	@go test -race ./...

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
	@go test -race -v ./integration-test/...

test-e2e: ## Run E2E tests (no race)
	@echo "Running E2E tests..."
	@go test -v ./e2e-test/...

test-e2e-race: ## Run E2E tests with race detection
	@echo "Running E2E tests (race)..."
	@go test -race -v ./e2e-test/...

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

test-coverage: ## Generate coverage report (all packages, no race)
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

# QUALITY ==============================================================================================================

lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./...

deps-audit: ## Check dependencies for vulnerabilities
	@echo "Checking dependencies..."
	@govulncheck ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	@go mod tidy
