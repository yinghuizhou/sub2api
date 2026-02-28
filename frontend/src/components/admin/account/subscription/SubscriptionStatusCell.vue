<script setup lang="ts">
import { computed } from 'vue'
import SubscriptionBadge from './SubscriptionBadge.vue'
import SubscriptionProgressBar from './SubscriptionProgressBar.vue'
import type { SubscriptionStatus } from '@/types/subscription'

interface Props {
  status: SubscriptionStatus
}

interface Emits {
  (e: 'click'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 计算用量和限额
const usage = computed(() => props.status.usage?.usage_usd || 0)
const limit = computed(() => props.status.config?.daily_limit_usd || 0)

// 处理点击
const handleClick = () => {
  emit('click')
}
</script>

<template>
  <div
    class="cursor-pointer hover:bg-gray-50 p-2 rounded transition-colors"
    @click="handleClick"
  >
    <!-- 状态徽章 -->
    <div class="mb-2">
      <SubscriptionBadge :status="status.status" />
    </div>

    <!-- 进度条（仅在配置启用时显示） -->
    <div v-if="status.config?.enabled" class="mt-2">
      <SubscriptionProgressBar
        :usage="usage"
        :limit="limit"
        :percentage="status.percentage"
      />
    </div>

    <!-- 未配置提示 -->
    <div v-else class="text-xs text-gray-400 mt-1">
      点击配置订阅限额
    </div>
  </div>
</template>
