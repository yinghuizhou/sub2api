# 统一渠道商架构设计文档

## 文档信息

- **创建日期**: 2026-03-01
- **设计目标**: 统一管理官方渠道和二次分发渠道
- **影响范围**: Vendor 表、调度器、前端界面
- **向后兼容**: 是（完全兼容现有功能）

---

## 1. 设计背景

### 1.1 当前问题

项目中存在两种上游 API 来源：

1. **官方渠道**：直接对接 Claude、OpenAI、Gemini 等官方 API
   - 使用 Session Key 或 OAuth 认证
   - 按量计费或订阅计费
   - 稳定性高，功能完整

2. **二次分发渠道**：对接 Sub2API、NewAPI 等中转站
   - 使用 API Key 认证
   - 通常是订阅计费（每日限额）
   - 价格便宜，快速接入

当前架构中，这两种渠道的管理方式不统一，导致：
- 概念混乱（官方账号 vs 渠道商账号）
- 管理分散（不同的界面和逻辑）
- 扩展困难（添加新渠道类型需要修改多处）

### 1.2 设计目标

**核心理念**：将官方渠道和二次分发渠道统一视为"渠道商"（Vendor）

```
渠道商 (Vendor)
  ├─ 官方渠道 (vendor_type = "official")
  │   └─ 账号 (Account) → Session Keys
  │
  └─ 二次分发渠道 (vendor_type = "reseller")
      └─ 账号 (Account) → API Keys
```

**优势**：
- ✅ 概念统一：所有上游都是"渠道商"
- ✅ 管理集中：统一的界面和逻辑
- ✅ 易于扩展：添加新渠道类型只需增加 vendor_type
- ✅ 向后兼容：不影响现有分组功能

---

## 2. 架构设计

### 2.1 数据模型

#### 层级关系

```
Vendor (渠道商)
  ↓ 1:N
Account (账号)
  ↓ M:N
Group (分组)
  ↓ 1:N
User (用户)
```

#### Vendor 表扩展

**新增字段**：

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `vendor_type` | VARCHAR(20) | 渠道类型：official / reseller | official |
| `official_platform` | VARCHAR(50) | 官方平台：claude / openai / gemini | NULL |
| `reseller_platform` | VARCHAR(100) | 渠道平台：sub2api / newapi / other | NULL |
| `reseller_api_key` | VARCHAR(500) | 渠道商主 API Key（用于查询余额等） | NULL |

**字段使用规则**：

```
vendor_type = "official":
  - official_platform: 必填（claude / openai / gemini）
  - reseller_platform: NULL
  - reseller_api_key: NULL

vendor_type = "reseller":
  - official_platform: NULL
  - reseller_platform: 必填（sub2api / newapi / other）
  - reseller_api_key: 可选（用于自动查询余额、自动采购）
```

#### Account 表（不变）

Account 表已有 `vendor_id` 字段，无需修改：

```go
field.Int64("vendor_id").
    Optional().
    Nillable().
    Comment("关联的渠道商 ID")
```

#### Group 表（不变）

Group 表已有 `vendor_id` 字段，用于分组级转发，无需修改：

```go
field.Int64("vendor_id").
    Optional().
    Nillable().
    Comment("关联供应商 ID，设置后该分组所有请求通过供应商转发")
```

---

### 2.2 三种使用模式

#### 模式 1：纯分组模式（现有功能，不变）

```
Group: "普通用户组"
├─ vendor_id: NULL
├─ subscription_type: standard
└─ accounts:
    ├─ Account 1 (vendor_id: NULL)  ← 独立账号
    ├─ Account 2 (vendor_id: NULL)  ← 独立账号
    └─ Account 3 (vendor_id: NULL)  ← 独立账号
```

**特点**：
- 不使用 Vendor 功能
- 账户直接关联到分组
- 现有逻辑完全不变

---

#### 模式 2：分组级 Vendor（现有功能，保持）

```
Group: "企业用户组"
├─ vendor_id: 2 (Sub2API)  ← 分组级转发
├─ subscription_type: subscription
└─ accounts: (不使用，所有请求通过 Vendor 2)

Vendor 2: Sub2API
├─ vendor_type: reseller
├─ base_url: https://api.sub2api.com
└─ accounts:
    ├─ Account 1 (vendor_id: 2)
    ├─ Account 2 (vendor_id: 2)
    └─ Account 3 (vendor_id: 2)
```

