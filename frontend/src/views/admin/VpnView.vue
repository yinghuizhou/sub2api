<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <!-- Agent Health Status -->
          <div class="flex items-center gap-2">
            <span :class="['inline-block h-2.5 w-2.5 rounded-full', agentHealthDot]" />
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              Agent: {{ agentHealth?.status || 'offline' }}
              <span v-if="agentHealth && agentHealth.status === 'ok'" class="text-gray-500">
                ({{ agentHealth.tunnels }} 隧道)
              </span>
            </span>
          </div>

          <!-- Tab Switcher -->
          <div class="flex items-center gap-1 rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
            <button @click="activeTab = 'configs'"
              :class="['rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                activeTab === 'configs'
                  ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200']">
              <Icon name="document" size="sm" class="mr-1 inline" />配置文件
            </button>
            <button @click="activeTab = 'tunnels'"
              :class="['rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                activeTab === 'tunnels'
                  ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200']">
              <Icon name="server" size="sm" class="mr-1 inline" />隧道管理
            </button>
          </div>

          <div class="flex flex-1 items-center justify-end gap-2">
            <button @click="refreshAll" :disabled="loading" class="btn btn-secondary" title="刷新">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <!-- ======= Tab: Config Files ======= -->
        <div v-if="activeTab === 'configs'">
          <!-- Upload Area -->
          <div class="border-b border-gray-200 p-4 dark:border-dark-700">
            <div
              class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 p-6 transition-colors dark:border-dark-600"
              :class="{ 'border-primary-400 bg-primary-50 dark:border-primary-500 dark:bg-primary-900/10': isDragging }"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop">
              <Icon name="upload" size="lg" class="mb-2 text-gray-400" />
              <p class="mb-2 text-sm text-gray-600 dark:text-gray-400">
                拖拽 .ovpn 文件到此处，或
                <label class="cursor-pointer font-medium text-primary-600 hover:text-primary-500 dark:text-primary-400">
                  点击选择
                  <input ref="fileInputRef" type="file" accept=".ovpn" multiple class="hidden" @change="handleFileSelect" />
                </label>
              </p>
              <p v-if="uploadStatus" :class="['text-sm', uploadStatus.ok ? 'text-green-600' : 'text-red-600']">
                {{ uploadStatus.message }}
              </p>
              <p v-if="uploading" class="text-sm text-gray-500">上传中...</p>
            </div>
          </div>

          <!-- Config List Table -->
          <DataTable :columns="configColumns" :data="configs" :loading="loading" row-key="name">
            <template #cell-name="{ value }">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
            </template>
            <template #cell-region="{ value }">
              <span class="badge badge-gray">{{ value || '-' }}</span>
            </template>
            <template #cell-server_ip="{ value }">
              <span class="text-sm font-mono">{{ value || '-' }}</span>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center gap-1">
                <button @click="openDeployDialog(row)"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400">
                  <Icon name="play" size="sm" /><span class="text-xs">部署</span>
                </button>
                <button @click="handleDeleteConfig(row)"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                  <Icon name="trash" size="sm" /><span class="text-xs">删除</span>
                </button>
              </div>
            </template>
            <template #empty>
              <EmptyState title="暂无配置文件" description="上传 .ovpn 配置文件以开始使用" />
            </template>
          </DataTable>
        </div>

        <!-- ======= Tab: Tunnel Management ======= -->
        <div v-if="activeTab === 'tunnels'">
          <DataTable :columns="tunnelColumns" :data="tunnels" :loading="loading" row-key="name">
            <template #cell-name="{ value }">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
            </template>
            <template #cell-config_name="{ value }">
              <span class="text-sm">{{ value }}</span>
            </template>
            <template #cell-region="{ value }">
              <span class="badge badge-gray">{{ value || '-' }}</span>
            </template>
            <template #cell-socks_port="{ value }">
              <span class="font-mono text-sm">{{ value }}</span>
            </template>
            <template #cell-status="{ value }">
              <span :class="['badge', tunnelStatusBadge(value)]">{{ tunnelStatusLabel(value) }}</span>
            </template>
            <template #cell-health="{ value }">
              <span :class="['inline-block h-2.5 w-2.5 rounded-full', tunnelHealthDot(value)]"
                :title="value || 'unknown'" />
            </template>
            <template #cell-latency_ms="{ value }">
              <span v-if="typeof value === 'number' && value > 0" class="text-sm text-gray-700 dark:text-gray-300">
                {{ value }}ms
              </span>
              <span v-else class="text-sm text-gray-400">-</span>
            </template>
            <template #cell-exit_ip="{ value }">
              <span class="font-mono text-sm">{{ value || '-' }}</span>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center gap-1">
                <button @click="handleRestartTunnel(row)" :disabled="actionLoading[row.name] === 'restart'"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-blue-900/20 dark:hover:text-blue-400">
                  <Icon :name="actionLoading[row.name] === 'restart' ? 'refresh' : 'sync'" size="sm"
                    :class="actionLoading[row.name] === 'restart' ? 'animate-spin' : ''" />
                  <span class="text-xs">重启</span>
                </button>
                <button @click="handleToggleTunnel(row)"
                  :disabled="actionLoading[row.name] === 'toggle'"
                  :class="['flex flex-col items-center gap-0.5 rounded-lg p-1.5 transition-colors disabled:cursor-not-allowed disabled:opacity-50',
                    row.status === 'running'
                      ? 'text-gray-500 hover:bg-yellow-50 hover:text-yellow-600 dark:hover:bg-yellow-900/20 dark:hover:text-yellow-400'
                      : 'text-gray-500 hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400']">
                  <Icon :name="actionLoading[row.name] === 'toggle' ? 'refresh' : (row.status === 'running' ? 'ban' : 'play')" size="sm"
                    :class="actionLoading[row.name] === 'toggle' ? 'animate-spin' : ''" />
                  <span class="text-xs">{{ row.status === 'running' ? '停止' : '启动' }}</span>
                </button>
                <button @click="handleDeleteTunnel(row)"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                  <Icon name="trash" size="sm" /><span class="text-xs">删除</span>
                </button>
              </div>
            </template>
            <template #empty>
              <EmptyState title="暂无隧道" description="从配置文件页签部署隧道" />
            </template>
          </DataTable>
        </div>
      </template>
    </TablePageLayout>

    <!-- Deploy Dialog -->
    <BaseDialog :show="showDeployDialog" title="部署隧道" width="normal" @close="showDeployDialog = false">
      <form id="deploy-form" @submit.prevent="handleDeploy" class="space-y-4">
        <div>
          <label class="input-label">配置文件</label>
          <input type="text" :value="deployForm.config_name" disabled class="input bg-gray-50 dark:bg-dark-800" />
        </div>
        <div>
          <label class="input-label">隧道名称</label>
          <input v-model="deployForm.name" type="text" required class="input" placeholder="tunnel-hk-01" />
        </div>
        <div>
          <label class="input-label">SOCKS 端口（留空自动分配）</label>
          <input v-model.number="deployForm.socks_port" type="number" min="1024" max="65535" class="input" placeholder="自动分配" />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="showDeployDialog = false" type="button" class="btn btn-secondary">取消</button>
          <button type="submit" form="deploy-form" :disabled="deploying" class="btn btn-primary">
            {{ deploying ? '部署中...' : '部署' }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useAppStore } from '@/stores/app'
