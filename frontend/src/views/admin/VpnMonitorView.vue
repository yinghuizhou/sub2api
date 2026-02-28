<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <span class="text-sm font-semibold text-gray-700 dark:text-gray-300">VPN 监控面板</span>
          <div class="flex flex-1 items-center justify-end gap-2">
            <span v-if="autoRefreshCountdown > 0" class="text-xs text-gray-400">{{ autoRefreshCountdown }}s</span>
            <button @click="refreshAll" :disabled="loading" class="btn btn-secondary" title="刷新">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <!-- Status Cards -->
        <div class="grid grid-cols-2 gap-4 border-b border-gray-200 p-4 dark:border-dark-700 md:grid-cols-4">
          <div class="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20">
            <div class="text-2xl font-bold text-green-700 dark:text-green-400">{{ dashboard?.active_tunnels ?? 0 }}</div>
            <div class="text-sm text-green-600 dark:text-green-500">活跃隧道</div>
          </div>
          <div class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20">
            <div class="text-2xl font-bold text-red-700 dark:text-red-400">{{ dashboard?.offline_tunnels ?? 0 }}</div>
            <div class="text-sm text-red-600 dark:text-red-500">离线隧道</div>
          </div>
          <div class="rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-800 dark:bg-yellow-900/20">
            <div class="text-2xl font-bold text-yellow-700 dark:text-yellow-400">{{ dashboard?.degraded_tunnels ?? 0 }}</div>
            <div class="text-sm text-yellow-600 dark:text-yellow-500">降级隧道</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-2xl font-bold text-gray-700 dark:text-gray-300">{{ dashboard?.total_configs ?? 0 }}</div>
            <div class="text-sm text-gray-500 dark:text-gray-400">配置总数</div>
          </div>
        </div>

        <!-- Tunnel Status Table -->
        <div class="border-b border-gray-200 dark:border-dark-700">
          <div class="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2 dark:border-dark-700 dark:bg-dark-800">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">隧道状态</span>
            <span class="text-xs text-gray-400">{{ tunnels.length }} 个</span>
          </div>
          <div v-if="loading && tunnels.length === 0" class="flex items-center justify-center py-8 text-gray-400">
            <Icon name="refresh" size="md" class="animate-spin" />
          </div>
          <div v-else-if="tunnels.length === 0" class="py-8">
            <EmptyState title="暂无隧道" description="尚未部署任何 VPN 隧道" />
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                <tr>
                  <th class="px-4 py-2">名称</th>
                  <th class="px-4 py-2">类型</th>
                  <th class="px-4 py-2">状态</th>
                  <th class="px-4 py-2">健康</th>
                  <th class="px-4 py-2">延迟</th>
                  <th class="px-4 py-2">出口 IP</th>
                  <th class="px-4 py-2">运行时间</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="t in tunnels" :key="t.name" class="hover:bg-gray-50 dark:hover:bg-dark-800/50">
                  <td class="px-4 py-2 font-medium text-gray-900 dark:text-white">{{ t.name }}</td>
                  <td class="px-4 py-2">
                    <span :class="['badge', t.tunnel_type === 'wireguard' ? 'badge-primary' : 'badge-gray']">
                      {{ t.tunnel_type === 'wireguard' ? 'WireGuard' : 'OpenVPN' }}
                    </span>
                  </td>
                  <td class="px-4 py-2">
                    <span :class="['badge', statusBadge(t.status)]">{{ statusLabel(t.status) }}</span>
                  </td>
                  <td class="px-4 py-2">
                    <span :class="['inline-block h-2.5 w-2.5 rounded-full', healthDot(t.health)]" :title="t.health || 'unknown'" />
                  </td>
                  <td class="px-4 py-2 text-gray-700 dark:text-gray-300">
                    <span v-if="t.latency_ms > 0">{{ t.latency_ms }}ms</span>
                    <span v-else class="text-gray-400">-</span>
                  </td>
                  <td class="px-4 py-2 font-mono text-gray-700 dark:text-gray-300">{{ t.exit_ip || '-' }}</td>
                  <td class="px-4 py-2 text-gray-500">{{ t.uptime || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Event Log -->
        <div class="border-b border-gray-200 dark:border-dark-700">
          <div class="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2 dark:border-dark-700 dark:bg-dark-800">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">事件日志</span>
            <select v-model="eventFilterTunnel" class="ml-2 rounded border border-gray-300 bg-white px-2 py-1 text-xs dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300">
              <option value="">全部隧道</option>
              <option v-for="name in tunnelNames" :key="name" :value="name">{{ name }}</option>
            </select>
            <select v-model="eventFilterType" class="rounded border border-gray-300 bg-white px-2 py-1 text-xs dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300">
              <option value="">全部类型</option>
              <option value="connected">connected</option>
              <option value="disconnected">disconnected</option>
              <option value="failover">failover</option>
              <option value="config_switch">config_switch</option>
              <option value="health_changed">health_changed</option>
            </select>
          </div>
          <div v-if="events.length === 0" class="py-6 text-center text-sm text-gray-400">暂无事件</div>
          <div v-else class="max-h-[300px] divide-y divide-gray-100 overflow-y-auto dark:divide-dark-700">
            <div v-for="ev in events" :key="ev.id" class="flex items-start gap-3 px-4 py-2 hover:bg-gray-50 dark:hover:bg-dark-800/50">
              <span class="shrink-0 pt-0.5 text-xs text-gray-400">{{ relativeTime(ev.created_at) }}</span>
              <span class="shrink-0 font-medium text-gray-900 dark:text-white">{{ ev.tunnel_name }}</span>
              <span :class="['badge shrink-0', eventTypeBadge(ev.event_type)]">{{ ev.event_type }}</span>
              <span class="min-w-0 truncate text-xs text-gray-500">{{ eventSummary(ev.details) }}</span>
            </div>
          </div>
          <div v-if="eventsTotal > events.length" class="border-t border-gray-200 px-4 py-2 dark:border-dark-700">
            <button @click="loadMoreEvents" :disabled="eventsLoading" class="text-sm text-primary-600 hover:text-primary-500 dark:text-primary-400">
              {{ eventsLoading ? '加载中...' : '加载更多' }}
            </button>
          </div>
        </div>

        <!-- Alert Rules -->
        <div>
          <div class="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2 dark:border-dark-700 dark:bg-dark-800">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">告警规则</span>
            <div class="flex-1" />
            <button @click="openCreateRule" class="btn btn-primary btn-sm text-xs">新建规则</button>
          </div>
          <div v-if="alertRules.length === 0" class="py-6 text-center text-sm text-gray-400">暂无告警规则</div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                <tr>
                  <th class="px-4 py-2">名称</th>
                  <th class="px-4 py-2">条件</th>
                  <th class="px-4 py-2">阈值</th>
                  <th class="px-4 py-2">Webhook</th>
                  <th class="px-4 py-2">启用</th>
                  <th class="px-4 py-2">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="rule in alertRules" :key="rule.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/50">
                  <td class="px-4 py-2 font-medium text-gray-900 dark:text-white">{{ rule.name }}</td>
                  <td class="px-4 py-2 text-gray-700 dark:text-gray-300">{{ rule.condition }}</td>
                  <td class="px-4 py-2 text-gray-700 dark:text-gray-300">{{ rule.threshold }}</td>
                  <td class="max-w-[200px] truncate px-4 py-2 font-mono text-xs text-gray-500" :title="rule.webhook_url">{{ rule.webhook_url }}</td>
                  <td class="px-4 py-2">
                    <button @click="toggleRuleEnabled(rule)" :class="['relative inline-flex h-5 w-9 items-center rounded-full transition-colors', rule.enabled ? 'bg-green-500' : 'bg-gray-300 dark:bg-dark-600']">
                      <span :class="['inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform', rule.enabled ? 'translate-x-4.5' : 'translate-x-0.5']" />
                    </button>
                  </td>
                  <td class="px-4 py-2">
                    <div class="flex items-center gap-1">
                      <button @click="openEditRule(rule)" class="rounded p-1 text-gray-400 hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20">
                        <Icon name="edit" size="sm" />
                      </button>
                      <button @click="handleDeleteRule(rule)" class="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20">
                        <Icon name="trash" size="sm" />
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </TablePageLayout>

    <!-- Alert Rule Dialog -->
    <BaseDialog :show="showRuleDialog" :title="editingRule ? '编辑告警规则' : '新建告警规则'" width="normal" @close="showRuleDialog = false">
      <form id="rule-form" @submit.prevent="handleSaveRule" class="space-y-4">
        <div>
          <label class="input-label">规则名称</label>
          <input v-model="ruleForm.name" type="text" required class="input" placeholder="如: 隧道离线告警" />
        </div>
        <div>
          <label class="input-label">条件</label>
          <select v-model="ruleForm.condition" required class="input">
            <option value="tunnel_down">隧道离线 (tunnel_down)</option>
            <option value="high_latency">高延迟 (high_latency)</option>
            <option value="health_degraded">健康降级 (health_degraded)</option>
            <option value="consecutive_failures">连续失败 (consecutive_failures)</option>
          </select>
        </div>
        <div>
          <label class="input-label">阈值</label>
          <input v-model.number="ruleForm.threshold" type="number" required min="0" class="input" placeholder="如: 3" />
        </div>
        <div>
          <label class="input-label">Webhook URL</label>
          <input v-model="ruleForm.webhook_url" type="url" required class="input" placeholder="https://..." />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="showRuleDialog = false" type="button" class="btn btn-secondary">取消</button>
          <button type="submit" form="rule-form" class="btn btn-primary" :disabled="ruleSaving">
            {{ ruleSaving ? '保存中...' : '保存' }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useAppStore } from '@/stores/app'
import {
  listTunnels, getDashboard, listEvents, listAlertRules,
  createAlertRule, updateAlertRule, deleteAlertRule,
  type VpnTunnel, type DashboardData, type VpnEvent, type AlertRule,
} from '@/api/admin/vpn'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()

// --- State ---
const loading = ref(false)
const dashboard = ref<DashboardData | null>(null)
const tunnels = ref<VpnTunnel[]>([])
const events = ref<VpnEvent[]>([])
const eventsTotal = ref(0)
const eventsPage = ref(1)
const eventsLoading = ref(false)
const alertRules = ref<AlertRule[]>([])
const autoRefreshCountdown = ref(30)

// Event filters
const eventFilterTunnel = ref('')
const eventFilterType = ref('')

// Alert rule dialog
const showRuleDialog = ref(false)
const editingRule = ref<AlertRule | null>(null)
const ruleSaving = ref(false)
const ruleForm = reactive({ name: '', condition: 'tunnel_down', threshold: 1, webhook_url: '' })

// --- Computed ---
const tunnelNames = computed(() => tunnels.value.map(t => t.name))

// --- Helpers ---
const statusBadge = (v: string) =>
  ({ running: 'badge-success', stopped: 'badge-gray', error: 'badge-danger' }[v] || 'badge-gray')
const statusLabel = (v: string) =>
  ({ running: '运行中', stopped: '已停止', error: '错误' }[v] || v)
const healthDot = (v: string) =>
  ({ healthy: 'bg-green-500', unhealthy: 'bg-red-500', degraded: 'bg-yellow-500' }[v] || 'bg-gray-300')

const eventTypeBadge = (t: string) => ({
  connected: 'badge-success',
  disconnected: 'badge-danger',
  failover: 'badge-warning',
  config_switch: 'badge-primary',
  health_changed: 'badge-warning',
}[t] || 'badge-gray')

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const sec = Math.floor(diff / 1000)
  if (sec < 60) return `${sec}秒前`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}分钟前`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}小时前`
  return `${Math.floor(hr / 24)}天前`
}

