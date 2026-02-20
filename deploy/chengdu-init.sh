#!/bin/bash
# =============================================================================
# Sub2API 成都服务器初始化脚本
# 服务器: 47.108.158.227
# 用途: 一键初始化服务器环境，安装依赖，配置 CI/CD
# =============================================================================

set -euo pipefail

SERVER_IP="47.108.158.227"
INSTALL_DIR="/opt/sub2api"
REPO_URL="git@gitee.com:xixi_24/sub2api.git"
WEBHOOK_PORT=9000
WEBHOOK_SECRET="${WEBHOOK_SECRET:-$(openssl rand -hex 32)}"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查是否为 root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 用户运行此脚本"
        exit 1
    fi
}

# 更新系统
update_system() {
    log "[1/10] 更新系统..."
    apt-get update -qq
    apt-get upgrade -y -qq
    log_success "✅ 系统更新完成"
}

# 安装基础工具
install_basic_tools() {
    log "[2/10] 安装基础工具..."
    apt-get install -y -qq \
        curl \
        wget \
        git \
        vim \
        htop \
        netcat \
        jq \
        unzip \
        build-essential
    log_success "✅ 基础工具安装完成"
}

# 安装 Docker
install_docker() {
    log "[3/10] 安装 Docker..."

    if command -v docker &> /dev/null; then
        log_warn "Docker 已安装，跳过"
        return
    fi

    curl -fsSL https://get.docker.com | sh
    systemctl start docker
    systemctl enable docker

    # 安装 Docker Compose
    apt-get install -y docker-compose-plugin

    log_success "✅ Docker 安装完成"
}

# 安装 Go
install_go() {
    log "[4/10] 安装 Go 1.25.7..."

    if command -v go &> /dev/null; then
        local current_version=$(go version | awk '{print $3}' | sed 's/go//')
        if [ "$current_version" == "1.25.7" ]; then
            log_warn "Go 1.25.7 已安装，跳过"
            return
        fi
    fi

    wget -q https://go.dev/dl/go1.25.7.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.25.7.linux-amd64.tar.gz
    rm go1.25.7.linux-amd64.tar.gz

    # 配置环境变量
    if ! grep -q "/usr/local/go/bin" /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
        echo 'export GOPATH=$HOME/go' >> /etc/profile
    fi

    export PATH=$PATH:/usr/local/go/bin

    log_success "✅ Go 安装完成: $(go version)"
}

# 安装 Node.js 和 pnpm
install_nodejs() {
    log "[5/10] 安装 Node.js 20 和 pnpm..."

    if command -v node &> /dev/null; then
        log_warn "Node.js 已安装，跳过"
    else
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
        apt-get install -y nodejs
    fi

    if ! command -v pnpm &> /dev/null; then
        npm install -g pnpm
    fi

    log_success "✅ Node.js 安装完成: $(node -v)"
    log_success "✅ pnpm 安装完成: $(pnpm -v)"
}

# 配置 Git
setup_git() {
    log "[6/10] 配置 Git..."

    # 配置 Gitee SSH
    if [ ! -f ~/.ssh/id_rsa ]; then
        log "生成 SSH 密钥..."
        ssh-keygen -t rsa -b 4096 -C "sub2api@$SERVER_IP" -f ~/.ssh/id_rsa -N ""

        log_warn "=========================================="
        log_warn "请将以下 SSH 公钥添加到 Gitee："
        log_warn "=========================================="
        cat ~/.ssh/id_rsa.pub
        log_warn "=========================================="
        log_warn "添加地址: https://gitee.com/profile/sshkeys"
        log_warn "按 Enter 继续..."
        read
    fi

    # 添加 Gitee 到 known_hosts
    ssh-keyscan -H gitee.com >> ~/.ssh/known_hosts 2>/dev/null

    log_success "✅ Git 配置完成"
}

# 克隆代码仓库
clone_repo() {
    log "[7/10] 克隆代码仓库..."

    mkdir -p /opt

    if [ -d "$INSTALL_DIR" ]; then
        log_warn "目录已存在，跳过克隆"
        return
    fi

    cd /opt
    git clone "$REPO_URL" sub2api

    log_success "✅ 代码仓库克隆完成"
}

