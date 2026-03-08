#!/bin/bash
set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
HK_SERVER="47.76.82.51"
PEM_FILE="$HOME/work/sub2api.pem"
DEPLOY_DIR="/opt/sub2api"
IMAGE_NAME="sub2api"
DATE_TAG=$(date +%Y%m%d-%H%M%S)

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Sub2API 手动部署脚本${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 步骤 0: 检查网络环境
echo -e "${YELLOW}步骤 0/6: 检查网络环境${NC}"
if ./scripts/check-network.sh; then
    echo -e "${GREEN}网络环境检测通过${NC}"
    # 根据检测结果选择 Dockerfile
    if [ -f "backend/internal/web/dist/index.html" ]; then
        DOCKERFILE="Dockerfile.prod"
        echo -e "${BLUE}使用预构建前端: ${DOCKERFILE}${NC}"
    else
        DOCKERFILE="Dockerfile"
        echo -e "${BLUE}使用完整构建: ${DOCKERFILE}${NC}"
    fi
else
    echo -e "${RED}网络环境检测失败，请根据建议调整后重试${NC}"
    exit 1
fi
echo ""

# 检查 Docker 是否运行
echo -n "检查 Docker 状态... "
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}✗${NC}"
    echo "Docker 未运行，请先启动 Docker Desktop"
    exit 1
fi
echo -e "${GREEN}✓${NC}"

# 检查 SSH 连接
echo -n "检查 SSH 连接... "
if ! ssh -i "$PEM_FILE" -o ConnectTimeout=5 -o StrictHostKeyChecking=no root@$HK_SERVER "echo ok" >/dev/null 2>&1; then
    echo -e "${RED}✗${NC}"
    echo "无法连接到 HK 服务器"
    exit 1
fi
echo -e "${GREEN}✓${NC}"

# 构建镜像
echo ""
echo -e "${YELLOW}步骤 1/6: 构建 Docker 镜像${NC}"
echo "镜像标签: ${IMAGE_NAME}:amd64-hk-${DATE_TAG}"
echo "使用 Dockerfile: ${DOCKERFILE}"
docker buildx build \
    --platform linux/amd64 \
    -t ${IMAGE_NAME}:amd64-hk-${DATE_TAG} \
    -f "${DOCKERFILE}" \
    --load \
    .

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ 镜像构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 镜像构建成功${NC}"

# 导出镜像
echo ""
echo -e "${YELLOW}步骤 2/6: 导出镜像${NC}"
TARBALL="/tmp/${IMAGE_NAME}-amd64-hk-${DATE_TAG}.tar.gz"
docker save ${IMAGE_NAME}:amd64-hk-${DATE_TAG} | gzip > ${TARBALL}

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ 镜像导出失败${NC}"
    exit 1
fi

TARBALL_SIZE=$(du -h ${TARBALL} | cut -f1)
echo -e "${GREEN}✓ 镜像已导出: ${TARBALL} (${TARBALL_SIZE})${NC}"

# 上传镜像
echo ""
echo -e "${YELLOW}步骤 3/6: 上传镜像到 HK 服务器${NC}"
scp -i "$PEM_FILE" ${TARBALL} root@${HK_SERVER}:${DEPLOY_DIR}/

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ 镜像上传失败${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 镜像上传成功${NC}"

# 滚动更新
echo ""
echo -e "${YELLOW}步骤 4/6: 执行滚动更新（零停机）${NC}"
ssh -i "$PEM_FILE" root@${HK_SERVER} << EOF
set -e

cd ${DEPLOY_DIR}

# 加载新镜像
echo "加载新镜像..."
docker load < ${IMAGE_NAME}-amd64-hk-${DATE_TAG}.tar.gz

# 标记为 latest
docker tag ${IMAGE_NAME}:amd64-hk-${DATE_TAG} ${IMAGE_NAME}:latest

# 滚动更新 3 个实例
for i in 1 2 3; do
    echo ""
    echo "更新实例 \${i}..."
    docker compose stop sub2api-\${i}
    docker compose up -d sub2api-\${i}

    echo "等待 15 秒..."
    sleep 15

    # 健康检查
    if curl -sf http://localhost:8888/health > /dev/null; then
        echo "✓ 实例 \${i} 健康检查通过"
    else
        echo "✗ 实例 \${i} 健康检查失败"
        exit 1
    fi
done

echo ""
echo "所有实例更新完成"
EOF

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ 滚动更新失败${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 滚动更新成功${NC}"

# 验证部署
echo ""
echo -e "${YELLOW}步骤 5/6: 验证部署${NC}"
ssh -i "$PEM_FILE" root@${HK_SERVER} << 'EOF'
set -e

echo "检查服务健康..."
curl -sf http://localhost:8888/health

echo ""
echo "检查 metrics 端点..."
curl -sf http://localhost:8888/metrics | head -10

echo ""
echo "检查容器状态..."
docker compose ps sub2api

echo ""
echo "检查版本信息..."
docker exec sub2api-sub2api-1 /app/sub2api -version 2>&1 | grep "Sub2API"
EOF

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ 部署验证失败${NC}"
    exit 1
fi

# 清理
echo ""
echo -e "${YELLOW}步骤 6/6: 清理临时文件${NC}"
rm -f ${TARBALL}
echo -e "${GREEN}✓ 清理完成${NC}"

# 完成
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  部署成功！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "部署信息："
echo "  - 镜像标签: ${IMAGE_NAME}:amd64-hk-${DATE_TAG}"
echo "  - 使用 Dockerfile: ${DOCKERFILE}"
echo "  - 服务地址: http://47.76.82.51:8888"
echo "  - 健康检查: http://47.76.82.51:8888/health"
echo "  - 监控指标: http://47.76.82.51:8888/metrics"
echo ""
