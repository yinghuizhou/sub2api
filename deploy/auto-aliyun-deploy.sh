#!/bin/bash
# =============================================================================
# Sub2API 阿里云全自动部署脚本（交互式密码输入版）
# 目标服务器: 47.76.82.51
# 用法: ./auto-aliyun-deploy.sh
# =============================================================================

set -euo pipefail

SERVER_IP="47.76.82.51"
SSH_KEY="${HOME}/.ssh/zhouyinghui"

echo "=============================================="
echo "Sub2API 阿里云全自动部署"
echo "目标服务器: $SERVER_IP"
echo "=============================================="
echo ""

# 检查依赖
check_deps() {
    if ! command -v expect &> /dev/null; then
        echo "❌ 需要安装 expect 工具"
        echo ""
        echo "macOS 安装方式:"
        echo "  xcode-select --install"
        echo ""
        echo "Ubuntu/Debian 安装方式:"
        echo "  sudo apt-get install -y expect"
        echo ""
        exit 1
    fi
}

# 生成部署脚本
generate_deploy_script() {
    cat > /tmp/sub2api-deploy-remote.sh << 'REMOTESCRIPT'
#!/bin/bash
set -euo pipefail

INSTALL_DIR="/opt/sub2api"
HTTP_PORT=8080
SERVER_IP="47.76.82.51"
ADMIN_EMAIL="admin@sub2api.local"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 生成安全密钥
JWT_SECRET=$(openssl rand -hex 32)
TOTP_KEY=$(openssl rand -hex 32)
PG_PASS=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)
ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d '/+=' | head -c 16)

log_info "开始部署 Sub2API..."

# 保存凭证到临时文件（部署后删除）
mkdir -p /tmp/sub2api-deploy
cat > /tmp/sub2api-deploy/credentials.txt << EOF
部署时间: $(date -Iseconds)
服务器IP: $SERVER_IP
访问地址: http://$SERVER_IP:$HTTP_PORT
管理员邮箱: $ADMIN_EMAIL
管理员密码: $ADMIN_PASSWORD
JWT_SECRET: $JWT_SECRET
TOTP_ENCRYPTION_KEY: $TOTP_KEY
PostgreSQL密码: $PG_PASS
EOF
chmod 600 /tmp/sub2api-deploy/credentials.txt

echo ""
echo "=========================================="
echo "📋 请立即保存以下信息："
echo "=========================================="
echo "访问地址: http://$SERVER_IP:$HTTP_PORT"
echo "管理员邮箱: $ADMIN_EMAIL"
echo "管理员密码: $ADMIN_PASSWORD"
echo "=========================================="
echo ""

# 更新系统
log_info "[1/9] 更新系统..."
apt-get update -qq > /dev/null 2>&1

# 安装 Docker
log_info "[2/9] 安装 Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sh > /dev/null 2>&1
    systemctl start docker
    systemctl enable docker > /dev/null 2>&1
fi

# 安装 Docker Compose
if ! docker compose version &> /dev/null; then
    apt-get install -y docker-compose-plugin -qq > /dev/null 2>&1
fi

# 设置时区
log_info "[3/9] 设置时区..."
timedatectl set-timezone Asia/Shanghai

# 创建目录
log_info "[4/9] 创建目录结构..."
mkdir -p "$INSTALL_DIR"/{data,backup,logs}
if ! id -u sub2api &>/dev/null; then
    useradd -r -s /bin/false -d "$INSTALL_DIR" sub2api
fi
chown -R sub2api:sub2api "$INSTALL_DIR"
chmod 750 "$INSTALL_DIR"

# 生成配置文件
log_info "[5/9] 生成配置文件..."
cd "$INSTALL_DIR"

# 生成 .env 文件
cat > .env << EOF
BIND_HOST=0.0.0.0
SERVER_PORT=$HTTP_PORT
SERVER_MODE=release
TZ=Asia/Shanghai
SERVER_MAX_REQUEST_BODY_SIZE=104857600
SERVER_H2C_ENABLED=true
SERVER_H2C_MAX_CONCURRENT_STREAMS=50
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=$PG_PASS
POSTGRES_DB=sub2api
REDIS_PASSWORD=
REDIS_DB=0
ADMIN_EMAIL=$ADMIN_EMAIL
ADMIN_PASSWORD=$ADMIN_PASSWORD
JWT_SECRET=$JWT_SECRET
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=$TOTP_KEY
GATEWAY_SCHEDULING_STICKY_SESSION_MAX_WAITING=3
GATEWAY_SCHEDULING_STICKY_SESSION_WAIT_TIMEOUT=120s
GATEWAY_SCHEDULING_LOAD_BATCH_ENABLED=true
DASHBOARD_AGGREGATION_ENABLED=true
DASHBOARD_AGGREGATION_INTERVAL_SECONDS=60
DASHBOARD_AGGREGATION_RETENTION_USAGE_LOGS_DAYS=90
DASHBOARD_AGGREGATION_RETENTION_HOURLY_DAYS=180
DASHBOARD_AGGREGATION_RETENTION_DAILY_DAYS=730
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=true
OPS_ENABLED=true
UPDATE_PROXY_URL=
EOF
chmod 600 .env

