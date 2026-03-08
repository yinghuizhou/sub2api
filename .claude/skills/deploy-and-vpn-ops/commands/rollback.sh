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

Rollback to a previous Docker image version.

Options:
  -l, --list       List available image versions
  -v, --version    Specify version to rollback to
  -h, --help       Show this help message

Example:
  $0 -l                    # List available versions
  $0 -v amd64-hk-20240308  # Rollback to specific version
  $0                       # Interactive selection
EOF
    exit 0
}

# Configuration
HK_SERVER="47.76.82.51"
PEM_FILE="$HOME/work/sub2api.pem"
IMAGE_NAME="sub2api"

# Validation function
validate_version() {
    local version=$1
    if [[ ! "$version" =~ ^[a-zA-Z0-9][a-zA-Z0-9._-]*$ ]]; then
        echo -e "${RED}✗ Invalid version format: $version${NC}"
        echo "  → Version must start with alphanumeric and contain only alphanumeric characters, dots, hyphens, and underscores"
        exit 1
    fi
}

# Parse arguments
LIST_ONLY=false
TARGET_VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -l|--list)
            LIST_ONLY=true
            shift
            ;;
        -v|--version)
            TARGET_VERSION="$2"
            shift 2
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

# Function: List available images
list_images() {
    echo "📦 Fetching available images from HK server..."
    echo ""

    IMAGES=$(ssh -i "$PEM_FILE" -o StrictHostKeyChecking=no root@$HK_SERVER \
        "docker images $IMAGE_NAME --format '{{.Tag}}\t{{.CreatedAt}}\t{{.Size}}'" 2>/dev/null)

    if [[ -z "$IMAGES" ]]; then
        echo -e "${RED}✗ No images found${NC}"
        exit 1
    fi

    echo "Available versions:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    printf "%-20s %-30s %-10s\n" "TAG" "CREATED" "SIZE"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    echo "$IMAGES" | while IFS=$'\t' read -r tag created size; do
        printf "%-20s %-30s %-10s\n" "$tag" "$created" "$size"
    done

    echo ""
}

# List only mode
if [[ "$LIST_ONLY" == "true" ]]; then
    list_images
    exit 0
fi

# Interactive selection
if [[ -z "$TARGET_VERSION" ]]; then
    list_images

    echo -n "Enter version tag to rollback to (or 'q' to quit): "
    read -r TARGET_VERSION

    if [[ "$TARGET_VERSION" == "q" ]]; then
        echo "Rollback cancelled."
        exit 0
    fi
fi

# Validate version format
validate_version "$TARGET_VERSION"

# Validate version exists
echo ""
echo "🔍 Validating version: $TARGET_VERSION"

VERSION_EXISTS=$(ssh -i "$PEM_FILE" -o StrictHostKeyChecking=no root@$HK_SERVER \
    "docker images \"$IMAGE_NAME:$TARGET_VERSION\" -q" 2>/dev/null)

if [[ -z "$VERSION_EXISTS" ]]; then
    echo -e "${RED}✗ Version not found: $TARGET_VERSION${NC}"
    echo "  → Run '$0 -l' to see available versions"
    exit 1
fi

echo -e "${GREEN}✓ Version found${NC}"

# Confirmation
echo ""
echo -e "${YELLOW}⚠ This will rollback to: $IMAGE_NAME:$TARGET_VERSION${NC}"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Rollback cancelled."
    exit 0
fi

# Execute rollback
echo ""
echo "🔄 Rolling back to $TARGET_VERSION..."
echo ""

# Step 1: Update docker-compose.yml to use target version
echo "1. Updating docker-compose.yml..."
# Escape special characters in version string for sed replacement
TARGET_VERSION_ESCAPED=$(printf '%s\n' "$TARGET_VERSION" | sed 's/[&\\|]/\\&/g')
ssh -i "$PEM_FILE" -o StrictHostKeyChecking=no root@$HK_SERVER <<EOF
cd /opt/sub2api
sed -i "s|image: $IMAGE_NAME:.*|image: $IMAGE_NAME:$TARGET_VERSION_ESCAPED|g" docker-compose.yml
EOF

if [[ $? -eq 0 ]]; then
    echo -e "   ${GREEN}✓ Updated${NC}"
else
    echo -e "   ${RED}✗ Failed${NC}"
    exit 1
fi

# Step 2: Restart container
echo "2. Restarting container..."
ssh -i "$PEM_FILE" -o StrictHostKeyChecking=no root@$HK_SERVER <<EOF
cd /opt/sub2api
docker compose up -d sub2api
EOF

if [[ $? -eq 0 ]]; then
    echo -e "   ${GREEN}✓ Restarted${NC}"
else
    echo -e "   ${RED}✗ Failed${NC}"
    exit 1
fi

# Step 3: Wait for service to be ready
echo "3. Waiting for service to be ready..."
sleep 5

# Step 4: Health check
echo "4. Running health check..."
HEALTH_STATUS=$(ssh -i "$PEM_FILE" -o StrictHostKeyChecking=no root@$HK_SERVER \
    "curl -sf http://localhost:8888/health" 2>/dev/null)

if [[ -n "$HEALTH_STATUS" ]]; then
    echo -e "   ${GREEN}✓ Service is healthy${NC}"
else
    echo -e "   ${RED}✗ Service is not responding${NC}"
    echo "   → Check logs: ssh -i $PEM_FILE root@$HK_SERVER 'docker logs sub2api'"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Rollback completed successfully!${NC}"
echo ""
echo "Current version: $TARGET_VERSION"
echo "Service URL: http://47.76.82.51:8888"
