#!/bin/bash
# =============================================================================
# Sub2API 自动化部署脚本
# 功能: 一键部署、配置初始化、安全加固、监控设置
# 适用: Ubuntu 20.04+/Debian 11+/CentOS 8+
# 用法: curl -sSL https://your-domain.com/deploy.sh | sudo bash
# =============================================================================

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 配置变量（可通过环境变量覆盖）
INSTALL_DIR="${INSTALL_DIR:-/opt/sub2api}"
DOMAIN="${DOMAIN:-}""
EMAIL="${EMAIL:-admin@example.com}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@sub2api.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
HTTP_PORT="${HTTP_PORT:-8080}"
BACKUP_DIR="${BACKUP_DIR:-/backup/sub2api}"

# 生成安全密钥
generate_secrets() {
    JWT_SECRET=$(openssl rand -hex 32)
    TOTP_KEY=$(openssl rand -hex 32)
    PG_PASS=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)

    # 如果未设置管理员密码，生成随机密码
    if [[ -z "$ADMIN_PASSWORD" ]]; then
        ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d '/+=' | head -c 16)
        log_warn "生成的管理员密码: $ADMIN_PASSWORD"
        log_warn "请保存此密码，只显示一次！"
    fi
}

# 检查系统要求
check_prerequisites() {
    log_info "检查系统要求..."

    # 检查 root 权限
    if [[ $EUID -ne 0 ]]; then
        log_error "请使用 sudo 运行此脚本"
        exit 1
    fi

    # 检查系统架构
    ARCH=$(uname -m)
    if [[ "$ARCH" != "x86_64" && "$ARCH" != "aarch64" ]]; then
        log_error "不支持的架构: $ARCH (仅支持 x86_64 和 aarch64)"
        exit 1
    fi

    # 检查内存
    MEM_GB=$(free -g | awk '/^Mem:/{print $2}')
    if [[ $MEM_GB -lt 2 ]]; then
        log_warn "内存不足 2GB (当前: ${MEM_GB}GB)，建议至少 4GB"
    fi

    # 检查磁盘空间
    DISK_GB=$(df -BG "$INSTALL_DIR" 2>/dev/null | awk 'NR==2 {print $4}' | sed 's/G//')
    if [[ -z "$DISK_GB" || "$DISK_GB" -lt 10 ]]; then
        log_warn "磁盘空间不足 10GB，建议至少 20GB"
    fi

    log_success "系统检查通过"
}

# 安装 Docker
install_docker() {
    log_info "检查 Docker 安装..."

    if command -v docker &> /dev/null; then
        log_success "Docker 已安装: $(docker --version)"
        return 0
    fi

    log_info "正在安装 Docker..."

    # 检测发行版
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
    else
        log_error "无法检测操作系统"
        exit 1
    fi

    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release
            curl -fsSL https://download.docker.com/linux/$OS/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/$OS $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
            apt-get update
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
            ;;
        centos|rhel|fedora|rocky|almalinux)
            yum install -y yum-utils
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
            systemctl start docker
            systemctl enable docker
            ;;
        *)
            log_error "不支持的操作系统: $OS"
            exit 1
            ;;
    esac

    # 验证安装
    docker --version
    docker compose version

    log_success "Docker 安装完成"
}

# 创建目录结构
setup_directories() {
    log_info "创建目录结构..."

    mkdir -p "$INSTALL_DIR"/{data,postgres_data,redis_data,backup,logs}

    # 创建非特权用户
    if ! id -u sub2api &>/dev/null; then
        useradd -r -s /bin/false -d "$INSTALL_DIR" sub2api
    fi

    chown -R sub2api:sub2api "$INSTALL_DIR"
    chmod 750 "$INSTALL_DIR"

    log_success "目录结构创建完成"
}

# 生成配置文件
generate_config() {
    log_info "生成配置文件..."

    cd "$INSTALL_DIR"

    # 生成环境变量文件
    cat > .env << EOF
# =============================================================================
# Sub2API 自动生成的配置文件
# 生成时间: $(date -Iseconds)
# =============================================================================

# 服务器配置
BIND_HOST=0.0.0.0
SERVER_PORT=${HTTP_PORT}
SERVER_MODE=release
TZ=Asia/Shanghai

# 性能调优
SERVER_MAX_REQUEST_BODY_SIZE=104857600
SERVER_H2C_ENABLED=true
SERVER_H2C_MAX_CONCURRENT_STREAMS=50

# 数据库配置
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=${PG_PASS}
POSTGRES_DB=sub2api

# Redis 配置
REDIS_PASSWORD=
REDIS_DB=0

# 管理员账户
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}

