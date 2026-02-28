# 二次分发配置示例

本文档提供二次分发功能的完整配置示例，帮助快速上手。

## 前提条件

- Sub2API 服务已部署并运行
- 拥有 Admin API Key
- 已选择上游中转站供应商

## 场景 1：按量计费上游

### 1.1 创建账户

```bash
curl -X POST http://localhost:8080/api/v1/admin/accounts \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Upstream Provider A - Pay as you go",
    "platform": "anthropic",
    "type": "apikey",
    "credentials": {
      "api_key": "sk-upstream-xxx",
      "base_url": "https://api.upstream-provider-a.com"
    },
    "rate_multiplier": 1.5,
    "priority": 30,
    "concurrency": 5,
    "group_ids": [1, 2]
  }'
```

**参数说明**：

- `base_url`: 上游中转站的 API 端点
- `rate_multiplier`: 1.5 表示成本 × 1.5 倍加价
- `priority`: 30（数字越小优先级越高）
- `concurrency`: 最大并发数 5
- `group_ids`: 分配到分组 1 和 2

### 1.2 测试请求

```bash
curl -X POST http://localhost:8080/v1/messages \
  -H "Authorization: Bearer your-user-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4",
    "max_tokens": 100,
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

### 1.3 验证计费

```bash
# 查询用户余额变化
curl -X GET http://localhost:8080/api/v1/admin/users/1 \
  -H "x-api-key: your-admin-api-key"

# 查询使用日志
curl -X GET "http://localhost:8080/api/v1/admin/usage-logs?user_id=1&limit=10" \
  -H "x-api-key: your-admin-api-key"
```

## 场景 2：订阅计费上游（每日限额）

### 2.1 创建账户

```bash
curl -X POST http://localhost:8080/api/v1/admin/accounts \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Upstream Provider B - Monthly $300",
    "platform": "anthropic",
    "type": "apikey",
    "credentials": {
      "api_key": "sk-upstream-yyy",
      "base_url": "https://api.upstream-provider-b.com"
    },
    "rate_multiplier": 1.2,
    "priority": 20,
    "concurrency": 10,
    "group_ids": [1]
  }'
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 124,
    "name": "Upstream Provider B - Monthly $300",
    ...
  }
}
```

### 2.2 配置订阅限额

```bash
curl -X POST http://localhost:8080/api/v1/admin/accounts/124/subscription \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "daily_limit_usd": 15.0,
    "subscription_period": "monthly",
    "subscription_start": "2026-03-01T00:00:00Z",
    "subscription_end": "2026-04-01T00:00:00Z"
  }'
```

**参数说明**：

- `daily_limit_usd`: 15.0（月订阅 $300 ÷ 30 天 × 1.5 倍缓冲）
- `subscription_period`: monthly（按月订阅）
- `subscription_start/end`: 订阅有效期

### 2.3 查询订阅配置

```bash
curl -X GET http://localhost:8080/api/v1/admin/accounts/124/subscription \
  -H "x-api-key: your-admin-api-key"
```

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "enabled": true,
    "daily_limit_usd": 15.0,
    "subscription_period": "monthly",
    "subscription_start": "2026-03-01T00:00:00Z",
    "subscription_end": "2026-04-01T00:00:00Z"
  }
}
```

### 2.4 监控日用量

```bash
# 查询今日用量
curl -X GET http://localhost:8080/api/v1/admin/accounts/124/daily-usage \
  -H "x-api-key: your-admin-api-key"

# 查询指定日期用量
curl -X GET "http://localhost:8080/api/v1/admin/accounts/124/daily-usage?date=2026-03-01" \
  -H "x-api-key: your-admin-api-key"
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "account_id": 124,
    "date": "2026-03-01",
    "usage_usd": 8.45
  }
}
```

### 2.5 订阅续费

```bash
curl -X POST http://localhost:8080/api/v1/admin/accounts/124/subscription \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "daily_limit_usd": 15.0,
    "subscription_period": "monthly",
    "subscription_start": "2026-03-01T00:00:00Z",
    "subscription_end": "2026-05-01T00:00:00Z"
  }'
```

## 场景 3：多账户负载均衡

### 3.1 配置主账户（高优先级）