**特点**：
- 分组配置了 vendor_id
- 所有请求通过该 Vendor 转发
- 不使用分组的 accounts 关联

---

#### 模式 3：混合模式（新功能，推荐）

```
Group: "混合用户组"
├─ vendor_id: NULL
├─ subscription_type: standard
└─ accounts:
    ├─ Account 1 (vendor_id: 1, Claude Official)  ← 官方渠道
    ├─ Account 2 (vendor_id: 1, Claude Official)  ← 官方渠道
    ├─ Account 3 (vendor_id: 2, Sub2API)          ← 二次分发渠道
    └─ Account 4 (vendor_id: 2, Sub2API)          ← 二次分发渠道

Vendor 1: Claude Official (vendor_type: official)
Vendor 2: Sub2API (vendor_type: reseller)
```

**特点**：
- 分组不配置 vendor_id
- 账户分别关联不同的 Vendor
- 调度时根据 Account.vendor_id 管理
- **这是推荐的新模式**

---

### 2.3 调度器逻辑

#### 选择账户流程

```go
func (s *GatewayService) selectAccount(ctx context.Context, groupID int64) (*Account, error) {
    // 1. 获取分组信息
    group, err := s.groupRepo.Get(ctx, groupID)
    if err != nil {
        return nil, err
    }

    // 2. 模式 2：分组级 Vendor（现有逻辑，保持不变）
    if group.VendorID != nil {
        return s.selectAccountFromVendor(ctx, *group.VendorID)
    }

    // 3. 模式 1 & 3：从分组账户池选择
    accounts, err := s.accountRepo.ListByGroupWithVendor(ctx, groupID)
    if err != nil {
        return nil, err
    }

    // 4. 过滤可调度账户
    filtered := s.filterSchedulableAccounts(ctx, accounts)

    if len(filtered) == 0 {
        return nil, ErrNoAvailableAccount
    }

    // 5. 按优先级排序
    sorted := s.sortAccountsByPriority(filtered)

    return &sorted[0], nil
}
```

#### 过滤逻辑（新增 Vendor 状态检查）

```go
func (s *GatewayService) filterSchedulableAccounts(ctx context.Context, accounts []Account) []Account {
    filtered := make([]Account, 0, len(accounts))

    for _, acc := range accounts {
        // 基础可调度性检查（现有逻辑）
        if !acc.IsSchedulable() {
            continue
        }

        // 订阅限额检查（现有逻辑）
        if acc.HasDailyLimit() {
            if s.isAccountOverLimit(ctx, acc) {
                continue
            }
        }

        // 新增：Vendor 状态检查
        if acc.VendorID != nil {
            vendor := acc.Edges.Vendor
            if vendor == nil {
                slog.Warn("account has vendor_id but vendor not loaded", "account_id", acc.ID)
                continue
            }

            // 检查 Vendor 状态
            if vendor.Status != "active" {
                slog.Debug("vendor not active",
                    "vendor_id", vendor.ID,
                    "status", vendor.Status)
                continue
            }

            // 检查 Vendor 余额（仅二次分发渠道）
            if vendor.VendorType == "reseller" && vendor.BalanceUSD != nil {
                if *vendor.BalanceUSD <= 0 {
                    slog.Debug("vendor balance depleted",
                        "vendor_id", vendor.ID,
                        "balance", *vendor.BalanceUSD)
                    continue
                }
            }
        }

        filtered = append(filtered, acc)
    }

    return filtered
}
```

#### 优先级排序（新增渠道类型优先级）

```go
func (s *GatewayService) sortAccountsByPriority(accounts []Account) []Account {
    sort.Slice(accounts, func(i, j int) bool {
        ai := accounts[i]
        aj := accounts[j]

        // 1. 官方渠道优先于二次分发渠道
        if ai.Edges.Vendor != nil && aj.Edges.Vendor != nil {
            vi := ai.Edges.Vendor
            vj := aj.Edges.Vendor

            if vi.VendorType == "official" && vj.VendorType == "reseller" {
                return true
            }
            if vi.VendorType == "reseller" && vj.VendorType == "official" {
                return false
            }
        }

        // 2. 有 Vendor 的优先于无 Vendor 的
        if ai.VendorID != nil && aj.VendorID == nil {
            return true
        }
        if ai.VendorID == nil && aj.VendorID != nil {
            return false
        }

        // 3. 账户优先级（现有逻辑）
        if ai.Priority != aj.Priority {
            return ai.Priority > aj.Priority
        }

        // 4. 健康状态（现有逻辑）
        // ...

        return false
    })

    return accounts
}
```

