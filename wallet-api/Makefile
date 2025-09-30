# Wallet API Makefile

# Variables
BINARY_NAME=wallet-api
DOCKER_IMAGE=wallet-api:latest
DOCKER_COMPOSE=docker-compose.yml

# Go commands
.PHONY: build
build:
	go build -o bin/$(BINARY_NAME) ./cmd/server

.PHONY: run
run:
	go run ./cmd/server

.PHONY: test
test:
	go test -v -race ./...

.PHONY: test-coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: clean
clean:
	go clean
	rm -f bin/$(BINARY_NAME)
	rm -f coverage.out coverage.html

# Docker commands
.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE)

.PHONY: docker-compose-up
docker-compose-up:
	docker-compose -f $(DOCKER_COMPOSE) up -d

.PHONY: docker-compose-down
docker-compose-down:
	docker-compose -f $(DOCKER_COMPOSE) down

.PHONY: docker-compose-logs
docker-compose-logs:
	docker-compose -f $(DOCKER_COMPOSE) logs -f

.PHONY: docker-compose-restart
docker-compose-restart:
	docker-compose -f $(DOCKER_COMPOSE) restart

# Development commands
.PHONY: dev-setup
dev-setup:
	go mod download
	go mod tidy

.PHONY: dev-services
dev-services:
	docker-compose up -d mongodb redis rabbitmq

.PHONY: dev-stop
dev-stop:
	docker-compose down

# Database commands
.PHONY: db-migrate
db-migrate:
	go run ./cmd/migrate

.PHONY: db-seed
db-seed:
	go run ./cmd/seed

# Monitoring
.PHONY: prometheus
prometheus:
	docker-compose up -d prometheus

.PHONY: grafana
grafana:
	docker-compose up -d grafana

# Production commands
.PHONY: deploy
deploy: docker-build
	docker-compose -f docker-compose.yml up -d

.PHONY: logs
logs:
	docker-compose logs -f wallet-api

.PHONY: health
health:
	curl -f http://localhost:8080/health || exit 1

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build            - Build the application"
	@echo "  run              - Run the application"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  clean            - Clean build artifacts"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  docker-compose-up    - Start all services"
	@echo "  docker-compose-down  - Stop all services"
	@echo "  docker-compose-logs  - View logs"
	@echo ""
	@echo "Development:"
	@echo "  dev-setup        - Setup development environment"
	@echo "  dev-services     - Start development services"
	@echo "  dev-stop         - Stop development services"
	@echo ""
	@echo "Production:"
	@echo "  deploy           - Deploy to production"
	@echo "  health           - Check application health"