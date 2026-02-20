# Batch 5：前端 - 充值页 + 邀请页 + 账单页

---

## Task 15: 充值页 `/recharge`

**Files:**
- Create: `frontend/src/views/RechargeView.vue`
- Modify: `frontend/src/router/index.ts`
- Create: `frontend/src/api/payment.ts`

**Step 1: 创建支付 API 客户端**

```typescript
// frontend/src/api/payment.ts
import { apiClient } from './client'

export interface RechargePackage {
  id: number
  amount_cny: number
  bonus_rate: number
  bonus_fixed: number
  label: string | null
  is_active: boolean
  sort_order: number
}

export interface CreateOrderRequest {
  amount_cny: number
  package_id?: number
  channel: 'wechat' | 'alipay'
}

export interface PaymentOrder {
  id: number
  order_no: string
  amount_cny: number
  total_credit: number
  channel: string
  status: 'pending' | 'paid' | 'failed'
}

export const paymentApi = {
  listPackages: () =>
    apiClient.get<RechargePackage[]>('/payment/packages'),

  createOrder: (data: CreateOrderRequest) =>
    apiClient.post<{ order: PaymentOrder; qr_url: string }>('/payment/create', data),

  getOrder: (id: number) =>
    apiClient.get<PaymentOrder>(`/payment/orders/${id}`),
}
```

**Step 2: 创建充值页组件**

```vue
<!-- frontend/src/views/RechargeView.vue -->
<template>
  <div class="max-w-2xl mx-auto p-6">
    <h1 class="text-2xl font-bold mb-6">充值余额</h1>

    <!-- 套餐选择 -->
    <div class="grid grid-cols-3 gap-3 mb-6">
      <button
        v-for="pkg in packages"
        :key="pkg.id"
        @click="selectPackage(pkg)"
        :class="[
          'relative p-4 rounded-lg border-2 text-center transition-all',
          selectedPackage?.id === pkg.id
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-gray-200 dark:border-gray-700 hover:border-blue-300'
        ]"
      >
        <div v-if="pkg.label" class="absolute -top-2 left-1/2 -translate-x-1/2">
          <span class="bg-orange-500 text-white text-xs px-2 py-0.5 rounded-full">{{ pkg.label }}</span>
        </div>
        <div class="text-xl font-bold">¥{{ pkg.amount_cny }}</div>
        <div v-if="pkg.bonus_rate > 0" class="text-sm text-green-600 mt-1">
          +{{ (pkg.bonus_rate * 100).toFixed(0) }}% 赠送
        </div>
      </button>
    </div>

    <!-- 自定义金额 -->
    <div class="mb-6">
      <label class="block text-sm text-gray-600 mb-1">自定义金额（元）</label>
      <input
        v-model.number="customAmount"
        type="number"
        min="1"
        placeholder="输入充值金额"
        class="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
        @input="selectedPackage = null"
      />
    </div>

    <!-- 支付方式 -->
    <div class="flex gap-3 mb-6">
      <button
        @click="channel = 'wechat'"
        :class="['flex-1 py-3 rounded-lg border-2 font-medium', channel === 'wechat' ? 'border-green-500 text-green-600' : 'border-gray-200']"
      >微信支付</button>
      <button
        @click="channel = 'alipay'"
        :class="['flex-1 py-3 rounded-lg border-2 font-medium', channel === 'alipay' ? 'border-blue-500 text-blue-600' : 'border-gray-200']"
      >支付宝</button>
    </div>

    <!-- 到账预览 -->
    <div v-if="finalAmount > 0" class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-6">
      <div class="flex justify-between text-sm">
        <span class="text-gray-500">实付金额</span>
        <span>¥{{ finalAmount.toFixed(2) }}</span>
      </div>
      <div v-if="bonusUSD > 0" class="flex justify-between text-sm mt-1">
        <span class="text-gray-500">赠送额度</span>
        <span class="text-green-600">+${{ bonusUSD.toFixed(4) }}</span>
      </div>
      <div class="flex justify-between font-bold mt-2 pt-2 border-t">
        <span>到账余额</span>
        <span class="text-blue-600">${{ totalCreditUSD.toFixed(4) }}</span>
  >
    </div>

    <button
      @click="handlePay"
      :disabled="finalAmount <= 0 || loading"
      class="w-full py-3 bg-blue-600 text-white rounded-lg font-medium disabled:opacity-50 hover:bg-blue-700 transition-colors"
    >
      {{ loading ? '创建订单中...' : '立即充值' }}
    </button>

    <!-- 二维码弹窗 -->
    <div v-if="qrUrl" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeQR">
      <div class="bg-white dark:bg-gray-900 rounded-xl p-6 text-center w-72">
        <h3 class="font-bold mb-4">{{ channel === 'wechat' ? '微信扫码支付' : '支付宝扫码支付' }}</h3>
        <img :src="qrUrl" alt="支付二维码" c8 h-48 mx-auto mb-4" />
        <p class="text-sm text-gray-500">请在 5 分钟内完成支付</p>
        <div v-if="pollStatus === 'paid'" class="mt-3 text-green-600 font-medium">✓ 支付成功！</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { paymentApi, type RechargePackage } from '@/api/payment'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const packages = ref<RechargePackage[]>([])
const selectedPackage = ref<RechargePackage | null>(null)
const customAmount = ref<number | null>(null)
const channel = ref<'wechat' | 'alipay'>('wechat')
const loading = ref(false)
const qrUrl = ref('')
const currentOrderId = ref<number | null>(null)
const pollStatus = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const CNY_TO_USD = 7.2 // 从 settings 读取，此处简化

const finalAmount = computed(() => selectedPackage.value?.amount_cny ?? customAmount.value ?? 0)
const bonusUSD = computed(() => {
  if (!selectedPackage.value) return 0
  return (finalAmount.value / CNY_TO_USD) * selectedPackage.value.bonus_rate + selectedPackage.value.bonus_fixed
})
const totalCreditUSD = computed(() => finalAmount.value / CNY_TO_USD + bonusUSD.value)

function selectPackage(pkg: RechargePackage) {
  selectedPackage.value = pkg
  customAmount.value = null
}

async function handlePay() {
  if (finalAmount.value <= 0) return
  loading.value = true
  try {
    const res = await paymentApi.createOrder({
      amount_cny: finalAmount.value,
      package_id: selectedPackage.value?.id,
      channel: channel.value,
    })
    qrUrl.value = res.qr_url
    currentOrderId.value = res.order.id
    startPolling()
  } finally {
    loading.value = false
  }
}

function startPolling() {
  pollTimer = setInterval(async () => {
    if (!currentOrderId.value) return
    const order = await paymentApi.getOrder(currentOrderId.value)
    if (order.status === 'paid') {
      pollStatus.value = 'paid'
      stopPolling()
      await authStore.refreshUser()
      setTimeout(closeQR, 2000)
    }
  }, 2000)
}

function stopPolling() { if (pollTimer) { clearInterval(pollTimer); pollTimer = null } }
function closeQR() { qrUrl.value = ''; pollStatus.value = ''; stopPolling() }

onMounted(async () => {
  packages.value = await paymentApi.listPackages()
})
onUnmounted(stopPolling)
</script>
```

