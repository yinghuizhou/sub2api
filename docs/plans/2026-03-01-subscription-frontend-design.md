# 二次分发功能前端界面设计文档

**日期**: 2026-03-01
**作者**: Claude Code
**状态**: 已批准

## 1. 概述

为二次分发功能开发完整的前端管理界面，支持订阅配置、用量监控、快速操作等功能。

### 1.1 目标

- 在现有账户管理页面中集成订阅配置功能
- 提供完整的用量监控仪表板
- 支持快速操作（续费、调整限额、重置用量、批量配置、导出报表、告警设置）
- 一次性开发完整功能集

### 1.2 用户需求

- **集成方式**: 混合方式（列表显示概览 + 点击进入详细配置）
- **监控面板**: 完整监控仪表板（图表 + 趋势 + 对比 + 告警）
- **快速操作**: 全部功能（续费、调整限额、重置、批量、导出、告警）
- **开发优先级**: 一次性全部开发完

## 2. 技术方案

### 2.1 方案选择

**方案 1: 组件化设计（已选择）**

创建独立的可复用组件，灵活组合使用。

**优点**:
- 组件独立，易于测试和维护
- 可复用性强
- 符合现有项目架构
- 开发效率高，可并行开发

### 2.2 技术栈

- Vue 3 Composition API
- TypeScript
- Chart.js（已有依赖）
- Pinia Store
- TailwindCSS（已有）
- Axios（已有）

## 3. 架构设计

### 3.1 组件层次结构

```
AccountsView.vue (现有页面)
├── AccountTable (现有表格)
│   ├── SubscriptionStatusCell (新增) - 订阅状态列
│   │   ├── SubscriptionBadge - 状态徽章
│   │   └── SubscriptionProgressBar - 用量进度条
│   └── AccountActionMenu (现有)
│       └── 新增"订阅配置"菜单项
│
└── Modals (弹窗层)
    ├── SubscriptionConfigModal (新增) - 订阅配置弹窗
    │   ├── SubscriptionBasicForm - 基础配置表单
    │   ├── SubscriptionUsageChart - 用量趋势图表
    │   └── SubscriptionQuickActions - 快速操作按钮
    │
    └── SubscriptionMonitorModal (新增) - 监控仪表板弹窗
        ├── SubscriptionOverviewCards - 概览卡片
        ├── SubscriptionUsageTrendChart - 用量趋势图
        ├── SubscriptionAccountComparison - 多账户对比
        └── SubscriptionAlertSettings - 告警设置
```

### 3.2 数据流

```
API Layer (api/admin/accounts.ts)
    ↓
Store Layer (stores/subscription.ts) - 新增
    ↓
Component Layer (组件)
    ↓
UI (用户界面)
```

### 3.3 文件结构

```
frontend/src/
├── api/admin/
│   └── accounts.ts (扩展现有文件)
│       - getSubscriptionConfig()
│       - setSubscriptionConfig()
│       - getDailyUsage()
│       - resetDailyUsage()
│       - batchSetSubscriptionConfig()
│       - exportUsageReport()
│
├── stores/
│   └── subscription.ts (新增)
│       - 订阅状态管理
│       - 用量数据缓存
│       - 告警设置
│
├── components/admin/account/subscription/
│   ├── SubscriptionBadge.vue (新增)
│   ├── SubscriptionProgressBar.vue (新增)
│   ├── SubscriptionStatusCell.vue (新增)
│   ├── SubscriptionConfigModal.vue (新增)
│   ├── SubscriptionBasicForm.vue (新增)
│   ├── SubscriptionUsageChart.vue (新增)
│   ├── SubscriptionQuickActions.vue (新增)
│   ├── SubscriptionMonitorModal.vue (新增)
│   ├── SubscriptionOverviewCards.vue (新增)
│   ├── SubscriptionUsageTrendChart.vue (新增)
│   ├── SubscriptionAccountComparison.vue (新增)
│   ├── SubscriptionAlertSettings.vue (新增)
│   └── SubscriptionBatchConfigModal.vue (新增)
│
├── types/
│   └── subscription.ts (新增)
│       - SubscriptionConfig
│       - DailyUsage
│       - SubscriptionStatus
│       - UsageTrend
│       - AlertRule
│
└── views/admin/
    └── AccountsView.vue (修改现有文件)
        - 集成订阅状态列
        - 添加监控面板入口
```

