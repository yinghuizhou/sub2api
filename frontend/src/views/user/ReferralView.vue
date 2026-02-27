<template>
  <div class="max-w-2xl mx-auto p-6">
    <h1 class="text-2xl font-bold mb-6">{{ $t('referral.title', '邀请返佣') }}</h1>

    <!-- 邀请码卡片 -->
    <div class="bg-gradient-to-r from-blue-500 to-blue-600 rounded-xl p-6 text-white mb-6">
      <p class="text-sm opacity-80 mb-2">{{ $t('referral.yourCode', '你的邀请码') }}</p>
      <div class="flex items-center gap-3">
        <span class="text-3xl font-mono font-bold tracking-wider">{{ info?.invite_code || '---' }}</span>
        <button
          @click="copyCode"
          class="px-3 py-1 bg-white/20 rounded-lg text-sm hover:bg-white/30 transition-colors"
        >
          {{ copied ? $t('referral.copied', '已复制') : $t('referral.copy', '复制') }}
        </button>
      </div>
      <p class="text-sm opacity-70 mt-3">
        {{ $t('referral.tip', '分享邀请码给好友，好友充值后你可获得 10% 返佣') }}
      </p>
    </div>

    <!-- 统计 -->
    <div class="grid grid-cols-2 gap-4 mb-6">
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 text-center">
        <div class="text-2xl font-bold text-blue-600">{{ info?.total_invitees ?? 0 }}</div>
        <div class="text-sm text-gray-500 mt-1">{{ $t('referral.invitees', '邀请人数') }}</div>
      </div>
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 text-center">
        <div class="text-2xl font-bold text-green-600">${{ (info?.total_commission ?? 0).toFixed(4) }}</div>
        <div class="text-sm text-gray-500 mt-1">{{ $t('referral.totalCommission', '累计返佣') }}</div>
      </div>
    </div>

    <!-- 返佣记录 -->
    <div>
      <h2 class="text-lg font-semibold mb-3">{{ $t('referral.records', '返佣记录') }}</h2>
      <div v-if="commissions.length === 0" class="text-center text-gray-400 py-8">
        {{ $t('referral.noRecords', '暂无返佣记录') }}
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="c in commissions"
          :key="c.id"
          class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg"
        >
          <div>
            <div class="text-sm font-medium">
              {{ $t('referral.orderAmount', '订单金额') }}: ${{ c.order_amount_usd.toFixed(4) }}
            </div>
            <div class="text-xs text-gray-500">{{ new Date(c.created_at).toLocaleDateString() }}</div>
          </div>
          <div class="text-green-600 font-medium">+${{ c.commission_amount.toFixed(4) }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { referralApi, type ReferralInfo, type ReferralCommission } from '@/api/referral'

const info = ref<ReferralInfo | null>(null)
const commissions = ref<ReferralCommission[]>([])
const copied = ref(false)

async function copyCode() {
  if (!info.value?.invite_code) return
  try {
    await navigator.clipboard.writeText(info.value.invite_code)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    // fallback: ignore
  }
}

onMounted(async () => {
  try {
    info.value = await referralApi.getInfo()
  } catch {
    // handle silently
  }
  try {
    const res = await referralApi.listCommissions()
    commissions.value = res.items
  } catch {
    // handle silently
  }
})
</script>