**Step 3: 注册路由**

在 `frontend/src/router/index.ts` 中添加：
```typescript
{
  path: '/recharge',
  name: 'recharge',
  component: () => import('@/views/RechargeView.vue'),
  meta: { requiresAuth: true }
}
```

**Step 4: 运行类型检查**

```bash
pnpm --dir frontend run typecheck
```

**Step 5: Commit**

```bash
git add frontend/src/views/RechargeView.vue \
        frontend/src/api/payment.ts \
        frontend/src/router/index.ts
git commit -m "feat(frontend): add recharge page with package selection and QR payment"
```

---

## Task 16: 邀请页 `/referral`

**Files:**
- Create: `frontend/src/views/ReferralView.vue`
- Create: `frontend/src/api/referral.ts`
- Modify: `frontend/src/router/index.ts`

**Step 1: 创建邀请 API 客户端**

```typescript
// frontend/src/api/referral.ts
import { apiClient } from './client'

export interface ReferralInfo {
  invite_code: string
  invite_url: string
  invitee_count: number
  total_commission_usd: number
}

export interface Commission {
  id: number
  invitee_email: string
  order_amount_usd: number
  commission_amount: number
  created_at: string
}

export const referralApi = {
  getInfo: () => apiClient.get<ReferralInfo>('/referral/info'),
  getCommissions: (page = 1) =>
    apiClient.get<{ items: Commission[]; total: number }>(`/referral/commissions?page=${page}`),
}
```

**Step 2: 创建邀请页组件**

