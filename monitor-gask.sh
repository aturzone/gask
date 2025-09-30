#!/bin/bash

# ═══════════════════════════════════════════════════════════
# GASK Advanced Monitoring Script
# Real-time monitoring with alerts
# ═══════════════════════════════════════════════════════════

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# Configuration
REFRESH_INTERVAL=${REFRESH_INTERVAL:-5}
ALERT_THRESHOLD_CPU=80
ALERT_THRESHOLD_MEMORY=80
ALERT_THRESHOLD_RESPONSE_TIME=2000

# Load environment
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

API_PORT=${API_PORT:-7890}
OWNER_PASSWORD=${OWNER_PASSWORD:-admin1234}

# Clear screen and show banner
clear_and_banner() {
    clear
    echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}    ${CYAN}GASK Real-Time Monitor${NC}                              ${BLUE}║${NC}"
    echo -e "${BLUE}║${NC}    Press Ctrl+C to exit                                 ${BLUE}║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Get container status
get_container_status() {
    local container=$1
    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        local health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "none")
        if [ "$health" = "healthy" ]; then
            echo -e "${GREEN}✓ Running (Healthy)${NC}"
        elif [ "$health" = "none" ]; then
            echo -e "${YELLOW}⚠ Running (No Health Check)${NC}"
        else
            echo -e "${YELLOW}⚠ Running (${health})${NC}"
        fi
    else
        echo -e "${RED}✗ Not Running${NC}"
    fi
}

# Get container stats
get_container_stats() {
    local container=$1
    docker stats --no-stream --format "{{.CPUPerc}}|{{.MemPerc}}|{{.MemUsage}}" "$container" 2>/dev/null || echo "N/A|N/A|N/A"
}

# Test API endpoint
test_api_endpoint() {
    local endpoint=$1
    local start_time=$(date +%s%3N)
    local response=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}${endpoint}" 2>/dev/null)
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ "$response" = "200" ]; then
        if [ $response_time -gt $ALERT_THRESHOLD_RESPONSE_TIME ]; then
            echo -e "${YELLOW}✓ ${response} (${response_time}ms - SLOW)${NC}"
        else
            echo -e "${GREEN}✓ ${response} (${response_time}ms)${NC}"
        fi
    elif [ "$response" = "000" ]; then
        echo -e "${RED}✗ Not Responding${NC}"
    else
        echo -e "${YELLOW}⚠ ${response}${NC}"
    fi
}

# Get Redis info
get_redis_info() {
    docker exec gaskRedis redis-cli INFO stats 2>/dev/null | grep "total_commands_processed" | cut -d: -f2 | tr -d '\r' || echo "N/A"
}

# Get PostgreSQL connections
get_postgres_connections() {
    docker exec gaskPostgres psql -U airflow -d airflow -t -c "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | tr -d ' ' || echo "N/A"
}

# Get system load
get_system_load() {
    uptime | awk -F'load average:' '{print $2}' | awk '{print $1,$2,$3}'
}