# 创建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  sub2api:
    image: weishaw/sub2api:latest
    container_name: sub2api
    restart: unless-stopped
    ulimits:
      nofile:
        soft: 100000
        hard: 100000
    ports:
      - "${BIND_HOST:-0.0.0.0}:${SERVER_PORT:-8080}:8080"
    volumes:
      - sub2api_data:/app/data
    environment:
      - AUTO_SETUP=true
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
      - SERVER_MODE=${SERVER_MODE:-release}
      - RUN_MODE=${RUN_MODE:-standard}
      - DATABASE_HOST=postgres
      - DATABASE_PORT=5432
      - DATABASE_USER=${POSTGRES_USER:-sub2api}
      - DATABASE_PASSWORD=${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
      - DATABASE_DBNAME=${POSTGRES_DB:-sub2api}
      - DATABASE_SSLMODE=disable
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD:-}
      - REDIS_DB=${REDIS_DB:-0}
      - REDIS_ENABLE_TLS=${REDIS_ENABLE_TLS:-false}
      - ADMIN_EMAIL=${ADMIN_EMAIL:-admin@sub2api.local}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-}
      - JWT_SECRET=${JWT_SECRET:-}
      - JWT_EXPIRE_HOUR=${JWT_EXPIRE_HOUR:-24}
      - TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY:-}
      - TZ=${TZ:-Asia/Shanghai}
      - SECURITY_URL_ALLOWLIST_ENABLED=${SECURITY_URL_ALLOWLIST_ENABLED:-false}
      - SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=${SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP:-false}
      - SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=${SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS:-false}
      - UPDATE_PROXY_URL=${UPDATE_PROXY_URL:-}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - sub2api-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

  postgres:
    image: postgres:15-alpine
    container_name: sub2api-postgres
    restart: unless-stopped
    ulimits:
      nofile:
        soft: 100000
        hard: 100000
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=${POSTGRES_USER:-sub2api}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
      - POSTGRES_DB=${POSTGRES_DB:-sub2api}
      - PGDATA=/var/lib/postgresql/data
      - TZ=${TZ:-Asia/Shanghai}
    networks:
      - sub2api-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-sub2api} -d ${POSTGRES_DB:-sub2api}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  redis:
    image: redis:7-alpine
    container_name: sub2api-redis
    restart: unless-stopped
    ulimits:
      nofile:
        soft: 100000
        hard: 100000
    volumes:
      - redis_data:/data
    command: >
        sh -c '
          redis-server
          --save 60 1
          --appendonly yes
          --appendfsync everysec
          ${REDIS_PASSWORD:+--requirepass "$REDIS_PASSWORD"}'
    environment:
      - TZ=${TZ:-Asia/Shanghai}
      - REDISCLI_AUTH=${REDIS_PASSWORD:-}
    networks:
      - sub2api-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 5s

volumes:
  sub2api_data:
    driver: local
  postgres_data:
    driver: local
  redis_data:
    driver: local

networks:
  sub2api-network:
    driver: bridge
EOF

# 配置防火墙
log_info "[6/9] 配置防火墙..."
ufw default deny incoming > /dev/null 2>&1 || true
ufw default allow outgoing > /dev/null 2>&1 || true
ufw allow 22/tcp comment 'SSH' > /dev/null 2>&1 || true
ufw allow 80/tcp comment 'HTTP' > /dev/null 2>&1 || true
ufw allow 443/tcp comment 'HTTPS' > /dev/null 2>&1 || true
ufw allow 8080/tcp comment 'Sub2API' > /dev/null 2>&1 || true
ufw --force enable > /dev/null 2>&1 || true

# 启动服务
log_info "[7/9] 拉取 Docker 镜像..."
docker compose pull -q

log_info "[8/9] 启动服务..."
docker compose up -d

# 等待服务就绪
log_info "[9/9] 等待服务就绪（约 30 秒）..."
for i in {1..30}; do
    if curl -sf http://localhost:$HTTP_PORT/health > /dev/null 2>&1; then
        log_success "✅ Sub2API 服务已就绪！"
        break
    fi
    echo -n "."
    sleep 2
