# 商业化功能审查报告与使用指南

> 分支: `feature/commercialization` | 审查日期: 2026-02-27
>
> 本文档涵盖商业化模块的功能概览、使用说明、配置指南、上线评估及待修复项。

---

## 目录

1. [功能概览](#1-功能概览)
2. [上线评估总结](#2-上线评估总结)
3. [支付系统](#3-支付系统)
4. [推荐返佣系统](#4-推荐返佣系统)
5. [免费试用](#5-免费试用)
6. [分销商系统](#6-分销商系统)
7. [管理后台操作指南](#7-管理后台操作指南)
8. [前端页面](#8-前端页面)
9. [配置项说明](#9-配置项说明)
10. [数据库迁移](#10-数据库迁移)
11. [Critical 问题清单](#11-critical-问题清单)
12. [Major 问题清单](#12-major-问题清单)
13. [Minor 问题与后续优化](#13-minor-问题与后续优化)

---

## 1. 功能概览

| 模块 | 状态 | 上线就绪 | 说明 |
|------|------|----------|------|
| 支付系统 | 骨架完成 | ❌ | 回调未实现、缺事务保护 |
| 推荐返佣 | 逻辑完整 | ⚠️ | 依赖支付回调触发佣金结算 |
| 免费试用 | 可上线 | ✅ | 注册时自动发放，开关+金额可配 |
| 分销商 | 半成品 | ❌ | 仅 service 层骨架，无 handler |
| 前端-充值页 | UI 完成 | ⚠️ | 缺错误反馈、余额刷新 |
| 前端-推荐页 | UI 完成 | ⚠️ | 后端 API 未暴露用户端接口 |

---

## 2. 上线评估总结

**结论：当前代码不适合直接上线。**

共发现 6 个 Critical、10 个 Major、5 个 Minor 问题。

### 建议上线路径

**Phase 1 — 最小可上线版本（仅免费试用）：**
- 免费试用功能已就绪，可直接上线
- 隐藏前端充值/推荐入口（已有 simple mode 支持）

**Phase 2 — 支付上线（需修复 Critical）：**
1. 实现微信/支付宝签名验证（C1）
2. 添加事务保护（C2, C3）
3. 修复 IDOR 越权漏洞（C5）
4. 修复状态更新竞态（C6）
5. 联通支付回调 → 佣金结算（C4）

**Phase 3 — 完善分销系统：**
- 补充分销商 HTTP handler
- 实现批量购码流程

---

## 3. 支付系统

### 3.1 架构

```
用户 → PaymentHandler → PaymentService → PaymentRepository → Ent ORM
                              ↓
                     gopay SDK (微信/支付宝)
```

### 3.2 已实现功能

| 功能 | 文件 | 状态 |
|------|------|------|
| 创建支付订单 | `service/payment_service.go:CreateOrder` | ✅ 逻辑完成 |
| 订单号生成 | `service/payment_service.go:generateOrderNo` | ⚠️ 碰撞风险 |
| 汇率换算 (CNY→USD) | `service/payment_service.go:CreateOrder` | ⚠️ 硬编码 7.2 |
| 奖励计算 | `service/payment_service.go:calculateBonus` | ✅ |
| 回调处理（幂等） | `service/payment_service.go:HandleCallback` | ⚠️ 缺事务 |
| 微信回调 | `handler/payment_handler.go:WechatCallback` | ❌ TODO |
| 支付宝回调 | `handler/payment_handler.go:AlipayCallback` | ❌ TODO |
| 查询单个订单 | `handler/payment_handler.go:GetOrder` | ⚠️ 缺鉴权 |
| 查询用户订单列表 | `handler/payment_handler.go:ListOrders` | ✅ |
| 管理后台-订单列表 | `handler/admin/payment_handler.go:ListOrders` | ❌ 返回空数组 |

### 3.3 API 接口

**用户端（需 JWT 认证）：**

```
POST   /api/v1/payment/create     # 创建支付订单
GET    /api/v1/payment/orders      # 查询订单列表
GET    /api/v1/payment/orders/:id  # 查询订单详情
```

**回调（无认证，需签名验证）：**

```
POST   /api/v1/payment/callback/wechat  # 微信支付回调
POST   /api/v1/payment/callback/alipay  # 支付宝支付回调
```

**管理后台（需 x-api-key）：**

```
GET    /api/v1/admin/payment/packages      # 充值套餐列表
POST   /api/v1/admin/payment/packages      # 创建套餐
PUT    /api/v1/admin/payment/packages/:id  # 更新套餐
DELETE /api/v1/admin/payment/packages/:id  # 删除套餐
GET    /api/v1/admin/payment/orders        # 订单列表（未实现）
```

### 3.4 创建订单请求/响应

```json
// POST /api/v1/payment/create
// Request
{
  "amount_cny": 100.00,
  "payment_method": "wechat"  // "wechat" | "alipay"
}

// Response
{
  "code": 0,
  "data": {
    "order_no": "PAY20260227064200123456",
    "amount_cny": 100.00,
    "amount_usd": 13.89,
    "bonus": 1.39,
    "total_credit": 15.28,
    "status": "pending",
    "payment_method": "wechat"
  }
}
```

### 3.5 支付流程图

```
用户选择套餐/输入金额
        ↓
POST /payment/create
        ↓
生成订单号 → 计算 USD + bonus → 存入 DB
        ↓
返回订单信息（⚠️ 当前缺少支付二维码/链接）
        ↓
用户完成支付（微信/支付宝）
        ↓
支付平台回调 → POST /payment/callback/wechat
        ↓ (⚠️ 当前未实现签名验证)
HandleCallback → 幂等检查 → 更新状态 → 充值余额
        ↓ (⚠️ 缺事务保护，缺佣金结算联动)
完成
```

---

## 4. 推荐返佣系统

### 4.1 架构

```
新用户注册(带 referral_code)
        ↓
AuthService.Register → ReferralService.RecordReferral
        ↓                        ↓
  ReferralService.EnsureInviteCode   记录推荐关系
        ↓
支付成功 → SettleCommission（⚠️ 当前未联动）
        ↓
计算佣金 → 创建佣金记录 → 邀请人余额增加
```

### 4.2 已实现功能

| 功能 | 方法 | 状态 |
|------|------|------|
| 生成邀请码 | `EnsureInviteCode` | ✅ 幂等，crypto/rand |
| 记录推荐关系 | `RecordReferral` | ✅ 含自推荐防护 |
| 佣金计算 | `SettleCommission` | ✅ 支持配置费率 |
| 佣金结算+入账 | `SettleCommission` | ⚠️ 缺事务 |
| 佣金统计 | `GetReferralStats` | ⚠️ 内存求和 |
| 注册时集成 | `auth_service.go:195-205` | ✅ |

### 4.3 佣金计算规则

```
佣金 = 支付金额(USD) × 佣金比例(默认 10%)

示例：
- 用户 B 通过用户 A 的邀请码注册
- 用户 B 充值 100 CNY ≈ 13.89 USD
- 用户 A 获得佣金 = 13.89 × 10% = 1.389 USD
```

### 4.4 邀请码格式

- 长度：8 字符 hex（`crypto/rand` 生成）
- 示例：`a3f7b2c1`
- 存储：`referrals` 表 `invite_code` 字段（需 unique 约束）

---

## 5. 免费试用

### 5.1 功能说明

新用户注册时自动发放免费试用额度，无需用户操作。

### 5.2 配置方式

通过管理后台 Settings 配置：

| 设置项 | 说明 | 默认值 |
|--------|------|--------|
| `free_trial_enabled` | 是否开启免费试用 | `false` |
| `free_trial_amount` | 试用额度（USD） | `0` |

### 5.3 逻辑流程

```
用户注册
  ↓
检查 free_trial_enabled == true
  ↓
获取 free_trial_amount
  ↓
amount > 0 → UpdateBalance(userID, amount)
  ↓
记录日志
```

**代码位置**: `backend/internal/service/auth_service.go:227-237`

### 5.4 状态：✅ 可上线

逻辑完整，有开关控制，有日志记录。建议配合管理后台设置使用。

---

## 6. 分销商系统

### 6.1 当前状态：半成品，不建议上线

仅有 service 层和数据库表，缺少 HTTP handler。

### 6.2 已实现

| 功能 | 说明 |
|------|------|
| 三级等级定义 | Bronze(1) / Silver(2) / Gold(3) |
| 佣金率配置 | 10% / 15% / 20% |
| 等级升级 | `UpgradeReseller(userID, level)` |

### 6.3 缺失

- 管理后台分销商 CRUD handler
- 批量购码流程
- 分销商专属面板

---

## 7. 管理后台操作指南

### 7.1 充值套餐管理

**创建套餐：**
```bash
curl -X POST http://HOST/api/v1/admin/payment/packages \
  -H "x-api-key: YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "基础套餐",
    "amount_cny": 50.00,
    "bonus_rate": 0.10,
    "description": "充值50元，额外赠送10%",
    "sort_order": 1,
    "is_active": true
  }'
```

**更新套餐：**
```bash
curl -X PUT http://HOST/api/v1/admin/payment/packages/1 \
  -H "x-api-key: YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "基础套餐（升级版）", "amount_cny": 68.00}'
```

**删除套餐：**
```bash
curl -X DELETE http://HOST/api/v1/admin/payment/packages/1 \
  -H "x-api-key: YOUR_ADMIN_KEY"
```

### 7.2 免费试用配置

通过管理后台 Settings 页面设置：
1. 登录管理后台
2. 进入"系统设置"
3. 找到"免费试用"区域
4. 开启开关，设置额度金额（USD）
5. 保存

---

## 8. 前端页面

### 8.1 充值页面 (`/recharge`)

- 套餐选择卡片展示
- 自定义金额输入
- 微信/支付宝切换
- 二维码支付弹窗
- 支付状态轮询

**已知问题：**
- 汇率 `CNY_TO_USD = 7.2` 硬编码在前端
- 支付成功后未刷新余额显示
- 错误处理静默失败，无用户提示

### 8.2 推荐页面 (`/referral`)

- 邀请码展示 + 一键复制
- 邀请统计（被邀请人数、佣金总额）
- 返佣记录表格

**已知问题：**
- 后端用户侧 referral API 未暴露

### 8.3 导航集成

- 侧边栏已添加"充值"和"邀请返佣"入口
- `RUN_MODE=simple` 时自动隐藏（正确）

---

## 9. 配置项说明

### backend/config.yaml

```yaml
# 推荐返佣配置
referral:
  commission_rate: 0.10  # 佣金比例，10%

# 幂等性配置（main 新增）
idempotency:
  observe_only: true
  default_ttl_seconds: 86400

# 运行模式
run_mode: "standard"  # standard=完整商业化 | simple=内部使用
```

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `RUN_MODE` | 运行模式 | `standard` |
| `REFERRAL_COMMISSION_RATE` | 佣金比例 | `0.10` |

---

## 10. 数据库迁移

### 迁移文件

| 文件 | 说明 |
|------|------|
| `056_create_payment_orders.sql` | 支付订单表 + 充值套餐表 |
| `057_create_referrals.sql` | 推荐关系表 + 佣金记录表 |
| `058_create_reseller.sql` | 分销商相关表 |

### 执行顺序

```bash
# 连接数据库后按顺序执行
psql -d sub2api < migrations/056_create_payment_orders.sql
psql -d sub2api < migrations/057_create_referrals.sql
psql -d sub2api < migrations/058_create_reseller.sql
```

### 新增表结构

**payment_orders:**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户 ID |
| order_no | varchar | 订单号（唯一） |
| amount_usd | float8 | 金额（USD） |
| bonus | float8 | 奖励额度 |
| total_credit | float8 | 总到账额度 |
| payment_method | varchar | 支付方式 |
| status | varchar | 状态(pending/paid) |
| trade_no | varchar | 第三方交易号 |

**recharge_packages:**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| name | varchar | 套餐名称 |
| amount_cny | float8 | 金额（CNY） |
| bonus_rate | float8 | 奖励比例 |
| description | text | 描述 |
| sort_order | int | 排序 |
| is_active | bool | 是否启用 |

**referrals:**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户 ID |
| invite_code | varchar | 邀请码（唯一） |
| invited_by | bigint | 邀请人 ID（nullable） |

**referral_commissions:**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| inviter_id | bigint | 邀请人 ID |
| invitee_id | bigint | 被邀请人 ID |
| order_id | bigint | 关联订单 ID |
| commission_amount | float8 | 佣金金额 |
| commission_rate | float8 | 佣金比例快照 |

---

## 11. Critical 问题清单

> 以下问题必须在上线前修复。

| ID | 问题 | 文件 | 风险 |
|----|------|------|------|
| C1 | 支付回调未实现签名验证 | `handler/payment_handler.go:81-90` | 任意伪造支付 |
| C2 | HandleCallback 缺数据库事务 | `service/payment_service.go:130-143` | 支付到账不一致 |
| C3 | SettleCommission 缺事务 | `service/referral_service.go:112-137` | 佣金结算不一致 |
| C4 | 支付成功未触发佣金结算 | `service/payment_service.go:130-143` | 佣金流程断裂 |
| C5 | GetOrder 无用户鉴权 (IDOR) | `handler/payment_handler.go:56-68` | 越权查看订单 |
| C6 | UpdateStatus 无状态条件约束 | `repository/payment_repo.go:56-66` | 并发重复充值 |

---

## 12. Major 问题清单

> 建议在正式商用前修复。

| ID | 问题 | 说明 |
|----|------|------|
| M1 | 金额使用 float64 | 浮点精度问题，建议用 int64(分) 或 decimal |
| M2 | 汇率硬编码 7.2 | 前后端均硬编码，需改为配置 |
| M3 | 订单号碰撞风险 | 高并发下可能重复，需加随机部分 |
| M4 | SumCommissions 全表扫描 | 应改用 SQL SUM 聚合 |
| M5 | Admin handler 未验证 ID | ParseInt 错误被忽略 |
| M6 | 邀请码碰撞未处理 | 缺重试逻辑 |
| M7 | RecordReferral 静默吞错误 | 未区分 not found vs DB 错误 |
| M8 | PaymentService.entClient 死代码 | 未使用的字段 |
| M9 | ResellerService.redeemRepo 死代码 | 未使用的字段 |
| M10 | Admin ListOrders 返回空数组 | 功能未实现 |

---

## 13. Minor 问题与后续优化

| ID | 问题 | 建议 |
|----|------|------|
| m1 | 缺少金额上限验证 | 添加 max 校验 |
| m2 | 订单状态魔法字符串 | 定义常量 |
| m3 | 测试覆盖不足 | 补充幂等/并发/错误测试 |
| m4 | 回调路由无 IP 白名单 | 签名验证后可补充 |
| m5 | AmountCNY 未持久化 | 补充 SetAmountCNY |
