# 统一渠道商架构实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标**: 统一管理官方渠道和二次分发渠道，通过 vendor_type 字段区分渠道类型，支持三种使用模式

**架构**: 扩展 Vendor 表添加 vendor_type 字段，增强调度器逻辑检查 Vendor 状态，更新前端界面支持渠道类型筛选和创建向导。完全向后兼容现有分组功能。

**技术栈**: Go 1.25+, Ent ORM, PostgreSQL 15+, Vue 3, TypeScript, Pinia

---

## Phase 1: 数据库和模型（预计 1 天）

### Task 1: 创建数据库迁移文件

**文件**:
- Create: `backend/migrations/067_add_vendor_type.sql`

**Step 1: 创建迁移文件**

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

**Step 2: 验证迁移文件语法**

Run: `cat backend/migrations/067_add_vendor_type.sql`
Expected: 文件内容正确显示

**Step 3: Commit**

```bash
git add backend/migrations/067_add_vendor_type.sql
git commit -m "migration: add vendor_type field to vendors table

- Add vendor_type (official/reseller) to distinguish channel types
- Add official_platform for official channels
- Add reseller_platform and reseller_api_key for reseller channels
- Update existing vendors to official type by default"
```

---

### Task 2: 更新 Ent Schema

**文件**:
- Modify: `backend/ent/schema/vendor.go:32-71`

**Step 1: 添加新字段到 Vendor schema**

在 `func (Vendor) Fields()` 中添加新字段（在 `field.String("name")` 之后）:

```go
// 渠道类型
field.String("vendor_type").
    MaxLen(20).
    NotEmpty().
    Default("official").
    Comment("official | reseller"),

// 官方渠道字段
field.String("official_platform").
    MaxLen(50).
    Optional().
    Nillable().
    Comment("claude | openai | gemini (仅 vendor_type=official)"),

// 二次分发渠道字段
field.String("reseller_platform").
    MaxLen(100).
    Optional().
    Nillable().
    Comment("sub2api | newapi | other (仅 vendor_type=reseller)"),

field.String("reseller_api_key").
    MaxLen(500).
    Optional().
    Nillable().
    Comment("渠道商主 API Key (仅 vendor_type=reseller)"),
```

**Step 2: 添加索引**

在 `func (Vendor) Indexes()` 中添加:

```go
index.Fields("vendor_type"),
```

**Step 3: 运行代码生成**

Run: `cd backend && go generate ./ent`
Expected: 成功生成代码，无错误

**Step 4: 验证生成的代码**

Run: `ls backend/ent/vendor*.go`
Expected: 看到更新的文件

**Step 5: Commit**

```bash
git add backend/ent/schema/vendor.go backend/ent/
git commit -m "feat(ent): add vendor_type fields to Vendor schema

- Add vendor_type field (official/reseller)
- Add official_platform for official channels
- Add reseller_platform and reseller_api_key for resellers
- Add index on vendor_type"
```

---

### Task 3: 添加 Vendor 辅助方法

**文件**:
- Modify: `backend/ent/vendor.go` (在生成的代码中添加方法)
- Create: `backend/internal/domain/vendor_helpers.go`

**Step 1: 创建辅助方法文件**

```go
package domain

const (
    VendorTypeOfficial = "official"
    VendorTypeReseller = "reseller"

    OfficialPlatformClaude  = "claude"
    OfficialPlatformOpenAI  = "openai"
    OfficialPlatformGemini  = "gemini"

    ResellerPlatformSub2API = "sub2api"
    ResellerPlatformNewAPI  = "newapi"
    ResellerPlatformOther   = "other"
)

// VendorHelper 提供 Vendor 相关的辅助方法
type VendorHelper struct{}

// IsOfficial 判断是否为官方渠道
func (VendorHelper) IsOfficial(vendorType string) bool {
    return vendorType == VendorTypeOfficial
}

// IsReseller 判断是否为二次分发渠道
func (VendorHelper) IsReseller(vendorType string) bool {
    return vendorType == VendorTypeReseller
}

// GetPlatformName 获取平台显示名称
func (VendorHelper) GetPlatformName(vendorType, officialPlatform, resellerPlatform string) string {
    if vendorType == VendorTypeOfficial && officialPlatform != "" {
        return officialPlatform
    }
    if vendorType == VendorTypeReseller && resellerPlatform != "" {
        return resellerPlatform
    }
    return "unknown"
}
```

