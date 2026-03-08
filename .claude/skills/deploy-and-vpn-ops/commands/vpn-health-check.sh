#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Usage
usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Check VPN Agent and tunnel health.

Options:
  -v, --verbose    Show detailed tunnel information
  -h, --help       Show this help message

Example:
  $0              # Basic health check
  $0 -v           # Detailed health check
EOF
    exit 0
}

# Parse arguments
VERBOSE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Configuration
VPN_AGENT_URL="http://localhost:8888"
HK_SERVER="47.76.82.51"
PEM_FILE="$HOME/work/sub2api.pem"

echo "🔍 VPN Health Check Report"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check 1: VPN Agent Service
echo -n "VPN Agent Service... "
if curl -sf "$VPN_AGENT_URL/health" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
    echo "  → Check if container is running: docker ps | grep vpn-agent"
    exit 1
fi

# Check 2: Get tunnel list
echo ""
echo "Fetching tunnel list..."
TUNNELS=$(curl -sf "$VPN_AGENT_URL/api/tunnels" 2>/dev/null)
if [[ -z "$TUNNELS" ]]; then
    echo -e "${RED}✗ Failed to fetch tunnels${NC}"
    exit 1
fi

TUNNEL_COUNT=$(echo "$TUNNELS" | jq '. | length' 2>/dev/null || echo "0")
echo -e "Total tunnels: ${BLUE}$TUNNEL_COUNT${NC}"
echo ""

# Check 3: Test each tunnel
if [[ "$TUNNEL_COUNT" -eq 0 ]]; then
    echo -e "${YELLOW}⚠ No tunnels configured${NC}"
    exit 0
fi

echo "Testing tunnel connectivity..."
echo ""

HEALTHY=0
UNHEALTHY=0

for i in $(seq 0 $((TUNNEL_COUNT - 1))); do
    TUNNEL=$(echo "$TUNNELS" | jq -r ".[$i]")
    TUNNEL_ID=$(echo "$TUNNEL" | jq -r '.id')
    TUNNEL_NAME=$(echo "$TUNNEL" | jq -r '.name')
    TUNNEL_PORT=$(echo "$TUNNEL" | jq -r '.port')
    TUNNEL_STATUS=$(echo "$TUNNEL" | jq -r '.status')

    # Validate port number (must be 1-65535)
    if [[ ! "$TUNNEL_PORT" =~ ^[0-9]+$ ]] || [[ "$TUNNEL_PORT" -lt 1 ]] || [[ "$TUNNEL_PORT" -gt 65535 ]]; then
        echo -e "${RED}✗ Invalid port: $TUNNEL_PORT${NC}"
        UNHEALTHY=$((UNHEALTHY + 1))
        continue
    fi

    echo -n "[$((i+1))/$TUNNEL_COUNT] $TUNNEL_NAME (Port: $TUNNEL_PORT)... "

    # Test connectivity via SSH
    if ssh -i "$PEM_FILE" -o ConnectTimeout=5 -o StrictHostKeyChecking=no root@$HK_SERVER \
        "curl -sf --socks5 \"localhost:$TUNNEL_PORT\" https://www.google.com >/dev/null 2>&1" 2>/dev/null; then
        echo -e "${GREEN}✓ Healthy${NC}"
        HEALTHY=$((HEALTHY + 1))

        if [[ "$VERBOSE" == "true" ]]; then
            echo "    ID: $TUNNEL_ID"
            echo "    Status: $TUNNEL_STATUS"
        fi
    else
        echo -e "${RED}✗ Unhealthy${NC}"
        UNHEALTHY=$((UNHEALTHY + 1))

        if [[ "$VERBOSE" == "true" ]]; then
            echo "    ID: $TUNNEL_ID"
            echo "    Status: $TUNNEL_STATUS"
            echo "    → Check tunnel logs in VPN Agent UI"
        fi
    fi

    if [[ "$VERBOSE" == "true" ]]; then
        echo ""
    fi
done

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Summary:"
echo -e "  Total: ${BLUE}$TUNNEL_COUNT${NC}"
echo -e "  Healthy: ${GREEN}$HEALTHY${NC}"
echo -e "  Unhealthy: ${RED}$UNHEALTHY${NC}"
echo ""

if [[ "$UNHEALTHY" -gt 0 ]]; then
    echo -e "${YELLOW}⚠ Some tunnels are unhealthy${NC}"
    echo "  → Check VPN Agent UI: $VPN_AGENT_URL"
    exit 1
else
    echo -e "${GREEN}✓ All tunnels are healthy!${NC}"
fi
