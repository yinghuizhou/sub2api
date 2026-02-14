# 系统架构总览

## 项目简介

Sub2API 是一个 AI API 网关平台，将 Claude、Gemini、OpenAI、Antigravity 等 AI 服务的订阅账户统一管理，通过自定义 API Key 对外提供服务，支持认证、计费、负载均衡、订阅管理等完整功能。

## 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端请求                            │
│         (API Key 认证 or 用户 JWT 认证)                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                    HTTP 服务器 (Gin)                          │
│  ┌──────────────┐  ┌────────────────┐  ┌──────────────────┐ │
│  │   中间件栈    │  │   路由注册      │  │  前端静态服务     │ │
│  │ Logger/CORS  │  │ /api/v1/...    │  │ Vue3 SPA 嵌入    │ │
│  │ SecurityHdr  │  │ /v1/messages   │  │                  │ │
│  │ RateLimit    │  │ /v1beta/...    │  │                  │ │
│  └──────────────┘  └────────────────┘  └──────────────────┘ │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                      Handler 层                              │
│  GatewayHandler │ AuthHandler │ UserHandler │ AdminHandlers  │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                      Service 层                              │
│  GatewayService │ AuthService │ BillingService │ OpsService  │
│  APIKeyService  │ UserService │ GroupService   │ ...         │
└──────────┬──────────────────────────────┬────────────────────┘
           │                              │
