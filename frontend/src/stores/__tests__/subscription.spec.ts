import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSubscriptionStore } from '../subscription'
import type { SubscriptionConfig, DailyUsage } from '@/types/subscription'

// Mock the API
vi.mock('@/api/admin/accounts', () => ({
  accountsAPI: {
    getSubscriptionConfig: vi.fn(),
    getDailyUsage: vi.fn()
  }
}))

import { accountsAPI } from '@/api/admin/accounts'

describe('useSubscriptionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('updateAlertRule', () => {
    it('updates alert rule correctly', () => {
      const store = useSubscriptionStore()

      store.updateAlertRule({
        threshold: 90,
        enabled: false
      })

      expect(store.alertRule.threshold).toBe(90)
      expect(store.alertRule.enabled).toBe(false)
    })
  })

  describe('clearCache', () => {
    it('clears the status cache', async () => {
      const store = useSubscriptionStore()

      // Mock API responses
      const mockConfig: SubscriptionConfig = {
        enabled: true,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2026-01-01T00:00:00Z',
        subscription_end: '2026-02-01T00:00:00Z'
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue({
        account_id: 1,
        date: '2026-01-15',
        usage_usd: 5
      })

      // Load some data
      await store.getStatus(1)

      // Clear cache
      store.clearCache()

      // Verify cache is empty by checking if API is called again
      await store.getStatus(1)
      expect(accountsAPI.getSubscriptionConfig).toHaveBeenCalledTimes(2)
    })
  })

  describe('getStatus', () => {
    it('returns cached status if not expired', async () => {
      const store = useSubscriptionStore()

      const mockConfig: SubscriptionConfig = {
        enabled: true,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2026-01-01T00:00:00Z',
        subscription_end: '2026-02-01T00:00:00Z'
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue({
        account_id: 1,
        date: '2026-01-15',
        usage_usd: 5
      })

      // First call
      await store.getStatus(1)

      // Second call should use cache
      await store.getStatus(1)

      // API should only be called once
      expect(accountsAPI.getSubscriptionConfig).toHaveBeenCalledTimes(1)
    })

    it('calculates status as normal when usage < 80%', async () => {
      const store = useSubscriptionStore()

      const mockConfig: SubscriptionConfig = {
        enabled: true,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2026-01-01T00:00:00Z',
        subscription_end: '2026-12-31T23:59:59Z'
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue({
        account_id: 1,
        date: '2026-01-15',
        usage_usd: 5
      })

      const status = await store.getStatus(1)

      expect(status.status).toBe('normal')
      expect(status.percentage).toBe(50)
    })

    it('calculates status as warning when usage >= 80%', async () => {
      const store = useSubscriptionStore()

      const mockConfig: SubscriptionConfig = {
        enabled: true,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2026-01-01T00:00:00Z',
        subscription_end: '2026-12-31T23:59:59Z'
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue({
        account_id: 1,
        date: '2026-01-15',
        usage_usd: 8.5
      })

      const status = await store.getStatus(1)

      expect(status.status).toBe('warning')
      expect(status.percentage).toBe(85)
    })

    it('calculates status as exceeded when usage >= 100%', async () => {
      const store = useSubscriptionStore()

      const mockConfig: SubscriptionConfig = {
        enabled: true,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2026-01-01T00:00:00Z',
        subscription_end: '2026-12-31T23:59:59Z'
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue({
        account_id: 1,
        date: '2026-01-15',
        usage_usd: 12
      })

      const status = await store.getStatus(1)

      expect(status.status).toBe('exceeded')
      expect(status.percentage).toBe(100) // Capped at 100
    })

    it('returns disabled status when config is not enabled', async () => {
      const store = useSubscriptionStore()

      const mockConfig: SubscriptionConfig = {
        enabled: false,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2026-01-01T00:00:00Z',
        subscription_end: '2026-12-31T23:59:59Z'
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue(null)

      const status = await store.getStatus(1)

      expect(status.status).toBe('disabled')
      expect(status.percentage).toBe(0)
    })

    it('returns expired status when subscription has expired', async () => {
      const store = useSubscriptionStore()

      const mockConfig: SubscriptionConfig = {
        enabled: true,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2025-01-01T00:00:00Z',
        subscription_end: '2025-02-01T00:00:00Z' // Expired
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue({
        account_id: 1,
        date: '2026-01-15',
        usage_usd: 5
      })

      const status = await store.getStatus(1)

      expect(status.status).toBe('expired')
    })
  })

  describe('refreshAll', () => {
    it('refreshes multiple accounts', async () => {
      const store = useSubscriptionStore()

      const mockConfig: SubscriptionConfig = {
        enabled: true,
        daily_limit_usd: 10,
        subscription_period: 'monthly',
        subscription_start: '2026-01-01T00:00:00Z',
        subscription_end: '2026-12-31T23:59:59Z'
      }

      vi.mocked(accountsAPI.getSubscriptionConfig).mockResolvedValue(mockConfig)
      vi.mocked(accountsAPI.getDailyUsage).mockResolvedValue({
        account_id: 1,
        date: '2026-01-15',
        usage_usd: 5
      })

      await store.refreshAll([1, 2, 3])

      expect(accountsAPI.getSubscriptionConfig).toHaveBeenCalledTimes(3)
    })
  })
})
