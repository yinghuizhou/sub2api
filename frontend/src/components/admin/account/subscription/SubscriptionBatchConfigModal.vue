<script setup lang="ts">
import { ref, watch } from 'vue'
import SubscriptionBasicForm from './SubscriptionBasicForm.vue'
import { accountsAPI } from '@/api/admin/accounts'
import type { SubscriptionConfig } from '@/types/subscription'

interface Props {
  visible: boolean
  accounts: Array<{ id: number; name: string }>
}

interface Emits {
  (e: 'close'): void
  (e: 'updated'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 选中的账户 ID
const selectedAccountIds = ref<number[]>([])

// 表单数据
const formData = ref<SubscriptionConfig>({
  enabled: true,
  daily_limit_usd: 10,
  subscription_period: 'monthly',
  subscription_start: new Date().toISOString(),
  subscription_end: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString()
})

// 保存状态
const saving = ref(false)

// 全选/取消全选
const selectAll = ref(false)

// 监听全选状态
watch(selectAll, (newVal) => {
  if (newVal) {
    selectedAccountIds.value = props.accounts.map(a => a.id)
  } else {
    selectedAccountIds.value = []
  }
})

// 监听选中账户变化
watch(selectedAccountIds, (newVal) => {
  selectAll.value = newVal.length === props.accounts.length && props.accounts.length > 0
})

// 切换账户选中状态
const toggleAccount = (accountId: number) => {
  const index = selectedAccountIds.value.indexOf(accountId)
  if (index > -1) {
    selectedAccountIds.value.splice(index, 1)
  } else {
    selectedAccountIds.value.push(accountId)
  }
}

// 批量应用配置
const handleApply = async () => {
  if (selectedAccountIds.value.length === 0) {
    alert('请至少选择一个账户')
    return
  }

  saving.value = true
  try {
    await accountsAPI.batchSetSubscriptionConfig({
      account_ids: selectedAccountIds.value,
      config: formData.value
    })
    emit('updated')
    emit('close')
  } catch (error) {
    console.error('Failed to batch set subscription config:', error)
    alert('批量配置失败，请重试')
  } finally {
    saving.value = false
  }
}

// 关闭弹窗
const handleClose = () => {
  emit('close')
}

// 重置表单
watch(() => props.visible, (newVal) => {
  if (newVal) {
    selectedAccountIds.value = []
    selectAll.value = false
  }
})
</script>

<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
    @click.self="handleClose"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-3xl max-h-[90vh] overflow-hidden">
      <!-- 标题栏 -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200">
        <h2 class="text-lg font-semibold text-gray-900">
          批量配置订阅
        </h2>
        <button
          @click="handleClose"
          class="text-gray-400 hover:text-gray-600 transition-colors"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- 内容区 -->
      <div class="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
        <div class="space-y-6">
          <!-- 账户选择 -->
          <div>
            <div class="flex items-center justify-between mb-3">
              <h3 class="text-sm font-medium text-gray-900">
                选择账户（已选 {{ selectedAccountIds.length }} / {{ accounts.length }}）
              </h3>
              <label class="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
                <input
                  v-model="selectAll"
                  type="checkbox"
                  class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                全选
              </label>
            </div>

            <div class="border border-gray-200 rounded-lg max-h-48 overflow-y-auto">
              <label
                v-for="account in accounts"
                :key="account.id"
                class="flex items-center gap-3 px-4 py-2 hover:bg-gray-50 cursor-pointer border-b border-gray-100 last:border-b-0"
              >
                <input
                  type="checkbox"
                  :checked="selectedAccountIds.includes(account.id)"
                  @change="toggleAccount(account.id)"
                  class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <span class="text-sm text-gray-900">{{ account.name }}</span>
              </label>
            </div>
          </div>

          <!-- 订阅配置 -->
          <div>
            <h3 class="text-sm font-medium text-gray-900 mb-3">订阅配置</h3>
            <SubscriptionBasicForm v-model="formData" />
          </div>
        </div>
      </div>

      <!-- 底部按钮 -->
      <div class="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-200 bg-gray-50">
        <button
          @click="handleClose"
          class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
        >
          取消
        </button>
        <button
          @click="handleApply"
          :disabled="saving || selectedAccountIds.length === 0"
          class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ saving ? '应用中...' : `应用到 ${selectedAccountIds.length} 个账户` }}
        </button>
      </div>
    </div>
  </div>
</template>
