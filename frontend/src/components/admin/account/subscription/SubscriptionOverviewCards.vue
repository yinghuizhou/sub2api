<script setup lang="ts">
interface Props {
  totalAccounts: number
  availableAccounts: number
  todayTotalUsage: number
  exceededAccounts: number
}

defineProps<Props>()

// 格式化金额
const formatCurrency = (value: number): string => {
  return `$${value.toFixed(2)}`
}
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- 总账户数 -->
    <div class="bg-white rounded-lg border border-gray-200 p-4">
      <div class="text-sm text-gray-600 mb-1">总账户数</div>
      <div class="text-2xl font-bold text-gray-900">{{ totalAccounts }}</div>
    </div>

    <!-- 可用账户 -->
    <div class="bg-white rounded-lg border border-gray-200 p-4">
      <div class="text-sm text-gray-600 mb-1">可用账户</div>
      <div class="text-2xl font-bold text-green-600">{{ availableAccounts }}</div>
    </div>

    <!-- 今日总用量 -->
    <div class="bg-white rounded-lg border border-gray-200 p-4">
      <div class="text-sm text-gray-600 mb-1">今日总用量</div>
      <div class="text-2xl font-bold text-blue-600">{{ formatCurrency(todayTotalUsage) }}</div>
    </div>

    <!-- 超限账户 -->
    <div class="bg-white rounded-lg border border-gray-200 p-4">
      <div class="text-sm text-gray-600 mb-1">超限账户</div>
      <div class="text-2xl font-bold text-red-600">{{ exceededAccounts }}</div>
    </div>
  </div>
</template>
