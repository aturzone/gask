#!/bin/bash

# ═══════════════════════════════════════════════════════════
# GASK Advanced Health Check Script
# Comprehensive health verification with exit codes
# ═══════════════════════════════════════════════════════════

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
TIMEOUT=5
VERBOSE=0
EXIT_ON_ERROR=0

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

API_PORT=${API_PORT:-7890}
OWNER_PASSWORD=${OWNER_PASSWORD:-admin1234}

# Health status
HEALTH_STATUS=0
WARNINGS=0
ERRORS=0

# Functions
log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
    ((WARNINGS++))
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    ((ERRORS++))
    HEALTH_STATUS=1
}

verbose() {
    if [ $VERBOSE -eq 1 ]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# Check Docker
check_docker() {
    log_info "Checking Docker..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        return 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        return 1
    fi
    
    log_success "Docker is running"
    return 0
}

# Check container status
check_container() {
    local container=$1
    local name=$2
    
    verbose "Checking container: $container"
    
    # Check if container exists
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
        log_error "$name container not found"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        log_error "$name container is not running"
        return 1
    fi
    
    # Check health status
    local health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "none")
    
    if [ "$health" = "healthy" ]; then
        log_success "$name is healthy"
        return 0
    elif [ "$health" = "none" ]; then
        log_warning "$name is running (no health check configured)"
        return 0
    elif [ "$health" = "starting" ]; then
        log_warning "$name is starting..."
        return 0
    else
        log_error "$name is unhealthy (status: $health)"
        return 1
    fi
}

# Check API endpoint
check_api_endpoint() {
    local endpoint=$1
    local expected_code=${2:-200}
    local name=$3
    
    verbose "Checking endpoint: $endpoint"
    
    local response=$(curl -s -o /dev/null -w "%{http_code}" \
        --max-time $TIMEOUT \
        "http://localhost:${API_PORT}${endpoint}" 2>/dev/null || echo "000")
    
    if [ "$response" = "$expected_code" ]; then
        log_success "$name endpoint responding ($response)"
        return 0
    elif [ "$response" = "000" ]; then
        log_error "$name endpoint not reachable"
        return 1
    else
        log_warning "$name endpoint returned $response (expected $expected_code)"
        return 0
    fi
}

# Check API with authentication
check_api_auth() {
    local endpoint=$1
    local name=$2
    
    verbose "Checking authenticated endpoint: $endpoint"
    
    local response=$(curl -s -o /dev/null -w "%{http_code}" \
        --max-time $TIMEOUT \
        -H "X-Owner-Password: ${OWNER_PASSWORD}" \
        "http://localhost:${API_PORT}${endpoint}" 2>/dev/null || echo "000")
    
    if [ "$response" = "200" ]; then
        log_success "$name authenticated endpoint working"
        return 0
    elif [ "$response" = "000" ]; then
        log_error "$name endpoint not reachable"
        return 1
    else
        log_warning "$name endpoint returned $response"
        return 0
    fi
}

# Check Redis connectivity
check_redis() {
    log_info "Checking Redis connectivity..."
    
    if docker exec gaskRedis redis-cli ping &> /dev/null; then
        log_success "Redis is responding to PING"
    else
        log_error "Redis is not responding"
        return 1
    fi
    
    # Check Redis info
    local commands=$(docker exec gaskRedis redis-cli INFO stats 2>/dev/null | \
        grep "total_commands_processed" | cut -d: -f2 | tr -d '\r' || echo "0")
    
    verbose "Redis total commands processed: $commands"
    
    return 0
}

