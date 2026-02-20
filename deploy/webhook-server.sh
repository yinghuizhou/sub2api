#!/bin/bash
# =============================================================================
# Sub2API Webhook 接收服务
# 用途：在服务器上监听 Gitee Webhook，自动触发部署
# 端口：9000
# =============================================================================

set -euo pipefail

WEBHOOK_PORT=9000
WEBHOOK_SECRET="${WEBHOOK_SECRET:?ERROR: WEBHOOK_SECRET environment variable is required}"
DEPLOY_SCRIPT="/opt/sub2api/deploy.sh"
LOG_FILE="/var/log/sub2api-webhook.log"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

# 检查依赖
check_deps() {
    if ! command -v nc &> /dev/null; then
        log_error "需要安装 netcat"
        exit 1
    fi
}

# 处理 Webhook 请求
handle_webhook() {
    local request="$1"

    # 提取 token
    local token=$(echo "$request" | grep -i "X-Gitee-Token:" | cut -d' ' -f2 | tr -d '\r')

    # 验证 token
    if [ "$token" != "$WEBHOOK_SECRET" ]; then
        log_error "❌ Webhook token 验证失败"
        echo -e "HTTP/1.1 401 Unauthorized\r\nContent-Type: text/plain\r\n\r\nUnauthorized"
        return
    fi

    # 提取请求体
    local body=$(echo "$request" | sed -n '/^{/,$p')

    log "📥 收到 Webhook 请求"
    log "请求体: $body"

    # 异步执行部署
    (
        log "🚀 开始部署..."
        if bash "$DEPLOY_SCRIPT" >> "$LOG_FILE" 2>&1; then
            log_success "✅ 部署成功"
        else
            log_error "❌ 部署失败"
        fi
    ) &

    # 立即返回响应
    echo -e "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"status\":\"deploying\"}"
}

# 启动 Webhook 服务器
start_server() {
    log "🚀 启动 Webhook 服务器..."
    log "监听端口: $WEBHOOK_PORT"
    log "日志文件: $LOG_FILE"

    while true; do
        # 使用 nc 监听端口
        request=$(nc -l -p "$WEBHOOK_PORT" -q 1)

        # 检查是否是 POST 请求
        if echo "$request" | grep -q "POST /webhook"; then
            handle_webhook "$request"
        else
            echo -e "HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\nNot Found"
        fi
    done
}

# 主函数
main() {
    check_deps

    # 创建日志文件
    touch "$LOG_FILE"
    chmod 644 "$LOG_FILE"

    log "=========================================="
    log "Sub2API Webhook 服务"
    log "=========================================="

    start_server
}

main "$@"
