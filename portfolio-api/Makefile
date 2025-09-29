# Portfolio API Makefile

.PHONY: help build run test clean docker-build docker-run docker-stop docker-logs deps lint format check coverage dev prod

# Variables
APP_NAME := portfolio-api
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT) -w -s"

# Docker variables
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_LATEST := $(APP_NAME):latest

help: ## Show this help message
	@echo "Portfolio API - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

## Development Commands

deps: ## Download and install dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/main.go

run: ## Run the application locally
	@echo "Running $(APP_NAME)..."
	$(GOCMD) run ./cmd/main.go

dev: ## Run in development mode with hot reload (requires air)
	@echo "Starting development server with hot reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Or run 'make run' for single execution"; \
	fi

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -timeout=300s ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## Code Quality

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with:"; \
		echo "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

format: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

check: format vet lint test ## Run all checks (format, vet, lint, test)

## Docker Commands

docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) -t $(DOCKER_LATEST) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 --name $(APP_NAME) $(DOCKER_LATEST)

docker-run-detached: ## Run Docker container in background
	@echo "Running Docker container in background..."
	docker run -d -p 8080:8080 --name $(APP_NAME) $(DOCKER_LATEST)

docker-stop: ## Stop Docker container
	@echo "Stopping Docker container..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

docker-logs: ## Show Docker container logs
	docker logs -f $(APP_NAME)

## Docker Compose Commands

compose-up: ## Start all services with docker-compose
	@echo "Starting all services..."
	docker-compose up -d

compose-up-build: ## Build and start all services
	@echo "Building and starting all services..."
	docker-compose up -d --build

compose-down: ## Stop all services
	@echo "Stopping all services..."
	docker-compose down

compose-logs: ## Show logs from all services
	docker-compose logs -f

compose-logs-api: ## Show logs from API service only
	docker-compose logs -f portfolio-api

compose-restart: ## Restart all services
	@echo "Restarting all services..."
	docker-compose restart

compose-restart-api: ## Restart API service only
	@echo "Restarting API service..."
	docker-compose restart portfolio-api

## Development Environment

dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	$(MAKE) deps
	@echo "Creating local config directory..."
	mkdir -p configs/local
	@echo "Development environment setup complete!"

dev-up: ## Start development environment
	@echo "Starting development environment..."
	docker-compose --profile development up -d

dev-down: ## Stop development environment
	@echo "Stopping development environment..."
	docker-compose --profile development down

## Production Environment

prod-build: ## Build for production
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o bin/$(APP_NAME) ./cmd/main.go

prod-up: ## Start production environment
	@echo "Starting production environment..."
	docker-compose up -d

prod-down: ## Stop production environment
	@echo "Stopping production environment..."
	docker-compose down

## Monitoring

monitor-up: ## Start monitoring stack
	@echo "Starting monitoring stack..."
	docker-compose --profile monitoring up -d

monitor-down: ## Stop monitoring stack
	@echo "Stopping monitoring stack..."
	docker-compose --profile monitoring down

## Database Commands

db-migrate: ## Run database migrations (placeholder)
	@echo "Database migrations would run here..."
	@echo "Currently using MongoDB initialization script"

db-seed: ## Seed database with sample data
	@echo "Seeding database..."
	docker-compose exec portfolio-mongo mongosh portfolio_db /docker-entrypoint-initdb.d/mongo-init.js

db-backup: ## Backup database
	@echo "Creating database backup..."
	mkdir -p backups
	docker-compose exec portfolio-mongo mongodump --db portfolio_db --out /tmp/backup
	docker cp portfolio-mongo:/tmp/backup ./backups/$(shell date +%Y%m%d_%H%M%S)

db-restore: ## Restore database from backup (requires BACKUP_PATH)
	@if [ -z "$(BACKUP_PATH)" ]; then \
		echo "Usage: make db-restore BACKUP_PATH=./backups/20231201_120000"; \
		exit 1; \
	fi
	@echo "Restoring database from $(BACKUP_PATH)..."
	docker cp $(BACKUP_PATH) portfolio-mongo:/tmp/restore
	docker-compose exec portfolio-mongo mongorestore --db portfolio_db /tmp/restore/portfolio_db

## Utility Commands

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf bin/
	rm -rf coverage.out coverage.html
	rm -rf tmp/

clean-docker: ## Clean Docker images and containers
	@echo "Cleaning Docker artifacts..."
	docker-compose down --volumes --remove-orphans
	docker system prune -f
	docker volume prune -f

logs: ## Show application logs (when running via docker-compose)
	docker-compose logs -f portfolio-api

health: ## Check service health
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health || echo "Service not responding"

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

## Documentation

docs: ## Generate API documentation (placeholder)
	@echo "API documentation generation would be implemented here"
	@echo "Consider using swaggo/swag for Swagger documentation"

## Security

security-scan: ## Run security scan (requires gosec)
	@echo "Running security scan..."
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with:"; \
		echo "go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

## Performance

load-test: ## Run load tests (requires k6)
	@echo "Running load tests..."
	@if command -v k6 > /dev/null; then \
		k6 run scripts/load-test.js; \
	else \
		echo "k6 not found. Install from https://k6.io/docs/getting-started/installation/"; \
	fi

profile: ## Run CPU profile
	@echo "Running with CPU profiling..."
	$(GOCMD) run -cpuprofile=cpu.prof ./cmd/main.go

## Installation Commands

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "Development tools installed!"

## Default target
all: check build ## Run checks and build