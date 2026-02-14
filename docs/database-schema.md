# 数据库表结构

使用 Ent ORM，所有表结构定义在 `backend/ent/schema/` 目录。

## 通用 Mixin

所有表（除 `settings` 和 `usage_cleanup_tasks`）包含以下公共字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int64 | 主键（自增）|
| `created_at` | timestamptz | 创建时间 |
| `updated_at` | timestamptz | 最后更新时间 |
| `deleted_at` | timestamptz | 软删除时间（NULL 表示未删除）|

> **软删除**：所有查询默认过滤 `deleted_at IS NULL`，删除操作只设置 `deleted_at` 而不真正删除数据。

---

## 用户相关表

### users（用户）

| 字段 | 类型 | 说明 |
|------|------|------|
| `email` | varchar(255) | 邮箱（唯一，部分索引支持软删除重注册）|
| `password_hash` | varchar(255) | bcrypt 哈希密码 |
| `role` | varchar(20) | 角色：`user` / `admin` |
| `balance` | decimal(20,8) | 账户余额（USD）|
| `concurrency` | int | 最大并发请求数 |
| `status` | varchar(20) | 状态：`active` / `suspended` |
| `totp_secret_encrypted` | text | 加密的 TOTP 密钥（可选）|
| `totp_enabled` | bool | 是否启用双因素认证 |
| `totp_enabled_at` | timestamptz | 启用时间（可选）|

**关联**：
- `api_keys`：一对多，用户拥有的 API Keys
- `subscriptions`：一对多，用户的订阅
- `usage_logs`：一对多，使用记录
- `allowed_groups`：多对多（通过 user_allowed_groups），允许访问的分组
- `attribute_values`：一对多，用户自定义属性值

---

### api_keys（API 密钥）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 所属用户 ID |
| `key` | varchar(100) | API Key 值（唯一，如 `sk-xxxx`）|
| `name` | varchar(100) | Key 名称 |
| `group_id` | int64 | 关联分组 ID（决定账户池和计费倍率）|
| `status` | varchar(20) | 状态：`active` / `disabled` |
| `ip_whitelist` | JSONB | IP 白名单列表（空=允许所有）|
| `ip_blacklist` | JSONB | IP 黑名单列表 |
| `quota` | decimal(20,8) | 配额限额（0=无限）|
| `quota_used` | decimal(20,8) | 已用配额 |
| `expires_at` | timestamptz | 过期时间（可选）|

**索引**：`key`（唯一）, `user_id`, `group_id`, `status`, `deleted_at`

---

### user_subscriptions（用户订阅）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 用户 ID |
| `group_id` | int64 | 订阅的分组 ID |
| `starts_at` | timestamptz | 开始时间 |
| `expires_at` | timestamptz | 到期时间 |
| `status` | varchar(20) | 状态：`active` / `expired` / `revoked` |
| `daily_window_start` | timestamptz | 当前日窗口开始时间（可选）|
| `weekly_window_start` | timestamptz | 当前周窗口开始时间（可选）|
| `monthly_window_start` | timestamptz | 当前月窗口开始时间（可选）|
| `daily_usage_usd` | decimal(20,10) | 当日用量（USD）|
| `weekly_usage_usd` | decimal(20,10) | 当周用量（USD）|
| `monthly_usage_usd` | decimal(20,10) | 当月用量（USD）|
| `assigned_by` | int64 | 分配人用户 ID（可选）|
| `assigned_at` | timestamptz | 分配时间 |
| `notes` | text | 备注（可选）|

**约束**：同一用户在同一分组只能有一个有效订阅（软删除部分唯一索引）

---

### user_allowed_groups（用户分组权限）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 用户 ID |
| `group_id` | int64 | 分组 ID |

记录哪些用户可以使用哪些分组（用于非订阅分组的访问控制）。

---

## 账户和分组表

### groups（账户分组）

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | varchar(100) | 分组名称（唯一）|
| `platform` | varchar(50) | 平台：`claude` / `gemini` / `openai` / `antigravity` |
| `subscription_type` | varchar(50) | 订阅类型标识（自定义，用于匹配订阅）|
| `rate_multiplier` | decimal(10,4) | 计费倍率（默认 1.0）|
| `is_exclusive` | bool | 是否为专属分组（需要订阅才能使用）|
| `daily_limit_usd` | decimal(20,10) | 每日限额 USD（可选）|
| `weekly_limit_usd` | decimal(20,10) | 每周限额 USD（可选）|
| `monthly_limit_usd` | decimal(20,10) | 每月限额 USD（可选）|
| `image_price_1k` | decimal(20,10) | 图片生成 1K 费率（可选）|
| `image_price_2k` | decimal(20,10) | 图片生成 2K 费率（可选）|
| `image_price_4k` | decimal(20,10) | 图片生成 4K 费率（可选）|
| `claude_code_only` | bool | 是否仅允许 Claude Code 客户端 |
| `fallback_group_id` | int64 | 备用分组 ID（可选）|
| `model_routing_enabled` | bool | 是否启用模型路由 |
| `model_routing` | JSONB | 模型 → 账户 ID 列表的映射 |
| `supported_model_scopes` | JSONB | 支持的模型范围列表 |
| `sort_order` | int | 排序权重（越小越靠前）|