# Main monitoring loop
monitor() {
    while true; do
        clear_and_banner
        
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        echo -e "${CYAN}Last Update: ${timestamp}${NC}"
        echo ""
        
        # ═══════════════════════════════════════════════════
        # Container Status
        # ═══════════════════════════════════════════════════
        echo -e "${BLUE}┌─────────────────────────────────────────────────────────┐${NC}"
        echo -e "${BLUE}│${NC} ${MAGENTA}Container Status${NC}                                        ${BLUE}│${NC}"
        echo -e "${BLUE}└─────────────────────────────────────────────────────────┘${NC}"
        
        printf "  %-20s %s\n" "gaskMain:" "$(get_container_status gaskMain)"
        printf "  %-20s %s\n" "gaskRedis:" "$(get_container_status gaskRedis)"
        printf "  %-20s %s\n" "gaskPostgres:" "$(get_container_status gaskPostgres)"
        echo ""
        
        # ═══════════════════════════════════════════════════
        # Resource Usage
        # ═══════════════════════════════════════════════════
        echo -e "${BLUE}┌─────────────────────────────────────────────────────────┐${NC}"
        echo -e "${BLUE}│${NC} ${MAGENTA}Resource Usage${NC}                                          ${BLUE}│${NC}"
        echo -e "${BLUE}└─────────────────────────────────────────────────────────┘${NC}"
        
        # API Server
        local api_stats=$(get_container_stats gaskMain)
        local api_cpu=$(echo "$api_stats" | cut -d'|' -f1 | sed 's/%//')
        local api_mem=$(echo "$api_stats" | cut -d'|' -f2 | sed 's/%//')
        local api_mem_usage=$(echo "$api_stats" | cut -d'|' -f3)
        
        printf "  %-20s" "API Server:"
        if [ "$api_cpu" != "N/A" ]; then
            printf "CPU: "
            if (( $(echo "$api_cpu > $ALERT_THRESHOLD_CPU" | bc -l 2>/dev/null || echo 0) )); then
                echo -e "${RED}${api_cpu}%%${NC}  Mem: ${api_mem}%% (${api_mem_usage})"
            else
                echo -e "${GREEN}${api_cpu}%%${NC}  Mem: ${api_mem}%% (${api_mem_usage})"
            fi
        else
            echo "N/A"
        fi
        
        # Redis
        local redis_stats=$(get_container_stats gaskRedis)
        local redis_cpu=$(echo "$redis_stats" | cut -d'|' -f1 | sed 's/%//')
        local redis_mem=$(echo "$redis_stats" | cut -d'|' -f2 | sed 's/%//')
        local redis_mem_usage=$(echo "$redis_stats" | cut -d'|' -f3)
        
        printf "  %-20s" "Redis:"
        if [ "$redis_cpu" != "N/A" ]; then
            echo "CPU: ${redis_cpu}%  Mem: ${redis_mem}% (${redis_mem_usage})"
        else
            echo "N/A"
        fi
        
        # PostgreSQL
        local pg_stats=$(get_container_stats gaskPostgres)
        local pg_cpu=$(echo "$pg_stats" | cut -d'|' -f1 | sed 's/%//')
        local pg_mem=$(echo "$pg_stats" | cut -d'|' -f2 | sed 's/%//')
        local pg_mem_usage=$(echo "$pg_stats" | cut -d'|' -f3)
        
        printf "  %-20s" "PostgreSQL:"
        if [ "$pg_cpu" != "N/A" ]; then
            echo "CPU: ${pg_cpu}%  Mem: ${pg_mem}% (${pg_mem_usage})"
        else
            echo "N/A"
        fi
        echo ""
        
        # ═══════════════════════════════════════════════════
        # API Health
        # ═══════════════════════════════════════════════════
        echo -e "${BLUE}┌─────────────────────────────────────────────────────────┐${NC}"
        echo -e "${BLUE}│${NC} ${MAGENTA}API Health${NC}                                              ${BLUE}│${NC}"
        echo -e "${BLUE}└─────────────────────────────────────────────────────────┘${NC}"
        
        printf "  %-20s " "/health"
        test_api_endpoint "/health"
        
        printf "  %-20s " "/admin/status"
        test_api_endpoint "/admin/status"
        
        printf "  %-20s " "/users"
        test_api_endpoint "/users"
        echo ""
        
        # ═══════════════════════════════════════════════════
        # Database Info
        # ═══════════════════════════════════════════════════
        echo -e "${BLUE}┌─────────────────────────────────────────────────────────┐${NC}"
        echo -e "${BLUE}│${NC} ${MAGENTA}Database Metrics${NC}                                        ${BLUE}│${NC}"
        echo -e "${BLUE}└─────────────────────────────────────────────────────────┘${NC}"
        
        local redis_commands=$(get_redis_info)
        printf "  %-30s %s\n" "Redis Total Commands:" "$redis_commands"
        
        local pg_connections=$(get_postgres_connections)
        printf "  %-30s %s\n" "PostgreSQL Connections:" "$pg_connections"
        echo ""
        
        # ═══════════════════════════════════════════════════
        # System Info
        # ═══════════════════════════════════════════════════
        echo -e "${BLUE}┌─────────────────────────────────────────────────────────┐${NC}"
        echo -e "${BLUE}│${NC} ${MAGENTA}System Load${NC}                                             ${BLUE}│${NC}"
        echo -e "${BLUE}└─────────────────────────────────────────────────────────┘${NC}"
        
        local load=$(get_system_load)
        printf "  %-30s %s\n" "Load Average (1m, 5m, 15m):" "$load"
        echo ""
        
        # ═══════════════════════════════════════════════════
        # Recent Logs
        # ═══════════════════════════════════════════════════
        echo -e "${BLUE}┌─────────────────────────────────────────────────────────┐${NC}"
        echo -e "${BLUE}│${NC} ${MAGENTA}Recent Logs (Last 5 lines)${NC}                              ${BLUE}│${NC}"
        echo -e "${BLUE}└─────────────────────────────────────────────────────────┘${NC}"
        
        docker logs gaskMain --tail 5 2>&1 | sed 's/^/  /'
        echo ""
        
        # Sleep before next refresh
        echo -e "${CYAN}Refreshing in ${REFRESH_INTERVAL} seconds... (Ctrl+C to exit)${NC}"
        sleep $REFRESH_INTERVAL
    done
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--interval)
            REFRESH_INTERVAL="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -i, --interval SECONDS    Refresh interval (default: 5)"
            echo "  -h, --help               Show this help message"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run monitor
trap 'echo ""; echo "Monitoring stopped."; exit 0' INT TERM

monitor