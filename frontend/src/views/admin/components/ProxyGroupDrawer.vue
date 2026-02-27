<template>
  <BaseDialog
    :show="show"
    :title="`${t('admin.proxies.groupDrawer.title')} - ${groupName}`"
    width="wide"
    @close="emit('close')"
  >
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent" />
    </div>

    <div v-else-if="summary" class="space-y-6">
      <!-- Stats cards -->
      <div class="grid grid-cols-3 gap-4">
        <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.totalProxies') }}</div>
          <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ summary.total_proxies }}</div>
        </div>
        <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.healthyProxies') }}</div>
          <div class="text-2xl font-bold text-green-600">{{ summary.healthy_count }}</div>
        </div>
        <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.totalAccounts') }}</div>
          <div class="text-2xl font-bold text-blue-600">{{ summary.total_accounts }}</div>
        </div>
      </div>

      <!-- Proxy list with assignment counts -->
      <div>
        <div class="mb-2 flex items-center justify-between">
          <h4 class="text-sm font-semibold text-gray-700 dark:text-dark-300">
            {{ t('admin.proxies.groupDrawer.proxyName') }}
          </h4>
          <button
            @click="handleDistribute"
            :disabled="distributing"
            class="btn btn-primary btn-sm"
          >
            {{ distributing ? '...' : t('admin.proxies.groupDrawer.distribute') }}
          </button>
        </div>
        <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-2 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.proxyName') }}</th>
                <th class="px-4 py-2 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.region') }}</th>
                <th class="px-4 py-2 text-center font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.healthStatus') }}</th>
                <th class="px-4 py-2 text-center font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.assignmentCount') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
              <tr v-for="proxy in summary.proxies" :key="proxy.id">
                <td class="px-4 py-2 font-medium text-gray-900 dark:text-white">{{ proxy.name }}</td>
                <td class="px-4 py-2 text-gray-600 dark:text-dark-300">{{ proxy.region || '-' }}</td>
                <td class="px-4 py-2 text-center">
                  <span
                    :class="proxy.health_status === 'healthy'
                      ? 'badge badge-success'
                      : proxy.health_status === 'degraded'
                        ? 'badge badge-warning'
                        : 'badge badge-danger'"
                  >
                    {{ proxy.health_status === 'healthy'
                      ? t('admin.proxies.groupDrawer.healthy')
                      : proxy.health_status === 'degraded'
                        ? t('admin.proxies.groupDrawer.degraded')
                        : t('admin.proxies.groupDrawer.unhealthy') }}
                  </span>
                </td>
                <td class="px-4 py-2 text-center font-mono text-gray-900 dark:text-white">
                  {{ proxy.assignment_count }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Assignment details -->
      <div>
        <h4 class="mb-2 text-sm font-semibold text-gray-700 dark:text-dark-300">
          {{ t('admin.proxies.groupDrawer.assignments') }}
        </h4>
        <div v-if="assignments.length === 0" class="rounded-lg border border-dashed border-gray-300 p-8 text-center dark:border-dark-600">
          <p class="text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.noAssignments') }}</p>
          <p class="mt-1 text-sm text-gray-400 dark:text-dark-500">{{ t('admin.proxies.groupDrawer.noAssignmentsHint') }}</p>
        </div>
        <div v-else class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-2 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.accountName') }}</th>
                <th class="px-4 py-2 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.platform') }}</th>
                <th class="px-4 py-2 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.proxyNode') }}</th>
                <th class="px-4 py-2 text-center font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.assignedBy') }}</th>
                <th class="px-4 py-2 text-center font-medium text-gray-500 dark:text-dark-400">{{ t('admin.proxies.groupDrawer.reassign') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
              <tr v-for="a in assignments" :key="a.account_id">
                <td class="px-4 py-2 font-medium text-gray-900 dark:text-white">{{ a.account_name }}</td>
                <td class="px-4 py-2 text-gray-600 dark:text-dark-300">{{ a.platform }}</td>
                <td class="px-4 py-2 text-gray-600 dark:text-dark-300">{{ a.proxy_name }}</td>
                <td class="px-4 py-2 text-center">
                  <span :class="a.assigned_by === 'admin' ? 'badge badge-warning' : 'badge badge-info'">
                    {{ a.assigned_by === 'admin'
                      ? t('admin.proxies.groupDrawer.admin')
                      : t('admin.proxies.groupDrawer.auto') }}
                  </span>
                </td>
                <td class="px-4 py-2 text-center">
                  <select
                    class="input input-sm w-32"
                    :value="a.proxy_id"
                    :disabled="reassigningAccountId === a.account_id"
                    :aria-label="t('admin.proxies.groupDrawer.reassign')"
                    @change="handleReassign(a.account_id, Number(($event.target as HTMLSelectElement).value), $event)"
                  >
                    <option v-for="p in summary.proxies" :key="p.id" :value="p.id">
                      {{ p.name }}
                    </option>
                  </select>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import {
  getGroupSummary,
  getGroupAssignments,
  distributeAccounts,
  reassignAccount,
  type ProxyGroupSummary,
  type ProxyAssignmentDetail
} from '@/api/admin/proxies'

const { t } = useI18n()
const appStore = useAppStore()

interface Props {
  show: boolean
  groupName: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'updated'): void
}>()

const loading = ref(false)
const distributing = ref(false)
const reassigningAccountId = ref<number | null>(null)
const summary = ref<ProxyGroupSummary | null>(null)
const assignments = ref<ProxyAssignmentDetail[]>([])

async function loadData() {
  if (!props.groupName) return
  loading.value = true
  try {
    const [s, a] = await Promise.all([
      getGroupSummary(props.groupName),
      getGroupAssignments(props.groupName)
    ])
    summary.value = s
    assignments.value = a
  } catch (e: any) {
    console.error('Load group data failed:', e)
    appStore.showError(e?.message || t('admin.proxies.groupDrawer.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleDistribute() {
  if (!confirm(t('admin.proxies.groupDrawer.distributeConfirm'))) return
  distributing.value = true
  try {
    const result = await distributeAccounts(props.groupName)
    appStore.showSuccess(t('admin.proxies.groupDrawer.distributeSuccess', {
      assigned: result.assigned,
      skipped: result.skipped
    }))
    await loadData()
    emit('updated')
  } catch (e: any) {
    console.error('Distribute failed:', e)
    appStore.showError(e?.message || t('admin.proxies.groupDrawer.distributeFailed'))
  } finally {
    distributing.value = false
  }
}

async function handleReassign(accountId: number, newProxyId: number, event: Event) {
  const selectEl = event.target as HTMLSelectElement
  const current = assignments.value.find(a => a.account_id === accountId)
  if (current && current.proxy_id === newProxyId) return
  if (reassigningAccountId.value !== null) return

  const newProxyName = summary.value?.proxies.find(p => p.id === newProxyId)?.name ?? String(newProxyId)
  if (!confirm(t('admin.proxies.groupDrawer.reassignConfirm', {
    account: current?.account_name ?? String(accountId),
    proxy: newProxyName
  }))) {
    selectEl.value = String(current?.proxy_id ?? '')
    return
  }

  reassigningAccountId.value = accountId
  try {
    await reassignAccount(accountId, newProxyId)
    appStore.showSuccess(t('admin.proxies.groupDrawer.reassignSuccess'))
    await loadData()
    emit('updated')
  } catch (e: any) {
    console.error('Reassign failed:', e)
    appStore.showError(e?.message || t('admin.proxies.groupDrawer.reassignFailed'))
    selectEl.value = String(current?.proxy_id ?? '')
  } finally {
    reassigningAccountId.value = null
  }
}

watch(() => props.show, (newVal) => {
  if (newVal) loadData()
})
</script>