# 配置防火墙
setup_firewall() {
    log "[8/10] 配置防火墙..."

    ufw default deny incoming
    ufw default allow outgoing
    ufw allow 22/tcp comment 'SSH'
    ufw allow 80/tcp comment 'HTTP'
    ufw allow 443/tcp comment 'HTTPS'
    ufw allow 8080/tcp comment 'Sub2API'
    ufw allow $WEBHOOK_PORT/tcp comment 'Webhook'
    ufw --force enable

    log_success "✅ 防火墙配置完成"
}

# 设置 Webhook 服务
setup_webhook() {
    log "[9/10] 设置 Webhook 服务..."

    # 复制 webhook 脚本
    cp "$INSTALL_DIR/deploy/webhook-server.sh" /usr/local/bin/sub2api-webhook
    chmod +x /usr/local/bin/sub2api-webhook

    # 复制部署脚本
    cp "$INSTALL_DIR/deploy/chengdu-deploy.sh" "$INSTALL_DIR/deploy.sh"
    chmod +x "$INSTALL_DIR/deploy.sh"

    # 创建 systemd 服务
    cat > /etc/systemd/system/sub2api-webhook.service << EOF
[Unit]
Description=Sub2API Webhook Service
After=network.target

[Service]
Type=simple
User=root
Environment="WEBHOOK_SECRET=$WEBHOOK_SECRET"
ExecStart=/usr/local/bin/sub2api-webhook
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable sub2api-webhook
    systemctl start sub2api-webhook

    log_success "✅ Webhook 服务已启动"
}

# 生成配置文件
generate_config() {
    log "[10/10] 生成配置文件..."

    cd "$INSTALL_DIR"

    # 生成 .env 文件
    if [ ! -f ".env" ]; then
        JWT_SECRET=$(openssl rand -hex 32)
        TOTP_KEY=$(openssl rand -hex 32)
        PG_PASS=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)
        ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d '/+=' | head -c 16)

        cat > .env << EOF
BIND_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_MODE=release
TZ=Asia/Shanghai
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=$PG_PASS
POSTGRES_DB=sub2api
REDIS_PASSWORD=
REDIS_DB=0
ADMIN_EMAIL=admin@sub2api.local
ADMIN_PASSWORD=$ADMIN_PASSWORD
JWT_SECRET=$JWT_SECRET
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=$TOTP_KEY
EOF
        chmod 600 .env

        # 保存凭证
        cat > /root/sub2api-credentials.txt << EOF
========================================
Sub2API 部署凭证
========================================
服务器IP: $SERVER_IP
访问地址: http://$SERVER_IP:8080
管理员邮箱: admin@sub2api.local
管理员密码: $ADMIN_PASSWORD
JWT_SECRET: $JWT_SECRET
TOTP_KEY: $TOTP_KEY
PostgreSQL密码: $PG_PASS
Webhook Secret: $WEBHOOK_SECRET
========================================
EOF
        chmod 600 /root/sub2api-credentials.txt
    fi

    log_success "✅ 配置文件生成完成"
}

# 显示完成信息
show_completion() {
    echo ""
    echo "=========================================="
    echo "🎉 服务器初始化完成！"
    echo "=========================================="
    echo ""
    echo "📋 重要信息："
    echo "  - 凭证文件: /root/sub2api-credentials.txt"
    echo "  - Webhook Secret: $WEBHOOK_SECRET"
    echo "  - Webhook 端口: $WEBHOOK_PORT"
    echo ""
    echo "🔧 下一步操作："
    echo "  1. 在 Gitee 仓库设置中添加 Webhook："
    echo "     URL: http://$SERVER_IP:$WEBHOOK_PORT/webhook"
    echo "     Secret: $WEBHOOK_SECRET"
    echo "     触发事件: Push"
    echo ""
    echo "  2. 首次部署："
    echo "     cd $INSTALL_DIR && bash deploy.sh"
    echo ""
    echo "  3. 查看 Webhook 日志："
    echo "     tail -f /var/log/sub2api-webhook.log"
    echo ""
    echo "=========================================="
}

# 主函数
main() {
    log "=========================================="
    log "Sub2API 成都服务器初始化"
    log "服务器: $SERVER_IP"
    log "=========================================="
    echo ""

    check_root
    update_system
    install_basic_tools
    install_docker
    install_go
    install_nodejs
    setup_git
    clone_repo
    setup_firewall
    setup_webhook
    generate_config
    show_completion
}

main "$@"
