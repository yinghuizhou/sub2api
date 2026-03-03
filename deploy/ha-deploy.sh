#\!/bin/bash
# =============================================================================
# Sub2API High Availability Deployment Script
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_env() {
    if [ \! -f .env ]; then
        log_error ".env file not found\!"
        log_info "Please copy .env.ha.example to .env and configure it:"
        log_info "  cp .env.ha.example .env"
        log_info "  vim .env"
        exit 1
    fi

    # Check required variables
    source .env
    if [ -z "$POSTGRES_PASSWORD" ]; then
        log_error "POSTGRES_PASSWORD is not set in .env"
        exit 1
    fi
    if [ -z "$JWT_SECRET" ]; then
        log_error "JWT_SECRET is not set in .env (required for HA deployment)"
        log_info "Generate one with: openssl rand -hex 32"
        exit 1
    fi
    if [ -z "$TOTP_ENCRYPTION_KEY" ]; then
        log_error "TOTP_ENCRYPTION_KEY is not set in .env (required for HA deployment)"
        log_info "Generate one with: openssl rand -hex 32"
        exit 1
    fi
}

check_nginx_config() {
    if [ \! -f nginx/nginx.conf ]; then
        log_error "nginx/nginx.conf not found\!"
        exit 1
    fi
    if [ \! -f nginx/conf.d/sub2api.conf ]; then
        log_error "nginx/conf.d/sub2api.conf not found\!"
        exit 1
    fi
}

# Main menu
show_menu() {
    echo ""
    echo "=========================================="
    echo "  Sub2API High Availability Deployment"
    echo "=========================================="
    echo "1. Deploy (3 instances)"
    echo "2. Deploy (5 instances)"
    echo "3. Deploy (custom instances)"
    echo "4. Scale up/down"
    echo "5. Rolling update"
    echo "6. Stop all"
    echo "7. View status"
    echo "8. View logs"
    echo "9. Health check"
    echo "0. Exit"
    echo "=========================================="
}

deploy_ha() {
    local instances=${1:-3}
    log_info "Deploying Sub2API with $instances instances..."
    
    check_env
    check_nginx_config
    
    log_info "Starting services..."
    docker compose -f docker-compose.ha.yml up -d
    
    log_info "Waiting for services to be healthy..."
    sleep 10
    
    log_info "Checking health..."
    for i in 1 2 3; do
        if docker exec sub2api-$i curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log_info "Instance $i: ${GREEN}HEALTHY${NC}"
        else
            log_warn "Instance $i: ${RED}UNHEALTHY${NC}"
        fi
    done
    
    log_info "Deployment complete\!"
    log_info "Access: http://localhost"
}

deploy_scale() {
    local instances=${1:-3}
    log_info "Deploying Sub2API with $instances instances (scale mode)..."
    
    check_env
    
    log_info "Starting services..."
    docker compose -f docker-compose.ha-scale.yml up -d --scale sub2api=$instances
    
    log_info "Waiting for services to be healthy..."
    sleep 10
    
    log_info "Deployment complete\!"
    log_info "Access: http://localhost"
}

scale_instances() {
    read -p "Enter number of instances (1-10): " instances
    if [ "$instances" -lt 1 ] || [ "$instances" -gt 10 ]; then
        log_error "Invalid number of instances (must be 1-10)"
        return
    fi
    
    log_info "Scaling to $instances instances..."
    docker compose -f docker-compose.ha-scale.yml up -d --scale sub2api=$instances
    
    log_info "Scale complete\!"
}

rolling_update() {
    log_info "Starting rolling update..."
    
    log_info "Pulling latest image..."
    docker pull weishaw/sub2api:latest
    
    for i in 1 2 3; do
        log_info "Updating instance $i..."
        docker compose -f docker-compose.ha.yml stop sub2api-$i
        docker compose -f docker-compose.ha.yml up -d sub2api-$i
        
        log_info "Waiting for instance $i to be healthy..."
        sleep 30
        
        if docker exec sub2api-$i curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log_info "Instance $i: ${GREEN}HEALTHY${NC}"
        else
            log_error "Instance $i: ${RED}UNHEALTHY${NC}"
            log_error "Rolling update failed\! Please check logs."
            return 1
        fi
    done
    
    log_info "Rolling update complete\!"
}

stop_all() {
    log_warn "Stopping all services..."
    docker compose -f docker-compose.ha.yml down
    log_info "All services stopped."
}

view_status() {
    log_info "Service status:"
    docker compose -f docker-compose.ha.yml ps
}

view_logs() {
    echo "Select service:"
    echo "1. All"
    echo "2. Nginx"
    echo "3. Sub2API Instance 1"
    echo "4. Sub2API Instance 2"
    echo "5. Sub2API Instance 3"
    echo "6. PostgreSQL"
    echo "7. Redis"
    read -p "Enter choice: " choice
    
    case $choice in
        1) docker compose -f docker-compose.ha.yml logs -f ;;
        2) docker compose -f docker-compose.ha.yml logs -f nginx ;;
        3) docker compose -f docker-compose.ha.yml logs -f sub2api-1 ;;
        4) docker compose -f docker-compose.ha.yml logs -f sub2api-2 ;;
        5) docker compose -f docker-compose.ha.yml logs -f sub2api-3 ;;
        6) docker compose -f docker-compose.ha.yml logs -f postgres ;;
        7) docker compose -f docker-compose.ha.yml logs -f redis ;;
        *) log_error "Invalid choice" ;;
    esac
}

health_check() {
    log_info "Checking health..."
    
    echo ""
    echo "Nginx:"
    if curl -sf http://localhost/health > /dev/null 2>&1; then
        echo -e "  ${GREEN}HEALTHY${NC}"
    else
        echo -e "  ${RED}UNHEALTHY${NC}"
    fi
    
    echo ""
    echo "Sub2API Instances:"
    for i in 1 2 3; do
        if docker exec sub2api-$i curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            echo -e "  Instance $i: ${GREEN}HEALTHY${NC}"
        else
            echo -e "  Instance $i: ${RED}UNHEALTHY${NC}"
        fi
    done
    
    echo ""
    echo "PostgreSQL:"
    if docker exec sub2api-postgres pg_isready -U sub2api > /dev/null 2>&1; then
        echo -e "  ${GREEN}HEALTHY${NC}"
    else
        echo -e "  ${RED}UNHEALTHY${NC}"
    fi
    
    echo ""
    echo "Redis:"
    if docker exec sub2api-redis redis-cli ping > /dev/null 2>&1; then
        echo -e "  ${GREEN}HEALTHY${NC}"
    else
        echo -e "  ${RED}UNHEALTHY${NC}"
    fi
}

# Main loop
while true; do
    show_menu
    read -p "Enter choice: " choice
    
    case $choice in
        1) deploy_ha 3 ;;
        2) deploy_ha 5 ;;
        3)
            read -p "Enter number of instances: " instances
            deploy_scale $instances
            ;;
        4) scale_instances ;;
        5) rolling_update ;;
        6) stop_all ;;
        7) view_status ;;
        8) view_logs ;;
        9) health_check ;;
        0) log_info "Goodbye\!"; exit 0 ;;
        *) log_error "Invalid choice" ;;
    esac
    
    read -p "Press Enter to continue..."
done
