# 前端开发指南

## 环境要求

- Node.js 20+
- pnpm（**必须使用 pnpm，不能用 npm**）

## 快速开始

```bash
# 安装依赖
pnpm --dir frontend install

# 开发服务器（localhost:5174）
pnpm --dir frontend run dev

# 或进入目录操作
cd frontend
pnpm install
pnpm run dev
```

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Vue 3 | 3.4.0 | UI 框架（Composition API）|
| Vue Router | 4.2.5 | SPA 路由管理 |
| Pinia | 2.1.7 | 全局状态管理 |
| Axios | 1.13.5 | HTTP 请求 |
| TailwindCSS | 3.4.0 | 原子化 CSS |
| Vue i18n | 9.14.5 | 国际化（英文/中文）|
| Chart.js | 4.4.1 | 数据图表 |
| TypeScript | 5.6.0 | 类型检查 |
| Vite | 5.0.10 | 构建工具 |
| Vitest | 2.1.9 | 单元测试 |

## 目录结构

```
frontend/src/
├── api/                    # API 调用层
│   ├── client.ts           # Axios 实例（拦截器、Token 刷新）
│   ├── auth.ts             # 认证 API
│   ├── keys.ts             # API Key 管理
│   ├── usage.ts            # 使用记录
│   ├── user.ts             # 用户资料
│   ├── subscriptions.ts    # 订阅
│   ├── redeem.ts           # 兑换码
│   ├── groups.ts           # 分组
│   ├── announcements.ts    # 公告
│   ├── totp.ts             # TOTP 双因素认证
│   └── admin/              # 管理员 API（16 个模块）
├── components/             # Vue 组件（150+）
│   ├── common/             # 通用基础组件
│   ├── layout/             # 布局组件
│   ├── account/            # 账号管理组件
│   ├── admin/              # 管理员界面组件
│   ├── charts/             # 图表组件
│   └── ...
├── composables/            # 可复用逻辑（11 个）
├── stores/                 # Pinia 状态（5 个）
├── views/                  # 页面组件（25 个）
│   ├── user/               # 用户页面（7 个）
│   ├── admin/              # 管理员页面（12 个）
│   └── auth/               # 认证页面（6 个）
├── router/                 # 路由配置（34 条路由）
├── i18n/                   # 国际化（en/zh）
├── types/                  # TypeScript 类型（1300+ 行）
├── utils/                  # 工具函数
│   ├── format.ts           # 格式化
│   └── url.ts              # URL 处理
├── styles/                 # 全局样式
├── App.vue                 # 根组件
└── main.ts                 # 应用入口
```

## 路由结构

路由定义在 `src/router/index.ts`，共 34 条路由。

### 路由 Meta 属性

```typescript
{
  path: '/my-route',
  meta: {
    requiresAuth: true,     // 需要登录（默认 true）
    requiresAdmin: false,   // 需要管理员权限
    title: '页面标题',       // 固定标题
    titleKey: 'nav.title',  // i18n 标题 key（优先）
  }
}
```

### 路由守卫

- `beforeEach`：检查认证和权限，设置页面标题，简易模式路由限制
- `afterEach`：结束导航加载状态，触发路由预加载

### 添加新路由

```typescript
// router/index.ts
{
  path: '/my-page',
  name: 'MyPage',
  component: () => import('@/views/user/MyPageView.vue'), // 懒加载
  meta: {
    requiresAuth: true,
    titleKey: 'nav.myPage'
  }
}
```

## Pinia 状态管理

### authStore（认证状态）

```typescript
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

// 状态
authStore.user           // 当前用户
authStore.token          // JWT access token
authStore.isAuthenticated// 是否已登录
authStore.isAdmin        // 是否是管理员

// 操作
await authStore.login({ email, password })
await authStore.logout()
await authStore.refreshUser()
authStore.checkAuth()    // 从 localStorage 恢复状态
```

**特性**：
- localStorage 持久化（`sub2api_auth_*` 前缀）
- Access Token 自动刷新（到期前 120 秒主动刷新）
- 用户数据每 60 秒自动刷新

### appStore（全局 UI 状态）

