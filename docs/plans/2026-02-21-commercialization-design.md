# Sub2API 商业化设计文档

**日期**: 2026-02-21
**版本**: v1.0
**状态**: 设计阶段

---

## 一、商业化目标与定位

Sub2API 作为 SaaS 平台运营，向付费用户出售 AI API 访问权限（Claude、Gemini、OpenAI 等）。

**核心模型**：预充值余额 + 按量计费（Token 单价）
**增长飞轮**：新用户免费试用 → 充值 → 邀请返佣 → 分销代理 → 阶梯优惠锁定大客户
**支付渠道**：微信支付 + 支付宝（国内）

---

## 二、整体架构

```
用户注册
  ├── 新用户赠送试用额度（注册即得）
  └── 可填写邀请码（邀请人获得返佣）

充值流程
  ├── 选择充值金额（阶梯优惠：充越多送越多）
  ├── 微信/支付宝扫码支付
  ├── 支付回调 → 余额到账
  └── 邀请人获得返佣（按充值金额比例）

API 调用
  ├── 每次请求扣除余额（Token 级别计费）
  ├── 余额不足 → 返回 402 错误
  └── 用量记录写入 usage_logs

分销体系
  ├── 用户生成专属邀请链接
  ├── 被邀请人充值时，邀请人获得佣金
  └── 代理商：更高返佣比例 + 专属充值码

增长工具
  ├── 优惠码（注册时使用，赠送余额）
  ├── 兑换码（充值/订阅兑换）
  └── 阶梯充值（充值金额越大，赠送比例越高）
```

---

## 三、模块详细设计

### 模块 1：支付集成（Payment）

**目标**：接入微信支付和支付宝，实现自动充值到账。

#### 数据模型（新增）

```sql
-- 支付订单表
CREATE TABLE payment_orders (
    id          BIGSERIAL PRIMARY KEY,
    order_no    VARCHAR(64) UNIQUE NOT NULL,   -- 平台订单号
    user_id     BIGINT NOT NULL,
    amount      DECIMAL(20,8) NOT NULL,         -- 实付金额（元）
    bonus       DECIMAL(20,8) NOT NULL DEFAULT 0, -- 赠送金额
    total_credit DECIMAL(20,8) NOT NULL,        -- 到账余额 = amount + bonus
    channel     VARCHAR(20) NOT NULL,           -- wechat / alipay
    status      VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending/paid/failed/refunded
    trade_no    VARCHAR(128),                   -- 第三方交易号
    paid_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 充值套餐配置（管理员可配置）
CREATE TABLE recharge_packages (
    id          BIGSERIAL PRIMARY KEY,
    amount      DECIMAL(20,8) NOT NULL,   -- 充值金额（元）
    bonus_rate  DECIMAL(5,4) NOT NULL DEFAULT 0, -- 赠送比例，如 0.1 = 10%
    bonus_fixed DECIM NULL DEFAULT 0, -- 固定赠送金额
    label       VARCHAR(50),              -- 显示标签，如"推荐"
    is_active   BOOLEAN NOT NULL DEFAULT true,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### 接口设计

```
POST /api/v1/payment/create          # 创建支付订单
GET  /api/v1/payment/orders          # 查询订单列表
GET  /api/v1/payment/orders/:id      # 查询单个订单
POST /api/v1/payment/callback/wechat # 微信支付回调（公网可访问）
POST /api/v1/payment/callback/alipay # 支付宝回调（公网可访问）

GET  /admin/api/payment/packages     # 管理员：查询充值套餐
POST /admin/api/payment/packages     # 管理员：创建套餐
PUT  /admin/api/payment/packages/:id # 管理员：更新套餐
GET  /admin/api/payment/orders       # 管理员：查询所有订单
POST /admin/api/payment/refund/:id   # 管理员：退款
```

#### 支付流程

```
前端选择套餐 → POST /payment/create
  → 生成 order_no（雪花ID）
  → 调用微信/支付宝 API 生成二维码
  → 返回 qr_code_url + order_id

前端轮询订单状态（每 2 秒）
  → GET /payment/orders/:id
  → status=paid 时刷新余额

支付回调（异步）
  → 验签
  → 幂等检查（order_no 已处理则忽略）
  → 更新 payment_orders.status = paid
  → 调用 UserRepository.UpdateBalance(+total_credit)
  → 触发邀请返佣逻辑
  → 写入 redeem_codes 记录（类型：充值流水）