## 4. 数据类型定义

### 4.1 核心类型

```typescript
// frontend/src/types/subscription.ts

/**
 * 订阅配置
 */
export interface SubscriptionConfig {
  enabled: boolean
  daily_limit_usd: number
  subscription_period: 'daily' | 'weekly' | 'monthly'
  subscription_start: string  // ISO 8601 格式
  subscription_end: string    // ISO 8601 格式
}

/**
 * 日用量
 */
export interface DailyUsage {
  account_id: number
  date: string  // YYYY-MM-DD
  usage_usd: number
}

/**
 * 订阅状态
 */
export interface SubscriptionStatus {
  config: SubscriptionConfig | null
  usage: DailyUsage | null
  status: 'normal' | 'warning' | 'exceeded' | 'expired' | 'disabled'
  percentage: number  // 用量百分比 (0-100)
}

/**
 * 用量趋势数据
 */
export interface UsageTrend {
  date: string
  usage_usd: number
  limit_usd: number
}

/**
 * 告警规则
 */
export interface AlertRule {
  threshold: number  // 告警阈值 (0-100)
  enabled: boolean
}

/**
 * 批量配置请求
 */
export interface BatchSubscriptionConfigRequest {
  account_ids: number[]
  config: SubscriptionConfig
}

/**
 * 导出报表请求
 */
export interface ExportUsageReportRequest {
  account_ids?: number[]
  start_date: string
  end_date: string
  format: 'csv' | 'excel'
}
```

## 5. 组件详细设计

### 5.1 SubscriptionStatusCell（表格单元格）

**功能**: 在账户列表中显示订阅状态概览

**显示内容**:
- 订阅状态徽章（正常/警告/超限/过期/未配置）
- 用量进度条（今日用量 / 日限额）
- 百分比数字

**交互**:
- 点击：打开订阅配置弹窗
- 悬停：显示详细信息 tooltip

**Props**:
```typescript
interface Props {
  account: Account
  status: SubscriptionStatus
}
```

### 5.2 SubscriptionConfigModal（配置弹窗）

**功能**: 配置单个账户的订阅设置

**区域划分**:
1. **基础配置区**（左侧）
   - 启用/禁用开关
   - 日限额输入框
   - 订阅周期选择（日/周/月）
   - 有效期日期选择器

2. **用量显示区**（右侧）
   - 今日用量卡片
   - 近 7 天用量趋势图（折线图）

3. **快速操作区**（底部）
   - 续费按钮（+1月 / +3月 / 自定义）
   - 调整限额按钮（+5 / -5 / 自定义）
   - 重置用量按钮（危险操作，需确认）

**Props**:
```typescript
interface Props {
  account: Account
  visible: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'updated'): void
}
```

### 5.3 SubscriptionMonitorModal（监控仪表板）

**功能**: 全局监控所有订阅账户

**布局**:
```
┌─────────────────────────────────────────────┐
│ 📊 订阅账户监控仪表板                        │
├─────────────────────────────────────────────┤
│ [概览卡片区域]                               │
│ ┌──────┬──────┬──────┬──────┐              │
│ │总账户│可用  │今日  │超限  │              │
│ │  8   │  6   │$45.23│  2   │              │
│ └──────┴──────┴──────┴──────┘              │
│                                              │
│ [用量趋势图 - 近 7 天]                       │
│ ┌────────────────────────────────────────┐ │
│ │  📈 折线图（多账户对比）                 │ │
│ └────────────────────────────────────────┘ │
│                                              │
│ [账户用量对比 - 柱状图]                     │
│ ┌────────────────────────────────────────┐ │
│ │  📊 横向柱状图（今日用量 vs 限额）       │ │
│ └────────────────────────────────────────┘ │
│                                              │
│ [告警设置]                                   │
│ ┌────────────────────────────────────────┐ │
│ │ 告警阈值: [80]% ☑️ 启用                  │ │
│ └────────────────────────────────────────┘ │
│                                              │
│ [导出报表] [批量配置] [刷新]                │
└─────────────────────────────────────────────┘
```

**Props**:
```typescript
interface Props {
  visible: boolean
}
```

## 6. API 集成

### 6.1 新增 API 方法

