# TaskMaster Core API Makefile

# Variables
BINARY_NAME=taskmaster
BUILD_DIR=build
MAIN_PATH=./cmd/api
MIGRATION_PATH=migrations

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
BUILD_FLAGS=-ldflags="-s -w"

.PHONY: help build clean test deps fmt lint run dev migrate-up migrate-down migrate-create docker-build docker-run

# Default target
help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build commands
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-linux: ## Build for Linux
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_PATH)

build-windows: ## Build for Windows
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe $(MAIN_PATH)

build-all: build-linux build-windows ## Build for all platforms
	@echo "All builds complete"

# Development commands
run: ## Run the application in development mode
	@echo "Running $(BINARY_NAME) in development mode..."
	$(GOCMD) run $(MAIN_PATH) serve

dev: ## Run the application with live reload (requires air)
	@echo "Starting development server with live reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		make run; \
	fi

# Testing commands
test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Database migration commands
migrate-up: ## Run database migrations up
	@echo "Running database migrations up..."
	$(BUILD_DIR)/$(BINARY_NAME) migrate up

migrate-down: ## Run database migrations down
	@echo "Running database migrations down..."
	$(BUILD_DIR)/$(BINARY_NAME) migrate down

migrate-version: ## Show current migration version
	@echo "Checking migration version..."
	$(BUILD_DIR)/$(BINARY_NAME) migrate version

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Please provide a migration name: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)"
	migrate create -ext sql -dir $(MIGRATION_PATH) -seq $(NAME)

# User management
create-admin: ## Create admin user
	@echo "Creating admin user..."
	$(BUILD_DIR)/$(BINARY_NAME) user create

# Code quality commands
fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) ./...

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) tidy

# Clean commands
clean: ## Clean build files
	@echo "Cleaning build files..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t taskmaster-api:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env taskmaster-api:latest

docker-dev: ## Run development environment with Docker Compose
	@echo "Starting development environment..."
	docker-compose up -d

docker-dev-down: ## Stop development environment
	@echo "Stopping development environment..."
	docker-compose down

# Database setup
db-setup: ## Setup database (PostgreSQL via Docker)
	@echo "Setting up PostgreSQL database..."
	docker run --name taskmaster-postgres -e POSTGRES_DB=taskmaster -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15

db-stop: ## Stop database container
	@echo "Stopping database container..."
	docker stop taskmaster-postgres

db-remove: ## Remove database container
	@echo "Removing database container..."
	docker rm taskmaster-postgres

# Installation commands
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest

# Swagger documentation
swagger: ## Generate Swagger documentation
	@echo "Generating Swagger documentation..."
	@if command -v swag > /dev/null; then \
		swag init -g cmd/api/main.go -o docs; \
	else \
		echo "swag not found. Install with: make install-tools"; \
	fi

# Complete setup
setup: deps install-tools db-setup build migrate-up create-admin ## Complete setup for development
	@echo "Setup complete! You can now run 'make run' to start the server"

# Production build
production: clean build-all ## Build for production
	@echo "Production build complete"