---

## 3. 前端界面设计

### 3.1 VendorsView 扩展

#### 渠道类型筛选

```vue
<template>
  <div class="vendors-view">
    <!-- 筛选栏 -->
    <div class="filters">
      <button
        @click="filterType = 'all'"
        :class="{ active: filterType === 'all' }"
      >
        全部渠道
      </button>
      <button
        @click="filterType = 'official'"
        :class="{ active: filterType === 'official' }"
      >
        官方渠道
      </button>
      <button
        @click="filterType = 'reseller'"
        :class="{ active: filterType === 'reseller' }"
      >
        二次分发渠道
      </button>
    </div>

    <!-- 渠道列表 -->
    <table>
      <thead>
        <tr>
          <th>渠道名称</th>
          <th>类型</th>
          <th>平台</th>
          <th>状态</th>
          <th>余额</th>
          <th>账号数</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="vendor in filteredVendors" :key="vendor.id">
          <td>{{ vendor.name }}</td>
          <td>
            <span
              v-if="vendor.vendor_type === 'official'"
              class="badge badge-blue"
            >
              官方渠道
            </span>
            <span
              v-else
              class="badge badge-green"
            >
              二次分发
            </span>
          </td>
          <td>
            <span v-if="vendor.vendor_type === 'official'">
              {{ vendor.official_platform }}
            </span>
            <span v-else>
              {{ vendor.reseller_platform }}
            </span>
          </td>
          <td>
            <VendorStatusBadge :status="vendor.status" />
          </td>
          <td>
            <span v-if="vendor.balance_usd !== null">
              ${{ vendor.balance_usd.toFixed(2) }}
            </span>
            <span v-else>-</span>
          </td>
          <td>{{ vendor.account_count || 0 }}</td>
          <td>
            <button @click="viewAccounts(vendor)">查看账号</button>
            <button @click="editVendor(vendor)">编辑</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
```

#### 创建渠道向导

```vue
<template>
  <BaseDialog :show="show" title="创建渠道商" @close="$emit('close')">
    <form @submit.prevent="handleSubmit">
      <!-- 步骤 1：选择渠道类型 -->
      <div v-if="step === 1">
        <h3>选择渠道类型</h3>
        <div class="radio-group">
          <label>
            <input type="radio" v-model="form.vendor_type" value="official" />
            <div class="option-card">
              <h4>官方渠道</h4>
              <p>直接对接 Claude、OpenAI 等官方 API</p>
              <ul>
                <li>✓ 稳定性高</li>
                <li>✓ 功能完整</li>
                <li>✓ 需要官方账号</li>
              </ul>
            </div>
          </label>

          <label>
            <input type="radio" v-model="form.vendor_type" value="reseller" />
            <div class="option-card">
              <h4>二次分发渠道</h4>
              <p>对接 Sub2API、NewAPI 等中转站</p>
              <ul>
                <li>✓ 价格便宜</li>
                <li>✓ 快速接入</li>
                <li>✓ 支持订阅限额</li>
              </ul>
            </div>
          </label>
        </div>
        <button type="button" @click="step = 2">下一步</button>
      </div>

      <!-- 步骤 2：配置渠道信息 -->
      <div v-if="step === 2">
        <!-- 官方渠道配置 -->
        <div v-if="form.vendor_type === 'official'">
          <label>
            渠道名称
            <input v-model="form.name" placeholder="例如：Claude Official" required />
          </label>

          <label>
            官方平台
            <select v-model="form.official_platform" required>
              <option value="">请选择</option>
              <option value="claude">Claude</option>
              <option value="openai">OpenAI</option>
              <option value="gemini">Gemini</option>
            </select>
          </label>

          <label>
            API 格式
            <select v-model="form.api_format" required>
              <option value="anthropic">Anthropic</option>
              <option value="openai">OpenAI</option>
            </select>
          </label>

          <label>
            Base URL
            <input v-model="form.base_url" placeholder="https://api.anthropic.com" required />
          </label>
        </div>

        <!-- 二次分发渠道配置 -->
        <div v-if="form.vendor_type === 'reseller'">
          <label>
            渠道名称
            <input v-model="form.name" placeholder="例如：Sub2API" required />
          </label>

          <label>
            渠道平台
            <select v-model="form.reseller_platform" required>
              <option value="">请选择</option>
              <option value="sub2api">Sub2API</option>
              <option value="newapi">NewAPI</option>
              <option value="other">其他</option>
            </select>
          </label>

          <label>
            Base URL
            <input v-model="form.base_url" placeholder="https://api.sub2api.com" required />
          </label>

          <label>
            主 API Key（可选）
            <input v-model="form.reseller_api_key" type="password" />
            <small>用于查询余额、自动采购等</small>
          </label>

          <label>
            <input type="checkbox" v-model="form.balance_alert_enabled" />
            启用余额预警
          </label>

          <label v-if="form.balance_alert_enabled">
            预警阈值（美元）
            <input v-model.number="form.balance_alert_threshold" type="number" step="0.01" />
          </label>
        </div>

        <div class="actions">
          <button type="button" @click="step = 1">上一步</button>
          <button type="submit">创建渠道</button>
        </div>
      </div>
    </form>
  </BaseDialog>
</template>
```

