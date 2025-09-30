#!/bin/bash

# ═══════════════════════════════════════════════════════════
# GASK Smart Startup Script
# Automatically finds available ports and starts services
# ═══════════════════════════════════════════════════════════

set -e  # Exit on error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
DEFAULT_API_PORT=7890
DEFAULT_REDIS_PORT=6380
DEFAULT_POSTGRES_PORT=5433
MAX_PORT_ATTEMPTS=100

# Banner
print_banner() {
    echo -e "${BLUE}"
    cat << "EOF"
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║    ██████╗  █████╗ ███████╗██╗  ██╗                     ║
║   ██╔════╝ ██╔══██╗██╔════╝██║ ██╔╝                     ║
║   ██║  ███╗███████║███████╗█████╔╝                      ║
║   ██║   ██║██╔══██║╚════██║██╔═██╗                      ║
║   ╚██████╔╝██║  ██║███████║██║  ██╗                     ║
║    ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝                     ║
║                                                           ║
║   Smart Startup Script v2.0                              ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# Logging functions
log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check if port is available
is_port_available() {
    local port=$1
    if command -v nc >/dev/null 2>&1; then
        ! nc -z localhost "$port" >/dev/null 2>&1
    elif command -v netstat >/dev/null 2>&1; then
        ! netstat -tuln | grep -q ":$port "
    elif command -v ss >/dev/null 2>&1; then
        ! ss -tuln | grep -q ":$port "
    else
        # Fallback: try to listen on port
        (echo >/dev/tcp/localhost/"$port") >/dev/null 2>&1 && return 1 || return 0
    fi
}

# Find available port
find_available_port() {
    local start_port=$1
    local max_attempts=${2:-$MAX_PORT_ATTEMPTS}
    
    for ((i=0; i<max_attempts; i++)); do
        local port=$((start_port + i))
        if is_port_available "$port"; then
            echo "$port"
            return 0
        fi
    done
    
    return 1
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing=0
    
    # Check Docker
    if command -v docker >/dev/null 2>&1; then
        log_success "Docker is installed"
    else
        log_error "Docker is not installed!"
        missing=1
    fi
    
    # Check Docker Compose
    if command -v docker-compose >/dev/null 2>&1; then
        log_success "Docker Compose is installed"
    else
        log_error "Docker Compose is not installed!"
        missing=1
    fi
    
    # Check if Docker is running
    if docker info >/dev/null 2>&1; then
        log_success "Docker daemon is running"
    else
        log_error "Docker daemon is not running!"
        missing=1
    fi
    
    if [ $missing -eq 1 ]; then
        log_error "Please install missing prerequisites"
        exit 1
    fi
}

# Setup environment
setup_environment() {
    log_info "Setting up environment..."
    
    # Check if .env exists
    if [ ! -f .env ]; then
        if [ -f .env.example ]; then
            log_warning ".env not found, creating from .env.example"
            cp .env.example .env
            log_success "Created .env file"
        else
            log_error ".env.example not found!"
            exit 1
        fi
    else
        log_success ".env file exists"
    fi
    
    # Source .env
    export $(grep -v '^#' .env | xargs)
}

# Find and configure ports
configure_ports() {
    log_info "Configuring ports..."
    
    # API Port
    local api_port=${API_PORT:-$DEFAULT_API_PORT}
    if ! is_port_available "$api_port"; then
        log_warning "Port $api_port is busy, finding alternative..."
        api_port=$(find_available_port "$api_port")
        if [ $? -eq 0 ]; then
            log_success "Found available API port: $api_port"
            # Update .env
            sed -i.bak "s/^API_PORT=.*/API_PORT=$api_port/" .env
        else
            log_error "Could not find available port for API"
            exit 1
        fi
    else
        log_success "API port $api_port is available"
    fi
    
    # Redis Port
    local redis_port=${REDIS_PORT:-$DEFAULT_REDIS_PORT}
    if ! is_port_available "$redis_port"; then
        log_warning "Port $redis_port is busy, finding alternative..."
        redis_port=$(find_available_port "$redis_port")
        if [ $? -eq 0 ]; then
            log_success "Found available Redis port: $redis_port"
            sed -i.bak "s/^REDIS_PORT=.*/REDIS_PORT=$redis_port/" .env
        else
            log_error "Could not find available port for Redis"
            exit 1
        fi
    else
        log_success "Redis port $redis_port is available"
    fi
    
    # PostgreSQL Port
    local postgres_port=${POSTGRES_PORT:-$DEFAULT_POSTGRES_PORT}
    if ! is_port_available "$postgres_port"; then
        log_warning "Port $postgres_port is busy, finding alternative..."
        postgres_port=$(find_available_port "$postgres_port")
        if [ $? -eq 0 ]; then
            log_success "Found available PostgreSQL port: $postgres_port"
            sed -i.bak "s/^POSTGRES_PORT=.*/POSTGRES_PORT=$postgres_port/" .env
        else
            log_error "Could not find available port for PostgreSQL"
            exit 1
        fi
    else
        log_success "PostgreSQL port $postgres_port is available"
    fi
    
    # Export ports
    export API_PORT=$api_port
    export REDIS_PORT=$redis_port
    export POSTGRES_PORT=$postgres_port
    
    # Show configuration
    echo ""
    log_info "Port Configuration:"
    echo -e "  ${BLUE}API:${NC}        localhost:${GREEN}$api_port${NC}"
    echo -e "  ${BLUE}Redis:${NC}      localhost:${GREEN}$redis_port${NC}"
    echo -e "  ${BLUE}PostgreSQL:${NC} localhost:${GREEN}$postgres_port${NC}"
    echo ""
}

# Start services
start_services() {
    log_info "Starting GASK services..."
    
    # Build images
    log_info "Building Docker images..."
    docker-compose build || {
        log_error "Failed to build images"
        exit 1
    }
    
    # Start services
    log_info "Starting containers..."
    docker-compose up -d || {
        log_error "Failed to start services"
        exit 1
    }
    
    log_success "Services started"
}

# Wait for services to be healthy
wait_for_services() {
    log_info "Waiting for services to be healthy..."
    
    local max_wait=60
    local waited=0
    
    while [ $waited -lt $max_wait ]; do
        if docker-compose ps | grep -q "(healthy)"; then
            log_success "Services are healthy"
            return 0
        fi
        
        echo -n "."
        sleep 2
        waited=$((waited + 2))
    done
    
    echo ""
    log_warning "Services did not become healthy within ${max_wait}s"
    log_info "Checking service status..."
    docker-compose ps
}

# Test connectivity
test_connectivity() {
    log_info "Testing connectivity..."
    
    # Test API
    local api_url="http://localhost:${API_PORT}/health"
    if curl -s "$api_url" | grep -q "healthy"; then
        log_success "API is responding"
    else
        log_warning "API is not responding yet"
    fi
    
    # Test Redis
    if docker exec gaskRedis redis-cli ping >/dev/null 2>&1; then
        log_success "Redis is responding"
    else
        log_warning "Redis is not responding yet"
    fi
    
    # Test PostgreSQL
    if docker exec gaskPostgres pg_isready -U airflow >/dev/null 2>&1; then
        log_success "PostgreSQL is responding"
    else
        log_warning "PostgreSQL is not responding yet"
    fi
}

# Show summary
show_summary() {
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}  ${YELLOW}GASK is running successfully!${NC}                         ${GREEN}║${NC}"
    echo -e "${GREEN}╠═══════════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║${NC}                                                           ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  ${CYAN}API:${NC}        http://localhost:${GREEN}${API_PORT}${NC}                      ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  ${CYAN}Health:${NC}     http://localhost:${GREEN}${API_PORT}${NC}/health              ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  ${CYAN}Admin:${NC}      http://localhost:${GREEN}${API_PORT}${NC}/admin/status        ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}                                                           ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  ${CYAN}Redis:${NC}      localhost:${GREEN}${REDIS_PORT}${NC}                          ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  ${CYAN}PostgreSQL:${NC} localhost:${GREEN}${POSTGRES_PORT}${NC}                       ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}                                                           ${GREEN}║${NC}"
    echo -e "${GREEN}╠═══════════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║${NC}  ${BLUE}Useful Commands:${NC}                                      ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}    make logs       - View logs                            ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}    make ps         - Show containers                      ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}    make health     - Check health                         ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}    make down       - Stop services                        ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}    make help       - Show all commands                    ${GREEN}║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Cleanup on error
cleanup_on_error() {
    log_error "Startup failed!"
    log_info "Cleaning up..."
    docker-compose down 2>/dev/null || true
    exit 1
}

# Main execution
main() {
    # Set trap for errors
    trap cleanup_on_error ERR
    
    print_banner
    
    check_prerequisites
    setup_environment
    configure_ports
    start_services
    wait_for_services
    test_connectivity
    show_summary
}

# Run main function
main "$@"