<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <!-- Vendor Type Filter -->
          <div class="flex gap-2">
            <button
              @click="filterType = 'all'"
              :class="[
                'px-4 py-2 rounded-lg text-sm font-medium transition-colors',
                filterType === 'all'
                  ? 'bg-primary-500 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
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
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
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
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
              ]"
            >
              {{ t('admin.vendors.filterReseller') }}
            </button>
          </div>

          <div class="relative w-full sm:w-64">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500" />
            <input v-model="searchQuery" type="text" placeholder="搜索供应商..." class="input pl-10" @input="handleSearch" />
          </div>
          <div class="w-full sm:w-36">
            <Select v-model="filters.status" :options="statusOptions" placeholder="全部状态" @change="loadVendors" />
          </div>
          <div class="w-full sm:w-40">
            <Select v-model="filters.api_format" :options="apiFormatOptions" placeholder="全部格式" @change="loadVendors" />
          </div>
          <div class="flex flex-1 items-center justify-end gap-2">
            <button @click="loadVendors" :disabled="loading" class="btn btn-secondary" title="刷新">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button @click="openCreateModal" class="btn btn-primary">
              <Icon name="plus" size="md" class="mr-2" />添加供应商
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredVendors" :loading="loading">
          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>
          <template #cell-vendor_type="{ row }">
            <VendorTypeBadge :vendor-type="row.vendor_type" />
          </template>
          <template #cell-api_format="{ value }">
            <span :class="['badge', value === 'anthropic' ? 'badge-primary' : 'badge-gray']">
              {{ value === 'anthropic' ? 'Anthropic' : 'OpenAI' }}
            </span>
          </template>
          <template #cell-billing_type="{ value }">
            <span class="badge badge-gray">{{ billingLabel(value) }}</span>
          </template>
          <template #cell-balance="{ row }">
            <span class="text-sm">{{ formatBalance(row) }}</span>
          </template>
          <template #cell-status="{ value }">
            <span :class="['badge', statusBadge(value)]">{{ statusLabel(value) }}</span>
          </template>
          <template #cell-health="{ row }">
            <span :class="['inline-block h-2.5 w-2.5 rounded-full', healthDot(row.last_health_status)]"
              :title="row.last_health_status || 'unknown'" />
          </template>
          <template #cell-latency="{ row }">
            <span v-if="typeof row.last_health_latency === 'number'" class="text-sm text-gray-700 dark:text-gray-300">
              {{ row.last_health_latency }}ms
            </span>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button @click="handleTest(row)" :disabled="testingIds.has(row.id)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-emerald-50 hover:text-emerald-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400">
                <Icon :name="testingIds.has(row.id) ? 'refresh' : 'checkCircle'" size="sm"
                  :class="testingIds.has(row.id) ? 'animate-spin' : ''" />
                <span class="text-xs">测试</span>
              </button>
              <button @click="handleEdit(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400">
                <Icon name="edit" size="sm" /><span class="text-xs">编辑</span>
              </button>
              <button @click="handleDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                <Icon name="trash" size="sm" /><span class="text-xs">删除</span>
              </button>
            </div>
          </template>
          <template #empty>
            <EmptyState title="暂无供应商" description="添加第一个供应商以开始使用" action-text="添加供应商" @action="openCreateModal" />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination v-if="pagination.total > 0" :page="pagination.page" :total="pagination.total"
          :page-size="pagination.page_size" @update:page="p => { pagination.page = p; loadVendors() }"
          @update:pageSize="s => { pagination.page_size = s; pagination.page = 1; loadVendors() }" />
      </template>
    </TablePageLayout>

    <!-- Create/Edit Modal -->
    <BaseDialog :show="showModal" :title="editingVendor ? '编辑供应商' : '添加供应商'" width="normal" @close="closeModal">
      <form id="vendor-form" @submit.prevent="handleSubmit" class="space-y-5">
        <!-- Auto Detect Section -->
        <div v-if="!editingVendor" class="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-900/20">
          <div class="mb-3 flex items-center gap-2">
            <Icon name="bolt" size="md" class="text-blue-600 dark:text-blue-400" />
            <span class="text-sm font-semibold text-blue-700 dark:text-blue-300">快速检测（可选）</span>
            <span class="text-xs text-blue-500 dark:text-blue-400">填写后自动识别供应商配置和余额</span>
          </div>
          <div class="space-y-3">
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label class="input-label">供应商 URL</label>
                <input v-model="detectForm.url" type="url" class="input" placeholder="https://ai.example.com" />
              </div>
              <div>
                <label class="input-label">API Key</label>
                <input v-model="detectForm.api_key" type="password" autocomplete="off" class="input" placeholder="sk-..." />
              </div>
            </div>
            <button type="button" @click="detectVendor"
              :disabled="detecting || !detectForm.url || !detectForm.api_key"
              class="btn btn-secondary w-full">
              <Icon :name="detecting ? 'refresh' : 'search'" size="sm"
                :class="['mr-2', detecting ? 'animate-spin' : '']" />
              {{ detecting ? '检测中...' : '自动检测并填充表单' }}
            </button>
            <div v-if="detectResult"
              :class="['rounded-lg p-3 text-sm', detectResult.success
                ? 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-300'
                : 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300']">
              <div class="font-medium">{{ detectResult.success ? '✓' : '✗' }} {{ detectResult.message }}</div>
              <template v-if="detectResult.success">
                <div v-if="detectResult.models.length > 0" class="mt-1.5">
                  <span class="text-xs opacity-75">检测到 {{ detectResult.models.length }} 个模型：</span>
                  <div class="mt-1 flex flex-wrap gap-1">
                    <span v-for="m in detectResult.models.slice(0, 8)" :key="m"
                      class="rounded bg-green-100 px-1.5 py-0.5 text-xs dark:bg-green-800/40">{{ m }}</span>
                    <span v-if="detectResult.models.length > 8"
                      class="rounded bg-green-100 px-1.5 py-0.5 text-xs dark:bg-green-800/40">
                      +{{ detectResult.models.length - 8 }} 更多
                    </span>
                  </div>
                </div>
                <div class="mt-1 text-xs opacity-75">
                  延迟 {{ detectResult.latency_ms }}ms · 平台: {{ detectResult.platform_type }}
                </div>
              </template>
            </div>
          </div>
        </div>
        <!-- Basic Info -->
        <fieldset>
          <legend class="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">基本信息</legend>
          <div class="space-y-4">
            <div>
              <label class="input-label">名称</label>
              <input v-model="form.name" type="text" required class="input" placeholder="供应商名称" />
            </div>
            <div>
              <label class="input-label">描述</label>
              <input v-model="form.description" type="text" class="input" placeholder="可选描述" />
            </div>
            <div>
              <label class="input-label">渠道类型</label>
              <Select v-model="form.vendor_type" :options="vendorTypeOptions" />
            </div>
            <div v-if="form.vendor_type === 'official'">
              <label class="input-label">官方平台 <span class="text-red-500">*</span></label>
              <Select v-model="form.official_platform" :options="officialPlatformOptions" />
            </div>
            <div v-if="form.vendor_type === 'reseller'">
              <label class="input-label">二次分发平台 <span class="text-red-500">*</span></label>
              <Select v-model="form.reseller_platform" :options="resellerPlatformOptions" />
            </div>
            <div v-if="form.vendor_type === 'reseller'">
              <label class="input-label">分发商 API Key</label>
              <input v-model="form.reseller_api_key" type="password" autocomplete="off" class="input" placeholder="分发商提供的 API Key" />
            </div>
          </div>
        </fieldset>
        <!-- API Config -->
        <fieldset>
          <legend class="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">API 配置</legend>
          <div class="space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="input-label">API 格式</label>
                <Select v-model="form.api_format" :options="apiFormatSelectOptions" />
              </div>
              <div>
                <label class="input-label">认证方式</label>
                <Select v-model="form.auth_type" :options="authTypeOptions" />
              </div>
            </div>
            <div>
              <label class="input-label">Base URL</label>
              <input v-model="form.base_url" type="url" required class="input" placeholder="https://api.example.com" />
            </div>
            <div>
              <label class="input-label">API 路径覆盖</label>
              <input v-model="form.api_path_override" type="text" class="input" placeholder="/v1/messages (可选)" />
            </div>
            <div>
              <label class="input-label">额外 Headers (JSON)</label>
              <textarea v-model="extraHeadersJson" rows="2" class="input font-mono text-sm" placeholder='{"X-Custom": "value"}' />
            </div>
          </div>
        </fieldset>
        <!-- Billing -->
        <fieldset>
          <legend class="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">计费信息</legend>
          <div class="space-y-4">
            <div>
              <label class="input-label">计费模式</label>
              <Select v-model="form.billing_type" :options="billingTypeOptions" />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="input-label">输入成本 ($/1K tokens)</label>
                <input v-model.number="form.cost_per_1k_input" type="number" step="0.001" min="0" class="input" />
              </div>
              <div>
                <label class="input-label">输出成本 ($/1K tokens)</label>
                <input v-model.number="form.cost_per_1k_output" type="number" step="0.001" min="0" class="input" />
              </div>
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="input-label">总额度 (USD)</label>
                <input v-model.number="form.total_quota_usd" type="number" step="0.01" min="0" class="input" />
              </div>
              <div>
                <label class="input-label">余额 (USD)</label>
                <input v-model.number="form.balance_usd" type="number" step="0.01" min="0" class="input" />
              </div>
            </div>
            <div>
              <label class="input-label">过期时间</label>
              <input v-model="form.expires_at" type="datetime-local" class="input" />
            </div>
          </div>
        </fieldset>
        <!-- Health & Alerts -->
        <fieldset>
          <legend class="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">健康监控 & 预警</legend>
          <div class="space-y-4">
            <label class="flex items-center gap-2">
              <input v-model="form.health_check_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
              <span class="text-sm text-gray-700 dark:text-gray-300">启用健康检查</span>
            </label>
            <div v-if="form.health_check_enabled" class="grid grid-cols-2 gap-4">
              <div>
                <label class="input-label">检查间隔 (秒)</label>
                <input v-model.number="form.health_check_interval" type="number" min="30" class="input" />
              </div>
              <div>
                <label class="input-label">检查模型</label>
                <input v-model="form.health_check_model" type="text" class="input" placeholder="claude-sonnet-4-20250514" />
              </div>
            </div>
            <label class="flex items-center gap-2">
              <input v-model="form.balance_alert_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
              <span class="text-sm text-gray-700 dark:text-gray-300">启用余额预警</span>
            </label>
            <div v-if="form.balance_alert_enabled">
              <label class="input-label">预警阈值 (USD)</label>
              <input v-model.number="form.balance_alert_threshold" type="number" step="0.01" min="0" class="input" />
            </div>
          </div>
        </fieldset>
        <!-- Status (edit only) -->
        <div v-if="editingVendor">
          <label class="input-label">状态</label>
          <Select v-model="form.status" :options="editStatusOptions" />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModal" type="button" class="btn btn-secondary">取消</button>
          <button type="submit" form="vendor-form" :disabled="submitting" class="btn btn-primary">
            {{ submitting ? '提交中...' : (editingVendor ? '更新' : '创建') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation -->
    <ConfirmDialog :show="showDeleteDialog" title="删除供应商"
      :message="`确定要删除供应商「${deletingVendor?.name}」吗？此操作不可撤销。`"
      confirm-text="删除" cancel-text="取消" :danger="true"
      @confirm="confirmDelete" @cancel="showDeleteDialog = false" />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { vendorApi, type Vendor, type CreateVendorRequest, type VendorProbeResult } from '@/api/vendor'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import VendorTypeBadge from '@/components/admin/vendor/VendorTypeBadge.vue'

const { t } = useI18n()
const appStore = useAppStore()

const filterType = ref<'all' | 'official' | 'reseller'>('all')

const columns = computed<Column[]>(() => [
  { key: 'name', label: '名称', sortable: true },
  { key: 'vendor_type', label: t('admin.vendors.vendorType'), sortable: true },
  { key: 'api_format', label: 'API 格式', sortable: true },
  { key: 'billing_type', label: '计费模式', sortable: false },
  { key: 'balance', label: '余额/额度', sortable: false },
  { key: 'status', label: '状态', sortable: true },
  { key: 'health', label: '健康', sortable: false },
  { key: 'latency', label: '延迟', sortable: false },
  { key: 'actions', label: '操作', sortable: false },
])

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'active', label: '正常' },
  { value: 'suspended', label: '暂停' },
  { value: 'depleted', label: '耗尽' },
  { value: 'error', label: '错误' },
]
const apiFormatOptions = [
  { value: '', label: '全部格式' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
]
const apiFormatSelectOptions = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
]
const authTypeOptions = [
  { value: 'api_key', label: 'API Key' },
  { value: 'session', label: 'Session' },
  { value: 'bearer', label: 'Bearer Token' },
]
const billingTypeOptions = [
  { value: 'token', label: 'Token 计费' },
  { value: 'quota', label: '额度计费' },
  { value: 'subscription', label: '订阅制' },
]
const vendorTypeOptions = [
  { value: 'official', label: '官方渠道' },
  { value: 'reseller', label: '二次分发' },
]
const officialPlatformOptions = [
  { value: 'claude', label: 'Claude (Anthropic)' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini (Google)' },
]
const resellerPlatformOptions = [
  { value: 'sub2api', label: 'Sub2API' },
  { value: 'newapi', label: 'NewAPI' },
  { value: 'other', label: '其他' },
]
const editStatusOptions = [
  { value: 'active', label: '正常' },
  { value: 'suspended', label: '暂停' },
  { value: 'depleted', label: '耗尽' },
  { value: 'error', label: '错误' },
]

const vendors = ref<Vendor[]>([])
const loading = ref(false)
const searchQuery = ref('')
const filters = reactive({ status: '', api_format: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const filteredVendors = computed(() => {
  if (filterType.value === 'all') return vendors.value
  return vendors.value.filter(v => v.vendor_type === filterType.value)
})

const showModal = ref(false)
const showDeleteDialog = ref(false)
const submitting = ref(false)
const editingVendor = ref<Vendor | null>(null)
const deletingVendor = ref<Vendor | null>(null)
const testingIds = ref<Set<number>>(new Set())
const extraHeadersJson = ref('{}')

// Auto-detect state
const detecting = ref(false)
const detectForm = reactive({ url: '', api_key: '' })
const detectResult = ref<VendorProbeResult | null>(null)

const defaultForm = (): CreateVendorRequest & { status?: string; health_check_enabled: boolean; health_check_interval: number; health_check_model: string; balance_alert_enabled: boolean; balance_alert_threshold?: number } => ({
  name: '', description: '', vendor_type: 'official', official_platform: undefined, reseller_platform: undefined, reseller_api_key: '', api_format: 'anthropic', base_url: '', auth_type: 'api_key',
  api_path_override: '', billing_type: 'token', cost_per_1k_input: 0, cost_per_1k_output: 0,
  total_quota_usd: 0, balance_usd: 0, expires_at: '', health_check_enabled: false,
  health_check_interval: 300, health_check_model: 'claude-sonnet-4-20250514',
  balance_alert_enabled: false, balance_alert_threshold: 10, status: 'active',
})
const form = reactive(defaultForm())

const billingLabel = (v: string) => ({ token: 'Token', quota: '额度', subscription: '订阅' }[v] || v)
const statusLabel = (v: string) => ({ active: '正常', suspended: '暂停', depleted: '耗尽', error: '错误' }[v] || v)
const statusBadge = (v: string) => ({ active: 'badge-success', suspended: 'badge-warning', depleted: 'badge-danger', error: 'badge-danger' }[v] || 'badge-gray')
const healthDot = (v?: string) => ({ ok: 'bg-green-500', slow: 'bg-yellow-500', error: 'bg-red-500', timeout: 'bg-red-500' }[v || ''] || 'bg-gray-300')
const formatBalance = (row: Vendor) => {
  if (row.billing_type === 'token' && row.balance_usd != null) return `$${row.balance_usd.toFixed(2)}`
  if (row.billing_type === 'quota' && row.total_quota_usd != null) return `$${((row.total_quota_usd || 0) - row.used_quota_usd).toFixed(2)} / $${row.total_quota_usd.toFixed(2)}`
  return '-'
}

let searchTimeout: ReturnType<typeof setTimeout>
const handleSearch = () => { clearTimeout(searchTimeout); searchTimeout = setTimeout(() => { pagination.page = 1; loadVendors() }, 300) }

const loadVendors = async () => {
  loading.value = true
  try {
    const params: Record<string, any> = { page: pagination.page, page_size: pagination.page_size }
    if (filters.status) params.status = filters.status
    if (filters.api_format) params.api_format = filters.api_format
    if (searchQuery.value) params.search = searchQuery.value
    const res = await vendorApi.list(params)
    const data = res.data ?? res
    vendors.value = data.items || []
    pagination.total = data.total || 0
  } catch (e: any) {
    appStore.showError(e?.message || e?.response?.data?.message || '加载供应商列表失败')
  } finally { loading.value = false }
}

const detectVendor = async () => {
  if (!detectForm.url || !detectForm.api_key) return
  detecting.value = true
  detectResult.value = null
  try {
    const res = await vendorApi.detect({ url: detectForm.url, api_key: detectForm.api_key })
    const data = (res as { data?: VendorProbeResult }).data ?? (res as unknown as VendorProbeResult)
    detectResult.value = data
    if (data.success) {
      form.base_url = data.base_url
      form.api_format = (data.api_format as 'openai' | 'anthropic') || 'openai'
      form.auth_type = 'api_key'
      if (data.balance_usd != null) form.balance_usd = data.balance_usd
      if (!form.name) {
        try { form.name = new URL(data.base_url).hostname } catch { /* ignore invalid URL */ }
      }
      // 自动填充 API Key 到 extra_headers
      if (detectForm.api_key) {
        const headerKey = data.api_format === 'anthropic' ? 'x-api-key' : 'Authorization'
        const headerVal = data.api_format === 'anthropic' ? detectForm.api_key : `Bearer ${detectForm.api_key}`
        let headers: Record<string, string> = {}
        try { headers = JSON.parse(extraHeadersJson.value || '{}') } catch { /* ignore parse error */ }
        headers[headerKey] = headerVal
        extraHeadersJson.value = JSON.stringify(headers, null, 2)
      }
    }
  } catch (e: any) {
    detectResult.value = { success: false, message: e?.message || e?.response?.data?.message || '检测请求失败', api_format: '', base_url: '', platform_type: '', models: [], latency_ms: 0, detected_at: '' }
  } finally {
    detecting.value = false
  }
}

const openCreateModal = () => {
  editingVendor.value = null
  Object.assign(form, defaultForm())
  extraHeadersJson.value = '{}'
  detectForm.url = ''
  detectForm.api_key = ''
  detectResult.value = null
  showModal.value = true
}
const handleEdit = (v: Vendor) => {
  editingVendor.value = v
  Object.assign(form, { name: v.name, description: v.description || '', vendor_type: v.vendor_type, official_platform: v.official_platform, reseller_platform: v.reseller_platform, reseller_api_key: v.reseller_api_key || '', api_format: v.api_format, base_url: v.base_url, auth_type: v.auth_type, api_path_override: v.api_path_override || '', billing_type: v.billing_type, cost_per_1k_input: v.cost_per_1k_input || 0, cost_per_1k_output: v.cost_per_1k_output || 0, total_quota_usd: v.total_quota_usd || 0, balance_usd: v.balance_usd || 0, expires_at: v.expires_at ? v.expires_at.slice(0, 16) : '', health_check_enabled: v.health_check_enabled, health_check_interval: v.health_check_interval, health_check_model: v.health_check_model, balance_alert_enabled: v.balance_alert_enabled, balance_alert_threshold: v.balance_alert_threshold || 10, status: v.status })
  extraHeadersJson.value = JSON.stringify(v.extra_headers || {}, null, 2)
  showModal.value = true
}
const closeModal = () => { showModal.value = false; editingVendor.value = null; detectResult.value = null }

const handleSubmit = async () => {
  if (!form.name.trim() || !form.base_url.trim()) { appStore.showError('名称和 Base URL 为必填项'); return }
  if (form.vendor_type === 'official' && !form.official_platform) { appStore.showError('官方渠道必须选择官方平台'); return }
  if (form.vendor_type === 'reseller' && !form.reseller_platform) { appStore.showError('二次分发渠道必须选择分发平台'); return }
  let parsedHeaders: Record<string, string> = {}
  try { parsedHeaders = JSON.parse(extraHeadersJson.value || '{}') } catch { appStore.showError('额外 Headers JSON 格式错误'); return }
  submitting.value = true
  try {
    const payload: any = { ...form, extra_headers: parsedHeaders }
    if (!payload.expires_at) {
      delete payload.expires_at
    } else {
      // datetime-local produces "YYYY-MM-DDTHH:mm", Go needs RFC3339
      const raw = payload.expires_at as string
      if (raw && !raw.includes('Z') && !raw.includes('+')) {
        payload.expires_at = raw.length <= 16 ? raw + ':00Z' : raw + 'Z'
      }
    }
    if (editingVendor.value) {
      await vendorApi.update(editingVendor.value.id, payload)
      appStore.showSuccess('供应商已更新')
    } else {
      delete payload.status
      await vendorApi.create(payload)
      appStore.showSuccess('供应商已创建')
    }
    closeModal(); loadVendors()
  } catch (e: any) { appStore.showError(e?.message || e?.response?.data?.message || '操作失败') }
  finally { submitting.value = false }
}

const handleDelete = (v: Vendor) => { deletingVendor.value = v; showDeleteDialog.value = true }
const confirmDelete = async () => {
  if (!deletingVendor.value) return
  try {
    await vendorApi.delete(deletingVendor.value.id)
    appStore.showSuccess('供应商已删除'); showDeleteDialog.value = false; deletingVendor.value = null; loadVendors()
  } catch (e: any) { appStore.showError(e?.message || e?.response?.data?.message || '删除失败') }
}

const handleTest = async (v: Vendor) => {
  testingIds.value = new Set([...testingIds.value, v.id])
  try {
    const res = await vendorApi.test(v.id)
    const data = (res as any).data ?? res
    const status = data?.status as string
    const latency = data?.latency_ms as number | undefined
    const errorMsg = data?.error as string | undefined

    if (status === 'ok' || status === 'slow') {
      if (status === 'slow') {
        appStore.showWarning(`连通成功但响应较慢 (${latency}ms)`)
      } else {
        appStore.showSuccess(latency ? `连通成功 (${latency}ms)` : '连通成功')
      }
    } else if (status === 'timeout') {
      appStore.showError('连通超时，供应商无响应')
    } else {
      appStore.showError(errorMsg ? `连通失败: ${errorMsg}` : '连通测试失败')
    }
    loadVendors()
  } catch (e: any) { appStore.showError(e?.message || e?.response?.data?.message || '连通测试失败') }
  finally { const s = new Set(testingIds.value); s.delete(v.id); testingIds.value = s }
}

onMounted(loadVendors)
</script>
