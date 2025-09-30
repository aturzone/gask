#!/bin/bash

# ═══════════════════════════════════════════════════════════
# GASK Redis Debug Script
# ═══════════════════════════════════════════════════════════

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}    ${CYAN}GASK Redis Debug Tool${NC}                                 ${BLUE}║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Load .env
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

REDIS_PORT=${REDIS_PORT:-6380}

# Check 1: Port availability
echo -e "${CYAN}[1/8]${NC} Checking if port ${REDIS_PORT} is available..."
if lsof -Pi :${REDIS_PORT} -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Port ${REDIS_PORT} is already in use:${NC}"
    lsof -Pi :${REDIS_PORT} -sTCP:LISTEN
    echo ""
    echo -e "${YELLOW}Solution: Change REDIS_PORT in .env or stop the service using this port${NC}"
else
    echo -e "${GREEN}✓ Port ${REDIS_PORT} is available${NC}"
fi
echo ""

# Check 2: Container exists
echo -e "${CYAN}[2/8]${NC} Checking if gaskRedis container exists..."
if docker ps -a | grep -q gaskRedis; then
    echo -e "${GREEN}✓ Container exists${NC}"
    
    # Check container status
    STATUS=$(docker inspect -f '{{.State.Status}}' gaskRedis 2>/dev/null)
    echo -e "   Status: ${YELLOW}${STATUS}${NC}"
    
    HEALTH=$(docker inspect -f '{{.State.Health.Status}}' gaskRedis 2>/dev/null || echo "none")
    if [ "$HEALTH" != "none" ]; then
        echo -e "   Health: ${YELLOW}${HEALTH}${NC}"
    fi
else
    echo -e "${RED}✗ Container does not exist${NC}"
fi
echo ""

# Check 3: Container logs
echo -e "${CYAN}[3/8]${NC} Checking container logs..."
if docker ps -a | grep -q gaskRedis; then
    echo -e "${YELLOW}Last 20 lines of logs:${NC}"
    docker logs --tail 20 gaskRedis 2>&1 || echo "No logs available"
else
    echo -e "${YELLOW}Container not found, skipping logs${NC}"
fi
echo ""

# Check 4: Redis connectivity test
echo -e "${CYAN}[4/8]${NC} Testing Redis connectivity..."
if docker ps | grep -q gaskRedis; then
    if docker exec gaskRedis redis-cli ping >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Redis is responding to PING${NC}"
        
        # Get Redis info
        echo -e "${CYAN}   Redis version:${NC} $(docker exec gaskRedis redis-cli INFO server | grep redis_version | cut -d: -f2 | tr -d '\r')"
        echo -e "${CYAN}   Memory used:${NC} $(docker exec gaskRedis redis-cli INFO memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')"
    else
        echo -e "${RED}✗ Redis is not responding${NC}"
    fi
else
    echo -e "${YELLOW}Container is not running, skipping connectivity test${NC}"
fi
echo ""

# Check 5: Volume status
echo -e "${CYAN}[5/8]${NC} Checking Redis volume..."
if docker volume ls | grep -q gask_redis_data; then
    echo -e "${GREEN}✓ Volume exists: gask_redis_data${NC}"
    
    # Show volume details
    SIZE=$(docker volume inspect gask_redis_data --format '{{ .Mountpoint }}' 2>/dev/null)
    if [ -n "$SIZE" ]; then
        echo -e "   Mount point: ${SIZE}"
    fi
else
    echo -e "${YELLOW}⚠ Volume does not exist (will be created on startup)${NC}"
fi
echo ""

# Check 6: Network status
echo -e "${CYAN}[6/8]${NC} Checking network connectivity..."
if docker network ls | grep -q gask_network; then
    echo -e "${GREEN}✓ Network exists: gask_network${NC}"
    
    # Check if Redis is connected
    if docker ps | grep -q gaskRedis; then
        if docker inspect gaskRedis | grep -q gask_network; then
            echo -e "${GREEN}✓ Redis is connected to gask_network${NC}"
        else
            echo -e "${YELLOW}⚠ Redis is not connected to gask_network${NC}"
        fi
    fi
else
    echo -e "${YELLOW}⚠ Network does not exist${NC}"
fi
echo ""

# Check 7: Docker resources
echo -e "${CYAN}[7/8]${NC} Checking Docker resources..."
if docker ps | grep -q gaskRedis; then
    echo -e "${YELLOW}Resource usage:${NC}"
    docker stats --no-stream gaskRedis 2>/dev/null || echo "Cannot get stats"
else
    echo -e "${YELLOW}Container not running, skipping resource check${NC}"
fi
echo ""

# Check 8: Health check history
echo -e "${CYAN}[8/8]${NC} Checking health check history..."
if docker ps -a | grep -q gaskRedis; then
    HEALTH_LOG=$(docker inspect gaskRedis --format='{{json .State.Health.Log}}' 2>/dev/null)
    if [ "$HEALTH_LOG" != "null" ] && [ -n "$HEALTH_LOG" ]; then
        echo -e "${YELLOW}Recent health checks:${NC}"
        echo "$HEALTH_LOG" | jq -r '.[] | "\(.Start) - \(.ExitCode) - \(.Output)"' 2>/dev/null | tail -5 || echo "No health check history"
    else
        echo -e "${YELLOW}No health check configured or no history available${NC}"
    fi
else
    echo -e "${YELLOW}Container not found${NC}"
fi
echo ""

# Summary and recommendations
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}    ${CYAN}Recommendations${NC}                                        ${BLUE}║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if container is running
if ! docker ps | grep -q gaskRedis; then
    echo -e "${YELLOW}1. Redis container is not running. Try:${NC}"
    echo -e "   ${CYAN}docker-compose up -d gaskRedis${NC}"
    echo ""
fi

# Check port conflict
if lsof -Pi :${REDIS_PORT} -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}2. Port ${REDIS_PORT} is in use. Options:${NC}"
    echo -e "   a) Stop the service using port ${REDIS_PORT}"
    echo -e "   b) Change REDIS_PORT in .env to a different port"
    echo -e "   c) Run: ${CYAN}./start-gask.sh${NC} (auto port detection)"
    echo ""
fi

# Docker issues
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}3. Docker daemon issue detected!${NC}"
    echo -e "   Try: ${CYAN}sudo systemctl restart docker${NC}"
    echo ""
fi

echo -e "${CYAN}Quick Fix Commands:${NC}"
echo -e "  ${GREEN}# Stop and remove Redis container${NC}"
echo -e "  docker-compose stop gaskRedis"
echo -e "  docker-compose rm -f gaskRedis"
echo ""
echo -e "  ${GREEN}# Recreate with fresh start${NC}"
echo -e "  docker-compose up -d gaskRedis"
echo ""
echo -e "  ${GREEN}# View real-time logs${NC}"
echo -e "  docker logs -f gaskRedis"
echo ""
echo -e "  ${GREEN}# Complete restart${NC}"
echo -e "  make restart"
echo ""

# Exit with status
if docker ps | grep -q gaskRedis && docker exec gaskRedis redis-cli ping >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Redis is healthy!${NC}"
    exit 0
else
    echo -e "${RED}✗ Redis has issues${NC}"
    exit 1
fi