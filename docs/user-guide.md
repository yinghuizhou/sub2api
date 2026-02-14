# 用户使用指南

本指南面向使用 Sub2API 平台的普通用户（非管理员），介绍如何管理 API Key、查看使用记录、管理订阅等功能。

## 注册与登录

### 注册账户

1. 访问平台注册页面 `/register`
2. 填写邮箱和密码
3. 如果系统开启了邮箱验证，需要先获取验证码：
   - 点击"发送验证码"按钮
   - 查收邮件中的 6 位验证码
   - 填入验证码
4. 如有促销码或邀请码，可在注册时填写以获取优惠
5. 点击注册完成

### 登录

1. 访问 `/login`
2. 填写邮箱和密码
3. 如果账户开启了双因素认证（TOTP），还需输入验证器 App 中的 6 位代码

### 忘记密码

1. 访问 `/forgot-password`
2. 输入注册邮箱
3. 查收重置密码邮件（有效期通常为 1 小时）
4. 点击邮件中的链接设置新密码

---

## 仪表板（Dashboard）

登录后首先看到的是个人仪表板 `/dashboard`，显示：

- **使用量统计**：今日/本周/本月的 Token 使用量和费用
- **趋势图表**：最近 7 天的使用趋势
- **模型分布**：按模型分类的使用情况
- **快捷操作**：快速访问常用功能

---

## API Key 管理

### 什么是 API Key？

API Key（格式：`sk-xxxxxx`）是您调用 AI API 的凭证。您可以创建多个 Key，用于不同的项目或场景。

### 创建 API Key

1. 进入 `/keys`（"我的密钥"页面）
2. 点击"创建 API Key"按钮
3. 填写配置：

| 字段 | 说明 |
|------|------|
| **名称** | 便于识别的名称（如"生产环境"、"测试项目"）|
| **分组** | 选择使用哪个账户池（影响可用模型和费率）|
| **配额** | 使用限额（USD），0 表示无限制 |
| **过期时间** | 设置天数后自动过期（可选）|
| **IP 白名单** | 只允许指定 IP 使用此 Key（格式：CIDR，如 `192.168.1.0/24`）|
| **IP 黑名单** | 禁止指定 IP 使用此 Key |

4. 创建成功后，点击复制图标保存 Key（**Key 只会完整显示一次，请妥善保存**）

### 使用 API Key 调用 AI API

以 Claude API 为例：

```bash
# 方式一：x-api-key 头
curl https://your-domain.com/v1/messages \
  -H "x-api-key: sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4-5-20250929",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# 方式二：Authorization 头
curl https://your-domain.com/v1/messages \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-opus-4-5-20250929", "max_tokens": 1024, "messages": [{"role": "user", "content": "Hello!"}]}'
```

### 在 Claude Code 中使用

```bash
# 配置环境变量
export ANTHROPIC_BASE_URL="https://your-domain.com"
export ANTHROPIC_API_KEY="sk-xxxxxxxx"

# 启动 Claude Code
claude
```

### 管理已有 Key

在 `/keys` 页面可以：
- **编辑**：修改名称、配额、IP 限制、过期时间
- **禁用/启用**：临时停用某个 Key 而不删除
- **删除**：永久删除 Key（不可恢复）
- **查看使用量**：查看该 Key 的配额使用情况

---

## 使用记录

进入 `/usage` 查看详细的 API 调用记录。

### 查询和筛选

| 筛选项 | 说明 |
|--------|------|
| 日期范围 | 按时间范围筛选（支持快捷选择：今天/本周/本月）|
| 模型 | 按 AI 模型筛选（如 `claude-opus-4-5-20250929`）|
| API Key | 按使用的 Key 筛选 |

### 记录字段说明

| 字段 | 说明 |
|------|------|
| 请求时间 | 发起请求的时间 |
| 模型 | 使用的 AI 模型 |
| 输入 Token | 发送给 AI 的 Token 数量 |
| 输出 Token | AI 回复的 Token 数量 |
| 缓存 Token | Prompt Caching 相关的 Token |
| 费用 | 本次请求扣除的金额（USD）|
| 耗时 | 请求响应时间（毫秒）|

### 导出记录

点击"导出"按钮，可以将使用记录导出为 Excel 文件（`.xlsx`），方便对账和分析。

---

## 订阅管理

### 查看我的订阅

进入 `/subscriptions` 查看当前的订阅情况：

- **订阅状态**：有效期、剩余时间
- **用量进度**：日/周/月用量进度条
- **分组信息**：订阅对应的分组（决定可用账户池）

### 用量限制说明

订阅可能有以下用量限制（由管理员配置）：
- **每日限额**：每天 00:00 重置
- **每周限额**：每周一 00:00 重置
- **每月限额**：每月 1 日 00:00 重置

超过限额后，当前窗口期内的请求将被拒绝，等到下一个窗口期重置后恢复。

---

## 兑换码

兑换码可以充值余额、激活订阅或提升并发数。

### 兑换步骤

1. 进入 `/redeem`
2. 输入兑换码
3. 点击"立即兑换"
4. 成功后会提示兑换内容（余额/订阅/并发）

### 查看兑换历史

在同一页面下方可以查看历史兑换记录。

---

## 个人资料

进入 `/profile` 管理个人信息：

### 修改资料

- 修改显示名称等基本信息

### 修改密码

1. 输入当前密码
2. 输入新密码（两次确认）
3. 点击保存

### 启用双因素认证（TOTP）

双因素认证大幅提升账户安全性，强烈建议开启。

**启用步骤**：

1. 在"安全设置"区域点击"启用双因素认证"
2. 使用 Google Authenticator、Authy 或其他支持 TOTP 的 App 扫描二维码
3. 输入 App 显示的 6 位验证码确认
4. **重要**：保存备用恢复码（用于手机丢失时恢复）

**禁用 TOTP**（需要输入当前 TOTP 验证码）：
1. 点击"禁用双因素认证"
2. 输入 App 中的 6 位验证码
3. 确认禁用

---

## 账户余额

余额以 USD 计，用于支付 API 调用费用（前提是系统使用余额扣费模式）。

### 查看余额

在仪表板或个人资料页可以看到当前余额。

### 充值

- 通过兑换码兑换余额
- 或联系管理员直接充值

---

## 常见问题

### Q: 请求返回 401 Unauthorized？

可能原因：
- API Key 不存在或已被删除
- API Key 已过期（检查过期时间）
- API Key 已被禁用

### Q: 请求返回 403 Forbidden？

可能原因：
- IP 地址不在白名单中
- IP 地址在黑名单中
- 账户余额不足（如启用了余额限制）
- 配额已用完（`quota_used >= quota`）

### Q: 请求返回 429 Too Many Requests？

表示当前请求过于频繁，或上游 AI 服务资源紧张。稍等片刻后重试。

### Q: 请求超时？

AI 生成大量内容时可能需要较长时间。对于流式请求（`"stream": true`），可以设置更长的超时时间（建议 300 秒以上）。

### Q: 流式响应中断？

在 Nginx 等反向代理中，需要配置：
- `proxy_read_timeout 300s`
- `proxy_buffering off`

### Q: 如何查看我的 API Key 使用了哪个 AI 账户？

使用记录中包含了具体的账户信息。如果需要固定使用某个账户（粘性会话），在请求中加入 `conversation_id` 字段，相同 ID 的请求会尽量路由到同一个账户。

### Q: 如何使用 Gemini API？

```bash
# Gemini API 调用示例
curl "https://your-domain.com/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "x-api-key: sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "parts": [{"text": "Hello!"}]
    }]
  }'
```
