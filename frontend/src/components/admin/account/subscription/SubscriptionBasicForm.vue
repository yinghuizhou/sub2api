<script setup lang="ts">
import { ref, watch } from 'vue'
import type { SubscriptionConfig } from '@/types/subscription'

interface Props {
  modelValue: SubscriptionConfig
}

interface Emits {
  (e: 'update:modelValue', value: SubscriptionConfig): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 本地表单数据
const formData = ref<SubscriptionConfig>({ ...props.modelValue })

// 监听 props 变化
watch(() => props.modelValue, (newVal) => {
  formData.value = { ...newVal }
}, { deep: true })

// 更新表单数据
const updateField = <K extends keyof SubscriptionConfig>(
  field: K,
  value: SubscriptionConfig[K]
) => {
  formData.value[field] = value
  emit('update:modelValue', { ...formData.value })
}

// 订阅周期选项
const periodOptions = [
  { value: 'daily', label: '每日' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' }
]
</script>

<template>
  <div class="space-y-4">
    <!-- 启用开关 -->
    <div class="flex items-center justify-between">
      <label class="text-sm font-medium text-gray-700">
        启用订阅限额
      </label>
      <input
        type="checkbox"
        :checked="formData.enabled"
        @change="updateField('enabled', ($event.target as HTMLInputElement).checked)"
        class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
      />
    </div>

    <!-- 日限额 -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        每日限额（美元）
      </label>
      <input
        type="number"
        :value="formData.daily_limit_usd"
        @input="updateField('daily_limit_usd', parseFloat(($event.target as HTMLInputElement).value) || 0)"
        :disabled="!formData.enabled"
        min="0"
        step="0.01"
        class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
        placeholder="0.00"
      />
    </div>

    <!-- 订阅周期 -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        订阅周期
      </label>
      <select
        :value="formData.subscription_period"
        @change="updateField('subscription_period', ($event.target as HTMLSelectElement).value as 'daily' | 'weekly' | 'monthly')"
        :disabled="!formData.enabled"
        class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
      >
        <option
          v-for="option in periodOptions"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </option>
      </select>
    </div>

    <!-- 开始日期 -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        开始日期
      </label>
      <input
        type="datetime-local"
        :value="formData.subscription_start?.slice(0, 16)"
        @input="updateField('subscription_start', ($event.target as HTMLInputElement).value + ':00Z')"
        :disabled="!formData.enabled"
        class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
      />
    </div>

    <!-- 结束日期 -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        结束日期
      </label>
      <input
        type="datetime-local"
        :value="formData.subscription_end?.slice(0, 16)"
        @input="updateField('subscription_end', ($event.target as HTMLInputElement).value + ':00Z')"
        :disabled="!formData.enabled"
        class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
      />
    </div>
  </div>
</template>
