#!/bin/bash
# =============================================================================
# Sub2API Google Cloud Platform 部署脚本
# =============================================================================
# 在 GCP Compute Engine (asia-east1) 上部署 Sub2API
# 使用 Docker Compose 运行完整服务栈（Sub2API + PostgreSQL + Redis）
#
# 用法:
#   第一步 (本地): 创建 GCP 资源
#     ./gcp-deploy.sh setup [--region REGION] [--zone ZONE] [--machine-type TYPE]
#
#   第二步 (VM 内): 部署应用
#     ./gcp-deploy.sh deploy [--domain DOMAIN] [--email EMAIL]
#
#   一键全自动 (本地运行，自动 SSH 到 VM 完成部署):
#     ./gcp-deploy.sh auto [--domain DOMAIN] [--region REGION] [--machine-type TYPE]
#
#   分步执行:
#     ./gcp-deploy.sh setup        # 第一步: 创建 GCP 资源 (本地)
#     ./gcp-deploy.sh deploy       # 第二步: 部署应用 (VM 内, sudo)
#
#   其他命令:
#     ./gcp-deploy.sh ssh          # SSH 连接到 VM
#     ./gcp-deploy.sh status       # 查看服务状态
#     ./gcp-deploy.sh destroy      # 删除所有 GCP 资源 (危险!)
#     ./gcp-deploy.sh help         # 显示帮助
#
# 预估费用:
#   e2-small (asia-east1): ~$15/月
#   e2-micro (asia-east1): ~$7/月 (适合极低流量)
#   e2-medium (asia-east1): ~$28/月 (适合中等流量)
#
# 环境变量覆盖:
#   GCP_PROJECT_ID    - GCP 项目 ID (默认: sub2api-$(date +%Y%m))
#   GCP_REGION        - GCP 区域 (默认: asia-east1)
#   GCP_ZONE          - GCP 可用区 (默认: asia-east1-a)
#   GCP_MACHINE_TYPE  - 机器类型 (默认: e2-small)
#   GCP_DISK_SIZE     - 磁盘大小 GB (默认: 20)
#   VM_NAME           - VM 实例名称 (默认: sub2api)
# =============================================================================

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()    { echo -e "${CYAN}[STEP]${NC} $1"; }

# ===================== 默认配置 =====================
GCP_PROJECT_ID="${GCP_PROJECT_ID:-}"
GCP_REGION="${GCP_REGION:-asia-east1}"
GCP_ZONE="${GCP_ZONE:-asia-east1-a}"
GCP_MACHINE_TYPE="${GCP_MACHINE_TYPE:-e2-small}"
GCP_DISK_SIZE="${GCP_DISK_SIZE:-20}"
VM_NAME="${VM_NAME:-sub2api}"
INSTALL_DIR="/opt/sub2api"

# deploy 命令的参数
DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-admin@example.com}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@sub2api.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
HTTP_PORT=8080

# ===================== 工具函数 =====================

check_gcloud() {
    if ! command -v gcloud &>/dev/null; then
        log_error "gcloud CLI 未安装"
        echo ""
        echo "请先安装 gcloud CLI:"
        echo "  macOS:  brew install google-cloud-sdk"
        echo "  Linux:  curl https://sdk.cloud.google.com | bash"
        echo "  详情:   https://cloud.google.com/sdk/docs/install"
        echo ""
        echo "安装后运行: gcloud auth login"
        exit 1
    fi

    # 检查是否已登录
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | head -1 | grep -q .; then
        log_error "gcloud 未登录"
        echo "请运行: gcloud auth login"
        exit 1
    fi

    log_success "gcloud CLI 已就绪 ($(gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null | head -1))"
}

ensure_project() {
    if [[ -z "$GCP_PROJECT_ID" ]]; then
        # 尝试使用当前配置的项目
        GCP_PROJECT_ID=$(gcloud config get-value project 2>/dev/null || true)
    fi

    if [[ -z "$GCP_PROJECT_ID" ]]; then
        log_warn "未设置 GCP 项目 ID"
        echo ""
        echo "你可以:"
        echo "  1. 设置环境变量: export GCP_PROJECT_ID=your-project-id"
        echo "  2. 创建新项目:   gcloud projects create your-project-id"
        echo "  3. 选择已有项目: gcloud config set project your-project-id"
        echo ""
        echo "已有项目列表:"
        gcloud projects list --format="table(projectId,name)" 2>/dev/null || true
        exit 1
    fi

    gcloud config set project "$GCP_PROJECT_ID" 2>/dev/null
    log_success "使用项目: $GCP_PROJECT_ID"
}

