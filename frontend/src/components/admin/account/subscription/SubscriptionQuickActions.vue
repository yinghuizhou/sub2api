<script setup lang="ts">
import { ref } from 'vue'

interface Emits {
  (e: 'renew', months: number): void
  (e: 'adjustLimit', amount: number): void
  (e: 'resetUsage'): void
}

const emit = defineEmits<Emits>()

// 自定义续费月数
const customRenewMonths = ref<number>(1)
// 自定义限额调整
const customLimitAdjust = ref<number>(0)
// 显示自定义输入
const showCustomRenew = ref(false)
const showCustomLimit = ref(false)

// 续费操作
const handleRenew = (months: number) => {
  emit('renew', months)
  showCustomRenew.value = false
}

// 调整限额
const handleAdjustLimit = (amount: number) => {
  emit('adjustLimit', amount)
  showCustomLimit.value = false
}

// 重置用量
const handleResetUsage = () => {
  if (confirm('确定要重置今日用量吗？此操作不可撤销。')) {
    emit('resetUsage')
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- 续费操作 -->
    <div>
      <h4 class="text-sm font-medium text-gray-700 mb-2">续费</h4>
      <div class="flex flex-wrap gap-2">
        <button
          @click="handleRenew(1)"
          class="px-3 py-1.5 text-sm bg-blue-50 text-blue-700 hover:bg-blue-100 rounded border border-blue-200 transition-colors"
        >
          +1 月
        </button>
        <button
          @click="handleRenew(3)"
          class="px-3 py-1.5 text-sm bg-blue-50 text-blue-700 hover:bg-blue-100 rounded border border-blue-200 transition-colors"
        >
          +3 月
        </button>
        <button
          @click="showCustomRenew = !showCustomRenew"
          class="px-3 py-1.5 text-sm bg-gray-50 text-gray-700 hover:bg-gray-100 rounded border border-gray-200 transition-colors"
        >
          自定义
        </button>
      </div>

      <!-- 自定义续费输入 -->
      <div v-if="showCustomRenew" class="mt-2 flex gap-2">
        <input
          v-model.number="customRenewMonths"
          type="number"
          min="1"
          class="flex-1 px-3 py-1.5 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          placeholder="月数"
        />
        <button
          @click="handleRenew(customRenewMonths)"
          class="px-3 py-1.5 text-sm bg-blue-600 text-white hover:bg-blue-700 rounded transition-colors"
        >
          确定
        </button>
      </div>
    </div>

    <!-- 调整限额 -->
    <div>
      <h4 class="text-sm font-medium text-gray-700 mb-2">调整限额</h4>
      <div class="flex flex-wrap gap-2">
        <button
          @click="handleAdjustLimit(5)"
          class="px-3 py-1.5 text-sm bg-green-50 text-green-700 hover:bg-green-100 rounded border border-green-200 transition-colors"
        >
          +$5
        </button>
        <button
          @click="handleAdjustLimit(-5)"
          class="px-3 py-1.5 text-sm bg-orange-50 text-orange-700 hover:bg-orange-100 rounded border border-orange-200 transition-colors"
        >
          -$5
        </button>
        <button
          @click="showCustomLimit = !showCustomLimit"
          class="px-3 py-1.5 text-sm bg-gray-50 text-gray-700 hover:bg-gray-100 rounded border border-gray-200 transition-colors"
        >
          自定义
        </button>
      </div>

      <!-- 自定义限额输入 -->
      <div v-if="showCustomLimit" class="mt-2 flex gap-2">
        <input
          v-model.number="customLimitAdjust"
          type="number"
          step="0.01"
          class="flex-1 px-3 py-1.5 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          placeholder="金额（可为负数）"
        />
        <button
          @click="handleAdjustLimit(customLimitAdjust)"
          class="px-3 py-1.5 text-sm bg-blue-600 text-white hover:bg-blue-700 rounded transition-colors"
        >
          确定
        </button>
      </div>
    </div>

    <!-- 重置用量 -->
    <div>
      <h4 class="text-sm font-medium text-gray-700 mb-2">危险操作</h4>
      <button
        @click="handleResetUsage"
        class="px-3 py-1.5 text-sm bg-red-50 text-red-700 hover:bg-red-100 rounded border border-red-200 transition-colors"
      >
        重置今日用量
      </button>
    </div>
  </div>
</template>
