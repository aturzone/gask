#!/bin/bash

# ═══════════════════════════════════════════════════════════
# GASK Cleanup Script
# Comprehensive cleanup with safety confirmations
# ═══════════════════════════════════════════════════════════

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Modes
MODE=""
FORCE=0
KEEP_VOLUMES=0
KEEP_BACKUPS=0

# Functions
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

# Confirmation prompt
confirm() {
    local message=$1
    if [ $FORCE -eq 1 ]; then
        return 0
    fi
    
    echo -e "${YELLOW}${message}${NC}"
    read -p "Type 'yes' to continue: " response
    
    if [ "$response" != "yes" ]; then
        log_info "Operation cancelled"
        exit 0
    fi
}

# Stop containers
stop_containers() {
    log_info "Stopping GASK containers..."
    
    if docker-compose ps | grep -q "Up"; then
        docker-compose stop
        log_success "Containers stopped"
    else
        log_info "No running containers found"
    fi
}

# Remove containers
remove_containers() {
    log_info "Removing containers..."
    
    docker-compose rm -f 2>/dev/null || true
    
    # Force remove if still exists
    for container in gaskMain gaskRedis gaskPostgres; do
        if docker ps -a | grep -q "$container"; then
            docker rm -f "$container" 2>/dev/null || true
        fi
    done
    
    log_success "Containers removed"
}

# Remove volumes
remove_volumes() {
    log_info "Removing volumes..."
    
    confirm "⚠️  This will DELETE ALL DATA! Are you sure?"
    
    docker volume rm gask_redis_data gask_postgres_data gask_logs 2>/dev/null || true
    
    log_success "Volumes removed"
}

# Remove network
remove_network() {
    log_info "Removing network..."
    
    docker network rm gask_network 2>/dev/null || true
    
    log_success "Network removed"
}

# Remove images
remove_images() {
    log_info "Removing images..."
    
    # Remove GASK image
    docker rmi gask:latest 2>/dev/null || true
    
    # Remove dangling images
    docker image prune -f
    
    log_success "Images removed"
}