import {
  getAgentHealth, listTunnels, createTunnel, removeTunnel,
  restartTunnel, startTunnel, stopTunnel, uploadConfigs, listConfigs, deleteConfig,
  type VpnTunnel, type VpnOvpnConfig, type AgentHealth, type CreateTunnelInput,
} from '@/api/admin/vpn'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()

// --- State ---
const activeTab = ref<'configs' | 'tunnels'>('configs')
const loading = ref(false)
const agentHealth = ref<AgentHealth | null>(null)
const configs = ref<VpnOvpnConfig[]>([])
const tunnels = ref<VpnTunnel[]>([])

// Upload
const fileInputRef = ref<HTMLInputElement | null>(null)
const isDragging = ref(false)
const uploading = ref(false)
const uploadStatus = ref<{ ok: boolean; message: string } | null>(null)

// Deploy dialog
const showDeployDialog = ref(false)
const deploying = ref(false)
const deployForm = reactive<CreateTunnelInput>({
  name: '',
  config_name: '',
  socks_port: undefined as unknown as number,
})

// Tunnel action loading
const actionLoading = reactive<Record<string, string>>({})

// --- Columns ---
const configColumns = computed<Column[]>(() => [
  { key: 'name', label: '文件名', sortable: true },
  { key: 'region', label: '地区', sortable: true },
  { key: 'server_ip', label: '服务器 IP', sortable: false },
  { key: 'actions', label: '操作', sortable: false },
])

const tunnelColumns = computed<Column[]>(() => [
  { key: 'name', label: '名称', sortable: true },
  { key: 'config_name', label: '配置', sortable: true },
  { key: 'region', label: '地区', sortable: true },
  { key: 'socks_port', label: 'SOCKS 端口', sortable: false },
  { key: 'status', label: '状态', sortable: true },
  { key: 'health', label: '健康', sortable: false },
  { key: 'latency_ms', label: '延迟', sortable: false },
  { key: 'exit_ip', label: '出口 IP', sortable: false },
  { key: 'actions', label: '操作', sortable: false },
])

// --- Helpers ---
const agentHealthDot = computed(() => {
  const s = agentHealth.value?.status
  if (s === 'ok') return 'bg-green-500'
  if (s === 'error') return 'bg-red-500'
  return 'bg-gray-300'
})

const tunnelStatusBadge = (v: string) =>
  ({ running: 'badge-success', stopped: 'badge-gray', error: 'badge-danger' }[v] || 'badge-gray')