# ===================== setup 命令 =====================

cmd_setup() {
    echo "============================================================================="
    echo "  Sub2API GCP 部署 - 第一步: 创建云资源"
    echo "============================================================================="
    echo ""

    check_gcloud
    ensure_project

    echo ""
    log_info "部署配置:"
    echo "  项目:     $GCP_PROJECT_ID"
    echo "  区域:     $GCP_REGION"
    echo "  可用区:   $GCP_ZONE"
    echo "  机器类型: $GCP_MACHINE_TYPE"
    echo "  磁盘:     ${GCP_DISK_SIZE}GB"
    echo "  实例名:   $VM_NAME"
    echo ""

    read -p "确认创建? (y/N): " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "已取消"
        exit 0
    fi

    echo ""

    # 1. 启用 API
    log_step "1/4 启用 Compute Engine API..."
    gcloud services enable compute.googleapis.com --quiet
    log_success "API 已启用"

    # 2. 创建防火墙规则
    log_step "2/4 配置防火墙规则..."

    # HTTP
    if ! gcloud compute firewall-rules describe allow-http --quiet &>/dev/null; then
        gcloud compute firewall-rules create allow-http \
            --allow=tcp:80 \
            --target-tags=http-server \
            --description="Allow HTTP traffic" \
            --quiet
        log_success "HTTP 防火墙规则已创建"
    else
        log_info "HTTP 防火墙规则已存在，跳过"
    fi

    # HTTPS
    if ! gcloud compute firewall-rules describe allow-https --quiet &>/dev/null; then
        gcloud compute firewall-rules create allow-https \
            --allow=tcp:443 \
            --target-tags=https-server \
            --description="Allow HTTPS traffic" \
            --quiet
        log_success "HTTPS 防火墙规则已创建"
    else
        log_info "HTTPS 防火墙规则已存在，跳过"
    fi

    # 3. 创建 VM 实例
    log_step "3/4 创建 VM 实例..."

    if gcloud compute instances describe "$VM_NAME" --zone="$GCP_ZONE" --quiet &>/dev/null; then
        log_warn "VM '$VM_NAME' 已存在于 $GCP_ZONE"
        read -p "是否跳过创建? (Y/n): " -r
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            log_info "跳过 VM 创建"
        else
            log_error "请先删除已有 VM 或使用不同名称"
            exit 1
        fi
    else
        gcloud compute instances create "$VM_NAME" \
            --zone="$GCP_ZONE" \
            --machine-type="$GCP_MACHINE_TYPE" \
            --boot-disk-size="${GCP_DISK_SIZE}GB" \
            --boot-disk-type=pd-standard \
            --image-family=ubuntu-2404-lts-amd64 \
            --image-project=ubuntu-os-cloud \
            --tags=http-server,https-server \
            --network-tier=STANDARD \
            --metadata=startup-script='#!/bin/bash
# 预安装 Docker (后台执行，加速后续部署)
if ! command -v docker &>/dev/null; then
    curl -fsSL https://get.docker.com | sh
fi' \
            --quiet

        log_success "VM 创建成功"
    fi

    # 4. 获取 VM 信息
    log_step "4/4 获取 VM 信息..."

    VM_IP=$(gcloud compute instances describe "$VM_NAME" \
        --zone="$GCP_ZONE" \
        --format='value(networkInterfaces[0].accessConfigs[0].natIP)')

    echo ""
    echo "============================================================================="
    echo "  GCP 资源创建完成!"
    echo "============================================================================="
    echo ""
    echo "  VM 外部 IP:  $VM_IP"
    echo "  VM 名称:     $VM_NAME"
    echo "  区域/可用区: $GCP_ZONE"
    echo "  机器类型:    $GCP_MACHINE_TYPE"
    echo ""
    echo "  下一步操作:"
    echo ""
    echo "  1. SSH 连接到 VM:"
    echo "     gcloud compute ssh $VM_NAME --zone=$GCP_ZONE"
    echo ""
    echo "  2. 在 VM 中运行部署命令 (等待 1-2 分钟让 Docker 预安装完成):"
    echo "     curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/gcp-deploy.sh -o gcp-deploy.sh"
    echo "     chmod +x gcp-deploy.sh"
    echo "     sudo ./gcp-deploy.sh deploy"
    echo ""
    echo "  或者使用域名部署:"
    echo "     sudo ./gcp-deploy.sh deploy --domain api.yourdomain.com --email you@email.com"
    echo ""
    echo "  预估月费: ~\$$(get_estimated_cost)/月"
    echo "============================================================================="
}