```typescript
// frontend/src/api/admin/accounts.ts

/**
 * 获取账户订阅配置
 */
export async function getSubscriptionConfig(
  accountId: number
): Promise<SubscriptionConfig | null> {
  const { data } = await apiClient.get(
    `/admin/accounts/${accountId}/subscription`
  )
  return data.data
}

/**
 * 设置账户订阅配置
 */
export async function setSubscriptionConfig(
  accountId: number,
  config: SubscriptionConfig
): Promise<void> {
  await apiClient.post(
    `/admin/accounts/${accountId}/subscription`,
    config
  )
}

/**
 * 获取账户日用量
 */
export async function getDailyUsage(
  accountId: number,
  date?: string
): Promise<DailyUsage> {
  const { data } = await apiClient.get(
    `/admin/accounts/${accountId}/daily-usage`,
    { params: { date } }
  )
  return data.data
}

/**
 * 重置账户日用量
 */
export async function resetDailyUsage(
  accountId: number,
  date: string
): Promise<void> {
  await apiClient.delete(
    `/admin/accounts/${accountId}/daily-usage`,
    { params: { date } }
  )
}

/**
 * 批量设置订阅配置
 */
export async function batchSetSubscriptionConfig(
  request: BatchSubscriptionConfigRequest
): Promise<void> {
  await apiClient.post(
    '/admin/accounts/batch-subscription',
    request
  )
}

/**
 * 导出用量报表
 */
export async function exportUsageReport(
  request: ExportUsageReportRequest
): Promise<Blob> {
  const { data } = await apiClient.post(
    '/admin/accounts/export-usage',
    request,
    { responseType: 'blob' }
  )
  return data
}
```

### 6.2 Pinia Store

```typescript
// frontend/src/stores/subscription.ts

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SubscriptionStatus, AlertRule } from '@/types/subscription'

export const useSubscriptionStore = defineStore('subscription', () => {
  // 状态缓存
  const statusCache = ref<Map<number, SubscriptionStatus>>(new Map())

  // 告警规则
  const alertRule = ref<AlertRule>({
    threshold: 80,
    enabled: true
  })

  // 获取账户订阅状态
  async function getStatus(accountId: number): Promise<SubscriptionStatus> {
    // 实现缓存逻辑
  }

  // 刷新状态
  async function refreshStatus(accountId: number): Promise<void> {
    // 实现刷新逻辑
  }

  // 批量刷新
  async function refreshAll(accountIds: number[]): Promise<void> {
    // 实现批量刷新逻辑
  }

  return {
    statusCache,
    alertRule,
    getStatus,
    refreshStatus,
    refreshAll
  }
})
```

## 7. 用户交互流程

### 7.1 查看订阅状态

```
用户打开账户列表
  ↓
系统加载账户数据
  ↓
并行加载每个账户的订阅状态
  ↓
在表格中显示订阅状态列
  ↓
用户可以看到：
  - 状态徽章（颜色编码）
  - 用量进度条
  - 百分比数字
```

### 7.2 配置订阅

```
用户点击账户行的订阅状态单元格
  ↓
打开 SubscriptionConfigModal
  ↓
加载账户的订阅配置和用量数据
  ↓
用户填写/修改配置：
  - 启用订阅限额
  - 设置日限额
  - 选择订阅周期
  - 设置有效期
  ↓
用户点击"保存"
  ↓
调用 API 保存配置
  ↓
刷新列表中的订阅状态
  ↓
显示成功提示
```

### 7.3 快速续费

```
用户在配置弹窗中点击"续费 +1月"
  ↓
系统自动计算新的 subscription_end
  ↓
调用 API 更新配置
  ↓
刷新显示
  ↓
显示成功提示
```

### 7.4 监控仪表板

```
用户点击"监控仪表板"按钮
  ↓
打开 SubscriptionMonitorModal
  ↓
加载所有订阅账户的数据
  ↓
显示：
  - 概览卡片（统计数据）
  - 用量趋势图（近 7 天）
  - 账户对比图（柱状图）
  - 告警设置
  ↓
用户可以：
  - 导出报表
  - 批量配置
  - 调整告警阈值
```

## 8. 错误处理

### 8.1 API 错误

