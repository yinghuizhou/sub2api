/**
 * App Store 单元测试
 * 测试全局 UI 状态管理：侧边栏、加载状态、Toast 通知
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAppStore } from '@/stores/app'

// Mock API
vi.mock('@/api/admin/system', () => ({
  checkUpdates: vi.fn()
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn()
}))

describe('App Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('侧边栏状态', () => {
    it('toggleSidebar 应切换折叠状态', () => {
      const store = useAppStore()
      expect(store.sidebarCollapsed).toBe(false)

      store.toggleSidebar()
      expect(store.sidebarCollapsed).toBe(true)

      store.toggleSidebar()
      expect(store.sidebarCollapsed).toBe(false)
    })

    it('setSidebarCollapsed 应直接设置状态', () => {
      const store = useAppStore()
      store.setSidebarCollapsed(true)
      expect(store.sidebarCollapsed).toBe(true)
    })

    it('toggleMobileSidebar 应切换移动端状态', () => {
      const store = useAppStore()
      expect(store.mobileOpen).toBe(false)

      store.toggleMobileSidebar()
      expect(store.mobileOpen).toBe(true)
    })
  })

  describe('加载状态', () => {
    it('setLoading 应管理嵌套加载计数', () => {
      const store = useAppStore()
      expect(store.loading).toBe(false)

      store.setLoading(true)
      expect(store.loading).toBe(true)

      store.setLoading(true) // 嵌套
      expect(store.loading).toBe(true)

      store.setLoading(false) // 减少计数
      expect(store.loading).toBe(true) // 仍在加载

      store.setLoading(false) // 计数归零
      expect(store.loading).toBe(false)
    })

    it('setLoading(false) 不应使计数变为负数', () => {
      const store = useAppStore()
      store.setLoading(false)
      store.setLoading(false)
      expect(store.loading).toBe(false)
    })

    it('withLoading 应管理加载状态并返回结果', async () => {
      const store = useAppStore()
      const result = await store.withLoading(async () => {
        expect(store.loading).toBe(true)
        return 'done'
      })

      expect(result).toBe('done')
      expect(store.loading).toBe(false)
    })

    it('withLoading 操作失败时应重置加载状态', async () => {
      const store = useAppStore()
      await expect(
        store.withLoading(async () => {
          throw new Error('fail')
        })
      ).rejects.toThrow('fail')

      expect(store.loading).toBe(false)
    })

    it('withLoadingAndError 应捕获错误并显示 toast', async () => {
      const store = useAppStore()
      const result = await store.withLoadingAndError(async () => {
        throw new Error('something went wrong')
      })

      expect(result).toBeNull()
      expect(store.loading).toBe(false)
      expect(store.toasts.length).toBe(1)
      expect(store.toasts[0].type).toBe('error')
    })
  })

  describe('Toast 通知', () => {
    it('showSuccess 应创建 success toast', () => {
      const store = useAppStore()
      const id = store.showSuccess('操作成功')

      expect(store.toasts.length).toBe(1)
      expect(store.toasts[0].type).toBe('success')
      expect(store.toasts[0].message).toBe('操作成功')
      expect(store.toasts[0].id).toBe(id)
      expect(store.hasActiveToasts).toBe(true)
    })

    it('showError 应创建 error toast', () => {
      const store = useAppStore()
      store.showError('出错了')

      expect(store.toasts[0].type).toBe('error')
      expect(store.toasts[0].message).toBe('出错了')
    })

    it('showInfo 和 showWarning 应创建对应类型', () => {
      const store = useAppStore()
      store.showInfo('提示信息')
      store.showWarning('警告信息')

      expect(store.toasts[0].type).toBe('info')
      expect(store.toasts[1].type).toBe('warning')
    })

    it('hideToast 应移除指定 toast', () => {
      const store = useAppStore()
      const id = store.showSuccess('test')
      expect(store.toasts.length).toBe(1)

      store.hideToast(id)
      expect(store.toasts.length).toBe(0)
      expect(store.hasActiveToasts).toBe(false)
    })

    it('hideToast 对不存在的 ID 应无操作', () => {
      const store = useAppStore()
      store.showSuccess('test')
      store.hideToast('nonexistent')
      expect(store.toasts.length).toBe(1)
    })

    it('clearAllToasts 应清除所有 toast', () => {
      const store = useAppStore()
      store.showSuccess('1')
      store.showError('2')
      store.showInfo('3')

      store.clearAllToasts()
      expect(store.toasts.length).toBe(0)
    })

    it('toast 应按指定时间自动消失', () => {
      const store = useAppStore()
      store.showSuccess('auto dismiss', 1000)

      expect(store.toasts.length).toBe(1)

      vi.advanceTimersByTime(1000)

      expect(store.toasts.length).toBe(0)
    })

    it('多个 toast 应有唯一 ID', () => {
      const store = useAppStore()
      const id1 = store.showSuccess('first')
      const id2 = store.showSuccess('second')

      expect(id1).not.toBe(id2)
    })
  })

  describe('reset', () => {
    it('应重置所有状态到初始值', () => {
      const store = useAppStore()
      store.setSidebarCollapsed(true)
      store.setLoading(true)
      store.showSuccess('test')

      store.reset()

      expect(store.sidebarCollapsed).toBe(false)
      expect(store.loading).toBe(false)
      expect(store.toasts.length).toBe(0)
    })
  })
})
