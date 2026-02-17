#!/bin/bash
# =============================================================================
# Sub2API 阿里云服务器部署脚本
# 目标服务器: 47.76.82.51
# 部署方式: Docker Compose
# =============================================================================

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP $1/10]${NC} $2"; }

# 配置变量
INSTALL_DIR="/opt/sub2api"
HTTP_PORT=8080
ADMIN_EMAIL="admin@sub2api.local"
SERVER_IP="47.76.82.51"

# 生成安全密钥
generate_secrets() {
    JWT_SECRET=$(openssl rand -hex 32)
    TOTP_KEY=$(openssl rand -hex 32)
    PG_PASS=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)
    ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d '/+=' | head -c 16)
}

# 检查系统要求
check_prerequisites() {
    log_step 1 "检查系统要求"

    # 检查 root 权限
    if [[ $EUID -ne 0 ]]; then
        log_error "请使用 sudo 运行此脚本"
        exit 1
    fi

    # 检查系统架构
    ARCH=$(uname -m)
    if [[ "$ARCH" != "x86_64" && "$ARCH" != "aarch64" ]]; then
        log_error "不支持的架构: $ARCH"
        exit 1
    fi

    # 检查内存
    MEM_GB=$(free -g | awk '/^Mem:/{print $2}')
    log_info "系统内存: ${MEM_GB}GB"
    if [[ $MEM_GB -lt 2 ]]; then
        log_warn "内存不足 2GB，建议至少 4GB"
    fi

    # 检查磁盘空间
    DISK_GB=$(df -BG / | awk 'NR==2 {print $4}' | sed 's/G//')
    log_info "磁盘空间: ${DISK_GB}GB 可用"
    if [[ "$DISK_GB" -lt 10 ]]; then
        log_warn "磁盘空间不足 10GB，建议至少 20GB"
    fi

    log_success "系统检查通过"
}

# 更新系统
update_system() {
    log_step 2 "更新系统软件包"

    apt-get update && apt-get upgrade -y

    # 安装必要工具
    apt-get install -y curl wget git vim htop net-tools ufail2ban ca-certificates gnupg lsb-release

    log_success "系统更新完成"
}

# 安装 Docker
install_docker() {
    log_step 3 "安装 Docker"

    if command -v docker &> /dev/null; then
        log_success "Docker 已安装: $(docker --version)"
    else
        log_info "正在安装 Docker..."
        curl -fsSL https://get.docker.com | sh
        systemctl start docker
        systemctl enable docker
        log_success "Docker 安装完成: $(docker --version)"
    fi

    # 检查 Docker Compose
    if docker compose version &> /dev/null; then
        log_success "Docker Compose 已安装"
    else
        log_info "安装 Docker Compose..."
        apt-get install -y docker-compose-plugin
        log_success "Docker Compose 安装完成"
    fi
}

# 设置时区
setup_timezone() {
    log_step 4 "设置时区"

    timedatectl set-timezone Asia/Shanghai
    log_info "当前时区: $(timedatectl | grep "Time zone")"
}

# 创建目录结构
setup_directories() {
    log_step 5 "创建目录结构"

    mkdir -p "$INSTALL_DIR"/{data,backup,logs}

    # 创建非特权用户
    if ! id -u sub2api &>/dev/null; then
        useradd -r -s /bin/false -d "$INSTALL_DIR" sub2api
        log_info "创建用户: sub2api"
    fi

    chown -R sub2api:sub2api "$INSTALL_DIR"
    chmod 750 "$INSTALL_DIR"

    log_success "目录结构创建完成"
}