# ===================== auto 命令 (一键全自动) =====================

cmd_auto() {
    echo "============================================================================="
    echo "  Sub2API GCP 一键自动部署"
    echo "============================================================================="
    echo ""

    check_gcloud
    ensure_project

    echo ""
    log_info "部署配置:"
    echo "  项目:     $GCP_PROJECT_ID"
    echo "  区域:     $GCP_REGION ($GCP_ZONE)"
    echo "  机器类型: $GCP_MACHINE_TYPE (磁盘 ${GCP_DISK_SIZE}GB)"
    echo "  实例名:   $VM_NAME"
    [[ -n "$DOMAIN" ]] && echo "  域名:     $DOMAIN"
    echo "  预估月费: ~\$$(get_estimated_cost)/月"
    echo ""

    read -p "确认开始自动部署? (y/N): " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "已取消"
        exit 0
    fi

    echo ""
    local TOTAL_STEPS=7

    # ---- 1. 启用 API ----
    log_step "1/${TOTAL_STEPS} 启用 Compute Engine API..."
    gcloud services enable compute.googleapis.com --quiet
    log_success "API 已启用"

    # ---- 2. 防火墙 ----
    log_step "2/${TOTAL_STEPS} 配置防火墙规则..."
    for rule_name in allow-http allow-https; do
        if ! gcloud compute firewall-rules describe "$rule_name" --quiet &>/dev/null; then
            local proto="80"; [[ "$rule_name" == "allow-https" ]] && proto="443"
            local tag="http-server"; [[ "$rule_name" == "allow-https" ]] && tag="https-server"
            gcloud compute firewall-rules create "$rule_name" \
                --allow="tcp:${proto}" --target-tags="$tag" --quiet
            log_success "$rule_name 已创建"
        else
            log_info "$rule_name 已存在，跳过"
        fi
    done

    # ---- 3. 创建 VM ----
    log_step "3/${TOTAL_STEPS} 创建 VM 实例..."
    if gcloud compute instances describe "$VM_NAME" --zone="$GCP_ZONE" --quiet &>/dev/null; then
        log_warn "VM '$VM_NAME' 已存在，跳过创建"
    else
        gcloud compute instances create "$VM_NAME" \
            --zone="$GCP_ZONE" \
            --machine-type="$GCP_MACHINE_TYPE" \
            --boot-disk-size="${GCP_DISK_SIZE}GB" \
            --boot-disk-type=pd-standard \
            --image-family=ubuntu-2404-lts-amd64 \
            --image-project=ubuntu-os-cloud \
            --tags=http-server,https-server \
            --network-tier=STANDARD \
            --quiet
        log_success "VM 已创建"
    fi

    VM_IP=$(gcloud compute instances describe "$VM_NAME" \
        --zone="$GCP_ZONE" \
        --format='value(networkInterfaces[0].accessConfigs[0].natIP)')
    log_info "VM 外部 IP: $VM_IP"

    # ---- 4. 等待 SSH 就绪 ----
    log_step "4/${TOTAL_STEPS} 等待 VM SSH 就绪..."
    local max_wait=60
    for i in $(seq 1 $max_wait); do
        if gcloud compute ssh "$VM_NAME" --zone="$GCP_ZONE" \
            --command="echo ok" --quiet -- -o ConnectTimeout=5 -o StrictHostKeyChecking=no &>/dev/null; then
            log_success "SSH 连接成功"
            break
        fi
        if [[ $i -eq $max_wait ]]; then
            log_error "SSH 连接超时，请手动完成部署:"
            echo "  gcloud compute ssh $VM_NAME --zone=$GCP_ZONE"
            exit 1
        fi
        echo -n "."
        sleep 3
    done
    echo ""

    # ---- 5. 上传脚本 ----
    log_step "5/${TOTAL_STEPS} 上传部署脚本到 VM..."
    # 获取当前脚本自身的路径
    local script_path
    script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
    gcloud compute scp "$script_path" "${VM_NAME}:/tmp/gcp-deploy.sh" --zone="$GCP_ZONE" --quiet
    gcloud compute ssh "$VM_NAME" --zone="$GCP_ZONE" --quiet \
        --command="chmod +x /tmp/gcp-deploy.sh"
    log_success "脚本已上传"

    # ---- 6. 远程执行部署 ----
    log_step "6/${TOTAL_STEPS} 在 VM 上远程执行部署..."
    echo ""

    # 构建 deploy 命令的参数
    local deploy_args="deploy"
    [[ -n "$DOMAIN" ]] && deploy_args="$deploy_args --domain $DOMAIN"
    [[ -n "$EMAIL" && "$EMAIL" != "admin@example.com" ]] && deploy_args="$deploy_args --email $EMAIL"
    [[ -n "$ADMIN_EMAIL" && "$ADMIN_EMAIL" != "admin@sub2api.local" ]] && deploy_args="$deploy_args --admin-email $ADMIN_EMAIL"
    [[ -n "$ADMIN_PASSWORD" ]] && deploy_args="$deploy_args --admin-password $ADMIN_PASSWORD"

    # 远程执行，使用 tee 同时显示输出并捕获结果
    gcloud compute ssh "$VM_NAME" --zone="$GCP_ZONE" \
        --command="sudo /tmp/gcp-deploy.sh $deploy_args" \
        -- -t

    # ---- 7. 验证 ----
    echo ""
    log_step "7/${TOTAL_STEPS} 验证部署..."
    sleep 3
    if curl -sf --connect-timeout 10 "http://${VM_IP}:8080/health" >/dev/null 2>&1; then
        log_success "健康检查通过"
    else
        log_warn "健康检查未通过 (服务可能仍在启动中，请稍后重试)"
    fi

    echo ""
    echo "============================================================================="
    echo "  一键部署完成!"
    echo "============================================================================="
    echo ""
    if [[ -n "$DOMAIN" ]]; then
        echo "  访问地址:   https://$DOMAIN"
    else
        echo "  访问地址:   http://${VM_IP}:8080"
    fi
    echo ""
    echo "  查看凭据:  gcloud compute ssh $VM_NAME --zone=$GCP_ZONE -- cat /opt/sub2api/deploy-info.txt"
    echo "  SSH 连接:  gcloud compute ssh $VM_NAME --zone=$GCP_ZONE"
    echo "  查看状态:  ./gcp-deploy.sh status"
    echo "  删除资源:  ./gcp-deploy.sh destroy"
    echo ""
    echo "  预估月费: ~\$$(get_estimated_cost)/月"
    echo "============================================================================="
}

