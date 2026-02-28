import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { SubscriptionStatus, AlertRule, SubscriptionConfig, DailyUsage } from '@/types/subscription'
import { accountsAPI } from '@/api/admin/accounts'

// 缓存项接口
interface CacheItem {
  data: SubscriptionStatus
  timestamp: number
  lastAccess: number  // 最后访问时间
}

export const useSubscriptionStore = defineStore('subscription', () => {
  // 状态缓存 (accountId -> CacheItem)
  const statusCache = ref<Map<number, CacheItem>>(new Map())

  // 缓存 TTL (5 分钟)
  const CACHE_TTL = 5 * 60 * 1000

  // 最大缓存数量 (LRU)
  const MAX_CACHE_SIZE = 100

  // 告警规则
  const alertRule = ref<AlertRule>({
    threshold: 80,
    enabled: true
  })

  /**
   * 检查缓存是否过期
   */
  function isCacheExpired(timestamp: number): boolean {
    return Date.now() - timestamp > CACHE_TTL
  }

  /**
   * 获取账户订阅状态（带缓存）
   */
  async function getStatus(accountId: number): Promise<SubscriptionStatus> {
    // 检查缓存
    const cached = statusCache.value.get(accountId)
    if (cached && !isCacheExpired(cached.timestamp)) {
      // 更新最后访问时间
      cached.lastAccess = Date.now()
      return cached.data
    }

    // 尝试刷新，失败时返回过期缓存
    try {
      return await refreshStatus(accountId)
    } catch (error) {
      // 如果有过期缓存，返回过期缓存
      if (cached) {
        console.warn(`Using stale cache for account ${accountId} due to API error`)
        return cached.data
      }
      throw error
    }
  }

  /**
   * 刷新单个账户状态
   */
  async function refreshStatus(accountId: number): Promise<SubscriptionStatus> {
    try {
      // 并行获取配置和用量
      const [config, usage] = await Promise.all([
        accountsAPI.getSubscriptionConfig(accountId),
        accountsAPI.getDailyUsage(accountId).catch(() => null)
      ])

      // 计算状态
      const status = calculateStatus(config, usage)

      // LRU: 如果缓存已满，删除最久未访问的项
      if (statusCache.value.size >= MAX_CACHE_SIZE && !statusCache.value.has(accountId)) {
        let oldestKey: number | undefined
        let oldestAccess = Infinity

        for (const [key, item] of statusCache.value.entries()) {
          if (item.lastAccess < oldestAccess) {
            oldestAccess = item.lastAccess
            oldestKey = key
          }
        }

        if (oldestKey !== undefined) {
          statusCache.value.delete(oldestKey)
        }
      }

      // 更新缓存
      const now = Date.now()
      statusCache.value.set(accountId, {
        data: status,
        timestamp: now,
        lastAccess: now  // 初始化最后访问时间
      })

      return status
    } catch (error) {
      console.error(`Failed to refresh subscription status for account ${accountId}:`, error)
      throw error
    }
  }

  /**
   * 批量刷新账户状态
   */
  async function refreshAll(accountIds: number[]): Promise<void> {
    await Promise.all(
      accountIds.map(id => refreshStatus(id).catch(err => {
        console.error(`Failed to refresh account ${id}:`, err)
      }))
    )
  }

  /**
   * 清除缓存
   */
  function clearCache(): void {
    statusCache.value.clear()
  }

  /**
   * 更新告警规则
   */
  function updateAlertRule(rule: AlertRule): void {
    alertRule.value = rule
  }

  /**
   * 计算订阅状态
   */
  function calculateStatus(
    config: SubscriptionConfig | null,
    usage: DailyUsage | null
  ): SubscriptionStatus {
    // 未配置
    if (!config || !config.enabled) {
      return {
        config,
        usage,
        status: 'disabled',
        percentage: 0
      }
    }

    // 检查是否过期
    const now = new Date()
    const endDate = new Date(config.subscription_end)
    if (now > endDate) {
      return {
        config,
        usage,
        status: 'expired',
        percentage: 0
      }
    }

    // 计算用量百分比
    const usageUsd = usage?.usage_usd || 0
    const limitUsd = config.daily_limit_usd
    const percentage = limitUsd > 0 ? (usageUsd / limitUsd) * 100 : 0

    // 判断状态
    let status: 'normal' | 'warning' | 'exceeded' = 'normal'
    if (percentage >= 100) {
      status = 'exceeded'
    } else if (percentage >= alertRule.value.threshold) {
      status = 'warning'
    }

    return {
      config,
      usage,
      status,
      percentage: Math.min(percentage, 100)
    }
  }

  return {
    alertRule,
    getStatus,
    refreshStatus,
    refreshAll,
    clearCache,
    updateAlertRule
  }
})
