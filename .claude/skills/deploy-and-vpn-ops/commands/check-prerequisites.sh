#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Usage
usage() {
    cat <<EOF
Usage: $0

Check deployment prerequisites:
  - Docker Desktop is running (local)
  - SSH connection to HK server (47.76.82.51)
  - Sub2API service health on HK server (http://localhost:8888/health)
  - Sub2API instance count on HK server (minimum 2 for HA)

Options:
  -h, --help    Show this help message

Example:
  $0
EOF
    exit 0
}

# Parse arguments
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    usage
fi

# Configuration
HK_SERVER="47.76.82.51"
PEM_FILE="$HOME/work/sub2api.pem"
SUB2API_HEALTH_URL="http://localhost:8888/health"
MIN_INSTANCES=2

echo "🔍 Checking deployment prerequisites..."
echo ""

# Check 1: Docker Desktop
echo -n "Checking Docker Desktop... "
if docker info >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Running"
else
    echo -e "${RED}✗${NC} Not running"
    echo "  → Start Docker Desktop: open /Applications/Docker.app"
    exit 1
fi

# Check 2: SSH Connection
echo -n "Checking SSH connection to HK server... "
if ssh -i "$PEM_FILE" -o ConnectTimeout=5 -o StrictHostKeyChecking=no root@$HK_SERVER "echo ok" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Connected"
else
    echo -e "${RED}✗${NC} Failed"
    echo "  → Check PEM file: $PEM_FILE"
    echo "  → Check server: $HK_SERVER"
    exit 1
fi

# Check 3: Sub2API Service Health (on HK server)
echo -n "Checking Sub2API service health on HK server... "
if ssh -i "$PEM_FILE" -o ConnectTimeout=5 -o StrictHostKeyChecking=no root@$HK_SERVER \
    "curl -sf $SUB2API_HEALTH_URL" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Healthy"
else
    echo -e "${RED}✗${NC} Unhealthy"
    echo "  → Check if Sub2API is running: ssh -i $PEM_FILE root@$HK_SERVER 'docker ps | grep sub2api'"
    echo "  → Check logs: ssh -i $PEM_FILE root@$HK_SERVER 'docker logs sub2api-sub2api-1'"
    echo "  → Check Nginx: ssh -i $PEM_FILE root@$HK_SERVER 'docker logs sub2api-nginx'"
    exit 1
fi

# Check 4: Sub2API Instance Count (on HK server)
echo -n "Checking Sub2API instance count on HK server... "
INSTANCE_COUNT=$(ssh -i "$PEM_FILE" -o ConnectTimeout=5 -o StrictHostKeyChecking=no root@$HK_SERVER \
    "cd /opt/sub2api && docker compose ps sub2api --format json 2>/dev/null | jq -s 'length' 2>/dev/null || echo '0'")
if [[ "$INSTANCE_COUNT" -ge "$MIN_INSTANCES" ]]; then
    echo -e "${GREEN}✓${NC} $INSTANCE_COUNT instances (minimum: $MIN_INSTANCES)"
else
    echo -e "${YELLOW}⚠${NC} Only $INSTANCE_COUNT instances (minimum: $MIN_INSTANCES)"
    echo "  → Scale up: ssh -i $PEM_FILE root@$HK_SERVER 'cd /opt/sub2api && docker compose up -d --scale sub2api=$MIN_INSTANCES'"
fi

echo ""
echo -e "${GREEN}✓ All prerequisites met!${NC}"