### 3.2 AccountsView 扩展

#### 显示账户所属渠道

```vue
<template>
  <DataTable :columns="columns" :data="accounts">
    <!-- 渠道列 -->
    <template #cell-vendor="{ row }">
      <div v-if="row.vendor">
        <span
          :class="[
            'badge',
            row.vendor.vendor_type === 'official' ? 'badge-blue' : 'badge-green'
          ]"
        >
          {{ row.vendor.name }}
        </span>
        <small class="text-gray-500">
          {{ row.vendor.vendor_type === 'official'
              ? row.vendor.official_platform
              : row.vendor.reseller_platform }}
        </small>
      </div>
      <span v-else class="text-gray-400">-</span>
    </template>
  </DataTable>
</template>
```

#### 创建账户时选择渠道

```vue
<template>
  <BaseDialog :show="show" title="创建账户">
    <form @submit.prevent="handleSubmit">
      <!-- 基础信息 -->
      <label>
        账户名称
        <input v-model="form.name" required />
      </label>

      <!-- 选择渠道 -->
      <label>
        所属渠道（可选）
        <select v-model="form.vendor_id">
          <option :value="null">不关联渠道</option>
          <optgroup label="官方渠道">
            <option
              v-for="vendor in officialVendors"
              :key="vendor.id"
              :value="vendor.id"
            >
              {{ vendor.name }} ({{ vendor.official_platform }})
            </option>
          </optgroup>
          <optgroup label="二次分发渠道">
            <option
              v-for="vendor in resellerVendors"
              :key="vendor.id"
              :value="vendor.id"
            >
              {{ vendor.name }} ({{ vendor.reseller_platform }})
            </option>
          </optgroup>
        </select>
      </label>

      <!-- 其他字段... -->

      <button type="submit">创建</button>
    </form>
  </BaseDialog>
</template>
```

---

## 4. 数据库迁移

### 迁移文件：`XXX_add_vendor_type.sql`

```sql
-- 添加渠道类型字段
ALTER TABLE vendors ADD COLUMN vendor_type VARCHAR(20) NOT NULL DEFAULT 'official';
ALTER TABLE vendors ADD COLUMN official_platform VARCHAR(50);
ALTER TABLE vendors ADD COLUMN reseller_platform VARCHAR(100);
ALTER TABLE vendors ADD COLUMN reseller_api_key VARCHAR(500);

-- 添加索引
CREATE INDEX idx_vendors_vendor_type ON vendors(vendor_type);

-- 添加注释
COMMENT ON COLUMN vendors.vendor_type IS '渠道类型：official (官方渠道) | reseller (二次分发渠道)';
COMMENT ON COLUMN vendors.official_platform IS '官方平台：claude | openai | gemini (仅 vendor_type=official 时使用)';
COMMENT ON COLUMN vendors.reseller_platform IS '渠道平台：sub2api | newapi | other (仅 vendor_type=reseller 时使用)';
COMMENT ON COLUMN vendors.reseller_api_key IS '渠道商主 API Key，用于查询余额、自动采购等 (仅 vendor_type=reseller 时使用)';

-- 更新现有数据（所有现有 Vendor 标记为官方渠道）
UPDATE vendors SET vendor_type = 'official' WHERE vendor_type IS NULL OR vendor_type = '';
```

