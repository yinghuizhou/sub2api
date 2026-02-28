<script setup lang="ts">
import { ref, watch } from 'vue'
import SubscriptionBasicForm from './SubscriptionBasicForm.vue'
import SubscriptionUsageChart from './SubscriptionUsageChart.vue'
import SubscriptionQuickActions from './SubscriptionQuickActions.vue'
import { accountsAPI } from '@/api/admin/accounts'
import type { SubscriptionConfig, UsageTrend } from '@/types/subscription'

interface Props {
  visible: boolean
  accountId: number
  accountName: string
}

interface Emits {
  (e: 'close'): void
  (e: 'updated'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 表单数据
const formData = ref<SubscriptionConfig>({
  enabled: false,
  daily_limit_usd: 0,
  subscription_period: 'monthly',
  subscription_start: new Date().toISOString(),
  subscription_end: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString()
})

// 用量趋势数据
const usageTrends = ref<UsageTrend[]>([])

// 加载状态
const loading = ref(false)
const saving = ref(false)

// 加载数据
const loadData = async () => {
  if (!props.visible || !props.accountId) return

  loading.value = true
  try {
    // 加载订阅配置
    const config = await accountsAPI.getSubscriptionConfig(props.accountId)
    if (config) {
      formData.value = { ...config }
    }

    // 加载近 7 天用量数据
    const today = new Date()
    const trends: UsageTrend[] = []
    for (let i = 6; i >= 0; i--) {
      const date = new Date(today)
      date.setDate(date.getDate() - i)
      const dateStr = date.toISOString().split('T')[0]

      try {
        const usage = await accountsAPI.getDailyUsage(props.accountId, dateStr)
        trends.push({
          date: dateStr,
          usage_usd: usage.usage_usd,
          limit_usd: formData.value.daily_limit_usd
        })
      } catch {
        trends.push({
          date: dateStr,
          usage_usd: 0,
          limit_usd: formData.value.daily_limit_usd
        })
      }
    }
    usageTrends.value = trends
  } catch (error) {
    console.error('Failed to load subscription data:', error)
  } finally {
    loading.value = false
  }
}

// 监听 visible 变化
watch(() => props.visible, (newVal) => {
  if (newVal) {
    loadData()
  }
})

// 保存配置
const handleSave = async () => {
  saving.value = true
  try {
    await accountsAPI.setSubscriptionConfig(props.accountId, formData.value)
    emit('updated')
    emit('close')
  } catch (error) {
    console.error('Failed to save subscription config:', error)
    alert('保存失败，请重试')
  } finally {
    saving.value = false
  }
}

// 快速操作：续费
const handleRenew = async (months: number) => {
  const endDate = new Date(formData.value.subscription_end)
  endDate.setMonth(endDate.getMonth() + months)
  formData.value.subscription_end = endDate.toISOString()
}

// 快速操作：调整限额
const handleAdjustLimit = async (amount: number) => {
  formData.value.daily_limit_usd = Math.max(0, formData.value.daily_limit_usd + amount)
}

// 快速操作：重置用量
const handleResetUsage = async () => {
  try {
    const today = new Date().toISOString().split('T')[0]
    await accountsAPI.resetDailyUsage(props.accountId, today)
    await loadData()
    alert('重置成功')
  } catch (error) {
    console.error('Failed to reset usage:', error)
    alert('重置失败，请重试')
  }
}

// 关闭弹窗
const handleClose = () => {
  emit('close')
}
</script>

<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
    @click.self="handleClose"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] overflow-hidden">
      <!-- 标题栏 -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200">
        <h2 class="text-lg font-semibold text-gray-900">
          订阅配置 - {{ accountName }}
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
        <div v-if="loading" class="text-center py-8 text-gray-500">
          加载中...
        </div>

        <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <!-- 左侧：基础配置 -->
          <div>
            <h3 class="text-sm font-medium text-gray-900 mb-4">基础配置</h3>
            <SubscriptionBasicForm v-model="formData" />
          </div>

          <!-- 右侧：用量趋势 -->
          <div>
            <h3 class="text-sm font-medium text-gray-900 mb-4">用量趋势（近 7 天）</h3>
            <SubscriptionUsageChart :data="usageTrends" />
          </div>

          <!-- 底部：快速操作 -->
          <div class="lg:col-span-2">
            <h3 class="text-sm font-medium text-gray-900 mb-4">快速操作</h3>
            <SubscriptionQuickActions
              @renew="handleRenew"
              @adjust-limit="handleAdjustLimit"
              @reset-usage="handleResetUsage"
            />
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
          @click="handleSave"
          :disabled="saving"
          class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>
