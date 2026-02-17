# 供应商采购与二次转售系统设计

> 日期: 2026-02-18
> 状态: 待实现
> 分支: feature/vendor-resale

## 1. 业务背景

从咸鱼/淘宝/拼多多等平台采购低价 API 转发服务（如 ai.9w7.cn 类站点），整合到 Sub2API 平台统一出售。自有高成本账户降级为兜底/VIP 专用。

## 2. 核心设计

### 2.1 数据模型

#### Vendor（供应商）实体 — 新增

```
Vendor {
  id                    int64       PK
  name                  string(100) NOT NULL     -- 供应商名称
  description           text        NULLABLE     -- 备注（采购渠道、联系方式等）

  // API 配置
  api_format            string(20)  NOT NULL     -- "anthropic" | "openai"
  base_url              string(500) NOT NULL     -- 供应商 API 地址 (如 https://ai.9w7.cn)
  auth_type             string(20)  NOT NULL     -- "api_key" | "session" | "bearer"
  api_path_override     string(500) NULLABLE     -- 自定义 API 路径覆盖（默认用标准路径）
  extra_headers         jsonb       DEFAULT {}   -- 额外请求头

  // 计费信息
  billing_type          string(20)  NOT NULL     -- "token" | "quota" | "subscription"
  cost_per_1k_input     decimal     NULLABLE     -- 采购成本：每千输入 token（USD）
  cost_per_1k_output    decimal     NULLABLE     -- 采购成本：每千输出 token（USD）
  total_quota_usd       decimal     NULLABLE     -- 总额度（固定额度包模式）
  used_quota_usd        decimal     DEFAULT 0    -- 已用额度
  balance_usd           decimal     NULLABLE     -- 当前余额（通过 API 查询或手动更新）
  expires_at            timestamptz NULLABLE     -- 到期时间（订阅模式）

  // 健康监控
  status                string(20)  DEFAULT 'active'  -- active | suspended | depleted | error
  health_check_enabled  bool        DEFAULT false     -- 是否启用自动健康检查
  health_check_interval int         DEFAULT 300       -- 健康检查间隔（秒）
  health_check_model    string(50)  DEFAULT 'claude-sonnet-4-20250514' -- 健康检查用的模型
  last_health_check_at  timestamptz NULLABLE
  last_health_status    string(20)  NULLABLE          -- ok | slow | error | timeout
  last_health_latency   int         NULLABLE          -- 最近一次健康检查延迟（ms）
  error_message         text        NULLABLE
  consecutive_failures  int         DEFAULT 0         -- 连续失败次数

  // 自动采购（Phase 3）
  auto_purchase_enabled bool        DEFAULT false
  auto_purchase_config  jsonb       DEFAULT {}   -- 自动采购配置

  // 余额预警
  balance_alert_enabled   bool      DEFAULT false
  balance_alert_threshold decimal   NULLABLE     -- 余额低于此值时告警（USD）

  // 时间戳
  created_at            timestamptz
  updated_at            timestamptz
  deleted_at            timestamptz NULLABLE
}
```

#### Account 扩展

```
Account {
  ... 现有字段不变 ...
  + vendor_id     int64   NULLABLE FK → Vendor   -- NULL = 自有账户
  + source_type   string  DEFAULT 'owned'        -- 'owned' | 'vendor'
}
```

### 2.2 调度策略

```
用户请求 → APIKey → Group A（供应商池，默认主力）
                        ↓ 全部 429/不可用
                    Group B（自有账户池，fallback_group_id）
```

- 供应商账户放在独立 Group，设为用户默认分组
- 自有账户 Group 通过现有 `fallback_group_id` 机制兜底
- 供应商账户 `priority` 按成本排序：低成本=10，高成本=30
- VIP 用户/活动可直接分配自有账户 Group

### 2.3 协议转换

对于 `api_format=openai` 的供应商，在 GatewayService 转发时做双向转换：

```
用户请求 (Anthropic 格式)
  ↓ convertAnthropicToOpenAI()
供应商 (OpenAI 格式)
  ↓ convertOpenAIToAnthropic()
用户响应 (Anthropic 格式)
```

转换范围：
- 请求体：messages 格式、model 名称映射、system prompt 处理
- 响应体：choice → content block、usage 字段映射
- SSE 流：data 格式转换、stop_reason 映射
- 错误码：HTTP status + error body 映射

### 2.4 网关转发流程变更

```go
// gateway_service.go 中的转发逻辑
func (s *GatewayService) forwardRequest(ctx, account, req) {
    targetURL := claudeAPIURL  // 默认 Anthropic 官方

    if account.VendorID != nil {
        vendor := s.vendorRepo.Get(account.VendorID)
        targetURL = vendor.BaseURL + vendor.APIPath()

        // 设置认证头
        switch vendor.AuthType {
        case "api_key":
            req.Header.Set("x-api-key", account.Credentials["api_key"])
        case "bearer":
            req.Header.Set("Authorization", "Bearer "+account.Credentials["api_key"])
        case "session":
            req.Header.Set("Cookie", "session="+account.Credentials["session_key"])
        }

        // 添加额外请求头
        for k, v := range vendor.ExtraHeaders {
            req.Header.Set(k, v)
        }

        // 协议转换
        if vendor.APIFormat == "openai" {
            req, responseConverter = s.vendorAdapter.ConvertRequest(req)
            defer responseConverter.ConvertResponse(resp)
        }
    }
}
```

## 3. 功能模块

### 3.1 Phase 1 — 手动采购 + 系统管理

