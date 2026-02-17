#!/bin/bash
# Sub2API 阿里云服务器初始化脚本
# 服务器: 47.76.82.51
# 用途: CI/CD 部署准备

set -e

echo "=============================================="
echo "Sub2API 阿里云服务器初始化脚本"
echo "服务器: 47.76.82.51"
echo "=============================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查root权限
if [ "$EUID" -ne 0 ]; then
   echo -e "${RED}请使用 root 权限运行此脚本${NC}"
   exit 1
fi

# 设置变量
DEPLOY_PATH="/opt/sub2api"
GITHUB_ACTOR="${1:-wei-shaw}"

echo ""
echo "步骤 1/10: 更新系统..."
apt update && apt upgrade -y

echo ""
echo "步骤 2/10: 安装必要工具..."
apt install -y curl wget git vim htop net-tools ufw fail2ban

echo ""
echo "步骤 3/10: 设置时区..."
timedatectl set-timezone Asia/Shanghai

echo ""
echo "步骤 4/10: 安装 Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sh
    systemctl start docker
    systemctl enable docker
    echo -e "${GREEN}Docker 安装完成: $(docker --version)${NC}"
else
    echo -e "${GREEN}Docker 已安装: $(docker --version)${NC}"
fi

echo ""
echo "步骤 5/10: 安装 Docker Compose..."
if ! docker compose version &> /dev/null; then
    apt install -y docker-compose-plugin
    echo -e "${GREEN}Docker Compose 安装完成${NC}"
else
    echo -e "${GREEN}Docker Compose 已安装${NC}"
fi

echo ""
echo "步骤 6/10: 创建部署目录..."
mkdir -p $DEPLOY_PATH
cd $DEPLOY_PATH

echo ""
echo "步骤 7/10: 下载部署文件..."
if [ ! -f "docker-compose.local.yml" ]; then
    curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/docker-compose.local.yml -o docker-compose.local.yml
    echo -e "${GREEN}docker-compose.local.yml 下载完成${NC}"
else
    echo -e "${YELLOW}docker-compose.local.yml 已存在，跳过${NC}"
fi

echo ""
echo "步骤 8/10: 生成环境变量..."
if [ ! -f ".env" ]; then
    # 生成随机密码
    POSTGRES_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)
    JWT_SECRET=$(openssl rand -hex 32)
    TOTP_KEY=$(openssl rand -hex 32)
    ADMIN_PASS=$(openssl rand -base64 12 | tr -d '/+=' | head -c 16)

    cat > .env << EOF
# 服务器配置
SERVER_PORT=8080
SERVER_MODE=release
RUN_MODE=standard
TZ=Asia/Shanghai

# 数据库配置
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=sub2api

# Redis 配置
REDIS_PASSWORD=
REDIS_DB=0

# 管理员配置
ADMIN_EMAIL=admin@sub2api.local
ADMIN_PASSWORD=${ADMIN_PASS}

# 安全配置
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=${TOTP_KEY}

# URL 白名单
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=true

# 运维监控
OPS_ENABLED=true
EOF

    echo -e "${GREEN}.env 文件已创建${NC}"
    echo ""
    echo "=============================================="
    echo "重要信息，请保存："
    echo "=============================================="
    echo -e "PostgreSQL 密码: ${YELLOW}${POSTGRES_PASSWORD}${NC}"
    echo -e "JWT 密钥: ${YELLOW}${JWT_SECRET}${NC}"
    echo -e "TOTP 密钥: ${YELLOW}${TOTP_KEY}${NC}"
    echo -e "管理员密码: ${YELLOW}${ADMIN_PASS}${NC}"
    echo "=============================================="
    echo ""
else
    echo -e "${YELLOW}.env 文件已存在，跳过${NC}"
fi

echo ""
echo "步骤 9/10: 配置 SSH 密钥（用于 GitHub Actions）..."
if [ ! -f "/root/.ssh/github_actions" ]; then
    ssh-keygen -t ed25519 -C "github-actions-deploy" -f /root/.ssh/github_actions -N ""
    cat /root/.ssh/github_actions.pub >> /root/.ssh/authorized_keys
    chmod 600 /root/.ssh/authorized_keys
    echo -e "${GREEN}SSH 密钥已生成${NC}"
    echo ""
    echo "=============================================="
    echo "请将以下私钥添加到 GitHub Secrets:"
    echo "Secret 名称: ALIYUN_SSH_KEY"
    echo "=============================================="
    cat /root/.ssh/github_actions
    echo "=============================================="
    echo ""
else
    echo -e "${YELLOW}SSH 密钥已存在，跳过${NC}"
fi

echo ""
echo "步骤 10/10: 配置防火墙..."
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp comment 'SSH'
ufw allow 80/tcp comment 'HTTP'
ufw allow 443/tcp comment 'HTTPS'
ufw allow 8080/tcp comment 'Sub2API'
ufw --force enable

echo -e "${GREEN}防火墙配置完成${NC}"
ufw status

echo ""
echo "=============================================="
echo "初始化完成！"
echo "=============================================="
echo ""
echo "下一步操作："
echo "1. 保存上面显示的密码信息"
echo "2. 将 SSH 私钥添加到 GitHub Secrets"
echo "3. 推送代码触发 CI/CD 部署"
echo ""
echo "访问地址: http://47.76.82.51:8080"
echo ""
