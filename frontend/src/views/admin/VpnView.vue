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
            <button @click="activeTab = 'certs'"
              :class="['rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                activeTab === 'certs'
                  ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200']">
              <Icon name="key" size="sm" class="mr-1 inline" />证书管理
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

          <!-- Config List (compact) -->
          <div v-if="loading" class="flex items-center justify-center py-8 text-gray-400">
            <Icon name="refresh" size="md" class="animate-spin" />
          </div>
          <div v-else-if="configs.length === 0" class="py-8">
            <EmptyState title="暂无配置文件" description="上传 .ovpn 配置文件以开始使用" />
          </div>
          <div v-else class="max-h-[60vh] overflow-y-auto">
            <div class="sticky top-0 z-10 flex items-center gap-2 border-b px-4 py-2"
              :class="selectedConfigs.size > 0
                ? 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20'
                : 'border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800'">
              <input type="checkbox" :checked="allConfigsSelected" @change="toggleAllConfigs"
                class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600" />
              <span v-if="selectedConfigs.size > 0" class="text-sm font-medium text-blue-700 dark:text-blue-300">
                已选 {{ selectedConfigs.size }} 项
              </span>
              <span v-else class="text-sm text-gray-500 dark:text-gray-400">{{ configs.length }} 个配置</span>
              <div class="flex-1" />
              <button v-if="selectedConfigs.size > 0" @click="batchDeleteConfigs" :disabled="batchLoading"
                class="rounded px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-100 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-900/20">
                批量删除
              </button>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <div v-for="c in configs" :key="c.name"
                class="flex items-center gap-3 px-4 py-2 hover:bg-gray-50 dark:hover:bg-dark-800/50">
                <input type="checkbox" :checked="selectedConfigs.has(c.name)" @change="toggleConfigSel(c.name)"
                  class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600" />
                <Icon name="document" size="sm" class="shrink-0 text-gray-400" />
                <span class="min-w-0 flex-1 truncate text-sm font-medium text-gray-900 dark:text-white" :title="c.name">{{ c.name }}</span>
                <span v-if="c.region" class="badge badge-gray shrink-0 text-xs">{{ c.region }}</span>
                <span class="hidden shrink-0 font-mono text-xs text-gray-500 sm:inline">{{ c.server_ip || '' }}</span>
                <div class="flex shrink-0 items-center gap-0.5">
                  <button @click="openDeployDialog(c)" title="部署"
                    class="rounded p-1 text-gray-400 transition-colors hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400">
                    <Icon name="play" size="sm" />
                  </button>
                  <button @click="handleDeleteConfig(c)" title="删除"
                    class="rounded p-1 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ======= Tab: Tunnel Management ======= -->
        <div v-if="activeTab === 'tunnels'">
          <div v-if="selectedTunnels.size > 0" class="flex items-center gap-2 border-b border-blue-200 bg-blue-50 px-4 py-2 dark:border-blue-800 dark:bg-blue-900/20">
            <span class="text-sm font-medium text-blue-700 dark:text-blue-300">已选 {{ selectedTunnels.size }} 项</span>
            <div class="flex-1" />
            <button @click="batchRestartTunnels" :disabled="batchLoading"
              class="rounded px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-100 disabled:opacity-50 dark:text-blue-400 dark:hover:bg-blue-800/30">
              批量重启
            </button>
            <button @click="batchStopTunnels" :disabled="batchLoading"
              class="rounded px-2 py-1 text-xs font-medium text-yellow-600 hover:bg-yellow-100 disabled:opacity-50 dark:text-yellow-400 dark:hover:bg-yellow-900/20">
              批量停止
            </button>
            <button @click="batchDeleteTunnels" :disabled="batchLoading"
              class="rounded px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-100 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-900/20">
              批量删除
            </button>
          </div>
          <DataTable :columns="tunnelColumns" :data="tunnels" :loading="loading" row-key="name">
            <template #header-select>
              <input type="checkbox" :checked="allTunnelsSelected" @change="toggleAllTunnels"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600" />
            </template>
            <template #cell-select="{ row }">
              <input type="checkbox" :checked="selectedTunnels.has(row.name)" @change="toggleTunnelSel(row.name)"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600" />
            </template>
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
            <template #cell-cert_id="{ row }">
              <span v-if="row.cert_id" class="badge badge-blue text-xs">{{ certLabel(row.cert_id) }}</span>
              <span v-else class="text-xs text-gray-400">内嵌</span>
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

        <!-- ======= Tab: Certificate Management ======= -->
        <div v-if="activeTab === 'certs'">
          <!-- Import Area -->
          <div class="border-b border-gray-200 p-4 dark:border-dark-700">
            <div class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 p-6 dark:border-dark-600"
              :class="{ 'border-primary-400 bg-primary-50 dark:border-primary-500 dark:bg-primary-900/10': isCertDragging }"
              @dragover.prevent="isCertDragging = true" @dragleave.prevent="isCertDragging = false"
              @drop.prevent="handleCertDrop">
              <Icon name="key" size="lg" class="mb-2 text-gray-400" />
              <p class="mb-2 text-sm text-gray-600 dark:text-gray-400">
                拖拽 .ovpn 文件提取证书，或
                <label class="cursor-pointer font-medium text-primary-600 hover:text-primary-500 dark:text-primary-400">
                  点击选择
                  <input ref="certFileRef" type="file" accept=".ovpn" class="hidden" @change="handleCertFileSelect" />
                </label>
              </p>
              <p v-if="certImportStatus" :class="['text-sm', certImportStatus.ok ? 'text-green-600' : 'text-red-600']">
                {{ certImportStatus.message }}
              </p>
              <p v-if="certImporting" class="text-sm text-gray-500">导入中...</p>
            </div>
          </div>
          <!-- Cert List -->
          <div v-if="loading" class="flex items-center justify-center py-8 text-gray-400">
            <Icon name="refresh" size="md" class="animate-spin" />
          </div>
          <div v-else-if="certs.length === 0" class="py-8">
            <EmptyState title="暂无证书" description="上传 .ovpn 文件提取客户端证书" />
          </div>
          <div v-else>
            <div class="sticky top-0 z-10 flex items-center gap-2 border-b px-4 py-2"
              :class="selectedCerts.size > 0
                ? 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20'
                : 'border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800'">
              <input type="checkbox" :checked="allCertsSelected" @change="toggleAllCerts"
                class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600" />
              <span v-if="selectedCerts.size > 0" class="text-sm font-medium text-blue-700 dark:text-blue-300">
                已选 {{ selectedCerts.size }} 项
              </span>
              <span v-else class="text-sm text-gray-500 dark:text-gray-400">{{ certs.length }} 个证书</span>
              <div class="flex-1" />
              <button v-if="selectedCerts.size > 0" @click="batchDeleteCerts" :disabled="batchLoading"
                class="rounded px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-100 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-900/20">
                批量删除
              </button>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <div v-for="cert in certs" :key="cert.id"
                class="flex items-center gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-dark-800/50">
                <input type="checkbox" :checked="selectedCerts.has(cert.id)" @change="toggleCertSel(cert.id)"
                  class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600" />
                <Icon name="key" size="sm" class="shrink-0 text-gray-400" />
                <div class="min-w-0 flex-1">
                  <div class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ cert.label }}</div>
                  <div class="truncate font-mono text-xs text-gray-400">{{ cert.id }}</div>
                </div>
                <span v-if="cert.in_use" class="badge badge-blue shrink-0 text-xs">{{ cert.active_tunnel }}</span>
                <span v-else class="badge badge-gray shrink-0 text-xs">空闲</span>
                <button @click="handleDeleteCert(cert)" :disabled="cert.in_use" title="删除"
                  class="rounded p-1 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-30 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>
    </TablePageLayout>

    <!-- Deploy Dialog -->
    <BaseDialog :show="showDeployDialog" title="部署隧道" width="normal" @close="!deploying && (showDeployDialog = false)">
      <form v-if="!deploying && !deployResult" id="deploy-form" @submit.prevent="handleDeploy" class="space-y-4">
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
        <div v-if="freeCerts.length > 0">
          <label class="input-label">证书（可选，多隧道需不同证书）</label>
          <select v-model="deployForm.cert_id" class="input">
            <option value="">使用配置内嵌证书</option>
            <option v-for="c in freeCerts" :key="c.id" :value="c.id">{{ c.label }}</option>
          </select>
        </div>
      </form>
      <!-- Deploy Progress Log -->
      <div v-if="deploying || deployResult" class="space-y-3">
        <div class="rounded-lg bg-gray-900 p-3 font-mono text-xs text-gray-300 dark:bg-dark-900">
          <div v-for="(line, i) in deployLog" :key="i" class="flex gap-2">
            <span class="shrink-0 text-gray-600">{{ line.time }}</span>
            <span :class="line.color">{{ line.text }}</span>
          </div>
          <div v-if="deploying" class="mt-1 flex items-center gap-2 text-yellow-400">
            <span class="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-yellow-400" />
            <span>{{ deployPhase }}</span>
          </div>
        </div>
        <!-- Result summary -->
        <div v-if="deployResult" :class="['rounded-lg border p-3 text-sm',
          deployResult.ok
            ? 'border-green-200 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-900/20 dark:text-green-300'
            : 'border-red-200 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300']">
          <template v-if="deployResult.ok">
            <div class="font-medium">部署成功</div>
            <div class="mt-1 text-xs opacity-80">SOCKS5: 127.0.0.1:{{ deployResult.port }} · IP: {{ deployResult.localIp }}</div>
          </template>
          <template v-else>
            <div class="font-medium">部署失败</div>
            <div class="mt-1 text-xs opacity-80">{{ deployResult.error }}</div>
          </template>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button v-if="!deploying" @click="closeDeploy" type="button" class="btn btn-secondary">
            {{ deployResult ? '关闭' : '取消' }}
          </button>
          <button v-if="!deploying && !deployResult" type="submit" form="deploy-form" class="btn btn-primary">
            部署
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useAppStore } from '@/stores/app'
import {
  getAgentHealth, listTunnels, createTunnel, removeTunnel,
  restartTunnel, startTunnel, stopTunnel, uploadConfigs, listConfigs, deleteConfig,
  listCertificates, importCertificate, deleteCertificate,
  type VpnTunnel, type VpnOvpnConfig, type AgentHealth, type CreateTunnelInput, type VpnCertificate,
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
const activeTab = ref<'configs' | 'tunnels' | 'certs'>('configs')
const loading = ref(false)
const agentHealth = ref<AgentHealth | null>(null)
const configs = ref<VpnOvpnConfig[]>([])
const tunnels = ref<VpnTunnel[]>([])
const certs = ref<VpnCertificate[]>([])

// Selection
const selectedConfigs = ref(new Set<string>())
const selectedTunnels = ref(new Set<string>())
const selectedCerts = ref(new Set<string>())
const batchLoading = ref(false)

// Upload
const fileInputRef = ref<HTMLInputElement | null>(null)
const isDragging = ref(false)
const uploading = ref(false)
const uploadStatus = ref<{ ok: boolean; message: string } | null>(null)

// Cert import
const certFileRef = ref<HTMLInputElement | null>(null)
const isCertDragging = ref(false)
const certImporting = ref(false)
const certImportStatus = ref<{ ok: boolean; message: string } | null>(null)

// Deploy dialog
const showDeployDialog = ref(false)
const deploying = ref(false)
const deployForm = reactive<CreateTunnelInput>({
  name: '',
  config_name: '',
  socks_port: undefined as unknown as number,
})
const deployLog = ref<{ time: string; text: string; color: string }[]>([])
const deployPhase = ref('')
const deployResult = ref<{ ok: boolean; port?: number; localIp?: string; error?: string } | null>(null)
const deployTimer = ref<ReturnType<typeof setInterval> | null>(null)
const deployStartTime = ref(0)

const logLine = (text: string, color = 'text-gray-300') => {
  const elapsed = ((Date.now() - deployStartTime.value) / 1000).toFixed(1)
  deployLog.value.push({ time: `[${elapsed}s]`, text, color })
}

// Tunnel action loading
const actionLoading = reactive<Record<string, string>>({})

// --- Columns ---
const tunnelColumns = computed<Column[]>(() => [
  { key: 'select', label: '', sortable: false },
  { key: 'name', label: '名称', sortable: true },
  { key: 'config_name', label: '配置', sortable: true },
  { key: 'region', label: '地区', sortable: true },
  { key: 'socks_port', label: 'SOCKS 端口', sortable: false },
  { key: 'cert_id', label: '证书', sortable: false },
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

const freeCerts = computed(() => certs.value.filter(c => !c.in_use))
const certLabel = (id: string) => certs.value.find(c => c.id === id)?.label || id.slice(0, 8)

// --- Selection ---
const allConfigsSelected = computed(() => configs.value.length > 0 && selectedConfigs.value.size === configs.value.length)
const allTunnelsSelected = computed(() => tunnels.value.length > 0 && selectedTunnels.value.size === tunnels.value.length)
const allCertsSelected = computed(() => certs.value.length > 0 && selectedCerts.value.size === certs.value.length)

const toggleConfigSel = (name: string) => {
  const s = new Set(selectedConfigs.value)
  s.has(name) ? s.delete(name) : s.add(name)
  selectedConfigs.value = s
}
const toggleAllConfigs = () => {
  selectedConfigs.value = allConfigsSelected.value ? new Set() : new Set(configs.value.map(c => c.name))
}
const toggleTunnelSel = (name: string) => {
  const s = new Set(selectedTunnels.value)
  s.has(name) ? s.delete(name) : s.add(name)
  selectedTunnels.value = s
}
const toggleAllTunnels = () => {
  selectedTunnels.value = allTunnelsSelected.value ? new Set() : new Set(tunnels.value.map(t => t.name))
}
const toggleCertSel = (id: string) => {
  const s = new Set(selectedCerts.value)
  s.has(id) ? s.delete(id) : s.add(id)
  selectedCerts.value = s
}
const toggleAllCerts = () => {
  selectedCerts.value = allCertsSelected.value ? new Set() : new Set(certs.value.map(c => c.id))
}

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

const loadCerts = async () => {
  try {
    certs.value = await listCertificates()
  } catch (e: any) {
    appStore.showError(e?.message || '加载证书列表失败')
  }
}

const refreshAll = async () => {
  loading.value = true
  await Promise.all([loadHealth(), loadConfigs(), loadTunnels(), loadCerts()])
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
  deployForm.cert_id = ''
  deployLog.value = []
  deployResult.value = null
  deployPhase.value = ''
  showDeployDialog.value = true
}

const closeDeploy = () => {
  if (deployTimer.value) { clearInterval(deployTimer.value); deployTimer.value = null }
  showDeployDialog.value = false
  if (deployResult.value?.ok) {
    activeTab.value = 'tunnels'
    loadTunnels()
    loadHealth()
  }
}

const phases = [
  { at: 0, text: '正在读取配置文件...' },
  { at: 2, text: '正在启动 OpenVPN 进程...' },
  { at: 4, text: '正在建立 TCP 连接...' },
  { at: 8, text: '正在进行 TLS 握手...' },
  { at: 15, text: '正在等待服务器推送路由...' },
  { at: 25, text: '正在初始化隧道设备...' },
  { at: 40, text: '正在启动 SOCKS5 代理...' },
  { at: 60, text: '仍在等待，某些服务器较慢...' },
]

const handleDeploy = async () => {
  if (!deployForm.name.trim()) {
    appStore.showError('请输入隧道名称')
    return
  }
  deploying.value = true
  deployLog.value = []
  deployResult.value = null
  deployStartTime.value = Date.now()

  logLine(`开始部署隧道 "${deployForm.name}"`, 'text-blue-400')
  logLine(`配置: ${deployForm.config_name}`, 'text-gray-400')

  // Animated phase updates
  let phaseIdx = 0
  deployPhase.value = phases[0].text
  deployTimer.value = setInterval(() => {
    const elapsed = (Date.now() - deployStartTime.value) / 1000
    while (phaseIdx < phases.length - 1 && elapsed >= phases[phaseIdx + 1].at) {
      phaseIdx++
      logLine(phases[phaseIdx].text, 'text-gray-400')
      deployPhase.value = phases[phaseIdx].text
    }
  }, 1000)

  try {
    const input: CreateTunnelInput = {
      name: deployForm.name,
      config_name: deployForm.config_name,
    }
    if (deployForm.socks_port) input.socks_port = deployForm.socks_port
    if (deployForm.cert_id) input.cert_id = deployForm.cert_id
    const tunnel = await createTunnel(input)
    logLine('隧道连接成功!', 'text-green-400')
    logLine(`分配端口: ${tunnel.socks_port}, 本地 IP: ${tunnel.local_ip}`, 'text-green-300')
    deployResult.value = { ok: true, port: tunnel.socks_port, localIp: tunnel.local_ip }
  } catch (e: any) {
    const msg = e?.message || '部署失败'
    logLine(`错误: ${msg}`, 'text-red-400')
    deployResult.value = { ok: false, error: msg }
  } finally {
    if (deployTimer.value) { clearInterval(deployTimer.value); deployTimer.value = null }
    deploying.value = false
    deployPhase.value = ''
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

// --- Certificate Actions ---
const handleCertFileSelect = (e: Event) => {
  const input = e.target as HTMLInputElement
  if (input.files?.[0]) { doCertImport(input.files[0]); input.value = '' }
}

const handleCertDrop = (e: DragEvent) => {
  isCertDragging.value = false
  const file = Array.from(e.dataTransfer?.files || []).find(f => f.name.endsWith('.ovpn'))
  if (file) doCertImport(file)
  else certImportStatus.value = { ok: false, message: '请选择 .ovpn 文件' }
}

const doCertImport = async (file: File) => {
  certImporting.value = true
  certImportStatus.value = null
  try {
    const text = await file.text()
    await importCertificate({ label: file.name.replace(/\.ovpn$/, ''), ovpn_data: text })
    certImportStatus.value = { ok: true, message: '证书导入成功' }
    await loadCerts()
  } catch (e: any) {
    certImportStatus.value = { ok: false, message: e?.message || '导入失败' }
  } finally {
    certImporting.value = false
  }
}

const handleDeleteCert = async (cert: VpnCertificate) => {
  if (!confirm(`确定要删除证书「${cert.label}」吗？`)) return
  try {
    await deleteCertificate(cert.id)
    appStore.showSuccess('证书已删除')
    await loadCerts()
  } catch (e: any) {
    appStore.showError(e?.message || '删除失败')
  }
}

// --- Init ---
onMounted(refreshAll)

// Clear selections on data reload
watch(configs, () => { selectedConfigs.value = new Set() })
watch(tunnels, () => { selectedTunnels.value = new Set() })
watch(certs, () => { selectedCerts.value = new Set() })

// --- Batch Actions ---
const batchDeleteConfigs = async () => {
  const names = [...selectedConfigs.value]
  if (!confirm(`确定要删除选中的 ${names.length} 个配置文件吗？`)) return
  batchLoading.value = true
  let ok = 0, fail = 0
  for (const name of names) {
    try { await deleteConfig(name); ok++ } catch { fail++ }
  }
  batchLoading.value = false
  selectedConfigs.value = new Set()
  if (ok) appStore.showSuccess(`成功删除 ${ok} 个配置`)
  if (fail) appStore.showError(`${fail} 个删除失败`)
  await loadConfigs()
}

const batchRestartTunnels = async () => {
  const names = [...selectedTunnels.value]
  if (!confirm(`确定要重启选中的 ${names.length} 个隧道吗？`)) return
  batchLoading.value = true
  let ok = 0, fail = 0
  for (const name of names) {
    try { await restartTunnel(name); ok++ } catch { fail++ }
  }
  batchLoading.value = false
  selectedTunnels.value = new Set()
  if (ok) appStore.showSuccess(`成功重启 ${ok} 个隧道`)
  if (fail) appStore.showError(`${fail} 个重启失败`)
  await loadTunnels()
}

const batchStopTunnels = async () => {
  const names = [...selectedTunnels.value]
  if (!confirm(`确定要停止选中的 ${names.length} 个隧道吗？`)) return
  batchLoading.value = true
  let ok = 0, fail = 0
  for (const name of names) {
    try { await stopTunnel(name); ok++ } catch { fail++ }
  }
  batchLoading.value = false
  selectedTunnels.value = new Set()
  if (ok) appStore.showSuccess(`成功停止 ${ok} 个隧道`)
  if (fail) appStore.showError(`${fail} 个停止失败`)
  await loadTunnels()
}

const batchDeleteTunnels = async () => {
  const names = [...selectedTunnels.value]
  if (!confirm(`确定要删除选中的 ${names.length} 个隧道吗？此操作将终止所有选中隧道的 VPN 连接。`)) return
  batchLoading.value = true
  let ok = 0, fail = 0
  for (const name of names) {
    try { await removeTunnel(name); ok++ } catch { fail++ }
  }
  batchLoading.value = false
  selectedTunnels.value = new Set()
  if (ok) appStore.showSuccess(`成功删除 ${ok} 个隧道`)
  if (fail) appStore.showError(`${fail} 个删除失败`)
  await Promise.all([loadTunnels(), loadHealth()])
}

const batchDeleteCerts = async () => {
  const ids = [...selectedCerts.value]
  const inUse = ids.filter(id => certs.value.find(c => c.id === id)?.in_use)
  if (inUse.length > 0) {
    appStore.showError(`${inUse.length} 个证书正在使用中，无法删除`)
    return
  }
  if (!confirm(`确定要删除选中的 ${ids.length} 个证书吗？`)) return
  batchLoading.value = true
  let ok = 0, fail = 0
  for (const id of ids) {
    try { await deleteCertificate(id); ok++ } catch { fail++ }
  }
  batchLoading.value = false
  selectedCerts.value = new Set()
  if (ok) appStore.showSuccess(`成功删除 ${ok} 个证书`)
  if (fail) appStore.showError(`${fail} 个删除失败`)
  await loadCerts()
}
</script>