# 生成配置文件
generate_config() {
    log_step 6 "生成配置文件"

    cd "$INSTALL_DIR"

    # 生成环境变量文件
    cat > .env << EOF
# =============================================================================
# Sub2API 自动生成的配置文件
# 生成时间: $(date -Iseconds)
# 服务器IP: $SERVER_IP
# =============================================================================

# 服务器配置
BIND_HOST=0.0.0.0
SERVER_PORT=$HTTP_PORT
SERVER_MODE=release
TZ=Asia/Shanghai

# 性能调优
SERVER_MAX_REQUEST_BODY_SIZE=104857600
SERVER_H2C_ENABLED=true
SERVER_H2C_MAX_CONCURRENT_STREAMS=50

# 数据库配置
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=$PG_PASS
POSTGRES_DB=sub2api

# Redis 配置
REDIS_PASSWORD=
REDIS_DB=0

# 管理员账户
ADMIN_EMAIL=$ADMIN_EMAIL
ADMIN_PASSWORD=$ADMIN_PASSWORD

# 安全密钥（请勿泄露）
JWT_SECRET=$JWT_SECRET
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=$TOTP_KEY

# 网关调度配置
GATEWAY_SCHEDULING_STICKY_SESSION_MAX_WAITING=3
GATEWAY_SCHEDULING_STICKY_SESSION_WAIT_TIMEOUT=120s
GATEWAY_SCHEDULING_LOAD_BATCH_ENABLED=true

# 数据保留策略
DASHBOARD_AGGREGATION_ENABLED=true
DASHBOARD_AGGREGATION_INTERVAL_SECONDS=60
DASHBOARD_AGGREGATION_RETENTION_USAGE_LOGS_DAYS=90
DASHBOARD_AGGREGATION_RETENTION_HOURLY_DAYS=180
DASHBOARD_AGGREGATION_RETENTION_DAILY_DAYS=730

# 安全配置
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=true

# 监控配置
OPS_ENABLED=true

# 更新配置
UPDATE_PROXY_URL=
EOF

    chmod 600 .env

    # 创建 docker-compose.yml
    cat > docker-compose.yml << 'EOF'
# =============================================================================
# Sub2API Docker Compose Configuration
# =============================================================================

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
      - GEMINI_OAUTH_CLIENT_ID=${GEMINI_OAUTH_CLIENT_ID:-}
      - GEMINI_OAUTH_CLIENT_SECRET=${GEMINI_OAUTH_CLIENT_SECRET:-}
      - GEMINI_OAUTH_SCOPES=${GEMINI_OAUTH_SCOPES:-}
      - GEMINI_QUOTA_POLICY=${GEMINI_QUOTA_POLICY:-}
      - SECURITY_URL_ALLOWLIST_ENABLED=${SECURITY_URL_ALLOWLIST_ENABLED:-false}
      - SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=${SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP:-false}
      - SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=${SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS:-false}
      - SECURITY_URL_ALLOWLIST_UPSTREAM_HOSTS=${SECURITY_URL_ALLOWLIST_UPSTREAM_HOSTS:-}
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

    log_success "配置文件生成完成"
}

# 配置防火墙
setup_firewall() {
    log_step 7 "配置防火墙"

    if command -v ufw &> /dev/null; then
        ufw default deny incoming
        ufw default allow outgoing
        ufw allow 22/tcp comment 'SSH'
        ufw allow 80/tcp comment 'HTTP'
        ufw allow 443/tcp comment 'HTTPS'
        ufw allow 8080/tcp comment 'Sub2API'
        ufw --force enable
        log_success "UFW 防火墙配置完成"
        ufw status
    else
        log_warn "未检测到 UFW，请手动配置防火墙"
    fi
}

# 启动服务
start_services() {
    log_step 8 "启动 Sub2API 服务"

    cd "$INSTALL_DIR"

    # 拉取镜像
    log_info "拉取 Docker 镜像..."
    docker compose pull

    # 启动服务
    log_info "启动服务..."
    docker compose up -d

    # 等待服务就绪
    log_info "等待服务就绪（约 30 秒）..."
    for i in {1..30}; do
        if curl -sf http://localhost:${HTTP_PORT}/health > /dev/null 2>&1; then
            log_success "Sub2API 服务已就绪"
            return 0
        fi
        echo -n "."
        sleep 2
    done

    log_error "服务启动超时，请检查日志: docker compose logs"
    return 1
}

# 设置自动备份
setup_backup() {
    log_step 9 "配置自动备份"

    # 创建备份脚本
    cat > "$INSTALL_DIR/backup.sh" << 'EOF'
#!/bin/bash
set -euo pipefail

INSTALL_DIR="/opt/sub2api"
BACKUP_DIR="/opt/sub2api/backup"
RETENTION_DAYS=30

# 加载环境变量
source "$INSTALL_DIR/.env"

# 创建备份目录
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="$BACKUP_DIR/$BACKUP_DATE"
mkdir -p "$BACKUP_PATH"

cd "$INSTALL_DIR"

# 备份数据卷
docker run --rm -v sub2api_postgres_data:/data -v "$BACKUP_PATH":/backup alpine tar czf /backup/postgres_data.tar.gz -C /data .
docker run --rm -v sub2api_redis_data:/data -v "$BACKUP_PATH":/backup alpine tar czf /backup/redis_data.tar.gz -C /data .

# 备份环境变量
cp .env "$BACKUP_PATH/"

# 数据库逻辑备份
docker compose exec -T postgres pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" | gzip > "$BACKUP_PATH/db.sql.gz"

# 创建备份信息
echo "Backup completed at $(date -Iseconds)" > "$BACKUP_PATH/INFO.txt"

# 压缩整个备份
cd "$BACKUP_DIR"
tar czf "${BACKUP_DATE}.tar.gz" "$BACKUP_DATE"
rm -rf "$BACKUP_PATH"

# 清理旧备份
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: ${BACKUP_DATE}.tar.gz"
EOF

    chmod +x "$INSTALL_DIR/backup.sh"

    # 创建定时任务
    (crontab -l 2>/dev/null || true; echo "0 2 * * * $INSTALL_DIR/backup.sh >> $INSTALL_DIR/logs/backup.log 2>&1") | crontab -

    log_success "自动备份配置完成（每天凌晨2点执行）"
}

