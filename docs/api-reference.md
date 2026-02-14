# API 接口参考

所有 API 均以 `/api/v1` 为前缀（网关接口除外）。响应格式统一为：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

错误响应中 `code != 0`，`message` 包含错误说明。

## 认证方式

### JWT Token（用户界面）

```http
Authorization: Bearer <access_token>
```

### API Key（AI 网关调用）

```http
x-api-key: sk-xxxxxxxx
# 或
Authorization: Bearer sk-xxxxxxxx
```

---

## 认证接口（/api/v1/auth）

### 用户注册

```http
POST /api/v1/auth/register
```

**请求体**

```json
{
  "email": "user@example.com",
  "password": "your_password",
  "verify_code": "123456",
  "promo_code": "PROMO20",
  "invitation_code": "INV-XXXX"
}
```

**说明**：`verify_code` 需要先调用"发送验证码"接口获取。`promo_code` 和 `invitation_code` 可选。

---

### 用户登录

```http
POST /api/v1/auth/login
```

**请求体**

```json
{
  "email": "user@example.com",
  "password": "your_password"
}
```

**响应**（无需 2FA）

```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 3600,
  "user": { "id": 1, "email": "user@example.com", "role": "user" }
}
```

**响应**（需要 2FA）

```json
{
  "requires_2fa": true,
  "temp_token": "temp_xxxx"
}
```

---

### 双因素认证登录

```http
POST /api/v1/auth/login/2fa
```

**请求体**

```json
{
  "temp_token": "temp_xxxx",
  "totp_code": "123456"
}
```

---

### 刷新 Token

```http
POST /api/v1/auth/refresh
```

**请求体**

```json
{
  "refresh_token": "eyJ..."
}
```

> 速率限制：30 次/分钟

---

### 登出

```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

---

### 获取当前用户

```http
GET /api/v1/auth/me
Authorization: Bearer <access_token>
```

---

### 撤销所有会话

```http
POST /api/v1/auth/revoke-all-sessions
Authorization: Bearer <access_token>
```

---

### 发送邮箱验证码

```http
POST /api/v1/auth/send-verify-code
```

**请求体**

```json
{
  "email": "user@example.com"
}
```

---

### 验证促销码

```http
POST /api/v1/auth/validate-promo-code
```

**请求体**

```json
{
  "code": "PROMO20"
}
```

> 速率限制：10 次/分钟

---

### 忘记密码

```http
POST /api/v1/auth/forgot-password
```

**请求体**

```json
{
  "email": "user@example.com"
}
```

> 速率限制：5 次/分钟

---

### 重置密码

```http
POST /api/v1/auth/reset-password
```

**请求体**

```json
{
  "token": "reset_token_from_email",
  "password": "new_password"
}
```

---

### 获取公开设置

```http
GET /api/v1/settings/public
```

返回站点名称、Logo、注册/支付策略等公开配置（无需认证）。

---

## 用户接口（/api/v1/user & /api/v1/keys）

> 所有接口需要 `Authorization: Bearer <access_token>`

### 获取用户资料

```http
GET /api/v1/user/profile
```

**响应**

```json
{
  "id": 1,
  "email": "user@example.com",
  "role": "user",
  "balance": 10.5,
  "concurrency": 5,
  "status": "active",
  "totp_enabled": false
}
```

---

### 更新用户资料

```http
PUT /api/v1/user
```

**请求体**

```json
{
  "display_name": "张三"
}
```

---

### 修改密码

```http
PUT /api/v1/user/password
```

**请求体**

```json
{
  "old_password": "old_password",
  "new_password": "new_password"
}
```

---

### TOTP 双因素认证

```http
# 获取状态
GET /api/v1/user/totp/status

# 初始化 TOTP（返回二维码 URL）
POST /api/v1/user/totp/setup

# 启用 TOTP（提交验证码确认）
POST /api/v1/user/totp/enable
{ "code": "123456" }