```

#### 支付 SDK 选型

推荐使用 `github.com/go-pay/gopay`（国内主流，支持微信V3 + 支付宝）。

---

### 模块 2：阶梯充值优惠（Tiered Pricing）

**目标**：充值越多，赠送比例越高，提升 ARPU 和用户粘性。

#### 套餐示例

| 充值金额 | 赠送比例 | 实际到账 | 标签 |
|---------|---------|---------|------|
| ¥10     | 0%      | ¥10     | -    |
| ¥50     | 5%      | ¥52.5   | -    |
| ¥100    | 10%     | ¥110    | 推荐 |
| ¥300    | 15%     | ¥345| 热门 |
| ¥500    | 20%     | ¥600    | 超值 |
| ¥1000   | 25%     | ¥1250   | 大客户 |

#### 实现方式

- `recharge_packages` 表由管理员配置，前端动态展示
- 支付时服务端根据 `package_id` 计算 `bonus`，防止前端篡改
- 支持自定义金额（无赠送），满足灵活充值需求

---

### 模块 3：邀请返佣系统（Referral）

**目标**：用户邀请新用户注册并充值，邀请人获得佣金，实现低成本裂变获客。

#### 数据模型（新增）

```sql
-- 邀请关系表
CREATE TABLE referrals (
    id              BIGSERIAL PRIMARY KEY,
    inviter_id      BIGINT NOT NULL,   -- 邀请人
    invitee_id      BIGINT NOT NULL UNIQUE, -- 被邀请人（一个用户只能被邀请一次）
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 返佣记录表
CREATE TABLE referral_commissions (
    id              BIGSERIAL PRIMARY KEY,
    inviter_id      BIGINT NOT NULL,
    invitee_id      BIGINT NOT NULL,
    order_id        BIGINT NOT NULL,   -- 关联充值订单
    order_amount    DECIMAL(20,8) NOT NULL,
    commission_rate DECIMAL(5,4) NOT NULL, -- 返佣比例（快照）
    commission_amount DECIMAL(20,8) NOT NULL, -- 返佣金额
    status          VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending/settled/cancelled
    settled_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### 邀请码设计

- 每个用户自动生成唯一邀请码（6位大写字母+数字，存入 `users` 表新增字段 `invite_code`）
- 邀请链接：`https://your-domain.com/register?ref=ABCD12`
- 注册时记录 `referrals` 关系

#### 返佣规则

- 默认返佣比例：被邀请人每次充值金额的 **10%**（管理员可配置）
- 返佣方式：直接增加邀请人余额（实时到账）
- 返佣层级：仅一级（不做多级分销，避免法律风险）
- 返佣上限：可配置单笔最大返佣金额

#### 接口设计

```
GET  /api/v1/referral/info           # 获取我的邀请码和邀请链接
GET  /api/v1/referral/stats          # 邀请统计（邀请人数、总返佣）
GET  /api/v1/referral/commissions    # 返佣记录列表

GET  /admin/api/referral/config      # 管理员：查询返佣配置
PUT  /admin/api/referral/config      # 管理员：更新返佣比例
GET  /admin/api/referral/commissions # 管理员：查询所有返佣记录
```

---

### 模块 4：分销代理体系（Reseller）

**目标**：允许代理商批量购买充值码转售，平台按批发价出售，代理商赚取差价。

#### 设计思路

复用现有 `redeem_codes` 表，新增代理商角色和批量购码功能。

#### 数据模型（扩展现有）

```sql
-- users 表新增字段
ALTER TABLE users ADD COLUMN reseller_level INT NOT NULL DEFAULT 0;
-- 0=普通用户, 1=代理商, 2=高级代理商

-- 代理商批量购码订单
CREATE TABLE reseller_orders (
    id          BIGSERIAL PRIMARY KEY,
    reseller_id BIGINT NOT NULL,
    quantity    INT NOT NULL,
    face_value  DECIMAL(20,8) NOT NULL,  -- 面值（用户使用时到账金额）
    unit_cost   DECIMAL(20,8) NOT NULL,  -- 代理商进价
    total_cost  DECIMAL(20,8) NOT NULL,
    payment_order_id BIGINT,             -- 关联支付订单
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### 代理商权益

| 等级 | 折扣 | 批量购码 | 专属返佣 |
|------|------|---------|---------|
| 普通用户 | 无 | 不支持 | 10% |
| 代理商 | 8折 | 支持 | 15% |
| 高级代理商 | 7折 | 支持 | 20% |

#### 接口设计

```
POST /admin/api/reseller/upgrade/:user_id  # 管理员：升级为代理商
POST /api/v1/reseller/bulk-order           # 代理商：批量购码
GET  /api/v1/reseller/orders               # 代理商：查询购码订单
GET  /api/v1/reseller/codes                # 代理商：查询已购兑换码
```

---

### 模块 5：新用户免费试用（Free Trial）

**目标**：降低注册门槛，让用户先体验再付费。

#### 实现方式

复用现有 `promo_codes` 机制，在注册流程中自动发放。

#### 配置项（新增到 settings 表）

```
free_trial_enabled: true
free_trial_amount: 0.5   # 赠送 $0.5 等值余额（约可调用 claude-haiku 数百次）
free_trial_once_per_ip: true  # 每个 IP 只能领一次（防刷）
```

#### 注册流程变更

```
用户注册成功
  → 检查 free_trial_enabled
  → 检查该 IP 是否已领取（Redis 记录）
  → 是：跳过
  → 否：调用 UserRepository.UpdateBalance(+free_trial_amount)
         写入 redeem_codes 记录（type=free_trial）
         Redis 记录该 IP
```

---

### 模块 6：前端用户界面

#### 新增页面

1. **充值页** (`/recharge`)
   - 展示充值套餐卡片（高亮推荐套餐）
   - 支持自定义金额输入
   - 微信/支付宝二维码弹窗
   - 支付成功动画 + 余额刷新

2. **邀请页** (`/referral`)
   - 我的邀请码 + 一键复制链接
   - 邀请统计：已邀请人数、累计返佣金额
   - 返佣记录列表

3. **账单页** (`/billing`)
   - 余额展示（当前余额 + 累计充值 + 累计消费）
   - 充值记录
   - 消费记录（复用现有 usage_logs）

4. **管理后台扩展**
   - 充值套餐管理
   - 支付订单查询
   - 返佣配置
   - 代理商管理

---

## 四、分批实现计划

### Batch 1：支付核心（最高优先级）

**目标**：用户能充值，钱能到账。

- [ ] `payment_orders` 表 + Ent schema
- [ ] `recharge_packages` 表 + 管理员 CRUD
- [ ] 接入 gopay（微信 Native + 支付宝 PC 扫码）
- [ ] 支付回调处理（验签 + 幂等 + 余额到账）
- [ ] 前端充值页（套餐选择 + 二维码 + 轮询）
- [ ] 管理后台：订单查询 + 退款

### Batch 2：增长工具

**目标**：邀请返佣 + 阶梯优惠上线。

- [ ] `referrals` + `referral_commissions` 表
- [ ] 用户注册时记录邀请关系
- [ ] 支付成功后触发返佣
- [ ] 前端邀请页
- [ ] 新用户免费试用额度（注册自动发放）

### Batch 3：分销代理

**目标**：代理商体系上线。

- [ ] `reseller_orders` 表 + 代理商等级字段
- [ ] 批量购码接口
- [ ] 管理后台代理商管理
- [ ] 代理商专属返佣比例

---

## 五、关键技术决策

### 支付安全

- 回调接口必须验签（微信 V3 RSA + 支付宝 RSA2）
- 使用数据库事务保证余额更新原子性
- 幂等键：`order_no` 唯一约束，防止重复到账
- 回调接口不需要 JWT 认证，但需要 IP 白名单（可选）

### 防刷策略

- 免费试用：IP + 设备指纹双重限制
- 邀请返佣：被邀请人首次充值才触发，防止自充自返
- 支付订单：同一用户同时只能有一个 pending 订单

### 货币单位

- 数据库存储：USD（与现有 balance 字段一致）
- 前端展示：人民币（按汇率换算，汇率可配置）
- 充值金额：人民币计价，到账时按汇率转为 USD 余额

### 与现有系统的集成点

| 现有功能 | 集成方式 |
|---------|---------|
| `users.balance` | 支付成功后调用 `UserRepository.UpdateBalance` |
| `redeem_codes` | 充值流水复用此表记录（type=recharge） |
| `promo_codes` | 免费试用复用此机制 |
| `BillingCacheService` | 余额更新后调用 `InvalidateUserBalance` |
| `auth_service` | 注册时记录邀请关系 |

---

## 六、配置项汇总（新增到 settings 表）

```
payment.wechat.enabled: true/false
payment.wechat.app_id: xxx
payment.wechat.mch_id: xxx
payment.wechat.api_v3_key: xxx
payment.wechat.serial_no: xxx
payment.wechat.private_key: xxx (PEM)

payment.alipay.enabled: true/false
payment.alipay.app_id: xxx
payment.alipay.private_key: xxx
payment.alipay.public_key: xxx

payment.cny_to_usd_rate: 7.2   # 人民币兑美元汇率

referral.enabled: true
referral.commission_rate: 0.10  # 10%
referral.max_commission_per_order: 50  # 单笔最大返佣（USD）

free_trial.enabled: true
free_trial.amount: 0.5          # USD
free_trial.once_per_ip: true
```

---

## 七、风险与注意事项

1. **支付资质**：微信/支付宝需要企业主体，个人无法直接申请商户号。可考虑使用第三方聚合支付（如 Ping++、码支付）作为过渡方案。

2. **多级分销合规**：国内法规对多级分销有限制，本设计仅做一级返佣，规避法律风险。

3. **汇率波动**：CNY→USD 汇率应定期更新，建议每日从外部 API 同步，或允许管理员手动配置。

4. **退款处理**：支付宝/微信退款有时效限制（通常 1 年内），超时需线下处理。

5. **数据安全**：支付密钥（API Key、私钥）必须加密存储，不能明文写入数据库或配置文件，建议使用环境变量或 Vault。
