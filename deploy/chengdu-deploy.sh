#!/bin/bash
# =============================================================================
# Sub2API 成都服务器部署脚本
# 服务器: 47.108.158.227
# 用途: 自动拉取代码、构建、部署
# =============================================================================

set -euo pipefail

# 配置
SERVER_IP="47.108.158.227"
INSTALL_DIR="/opt/sub2api"
REPO_URL="git@gitee.com:xixi_24/sub2api.git"
BRANCH="main"
BACKUP_DIR="/opt/sub2api/backups"
MAX_BACKUPS=5

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"; }
log_success() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"; }
log_error() { echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"; }

# 备份当前版本
backup_current() {
    log "📦 备份当前版本..."

    mkdir -p "$BACKUP_DIR"

    local backup_name="backup-$(date +%Y%m%d-%H%M%S)"
    local backup_path="$BACKUP_DIR/$backup_name"

    # 备份二进制文件和配置
    if [ -f "$INSTALL_DIR/server" ]; then
        mkdir -p "$backup_path"
        cp "$INSTALL_DIR/server" "$backup_path/"
        cp "$INSTALL_DIR/.env" "$backup_path/" 2>/dev/null || true
        cp "$INSTALL_DIR/docker-compose.yml" "$backup_path/" 2>/dev/null || true

        log_success "✅ 备份完成: $backup_path"

        # 清理旧备份（保留最近 5 个）
        cd "$BACKUP_DIR"
        ls -t | tail -n +$((MAX_BACKUPS + 1)) | xargs -r rm -rf
    else
        log_warn "⚠️  未找到现有版本，跳过备份"
    fi
}

# 拉取最新代码
pull_code() {
    log "📥 拉取最新代码..."

    cd "$INSTALL_DIR"

    # 如果是首次部署，克隆仓库
    if [ ! -d ".git" ]; then
        log "首次部署，克隆仓库..."
        cd /opt
        git clone "$REPO_URL" sub2api
        cd sub2api
    fi

    # 拉取最新代码
    git fetch origin

    # 检查是否有未提交的修改
    if ! git diff --quiet HEAD 2>/dev/null; then
        log_warn "⚠️  检测到未提交的修改，已自动 stash"
        git stash --include-untracked -m "auto-stash before deploy $(date +%Y%m%d-%H%M%S)"
    fi

    git reset --hard origin/$BRANCH

    local commit=$(git rev-parse --short HEAD)
    local message=$(git log -1 --pretty=%B)

    log_success "✅ 代码已更新到: $commit"
    log "提交信息: $message"
}

# 构建应用
build_app() {
    log "🔨 构建应用..."

    cd "$INSTALL_DIR"

    # 构建前端
    log "构建前端..."
    if ! command -v pnpm &> /dev/null; then
        npm install -g pnpm
    fi

    cd frontend
    pnpm install --frozen-lockfile
    pnpm run build

    # 构建后端
    log "构建后端..."
    cd "$INSTALL_DIR/backend"
    go mod download
    go build -tags embed -ldflags="-s -w" -o "$INSTALL_DIR/server" ./cmd/server

    log_success "✅ 构建完成"
}

# 停止旧服务
stop_service() {
    log "🛑 停止旧服务..."

    cd "$INSTALL_DIR"

    if docker compose ps | grep -q "sub2api"; then
        docker compose down
        log_success "✅ 服务已停止"
    else
        log_warn "⚠️  服务未运行"
    fi
}

# 启动新服务
start_service() {
    log "🚀 启动新服务..."

    cd "$INSTALL_DIR"

    # 拉取最新镜像（如果使用 Docker）
    if [ -f "docker-compose.yml" ]; then
        docker compose pull --quiet 2>/dev/null || log_warn "⚠️  镜像拉取跳过（可能无法访问 Docker Hub，请本地构建后传输）"
        docker compose up -d
    else
        # 直接运行二进制文件
        nohup ./server > logs/server.log 2>&1 &
    fi

    log_success "✅ 服务已启动"
}

# 健康检查
health_check() {
    log "🏥 健康检查..."

    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log_success "✅ 服务健康检查通过"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    log_error "❌ 健康检查失败"
    return 1
}

# 回滚
rollback() {
    log_error "🔄 开始回滚..."

    local latest_backup=$(ls -t "$BACKUP_DIR" | head -1)

    if [ -z "$latest_backup" ]; then
        log_error "❌ 没有可用的备份"
        return 1
    fi

    log "回滚到: $latest_backup"

    # 停止当前服务
    stop_service

    # 恢复备份
    cp "$BACKUP_DIR/$latest_backup/server" "$INSTALL_DIR/"
    cp "$BACKUP_DIR/$latest_backup/.env" "$INSTALL_DIR/" 2>/dev/null || true
    cp "$BACKUP_DIR/$latest_backup/docker-compose.yml" "$INSTALL_DIR/" 2>/dev/null || true

    # 启动服务
    start_service

    if health_check; then
        log_success "✅ 回滚成功"
        return 0
    else
        log_error "❌ 回滚失败"
        return 1
    fi
}

# 主函数
main() {
    log "=========================================="
    log "Sub2API 自动部署"
    log "服务器: $SERVER_IP"
    log "=========================================="

    # 防止并发部署
    exec 200>/var/lock/sub2api-deploy.lock
    if ! flock -n 200; then
        log_warn "⚠️  另一个部署正在进行中，退出"
        exit 0
    fi

    # 备份当前版本
    backup_current

    # 拉取最新代码
    if ! pull_code; then
        log_error "❌ 拉取代码失败"
        exit 1
    fi

    # 构建应用
    if ! build_app; then
        log_error "❌ 构建失败"
        exit 1
    fi

    # 停止旧服务
    stop_service

    # 启动新服务
    start_service

    # 健康检查
    if ! health_check; then
        log_error "❌ 部署失败，开始回滚..."
        rollback
        exit 1
    fi

    log_success "=========================================="
    log_success "🎉 部署成功！"
    log_success "=========================================="
    log_success "访问地址: http://$SERVER_IP:8080"
    log_success "提交版本: $(cd $INSTALL_DIR && git rev-parse --short HEAD)"
}

main "$@"