done

# 验证服务
if curl -sf http://localhost:$HTTP_PORT/health > /dev/null 2>&1; then
    echo ""
    echo ""
    echo "=========================================="
    echo "🎉 部署成功！"
    echo "=========================================="
    echo ""
    echo "🌐 访问地址: http://$SERVER_IP:$HTTP_PORT"
    echo "👤 管理员邮箱: $ADMIN_EMAIL"
    echo "🔑 管理员密码: $ADMIN_PASSWORD"
    echo ""
    echo "⚠️  请立即保存以下信息："
    echo "   - 管理员密码: $ADMIN_PASSWORD"
    echo "   - JWT_SECRET: $JWT_SECRET"
    echo "   - TOTP密钥: $TOTP_KEY"
    echo ""
    echo "📋 完整凭证文件: /tmp/sub2api-deploy/credentials.txt"
    echo ""
    echo "常用命令："
    echo "  查看日志: cd $INSTALL_DIR && docker compose logs -f"
    echo "  重启服务: cd $INSTALL_DIR && docker compose restart"
    echo "  停止服务: cd $INSTALL_DIR && docker compose down"
    echo ""
    echo "=========================================="
else
    log_error "❌ 服务启动超时"
    echo "请检查日志: cd $INSTALL_DIR && docker compose logs"
    exit 1
fi

# 移动凭证文件到安全位置
mkdir -p $INSTALL_DIR
mv /tmp/sub2api-deploy/credentials.txt $INSTALL_DIR/
chmod 600 $INSTALL_DIR/credentials.txt

REMOTESCRIPT
    chmod +x /tmp/sub2api-deploy-remote.sh
}

# 使用 expect 自动输入密码执行
deploy_with_password() {
    echo "请输入服务器 root 密码（输入时不会显示）："
    read -s SERVER_PASSWORD
    echo ""

    if [ -z "$SERVER_PASSWORD" ]; then
        echo "❌ 密码不能为空"
        exit 1
    fi

    # 检查密钥是否存在
    if [ ! -f "$SSH_KEY" ]; then
        echo "❌ SSH 密钥文件不存在: $SSH_KEY"
        echo "请确认密钥路径，或使用密码方式:"
        echo "  ssh-copy-id -i ~/.ssh/你的密钥 root@$SERVER_IP"
        exit 1
    fi

    # 生成远程脚本
    generate_deploy_script

    echo "🚀 开始部署..."
    echo ""

    # 复制脚本到服务器
    expect -c "
        set timeout 30
        spawn scp -i $SSH_KEY -o StrictHostKeyChecking=no /tmp/sub2api-deploy-remote.sh root@$SERVER_IP:/tmp/
        expect {
            \"password:\" {
                send \"$SERVER_PASSWORD\r\"
                exp_continue
            }
            \"Permission denied\" {
                puts \"❌ 密码错误或认证失败\"
                exit 1
            }
            timeout {
                puts \"❌ 连接超时\"
                exit 1
            }
            eof
        }
    "

    if [ $? -ne 0 ]; then
        echo "❌ 复制脚本失败"
        exit 1
    fi

    # 在服务器上执行脚本
    echo "📤 正在服务器上执行部署..."
    expect -c "
        set timeout 600
        spawn ssh -i $SSH_KEY -o StrictHostKeyChecking=no root@$SERVER_IP \"bash /tmp/sub2api-deploy-remote.sh\"
        expect {
            \"password:\" {
                send \"$SERVER_PASSWORD\r\"
                exp_continue
            }
            \"Permission denied\" {
                puts \"❌ 密码错误或认证失败\"
                exit 1
            }
            timeout {
                puts \"❌ 执行超时\"
                exit 1
            }
            eof
        }
    "

    DEPLOY_STATUS=$?

    # 清理本地临时文件
    rm -f /tmp/sub2api-deploy-remote.sh

    if [ $DEPLOY_STATUS -eq 0 ]; then
        echo ""
        echo "✅ 部署完成！"
        echo ""
        echo "🌐 访问地址: http://$SERVER_IP:8080"
        echo ""
        echo "如需查看服务器上的凭证文件，请执行："
        echo "  ssh root@$SERVER_IP 'cat /opt/sub2api/credentials.txt'"
    else
        echo ""
        echo "❌ 部署可能失败，请检查上述输出"
        exit 1
    fi
}

# 主函数
main() {
    check_deps
    deploy_with_password
}

main "$@"