# 安全密钥（请勿泄露）
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=${TOTP_KEY}

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
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=false
SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=false

# 监控配置
OPS_ENABLED=true

# 更新配置
UPDATE_PROXY_URL=
EOF

    chmod 600 .env

    # 下载 docker-compose.yml
    curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/docker-compose.local.yml -o docker-compose.yml

    log_success "配置文件生成完成"
}

# 配置防火墙
setup_firewall() {
    log_info "配置防火墙..."

    # 检测防火墙类型
    if command -v ufw &> /dev/null; then
        ufw default deny incoming
        ufw default allow outgoing
        ufw allow 22/tcp comment 'SSH'
        ufw allow 80/tcp comment 'HTTP'
        ufw allow 443/tcp comment 'HTTPS'

        # 如果配置了非标准端口，也开放
        if [[ "$HTTP_PORT" != "8080" ]]; then
            ufw allow "$HTTP_PORT/tcp" comment 'Sub2API'
        fi

        ufw --force enable
        log_success "UFW 防火墙配置完成"

    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-service=ssh
        firewall-cmd --permanent --add-service=http
        firewall-cmd --permanent --add-service=https

        if [[ "$HTTP_PORT" != "8080" ]]; then
            firewall-cmd --permanent --add-port="$HTTP_PORT/tcp"
        fi

        firewall-cmd --reload
        log_success "Firewalld 配置完成"
    else
        log_warn "未检测到支持的防火墙，请手动配置"
    fi
}

# 启动服务
start_services() {
    log_info "启动 Sub2API 服务..."

    cd "$INSTALL_DIR"

    # 拉取镜像
    docker-compose pull

    # 启动服务
    docker-compose up -d

    # 等待服务就绪
    log_info "等待服务就绪..."
    sleep 10

    # 健康检查
    for i in {1..30}; do
        if curl -sf http://localhost:${HTTP_PORT}/health > /dev/null 2>&1; then
            log_success "Sub2API 服务已就绪"
            break
        fi

        if [[ $i -eq 30 ]]; then
            log_error "服务启动超时，请检查日志: docker-compose logs"
            exit 1
        fi

        echo -n "."
        sleep 2
    done
}