# Check PostgreSQL connectivity
check_postgres() {
    log_info "Checking PostgreSQL connectivity..."
    
    if docker exec gaskPostgres pg_isready -U airflow &> /dev/null; then
        log_success "PostgreSQL is accepting connections"
    else
        log_error "PostgreSQL is not accepting connections"
        return 1
    fi
    
    # Check database
    if docker exec gaskPostgres psql -U airflow -d airflow -c "SELECT 1;" &> /dev/null; then
        log_success "PostgreSQL database is accessible"
    else
        log_error "PostgreSQL database is not accessible"
        return 1
    fi
    
    # Check connections
    local connections=$(docker exec gaskPostgres psql -U airflow -d airflow -t \
        -c "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | tr -d ' ' || echo "0")
    
    verbose "PostgreSQL active connections: $connections"
    
    return 0
}

# Check sync status
check_sync() {
    log_info "Checking sync service..."
    
    local status=$(curl -s \
        --max-time $TIMEOUT \
        -H "X-Owner-Password: ${OWNER_PASSWORD}" \
        "http://localhost:${API_PORT}/admin/status" 2>/dev/null)
    
    if [ -z "$status" ]; then
        log_error "Could not retrieve sync status"
        return 1
    fi
    
    # Parse JSON (requires jq)
    if command -v jq &> /dev/null; then
        local running=$(echo "$status" | jq -r '.data.running' 2>/dev/null || echo "false")
        local healthy=$(echo "$status" | jq -r '.data.healthy' 2>/dev/null || echo "false")
        local pending=$(echo "$status" | jq -r '.data.pending_changes' 2>/dev/null || echo "false")
        
        if [ "$running" = "true" ]; then
            log_success "Sync service is running"
        else
            log_warning "Sync service is not running"
        fi
        
        if [ "$healthy" = "true" ]; then
            log_success "Sync service is healthy"
        else
            log_warning "Sync service health check failed"
        fi
        
        if [ "$pending" = "true" ]; then
            verbose "Pending sync changes detected"
        fi
    else
        verbose "jq not installed, skipping detailed sync check"
        log_success "Sync status retrieved"
    fi
    
    return 0
}

# Check resource usage
check_resources() {
    log_info "Checking resource usage..."
    
    # Check disk space
    local disk_usage=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
    
    if [ "$disk_usage" -gt 90 ]; then
        log_warning "Disk usage is high: ${disk_usage}%"
    elif [ "$disk_usage" -gt 80 ]; then
        verbose "Disk usage: ${disk_usage}%"
    else
        log_success "Disk usage is healthy: ${disk_usage}%"
    fi
    
    # Check memory
    if command -v free &> /dev/null; then
        local mem_usage=$(free | grep Mem | awk '{printf("%.0f"), $3/$2 * 100}')
        
        if [ "$mem_usage" -gt 90 ]; then
            log_warning "Memory usage is high: ${mem_usage}%"
        else
            verbose "Memory usage: ${mem_usage}%"
        fi
    fi
    
    return 0
}

# Performance test
check_performance() {
    log_info "Checking API performance..."
    
    local start_time=$(date +%s%3N)
    curl -s -o /dev/null --max-time $TIMEOUT "http://localhost:${API_PORT}/health" &>/dev/null
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ $response_time -lt 100 ]; then
        log_success "API response time: ${response_time}ms (excellent)"
    elif [ $response_time -lt 500 ]; then
        log_success "API response time: ${response_time}ms (good)"
    elif [ $response_time -lt 1000 ]; then
        log_warning "API response time: ${response_time}ms (slow)"
    else
        log_warning "API response time: ${response_time}ms (very slow)"
    fi
    
    return 0
}

# Main health check
run_health_check() {
    echo ""
    echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}    ${CYAN}GASK Health Check${NC}                                     ${BLUE}║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    # Docker checks
    check_docker || true
    echo ""
    
    # Container checks
    log_info "Checking containers..."
    check_container "gaskMain" "API Server" || true
    check_container "gaskRedis" "Redis" || true
    check_container "gaskPostgres" "PostgreSQL" || true
    echo ""
    
    # Service connectivity checks
    check_redis || true
    echo ""
    
    check_postgres || true
    echo ""
    
    # API endpoint checks
    log_info "Checking API endpoints..."
    check_api_endpoint "/health" "200" "Health" || true
    check_api_auth "/admin/status" "Admin status" || true
    check_api_auth "/users" "Users" || true
    echo ""
    
    # Sync check
    check_sync || true
    echo ""
    
    # Resource checks
    check_resources || true
    echo ""
    
    # Performance check
    check_performance || true
    echo ""
    
    # Summary
    echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}    ${CYAN}Health Check Summary${NC}                                  ${BLUE}║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}✓ All checks passed!${NC}"
        echo -e "  Status: ${GREEN}HEALTHY${NC}"
    elif [ $ERRORS -eq 0 ]; then
        echo -e "${YELLOW}⚠ Some warnings detected${NC}"
        echo -e "  Status: ${YELLOW}DEGRADED${NC}"
        echo -e "  Warnings: ${WARNINGS}"
    else
        echo -e "${RED}✗ Health check failed${NC}"
        echo -e "  Status: ${RED}UNHEALTHY${NC}"
        echo -e "  Errors: ${ERRORS}"
        echo -e "  Warnings: ${WARNINGS}"
    fi
    
    echo ""
    
    return $HEALTH_STATUS
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=1
            shift
            ;;
        -e|--exit-on-error)
            EXIT_ON_ERROR=1
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose         Enable verbose output"
            echo "  -e, --exit-on-error   Exit immediately on first error"
            echo "  -t, --timeout SEC     Set timeout for checks (default: 5)"
            echo "  -h, --help           Show this help message"
            echo ""
            echo "Exit codes:"
            echo "  0 - All checks passed"
            echo "  1 - One or more checks failed"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run health check
run_health_check

# Exit with appropriate code
exit $HEALTH_STATUS