get_estimated_cost() {
    case "$GCP_MACHINE_TYPE" in
        e2-micro)  echo "7-8" ;;
        e2-small)  echo "14-16" ;;
        e2-medium) echo "28-30" ;;
        *)         echo "??" ;;
    esac
}

# ===================== deploy 命令 =====================

cmd_deploy() {
    echo "============================================================================="
    echo "  Sub2API GCP 部署 - 第二步: 部署应用"
    echo "============================================================================="
    echo ""

    # 检查 root
    if [[ $EUID -ne 0 ]]; then
        log_error "请使用 sudo 运行: sudo $0 deploy"
        exit 1
    fi

    # 检查 Docker
    log_step "1/6 检查 Docker..."
    if ! command -v docker &>/dev/null; then
        log_info "正在安装 Docker..."
        curl -fsSL https://get.docker.com | sh
    fi
    docker --version
    docker compose version 2>/dev/null || {
        log_error "docker compose 未安装"
        exit 1
    }
    log_success "Docker 已就绪"

    # 生成安全密钥
    log_step "2/6 生成安全密钥..."
    JWT_SECRET=$(openssl rand -hex 32)
    TOTP_KEY=$(openssl rand -hex 32)
    PG_PASS=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)
    if [[ -z "$ADMIN_PASSWORD" ]]; then
        ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d '/+=' | head -c 16)
    fi
    log_success "密钥已生成"

    # 创建目录和配置
    log_step "3/6 创建配置文件..."
    mkdir -p "$INSTALL_DIR"/{data,postgres_data,redis_data,logs}
    cd "$INSTALL_DIR"

    # 生成 .env
    cat > .env << ENVEOF
