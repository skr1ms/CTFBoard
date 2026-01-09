.PHONY: help swagger test build run clean install-tools

help:
	@echo "Available commands:"
	@echo "  make swagger      - Generate Swagger documentation"
	@echo "  make test         - Run all tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make build        - Build binary"
	@echo "  make run          - Run application"
	@echo "  make clean        - Clean generated files"
	@echo "  make install-tools - Install required tools"
	@echo "  make lint         - Run linter"
	@echo "  make fmt          - Format code"
	@echo "  make mod-tidy     - Update dependencies"
	@echo "  make mod-vendor   - Create vendor directory"

install-tools:
	@echo "Installing swag..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed"

swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g cmd/app/main.go -o docs
	@echo "Swagger documentation generated"

test:
	@echo "Running tests..."
	@go test ./...

test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

build:
	@echo "Building binary..."
	@go build -o bin/backend cmd/app/main.go
	@echo "Binary built: bin/backend"

run:
	@echo "Running application..."
	@go run cmd/app/main.go

clean:
	@echo "Cleaning generated files..."
	@rm -rf docs
	@rm -rf bin
	@rm -f coverage.out coverage.html
	@echo "Clean completed"

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...

mod-tidy:
	@echo "Updating dependencies..."
	@go mod tidy

mod-vendor:
	@echo "Creating vendor directory..."
	@go mod vendor

