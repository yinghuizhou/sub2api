#!/bin/bash
# =============================================================================
# Sub2API 快速回滚脚本
# 用途：快速回滚到上一个可用版本
# =============================================================================

set -euo pipefail

INSTALL_DIR="/opt/sub2api"
BACKUP_DIR="/opt/sub2api/backups"

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

# 列出可用备份
list_backups() {
    log "📋 可用备份列表："
    echo ""

    if [ ! -d "$BACKUP_DIR" ] || [ -z "$(ls -A $BACKUP_DIR 2>/dev/null)" ]; then
        log_error "❌ 没有可用的备份"
        exit 1
    fi

    local i=1
    for backup in $(ls -t "$BACKUP_DIR"); do
        echo "  [$i] $backup"
        i=$((i + 1))
    done
    echo ""
}

# 选择备份
select_backup() {
    list_backups

    echo -n "请选择要回滚的备份编号 (默认: 1 - 最新备份): "
    read selection

    selection=${selection:-1}

    local backup=$(ls -t "$BACKUP_DIR" | sed -n "${selection}p")

    if [ -z "$backup" ]; then
        log_error "❌ 无效的选择"
        exit 1
    fi

    echo "$backup"
}

# 执行回滚
rollback() {
    local backup_name="$1"
    local backup_path="$BACKUP_DIR/$backup_name"

    log "🔄 开始回滚到: $backup_name"

    # 检查备份是否存在
    if [ ! -d "$backup_path" ]; then
        log_error "❌ 备份不存在: $backup_path"
        exit 1
    fi

    # 停止当前服务
    log "🛑 停止当前服务..."
    cd "$INSTALL_DIR"
    docker compose down 2>/dev/null || true

    # 备份当前版本（以防回滚失败）
    log "📦 备份当前版本..."
    local emergency_backup="$BACKUP_DIR/emergency-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$emergency_backup"
    cp "$INSTALL_DIR/server" "$emergency_backup/" 2>/dev/null || true
    cp "$INSTALL_DIR/.env" "$emergency_backup/" 2>/dev/null || true

    # 恢复备份
    log "📥 恢复备份..."
    cp "$backup_path/server" "$INSTALL_DIR/"
    cp "$backup_path/.env" "$INSTALL_DIR/" 2>/dev/null || true
    cp "$backup_path/docker-compose.yml" "$INSTALL_DIR/" 2>/dev/null || true

    # 启动服务
    log "🚀 启动服务..."
    docker compose up -d 2>/dev/null || nohup ./server > logs/server.log 2>&1 &

    # 健康检查
    log "🏥 健康检查..."
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log_success "✅ 回滚成功！"
            log_success "服务已恢复到: $backup_name"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    log_error "❌ 回滚失败，健康检查未通过"
    log_error "紧急备份位置: $emergency_backup"
    return 1
}

# 主函数
main() {
    log "=========================================="
    log "Sub2API 快速回滚"
    log "=========================================="
    echo ""

    # 检查是否为 root
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 用户运行此脚本"
        exit 1
    fi

    # 选择备份
    local backup_name
    if [ $# -eq 0 ]; then
        backup_name=$(select_backup)
    else
        backup_name="$1"
    fi

    # 确认回滚
    log_warn "⚠️  即将回滚到: $backup_name"
    echo -n "确认继续? (y/N): "
    read confirm

    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        log "取消回滚"
        exit 0
    fi

    # 执行回滚
    if rollback "$backup_name"; then
        log_success "=========================================="
        log_success "🎉 回滚完成！"
        log_success "=========================================="
    else
        log_error "=========================================="
        log_error "❌ 回滚失败"
        log_error "=========================================="
        exit 1
    fi
}

main "$@"
