# Variables
APP_NAME = orders-api
DOCKER_IMAGE = $(APP_NAME):latest
DOCKER_REGISTRY = localhost:5000
GO_VERSION = 1.21
MONGODB_VERSION = 7.0
RABBITMQ_VERSION = 3.12

# Build configuration
BUILD_DIR = ./bin
MAIN_PATH = ./cmd/server
LDFLAGS = -w -s -X main.version=$(shell git describe --tags --always --dirty) -X main.buildTime=$(shell date +%Y-%m-%dT%H:%M:%S%z)

# Docker Compose files
COMPOSE_FILE = docker-compose.yml
COMPOSE_DEV_FILE = docker-compose.dev.yml
COMPOSE_TEST_FILE = docker-compose.test.yml

.PHONY: help build run test docker-build docker-run docker-compose clean deps lint format coverage

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development targets
deps: ## Download and install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

build-dev: ## Build the application for development
	@echo "Building $(APP_NAME) for development..."
	mkdir -p $(BUILD_DIR)
	go build -race -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

run: build-dev ## Run the application locally
	@echo "Running $(APP_NAME)..."
	$(BUILD_DIR)/$(APP_NAME)

run-dev: ## Run the application in development mode with hot reload (requires air)
	@echo "Running $(APP_NAME) in development mode..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Install it with: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi

# Testing targets
test: ## Run unit tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Code quality targets
lint: ## Run golangci-lint
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not found. Install it from: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not found. Install it with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-build-dev: ## Build Docker image for development
	@echo "Building Docker development image..."
	docker build -f Dockerfile.dev -t $(APP_NAME):dev .

docker-run: docker-build ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 --name $(APP_NAME) $(DOCKER_IMAGE)

docker-push: docker-build ## Push Docker image to registry
	@echo "Pushing Docker image to registry..."
	docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)

# Docker Compose targets
compose-up: ## Start all services with docker-compose
	@echo "Starting all services..."
	docker-compose -f $(COMPOSE_FILE) up -d

compose-up-build: ## Start all services and rebuild images
	@echo "Starting all services with build..."
	docker-compose -f $(COMPOSE_FILE) up -d --build

compose-dev: ## Start development environment
	@echo "Starting development environment..."
	@if [ -f $(COMPOSE_DEV_FILE) ]; then \
		docker-compose -f $(COMPOSE_FILE) -f $(COMPOSE_DEV_FILE) up -d; \
	else \
		docker-compose -f $(COMPOSE_FILE) up -d; \
	fi

compose-test: ## Start test environment
	@echo "Starting test environment..."
	@if [ -f $(COMPOSE_TEST_FILE) ]; then \
		docker-compose -f $(COMPOSE_FILE) -f $(COMPOSE_TEST_FILE) up -d; \
	else \
		echo "Test compose file not found"; \
		exit 1; \
	fi

compose-down: ## Stop all services
	@echo "Stopping all services..."
	docker-compose -f $(COMPOSE_FILE) down

compose-down-volumes: ## Stop all services and remove volumes
	@echo "Stopping all services and removing volumes..."
	docker-compose -f $(COMPOSE_FILE) down -v

compose-logs: ## Show logs from all services
	docker-compose -f $(COMPOSE_FILE) logs -f

compose-logs-api: ## Show logs from orders-api service
	docker-compose -f $(COMPOSE_FILE) logs -f orders-api

compose-restart: ## Restart all services
	@echo "Restarting all services..."
	docker-compose -f $(COMPOSE_FILE) restart

compose-restart-api: ## Restart orders-api service
	@echo "Restarting orders-api service..."
	docker-compose -f $(COMPOSE_FILE) restart orders-api

# Database targets
db-up: ## Start only MongoDB
	@echo "Starting MongoDB..."
	docker-compose -f $(COMPOSE_FILE) up -d mongodb

db-down: ## Stop MongoDB
	@echo "Stopping MongoDB..."
	docker-compose -f $(COMPOSE_FILE) stop mongodb

db-shell: ## Connect to MongoDB shell
	@echo "Connecting to MongoDB shell..."
	docker-compose -f $(COMPOSE_FILE) exec mongodb mongosh cryptosim_orders

db-backup: ## Backup MongoDB database
	@echo "Backing up MongoDB database..."
	@mkdir -p ./backups
	docker-compose -f $(COMPOSE_FILE) exec -T mongodb mongodump --db cryptosim_orders --out /tmp/backup
	docker-compose -f $(COMPOSE_FILE) exec -T mongodb tar -czf /tmp/backup.tar.gz -C /tmp backup
	docker cp $(shell docker-compose -f $(COMPOSE_FILE) ps -q mongodb):/tmp/backup.tar.gz ./backups/mongodb-backup-$(shell date +%Y%m%d-%H%M%S).tar.gz