- 网络错误：显示重试按钮
- 401 未授权：跳转登录页
- 403 权限不足：显示权限提示
- 404 账户不存在：显示错误提示
- 500 服务器错误：显示错误详情

### 8.2 数据验证

- 日限额必须 > 0
- 订阅结束时间必须晚于开始时间
- 日期格式验证
- 数字范围验证

### 8.3 用户提示

- 成功操作：Toast 提示（绿色）
- 警告信息：Toast 提示（黄色）
- 错误信息：Toast 提示（红色）
- 危险操作：确认对话框

## 9. 性能优化

### 9.1 数据加载

- 使用 Pinia Store 缓存订阅状态
- 避免重复请求（5 分钟缓存）
- 批量加载优化（一次请求获取多个账户状态）

### 9.2 渲染优化

- 虚拟滚动（如果账户数量 > 100）
- 图表懒加载（仅在弹窗打开时加载）
- 防抖处理（搜索、筛选）

### 9.3 用户体验

- 加载状态（Skeleton）
- 乐观更新（先更新 UI，后调用 API）
- 错误重试机制

## 10. 测试计划

### 10.1 单元测试

- 组件渲染测试
- Props 验证测试
- 事件触发测试
- 计算属性测试

### 10.2 集成测试

- API 调用测试
- Store 状态管理测试
- 组件交互测试

### 10.3 E2E 测试

- 完整用户流程测试
- 边界情况测试
- 错误处理测试

## 11. 国际化

### 11.1 新增翻译 Key

```typescript
// zh-CN
{
  "admin.accounts.subscription": {
    "title": "订阅配置",
    "enabled": "启用订阅限额",
    "dailyLimit": "每日限额",
    "period": "订阅周期",
    "startDate": "开始日期",
    "endDate": "结束日期",
    "todayUsage": "今日用量",
    "status": {
      "normal": "正常",
      "warning": "警告",
      "exceeded": "超限",
      "expired": "已过期",
      "disabled": "未配置"
    },
    "quickActions": {
      "renew": "续费",
      "adjustLimit": "调整限额",
      "resetUsage": "重置用量"
    },
    "monitor": {
      "title": "监控仪表板",
      "totalAccounts": "总账户数",
      "availableAccounts": "可用账户",
      "todayUsage": "今日总用量",
      "exceededAccounts": "超限账户"
    }
  }
}
```

## 12. 实施计划

### 阶段 1: 基础设施（1 天）
- [ ] 创建类型定义文件
- [ ] 扩展 API 方法
- [ ] 创建 Pinia Store
- [ ] 添加国际化翻译

### 阶段 2: 核心组件（2 天）
- [ ] SubscriptionBadge
- [ ] SubscriptionProgressBar
- [ ] SubscriptionStatusCell
- [ ] SubscriptionBasicForm
- [ ] SubscriptionConfigModal

### 阶段 3: 监控功能（1 天）
- [ ] SubscriptionUsageChart
- [ ] SubscriptionOverviewCards
- [ ] SubscriptionUsageTrendChart
- [ ] SubscriptionAccountComparison
- [ ] SubscriptionMonitorModal

### 阶段 4: 高级功能（1 天）
- [ ] SubscriptionQuickActions
- [ ] SubscriptionAlertSettings
- [ ] SubscriptionBatchConfigModal
- [ ] 导出报表功能

### 阶段 5: 集成与测试（1 天）
- [ ] 集成到 AccountsView
- [ ] 单元测试
- [ ] 集成测试
- [ ] 用户验收测试

**总计**: 约 6 天

## 13. 风险与挑战

### 13.1 技术风险

- Chart.js 图表性能（大量数据点）
  - 缓解：限制数据点数量，使用采样

- 并发请求过多（加载所有账户状态）
  - 缓解：批量 API、分页加载、缓存

### 13.2 用户体验风险

- 弹窗过多可能影响体验
  - 缓解：使用抽屉式侧边栏替代部分弹窗

- 数据刷新不及时
  - 缓解：WebSocket 实时推送（后续优化）

## 14. 后续优化

- WebSocket 实时推送用量更新
- 移动端适配
- 暗色主题优化
- 更多图表类型（饼图、雷达图）
- 导出 PDF 报表
- 邮件/Slack 告警通知集成

---

**文档版本**: 1.0
**最后更新**: 2026-03-01
