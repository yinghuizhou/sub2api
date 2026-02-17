/**
 * Auth Store 单元测试
 * 测试认证状态管理、登录/登出、Token 刷新逻辑
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'

// Mock API
const mockLogin = vi.fn()
const mockRegister = vi.fn()
const mockLogout = vi.fn()
const mockGetCurrentUser = vi.fn()
const mockRefreshToken = vi.fn()
const mockIsTotp2FARequired = vi.fn()

vi.mock('@/api', () => ({
  authAPI: {
    login: (...args: unknown[]) => mockLogin(...args),
    register: (...args: unknown[]) => mockRegister(...args),
    logout: (...args: unknown[]) => mockLogout(...args),
    getCurrentUser: (...args: unknown[]) => mockGetCurrentUser(...args),
    refreshToken: (...args: unknown[]) => mockRefreshToken(...args)
  },
  isTotp2FARequired: (...args: unknown[]) => mockIsTotp2FARequired(...args)
}))

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value }),
    removeItem: vi.fn((key: string) => { delete store[key] }),
    clear: vi.fn(() => { store = {} }),
    get length() { return Object.keys(store).length },
    key: vi.fn((index: number) => Object.keys(store)[index] ?? null),
    _store: store,
    _reset: () => { store = {} }
  }
})()

Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, writable: true })

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorageMock._reset()
    vi.useFakeTimers()
    vi.clearAllMocks()
    mockIsTotp2FARequired.mockReturnValue(false)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('初始状态', () => {
    it('应该默认未认证', () => {
      const store = useAuthStore()
      expect(store.isAuthenticated).toBe(false)
      expect(store.isAdmin).toBe(false)
      expect(store.user).toBeNull()
      expect(store.token).toBeNull()
    })
  })

  describe('登录', () => {
    it('成功登录应设置认证状态', async () => {
      const mockResponse = {
        access_token: 'test-token-123',
        refresh_token: 'refresh-token-456',
        expires_in: 3600,
        user: { id: 1, email: 'test@example.com', role: 'user' }
      }
      mockLogin.mockResolvedValue(mockResponse)

      const store = useAuthStore()
      await store.login({ email: 'test@example.com', password: 'password123' })

      expect(store.isAuthenticated).toBe(true)
      expect(store.token).toBe('test-token-123')
      expect(store.user?.email).toBe('test@example.com')
      expect(localStorage.getItem('auth_token')).toBe('test-token-123')
      expect(localStorage.getItem('refresh_token')).toBe('refresh-token-456')
    })

    it('登录失败应清除状态', async () => {
      mockLogin.mockRejectedValue(new Error('Invalid credentials'))

      const store = useAuthStore()
      await expect(store.login({ email: 'bad@example.com', password: 'wrong' })).rejects.toThrow()

      expect(store.isAuthenticated).toBe(false)
      expect(store.token).toBeNull()
    })

    it('需要 2FA 时不应设置认证状态', async () => {
      const mockResponse = { temp_token: 'temp-123', requires_2fa: true }
      mockLogin.mockResolvedValue(mockResponse)
      mockIsTotp2FARequired.mockReturnValue(true)

      const store = useAuthStore()
      const result = await store.login({ email: 'test@example.com', password: 'password123' })

      expect(store.isAuthenticated).toBe(false)
      expect(result).toEqual(mockResponse)
    })

    it('admin 角色应被正确识别', async () => {
      const mockResponse = {
        access_token: 'admin-token',
        refresh_token: 'refresh-token',
        expires_in: 3600,
        user: { id: 1, email: 'admin@example.com', role: 'admin' }
      }
      mockLogin.mockResolvedValue(mockResponse)

      const store = useAuthStore()
      await store.login({ email: 'admin@example.com', password: 'admin123' })

      expect(store.isAdmin).toBe(true)
    })
  })

  describe('注册', () => {
    it('成功注册应设置认证状态', async () => {
      const mockResponse = {
        access_token: 'new-token',
        refresh_token: 'new-refresh',
        expires_in: 3600,
        user: { id: 2, email: 'new@example.com', role: 'user' }
      }
      mockRegister.mockResolvedValue(mockResponse)

      const store = useAuthStore()
      const user = await store.register({ email: 'new@example.com', password: 'password123' })

      expect(store.isAuthenticated).toBe(true)
      expect(user.email).toBe('new@example.com')
    })
  })

  describe('登出', () => {
    it('登出应清除所有认证状态', async () => {
      // 先登录
      const mockResponse = {
        access_token: 'test-token',
        refresh_token: 'refresh-token',
        expires_in: 3600,
        user: { id: 1, email: 'test@example.com', role: 'user' }
      }
      mockLogin.mockResolvedValue(mockResponse)
      mockLogout.mockResolvedValue(undefined)

      const store = useAuthStore()
      await store.login({ email: 'test@example.com', password: 'password123' })
      expect(store.isAuthenticated).toBe(true)

      await store.logout()

      expect(store.isAuthenticated).toBe(false)
      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(localStorage.getItem('auth_token')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
      expect(localStorage.getItem('auth_user')).toBeNull()
    })
  })

  describe('checkAuth - 从 localStorage 恢复', () => {
    it('有有效 token 和 user 应恢复认证状态', () => {
      localStorage.setItem('auth_token', 'saved-token')
      localStorage.setItem('auth_user', JSON.stringify({ id: 1, email: 'user@example.com', role: 'user' }))
      mockGetCurrentUser.mockResolvedValue({ data: { id: 1, email: 'user@example.com', role: 'user' } })

      const store = useAuthStore()
      store.checkAuth()

      expect(store.isAuthenticated).toBe(true)
      expect(store.token).toBe('saved-token')
    })

    it('没有保存的 token 应保持未认证', () => {
      const store = useAuthStore()
      store.checkAuth()

      expect(store.isAuthenticated).toBe(false)
    })

    it('损坏的 user JSON 应清除状态', () => {
      localStorage.setItem('auth_token', 'saved-token')
      localStorage.setItem('auth_user', 'invalid-json')

      const store = useAuthStore()
      store.checkAuth()

      expect(store.isAuthenticated).toBe(false)
      expect(localStorage.getItem('auth_token')).toBeNull()
    })
  })

  describe('refreshUser', () => {
    it('未认证时调用 refreshUser 应抛出错误', async () => {
      const store = useAuthStore()
      await expect(store.refreshUser()).rejects.toThrow('Not authenticated')
    })

    it('401 响应应清除认证状态', async () => {
      localStorage.setItem('auth_token', 'expired-token')
      localStorage.setItem('auth_user', JSON.stringify({ id: 1, email: 'user@example.com', role: 'user' }))
      mockGetCurrentUser.mockRejectedValueOnce({ status: 401 })

      const store = useAuthStore()
      store.checkAuth()

      // The initial refreshUser call from checkAuth should clear auth on 401
      // Wait for the async refreshUser triggered by checkAuth
      await vi.advanceTimersByTimeAsync(100)

      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('run mode', () => {
    it('simple mode 应被正确设置', async () => {
      const mockResponse = {
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 3600,
        user: { id: 1, email: 'test@example.com', role: 'user', run_mode: 'simple' as const }
      }
      mockLogin.mockResolvedValue(mockResponse)

      const store = useAuthStore()
      await store.login({ email: 'test@example.com', password: 'password123' })

      expect(store.isSimpleMode).toBe(true)
    })
  })
})
