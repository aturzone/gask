.PHONY: build test clean run docker-build docker-run migrate lint

APP_NAME := taskmaster
VERSION := v1.0.0
BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

build:
	@echo "Building $(APP_NAME)..."
	@go build $(LDFLAGS) -o bin/$(APP_NAME) cmd/api/main.go

test:
	@echo "Running tests..."
	@go test -race -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf bin/

run:
	@echo "Running $(APP_NAME)..."
	@go run cmd/api/main.go serve

migrate-up:
	@echo "Running migrations..."
	@go run cmd/api/main.go migrate up

migrate-down:
	@echo "Rolling back migrations..."
	@go run cmd/api/main.go migrate down

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) .
	@docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

docker-run:
	@echo "Running with Docker Compose..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose logs -f api

create-admin:
	@echo "Creating admin user..."
	@docker-compose exec api ./taskmaster user create --email admin@taskmaster.dev --password admin123 --role admin --first-name Admin --last-name User

dev-setup:
	@echo "Setting up development environment..."
	@go mod download
	@cp .env.example .env || true
	@echo "Development environment setup complete!"

help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  test          - Run tests"
	@echo "  run           - Run the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  migrate-up    - Run database migrations"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  docker-stop   - Stop Docker containers"
	@echo "  create-admin  - Create admin user"
	@echo "  dev-setup     - Setup development environment"
	@echo "  help          - Show this help message"
