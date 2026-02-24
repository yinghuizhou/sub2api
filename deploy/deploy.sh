#!/bin/bash
# =============================================================================
# Sub2API Auto Deploy Script
# Called by webhook-server.sh when Gitee push event is received.
# Flow: git pull from Gitee → docker build → rolling restart
# =============================================================================

set -euo pipefail

INSTALL_DIR="/opt/sub2api"
SOURCE_DIR="/opt/sub2api/source"
COMPOSE_FILE="docker-compose.local.yml"
LOG_PREFIX="[deploy]"

log() { echo "$LOG_PREFIX [$(date +'%Y-%m-%d %H:%M:%S')] $1"; }

# --- Step 1: Pull latest code from Gitee ---
log "Pulling latest code from Gitee..."
cd "$SOURCE_DIR"
git fetch origin main
git reset --hard origin/main
COMMIT=$(git rev-parse --short HEAD)
log "Updated to commit: $COMMIT"

# --- Step 2: Build Docker image ---
log "Building Docker image..."
docker build -t sub2api:$COMMIT -t weishaw/sub2api:latest "$SOURCE_DIR"
log "Docker image built: sub2api:$COMMIT"

# --- Step 3: Restart services ---
log "Restarting services..."
cd "$INSTALL_DIR"
docker compose -f "$COMPOSE_FILE" up -d --no-deps sub2api

# --- Step 4: Health check ---
log "Waiting for health check..."
for i in $(seq 1 30); do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        log "Health check passed after ${i}s"
        break
    fi
    if [ "$i" -eq 30 ]; then
        log "ERROR: Health check failed after 30s"
        docker compose -f "$COMPOSE_FILE" logs --tail=30 sub2api
        exit 1
    fi
    sleep 1
done

# --- Step 5: Cleanup old images ---
docker image prune -f --filter "until=24h" > /dev/null 2>&1 || true

log "Deploy complete! Running commit: $COMMIT"