```typescript
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

// Toast 通知
appStore.showSuccess('操作成功')
appStore.showError('操作失败')
appStore.showInfo('提示信息')
appStore.showWarning('警告信息')
appStore.showToast('custom', '自定义消息', 5000)  // 自定义 5 秒

// Loading 状态
appStore.setLoading(true)
await appStore.withLoading(async () => {
    // 自动管理 loading 状态
})

// 站点信息（从后端获取）
appStore.siteName
appStore.siteLogo
appStore.apiBaseUrl

// 版本信息
appStore.currentVersion
appStore.hasUpdate
```

### subscriptionStore（订阅缓存）

```typescript
import { useSubscriptionStore } from '@/stores/subscriptions'

const subStore = useSubscriptionStore()

// 获取活跃订阅（带 60s TTL 缓存）
await subStore.fetchActiveSubscriptions()
subStore.activeSubscriptions   // 订阅列表
subStore.hasActiveSubscriptions// 是否有活跃订阅

// 强制刷新
await subStore.fetchActiveSubscriptions(true)

// 失效缓存
subStore.invalidateCache()
```

## API 客户端

`src/api/client.ts` 配置的 Axios 实例：

```typescript
import { apiClient } from '@/api/client'

// 直接使用
const response = await apiClient.get('/user/profile')
const result = await apiClient.post('/keys', { name: 'My Key' })
```

### 自动功能

1. **Token 注入**：自动添加 `Authorization: Bearer <token>` 头
2. **响应解包**：自动解包 `{ code, message, data }` 格式，只返回 `data`
3. **Token 刷新**：遇到 401 自动尝试刷新 Token，并重试原请求
4. **并发控制**：多个请求同时 401 时，只发一个刷新请求，其他等待
5. **语言头**：自动注入 `Accept-Language` 和 `X-Timezone`

### API 模块调用示例

```typescript
import { authApi } from '@/api/auth'
import { keysApi } from '@/api/keys'
import { usageApi } from '@/api/usage'

// 登录
const result = await authApi.login({ email, password })

// 获取 API Keys（分页）
const { items, total } = await keysApi.list({ page: 1, page_size: 20 })

// 创建 API Key
const newKey = await keysApi.create({
    name: '生产 Key',
    group_id: 1,
    quota: 0
})

// 使用记录统计
const stats = await usageApi.getDashboardStats()
const trend = await usageApi.getDashboardTrend('7d')
```

## Composables

### useForm - 表单提交

```typescript
import { useForm } from '@/composables/useForm'

const { submit, loading } = useForm()

const handleSubmit = () => submit(async () => {
    await keysApi.create({ name: formData.value.name })
    // 成功后会自动显示 success toast
    emit('success')
}, {
    successMessage: '创建成功',
    errorMessage: '创建失败'
})
```

### useTableLoader - 表格数据加载

```typescript
import { useTableLoader } from '@/composables/useTableLoader'

const { items, loading, pagination, load, debouncedReload } = useTableLoader(
    async (page, pageSize, params, signal) => {
        return await keysApi.list({ page, page_size: pageSize, ...params })
    },
    {
        pageSize: 20,
        params: computed(() => ({ status: filterStatus.value }))
    }
)

// 初始加载
onMounted(() => load())

// 搜索（带防抖）
watch(searchQuery, () => debouncedReload())

// 强制刷新
await load()
```

**特性**：
- 自动分页管理
- 请求取消（页面切换时取消进行中的请求）
- 参数防抖（300ms）
- 错误上抛给父组件处理

### useClipboard - 复制到剪贴板

```typescript
import { useClipboard } from '@/composables/useClipboard'

const { copy } = useClipboard()

// 复制并显示 toast
await copy('sk-xxxxx')
await copy('sk-xxxxx', '密钥已复制')
```

### useTableLoader - 账号 OAuth

```typescript
import { useAccountOAuth } from '@/composables/useAccountOAuth'

const { startOAuth, isPolling } = useAccountOAuth()

// 启动 Anthropic OAuth 流程
const token = await startOAuth()
```

## 国际化（i18n）

语言文件在 `src/i18n/locales/en.ts` 和 `src/i18n/locales/zh.ts`。

```typescript
// 在组件中使用
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// 基本翻译
t('common.save')           // "保存" / "Save"
t('keys.create')           // "创建 API Key" / "Create API Key"

// 带参数的翻译
t('usage.count', { count: 100 })

// 切换语言
const { locale } = useI18n()
locale.value = 'zh'
```

### 添加翻译