# Sub2API GCP 部署配置
# 生成时间: $(date -Iseconds)

# 服务器
BIND_HOST=0.0.0.0
SERVER_PORT=${HTTP_PORT}
SERVER_MODE=release
RUN_MODE=standard
TZ=Asia/Shanghai

# H2C (HTTP/2 Cleartext)
SERVER_H2C_ENABLED=true
SERVER_H2C_MAX_CONCURRENT_STREAMS=50

# 数据库
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=${PG_PASS}
POSTGRES_DB=sub2api

# Redis
REDIS_PASSWORD=
REDIS_DB=0
REDIS_ENABLE_TLS=false

# 管理员
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}

# 安全密钥
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=${TOTP_KEY}

# 调度
GATEWAY_SCHEDULING_STICKY_SESSION_MAX_WAITING=3
GATEWAY_SCHEDULING_STICKY_SESSION_WAIT_TIMEOUT=120s
GATEWAY_SCHEDULING_LOAD_BATCH_ENABLED=true

# 仪表盘聚合
DASHBOARD_AGGREGATION_ENABLED=true
DASHBOARD_AGGREGATION_INTERVAL_SECONDS=60
DASHBOARD_AGGREGATION_RETENTION_USAGE_LOGS_DAYS=90
DASHBOARD_AGGREGATION_RETENTION_HOURLY_DAYS=180
DASHBOARD_AGGREGATION_RETENTION_DAILY_DAYS=730

# 安全 (生产环境推荐关闭 insecure http 和 private hosts)
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=false
SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=false

# 监控
OPS_ENABLED=true

# 更新 (GCP 海外服务器直连 GitHub，无需代理)
UPDATE_PROXY_URL=
ENVEOF
    chmod 600 .env

    # 生成 docker-compose.yml (使用本地目录方式)
    cat > docker-compose.yml << 'COMPOSEEOF'
# Sub2API GCP Docker Compose Configuration

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
      - ./data:/app/data
    environment:
      - AUTO_SETUP=true
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
      - SERVER_MODE=${SERVER_MODE:-release}
      - SERVER_H2C_ENABLED=${SERVER_H2C_ENABLED:-true}
      - SERVER_H2C_MAX_CONCURRENT_STREAMS=${SERVER_H2C_MAX_CONCURRENT_STREAMS:-50}
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
      - OPS_ENABLED=${OPS_ENABLED:-true}
      - DASHBOARD_AGGREGATION_ENABLED=${DASHBOARD_AGGREGATION_ENABLED:-true}
      - DASHBOARD_AGGREGATION_INTERVAL_SECONDS=${DASHBOARD_AGGREGATION_INTERVAL_SECONDS:-60}
      - GATEWAY_SCHEDULING_STICKY_SESSION_MAX_WAITING=${GATEWAY_SCHEDULING_STICKY_SESSION_MAX_WAITING:-3}
      - GATEWAY_SCHEDULING_STICKY_SESSION_WAIT_TIMEOUT=${GATEWAY_SCHEDULING_STICKY_SESSION_WAIT_TIMEOUT:-120s}
      - GATEWAY_SCHEDULING_LOAD_BATCH_ENABLED=${GATEWAY_SCHEDULING_LOAD_BATCH_ENABLED:-true}
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
    image: postgres:18-alpine
    container_name: sub2api-postgres
    restart: unless-stopped
    ulimits:
      nofile:
        soft: 100000
        hard: 100000
    volumes:
      - ./postgres_data:/var/lib/postgresql/data
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
    image: redis:8-alpine
    container_name: sub2api-redis
    restart: unless-stopped
    ulimits:
      nofile:
        soft: 100000
        hard: 100000
    volumes:
      - ./redis_data:/data
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

networks:
  sub2api-network:
    driver: bridge
