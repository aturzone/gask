.PHONY: help build up down restart logs clean ps health test

# Colors for terminal output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
NC     := \033[0m # No Color

help: ## Show this help message
	@echo "$(GREEN)Task Manager API - Docker Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

setup: ## Initial setup (create .env from example)
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env file from .env.example...$(NC)"; \
		cp .env.example .env; \
		echo "$(GREEN)✓ .env file created. Please edit it with your values.$(NC)"; \
	else \
		echo "$(RED)✗ .env file already exists.$(NC)"; \
	fi

build: ## Build all Docker images
	@echo "$(YELLOW)Building Docker images...$(NC)"
	docker-compose build
	@echo "$(GREEN)✓ Build complete!$(NC)"

up: ## Start all services
	@echo "$(YELLOW)Starting services...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)✓ Services started!$(NC)"
	@echo "$(YELLOW)API available at: http://localhost:7890$(NC)"
	@echo "$(YELLOW)Run 'make logs' to see logs$(NC)"

down: ## Stop all services
	@echo "$(YELLOW)Stopping services...$(NC)"
	docker-compose down
	@echo "$(GREEN)✓ Services stopped!$(NC)"

restart: down up ## Restart all services

logs: ## Show logs from all services
	docker-compose logs -f

logs-api: ## Show logs from API service only
	docker-compose logs -f api

logs-redis: ## Show logs from Redis service only
	docker-compose logs -f redis

logs-postgres: ## Show logs from PostgreSQL service only
	docker-compose logs -f postgres

ps: ## Show running containers
	@docker-compose ps

health: ## Check health of all services
	@echo "$(YELLOW)Checking service health...$(NC)"
	@echo ""
	@echo "$(GREEN)API Health:$(NC)"
	@curl -s http://localhost:7890/health | jq . || echo "$(RED)✗ API is not responding$(NC)"
	@echo ""
	@echo "$(GREEN)Redis Health:$(NC)"
	@docker-compose exec redis redis-cli ping || echo "$(RED)✗ Redis is not responding$(NC)"
	@echo ""
	@echo "$(GREEN)PostgreSQL Health:$(NC)"
	@docker-compose exec postgres pg_isready -U airflow || echo "$(RED)✗ PostgreSQL is not responding$(NC)"

clean: ## Remove all containers, volumes, and networks
	@echo "$(RED)⚠️  This will remove all data! Press Ctrl+C to cancel.$(NC)"
	@sleep 5
	docker-compose down -v
	@echo "$(GREEN)✓ Cleanup complete!$(NC)"

clean-build: clean build up ## Clean, rebuild, and start

shell-api: ## Open shell in API container
	docker-compose exec api sh

shell-redis: ## Open Redis CLI
	docker-compose exec redis redis-cli

shell-postgres: ## Open PostgreSQL shell
	docker-compose exec postgres psql -U airflow -d airflow

test: ## Run API tests
	@echo "$(YELLOW)Running API tests...$(NC)"
	@chmod +x test_api.sh
	@./test_api.sh

backup-postgres: ## Backup PostgreSQL database
	@echo "$(YELLOW)Backing up PostgreSQL database...$(NC)"
	@mkdir -p backups
	docker-compose exec -T postgres pg_dump -U airflow airflow > backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)✓ Backup complete! Saved to backups/$(NC)"

restore-postgres: ## Restore PostgreSQL database (specify file with FILE=path/to/backup.sql)
	@if [ -z "$(FILE)" ]; then \
		echo "$(RED)✗ Please specify backup file: make restore-postgres FILE=backups/backup.sql$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Restoring PostgreSQL database from $(FILE)...$(NC)"
	docker-compose exec -T postgres psql -U airflow airflow < $(FILE)
	@echo "$(GREEN)✓ Restore complete!$(NC)"

stats: ## Show container resource usage
	docker stats --no-stream $$(docker-compose ps -q)

update: ## Pull latest images
	@echo "$(YELLOW)Pulling latest images...$(NC)"
	docker-compose pull
	@echo "$(GREEN)✓ Images updated!$(NC)"

dev: ## Start in development mode with live logs
	docker-compose up --build

prod: build up ## Deploy for production
	@echo "$(GREEN)✓ Production deployment complete!$(NC)"
	@echo "$(YELLOW)API: http://localhost:7890$(NC)"
	@echo "$(YELLOW)Health: http://localhost:7890/health$(NC)"