const tunnelStatusLabel = (v: string) =>
  ({ running: '运行中', stopped: '已停止', error: '错误' }[v] || v)
const tunnelHealthDot = (v: string) =>
  ({ healthy: 'bg-green-500', unhealthy: 'bg-red-500' }[v] || 'bg-gray-300')

// --- Data Loading ---
const loadHealth = async () => {
  try {
    agentHealth.value = await getAgentHealth()
  } catch {
    agentHealth.value = null
  }
}

const loadConfigs = async () => {
  try {
    configs.value = await listConfigs()
  } catch (e: any) {
    appStore.showError(e?.message || '加载配置列表失败')
  }
}

const loadTunnels = async () => {
  try {
    tunnels.value = await listTunnels()
  } catch (e: any) {
    appStore.showError(e?.message || '加载隧道列表失败')
  }
}

const refreshAll = async () => {
  loading.value = true
  await Promise.all([loadHealth(), loadConfigs(), loadTunnels()])
  loading.value = false
}

// --- File Upload ---
const handleFileSelect = (e: Event) => {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    doUpload(Array.from(input.files))
    input.value = ''
  }
}

const handleDrop = (e: DragEvent) => {
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    const ovpnFiles = Array.from(files).filter(f => f.name.endsWith('.ovpn'))
    if (ovpnFiles.length === 0) {
      uploadStatus.value = { ok: false, message: '请选择 .ovpn 文件' }
      return
    }
    doUpload(ovpnFiles)
  }
}

const doUpload = async (files: File[]) => {
  uploading.value = true
  uploadStatus.value = null
  try {
    const result = await uploadConfigs(files)
    uploadStatus.value = { ok: true, message: `成功上传 ${result.count} 个配置文件` }
    appStore.showSuccess(`上传成功: ${result.count} 个文件`)
    await loadConfigs()
  } catch (e: any) {
    uploadStatus.value = { ok: false, message: e?.message || '上传失败' }
    appStore.showError(e?.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

// --- Config Actions ---
const handleDeleteConfig = async (config: VpnOvpnConfig) => {
  if (!confirm(`确定要删除配置「${config.name}」吗？`)) return
  try {
    await deleteConfig(config.name)
    appStore.showSuccess('配置已删除')
    await loadConfigs()
  } catch (e: any) {
    appStore.showError(e?.message || '删除失败')
  }
}

// --- Deploy Dialog ---
const openDeployDialog = (config: VpnOvpnConfig) => {
  deployForm.config_name = config.name
  deployForm.name = config.name.replace(/\.ovpn$/, '').replace(/[^a-zA-Z0-9_-]/g, '-')
  deployForm.socks_port = undefined as unknown as number
  showDeployDialog.value = true
}

const handleDeploy = async () => {
  if (!deployForm.name.trim()) {
    appStore.showError('请输入隧道名称')
    return
  }
  deploying.value = true
  try {
    const input: CreateTunnelInput = {
      name: deployForm.name,
      config_name: deployForm.config_name,
    }
    if (deployForm.socks_port) input.socks_port = deployForm.socks_port
    await createTunnel(input)
    appStore.showSuccess('隧道已部署')
    showDeployDialog.value = false
    activeTab.value = 'tunnels'
    await loadTunnels()
    await loadHealth()
  } catch (e: any) {
    appStore.showError(e?.message || '部署失败')
  } finally {
    deploying.value = false
  }
}

// --- Tunnel Actions ---
const handleRestartTunnel = async (tunnel: VpnTunnel) => {
  actionLoading[tunnel.name] = 'restart'
  try {
    await restartTunnel(tunnel.name)
    appStore.showSuccess(`隧道 ${tunnel.name} 已重启`)
    await loadTunnels()
  } catch (e: any) {
    appStore.showError(e?.message || '重启失败')
  } finally {
    delete actionLoading[tunnel.name]
  }
}

const handleToggleTunnel = async (tunnel: VpnTunnel) => {
  actionLoading[tunnel.name] = 'toggle'
  try {
    if (tunnel.status === 'running') {
      await stopTunnel(tunnel.name)
      appStore.showSuccess(`隧道 ${tunnel.name} 已停止`)
    } else {
      await startTunnel(tunnel.name)
      appStore.showSuccess(`隧道 ${tunnel.name} 已启动`)
    }
    await loadTunnels()
  } catch (e: any) {
    appStore.showError(e?.message || '操作失败')
  } finally {
    delete actionLoading[tunnel.name]
  }
}

const handleDeleteTunnel = async (tunnel: VpnTunnel) => {
  if (!confirm(`确定要删除隧道「${tunnel.name}」吗？此操作将终止 VPN 连接。`)) return
  try {
    await removeTunnel(tunnel.name)
    appStore.showSuccess('隧道已删除')
    await Promise.all([loadTunnels(), loadHealth()])
  } catch (e: any) {
    appStore.showError(e?.message || '删除失败')
  }
}

// --- Init ---
onMounted(refreshAll)
</script>