```bash
curl -X POST http://localhost:8080/api/v1/admin/accounts \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Primary Upstream - High Priority",
    "platform": "anthropic",
    "type": "apikey",
    "credentials": {
      "api_key": "sk-primary-xxx",
      "base_url": "https://api.primary-upstream.com"
    },
    "rate_multiplier": 1.3,
    "priority": 10,
    "concurrency": 20,
    "group_ids": [1]
  }'
```

### 3.2 配置备用账户（低优先级）

```bash
curl -X POST http://localhost:8080/api/v1/admin/accounts \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Backup Upstream - Low Priority",
    "platform": "anthropic",
    "type": "apikey",
    "credentials": {
      "api_key": "sk-backup-xxx",
      "base_url": "https://api.backup-upstream.com"
    },
    "rate_multiplier": 1.5,
    "priority": 50,
    "concurrency": 10,
    "group_ids": [1]
  }'
```

### 3.3 配置订阅限额（两个账户）

```bash
# 主账户：月订阅 $500，日限额 $25
curl -X POST http://localhost:8080/api/v1/admin/accounts/125/subscription \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "daily_limit_usd": 25.0,
    "subscription_period": "monthly",
    "subscription_start": "2026-03-01T00:00:00Z",
    "subscription_end": "2026-04-01T00:00:00Z"
  }'

# 备用账户：月订阅 $300，日限额 $15
curl -X POST http://localhost:8080/api/v1/admin/accounts/126/subscription \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "daily_limit_usd": 15.0,
    "subscription_period": "monthly",
    "subscription_start": "2026-03-01T00:00:00Z",
    "subscription_end": "2026-04-01T00:00:00Z"
  }'
```

### 3.4 调度行为

调度器会按以下顺序选择账户：

1. **优先级排序**：priority 10 > priority 50
2. **日限额检查**：过滤超限账户
3. **轮转选择**：在可用账户中轮转

**示例流程**：

- 主账户（priority 10）日用量 < $25 → 优先使用
- 主账户超限 → 自动切换到备用账户（priority 50）
- 备用账户也超限 → 返回 503 错误

## 场景 4：混合模式（按量 + 订阅）

### 4.1 配置策略

```bash
# 账户 A：按量计费，高优先级（用于高峰期）
curl -X POST http://localhost:8080/api/v1/admin/accounts \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pay-as-you-go Upstream",
    "platform": "anthropic",
    "type": "apikey",
    "credentials": {
      "api_key": "sk-payg-xxx",
      "base_url": "https://api.payg-upstream.com"
    },
    "rate_multiplier": 1.8,
    "priority": 60,
    "group_ids": [1]
  }'

# 账户 B：订阅计费，低优先级（日常使用）
curl -X POST http://localhost:8080/api/v1/admin/accounts \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Subscription Upstream",
    "platform": "anthropic",
    "type": "apikey",
    "credentials": {
      "api_key": "sk-sub-xxx",
      "base_url": "https://api.sub-upstream.com"
    },
    "rate_multiplier": 1.2,
    "priority": 20,
    "group_ids": [1]
  }'

# 配置订阅限额
curl -X POST http://localhost:8080/api/v1/admin/accounts/128/subscription \
  -H "x-api-key: your-admin-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "daily_limit_usd": 20.0,
    "subscription_period": "monthly",
    "subscription_start": "2026-03-01T00:00:00Z",
    "subscription_end": "2026-04-01T00:00:00Z"
  }'
```

### 4.2 成本优化策略

- **日常流量**：优先使用订阅账户（priority 20，成本低）
- **订阅超限**：自动切换到按量账户（priority 60，成本高但无限额）
- **成本控制**：订阅账户承担 80% 流量，按量账户承担 20% 峰值

## 监控脚本

### 每日用量监控

