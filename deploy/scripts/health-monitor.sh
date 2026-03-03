#!/bin/bash
# 健康监控脚本
# 用法: ./health-monitor.sh

set -e

# 配置
HEALTH_URL="http://localhost:8080/health"
LOG_FILE="/var/log/sub2api-health.log"
ALERT_WEBHOOK="${ALERT_WEBHOOK_URL:-}"

# 检查健康状态
check_health() {
    RESPONSE=$(curl -sf "$HEALTH_URL" 2>&1)
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ]; then
        echo "[$(date)] ✓ 健康检查通过: $RESPONSE" >> "$LOG_FILE"
        return 0
    else
        echo "[$(date)] ✗ 健康检查失败: $RESPONSE" >> "$LOG_FILE"
        return 1
    fi
}

# 发送告警
send_alert() {
    local message="$1"

    if [ -n "$ALERT_WEBHOOK" ]; then
        curl -X POST "$ALERT_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"msgtype\":\"text\",\"text\":{\"content\":\"[Sub2API 告警] $message\"}}" \
            2>&1 >> "$LOG_FILE"
    fi
}

# 主逻辑
if ! check_health; then
    send_alert "服务健康检查失败，请立即检查！"

    # 尝试重启服务
    echo "[$(date)] 尝试重启服务..." >> "$LOG_FILE"
    cd /opt/sub2api && docker compose restart sub2api

    # 等待 10 秒后再次检查
    sleep 10
    if check_health; then
        send_alert "服务已自动恢复"
    else
        send_alert "服务重启后仍然失败，需要人工介入！"
    fi
fi

# 清理旧日志（保留最近 1000 行）
tail -n 1000 "$LOG_FILE" > "${LOG_FILE}.tmp" && mv "${LOG_FILE}.tmp" "$LOG_FILE"