# Clean backup files
clean_backups() {
    if [ -d "backups" ]; then
        log_info "Cleaning backup files..."
        
        confirm "Remove all backup files in ./backups?"
        
        rm -rf backups/*
        log_success "Backups cleaned"
    fi
}

# Clean logs
clean_logs() {
    if [ -d "logs" ]; then
        log_info "Cleaning log files..."
        rm -rf logs/*
        log_success "Logs cleaned"
    fi
}

# Clean build artifacts
clean_build() {
    log_info "Cleaning build artifacts..."
    
    rm -f gask *.exe
    rm -rf build/ dist/
    
    log_success "Build artifacts cleaned"
}

# Clean temp files
clean_temp() {
    log_info "Cleaning temporary files..."
    
    rm -f .env.bak
    rm -rf tmp/ temp/
    rm -f *.tmp
    
    log_success "Temporary files cleaned"
}

# Docker system prune
docker_prune() {
    log_info "Running Docker system prune..."
    
    docker system prune -f --volumes
    
    log_success "Docker system pruned"
}

# Show usage
show_usage() {
    cat << EOF
${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}
${BLUE}║${NC}    ${CYAN}GASK Cleanup Script${NC}                                   ${BLUE}║${NC}
${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}

Usage: $0 [MODE] [OPTIONS]

${YELLOW}Cleanup Modes:${NC}
  soft          Stop and remove containers only (keep data)
  medium        Remove containers and images (keep data)
  hard          Remove everything including volumes (⚠️  DATA LOSS!)
  full          Hard cleanup + Docker system prune
  logs          Clean only log files
  backups       Clean only backup files
  build         Clean only build artifacts

${YELLOW}Options:${NC}
  -f, --force           Skip confirmation prompts
  --keep-volumes        Don't remove volumes (with hard/full mode)
  --keep-backups        Don't remove backups
  -h, --help           Show this help message

${YELLOW}Examples:${NC}
  $0 soft                    # Safe cleanup
  $0 hard -f                 # Full cleanup, no confirmation
  $0 full --keep-backups     # Full cleanup but keep backups
  $0 logs                    # Clean only logs

${RED}⚠️  Warning: 'hard' and 'full' modes will DELETE ALL DATA!${NC}

EOF
}

# Soft cleanup
cleanup_soft() {
    echo ""
    log_info "Running SOFT cleanup..."
    echo ""
    
    stop_containers
    remove_containers
    clean_logs
    clean_temp
    
    echo ""
    log_success "Soft cleanup completed!"
    log_info "Data volumes preserved"
}

# Medium cleanup
cleanup_medium() {
    echo ""
    log_info "Running MEDIUM cleanup..."
    echo ""
    
    stop_containers
    remove_containers
    remove_images
    clean_logs
    clean_temp
    
    echo ""
    log_success "Medium cleanup completed!"
    log_info "Data volumes preserved"
}

# Hard cleanup
cleanup_hard() {
    echo ""
    log_warning "Running HARD cleanup..."
    log_warning "This will DELETE ALL DATA!"
    echo ""
    
    if [ $KEEP_VOLUMES -eq 0 ]; then
        confirm "⚠️  ALL DATA WILL BE LOST! Continue?"
    fi
    
    stop_containers
    remove_containers
    
    if [ $KEEP_VOLUMES -eq 0 ]; then
        remove_volumes
    fi
    
    remove_network
    remove_images
    clean_logs
    clean_build
    clean_temp
    
    if [ $KEEP_BACKUPS -eq 0 ]; then
        clean_backups
    fi
    
    echo ""
    if [ $KEEP_VOLUMES -eq 0 ]; then
        log_success "Hard cleanup completed!"
        log_warning "All data has been removed"
    else
        log_success "Hard cleanup completed (volumes preserved)"
    fi
}

# Full cleanup
cleanup_full() {
    echo ""
    log_warning "Running FULL cleanup..."
    log_warning "This is the most aggressive cleanup!"
    echo ""
    
    cleanup_hard
    
    echo ""
    log_info "Running Docker system prune..."
    docker_prune
    
    echo ""
    log_success "Full cleanup completed!"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        soft|medium|hard|full|logs|backups|build)
            MODE=$1
            shift
            ;;
        -f|--force)
            FORCE=1
            shift
            ;;
        --keep-volumes)
            KEEP_VOLUMES=1
            shift
            ;;
        --keep-backups)
            KEEP_BACKUPS=1
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Check if mode is specified
if [ -z "$MODE" ]; then
    show_usage
    exit 1
fi

# Banner
echo ""
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}    ${CYAN}GASK Cleanup Script${NC}                                   ${BLUE}║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"

# Execute cleanup based on mode
case $MODE in
    soft)
        cleanup_soft
        ;;
    medium)
        cleanup_medium
        ;;
    hard)
        cleanup_hard
        ;;
    full)
        cleanup_full
        ;;
    logs)
        clean_logs
        log_success "Logs cleaned"
        ;;
    backups)
        clean_backups
        ;;
    build)
        clean_build
        log_success "Build artifacts cleaned"
        ;;
    *)
        log_error "Invalid mode: $MODE"
        exit 1
        ;;
esac

echo ""
log_info "Cleanup finished!"
echo ""

# Show remaining resources
if command -v docker &> /dev/null; then
    log_info "Remaining GASK resources:"
    echo ""
    
    # Containers
    if docker ps -a | grep -q "gask"; then
        echo "Containers:"
        docker ps -a | grep "gask" || true
        echo ""
    fi
    
    # Volumes
    if docker volume ls | grep -q "gask"; then
        echo "Volumes:"
        docker volume ls | grep "gask" || true
        echo ""
    fi
    
    # Networks
    if docker network ls | grep -q "gask"; then
        echo "Networks:"
        docker network ls | grep "gask" || true
        echo ""
    fi
    
    # Images
    if docker images | grep -q "gask"; then
        echo "Images:"
        docker images | grep "gask" || true
        echo ""
    fi
fi

log_success "All done! 🎉"
echo ""