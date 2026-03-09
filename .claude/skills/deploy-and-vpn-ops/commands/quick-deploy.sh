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
Usage: $0 [OPTIONS]

Quick deployment to production server.

Options:
  -y, --yes     Skip confirmation prompt
  -h, --help    Show this help message

Example:
  $0              # Deploy with confirmation
  $0 -y           # Deploy without confirmation
EOF
    exit 0
}

# Parse arguments
SKIP_CONFIRM=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -y|--yes)
            SKIP_CONFIRM=true
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
PROJECT_ROOT="/Users/zhouyinghui/work/ai/sub2api"
DEPLOY_SCRIPT="$PROJECT_ROOT/scripts/deploy-production.sh"
LOG_FILE="/tmp/deploy-$(date +%Y%m%d-%H%M%S).log"

# Check if deploy script exists
if [[ ! -f "$DEPLOY_SCRIPT" ]]; then
    echo -e "${RED}✗ Deploy script not found: $DEPLOY_SCRIPT${NC}"
    exit 1
fi

# Confirmation
if [[ "$SKIP_CONFIRM" == "false" ]]; then
    echo -e "${YELLOW}⚠ This will deploy to production server (47.76.82.51)${NC}"
    echo ""
    echo "Steps:"
    echo "  1. Build Docker image (linux/amd64)"
    echo "  2. Export and compress image"
    echo "  3. Upload to HK server"
    echo "  4. Load and restart container"
    echo ""
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Deployment cancelled."
        exit 0
    fi
fi

# Run deployment
echo ""
echo "🚀 Starting deployment..."
echo "📝 Log file: $LOG_FILE"
echo ""

if bash "$DEPLOY_SCRIPT" 2>&1 | tee "$LOG_FILE"; then
    echo ""
    echo -e "${GREEN}✓ Deployment completed successfully!${NC}"
    echo "📝 Log saved to: $LOG_FILE"
else
    echo ""
    echo -e "${RED}✗ Deployment failed!${NC}"
    echo "📝 Check log: $LOG_FILE"
    exit 1
fi
