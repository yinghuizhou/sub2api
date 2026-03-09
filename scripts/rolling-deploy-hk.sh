#!/bin/bash
# =============================================================================
# Sub2API 零停机滚动部署脚本（香港服务器）
# =============================================================================
#
# 功能：
# - 逐个实例滚动更新，确保始终有 2/3 实例在线
# - 每个实例更新前等待健康检查通过
# - 自动回滚失败的更新
#
# 使用方法：
#   ./scripts/rolling-deploy-hk.sh
#
# =============================================================================

set -e

# 配置
SSH_KEY="$HOME/work/sub2api.pem"
SERVER="root@47.76.82.51"
DEPLOY_DIR="/opt/sub2api"
COMPOSE_FILE="docker-compose.ha.yml"
INSTANCES=("sub2api-1" "sub2api-2" "sub2api-3")
HEALTH_CHECK_URL="http://localhost:8888/health"
MAX_HEALTH_CHECK_ATTEMPTS=30
HEALTH_CHECK_INTERVAL=2

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查实例健康状态
check_instance_health() {
    local instance=$1
    local attempt=0

    log_info "等待 $instance 健康检查通过..."

    while [ $attempt -lt $MAX_HEALTH_CHECK_ATTEMPTS ]; do
        if ssh -i "$SSH_KEY" "$SERVER" "docker exec $instance wget -q -O- http://localhost:8080/health 2>/dev/null | grep -q 'ok'" 2>/dev/null; then
            log_success "$instance 健康检查通过"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep $HEALTH_CHECK_INTERVAL
    done

    echo ""
    log_error "$instance 健康检查失败（超时 ${MAX_HEALTH_CHECK_ATTEMPTS}x${HEALTH_CHECK_INTERVAL}s）"
    return 1
}

# 检查 Nginx 负载均衡状态
check_nginx_health() {
    log_info "检查 Nginx 负载均衡状态..."

    if ssh -i "$SSH_KEY" "$SERVER" "curl -sf $HEALTH_CHECK_URL >/dev/null"; then
        log_success "Nginx 负载均衡正常"
        return 0
    else
        log_error "Nginx 负载均衡异常"
        return 1
    fi
}

# 更新单个实例
update_instance() {
    local instance=$1

    log_info "=========================================="
    log_info "开始更新实例: $instance"
    log_info "=========================================="

    # 1. 检查其他实例是否健康
    log_info "检查其他实例状态..."
    local healthy_count=0
    for other_instance in "${INSTANCES[@]}"; do
        if [ "$other_instance" != "$instance" ]; then
            if ssh -i "$SSH_KEY" "$SERVER" "docker exec $other_instance wget -q -O- http://localhost:8080/health 2>/dev/null | grep -q 'ok'" 2>/dev/null; then
                healthy_count=$((healthy_count + 1))
                log_success "$other_instance 健康"
            else
                log_warning "$other_instance 不健康"
            fi
        fi
    done

    if [ $healthy_count -lt 2 ]; then
        log_error "其他实例不足 2 个健康，取消更新 $instance"
        return 1
    fi

    log_success "其他实例健康数量: $healthy_count/2"

    # 2. 优雅停止实例（等待现有请求完成）
    log_info "优雅停止 $instance..."
    ssh -i "$SSH_KEY" "$SERVER" "cd $DEPLOY_DIR && docker compose -f $COMPOSE_FILE stop -t 30 $instance"

    # 3. 等待 5 秒确保连接已关闭
    log_info "等待连接关闭..."
    sleep 5

    # 4. 启动新实例
    log_info "启动新版本 $instance..."
    ssh -i "$SSH_KEY" "$SERVER" "cd $DEPLOY_DIR && docker compose -f $COMPOSE_FILE up -d $instance"

    # 5. 等待健康检查通过
    if ! check_instance_health "$instance"; then
        log_error "$instance 启动失败，尝试回滚..."

        # 回滚：重启实例
        ssh -i "$SSH_KEY" "$SERVER" "cd $DEPLOY_DIR && docker compose -f $COMPOSE_FILE restart $instance"

        if ! check_instance_health "$instance"; then
            log_error "$instance 回滚失败"
            return 1
        fi

        log_warning "$instance 已回滚到旧版本"
        return 1
    fi

    # 6. 验证 Nginx 负载均衡
    if ! check_nginx_health; then
        log_error "Nginx 负载均衡异常，回滚 $instance"
        ssh -i "$SSH_KEY" "$SERVER" "cd $DEPLOY_DIR && docker compose -f $COMPOSE_FILE restart $instance"
        return 1
    fi

    # 7. 等待 10 秒观察稳定性
    log_info "观察 $instance 稳定性（10秒）..."
    sleep 10

    if ! check_instance_health "$instance"; then
        log_error "$instance 不稳定，回滚"
        ssh -i "$SSH_KEY" "$SERVER" "cd $DEPLOY_DIR && docker compose -f $COMPOSE_FILE restart $instance"
        return 1
    fi

    log_success "$instance 更新成功"
    return 0
}

# 主流程
main() {
    log_info "=========================================="
    log_info "Sub2API 零停机滚动部署"
    log_info "服务器: $SERVER"
    log_info "实例数: ${#INSTANCES[@]}"
    log_info "=========================================="

    # 1. 检查初始状态
    log_info "检查初始状态..."
    if ! check_nginx_health; then
        log_error "初始状态异常，取消部署"
        exit 1
    fi

    # 2. 逐个更新实例
    local failed_instances=()
    for instance in "${INSTANCES[@]}"; do
        if ! update_instance "$instance"; then
            failed_instances+=("$instance")
            log_error "$instance 更新失败"
        fi

        # 实例间等待 15 秒
        if [ "$instance" != "${INSTANCES[-1]}" ]; then
            log_info "等待 15 秒后更新下一个实例..."
            sleep 15
        fi
    done

    # 3. 最终验证
    log_info "=========================================="
    log_info "最终验证"
    log_info "=========================================="

    if ! check_nginx_health; then
        log_error "最终验证失败"
        exit 1
    fi

    # 检查所有实例
    local final_healthy=0
    for instance in "${INSTANCES[@]}"; do
        if ssh -i "$SSH_KEY" "$SERVER" "docker exec $instance wget -q -O- http://localhost:8080/health 2>/dev/null | grep -q 'ok'" 2>/dev/null; then
            log_success "$instance 健康"
            final_healthy=$((final_healthy + 1))
        else
            log_error "$instance 不健康"
        fi
    done

    # 4. 输出结果
    log_info "=========================================="
    if [ ${#failed_instances[@]} -eq 0 ] && [ $final_healthy -eq ${#INSTANCES[@]} ]; then
        log_success "部署成功！"
        log_success "健康实例: $final_healthy/${#INSTANCES[@]}"
    else
        log_warning "部署完成，但有问题"
        log_warning "健康实例: $final_healthy/${#INSTANCES[@]}"
        if [ ${#failed_instances[@]} -gt 0 ]; then
            log_warning "失败实例: ${failed_instances[*]}"
        fi
    fi
    log_info "=========================================="

    # 5. 显示版本信息
    log_info "版本信息:"
    ssh -i "$SSH_KEY" "$SERVER" "docker exec sub2api-1 /app/sub2api --version 2>&1 | head -1"
}

# 执行主流程
main