**Step 2: Commit**

```bash
git add backend/internal/domain/vendor_helpers.go
git commit -m "feat(domain): add vendor helper methods and constants

- Add vendor type constants
- Add platform constants
- Add helper methods for vendor type checking"
```

---

## Phase 2: 后端逻辑（预计 2 天）

### Task 4: 更新调度器 - 添加 Vendor 状态过滤

**文件**:
- Modify: `backend/internal/service/gateway_service.go`

**Step 1: 找到 filterSchedulableAccounts 方法**

Run: `grep -n "filterSchedulableAccounts" backend/internal/service/gateway_service.go`
Expected: 找到方法定义位置

**Step 2: 添加 Vendor 状态检查逻辑**

在现有的过滤逻辑后添加（假设在第 1850 行左右）:

```go
// 新增：Vendor 状态检查
if acc.VendorID != nil {
    vendor := acc.Edges.Vendor
    if vendor == nil {
        slog.Warn("account has vendor_id but vendor not loaded",
            "account_id", acc.ID,
            "vendor_id", *acc.VendorID)
        continue
    }

    // 检查 Vendor 状态
    if vendor.Status != "active" {
        slog.Debug("vendor not active",
            "vendor_id", vendor.ID,
            "vendor_name", vendor.Name,
            "status", vendor.Status)
        continue
    }

    // 检查 Vendor 余额（仅二次分发渠道）
    if vendor.VendorType == domain.VendorTypeReseller && vendor.BalanceUSD != nil {
        if *vendor.BalanceUSD <= 0 {
            slog.Debug("vendor balance depleted",
                "vendor_id", vendor.ID,
                "vendor_name", vendor.Name,
                "balance", *vendor.BalanceUSD)
            continue
        }
    }
}
```

**Step 3: 确保 Vendor 预加载**

找到 `ListByGroupWithVendor` 或类似方法，确保使用 `WithVendor()`:

```go
accounts, err := s.accountRepo.Query().
    Where(account.HasGroupsWith(group.ID(groupID))).
    WithVendor().  // 确保预加载 Vendor
    All(ctx)
```

**Step 4: Commit**

```bash
git add backend/internal/service/gateway_service.go
git commit -m "feat(scheduler): add vendor status filtering

- Check vendor status before scheduling account
- Skip inactive vendors
- Check reseller vendor balance
- Add debug logging for filtered accounts"
```

---

### Task 5: 更新调度器 - 添加优先级排序

**文件**:
- Modify: `backend/internal/service/gateway_service.go`

**Step 1: 找到账户排序逻辑**

Run: `grep -n "sortAccountsByPriority\|sort.Slice.*account" backend/internal/service/gateway_service.go | head -20`
Expected: 找到排序逻辑位置

**Step 2: 在现有排序逻辑前添加渠道类型优先级**

```go
sort.Slice(accounts, func(i, j int) bool {
    ai := accounts[i]
    aj := accounts[j]

    // 1. 官方渠道优先于二次分发渠道
    if ai.Edges.Vendor != nil && aj.Edges.Vendor != nil {
        vi := ai.Edges.Vendor
        vj := aj.Edges.Vendor

        if vi.VendorType == domain.VendorTypeOfficial && vj.VendorType == domain.VendorTypeReseller {
            return true
        }
        if vi.VendorType == domain.VendorTypeReseller && vj.VendorType == domain.VendorTypeOfficial {
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

    // 3. 现有的优先级逻辑（账户优先级、健康状态等）
    // ... 保持原有逻辑
})
```

**Step 3: Commit**

```bash
git add backend/internal/service/gateway_service.go
git commit -m "feat(scheduler): prioritize official channels over resellers

- Official vendors have higher priority
- Accounts with vendors prioritized over standalone accounts
- Maintain existing priority logic for same vendor type"
```

---

### Task 6: 更新 Admin API - Vendor CRUD

**文件**:
- Modify: `backend/internal/handler/admin/vendor_handler.go`

**Step 1: 更新创建 Vendor 的 DTO**