---

### accounts（AI API 账户）

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | varchar(200) | 账户名称 |
| `platform` | varchar(50) | 平台：`claude` / `gemini` / `openai` / `antigravity` |
| `type` | varchar(50) | 类型：`api_key` / `oauth` / `cookie` |
| `credentials` | JSONB | 凭证（加密存储，含 access_token、refresh_token 等）|
| `extra` | JSONB | 额外元数据（账户等级、配额信息等）|
| `concurrency` | int | 最大并发请求数（默认 1）|
| `priority` | int | 调度优先级（越高越优先）|
| `rate_multiplier` | decimal(10,4) | 账户级计费倍率 |
| `status` | varchar(20) | 状态：`active` / `disabled` / `suspended` |
| `error_message` | text | 最后的错误信息（可选）|
| `last_used_at` | timestamptz | 最后使用时间（可选）|
| `expires_at` | timestamptz | 账户到期时间（可选）|
| `proxy_id` | int64 | 关联代理 ID（可选）|
| `schedulable` | bool | 是否可被调度器选中（true=可用）|
| `rate_limited_at` | timestamptz | 最后触发 429 的时间（可选）|
| `rate_limit_reset_at` | timestamptz | 429 解除时间（可选）|
| `overload_until` | timestamptz | 529 过载截止时间（可选）|

**调度逻辑**：调度器选账户时跳过 `schedulable=false`、当前时间在 `rate_limit_reset_at` 之前、`overload_until` 之后等情况。

**关联**：
- `groups`：多对多（通过 account_groups），账户属于哪些分组
- `proxy`：多对一，使用的代理
- `usage_logs`：一对多，该账户的使用记录

---

### account_groups（账户-分组中间表）

| 字段 | 类型 | 说明 |
|------|------|------|
| `account_id` | int64 | 账户 ID |
| `group_id` | int64 | 分组 ID |

---

## 使用记录表

### usage_logs（使用日志）

**注意**：此表为追加写入（append-only），`created_at` 不可修改。使用硬删除（通过清理任务）。

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 用户 ID |
| `api_key_id` | int64 | API Key ID |
| `account_id` | int64 | 使用的账户 ID |
| `group_id` | int64 | 使用的分组 ID（可选）|
| `subscription_id` | int64 | 关联订阅 ID（可选）|
| `request_id` | varchar(100) | 请求 ID |
| `model` | varchar(100) | 模型名称（如 `claude-opus-4-5-20250929`）|
| `input_tokens` | int | 输入 token 数 |
| `output_tokens` | int | 输出 token 数 |
| `cache_creation_tokens` | int | Prompt Caching 写入 token 数 |
| `cache_read_tokens` | int | Prompt Caching 读取 token 数 |
| `input_cost` | decimal(20,10) | 输入费用（USD）|
| `output_cost` | decimal(20,10) | 输出费用（USD）|
| `cache_creation_cost` | decimal(20,10) | 缓存写入费用（USD）|
| `cache_read_cost` | decimal(20,10) | 缓存读取费用（USD）|
| `total_cost` | decimal(20,10) | 原始总费用（USD）|
| `actual_cost` | decimal(20,10) | 实际扣除金额（乘以倍率后）|
| `rate_multiplier` | decimal(10,4) | 分组倍率快照 |
| `account_rate_multiplier` | decimal(10,4) | 账户倍率快照 |
| `image_count` | int | 图片数量（图片模型）|
| `image_size` | varchar(20) | 图片尺寸（如 `1024x1024`）|
| `stream` | bool | 是否为流式请求 |
| `duration_ms` | int | 请求耗时（毫秒，可选）|
| `user_agent` | varchar(500) | 客户端 User-Agent（可选）|
| `ip_address` | varchar(50) | 客户端 IP 地址（可选）|
| `billing_type` | int8 | 计费类型标志位 |

---

### usage_cleanup_tasks（使用记录清理任务）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 目标用户 ID（可选，NULL=全部用户）|
| `status` | varchar(20) | 状态：`pending` / `running` / `completed` / `failed` / `cancelled` |
| `start_date` | date | 清理起始日期 |
| `end_date` | date | 清理截止日期 |
| `deleted_count` | int64 | 已删除记录数 |
| `error_message` | text | 错误信息（可选）|
| `started_at` | timestamptz | 开始执行时间（可选）|
| `completed_at` | timestamptz | 完成时间（可选）|

