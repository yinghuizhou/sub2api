<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  usage: number      // 当前用量（美元）
  limit: number      // 限额（美元）
  percentage: number // 百分比 (0-100)
}

const props = defineProps<Props>()

// 进度条颜色类
const progressColorClass = computed(() => {
  if (props.percentage >= 100) {
    return 'bg-red-500'
  } else if (props.percentage >= 80) {
    return 'bg-yellow-500'
  } else {
    return 'bg-green-500'
  }
})

// 格式化金额
const formatCurrency = (value: number): string => {
  return `$${value.toFixed(2)}`
}
</script>

<template>
  <div class="w-full">
    <!-- 进度条 -->
    <div class="relative w-full h-2 bg-gray-200 rounded-full overflow-hidden">
      <div
        :class="['h-full transition-all duration-300', progressColorClass]"
        :style="{ width: `${Math.min(percentage, 100)}%` }"
      />
    </div>

    <!-- 用量信息 -->
    <div class="mt-1 flex items-center justify-between text-xs text-gray-600">
      <span>{{ formatCurrency(usage) }} / {{ formatCurrency(limit) }}</span>
      <span class="font-medium">{{ percentage.toFixed(1) }}%</span>
    </div>
  </div>
</template>
