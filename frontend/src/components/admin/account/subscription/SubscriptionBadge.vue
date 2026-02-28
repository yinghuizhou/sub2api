<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  status: 'normal' | 'warning' | 'exceeded' | 'expired' | 'disabled'
}

const props = defineProps<Props>()

// 状态文本映射
const statusText = computed(() => {
  const textMap = {
    normal: '正常',
    warning: '警告',
    exceeded: '超限',
    expired: '已过期',
    disabled: '未配置'
  }
  return textMap[props.status]
})

// 状态颜色类映射
const statusClass = computed(() => {
  const classMap = {
    normal: 'bg-green-100 text-green-800 border-green-200',
    warning: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    exceeded: 'bg-red-100 text-red-800 border-red-200',
    expired: 'bg-gray-100 text-gray-600 border-gray-200',
    disabled: 'bg-gray-50 text-gray-400 border-gray-100'
  }
  return classMap[props.status]
})
</script>

<template>
  <span
    :class="[
      'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border',
      statusClass
    ]"
  >
    {{ statusText }}
  </span>
</template>