---

## 系统配置表

### settings（系统设置）

使用硬删除（无软删除），简单键值对存储。

| 字段 | 类型 | 说明 |
|------|------|------|
| `key` | varchar(100) | 设置键（唯一）|
| `value` | text | 设置值（JSON 字符串）|
| `updated_at` | timestamptz | 最后更新时间 |

---

### proxies（HTTP 代理）

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | varchar(100) | 代理名称 |
| `protocol` | varchar(20) | 协议：`http` / `https` / `socks5` |
| `host` | varchar(255) | 代理主机 |
| `port` | int | 端口 |
| `username` | varchar(100) | 认证用户名（可选）|
| `password` | varchar(100) | 认证密码（可选）|
| `status` | varchar(20) | 状态：`active` / `disabled` |

---

### error_passthrough_rules（错误透传规则）

| 字段 | 类型 | 说明 |
|------|------|------|
| `platform` | varchar(50) | 平台：`claude` / `gemini` / `openai` |
| `status_code` | int | HTTP 状态码 |
| `match_pattern` | text | 匹配规则（正则，可选）|
| `action` | varchar(20) | 处理方式：`passthrough` / `block` |
| `description` | text | 规则描述 |

---

## 公告相关表

### announcements（公告）

| 字段 | 类型 | 说明 |
|------|------|------|
| `title` | varchar(200) | 公告标题 |
| `content` | text | 公告内容（支持 Markdown）|
| `status` | varchar(20) | 状态：`draft` / `published` / `archived` |
| `target_users` | JSONB | 定向条件（订阅、余额等，null=所有人）|
| `published_at` | timestamptz | 发布时间（可选）|

---

### announcement_reads（公告阅读记录）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 用户 ID |
| `announcement_id` | int64 | 公告 ID |
| `read_at` | timestamptz | 阅读时间 |

---

## 促销码相关表

### promo_codes（促销码）

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | varchar(50) | 促销码（唯一）|
| `discount_percent` | int | 折扣百分比（0-100）|
| `max_uses` | int | 最大使用次数（0=无限）|
| `uses` | int | 已使用次数 |
| `status` | varchar(20) | 状态：`active` / `disabled` |
| `expires_at` | timestamptz | 过期时间（可选）|

---

### promo_code_usages（促销码使用记录）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 使用用户 ID |
| `promo_code_id` | int64 | 促销码 ID |
| `used_at` | timestamptz | 使用时间 |

---

## 兑换码表

### redeem_codes（兑换码）

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | varchar(50) | 兑换码（唯一）|
| `type` | varchar(20) | 类型：`balance` / `invitation` / `concurrency` |
| `balance` | decimal(20,8) | 兑换余额（USD，`balance` 类型）|
| `concurrency` | int | 并发数（`concurrency` 类型）|
| `validity_days` | int | 有效期（天，0=永久）|
| `status` | varchar(20) | 状态：`unused` / `used` / `expired` |
| `redeemed_by` | int64 | 兑换人用户 ID（可选）|
| `redeemed_at` | timestamptz | 兑换时间（可选）|
| `notes` | text | 备注（可选）|

---

## 用户属性扩展表

### user_attribute_definitions（用户属性定义）

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | varchar(100) | 属性名称（如"公司名"、"手机号"）|
| `description` | text | 属性描述（可选）|
| `type` | varchar(20) | 类型：`string` / `number` / `boolean` / `select` |
| `options` | JSONB | 选项列表（`select` 类型用）|
| `sort_order` | int | 排序权重 |
| `required` | bool | 是否必填 |

---

### user_attribute_values（用户属性值）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | int64 | 用户 ID |
| `definition_id` | int64 | 属性定义 ID |
| `value` | JSONB | 属性值（支持任意类型）|

---

## 实体关系图（简化）

```
User
├── APIKey (1:N)
│   └── Group (N:1) ───────── Account (N:M via account_groups)
│                            │
│                            └── Proxy (N:1)
├── UserSubscription (1:N)
│   └── Group (N:1)
├── UsageLog (1:N)
│   ├── APIKey (N:1)
│   ├── Account (N:1)
│   ├── Group (N:1)
│   └── UserSubscription (N:1)
└── UserAllowedGroup (1:N)
    └── Group (N:1)
```

---

## 数据库迁移

迁移文件位于 `backend/migrations/`，按版本号顺序执行：

```bash
# 自动执行迁移（应用启动时）
# 无需手动操作，代码自动处理

# 手动执行（开发时）
cd backend && go run ./cmd/server --migrate
```

**特殊迁移说明**：
- `016_soft_delete_partial_unique_indexes.sql`：通过 `WHERE deleted_at IS NULL` 部分索引实现软删除后可重复注册/订阅
