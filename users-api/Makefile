# Variables
BINARY_NAME=users-api
BUILD_DIR=build
GO_VERSION=1.21

# Build commands
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/main.go

.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux cmd/main.go

.PHONY: clean
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

# Development commands
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	@go run cmd/main.go

.PHONY: dev
dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	@air

.PHONY: install-deps
install-deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Testing commands
.PHONY: test
test:
	@echo "Running tests..."
	@go test ./tests/unit/... -v

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@go test ./tests/integration/... -v

.PHONY: test-all
test-all:
	@echo "Running all tests..."
	@go test ./... -v

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Database commands
.PHONY: migrate-up
migrate-up:
	@echo "Running database migrations..."
	@migrate -path migrations -database "mysql://root:password@tcp(localhost:3306)/users_db" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back database migrations..."
	@migrate -path migrations -database "mysql://root:password@tcp(localhost:3306)/users_db" down

.PHONY: migrate-force
migrate-force:
	@echo "Forcing migration version..."
	@migrate -path migrations -database "mysql://root:password@tcp(localhost:3306)/users_db" force $(VERSION)

# Docker commands
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8001:8001 --env-file .env $(BINARY_NAME):latest

.PHONY: docker-compose-up
docker-compose-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down:
	@echo "Stopping services with docker-compose..."
	@docker-compose down

.PHONY: docker-compose-logs
docker-compose-logs:
	@echo "Showing docker-compose logs..."
	@docker-compose logs -f

# Linting and formatting
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Security scanning
.PHONY: security
security:
	@echo "Running security scan..."
	@gosec ./...

# Documentation
.PHONY: docs
docs:
	@echo "Generating API documentation..."
	@swag init -g cmd/main.go -o docs/

# Development setup
.PHONY: setup
setup: install-deps
	@echo "Setting up development environment..."
	@cp .env.example .env
	@echo "Created .env file from .env.example"
	@echo "Please update the .env file with your configuration"

# Full development cycle
.PHONY: dev-cycle
dev-cycle: fmt vet lint test build
	@echo "Development cycle complete!"

# Production build
.PHONY: prod-build
prod-build: clean fmt vet lint test-all build-linux
	@echo "Production build complete!"

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux"
	@echo "  clean         - Clean build directory"
	@echo "  run           - Run the application"
	@echo "  dev           - Run in development mode with hot reload"
	@echo "  test          - Run unit tests"
	@echo "  test-all      - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  security      - Run security scan"
	@echo "  docs          - Generate API documentation"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  setup         - Setup development environment"
	@echo "  help          - Show this help message"