# 禁用 TOTP
POST /api/v1/user/totp/disable
{ "code": "123456" }
```

---

### API Key 管理

#### 列出 API Keys

```http
GET /api/v1/keys?page=1&page_size=20
```

**响应**

```json
{
  "items": [
    {
      "id": 1,
      "key": "sk-xxxxx",
      "name": "我的 Key",
      "group_id": 2,
      "status": "active",
      "quota": 0,
      "quota_used": 1.234,
      "expires_at": null,
      "ip_whitelist": [],
      "ip_blacklist": []
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### 创建 API Key

```http
POST /api/v1/keys
```

**请求体**

```json
{
  "name": "生产 Key",
  "group_id": 2,
  "quota": 0,
  "expires_in_days": 30,
  "ip_whitelist": ["192.168.1.0/24"],
  "ip_blacklist": []
}
```

**说明**：`quota` 为 0 表示无限制。`group_id` 决定可使用的账户池和计费倍率。

#### 更新 API Key

```http
PUT /api/v1/keys/:id
```

#### 删除 API Key

```http
DELETE /api/v1/keys/:id
```

---

### 获取可用分组

```http
GET /api/v1/groups/available
```

返回当前用户可使用的分组列表（根据用户权限过滤）。

---

### 使用记录

#### 列出使用记录

```http
GET /api/v1/usage?page=1&page_size=20&start_date=2025-01-01&end_date=2025-01-31&model=claude-3-opus-20240229
```

**查询参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码（默认 1）|
| page_size | int | 每页数量（默认 20）|
| start_date | string | 开始日期（ISO 8601）|
| end_date | string | 结束日期（ISO 8601）|
| model | string | 按模型筛选 |
| api_key_id | int | 按 API Key 筛选 |

#### Dashboard 统计

```http
GET /api/v1/usage/dashboard/stats
GET /api/v1/usage/dashboard/trend?period=7d
GET /api/v1/usage/dashboard/models
```

---

### 公告

```http
GET /api/v1/announcements
POST /api/v1/announcements/:id/read
```

---

### 兑换码

```http
# 兑换
POST /api/v1/redeem
{ "code": "XXXXXX" }

# 历史记录
GET /api/v1/redeem/history
```

---

### 订阅

```http
GET /api/v1/subscriptions/active       # 活跃订阅列表
GET /api/v1/subscriptions/progress     # 订阅用量进度
GET /api/v1/subscriptions/summary      # 订阅摘要
```

---

## AI 网关接口

> 使用 `x-api-key: sk-xxx` 或 `Authorization: Bearer sk-xxx` 认证

### Claude API

#### 发送消息

```http
POST /v1/messages
x-api-key: sk-xxx
Content-Type: application/json
```

**请求体**（标准 Claude API 格式）

```json
{
  "model": "claude-opus-4-5-20250929",
  "max_tokens": 1024,
  "messages": [
    { "role": "user", "content": "Hello!" }
  ],
  "stream": false
}
```

**流式请求**：设置 `"stream": true`，响应为 SSE 格式。

#### 统计 Token

```http
POST /v1/messages/count_tokens
```

#### 列出可用模型

```http
GET /v1/models
```

#### 查询使用情况

```http
GET /v1/usage
```

---

### Gemini API

```http
GET /v1beta/models                          # 列出模型
GET /v1beta/models/:model                   # 获取模型详情
POST /v1beta/models/:model:generateContent  # 生成内容
POST /v1beta/models/:model:streamGenerateContent  # 流式生成
```

---

### Antigravity 专用路由

使用与 Claude 相同格式，但路由前缀不同：

```http
POST /antigravity/v1/messages
GET  /antigravity/v1/models
POST /antigravity/v1beta/models/:model:generateContent
```

---

## 管理员接口（/api/v1/admin）

> 需要管理员权限（role = "admin"）

### Dashboard

```http
GET /api/v1/admin/dashboard/stats           # 全局统计
GET /api/v1/admin/dashboard/realtime        # 实时指标（RPM/TPM）
GET /api/v1/admin/dashboard/trend?period=7d # 使用趋势
GET /api/v1/admin/dashboard/models          # 模型分布统计
```

---

### 用户管理

```http
GET    /api/v1/admin/users              # 列表（支持搜索/筛选）
GET    /api/v1/admin/users/:id          # 获取单个用户
POST   /api/v1/admin/users              # 创建用户
PUT    /api/v1/admin/users/:id          # 更新用户
DELETE /api/v1/admin/users/:id          # 删除用户
POST   /api/v1/admin/users/:id/balance  # 修改余额
GET    /api/v1/admin/users/:id/api-keys          # 用户的 API Keys
GET    /api/v1/admin/users/:id/balance-history   # 余额历史
GET    /api/v1/admin/users/:id/attributes        # 用户属性值
PUT    /api/v1/admin/users/:id/attributes        # 更新用户属性
```

**创建用户请求体**

```json
{
  "email": "newuser@example.com",
  "password": "password123",
  "role": "user",
  "balance": 10.0,
  "concurrency": 5
}
```

**修改余额请求体**

```json
{
  "amount": 50.0,
  "operation": "add",
  "notes": "充值"
}
```

---

### 分组管理

```http
GET    /api/v1/admin/groups             # 列表
GET    /api/v1/admin/groups/all         # 全部（不分页）
POST   /api/v1/admin/groups             # 创建
PUT    /api/v1/admin/groups/:id         # 更新
DELETE /api/v1/admin/groups/:id         # 删除
PUT    /api/v1/admin/groups/sort-order  # 更新排序
GET    /api/v1/admin/groups/:id/stats   # 分组统计
```

**创建分组请求体**

```json
{
  "name": "Claude 高级组",
  "platform": "claude",
  "subscription_type": "monthly",
  "rate_multiplier": 1.2,
  "is_exclusive": true,
  "daily_limit_usd": 100.0,
  "model_routing_enabled": false
}
```

---

### 账户管理

```http
GET    /api/v1/admin/accounts                     # 列表
POST   /api/v1/admin/accounts                     # 创建账户
PUT    /api/v1/admin/accounts/:id                 # 更新账户
DELETE /api/v1/admin/accounts/:id                 # 删除账户
POST   /api/v1/admin/accounts/:id/test            # 测试账户可用性
POST   /api/v1/admin/accounts/:id/refresh         # 刷新 OAuth Token
POST   /api/v1/admin/accounts/:id/refresh-tier    # 刷新账户等级
GET    /api/v1/admin/accounts/:id/stats           # 账户统计
POST   /api/v1/admin/accounts/:id/clear-error     # 清除错误状态
POST   /api/v1/admin/accounts/:id/schedulable     # 设置可调度性
POST   /api/v1/admin/accounts/bulk-update         # 批量更新
POST   /api/v1/admin/accounts/data                # 导入账户数据
GET    /api/v1/admin/accounts/data                # 导出账户数据
POST   /api/v1/admin/accounts/sync/crs            # 从 CRS 同步
```

**创建账户请求体**

```json
{
  "name": "Claude 账户 1",
  "platform": "claude",
  "type": "oauth",
  "credentials": {
    "access_token": "xxx",
    "refresh_token": "xxx"
  },
  "concurrency": 3,
  "priority": 1,
  "group_ids": [1, 2]
}
```

---

### 代理管理

```http
GET    /api/v1/admin/proxies            # 列表
GET    /api/v1/admin/proxies/all        # 全部
POST   /api/v1/admin/proxies            # 创建
PUT    /api/v1/admin/proxies/:id        # 更新
DELETE /api/v1/admin/proxies/:id        # 删除
POST   /api/v1/admin/proxies/:id/test   # 测试延迟
GET    /api/v1/admin/proxies/:id/accounts # 关联账户
POST   /api/v1/admin/proxies/batch-delete # 批量删除
```

**创建代理请求体**

```json
{
  "name": "代理 1",
  "protocol": "http",
  "host": "proxy.example.com",
  "port": 8080,
  "username": "user",
  "password": "pass"
}
```

---

### 兑换码管理

```http
GET  /api/v1/admin/redeem-codes             # 列表
POST /api/v1/admin/redeem-codes/generate    # 生成
GET  /api/v1/admin/redeem-codes/export      # 导出 CSV
POST /api/v1/admin/redeem-codes/batch-delete # 批量删除
POST /api/v1/admin/redeem-codes/:id/expire  # 使码过期
```

**生成兑换码请求体**

```json
{
  "type": "balance",
  "count": 10,
  "balance": 5.0,
  "validity_days": 30
}
```

---

### 订阅管理

```http
GET    /api/v1/admin/subscriptions              # 列表
POST   /api/v1/admin/subscriptions/assign       # 分配订阅
POST   /api/v1/admin/subscriptions/bulk-assign  # 批量分配
POST   /api/v1/admin/subscriptions/:id/extend   # 延期
DELETE /api/v1/admin/subscriptions/:id          # 撤销
```

**分配订阅请求体**

```json
{
  "user_id": 100,
  "group_id": 2,
  "starts_at": "2025-01-01T00:00:00Z",
  "expires_at": "2025-02-01T00:00:00Z",
  "notes": "测试订阅"
}
```

---

### 公告管理

```http
GET    /api/v1/admin/announcements              # 列表
POST   /api/v1/admin/announcements              # 创建
PUT    /api/v1/admin/announcements/:id          # 更新
DELETE /api/v1/admin/announcements/:id          # 删除
GET    /api/v1/admin/announcements/:id/read-status # 阅读统计
```

---

### 系统设置

```http
GET  /api/v1/admin/settings                         # 获取所有设置
PUT  /api/v1/admin/settings                         # 更新设置
POST /api/v1/admin/settings/test-smtp               # 测试 SMTP 连接
POST /api/v1/admin/settings/send-test-email         # 发送测试邮件
GET  /api/v1/admin/settings/admin-api-key           # 获取管理 API Key
POST /api/v1/admin/settings/admin-api-key/regenerate # 重新生成
```

**可配置项**（部分）

| Key | 说明 |
|-----|------|
| `site_name` | 站点名称 |
| `site_logo` | 站点 Logo URL |
| `registration_enabled` | 是否开放注册 |
| `email_verification_required` | 注册是否需要邮箱验证 |
| `smtp_host` | SMTP 服务器地址 |
| `smtp_port` | SMTP 端口 |
| `smtp_user` | SMTP 用户名 |
| `smtp_password` | SMTP 密码 |

---

### 运维监控（Ops）

```http
GET  /api/v1/admin/ops/concurrency             # 并发统计
GET  /api/v1/admin/ops/account-availability    # 账户可用性
GET  /api/v1/admin/ops/realtime-traffic        # 实时流量
GET  /api/v1/admin/ops/errors                  # 错误日志列表
GET  /api/v1/admin/ops/errors/:id              # 错误详情
POST /api/v1/admin/ops/errors/:id/retry        # 重试错误请求
PUT  /api/v1/admin/ops/errors/:id/resolve      # 标记已解决
GET  /api/v1/admin/ops/alert-rules             # 告警规则列表
POST /api/v1/admin/ops/alert-rules             # 创建告警规则
PUT  /api/v1/admin/ops/alert-rules/:id         # 更新告警规则
DELETE /api/v1/admin/ops/alert-rules/:id       # 删除告警规则
GET  /api/v1/admin/ops/alert-events            # 告警事件列表
GET  /api/v1/admin/ops/ws/qps                  # QPS WebSocket 推送
```

---

### 使用记录管理

```http
GET  /api/v1/admin/usage                       # 高级筛选列表
GET  /api/v1/admin/usage/stats                 # 统计汇总
POST /api/v1/admin/usage/cleanup-tasks         # 创建清理任务
GET  /api/v1/admin/usage/cleanup-tasks         # 清理任务列表
POST /api/v1/admin/usage/cleanup-tasks/:id/cancel # 取消清理任务
```

---

### 用户属性定义

```http
GET    /api/v1/admin/user-attributes          # 属性定义列表
POST   /api/v1/admin/user-attributes          # 创建属性定义
PUT    /api/v1/admin/user-attributes/:id      # 更新属性定义
DELETE /api/v1/admin/user-attributes/:id      # 删除属性定义
PUT    /api/v1/admin/user-attributes/reorder  # 重新排序
```

---

### 错误透传规则

```http
GET    /api/v1/admin/error-passthrough-rules
POST   /api/v1/admin/error-passthrough-rules
PUT    /api/v1/admin/error-passthrough-rules/:id
DELETE /api/v1/admin/error-passthrough-rules/:id
```

---

### OAuth 管理

```http
# Anthropic (Claude) OAuth
POST /api/v1/admin/openai/generate-auth-url     # 生成授权 URL
POST /api/v1/admin/openai/exchange-code         # 交换授权码
POST /api/v1/admin/openai/accounts/:id/refresh  # 刷新账户 Token

# Gemini OAuth
POST /api/v1/admin/gemini/oauth/auth-url
POST /api/v1/admin/gemini/oauth/exchange-code
GET  /api/v1/admin/gemini/oauth/capabilities

# Antigravity OAuth
POST /api/v1/admin/antigravity/oauth/auth-url
POST /api/v1/admin/antigravity/oauth/exchange-code
POST /api/v1/admin/antigravity/oauth/refresh-token
```

---

### 系统管理

```http
GET  /api/v1/admin/system/version          # 获取版本信息
GET  /api/v1/admin/system/check-updates    # 检查更新
POST /api/v1/admin/system/update           # 执行更新
POST /api/v1/admin/system/rollback         # 回滚更新
POST /api/v1/admin/system/restart          # 重启服务
```

---

## 通用接口

```http
GET /health                    # 健康检查（无需认证）
GET /setup/status              # 初始化状态（无需认证）
```
