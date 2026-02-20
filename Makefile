.PHONY: help dev build test clean install deps lint fmt vet sec server

# Variables
APP_NAME=concord
SERVER_NAME=concord-server
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_DIR=./build
FRONTEND_DIR=./frontend
GO_FILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Colors for output
BLUE=\033[0;34m
GREEN=\033[0;32m
RED=\033[0;31m
NC=\033[0m # No Color

help: ## Show this help message
	@echo '$(BLUE)Concord - Development Makefile$(NC)'
	@echo ''
	@echo 'Usage:'
	@echo '  make $(GREEN)<target>$(NC)'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install all dependencies (Go + Node)
	@echo "$(BLUE)Installing Go dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(BLUE)Installing frontend dependencies...$(NC)"
	cd $(FRONTEND_DIR) && npm install
	@echo "$(GREEN)Dependencies installed successfully$(NC)"

dev: ## Run development server with hot reload
	@echo "$(BLUE)Starting Concord in development mode...$(NC)"
	wails dev

build: ## Build production binaries
	@echo "$(BLUE)Building Concord desktop app...$(NC)"
	wails build -clean -upx
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/bin/$(APP_NAME)$(NC)"

build-server: ## Build central server binary
	@echo "$(BLUE)Building Concord server...$(NC)"
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(SERVER_NAME) ./cmd/server
	@echo "$(GREEN)Server build complete: $(BUILD_DIR)/$(SERVER_NAME)$(NC)"

test: ## Run all tests
	@echo "$(BLUE)Running Go tests...$(NC)"
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "$(BLUE)Running frontend tests...$(NC)"
	cd $(FRONTEND_DIR) && npm test
	@echo "$(GREEN)All tests passed$(NC)"

test-unit: ## Run only unit tests
	@echo "$(BLUE)Running unit tests...$(NC)"
	go test -v -short -race ./...

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	go test -v -run Integration ./...

test-coverage: ## Generate test coverage report
	@echo "$(BLUE)Generating coverage report...$(NC)"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

lint: ## Run linters
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	golangci-lint run ./...
	@echo "$(BLUE)Running frontend linter...$(NC)"
	cd $(FRONTEND_DIR) && npm run lint

fmt: ## Format code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	gofmt -s -w $(GO_FILES)
	goimports -w $(GO_FILES)
	@echo "$(BLUE)Formatting frontend code...$(NC)"
	cd $(FRONTEND_DIR) && npm run format
	@echo "$(GREEN)Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	go vet ./...

sec: ## Run security scan
	@echo "$(BLUE)Running security scan with govulncheck...$(NC)"
	govulncheck ./...
	@echo "$(GREEN)Security scan complete$(NC)"

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

install: build ## Install the application
	@echo "$(BLUE)Installing $(APP_NAME)...$(NC)"
	cp $(BUILD_DIR)/bin/$(APP_NAME) /usr/local/bin/
	@echo "$(GREEN)Installation complete$(NC)"

server: build-server ## Run the central server
	@echo "$(BLUE)Starting Concord server...$(NC)"
	$(BUILD_DIR)/$(SERVER_NAME)

docker-build: ## Build Docker image for server
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -f deployments/docker/Dockerfile.server -t concord-server:$(VERSION) .
	@echo "$(GREEN)Docker image built: concord-server:$(VERSION)$(NC)"

docker-compose: ## Start full stack with docker-compose
	@echo "$(BLUE)Starting Concord stack with docker-compose...$(NC)"
	cd deployments/docker && docker-compose up -d
	@echo "$(GREEN)Stack started$(NC)"

generate: ## Generate code (bindings, mocks, etc.)
	@echo "$(BLUE)Running go generate...$(NC)"
	go generate ./...
	@echo "$(GREEN)Code generation complete$(NC)"

migrate-up: ## Run database migrations up
	@echo "$(BLUE)Running migrations...$(NC)"
	go run cmd/concord/main.go migrate up
	@echo "$(GREEN)Migrations complete$(NC)"

migrate-down: ## Run database migrations down
	@echo "$(BLUE)Rolling back migrations...$(NC)"
	go run cmd/concord/main.go migrate down
	@echo "$(GREEN)Rollback complete$(NC)"

# Default target
.DEFAULT_GOAL := help