```bash
#!/bin/bash
# monitor-daily-usage.sh

ADMIN_API_KEY="your-admin-api-key"
BASE_URL="http://localhost:8080"
ACCOUNT_IDS=(124 125 126)

for account_id in "${ACCOUNT_IDS[@]}"; do
  echo "Checking account $account_id..."

  # 查询订阅配置
  config=$(curl -s -X GET "$BASE_URL/api/v1/admin/accounts/$account_id/subscription" \
    -H "x-api-key: $ADMIN_API_KEY")

  daily_limit=$(echo "$config" | jq -r '.data.daily_limit_usd')

  if [ "$daily_limit" != "null" ] && [ "$daily_limit" != "0" ]; then
    # 查询今日用量
    usage=$(curl -s -X GET "$BASE_URL/api/v1/admin/accounts/$account_id/daily-usage" \
      -H "x-api-key: $ADMIN_API_KEY")

    usage_usd=$(echo "$usage" | jq -r '.data.usage_usd')
    percentage=$(echo "scale=2; $usage_usd / $daily_limit * 100" | bc)

    echo "  Daily limit: \$$daily_limit"
    echo "  Current usage: \$$usage_usd ($percentage%)"

    # 告警阈值 80%
    if (( $(echo "$percentage > 80" | bc -l) )); then
      echo "  ⚠️  WARNING: Usage exceeds 80%!"
      # 发送告警通知（邮件/Slack/钉钉等）
    fi
  fi

  echo ""
done
```

### 订阅到期提醒

```bash
#!/bin/bash
# check-subscription-expiry.sh

ADMIN_API_KEY="your-admin-api-key"
BASE_URL="http://localhost:8080"
ACCOUNT_IDS=(124 125 126)
WARN_DAYS=3

for account_id in "${ACCOUNT_IDS[@]}"; do
  config=$(curl -s -X GET "$BASE_URL/api/v1/admin/accounts/$account_id/subscription" \
    -H "x-api-key: $ADMIN_API_KEY")

  subscription_end=$(echo "$config" | jq -r '.data.subscription_end')

  if [ "$subscription_end" != "null" ]; then
    end_timestamp=$(date -d "$subscription_end" +%s)
    now_timestamp=$(date +%s)
    days_remaining=$(( ($end_timestamp - $now_timestamp) / 86400 ))

    echo "Account $account_id expires in $days_remaining days"

    if [ $days_remaining -le $WARN_DAYS ]; then
      echo "  ⚠️  WARNING: Subscription expires soon!"
      # 发送续费提醒
    fi
  fi
done
```

## 故障排查

### 问题 1：请求不路由到上游账户

```bash
# 1. 检查账户状态
curl -X GET http://localhost:8080/api/v1/admin/accounts/124 \
  -H "x-api-key: your-admin-api-key"

# 2. 检查订阅配置
curl -X GET http://localhost:8080/api/v1/admin/accounts/124/subscription \
  -H "x-api-key: your-admin-api-key"

# 3. 检查日用量
curl -X GET http://localhost:8080/api/v1/admin/accounts/124/daily-usage \
  -H "x-api-key: your-admin-api-key"

# 4. 检查分组配置
curl -X GET http://localhost:8080/api/v1/admin/groups/1 \
  -H "x-api-key: your-admin-api-key"
```

### 问题 2：日用量不更新

```bash
# 1. 检查 Redis 连接
redis-cli ping

# 2. 查看 Redis Key
redis-cli keys "billing:account_daily:*"

# 3. 查看具体值
redis-cli get "billing:account_daily:124:2026-03-01"

# 4. 检查服务日志
docker logs sub2api | grep "failed to increment account daily usage"
```

### 问题 3：超限后仍被调度

```bash
# 1. 重启服务刷新缓存
docker restart sub2api

# 2. 手动重置日用量（仅测试环境）
redis-cli del "billing:account_daily:124:2026-03-01"

# 3. 检查时区设置
date -u  # 确保使用 UTC 时间
```

## 最佳实践

### 1. 日限额设置

```
daily_limit = (monthly_subscription / 30) × 1.5
```

留出 50% 缓冲，避免意外超限。

### 2. 优先级规划

- 10-20：高优先级（订阅账户，成本低）
- 30-50：中优先级（按量账户，成本中等）
- 60-100：低优先级（备用账户，成本高）

### 3. 并发数设置

根据上游限制设置：

- 小型上游：concurrency = 3-5
- 中型上游：concurrency = 10-20
- 大型上游：concurrency = 50-100

### 4. 监控频率

- 日用量监控：每小时
- 订阅到期检查：每天
- 账户健康检查：每 5 分钟

## 相关文档

- [Admin API 文档](../docs/api/admin-subscription.md)
- [技术方案](../RESELLING_PLAN.md)
- [调度器文档](../docs/scheduler.md)