| 模块 | 说明 |
|------|------|
| Vendor CRUD API | 供应商增删改查 |
| Account-Vendor 关联 | Account 新增 vendor_id，创建时可选关联供应商 |
| 协议转换层 | Anthropic ↔ OpenAI 双向转换 |
| 调度集成 | 供应商账户参与正常调度，自有账户 fallback |
| Admin UI — 供应商管理 | 供应商列表、创建、编辑、删除 |
| Admin UI — Account 筛选 | 按 source_type 筛选自有/采购账户 |
| 供应商成本统计 | 按供应商统计用量和成本 |

### 3.2 Phase 2 — 半自动化监控

| 模块 | 说明 |
|------|------|
| 健康检查服务 | 定时向供应商发送测试请求，检测可用性和延迟 |
| 余额检测 | 通过供应商 API 或页面抓取检测余额 |
| 余额预警 | 余额低于阈值时通知管理员 |
| 自动暂停 | 供应商余额耗尽或到期时自动暂停其下所有账户 |
| 成本分析仪表盘 | 供应商成本 vs 售价利润率分析 |
| 自动定价建议 | 根据供应商成本自动计算建议售价 |

### 3.3 Phase 3 — 全自动化

| 模块 | 说明 |
|------|------|
| 供应商发现爬虫 | AI 驱动的供应商发现（咸鱼/淘宝搜索） |
| 自动注册 | 自动在供应商平台注册账号 |
| 自动充值 | 对接支付接口自动充值 |
| 自动上架 | 获取 API Key 后自动创建 Account + 加入 Group |
| 动态定价引擎 | 根据供需、成本、竞品价格动态调整售价 |
| 供应商评分系统 | 根据可用性、延迟、性价比自动评分排序 |

## 4. API 设计

### 4.1 供应商管理 API（Admin）

```
POST   /api/admin/vendors              — 创建供应商
GET    /api/admin/vendors              — 供应商列表
GET    /api/admin/vendors/:id          — 供应商详情
PUT    /api/admin/vendors/:id          — 更新供应商
DELETE /api/admin/vendors/:id          — 删除供应商（软删除）
POST   /api/admin/vendors/:id/test     — 测试供应商连通性
POST   /api/admin/vendors/:id/refresh-balance — 刷新供应商余额
GET    /api/admin/vendors/:id/stats    — 供应商用量统计
GET    /api/admin/vendors/dashboard    — 供应商总览仪表盘
```

### 4.2 Account 扩展 API

```
POST   /api/admin/accounts             — 创建账户（新增 vendor_id 字段）
GET    /api/admin/accounts?source_type=vendor  — 按来源筛选
POST   /api/admin/vendors/:id/accounts — 批量为供应商创建账户
```

### 4.3 健康检查 API

```
POST   /api/admin/vendors/:id/health-check  — 手动触发健康检查
GET    /api/admin/vendors/:id/health-history — 健康检查历史
PUT    /api/admin/vendors/:id/health-config  — 更新健康检查配置
```

### 4.4 自动采购 API（Phase 3）

```
PUT    /api/admin/vendors/:id/auto-purchase  — 配置自动采购
GET    /api/admin/vendors/discovery           — 供应商发现结果
POST   /api/admin/vendors/discovery/scan      — 触发供应商扫描
```

## 5. 后台 UI 设计

### 5.1 供应商管理页面

- 供应商列表（表格：名称、API格式、计费模式、余额、状态、健康、操作）
- 创建/编辑供应商表单
- 供应商详情页（基本信息 + 关联账户列表 + 用量统计图表）
- 连通性测试按钮（实时测试 API 可用性和延迟）

### 5.2 供应商仪表盘

- 总成本 vs 总收入利润率
- 各供应商健康状态概览
- 余额预警列表
- 即将到期供应商列表

### 5.3 Account 列表增强

- 新增 "来源" 列（自有/供应商名称）
- 新增 source_type 筛选器
- 供应商账户显示供应商名称和余额信息

## 6. 配置开关

所有新功能通过 Setting 表控制，默认关闭：

```
vendor_system_enabled: false          -- 供应商系统总开关
vendor_health_check_enabled: false    -- 全局健康检查开关
vendor_balance_alert_enabled: false   -- 余额预警开关
vendor_auto_purchase_enabled: false   -- 自动采购开关（Phase 3）
vendor_auto_pricing_enabled: false    -- 自动定价开关
```

## 7. 数据库迁移

需要新增的迁移：
1. `xxx_create_vendors_table.sql` — 创建 vendors 表
2. `xxx_add_vendor_id_to_accounts.sql` — accounts 表新增 vendor_id + source_type
3. `xxx_add_vendor_settings.sql` — settings 表新增供应商相关配置项

## 8. 对现有系统的影响

### 零影响保证
- Account 新增字段均为 NULLABLE 或有 DEFAULT，不影响现有数据
- 调度器逻辑不变，只是 Account 多了 vendor 维度信息
- 现有 API 接口完全兼容，新字段为可选
- 功能默认关闭，不影响现有业务流程

### 需要修改的现有文件
- `ent/schema/account.go` — 新增 vendor_id, source_type 字段和 vendor edge
- `gateway_service.go` — 转发时检查 vendor，替换 base_url 和认证头
- `openai_gateway_service.go` — 复用协议转换逻辑
- `handler/admin_account_handler.go` — Account CRUD 支持 vendor_id
- 前端 Account 管理页面 — 新增供应商筛选和关联

### 新增文件
- `ent/schema/vendor.go` — Vendor 实体定义
- `internal/service/vendor_service.go` — 供应商业务逻辑
- `internal/service/vendor_health_service.go` — 健康检查服务
- `internal/service/vendor_adapter.go` — 协议转换适配器
- `internal/handler/admin_vendor_handler.go` — 供应商 Admin API
- `internal/repository/vendor_repo.go` — 供应商数据访问
- 前端供应商管理页面组件
