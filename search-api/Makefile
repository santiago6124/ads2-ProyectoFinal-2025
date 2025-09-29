# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=search-api
BINARY_UNIX=$(BINARY_NAME)_unix

# Build info
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Ldflags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commit=$(COMMIT) -w -s"

.PHONY: all build clean test coverage deps help run docker-build docker-run lint format

# Default target
all: test build

## Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v ./cmd/server

## Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) -v ./cmd/server

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

## Run tests
test:
	$(GOTEST) -v ./...

## Run tests with race detection
test-race:
	$(GOTEST) -race -short ./...

## Generate test coverage
coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

## Tidy dependencies
tidy:
	$(GOMOD) tidy

## Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server
	./$(BINARY_NAME)

## Run with hot reload (requires air)
dev:
	air

## Lint the code
lint:
	golangci-lint run

## Format the code
format:
	gofmt -s -w .
	$(GOCMD) mod tidy

## Security scan
security:
	gosec ./...

## Build Docker image
docker-build:
	docker build -t search-api:$(VERSION) .
	docker tag search-api:$(VERSION) search-api:latest

## Run with Docker Compose
docker-run:
	docker-compose up -d

## Stop Docker Compose
docker-stop:
	docker-compose down

## View Docker Compose logs
docker-logs:
	docker-compose logs -f

## Start only infrastructure (Solr, Memcached, RabbitMQ)
infra-up:
	docker-compose up -d solr memcached rabbitmq

## Stop infrastructure
infra-down:
	docker-compose down

## Restart the application service
app-restart:
	docker-compose restart search-api

## Check application health
health:
	curl -f http://localhost:8080/api/v1/health || echo "Health check failed"

## Run performance test
perf-test:
	@echo "Running basic performance test..."
	@for i in {1..100}; do \
		curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/health; \
	done | sort | uniq -c

## Install development tools
install-tools:
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/go-cryptocheck/cmd/gosec@latest

## Generate API documentation
docs:
	@echo "API documentation available at: http://localhost:8080/api/v1/health"
	@echo "Solr Admin: http://localhost:8983/solr/#/"
	@echo "RabbitMQ Management: http://localhost:15672/"

## Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Variables for compose profiles
COMPOSE_PROFILES ?= ""

## Start with monitoring stack
monitoring-up:
	COMPOSE_PROFILES=monitoring docker-compose --profile monitoring up -d

## View all service status
status:
	docker-compose ps

## Reset everything (careful!)
reset: docker-stop
	docker-compose down -v
	docker system prune -f
	docker volume prune -f