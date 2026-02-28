<script setup lang="ts">
import { ref, watch } from 'vue'
import SubscriptionOverviewCards from './SubscriptionOverviewCards.vue'
import SubscriptionUsageTrendChart from './SubscriptionUsageTrendChart.vue'
import SubscriptionAccountComparison from './SubscriptionAccountComparison.vue'
import SubscriptionAlertSettings from './SubscriptionAlertSettings.vue'
import { accountsAPI } from '@/api/admin/accounts'

interface Props {
  visible: boolean
  accounts: Array<{ id: number; name: string }>
}

interface Emits {
  (e: 'close'): void
  (e: 'export'): void
  (e: 'batchConfig'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 加载状态
const loading = ref(false)

// 统计数据
const stats = ref({
  totalAccounts: 0,
  availableAccounts: 0,
  todayTotalUsage: 0,
  exceededAccounts: 0
})

// 趋势数据
const trendData = ref<Array<{
  accountId: number
  accountName: string
  data: Array<{ date: string; usage: number }>
}>>([])

// 对比数据
const comparisonData = ref<Array<{
  accountId: number
  accountName: string
  usage: number
  limit: number
}>>([])

// 加载数据
const loadData = async () => {
  if (!props.visible || !props.accounts.length) return

  loading.value = true
  try {
    // 获取所有账户的订阅状态
    const statusPromises = props.accounts.map(async (account) => {
      try {
        const config = await accountsAPI.getSubscriptionConfig(account.id)
        const usage = await accountsAPI.getDailyUsage(account.id).catch(() => null)

        return {
          accountId: account.id,
          accountName: account.name,
          config,
          usage
        }
      } catch {
        return null
      }
    })

    const statuses = (await Promise.all(statusPromises)).filter(Boolean)

    // 计算统计数据
    stats.value.totalAccounts = statuses.length
    stats.value.availableAccounts = statuses.filter(s =>
      s!.config?.enabled &&
      (!s!.usage || s!.usage.usage_usd < s!.config.daily_limit_usd)
    ).length
    stats.value.todayTotalUsage = statuses.reduce((sum, s) =>
      sum + (s!.usage?.usage_usd || 0), 0
    )
    stats.value.exceededAccounts = statuses.filter(s =>
      s!.config?.enabled &&
      s!.usage &&
      s!.usage.usage_usd >= s!.config.daily_limit_usd
    ).length

    // 准备趋势数据（近 7 天）
    const trends = []
    for (const status of statuses.slice(0, 5)) { // 只显示前 5 个账户
      if (!status) continue

      const accountTrend = {
        accountId: status.accountId,
        accountName: status.accountName,
        data: [] as Array<{ date: string; usage: number }>
      }

      const today = new Date()
      for (let i = 6; i >= 0; i--) {
        const date = new Date(today)
        date.setDate(date.getDate() - i)
        const dateStr = date.toISOString().split('T')[0]

        try {
          const usage = await accountsAPI.getDailyUsage(status.accountId, dateStr)
          accountTrend.data.push({
            date: dateStr,
            usage: usage.usage_usd
          })
        } catch {
          accountTrend.data.push({
            date: dateStr,
            usage: 0
          })
        }
      }

      trends.push(accountTrend)
    }
    trendData.value = trends

    // 准备对比数据
    comparisonData.value = statuses
      .filter(s => s!.config?.enabled)
      .map(s => ({
        accountId: s!.accountId,
        accountName: s!.accountName,
        usage: s!.usage?.usage_usd || 0,
        limit: s!.config!.daily_limit_usd
      }))
  } catch (error) {
    console.error('Failed to load monitor data:', error)
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

// 关闭弹窗
const handleClose = () => {
  emit('close')
}

// 导出报表
const handleExport = () => {
  emit('export')
}

// 批量配置
const handleBatchConfig = () => {
  emit('batchConfig')
}

// 刷新数据
const handleRefresh = () => {
  loadData()
}
</script>

<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
    @click.self="handleClose"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-6xl max-h-[90vh] overflow-hidden">
      <!-- 标题栏 -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200">
        <h2 class="text-lg font-semibold text-gray-900">
          订阅账户监控仪表板
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

        <div v-else class="space-y-6">
          <!-- 概览卡片 -->
          <SubscriptionOverviewCards
            :total-accounts="stats.totalAccounts"
            :available-accounts="stats.availableAccounts"
            :today-total-usage="stats.todayTotalUsage"
            :exceeded-accounts="stats.exceededAccounts"
          />

          <!-- 用量趋势图 -->
          <div>
            <h3 class="text-sm font-medium text-gray-900 mb-4">用量趋势（近 7 天）</h3>
            <SubscriptionUsageTrendChart :trends="trendData" />
          </div>

          <!-- 账户对比图 -->
          <div>
            <h3 class="text-sm font-medium text-gray-900 mb-4">账户用量对比（今日）</h3>
            <SubscriptionAccountComparison :accounts="comparisonData" />
          </div>

          <!-- 告警设置 -->
          <SubscriptionAlertSettings />
        </div>
      </div>

      <!-- 底部按钮 -->
      <div class="flex items-center justify-between px-6 py-4 border-t border-gray-200 bg-gray-50">
        <div class="flex gap-3">
          <button
            @click="handleExport"
            class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            导出报表
          </button>
          <button
            @click="handleBatchConfig"
            class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            批量配置
          </button>
        </div>
        <div class="flex gap-3">
          <button
            @click="handleRefresh"
            class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            刷新
          </button>
          <button
            @click="handleClose"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 transition-colors"
          >
            关闭
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
