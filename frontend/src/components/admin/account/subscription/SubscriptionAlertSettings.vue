<script setup lang="ts">
import { ref, watch } from 'vue'
import { useSubscriptionStore } from '@/stores/subscription'
import type { AlertRule } from '@/types/subscription'

const subscriptionStore = useSubscriptionStore()

// 本地状态
const localRule = ref<AlertRule>({ ...subscriptionStore.alertRule })

// 监听 store 变化
watch(() => subscriptionStore.alertRule, (newVal) => {
  localRule.value = { ...newVal }
}, { deep: true })

// 更新阈值
const updateThreshold = (value: number) => {
  localRule.value.threshold = value
  subscriptionStore.updateAlertRule(localRule.value)
}

// 切换启用状态
const toggleEnabled = () => {
  localRule.value.enabled = !localRule.value.enabled
  subscriptionStore.updateAlertRule(localRule.value)
}
</script>

<template>
  <div class="bg-white rounded-lg border border-gray-200 p-4">
    <h3 class="text-sm font-medium text-gray-900 mb-4">告警设置</h3>

    <div class="space-y-4">
      <!-- 启用开关 -->
      <div class="flex items-center justify-between">
        <label class="text-sm text-gray-700">启用告警</label>
        <input
          type="checkbox"
          :checked="localRule.enabled"
          @change="toggleEnabled"
          class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
        />
      </div>

      <!-- 阈值滑块 -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <label class="text-sm text-gray-700">告警阈值</label>
          <span class="text-sm font-medium text-gray-900">{{ localRule.threshold }}%</span>
        </div>
        <input
          type="range"
          :value="localRule.threshold"
          @input="updateThreshold(parseInt(($event.target as HTMLInputElement).value))"
          :disabled="!localRule.enabled"
          min="0"
          max="100"
          step="5"
          class="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          :class="{
            'accent-blue-600': localRule.enabled
          }"
        />
        <div class="flex justify-between text-xs text-gray-500 mt-1">
          <span>0%</span>
          <span>50%</span>
          <span>100%</span>
        </div>
      </div>

      <!-- 说明文本 -->
      <p class="text-xs text-gray-500">
        当账户用量达到限额的 {{ localRule.threshold }}% 时触发告警
      </p>
    </div>
  </div>
</template>