# 安装 Caddy 反向代理
install_caddy() {
    if [[ -z "$DOMAIN" ]]; then
        log_warn "未设置域名，跳过 Caddy 安装"
        log_warn "如需后续配置域名，可重新运行脚本或手动安装 Caddy"
        return 0
    fi

    log_info "安装 Caddy 反向代理..."

    # 安装 Caddy
    apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
    apt-get update
    apt-get install -y caddy

    # 生成 Caddyfile
    cat > /etc/caddy/Caddyfile << EOF
{
    email ${EMAIL}
    servers {
        protocols h1 h2 h3
        timeouts {
            read_body 30s
            read_header 10s
            write 300s
            idle 300s
        }
    }
}

${DOMAIN} {
    @static {
        path /assets/*
        path /logo.png
        path /favicon.ico
    }
    header @static {
        Cache-Control "public, max-age=31536000, immutable"
        -Pragma
        -Expires
    }

    tls {
        protocols tls1.2 tls1.3
        ciphers TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256 TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256 TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
    }

    reverse_proxy localhost:${HTTP_PORT} {
        health_uri /health
        health_interval 30s
        health_timeout 10s
        health_status 200

        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
        header_up X-Forwarded-Host {host}
        header_up CF-Connecting-IP {http.request.header.CF-Connecting-IP}

        transport http {
            keepalive 120s
            keepalive_idle_conns 256
        }

        flush_interval -1
    }

    encode {
        zstd
        gzip 6
        minimum_length 256
        match {
            not header Accept text/event-stream*
            not path /v1/messages /v1/responses /responses /antigravity/v1/messages /v1beta/models/* /antigravity/v1beta/models/*
            header Content-Type text/*
            header Content-Type application/json*
            header Content-Type application/javascript*
        }
    }

    header {
        X-Frame-Options "SAMEORIGIN"
        X-XSS-Protection "1; mode=block"
        X-Content-Type-Options "nosniff"
        Referrer-Policy "strict-origin-when-cross-origin"
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        Permissions-Policy "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"
        Cross-Origin-Opener-Policy "same-origin"
        Cross-Origin-Embedder-Policy "require-corp"
        Cross-Origin-Resource-Policy "same-origin"
        -Server
        -X-Powered-By
    }

    request_body {
        max_size 100MB
    }

    log {
        output file /var/log/caddy/sub2api.log {
            roll_size 50mb
            roll_keep 10
            roll_keep_for 720h
        }
        format json
        level INFO
    }
}
EOF

    # 创建日志目录
    mkdir -p /var/log/caddy
    chown caddy:caddy /var/log/caddy

    # 重载 Caddy
    systemctl reload caddy
    systemctl enable caddy

    log_success "Caddy 安装并配置完成"
    log_info "网站地址: https://${DOMAIN}"
}

# 设置自动备份
setup_backup() {
    log_info "配置自动备份..."

    # 创建备份脚本
    cat > "$INSTALL_DIR/backup.sh" << 'EOF'
#!/bin/bash
set -euo pipefail

INSTALL_DIR="/opt/sub2api"
BACKUP_DIR="/backup/sub2api"
RETENTION_DAYS=30

# 加载环境变量
source "$INSTALL_DIR/.env"

# 创建备份目录
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="$BACKUP_DIR/$BACKUP_DATE"
mkdir -p "$BACKUP_PATH"

# 备份数据目录
cd "$INSTALL_DIR"
tar czf "$BACKUP_PATH/data.tar.gz" data/ postgres_data/ redis_data/ .env

# 数据库逻辑备份
docker-compose exec -T postgres pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" | gzip > "$BACKUP_PATH/db.sql.gz"

# 创建备份信息
echo "Backup completed at $(date -Iseconds)" > "$BACKUP_PATH/INFO.txt"
echo "Sub2API Version: $(docker-compose exec -T sub2api /app/sub2api --version 2>/dev/null || echo 'unknown')" >> "$BACKUP_PATH/INFO.txt"

# 压缩整个备份
cd "$BACKUP_DIR"
tar czf "${BACKUP_DATE}.tar.gz" "$BACKUP_DATE"
rm -rf "$BACKUP_PATH"

# 清理旧备份
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete

# 可选：上传到远程存储（如果配置了 rclone）
if command -v rclone &> /dev/null && rclone listremotes | grep -q "backup"; then
    rclone copy "${BACKUP_DATE}.tar.gz" backup:sub2api/
fi

echo "Backup completed: ${BACKUP_DATE}.tar.gz"
EOF

    chmod +x "$INSTALL_DIR/backup.sh"

    # 创建定时任务
    (crontab -l 2>/dev/null || true; echo "0 2 * * * $INSTALL_DIR/backup.sh >> $INSTALL_DIR/logs/backup.log 2>&1") | crontab -

    log_success "自动备份配置完成（每天凌晨2点执行）"
}

# 设置监控（可选）
setup_monitoring() {
    log_info "配置基础监控..."

    # 创建健康检查脚本
    cat > "$INSTALL_DIR/health-check.sh" << 'EOF'
#!/bin/bash
# 健康检查脚本，可用于 Prometheus node_exporter 或自定义监控

INSTALL_DIR="/opt/sub2api"
LOG_FILE="$INSTALL_DIR/logs/health-check.log"

# 检查应用健康
if ! curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo "$(date): ALERT - Sub2API health check failed" >> "$LOG_FILE"

    # 尝试自动重启
    cd "$INSTALL_DIR"
    docker-compose restart sub2api

    # 发送通知（如果配置了 webhook）
    if [[ -n "${ALERT_WEBHOOK:-}" ]]; then
        curl -s -X POST -H 'Content-type: application/json' \
            --data '{"text":"Sub2API 健康检查失败，已自动重启"}' \
            "$ALERT_WEBHOOK" > /dev/null 2>&1 || true
    fi

    exit 1
fi

# 检查磁盘空间
DISK_USAGE=$(df "$INSTALL_DIR" | awk 'NR==2 {print $5}' | sed 's/%//')
if [[ "$DISK_USAGE" -gt 90 ]]; then
    echo "$(date): WARNING - Disk usage is ${DISK_USAGE}%" >> "$LOG_FILE"
fi

# 检查内存使用
MEM_USAGE=$(free | grep Mem | awk '{printf "%.0f", $3/$2 * 100.0}')
if [[ "$MEM_USAGE" -gt 90 ]]; then
    echo "$(date): WARNING - Memory usage is ${MEM_USAGE}%" >> "$LOG_FILE"
fi

exit 0
EOF

    chmod +x "$INSTALL_DIR/health-check.sh"

    # 每5分钟执行一次健康检查
    (crontab -l 2>/dev/null || true; echo "*/5 * * * * $INSTALL_DIR/health-check.sh > /dev/null 2>&1") | crontab -

    log_success "基础监控配置完成"
}

# 生成部署报告
generate_report() {
    cd "$INSTALL_DIR"

    # 获取容器状态
    CONTAINER_STATUS=$(docker-compose ps --format "table {{.Service}}\t{{.Status}}\t{{.Ports}}")

    cat > "$INSTALL_DIR/install-report.txt" << EOF
=============================================================================
Sub2API 自动化部署报告
=============================================================================
部署时间: $(date -Iseconds)
服务器IP: $(hostname -I | awk '{print $1}')
安装目录: $INSTALL_DIR

【访问信息】
$(if [[ -n "$DOMAIN" ]]; then
    echo "域名地址: https://$DOMAIN"
else
    echo "IP 地址: http://$(hostname -I | awk '{print $1}'):$HTTP_PORT"
fi)
管理员邮箱: $ADMIN_EMAIL
管理员密码: $ADMIN_PASSWORD

【安全密钥】（请妥善保管）
JWT_SECRET: $JWT_SECRET
TOTP_ENCRYPTION_KEY: $TOTP_KEY
PostgreSQL密码: $PG_PASS

【服务状态】
$CONTAINER_STATUS

【常用命令】
查看日志:    cd $INSTALL_DIR && docker-compose logs -f
重启服务:    cd $INSTALL_DIR && docker-compose restart
备份数据:    $INSTALL_DIR/backup.sh
更新版本:    cd $INSTALL_DIR && docker-compose pull && docker-compose up -d

【文件位置】
配置文件:    $INSTALL_DIR/.env
数据目录:    $INSTALL_DIR/data/
备份目录:    $BACKUP_DIR/
日志目录:    $INSTALL_DIR/logs/

【监控告警】
健康检查:    $INSTALL_DIR/health-check.sh（每5分钟）
自动备份:    每天凌晨2点执行

=============================================================================
重要提示：
1. 请立即修改默认管理员密码
2. 请保存安全密钥到密码管理器
3. 建议配置异地备份（rclone 支持 S3/阿里云OSS等）
4. 生产环境建议启用防火墙和Fail2ban
=============================================================================
EOF

    # 设置权限
    chmod 600 "$INSTALL_DIR/install-report.txt"

    log_success "部署报告已保存到: $INSTALL_DIR/install-report.txt"

    # 显示关键信息
    echo ""
    echo "============================================================================="
    echo "                          Sub2API 部署成功！"
    echo "============================================================================="
    echo ""
    if [[ -n "$DOMAIN" ]]; then
        echo "  🌐 访问地址: https://$DOMAIN"
    else
        echo "  🌐 访问地址: http://$(hostname -I | awk '{print $1}'):$HTTP_PORT"
    fi
    echo "  👤 管理员邮箱: $ADMIN_EMAIL"
    echo "  🔑 管理员密码: $ADMIN_PASSWORD"
    echo ""
    echo "  ⚠️  请立即保存以下信息到安全位置："
    echo "     - 管理员密码"
    echo "     - JWT_SECRET"
    echo "     - TOTP_ENCRYPTION_KEY"
    echo ""
    echo "  📋 完整报告: $INSTALL_DIR/install-report.txt"
    echo ""
    echo "============================================================================="
}

# 清理函数
cleanup() {
    if [[ $? -ne 0 ]]; then
        log_error "部署过程中出现错误，开始清理..."
        cd "$INSTALL_DIR" 2>/dev/null && docker-compose down -v 2>/dev/null || true
    fi
}
trap cleanup EXIT

# 主函数
main() {
    echo "============================================================================="
    echo "                    Sub2API 自动化部署脚本"
    echo "============================================================================="
    echo ""

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --email)
                EMAIL="$2"
                shift 2
                ;;
            --admin-email)
                ADMIN_EMAIL="$2"
                shift 2
                ;;
            --admin-password)
                ADMIN_PASSWORD="$2"
                shift 2
                ;;
            --port)
                HTTP_PORT="$2"
                shift 2
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo ""
                echo "选项:"
                echo "  --domain <域名>           配置自定义域名（自动申请SSL）"
                echo "  --email <邮箱>            用于SSL证书通知"
                echo "  --admin-email <邮箱>      管理员账户邮箱"
                echo "  --admin-password <密码>   管理员账户密码（默认随机生成）"
                echo "  --port <端口>             内部HTTP端口（默认8080）"
                echo "  --help                    显示此帮助信息"
                echo ""
                echo "示例:"
                echo "  $0 --domain api.example.com --email admin@example.com"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done

    # 执行部署步骤
    check_prerequisites
    generate_secrets
    install_docker
    setup_directories
    generate_config
    setup_firewall
    start_services
    install_caddy
    setup_backup
    setup_monitoring
    generate_report

    log_success "🎉 Sub2API 部署完成！"
}

# 执行主函数
main "$@"