```vue
<!-- frontend/src/views/ReferralView.vue -->
<template>
  <div class="max-w-2xl mx-auto p-6">
    <h1 class="text-2xl font-bold mb-6">邀请好友</h1>

    <!-- 邀请码卡片 -->
    <div class="bg-gradient-to-r from-blue-500 to-purple-600 rounded-xl p-6 text-white mb-6">
      <p class="text-sm opacity-80 mb-2">我的邀请码</p>
      <div class="flex items-center gap-3">
        <span class="text-3xl font-mono font-bold tracking-widest">{{ info?.invite_code }}</span>
        <button @click="copyCode" class="bg-white/20 hover:bg-white/30 px-3 py-1 rounded-lg text-sm transition-colors">
          {{ copied ? '已复制' : '复制' }}
        </button>
      </div>
      <div class="mt-4 flex items-center gap-2 bg-white/10 rounded-lg px-3 py-2">
        <span class="text-xs truncate flex-1">{{ info?.invite_url }}</span>
        <button @click="copyLink" class="text-xs bg-white/20 px-2 py-1 rounded">复制链接</button>
      </div>
    </div>

    <!-- 统计 -->
    <div class="grid grid-cols-2 gap-4 mb-6">
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 text-center">
        <div class="text-2xl font-bold text-blue-600">{{ info?.invitee_count ?? 0 }}</div>
        <div class="text-sm text-gray-500 mt-1">已邀请人数</div>
      </div>
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 text-center">
        <div class="text-2xl font-bold text-green-600">${{ (info?.total_commission_usd ?? 0).toFixed(4) }}</div>
        <div class="text-sm text-gray-500 mt-1">累计返佣</div>
      </div>
    </div>

    <!-- 规则说明 -->
    <div class="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 rounded-lg p-4 mb-6 text-sm text-amber-800 dark:text-amber-200">
      <p class="font-medium mb-1">返佣规则</p>
      <p>好友通过你的邀请链接注册并充值后，你将获得其充值金额 <strong>10%</strong> 的余额奖励，实时到账。</p>
    </div>

    <!-- 返佣记录 -->
    <h2 class="font-semibold mb-3">返佣记录</h2>
    <div v-if="commissions.length === 0" class="text-center text-gray-400 py-8">暂无记录</div>
    <div v-else class="space-y-2">
      <div
        v-for="c in commissions"
        :key="c.id"
        class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg"
      >
        <div>
          <div class="text-sm font-medium">{{ maskEmail(c.invitee_email) }}</div>
          <div class="text-xs text-gray-400">{{ formatDate(c.created_at) }}</div>
        </div>
        <div class="text-green-600 font-medium">+${{ c.commission_amount.toFixed(4) }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { referralApi, type ReferralInfo, type Commission } from '@/api/referral'

const info = ref<ReferralInfo | null>(null)
const commissions = ref<Commission[]>([])
const copied = ref(false)

function maskEmail(email: string) {
  const [user, domain] = email.split('@')
  return user.slice(0, 2) + '***@' + domain
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}

async function copyCode() {
  await navigator.clipboard.writeText(info.value?.invite_code ?? '')
  copied.value = true
  setTimeout(() => copied.value = false, 2000)
}

async function copyLink() {
  await navigator.clipboard.writeText(info.value?.invite_url ?? '')
}

onMounted(async () => {
  info.value = await referralApi.getInfo()
  const res = await referralApi.getCommissions()
  commissions.value = res.items
})
</script>
```

**Step 3: 注册路由**

```typescript
{
  path: '/referral',
  name: 'referral',
  component: () => import('@/views/ReferralView.vue'),
  meta: { requiresAuth: true }
}
```

**Step 4: 运行类型检查**

```bash
pnpm --dir frontend run typecheck
```

**Step 5: Commit**

```bash
git add frontend/src/views/ReferralView.vue \
        frontend/src/api/referral.ts \
        frontend/src/router/index.ts
git commit -m "feat(frontend): add referral page with invite code and commission history"
```

---

## Task 17: 导航栏添加充值和邀请入口

**Files:**
- Modify: `frontend/src/components/layout/Sidebar.vue` 或对应导航组件

**Step 1: 找到导航组件**

```bash
grep -rl "router-link\|NavLink\|sidebar" frontend/src/components/ --include="*.vue" | head -5
```

**Step 2: 添加菜单项**

在用户菜单区域添加：
```vue
<router-link to="/recharge" class="nav-item">充值</router-link>
<router-link to="/referral" class="nav-item">邀请返佣</router-link>
```

**Step 3: 运行前端开发服务器验证**

```bash
# 用户手动运行：
pnpm --dir frontend run dev
# 访问 http://192.168.x.x:5174/recharge 验证页面
```

**Step 4: 运行完整检查**

```bash
pnpm --dir frontend run typecheck && pnpm --dir frontend run lint:check
```

**Step 5: Commit**

```bash
git add frontend/src/components/
git commit -m "feat(frontend): add recharge and referral links to navigation"
```

---

## Task 18: 最终集成验证

**Step 1: 运行后端单元测试**

```bash
cd backend && go test -tags=unit ./... 2>&1 | tail -20
```

Expected: 全部 PASS，无新增失败

**Step 2: 运行前端类型检查**

```bash
pnpm --dir frontend run typecheck
```

Expected: 无类型错误

**Step 3: 构建生产版本**

```bash
make build
```

Expected: 编译成功

**Step 4: 最终 Commit**

```bash
git add -A
git commit -m "feat(commercialization): complete batch 5 frontend implementation"
```