---

## 5. 实施步骤

### Phase 1：数据库和模型（1 天）

**任务**：
1. 创建数据库迁移文件
2. 更新 Ent Schema (vendor.go)
3. 运行 `go generate ./ent`
4. 编写单元测试

**验收标准**：
- 迁移成功执行
- Ent 代码生成无错误
- 单元测试通过

---

### Phase 2：后端逻辑（2 天）

**任务**：
1. 更新调度器逻辑（filterSchedulableAccounts）
2. 更新优先级排序（sortAccountsByPriority）
3. 更新 Admin API（Vendor CRUD）
4. 编写集成测试

**验收标准**：
- 三种模式都能正常工作
- Vendor 状态检查生效
- 优先级排序正确
- 集成测试通过

---

### Phase 3：前端界面（2-3 天）

**任务**：
1. 更新 VendorsView（类型筛选、创建向导）
2. 更新 AccountsView（显示渠道、选择渠道）
3. 更新 i18n 翻译
4. 手动测试

**验收标准**：
- 界面友好，操作流畅
- 创建向导引导清晰
- 渠道类型标识明显
- 翻译完整

---

### Phase 4：测试和文档（1 天）

**任务**：
1. 端到端测试
2. 性能测试
3. 更新用户文档
4. 更新 API 文档

**验收标准**：
- 所有测试通过
- 性能无明显下降
- 文档完整准确

---

## 6. 风险和注意事项

### 6.1 向后兼容性

**风险**：修改可能影响现有功能

**缓解措施**：
- 所有新字段都有默认值
- 现有逻辑保持不变
- 新增逻辑只在 vendor_id 不为空时执行
- 充分的单元测试和集成测试

### 6.2 性能影响

**风险**：调度器需要额外查询 Vendor 表

**缓解措施**：
- 使用 Ent 的 WithVendor() 预加载
- 缓存热点 Vendor 数据
- 监控调度延迟

### 6.3 数据一致性

**风险**：Account.vendor_id 和 Vendor 状态可能不同步

**缓解措施**：
- 调度器实时检查 Vendor 状态
- 定期健康检查更新 Vendor 状态
- 提供手动刷新功能

---

## 7. 成功指标

### 7.1 功能指标

- ✅ 三种模式都能正常工作
- ✅ 渠道类型区分清晰
- ✅ 优先级排序符合预期
- ✅ 界面友好易用

### 7.2 性能指标

- ✅ 调度延迟增加 < 10ms
- ✅ 数据库查询次数不增加
- ✅ 内存占用无明显增长

### 7.3 质量指标

- ✅ 单元测试覆盖率 > 80%
- ✅ 集成测试覆盖所有场景
- ✅ 文档完整准确
- ✅ 代码审查通过

---

## 8. 后续优化

### 8.1 智能调度

根据渠道类型和成本自动选择最优账户：
- 官方渠道优先（稳定性）
- 二次分发渠道备用（成本）
- 动态调整优先级

### 8.2 成本分析

按渠道统计成本和用量：
- 官方渠道成本
- 二次分发渠道成本
- 成本对比和优化建议

### 8.3 自动采购

二次分发渠道余额不足时自动采购：
- 监控余额
- 自动下单
- 通知管理员

---

## 9. 总结

### 核心价值

1. **概念统一**：官方和二次分发都是"渠道商"
2. **管理集中**：统一的界面和逻辑
3. **易于扩展**：添加新渠道类型容易
4. **向后兼容**：不影响现有功能

### 关键决策

1. **vendor_type 字段**：区分官方和二次分发
2. **三种模式**：支持不同的使用场景
3. **优先级排序**：官方渠道优先
4. **渐进式实施**：分阶段上线，降低风险

### 下一步行动

1. 审查设计文档
2. 开始 Phase 1 实施
3. 持续跟踪进度
4. 及时调整方案