1. 在 `locales/zh.ts` 和 `locales/en.ts` 中添加对应的 key
2. 按功能模块组织 key（如 `keys.*`, `admin.users.*`）

## 组件开发规范

### 通用组件使用

项目有完整的自建组件系统（无第三方 UI 库）：

```vue
<template>
  <!-- 数据表格 -->
  <DataTable :items="items" :loading="loading">
    <template #column-name="{ item }">
      {{ item.name }}
    </template>
  </DataTable>

  <!-- 分页 -->
  <Pagination
    v-model:page="pagination.page"
    :total="pagination.total"
    :page-size="pagination.pageSize"
    @change="load"
  />

  <!-- 确认对话框 -->
  <ConfirmDialog
    v-model="showConfirm"
    title="确认删除"
    message="此操作不可撤销"
    @confirm="handleDelete"
  />

  <!-- 骨架屏 -->
  <Skeleton v-if="loading" :rows="5" />

  <!-- Toast 通知（全局，通过 appStore 触发）-->
</template>
```

### 新建页面组件

```vue
<!-- views/user/MyPageView.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useTableLoader } from '@/composables/useTableLoader'
import { someApi } from '@/api/something'

const { t } = useI18n()
const appStore = useAppStore()

const { items, loading, pagination, load } = useTableLoader(
    async (page, pageSize) => await someApi.list({ page, page_size: pageSize })
)

onMounted(() => load())
</script>

<template>
  <div class="space-y-6">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
        {{ t('myPage.title') }}
      </h1>
    </div>

    <!-- 内容区 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm">
      <!-- ... -->
    </div>
  </div>
</template>
```

## TailwindCSS 使用

### 深色模式

使用 `class` 策略，在所有元素上加 `dark:` 前缀：

```html
<div class="bg-white dark:bg-gray-800 text-gray-900 dark:text-white">
```

### 项目特定样式

```html
<!-- 卡片样式 -->
<div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">

<!-- 主要按钮 -->
<button class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors">

<!-- 输入框 -->
<input class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500">
```

## 类型系统

所有核心类型定义在 `src/types/index.ts`（1300+ 行）。

```typescript
// 直接导入使用
import type { User, ApiKey, UsageLog, Group, Account } from '@/types'
import type { DashboardStats, TrendDataPoint } from '@/types'

// API 请求/响应类型
import type { PaginatedResponse, ApiResponse } from '@/types'
```

## 测试

```bash
# 运行单元测试
pnpm run test:run

# 带 UI 的测试（可交互）
pnpm run test

# 覆盖率报告
pnpm run test:coverage
```

### 测试文件位置

```
src/__tests__/
├── composables/    # Composable 测试
├── stores/         # Store 测试
└── integration/    # 集成测试
```

### 编写测试

```typescript
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'

describe('MyComponent', () => {
    it('should render correctly', () => {
        const wrapper = mount(MyComponent, {
            global: {
                plugins: [createTestingPinia({ createSpy: vi.fn })]
            }
        })
        expect(wrapper.text()).toContain('Expected text')
    })
})
```

## 代码规范

```bash
# ESLint 检查
pnpm run lint:check

# 自动修复
pnpm run lint

# TypeScript 类型检查
pnpm run typecheck
```

## 构建

```bash
# 生产构建（输出到 ../backend/internal/web/dist/）
pnpm run build

# 预览生产构建
pnpm run preview
```

构建输出会被 Go 后端以 `embed` 构建标签嵌入到二进制文件中。

## 开发代理配置

`vite.config.ts` 中配置的开发代理：

```
/api/*      → http://localhost:8080
/v1/*       → http://localhost:8080
/v1beta/*   → http://localhost:8080
/setup      → http://localhost:8080
```

## 常见问题

**Q: pnpm lock 文件冲突？**

修改 `package.json` 后必须提交 `pnpm-lock.yaml`：
```bash
pnpm install  # 更新 lock 文件
git add package.json pnpm-lock.yaml
```

**Q: 国际化 key 找不到？**

检查 `locales/en.ts` 和 `locales/zh.ts` 是否都添加了对应的翻译 key。

**Q: TypeScript 类型报错？**

在 `src/types/index.ts` 中添加或修改类型定义，然后运行 `pnpm run typecheck` 验证。

**Q: 组件样式在深色模式下不对？**

确保所有颜色类都有对应的 `dark:` 变体，参考已有组件的写法。