COMPOSEEOF

    log_success "配置文件已创建"

    # 启动服务
    log_step "4/6 拉取镜像并启动服务..."
    docker compose pull
    docker compose up -d

    # 等待就绪
    log_step "5/6 等待服务就绪..."
    for i in $(seq 1 30); do
        if curl -sf http://localhost:${HTTP_PORT}/health >/dev/null 2>&1; then
            log_success "Sub2API 已启动"
            break
        fi
        if [[ $i -eq 30 ]]; then
            log_error "启动超时，请检查: docker compose logs"
            exit 1
        fi
        echo -n "."
        sleep 2
    done
    echo ""

    # 配置 Caddy (如果有域名)
    log_step "6/6 配置反向代理..."
    if [[ -n "$DOMAIN" ]]; then
        install_caddy_proxy
    else
        log_info "未指定域名，跳过 HTTPS 配置"
        log_info "可通过 IP:${HTTP_PORT} 直接访问，或后续手动配置域名"
    fi

    # 设置备份和监控
    setup_backup_cron
    setup_health_monitor

    # 配置 Docker 和服务自启动
    systemctl enable docker 2>/dev/null || true

    # 显示部署结果
    VM_IP=$(hostname -I | awk '{print $1}')
    echo ""
    echo "============================================================================="
    echo "  Sub2API 部署成功!"
    echo "============================================================================="
    echo ""
    if [[ -n "$DOMAIN" ]]; then
        echo "  访问地址:     https://$DOMAIN"
    else
        echo "  访问地址:     http://${VM_IP}:${HTTP_PORT}"
    fi
    echo "  管理员邮箱:   $ADMIN_EMAIL"
    echo "  管理员密码:   $ADMIN_PASSWORD"
    echo ""
    echo "  安全密钥 (请保存到安全位置):"
    echo "    JWT_SECRET:            $JWT_SECRET"
    echo "    TOTP_ENCRYPTION_KEY:   $TOTP_KEY"
    echo "    POSTGRES_PASSWORD:     $PG_PASS"
    echo ""
    echo "  常用命令 (在 $INSTALL_DIR 目录下):"
    echo "    docker compose logs -f sub2api  # 查看日志"
    echo "    docker compose restart          # 重启服务"
    echo "    docker compose pull && docker compose up -d  # 更新版本"
    echo ""
    echo "  配置文件: $INSTALL_DIR/.env"
    echo "  数据目录: $INSTALL_DIR/data/"
    echo "============================================================================="

    # 保存部署信息
    cat > "$INSTALL_DIR/deploy-info.txt" << INFOEOF
Sub2API GCP 部署信息
部署时间: $(date -Iseconds)
访问地址: ${DOMAIN:+https://$DOMAIN}${DOMAIN:-http://${VM_IP}:${HTTP_PORT}}
管理员邮箱: $ADMIN_EMAIL
管理员密码: $ADMIN_PASSWORD
JWT_SECRET: $JWT_SECRET
TOTP_ENCRYPTION_KEY: $TOTP_KEY
POSTGRES_PASSWORD: $PG_PASS
INFOEOF
    chmod 600 "$INSTALL_DIR/deploy-info.txt"
}