┌──────────▼──────────┐      ┌────────────▼────────────────────┐
│   Repository 层      │      │        外部 AI API              │
│  Ent ORM + 缓存     │      │  Claude / Gemini / OpenAI /     │
│  (PostgreSQL+Redis) │      │  Antigravity                    │
└─────────────────────┘      └─────────────────────────────────┘
```

## 技术栈

### 后端

| 组件 | 技术 | 版本 | 用途 |
|------|------|------|------|
| 语言 | Go | 1.25.7 | 主要业务逻辑 |
| Web 框架 | Gin | latest | HTTP 路由和中间件 |
| ORM | Ent | latest | 数据库访问和 schema 管理 |
| 依赖注入 | Google Wire | latest | 编译时依赖注入 |
| 缓存 L1 | Ristretto | latest | 内存缓存（API Key 认证、调度器快照）|
| 缓存 L2 | Redis | 7+ | 分布式缓存、速率限制 |
| 数据库 | PostgreSQL | 15+ | 主数据存储 |

### 前端

| 组件 | 技术 | 版本 | 用途 |
|------|------|------|------|
| 框架 | Vue 3 | 3.4.0 | UI 框架（Composition API）|
| 路由 | Vue Router | 4.2.5 | SPA 路由管理 |
| 状态管理 | Pinia | 2.1.7 | 全局状态 |
| HTTP 客户端 | Axios | 1.13.5 | API 请求（含拦截器）|
| 样式系统 | TailwindCSS | 3.4.0 | 原子化 CSS |
| 国际化 | Vue i18n | 9.14.5 | 英文/中文 |
| 图表 | Chart.js + vue-chartjs | 4.4.1 | 数据可视化 |
| 构建工具 | Vite | 5.0.10 | 开发服务器和生产构建 |
| 测试框架 | Vitest | 2.1.9 | 单元/集成测试 |

## 目录结构

```
sub2api/
├── backend/                    # Go 后端
│   ├── cmd/
│   │   ├── server/             # 服务入口（main.go, wire.go, wire_gen.go）
│   │   └── jwtgen/             # JWT 生成工具
│   ├── ent/
│   │   └── schema/             # 数据库 schema 定义（21 个实体）
│   ├── internal/
│   │   ├── config/             # 配置加载
│   │   ├── handler/            # HTTP 请求处理层
│   │   │   ├── admin/          # 管理后台 handlers
│   │   │   └── dto/            # 数据传输对象
│   │   ├── middleware/         # 认证中间件（JWT/APIKey/Admin）
│   │   ├── repository/         # 数据访问层（Ent + Redis）
│   │   ├── service/            # 业务逻辑层
│   │   ├── server/
│   │   │   ├── middleware/     # 服务器级中间件（CORS/Logger/Rate Limit）
│   │   │   └── routes/         # 路由注册
│   │   ├── pkg/                # 通用工具包
│   │   │   ├── claude/         # Claude API 协议处理
│   │   │   ├── gemini/         # Gemini API 协议处理
│   │   │   ├── openai/         # OpenAI API 协议处理
│   │   │   └── antigravity/    # Antigravity API 协议处理
│   │   └── web/                # 嵌入式前端服务
│   └── migrations/             # 数据库迁移文件
├── frontend/                   # Vue 3 前端
│   └── src/
│       ├── api/                # API 调用层（18 个模块）
│       ├── components/         # Vue 组件（150+）
│       ├── composables/        # 可复用逻辑（11 个）
│       ├── stores/             # Pinia 状态（5 个 store）
│       ├── views/              # 页面组件（25 个页面）
│       ├── router/             # 路由配置（34 条路由）
│       ├── i18n/               # 国际化（英文/中文）
│       └── types/              # TypeScript 类型（1300+ 行）
└── docs/                       # 项目文档（本目录）
```

## 核心模块说明

### API 网关核心（GatewayService）

这是系统最核心的模块，负责：

1. **智能账户调度**：基于负载感知的账户选择，支持：
   - 优先级加权选择
   - 并发限制（账户级/用户级）
   - 429/529 自动故障转移
   - 粘性会话（通过 conversation_id 路由）
   - 模型路由（指定账户处理特定模型）

2. **多平台请求转发**：
   - Claude `/v1/messages`, `/v1/messages/count_tokens`
   - Gemini `/v1beta/models/*`
   - OpenAI `/v1/chat/completions` (兼容接口)
   - Antigravity（专属路由）

3. **流式 SSE 转发**：服务器发送事件的实时传输

4. **Token 级计费**：精确计算每次请求的费用，支持：
   - 输入/输出 token 计费
   - Prompt Caching（缓存创建/缓存命中不同费率）
   - 图片生成计费
   - 分组/账户级倍率乘数
   - 订阅用量追踪（日/周/月）

### 认证体系

三种认证方式：

| 方式 | 适用场景 | 说明 |
|------|---------|------|
| JWT Token | 用户界面操作 | Access Token（短期）+ Refresh Token（长期），支持 TOTP 2FA |
| API Key | 调用 AI API | 由用户创建，支持 IP 白/黑名单、配额限制、过期时间 |
| Admin Token | 管理后台操作 | 检查用户角色是否为 admin |

### 两层缓存架构

```
请求
  │
  ▼
L1 缓存（Ristretto 内存缓存）
  │  命中率高，访问极快
  │  TTL: 2-5 分钟
  │  失效：主动失效 + 过期
  │
  ▼（L1 Miss）
L2 缓存（Redis）
  │  分布式，持久
  │  TTL: 5-30 分钟
  │
  ▼（L2 Miss）
数据库（PostgreSQL）
```

缓存内容包括：
- API Key 认证结果（避免每次查库）
- 调度器快照（账户状态和负载）
- 计费配置（模型价格表）
- Dashboard 聚合数据

### 后台服务

系统启动时自动启动以下后台服务：

| 服务 | 功能 | 周期 |
|------|------|------|
| TokenRefreshService | 刷新所有 OAuth Token | 定期 |
| AccountExpiryService | 处理过期账户的暂停 | 定期 |
| SchedulerSnapshotService | 更新调度器账户快照 | 定期 |
| OpsMetricsCollector | 收集实时运维指标 | 实时 |
| OpsAggregationService | 聚合指标数据（滑动窗口）| 定期 |
| OpsAlertEvaluatorService | 评估告警规则触发条件 | 每分钟 |
| PricingService | 同步最新模型价格表 | 定期 |
| EmailQueueService | 处理邮件发送队列 | 实时 |
| UsageCleanupService | 执行使用记录清理任务 | 定期 |

### 依赖注入（Google Wire）

使用 Google Wire 编译时生成依赖注入代码：

```
config → database/redis
  → repository（数据访问层）
    → service（业务逻辑层）
      → middleware（认证中间件）
        → handler（HTTP 处理层）
          → server（HTTP 服务器）
```

修改 `wire.go` 后需运行：
```bash
cd backend && go generate ./cmd/server
```

## 运行模式

系统支持两种运行模式，通过配置 `run_mode` 设置：

| 模式 | 说明 | 适用场景 |
|------|------|---------|
| `standard` | 完整功能模式 | 默认，支持所有功能 |
| `simple` | 简化模式 | 限制部分前端路由访问 |

## 数据流示例：一次 AI API 请求

```
客户端 (API Key: sk-xxx)
    │
    ▼
APIKeyAuthMiddleware
    ├─ L1/L2 缓存查找 API Key
    ├─ 验证 IP 白/黑名单
    ├─ 检查 Key 状态、过期、配额
    └─ 写入 Context（user_id, group_id 等）
    │
    ▼
GatewayHandler.Messages()
    ├─ 解析请求体（model, messages, stream 等）
    ├─ 检查用户并发限制
    └─ 调用 GatewayService
    │
    ▼
GatewayService.SelectAccount()（账户调度）
    ├─ 检查粘性会话（conversation_id → 固定账户）
    ├─ 过滤不可用账户（rate_limited, overloaded, schedulable=false）
    ├─ 按优先级 + 负载选择账户
    └─ 返回 AccountSelectionResult
    │
    ▼
GatewayService.Forward()（请求转发）
    ├─ 获取账户 Access Token（OAuth 流程或直接使用 API Key）
    ├─ 构建上游请求（注入认证头）
    ├─ 转发到对应平台的 API
    ├─ 流式：SSE 事件转发
    └─ 非流式：等待完整响应
    │
    ▼
BillingService（计费）
    ├─ 统计 token 使用量（input/output/cache）
    ├─ 计算费用（价格 × 倍率）
    └─ 更新用户余额、API Key 配额、订阅用量
    │
    ▼
OpsService（记录）
    └─ 写入 UsageLog（异步）
```