找到 CreateVendor 的请求结构体，添加新字段:

```go
type CreateVendorRequest struct {
    Name        string  `json:"name" binding:"required"`
    Description *string `json:"description"`

    // 新增字段
    VendorType         string  `json:"vendor_type" binding:"required,oneof=official reseller"`
    OfficialPlatform   *string `json:"official_platform"`
    ResellerPlatform   *string `json:"reseller_platform"`
    ResellerAPIKey     *string `json:"reseller_api_key"`

    // 现有字段...
    APIFormat   string `json:"api_format" binding:"required"`
    BaseURL     string `json:"base_url" binding:"required"`
    // ...
}
```

**Step 2: 添加字段验证逻辑**

```go
// 验证：官方渠道必须提供 official_platform
if req.VendorType == domain.VendorTypeOfficial && req.OfficialPlatform == nil {
    c.JSON(400, gin.H{"error": "official_platform is required for official vendor"})
    return
}

// 验证：二次分发渠道必须提供 reseller_platform
if req.VendorType == domain.VendorTypeReseller && req.ResellerPlatform == nil {
    c.JSON(400, gin.H{"error": "reseller_platform is required for reseller vendor"})
    return
}
```

**Step 3: 更新创建逻辑**

```go
vendor, err := s.vendorService.Create(ctx, &service.CreateVendorInput{
    Name:               req.Name,
    Description:        req.Description,
    VendorType:         req.VendorType,
    OfficialPlatform:   req.OfficialPlatform,
    ResellerPlatform:   req.ResellerPlatform,
    ResellerAPIKey:     req.ResellerAPIKey,
    // ... 其他字段
})
```

**Step 4: 同样更新 UpdateVendor 方法**

**Step 5: Commit**

```bash
git add backend/internal/handler/admin/vendor_handler.go
git commit -m "feat(api): add vendor_type fields to vendor CRUD

- Add vendor_type, official_platform, reseller_platform to request DTOs
- Add validation for required fields based on vendor_type
- Update create and update handlers"
```

---

## Phase 3: 前端界面（预计 2-3 天）

### Task 7: 更新 Vendor 类型定义

**文件**:
- Modify: `frontend/src/types/index.ts`

**Step 1: 找到 Vendor 接口定义**

Run: `grep -n "interface Vendor\|type Vendor" frontend/src/types/index.ts`
Expected: 找到 Vendor 类型定义

**Step 2: 添加新字段**

```typescript
export interface Vendor {
  id: number
  name: string
  description?: string

  // 新增字段
  vendor_type: 'official' | 'reseller'
  official_platform?: 'claude' | 'openai' | 'gemini'
  reseller_platform?: 'sub2api' | 'newapi' | 'other'
  reseller_api_key?: string

  // 现有字段...
  api_format: string
  base_url: string
  status: string
  balance_usd?: number
  // ...
}
```

**Step 3: Commit**

```bash
git add frontend/src/types/index.ts
git commit -m "feat(types): add vendor_type fields to Vendor interface

- Add vendor_type with official/reseller union type
- Add official_platform and reseller_platform
- Add reseller_api_key field"
```

---

### Task 8: 更新 i18n 翻译

**文件**:
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

**Step 1: 添加中文翻译**

在 `zh.ts` 的 `admin.vendors` 部分添加:

```typescript
vendors: {
  // 现有翻译...

  // 新增翻译
  vendorType: '渠道类型',
  vendorTypeOfficial: '官方渠道',
  vendorTypeReseller: '二次分发',
  officialPlatform: '官方平台',
  resellerPlatform: '渠道平台',
  resellerApiKey: '渠道商 API Key',

  filterAll: '全部渠道',
  filterOfficial: '官方渠道',
  filterReseller: '二次分发渠道',

  createWizard: {
    title: '创建渠道商',
    step1Title: '选择渠道类型',
    step2Title: '配置渠道信息',

    officialTitle: '官方渠道',
    officialDesc: '直接对接 Claude、OpenAI 等官方 API',
    officialFeatures: ['稳定性高', '功能完整', '需要官方账号'],

    resellerTitle: '二次分发渠道',
    resellerDesc: '对接 Sub2API、NewAPI 等中转站',
    resellerFeatures: ['价格便宜', '快速接入', '支持订阅限额'],
  }
}
```

