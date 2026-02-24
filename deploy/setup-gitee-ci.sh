#!/bin/bash
# =============================================================================
# Sub2API Gitee CI/CD Setup Script
# Run this once on the HK server to set up auto-deploy from Gitee.
#
# What it does:
#   1. Clones the Gitee repo to /opt/sub2api/source/
#   2. Installs webhook-server.sh + deploy.sh
#   3. Creates systemd service for the webhook server
#   4. Configures Caddy to reverse proxy the webhook endpoint
#   5. Opens firewall for webhook if needed
#
# Prerequisites:
#   - Server already has Docker + Docker Compose
#   - Server already has /opt/sub2api/ with docker-compose.local.yml + .env
#   - Caddy is installed and running
#   - Git is installed
#
# Usage:
#   sudo bash setup-gitee-ci.sh
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()    { echo -e "${CYAN}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

INSTALL_DIR="/opt/sub2api"
SOURCE_DIR="$INSTALL_DIR/source"
GITEE_REPO="git@gitee.com:xixi_24/sub2api.git"
WEBHOOK_PORT=9000

# --- Check root ---
if [[ $EUID -ne 0 ]]; then
    log_error "请使用 sudo 运行: sudo bash $0"
    exit 1
fi

echo "============================================="
echo "  Sub2API Gitee CI/CD 自动部署安装"
echo "============================================="
echo ""

# --- Step 1: Clone Gitee repo ---
log_info "Step 1/5: 克隆 Gitee 仓库..."
if [ -d "$SOURCE_DIR/.git" ]; then
    log_warn "源码目录已存在，执行 git pull..."
    cd "$SOURCE_DIR" && git pull origin main
else
    git clone "$GITEE_REPO" "$SOURCE_DIR"
fi
log_success "源码就绪: $SOURCE_DIR"

# --- Step 2: Install scripts ---
log_info "Step 2/5: 安装部署脚本..."
cp "$SOURCE_DIR/deploy/webhook-server.sh" "$INSTALL_DIR/webhook-server.sh"
cp "$SOURCE_DIR/deploy/deploy.sh" "$INSTALL_DIR/deploy.sh"
chmod +x "$INSTALL_DIR/webhook-server.sh" "$INSTALL_DIR/deploy.sh"
log_success "脚本已安装"

# --- Step 3: Generate webhook secret & env file ---
log_info "Step 3/5: 配置 Webhook..."
if [ -f "$INSTALL_DIR/.webhook.env" ]; then
    log_warn ".webhook.env 已存在，保留现有配置"
    source "$INSTALL_DIR/.webhook.env"
else
    WEBHOOK_SECRET=$(openssl rand -hex 20)
    cat > "$INSTALL_DIR/.webhook.env" << EOF
WEBHOOK_SECRET=$WEBHOOK_SECRET
WEBHOOK_PORT=$WEBHOOK_PORT
DEPLOY_SCRIPT=$INSTALL_DIR/deploy.sh
LOG_FILE=/var/log/sub2api-webhook.log
EOF
    chmod 600 "$INSTALL_DIR/.webhook.env"
    log_success "Webhook Secret 已生成"
fi

# Read the secret for display later
source "$INSTALL_DIR/.webhook.env"

# --- Step 4: Install systemd service ---
log_info "Step 4/5: 安装 systemd 服务..."
cp "$SOURCE_DIR/deploy/sub2api-webhook.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable sub2api-webhook
systemctl restart sub2api-webhook
sleep 2

if systemctl is-active --quiet sub2api-webhook; then
    log_success "Webhook 服务运行中 (端口 $WEBHOOK_PORT)"
else
    log_error "Webhook 服务启动失败，请检查: journalctl -u sub2api-webhook"
    exit 1
fi

# --- Step 5: Configure Caddy reverse proxy ---
log_info "Step 5/5: 配置 Caddy 代理..."
CADDY_FILE="/etc/caddy/Caddyfile"
if [ -f "$CADDY_FILE" ]; then
    if grep -q "webhook" "$CADDY_FILE"; then
        log_warn "Caddy 已包含 webhook 配置，跳过"
    else
        log_warn "请手动在 Caddyfile 中添加 webhook 路由 (详见下方说明)"
    fi
else
    log_warn "未检测到 Caddy 配置，webhook 将通过端口 $WEBHOOK_PORT 直接访问"
    # Open firewall
    if command -v ufw &> /dev/null; then
        ufw allow "$WEBHOOK_PORT/tcp" comment 'Sub2API Webhook'
        log_success "防火墙已放行端口 $WEBHOOK_PORT"
    fi
fi

# --- Done ---
SERVER_IP=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')

echo ""
echo "============================================="
echo "  ✅ Gitee CI/CD 安装完成！"
echo "============================================="
echo ""
echo "  Webhook URL:    http://$SERVER_IP:$WEBHOOK_PORT/webhook"
echo "  Webhook Secret: $WEBHOOK_SECRET"
echo "  Health Check:   http://$SERVER_IP:$WEBHOOK_PORT/health"
echo ""
echo "  ⚡ Gitee 配置步骤:"
echo "  1. 打开 Gitee 仓库 → 管理 → WebHooks"
echo "  2. 添加 WebHook:"
echo "     URL:    http://$SERVER_IP:$WEBHOOK_PORT/webhook"
echo "     密码:   $WEBHOOK_SECRET"
echo "     事件:   Push"
echo "     分支:   main"
echo ""
echo "  📋 常用命令:"
echo "     查看日志:  journalctl -u sub2api-webhook -f"
echo "     重启服务:  systemctl restart sub2api-webhook"
echo "     查看状态:  systemctl status sub2api-webhook"
echo "     手动部署:  bash $INSTALL_DIR/deploy.sh"
echo ""
echo "============================================="