function eventSummary(details: Record<string, unknown>): string {
  if (!details) return ''
  const parts: string[] = []
  for (const [k, v] of Object.entries(details)) {
    parts.push(`${k}=${typeof v === 'object' ? JSON.stringify(v) : String(v)}`)
  }
  return parts.join(', ')
}

// --- Data Loading ---
async function loadDashboard() {
  try { dashboard.value = await getDashboard() } catch { /* ignore */ }
}

async function loadTunnels() {
  try { tunnels.value = await listTunnels() } catch { /* ignore */ }
}

async function loadEvents(reset = false) {
  if (reset) { eventsPage.value = 1; events.value = [] }
  eventsLoading.value = true
  try {
    const params: Record<string, unknown> = { page: eventsPage.value, page_size: 20 }
    if (eventFilterTunnel.value) params.tunnel_name = eventFilterTunnel.value
    if (eventFilterType.value) params.event_type = eventFilterType.value
    const res = await listEvents(params as any)
    if (reset) { events.value = res.items } else { events.value.push(...res.items) }
    eventsTotal.value = res.total
  } catch (e: any) {
    appStore.showError(e?.message || '加载事件失败')
  } finally {
    eventsLoading.value = false
  }
}

function loadMoreEvents() {
  eventsPage.value++
  loadEvents()
}

