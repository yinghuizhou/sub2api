<template>
  <div class="max-w-2xl mx-auto p-6">
    <h1 class="text-2xl font-bold mb-6">{{ $t('recharge.title', '充值余额') }}</h1>

    <!-- 套餐选择 -->
    <div class="grid grid-cols-3 gap-3 mb-6">
      <button
        v-for="pkg in packages"
        :key="pkg.id"
        @click="selectPackage(pkg)"
        :class="[
          'relative p-4 rounded-lg border-2 text-center transition-all',
          selectedPackage?.id === pkg.id
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-gray-200 dark:border-gray-700 hover:border-blue-300',
        ]"
      >
        <div v-if="pkg.label" class="absolute -top-2 left-1/2 -translate-x-1/2">
          <span class="bg-orange-500 text-white text-xs px-2 py-0.5 rounded-full">{{ pkg.label }}</span>
        </div>
        <div class="text-xl font-bold">¥{{ pkg.amount_cny }}</div>
        <div v-if="pkg.bonus_rate > 0" class="text-sm text-green-600 mt-1">
          +{{ (pkg.bonus_rate * 100).toFixed(0) }}% {{ $t('recharge.bonus', '赠送') }}
        </div>
      </button>
    </div>

    <!-- 自定义金额 -->
    <div class="mb-6">
      <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">
        {{ $t('recharge.customAmount', '自定义金额（元）') }}
      </label>
      <input
        v-model.number="customAmount"
        type="number"
        min="1"
        placeholder="输入充值金额"
        class="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:border-gray-600"
        @input="selectedPackage = null"
      />
    </div>

    <!-- 支付方式 -->
    <div class="flex gap-3 mb-6">
      <button
        @click="channel = 'wechat'"
        :class="[
          'flex-1 py-3 rounded-lg border-2 font-medium transition-all',
          channel === 'wechat' ? 'border-green-500 text-green-600' : 'border-gray-200 dark:border-gray-600',
        ]"
      >
        {{ $t('recharge.wechat', '微信支付') }}
      </button>
      <button
        @click="channel = 'alipay'"
        :class="[
          'flex-1 py-3 rounded-lg border-2 font-medium transition-all',
          channel === 'alipay' ? 'border-blue-500 text-blue-600' : 'border-gray-200 dark:border-gray-600',
        ]"
      >
        {{ $t('recharge.alipay', '支付宝') }}
      </button>
    </div>

    <!-- 到账预览 -->
    <div v-if="finalAmount > 0" class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-6">
      <div class="flex justify-between text-sm">
        <span class="text-gray-500">{{ $t('recharge.payAmount', '实付金额') }}</span>
        <span>¥{{ finalAmount.toFixed(2) }}</span>
      </div>
      <div v-if="bonusUSD > 0" class="flex justify-between text-sm mt-1">
        <span class="text-gray-500">{{ $t('recharge.bonusCredit', '赠送额度') }}</span>
        <span class="text-green-600">+${{ bonusUSD.toFixed(4) }}</span>
      </div>
      <div class="flex justify-between font-bold mt-2 pt-2 border-t dark:border-gray-700">
        <span>{{ $t('recharge.totalCredit', '到账余额') }}</span>
        <span class="text-blue-600">${{ totalCreditUSD.toFixed(4) }}</span>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="errorMsg" class="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-lg text-sm">
      {{ errorMsg }}
    </div>

    <button
      @click="handlePay"
      :disabled="finalAmount <= 0 || loading"
      class="w-full py-3 bg-blue-600 text-white rounded-lg font-medium disabled:opacity-50 hover:bg-blue-700 transition-colors"
    >
      {{ loading ? $t('recharge.creating', '创建订单中...') : $t('recharge.payNow', '立即充值') }}
    </button>

    <!-- 二维码弹窗 -->
    <div
      v-if="qrDataUrl"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="closeQR"
    >
      <div class="bg-white dark:bg-gray-900 rounded-xl p-6 text-center w-72">
        <h3 class="font-bold mb-4">
          {{ channel === 'wechat' ? $t('recharge.wechatScan', '微信扫码支付') : $t('recharge.alipayScan', '支付宝扫码支付') }}
        </h3>
        <img :src="qrDataUrl" alt="QR Code" class="w-48 h-48 mx-auto mb-4" />
        <p class="text-sm text-gray-500">{{ $t('recharge.scanTip', '请在 5 分钟内完成支付') }}</p>
        <div v-if="pollStatus === 'paid'" class="mt-3 text-green-600 font-medium">
          {{ $t('recharge.paySuccess', '支付成功！') }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import QRCode from 'qrcode'
import { paymentApi, type RechargePackage } from '@/api/payment'

const packages = ref<RechargePackage[]>([])
const selectedPackage = ref<RechargePackage | null>(null)
const customAmount = ref<number | null>(null)
const channel = ref<'wechat' | 'alipay'>('wechat')
const loading = ref(false)
const qrDataUrl = ref('')
const currentOrderId = ref<number | null>(null)
const pollStatus = ref('')
const errorMsg = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const CNY_TO_USD = 7.2

const finalAmount = computed(() => selectedPackage.value?.amount_cny ?? customAmount.value ?? 0)
const bonusUSD = computed(() => {
  if (!selectedPackage.value) return 0
  return (finalAmount.value / CNY_TO_USD) * selectedPackage.value.bonus_rate + selectedPackage.value.bonus_fixed
})
const totalCreditUSD = computed(() => finalAmount.value / CNY_TO_USD + bonusUSD.value)

function selectPackage(pkg: RechargePackage) {
  selectedPackage.value = pkg
  customAmount.value = null
}

async function handlePay() {
  if (finalAmount.value <= 0) return
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await paymentApi.createOrder({
      amount_cny: finalAmount.value,
      package_id: selectedPackage.value?.id,
      channel: channel.value,
    })
    // Validate payment URL before generating QR code
    if (!res.qr_url) {
      throw new Error('支付二维码生成失败，请稍后重试')
    }
    qrDataUrl.value = await QRCode.toDataURL(res.qr_url, { width: 256, margin: 2 })
    currentOrderId.value = res.order.id
    startPolling()
  } catch (e: any) {
    errorMsg.value = e?.message || '创建订单失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

function startPolling() {
  pollTimer = setInterval(async () => {
    if (!currentOrderId.value) return
    try {
      const order = await paymentApi.getOrder(currentOrderId.value)
      if (order.status === 'paid') {
        pollStatus.value = 'paid'
        stopPolling()
        setTimeout(closeQR, 2000)
      }
    } catch {
      // ignore polling errors
    }
  }, 2000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function closeQR() {
  qrDataUrl.value = ''
  pollStatus.value = ''
  stopPolling()
}

onMounted(async () => {
  try {
    packages.value = await paymentApi.listPackages()
  } catch {
    // handle error silently - packages will be empty
  }
})

onUnmounted(stopPolling)
</script>