db-restore: ## Restore MongoDB database from backup
	@echo "Restoring MongoDB database..."
	@read -p "Enter backup file path: " backup_file; \
	if [ -f "$$backup_file" ]; then \
		docker cp "$$backup_file" $(shell docker-compose -f $(COMPOSE_FILE) ps -q mongodb):/tmp/restore.tar.gz; \
		docker-compose -f $(COMPOSE_FILE) exec mongodb tar -xzf /tmp/restore.tar.gz -C /tmp; \
		docker-compose -f $(COMPOSE_FILE) exec mongodb mongorestore --db cryptosim_orders --drop /tmp/backup/cryptosim_orders; \
	else \
		echo "Backup file not found: $$backup_file"; \
		exit 1; \
	fi

# RabbitMQ targets
rabbitmq-up: ## Start only RabbitMQ
	@echo "Starting RabbitMQ..."
	docker-compose -f $(COMPOSE_FILE) up -d rabbitmq

rabbitmq-down: ## Stop RabbitMQ
	@echo "Stopping RabbitMQ..."
	docker-compose -f $(COMPOSE_FILE) stop rabbitmq

rabbitmq-management: ## Open RabbitMQ management interface
	@echo "Opening RabbitMQ management interface..."
	@echo "URL: http://localhost:15672"
	@echo "Username: guest"
	@echo "Password: guest"

# Monitoring targets
monitoring-up: ## Start monitoring services (Prometheus, Grafana)
	@echo "Starting monitoring services..."
	docker-compose -f $(COMPOSE_FILE) up -d prometheus grafana

monitoring-down: ## Stop monitoring services
	@echo "Stopping monitoring services..."
	docker-compose -f $(COMPOSE_FILE) stop prometheus grafana

grafana-dashboard: ## Open Grafana dashboard
	@echo "Opening Grafana dashboard..."
	@echo "URL: http://localhost:3000"
	@echo "Username: admin"
	@echo "Password: admin"

prometheus-ui: ## Open Prometheus UI
	@echo "Opening Prometheus UI..."
	@echo "URL: http://localhost:9090"

# Health check targets
health-check: ## Check application health
	@echo "Checking application health..."
	curl -f http://localhost:8080/health || exit 1

health-check-all: ## Check health of all services
	@echo "Checking health of all services..."
	@echo "Orders API:"
	curl -s http://localhost:8080/health | grep -q "healthy" && echo "✅ Orders API is healthy" || echo "❌ Orders API is unhealthy"
	@echo "MongoDB:"
	docker-compose -f $(COMPOSE_FILE) exec mongodb mongosh --eval "db.adminCommand('ping')" >/dev/null 2>&1 && echo "✅ MongoDB is healthy" || echo "❌ MongoDB is unhealthy"
	@echo "RabbitMQ:"
	docker-compose -f $(COMPOSE_FILE) exec rabbitmq rabbitmq-diagnostics ping >/dev/null 2>&1 && echo "✅ RabbitMQ is healthy" || echo "❌ RabbitMQ is unhealthy"

# Utility targets
clean: ## Clean build artifacts and temporary files
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -cache
	go clean -testcache
	docker system prune -f

clean-all: ## Clean everything including Docker images and volumes
	@echo "Cleaning everything..."
	$(MAKE) clean
	docker-compose -f $(COMPOSE_FILE) down -v --rmi all
	docker system prune -af --volumes

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest

generate: ## Generate code (mocks, swagger docs, etc.)
	@echo "Generating code..."
	go generate ./...

# Environment setup
env-setup: ## Setup development environment
	@echo "Setting up development environment..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from .env.example"; \
		echo "Please update the values in .env file"; \
	fi

# CI/CD targets
ci-test: deps lint vet test ## Run all CI tests
	@echo "All CI tests passed!"

ci-build: deps build docker-build ## Run CI build
	@echo "CI build completed!"

# Release targets
release-prepare: ## Prepare for release
	@echo "Preparing for release..."
	$(MAKE) clean
	$(MAKE) deps
	$(MAKE) test
	$(MAKE) lint
	$(MAKE) build
	$(MAKE) docker-build

release-tag: ## Tag current commit for release
	@read -p "Enter version (e.g., v1.0.0): " version; \
	git tag -a $$version -m "Release $$version"; \
	echo "Tagged release: $$version"

# Documentation
docs-serve: ## Serve documentation locally
	@echo "Serving documentation..."
	@if command -v mkdocs >/dev/null 2>&1; then \
		mkdocs serve; \
	else \
		echo "mkdocs not found. Install it with: pip install mkdocs"; \
		exit 1; \
	fi

# Show current version
version: ## Show current version
	@echo "Current version: $(shell git describe --tags --always --dirty)"

# Show service status
status: ## Show status of all services
	@echo "Service status:"
	docker-compose -f $(COMPOSE_FILE) ps