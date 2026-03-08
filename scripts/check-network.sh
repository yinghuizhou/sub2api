#!/bin/bash
# =============================================================================
# Network Environment Check Script
# =============================================================================
# Purpose: Detect network connectivity and recommend build strategy
# Usage: ./scripts/check-network.sh
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Timeout command wrapper (macOS compatible)
run_with_timeout() {
    local timeout=$1
    shift
    if command -v timeout >/dev/null 2>&1; then
        timeout "$timeout" "$@"
    elif command -v gtimeout >/dev/null 2>&1; then
        gtimeout "$timeout" "$@"
    else
        # Fallback: run without timeout on macOS
        "$@"
    fi
}

# Test results
DOCKER_HUB_OK=false
ALPINE_CDN_OK=false
NPM_REGISTRY_OK=false
GITHUB_ACTIONS_AVAILABLE=false

echo -e "${BLUE}=== 网络环境检测 ===${NC}\n"

# 1. Check Docker Hub connectivity (test actual docker pull)
echo -n "检测 Docker 镜像拉取能力... "
if run_with_timeout 10 docker pull alpine:3.21 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 可拉取镜像${NC}"
    DOCKER_HUB_OK=true
else
    echo -e "${RED}✗ 无法拉取镜像${NC}"
fi

# 2. Check Alpine packages availability (test apk in Docker)
echo -n "检测 Alpine 软件包可用性... "
if run_with_timeout 10 docker run --rm alpine:3.21 apk update > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 可安装软件包${NC}"
    ALPINE_CDN_OK=true
else
    echo -e "${RED}✗ 无法安装软件包${NC}"
fi

# 3. Check npm registry
echo -n "检测 npm registry 连接... "
if run_with_timeout 5 curl -sf https://registry.npmjs.org/vue > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 可访问${NC}"
    NPM_REGISTRY_OK=true
else
    echo -e "${RED}✗ 无法访问${NC}"
fi

# 4. Check GitHub Actions availability
echo -n "检测 GitHub Actions 可用性... "
if command -v gh >/dev/null 2>&1 && gh auth status > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 已认证${NC}"
    GITHUB_ACTIONS_AVAILABLE=true
else
    echo -e "${YELLOW}⚠ 未认证或未安装 gh CLI${NC}"
fi

# 5. Check Docker daemon
echo -n "检测 Docker 守护进程... "
if docker info > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 运行中${NC}"
else
    echo -e "${RED}✗ 未运行${NC}"
    echo -e "${YELLOW}提示: 请启动 Docker Desktop${NC}"
    exit 1
fi

# 6. Check frontend build status
echo -n "检测前端构建状态... "
if [ -d "backend/internal/web/dist" ] && [ -f "backend/internal/web/dist/index.html" ]; then
    echo -e "${GREEN}✓ 已构建${NC}"
    FRONTEND_BUILT=true
else
    echo -e "${YELLOW}⚠ 未构建${NC}"
    FRONTEND_BUILT=false
fi

echo -e "\n${BLUE}=== 构建方案推荐 ===${NC}\n"

# Determine build strategy
if $DOCKER_HUB_OK && $ALPINE_CDN_OK && $NPM_REGISTRY_OK; then
    echo -e "${GREEN}✓ 网络环境良好${NC}"
    echo -e "推荐方案: ${GREEN}本地完整构建${NC}"
    echo -e "命令: ${BLUE}docker buildx build --platform linux/amd64 -t sub2api:amd64-hk --load .${NC}"
    exit 0
fi

if $FRONTEND_BUILT && $DOCKER_HUB_OK; then
    echo -e "${YELLOW}⚠ 部分网络受限，但前端已构建${NC}"
    echo -e "推荐方案: ${GREEN}使用预构建前端 (Dockerfile.prod)${NC}"
    echo -e "命令: ${BLUE}docker buildx build --platform linux/amd64 -t sub2api:amd64-hk -f Dockerfile.prod --load .${NC}"
    exit 0
fi

if ! $FRONTEND_BUILT && $NPM_REGISTRY_OK; then
    echo -e "${YELLOW}⚠ Docker 网络受限，但 npm 可用${NC}"
    echo -e "推荐方案: ${GREEN}先本地构建前端，再使用 Dockerfile.prod${NC}"
    echo -e "步骤:"
    echo -e "  1. ${BLUE}cd frontend && pnpm install && pnpm run build && cd ..${NC}"
    echo -e "  2. ${BLUE}docker buildx build --platform linux/amd64 -t sub2api:amd64-hk -f Dockerfile.prod --load .${NC}"
    exit 0
fi

if $GITHUB_ACTIONS_AVAILABLE; then
    echo -e "${RED}✗ 本地网络环境不适合构建${NC}"
    echo -e "推荐方案: ${GREEN}使用 GitHub Actions 自动部署${NC}"
    echo -e "步骤:"
    echo -e "  1. 配置 GitHub Secret: ${BLUE}HK_SERVER_SSH_KEY${NC}"
    echo -e "     ${BLUE}gh secret set HK_SERVER_SSH_KEY < ~/work/sub2api.pem${NC}"
    echo -e "  2. 推送代码触发部署:"
    echo -e "     ${BLUE}git checkout main && git merge dev && git push origin main${NC}"
    echo -e "  3. 查看部署进度:"
    echo -e "     ${BLUE}gh run watch${NC}"
    exit 0
fi

echo -e "${RED}✗ 所有构建方案都不可用${NC}"
echo -e "建议:"
echo -e "  1. 检查网络连接和代理设置"
echo -e "  2. 配置 Docker 镜像源 (~/.docker/daemon.json)"
echo -e "  3. 配置 npm 镜像源 (npm config set registry)"
echo -e "  4. 或使用 GitHub Actions 进行远程构建"
exit 1
