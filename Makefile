.PHONY: help build up down restart logs clean ps health test dev prod status

# ═══════════════════════════════════════════════════════════
# GASK - Makefile
# ═══════════════════════════════════════════════════════════

# Colors
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
BLUE   := \033[0;34m
NC     := \033[0m

# Variables
COMPOSE := docker-compose
APP_NAME := gask

help: ## Show this help message
	@echo "$(BLUE)╔═══════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║$(NC)  $(GREEN)GASK - Go-based Advanced taSK management system$(NC)    $(BLUE)║$(NC)"
	@echo "$(BLUE)╚═══════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Available commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

# ┌─────────────────────────────────────────────────────────┐
# │ Setup & Configuration                                   │
# └─────────────────────────────────────────────────────────┘

setup: ## Initial setup - create .env file
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env file...$(NC)"; \
		cp .env.example .env; \
		echo "$(GREEN)✓ .env created. Please edit it with your values.$(NC)"; \
	else \
		echo "$(RED)✗ .env already exists.$(NC)"; \
	fi

check-env: ## Check environment configuration
	@echo "$(YELLOW)Checking environment...$(NC)"
	@if [ -f .env ]; then \
		echo "$(GREEN)✓ .env file exists$(NC)"; \
		grep -v '^#' .env | grep -v '^$$' | head -10; \
	else \
		echo "$(RED)✗ .env file missing! Run 'make setup'$(NC)"; \
		exit 1; \
	fi

# ┌─────────────────────────────────────────────────────────┐
# │ Docker Operations                                        │
# └─────────────────────────────────────────────────────────┘

build: ## Build Docker images
	@echo "$(YELLOW)Building GASK images...$(NC)"
	@$(COMPOSE) build --no-cache
	@echo "$(GREEN)✓ Build complete!$(NC)"

up: ## Start all services
	@echo "$(YELLOW)Starting GASK services...$(NC)"
	@$(COMPOSE) up -d
	@sleep 5
	@$(MAKE) status
	@echo ""
	@echo "$(GREEN)✓ GASK is running!$(NC)"
	@echo "$(BLUE)  API: http://localhost:$$(grep API_PORT .env | cut -d '=' -f2)$(NC)"

down: ## Stop all services
	@echo "$(YELLOW)Stopping GASK services...$(NC)"
	@$(COMPOSE) down
	@echo "$(GREEN)✓ Services stopped$(NC)"

restart: down up ## Restart all services

# ┌─────────────────────────────────────────────────────────┐
# │ Monitoring & Logs                                        │
# └─────────────────────────────────────────────────────────┘

logs: ## Show logs from all services
	@$(COMPOSE) logs -f

logs-api: ## Show API logs only
	@$(COMPOSE) logs -f gaskMain

logs-redis: ## Show Redis logs only
	@$(COMPOSE) logs -f gaskRedis

logs-postgres: ## Show PostgreSQL logs only
	@$(COMPOSE) logs -f gaskPostgres

ps: ## Show running containers
	@echo "$(YELLOW)GASK Services:$(NC)"
	@$(COMPOSE) ps
	@echo ""
	@echo "$(YELLOW)Resource Usage:$(NC)"
	@docker stats --no-stream $$($(COMPOSE) ps -q) 2>/dev/null || true

status: ## Check status of all services
	@echo "$(YELLOW)Service Status:$(NC)"
	@$(COMPOSE) ps
	@echo ""
	@$(MAKE) health

health: ## Check health of all services
	@echo "$(YELLOW)Health Checks:$(NC)"
	@echo ""
	@echo -n "$(BLUE)API:$(NC)        "
	@curl -s http://localhost:$$(grep API_PORT .env | cut -d '=' -f2)/health | jq -r '.status' 2>/dev/null || echo "$(RED)✗ Not responding$(NC)"
	@echo -n "$(BLUE)Redis:$(NC)      "
	@docker exec gaskRedis redis-cli ping 2>/dev/null || echo "$(RED)✗ Not responding$(NC)"
	@echo -n "$(BLUE)PostgreSQL:$(NC) "
	@docker exec gaskPostgres pg_isready -U airflow 2>/dev/null | grep -q "accepting" && echo "$(GREEN)✓ Ready$(NC)" || echo "$(RED)✗ Not ready$(NC)"

# ┌─────────────────────────────────────────────────────────┐
# │ Development                                              │
# └─────────────────────────────────────────────────────────┘

dev: ## Start in development mode (with live logs)
	@echo "$(YELLOW)Starting GASK in development mode...$(NC)"
	@$(COMPOSE) up --build

shell-api: ## Open shell in API container
	@docker exec -it gaskMain sh

shell-redis: ## Open Redis CLI
	@docker exec -it gaskRedis redis-cli

shell-postgres: ## Open PostgreSQL shell
	@docker exec -it gaskPostgres psql -U airflow -d airflow

# ┌─────────────────────────────────────────────────────────┐
# │ Testing                                                  │
# └─────────────────────────────────────────────────────────┘

test: ## Run API tests
	@echo "$(YELLOW)Running GASK tests...$(NC)"
	@chmod +x test_api.sh
	@./test_api.sh

# ┌─────────────────────────────────────────────────────────┐
# │ Backup & Restore                                         │
# └─────────────────────────────────────────────────────────┘

backup: ## Backup PostgreSQL database
	@echo "$(YELLOW)Creating backup...$(NC)"
	@mkdir -p backups
	@docker exec gaskPostgres pg_dump -U airflow airflow > backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)✓ Backup created in backups/$(NC)"

restore: ## Restore PostgreSQL (use FILE=path/to/backup.sql)
	@if [ -z "$(FILE)" ]; then \
		echo "$(RED)✗ Specify backup file: make restore FILE=backups/backup.sql$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Restoring from $(FILE)...$(NC)"
	@docker exec -i gaskPostgres psql -U airflow airflow < $(FILE)
	@echo "$(GREEN)✓ Restore complete$(NC)"

backup-redis: ## Backup Redis data
	@echo "$(YELLOW)Creating Redis backup...$(NC)"
	@mkdir -p backups
	@docker exec gaskRedis redis-cli BGSAVE
	@sleep 2
	@docker cp gaskRedis:/data/dump.rdb backups/redis_$$(date +%Y%m%d_%H%M%S).rdb
	@echo "$(GREEN)✓ Redis backup created$(NC)"

# ┌─────────────────────────────────────────────────────────┐
# │ Maintenance                                              │
# └─────────────────────────────────────────────────────────┘

clean: ## Remove containers and networks (keeps volumes)
	@echo "$(YELLOW)Cleaning up...$(NC)"
	@$(COMPOSE) down
	@echo "$(GREEN)✓ Cleanup complete$(NC)"

clean-all: ## Remove everything including volumes (⚠️ DATA LOSS!)
	@echo "$(RED)⚠️  This will DELETE ALL DATA! Press Ctrl+C to cancel.$(NC)"
	@sleep 5
	@$(COMPOSE) down -v
	@docker volume rm gask_redis_data gask_postgres_data gask_logs 2>/dev/null || true
	@echo "$(GREEN)✓ All data removed$(NC)"

prune: ## Remove unused Docker resources
	@echo "$(YELLOW)Pruning Docker resources...$(NC)"
	@docker system prune -f
	@echo "$(GREEN)✓ Prune complete$(NC)"

update: ## Pull latest images and rebuild
	@echo "$(YELLOW)Updating GASK...$(NC)"
	@$(COMPOSE) pull
	@$(COMPOSE) build --no-cache
	@echo "$(GREEN)✓ Update complete$(NC)"

# ┌─────────────────────────────────────────────────────────┐
# │ Production                                               │
# └─────────────────────────────────────────────────────────┘

prod: check-env build up ## Deploy for production
	@echo ""
	@echo "$(GREEN)╔═══════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(GREEN)║$(NC)  $(YELLOW)GASK deployed successfully!$(NC)                           $(GREEN)║$(NC)"
	@echo "$(GREEN)╠═══════════════════════════════════════════════════════════╣$(NC)"
	@echo "$(GREEN)║$(NC)  API: http://localhost:$$(grep API_PORT .env | cut -d '=' -f2)                           $(GREEN)║$(NC)"
	@echo "$(GREEN)║$(NC)  Health: http://localhost:$$(grep API_PORT .env | cut -d '=' -f2)/health                   $(GREEN)║$(NC)"
	@echo "$(GREEN)╚═══════════════════════════════════════════════════════════╝$(NC)"

stop-prod: ## Stop production deployment
	@$(MAKE) down

# ┌─────────────────────────────────────────────────────────┐
# │ Advanced                                                 │
# └─────────────────────────────────────────────────────────┘

inspect: ## Inspect containers and networks
	@echo "$(YELLOW)Container Inspection:$(NC)"
	@docker inspect gaskMain gaskRedis gaskPostgres | jq '.[].Name, .[].State.Status, .[].NetworkSettings.Networks'

network: ## Show network information
	@echo "$(YELLOW)Network Information:$(NC)"
	@docker network inspect gask_network

volumes: ## Show volume information
	@echo "$(YELLOW)Volume Information:$(NC)"
	@docker volume ls | grep gask

stats: ## Show detailed resource usage
	@docker stats --no-stream $$($(COMPOSE) ps -q)

monitor: ## Real-time monitoring (press Ctrl+C to exit)
	@watch -n 2 '$(MAKE) ps && echo "" && $(MAKE) health'