async function loadAlertRules() {
  try { alertRules.value = await listAlertRules() } catch { /* ignore */ }
}

async function refreshAll() {
  loading.value = true
  autoRefreshCountdown.value = 30
  await Promise.all([loadDashboard(), loadTunnels(), loadEvents(true), loadAlertRules()])
  loading.value = false
}

// --- Event filter watchers ---
watch([eventFilterTunnel, eventFilterType], () => loadEvents(true))

// --- Alert Rule CRUD ---
function openCreateRule() {
  editingRule.value = null
  ruleForm.name = ''; ruleForm.condition = 'tunnel_down'; ruleForm.threshold = 1; ruleForm.webhook_url = ''
  showRuleDialog.value = true
}

function openEditRule(rule: AlertRule) {
  editingRule.value = rule
  ruleForm.name = rule.name; ruleForm.condition = rule.condition
  ruleForm.threshold = rule.threshold; ruleForm.webhook_url = rule.webhook_url
  showRuleDialog.value = true
}

async function handleSaveRule() {
  ruleSaving.value = true
  try {
    if (editingRule.value) {
      await updateAlertRule(editingRule.value.id, {
        name: ruleForm.name, condition: ruleForm.condition,
        threshold: ruleForm.threshold, webhook_url: ruleForm.webhook_url,
      })
      appStore.showSuccess('规则已更新')
    } else {
      await createAlertRule({
        name: ruleForm.name, condition: ruleForm.condition,
        threshold: ruleForm.threshold, webhook_url: ruleForm.webhook_url,
      })
      appStore.showSuccess('规则已创建')
    }
    showRuleDialog.value = false
    await loadAlertRules()
  } catch (e: any) {
    appStore.showError(e?.message || '保存规则失败')
  } finally {
    ruleSaving.value = false
  }
}

async function toggleRuleEnabled(rule: AlertRule) {
  try {
    await updateAlertRule(rule.id, { enabled: !rule.enabled })
    rule.enabled = !rule.enabled
  } catch (e: any) {
    appStore.showError(e?.message || '切换规则状态失败')
  }
}

async function handleDeleteRule(rule: AlertRule) {
  if (!confirm(`确定要删除告警规则「${rule.name}」吗？`)) return
  try {
    await deleteAlertRule(rule.id)
    appStore.showSuccess('规则已删除')
    await loadAlertRules()
  } catch (e: any) {
    appStore.showError(e?.message || '删除规则失败')
  }
}

// --- Auto-refresh ---
let refreshTimer: ReturnType<typeof setInterval> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  refreshAll()
  refreshTimer = setInterval(() => refreshAll(), 30000)
  countdownTimer = setInterval(() => {
    autoRefreshCountdown.value = Math.max(0, autoRefreshCountdown.value - 1)
  }, 1000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (countdownTimer) clearInterval(countdownTimer)
})
</script>