# 生成部署报告
generate_report() {
    log_step 10 "生成部署报告"

    cd "$INSTALL_DIR"

    # 获取容器状态
    CONTAINER_STATUS=$(docker compose ps --format "table {{.Service}}\t{{.Status}}\t{{.Ports}}")

    cat > "$INSTALL_DIR/install-report.txt" << EOF
=============================================================================
Sub2API 阿里云服务器部署报告
=============================================================================
部署时间: $(date -Iseconds)
服务器IP: $SERVER_IP
安装目录: $INSTALL_DIR

【访问信息】
访问地址: http://$SERVER_IP:$HTTP_PORT
管理员邮箱: $ADMIN_EMAIL
管理员密码: $ADMIN_PASSWORD

【安全密钥】（请妥善保管）
JWT_SECRET: $JWT_SECRET
TOTP_ENCRYPTION_KEY: $TOTP_KEY
PostgreSQL密码: $PG_PASS

【服务状态】
$CONTAINER_STATUS

【常用命令】
查看日志:    cd $INSTALL_DIR && docker compose logs -f
重启服务:    cd $INSTALL_DIR && docker compose restart
停止服务:    cd $INSTALL_DIR && docker compose down
备份数据:    $INSTALL_DIR/backup.sh
更新版本:    cd $INSTALL_DIR && docker compose pull && docker compose up -d

【文件位置】
配置文件:    $INSTALL_DIR/.env
docker配置:  $INSTALL_DIR/docker-compose.yml
数据目录:    $INSTALL_DIR/data/
备份目录:    $INSTALL_DIR/backup/
日志目录:    $INSTALL_DIR/logs/

【监控告警】
自动备份:    每天凌晨2点执行

=============================================================================
重要提示：
1. 请立即保存管理员密码和安全密钥到密码管理器
2. 首次登录后建议修改管理员密码
3. 生产环境建议配置 HTTPS
4. 建议配置异地备份
=============================================================================
EOF

    chmod 600 "$INSTALL_DIR/install-report.txt"

    log_success "部署报告已保存到: $INSTALL_DIR/install-report.txt"
}

# 显示最终结果
show_result() {
    echo ""
    echo "============================================================================="
    echo "                          🎉 Sub2API 部署成功！"
    echo "============================================================================="
    echo ""
    echo -e "  🌐 访问地址: ${CYAN}http://$SERVER_IP:$HTTP_PORT${NC}"
    echo -e "  👤 管理员邮箱: ${CYAN}$ADMIN_EMAIL${NC}"
    echo -e "  🔑 管理员密码: ${CYAN}$ADMIN_PASSWORD${NC}"
    echo ""
    echo -e "  ${YELLOW}⚠️  请立即保存以下信息到安全位置：${NC}"
    echo "     - 管理员密码"
    echo "     - JWT_SECRET"
    echo "     - TOTP_ENCRYPTION_KEY"
    echo "     - PostgreSQL 密码"
    echo ""
    echo -e "  📋 完整报告: ${CYAN}$INSTALL_DIR/install-report.txt${NC}"
    echo ""
    echo "============================================================================="
}

# 清理函数
cleanup() {
    if [[ $? -ne 0 ]]; then
        log_error "部署过程中出现错误"
        log_info "查看日志: cd $INSTALL_DIR && docker compose logs"
    fi
}
trap cleanup EXIT

# 主函数
main() {
    echo "============================================================================="
    echo "                    Sub2API 阿里云服务器部署脚本"
    echo "                    目标: $SERVER_IP"
    echo "============================================================================="
    echo ""

    # 生成密钥
    generate_secrets

    # 执行部署步骤
    check_prerequisites
    update_system
    install_docker
    setup_timezone
    setup_directories
    generate_config
    setup_firewall
    start_services
    setup_backup
    generate_report

    show_result

    log_success "部署完成！"
}

# 执行主函数
main "$@"