install_caddy_proxy() {
    log_info "安装 Caddy 反向代理..."

    # 检查系统
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
    fi

    # 安装 Caddy
    apt-get update -qq
    apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https curl
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg 2>/dev/null
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
    apt-get update -qq
    apt-get install -y -qq caddy

    # 生成 Caddyfile
    mkdir -p /var/log/caddy
    chown caddy:caddy /var/log/caddy

    cat > /etc/caddy/Caddyfile << CADDYEOF
{
    email ${EMAIL}
    servers {
        protocols h1 h2 h3
    }
}

${DOMAIN} {
    reverse_proxy localhost:${HTTP_PORT} {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
        flush_interval -1
    }

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        -Server
    }

    encode {
        zstd
        gzip
        match {
            not header Accept text/event-stream*
            not path /v1/messages /v1/responses /responses /antigravity/v1/messages /v1beta/models/*
        }
    }

    log {
        output file /var/log/caddy/sub2api.log {
            roll_size 50mb
            roll_keep 5
        }
    }
}
CADDYEOF

    systemctl restart caddy
    systemctl enable caddy
    log_success "Caddy 已配置: https://${DOMAIN}"
}

setup_backup_cron() {
    cat > "$INSTALL_DIR/backup.sh" << 'BACKUPEOF'
#!/bin/bash
set -euo pipefail
INSTALL_DIR="/opt/sub2api"
BACKUP_DIR="$INSTALL_DIR/backup"
RETENTION_DAYS=30

source "$INSTALL_DIR/.env"
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p "$BACKUP_DIR"

cd "$INSTALL_DIR"
docker compose exec -T postgres pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" | gzip > "$BACKUP_DIR/db_${BACKUP_DATE}.sql.gz"

# 清理旧备份
find "$BACKUP_DIR" -name "db_*.sql.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
echo "$(date): Backup completed: db_${BACKUP_DATE}.sql.gz" >> "$INSTALL_DIR/logs/backup.log"
BACKUPEOF
    chmod +x "$INSTALL_DIR/backup.sh"

    # 每天凌晨 3 点备份
    (crontab -l 2>/dev/null | grep -v "$INSTALL_DIR/backup.sh" || true; echo "0 3 * * * $INSTALL_DIR/backup.sh >/dev/null 2>&1") | crontab -
    log_success "自动备份已配置 (每天 03:00)"
}

setup_health_monitor() {
    cat > "$INSTALL_DIR/health-check.sh" << 'HEALTHEOF'
#!/bin/bash
INSTALL_DIR="/opt/sub2api"
if ! curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    echo "$(date): Health check failed, restarting..." >> "$INSTALL_DIR/logs/health.log"
    cd "$INSTALL_DIR" && docker compose restart sub2api
fi
HEALTHEOF
    chmod +x "$INSTALL_DIR/health-check.sh"

    (crontab -l 2>/dev/null | grep -v "$INSTALL_DIR/health-check.sh" || true; echo "*/5 * * * * $INSTALL_DIR/health-check.sh >/dev/null 2>&1") | crontab -
    log_success "健康检查已配置 (每 5 分钟)"
}

# ===================== ssh 命令 =====================

cmd_ssh() {
    check_gcloud
    ensure_project
    gcloud compute ssh "$VM_NAME" --zone="$GCP_ZONE"
}

# ===================== status 命令 =====================

cmd_status() {
    check_gcloud
    ensure_project

    echo ""
    log_info "VM 实例状态:"
    gcloud compute instances describe "$VM_NAME" \
        --zone="$GCP_ZONE" \
        --format="table(name,status,networkInterfaces[0].accessConfigs[0].natIP,machineType.basename(),zone.basename())" 2>/dev/null || {
        log_error "VM '$VM_NAME' 不存在于 $GCP_ZONE"
        exit 1
    }

    echo ""
    log_info "尝试健康检查..."
    VM_IP=$(gcloud compute instances describe "$VM_NAME" \
        --zone="$GCP_ZONE" \
        --format='value(networkInterfaces[0].accessConfigs[0].natIP)' 2>/dev/null)

    if curl -sf --connect-timeout 5 "http://${VM_IP}:8080/health" 2>/dev/null; then
        echo ""
        log_success "服务运行正常"
    else
        log_warn "服务不可达 (可能正在启动或防火墙未开放)"
    fi
}

# ===================== destroy 命令 =====================

cmd_destroy() {
    check_gcloud
    ensure_project

    echo ""
    log_warn "即将删除以下 GCP 资源:"
    echo "  - VM 实例: $VM_NAME ($GCP_ZONE)"
    echo "  - 防火墙规则: allow-http, allow-https"
    echo ""
    log_warn "此操作不可恢复! 所有数据将被永久删除!"
    echo ""
    read -p "输入 'DELETE' 确认删除: " -r
    if [[ "$REPLY" != "DELETE" ]]; then
        log_info "已取消"
        exit 0
    fi

    echo ""

    # 删除 VM
    if gcloud compute instances describe "$VM_NAME" --zone="$GCP_ZONE" --quiet &>/dev/null; then
        log_info "删除 VM 实例..."
        gcloud compute instances delete "$VM_NAME" --zone="$GCP_ZONE" --quiet
        log_success "VM 已删除"
    else
        log_info "VM 不存在，跳过"
    fi

    # 删除防火墙规则
    for rule in allow-http allow-https; do
        if gcloud compute firewall-rules describe "$rule" --quiet &>/dev/null; then
            gcloud compute firewall-rules delete "$rule" --quiet
            log_success "防火墙规则 $rule 已删除"
        fi
    done

    echo ""
    log_success "所有资源已清理"
}

# ===================== help 命令 =====================

cmd_help() {
    cat << 'HELPEOF'
Sub2API Google Cloud Platform 部署脚本

用法: ./gcp-deploy.sh <命令> [选项]

命令:
  auto      一键全自动部署 (创建 VM + SSH + 部署, 推荐!)
  setup     仅创建 GCP VM 和防火墙规则 (本地运行)
  deploy    仅在 VM 内部署服务 (VM 内运行, 需要 sudo)
  ssh       SSH 连接到 VM
  status    查看 VM 和服务状态
  destroy   删除所有 GCP 资源 (危险!)
  help      显示此帮助

auto 选项 (同时支持 setup + deploy 的所有选项):
  --region REGION          GCP 区域 (默认: asia-east1)
  --zone ZONE              GCP 可用区 (默认: asia-east1-a)
  --machine-type TYPE      机器类型 (默认: e2-small)
  --disk-size SIZE         磁盘大小 GB (默认: 20)
  --vm-name NAME           VM 名称 (默认: sub2api)
  --domain DOMAIN          自定义域名 (自动申请 Let's Encrypt SSL)
  --email EMAIL            SSL 证书通知邮箱
  --admin-email EMAIL      管理员账户邮箱
  --admin-password PASS    管理员密码 (默认随机生成)

区域选择建议 (面向中国用户):
  asia-east1   (台湾)   ~$14/月  到中国 40-80ms   性价比最高
  asia-east2   (香港)   ~$17/月  到中国 20-50ms   延迟最低
  asia-northeast1 (东京) ~$16/月  到中国 60-120ms  中等

示例:
  # 一键部署 (最简单)
  ./gcp-deploy.sh auto

  # 一键部署 + 指定域名
  ./gcp-deploy.sh auto --domain api.example.com --email admin@example.com

  # 一键部署到香港 + 更小机器
  ./gcp-deploy.sh auto --region asia-east2 --zone asia-east2-a --machine-type e2-micro

  # 分步执行
  ./gcp-deploy.sh setup
  gcloud compute ssh sub2api --zone=asia-east1-a
  sudo ./gcp-deploy.sh deploy

HELPEOF
}

# ===================== 参数解析和入口 =====================

parse_setup_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --region)       GCP_REGION="$2"; shift 2 ;;
            --zone)         GCP_ZONE="$2"; shift 2 ;;
            --machine-type) GCP_MACHINE_TYPE="$2"; shift 2 ;;
            --disk-size)    GCP_DISK_SIZE="$2"; shift 2 ;;
            --vm-name)      VM_NAME="$2"; shift 2 ;;
            *) log_error "未知参数: $1"; exit 1 ;;
        esac
    done
}

