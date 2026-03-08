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
  - Docker Desktop is running
  - SSH connection to HK server (47.76.82.51)
  - VPN Agent health (http://localhost:8888/)
  - VPN instance count (minimum 2)

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
VPN_AGENT_URL="http://localhost:8888"
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

# Check 3: Sub2API Service Health
echo -n "Checking Sub2API service health... "
if curl -sf "$VPN_AGENT_URL/health" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Healthy"
else
    echo -e "${RED}✗${NC} Unhealthy"
    echo "  → Check if Sub2API is running: docker ps | grep sub2api"
    echo "  → Check logs: docker logs sub2api-sub2api-1"
    exit 1
fi

# Check 4: Sub2API Instance Count
echo -n "Checking Sub2API instance count... "
INSTANCE_COUNT=$(docker compose ps sub2api --format json 2>/dev/null | jq -s 'length' 2>/dev/null || echo "0")
if [[ "$INSTANCE_COUNT" -ge "$MIN_INSTANCES" ]]; then
    echo -e "${GREEN}✓${NC} $INSTANCE_COUNT instances (minimum: $MIN_INSTANCES)"
else
    echo -e "${YELLOW}⚠${NC} Only $INSTANCE_COUNT instances (minimum: $MIN_INSTANCES)"
    echo "  → Add more instances via VPN Agent UI: $VPN_AGENT_URL"
fi

echo ""
echo -e "${GREEN}✓ All prerequisites met!${NC}"