**Step 2: 添加英文翻译**

在 `en.ts` 中添加对应的英文翻译

**Step 3: Commit**

```bash
git add frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat(i18n): add vendor type translations

- Add vendor_type related translations
- Add filter labels
- Add creation wizard translations"
```

---

### Task 9: 创建 VendorTypeBadge 组件

**文件**:
- Create: `frontend/src/components/admin/vendor/VendorTypeBadge.vue`

**Step 1: 创建组件文件**

```vue
<script setup lang="ts">
import type { Vendor } from '@/types'

interface Props {
  vendorType: Vendor['vendor_type']
}

const props = defineProps<Props>()
</script>

<template>
  <span
    :class="[
      'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
      vendorType === 'official'
        ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
        : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    ]"
  >
    {{ vendorType === 'official' ? '官方渠道' : '二次分发' }}
  </span>
</template>
```

**Step 2: Commit**

```bash
git add frontend/src/components/admin/vendor/VendorTypeBadge.vue
git commit -m "feat(component): add VendorTypeBadge component

- Display vendor type with color-coded badge
- Blue for official, green for reseller"
```

---

### Task 10: 更新 VendorsView - 添加类型筛选

**文件**:
- Modify: `frontend/src/views/admin/VendorsView.vue`

**Step 1: 添加筛选状态**

在 `<script setup>` 中添加:

```typescript
const filterType = ref<'all' | 'official' | 'reseller'>('all')

const filteredVendors = computed(() => {
  if (filterType.value === 'all') return vendors.value
  return vendors.value.filter(v => v.vendor_type === filterType.value)
})
```

**Step 2: 添加筛选按钮**

在表格上方添加筛选栏:

```vue
<div class="mb-4 flex gap-2">
  <button
    @click="filterType = 'all'"
    :class="[
      'px-4 py-2 rounded-lg text-sm font-medium transition-colors',
      filterType === 'all'
        ? 'bg-primary-500 text-white'
        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
    ]"
  >
    {{ t('admin.vendors.filterAll') }}
  </button>
  <button
    @click="filterType = 'official'"
    :class="[
      'px-4 py-2 rounded-lg text-sm font-medium transition-colors',
      filterType === 'official'
        ? 'bg-primary-500 text-white'
        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
    ]"
  >
    {{ t('admin.vendors.filterOfficial') }}
  </button>
  <button
    @click="filterType = 'reseller'"
    :class="[
      'px-4 py-2 rounded-lg text-sm font-medium transition-colors',
      filterType === 'reseller'
        ? 'bg-primary-500 text-white'
        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
    ]"
  >
    {{ t('admin.vendors.filterReseller') }}
  </button>
</div>
```

**Step 3: 更新表格显示**

在表格中添加类型列，使用 VendorTypeBadge:

```vue
<template #cell-vendor_type="{ row }">
  <VendorTypeBadge :vendor-type="row.vendor_type" />
</template>

<template #cell-platform="{ row }">
  <span v-if="row.vendor_type === 'official'">
    {{ row.official_platform }}
  </span>
  <span v-else>
    {{ row.reseller_platform }}
  </span>
</template>
```

**Step 4: Commit**

```bash
git add frontend/src/views/admin/VendorsView.vue
git commit -m "feat(vendors): add vendor type filtering

- Add filter buttons for all/official/reseller
- Display vendor type badge in table
- Show appropriate platform based on vendor type"
```

---

由于实施计划内容很长，我已经创建了前 10 个任务。完整的计划还包括：

- Task 11-15: 创建渠道向导（前端）
- Task 16-18: 更新 AccountsView 显示渠道信息
- Task 19-20: 单元测试
- Task 21-22: 集成测试
- Task 23-24: 文档更新

**计划已保存到**: `docs/plans/2026-03-01-unified-vendor-architecture-implementation.md`

现在有两种执行方式：

**1. Subagent-Driven（本会话）** - 我为每个任务派发新的 subagent，任务间进行代码审查，快速迭代

**2. Parallel Session（独立会话）** - 在新会话中使用 executing-plans skill，批量执行并设置检查点

你想选择哪种方式？