parse_deploy_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --domain)         DOMAIN="$2"; shift 2 ;;
            --email)          EMAIL="$2"; shift 2 ;;
            --admin-email)    ADMIN_EMAIL="$2"; shift 2 ;;
            --admin-password) ADMIN_PASSWORD="$2"; shift 2 ;;
            *) log_error "未知参数: $1"; exit 1 ;;
        esac
    done
}

parse_auto_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --region)         GCP_REGION="$2"; shift 2 ;;
            --zone)           GCP_ZONE="$2"; shift 2 ;;
            --machine-type)   GCP_MACHINE_TYPE="$2"; shift 2 ;;
            --disk-size)      GCP_DISK_SIZE="$2"; shift 2 ;;
            --vm-name)        VM_NAME="$2"; shift 2 ;;
            --domain)         DOMAIN="$2"; shift 2 ;;
            --email)          EMAIL="$2"; shift 2 ;;
            --admin-email)    ADMIN_EMAIL="$2"; shift 2 ;;
            --admin-password) ADMIN_PASSWORD="$2"; shift 2 ;;
            *) log_error "未知参数: $1"; exit 1 ;;
        esac
    done
}

main() {
    if [[ $# -lt 1 ]]; then
        cmd_help
        exit 0
    fi

    local command="$1"
    shift

    case "$command" in
        auto)
            parse_auto_args "$@"
            cmd_auto
            ;;
        setup)
            parse_setup_args "$@"
            cmd_setup
            ;;
        deploy)
            parse_deploy_args "$@"
            cmd_deploy
            ;;
        ssh)
            cmd_ssh
            ;;
        status)
            cmd_status
            ;;
        destroy)
            cmd_destroy
            ;;
        help|--help|-h)
            cmd_help
            ;;
        *)
            log_error "未知命令: $command"
            echo "运行 '$0 help' 查看帮助"
            exit 1
            ;;
    esac
}

main "$@"
