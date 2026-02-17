/**
 * useForm Composable 单元测试
 * 测试统一表单提交逻辑：加载状态、成功/失败通知、错误处理
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useForm } from '@/composables/useForm'

// Mock app store
const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mockShowSuccess,
    showError: mockShowError
  })
}))

describe('useForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('成功提交应调用 submitFn 并显示成功消息', async () => {
    const submitFn = vi.fn().mockResolvedValue(undefined)
    const formData = { name: 'test', email: 'test@example.com' }

    const { submit, loading } = useForm({
      form: formData,
      submitFn,
      successMsg: '保存成功'
    })

    expect(loading.value).toBe(false)

    await submit()

    expect(submitFn).toHaveBeenCalledWith(formData)
    expect(mockShowSuccess).toHaveBeenCalledWith('保存成功')
    expect(loading.value).toBe(false)
  })

  it('提交失败应显示错误消息并重新抛出', async () => {
    const error = { response: { data: { message: '服务器错误' } }, message: 'Request failed' }
    const submitFn = vi.fn().mockRejectedValue(error)

    const { submit, loading } = useForm({
      form: { name: 'test' },
      submitFn
    })

    await expect(submit()).rejects.toEqual(error)

    expect(mockShowError).toHaveBeenCalledWith('服务器错误')
    expect(loading.value).toBe(false)
  })

  it('自定义 errorMsg 应覆盖 API 错误消息', async () => {
    const submitFn = vi.fn().mockRejectedValue(new Error('API error'))

    const { submit } = useForm({
      form: {},
      submitFn,
      errorMsg: '自定义错误提示'
    })

    await expect(submit()).rejects.toThrow()

    expect(mockShowError).toHaveBeenCalledWith('自定义错误提示')
  })

  it('提交期间 loading 应为 true', async () => {
    let resolveSubmit: () => void
    const submitFn = vi.fn().mockImplementation(() => new Promise<void>((resolve) => {
      resolveSubmit = resolve
    }))

    const { submit, loading } = useForm({
      form: {},
      submitFn
    })

    const promise = submit()
    expect(loading.value).toBe(true)

    resolveSubmit!()
    await promise

    expect(loading.value).toBe(false)
  })

  it('loading 期间重复提交应被忽略', async () => {
    let resolveSubmit: () => void
    const submitFn = vi.fn().mockImplementation(() => new Promise<void>((resolve) => {
      resolveSubmit = resolve
    }))

    const { submit } = useForm({
      form: {},
      submitFn
    })

    const promise1 = submit()
    submit() // 应被忽略
    submit() // 应被忽略

    resolveSubmit!()
    await promise1

    expect(submitFn).toHaveBeenCalledTimes(1)
  })

  it('无 successMsg 时不应调用 showSuccess', async () => {
    const submitFn = vi.fn().mockResolvedValue(undefined)

    const { submit } = useForm({
      form: {},
      submitFn
      // 没有 successMsg
    })

    await submit()

    expect(mockShowSuccess).not.toHaveBeenCalled()
  })

  it('应从 error.response.data.detail 提取错误信息', async () => {
    const error = { response: { data: { detail: '详细错误信息' } }, message: 'generic' }
    const submitFn = vi.fn().mockRejectedValue(error)

    const { submit } = useForm({
      form: {},
      submitFn
    })

    await expect(submit()).rejects.toEqual(error)

    expect(mockShowError).toHaveBeenCalledWith('详细错误信息')
  })

  it('无 API 错误详情时应使用 error.message', async () => {
    const error = { message: 'Network error' }
    const submitFn = vi.fn().mockRejectedValue(error)

    const { submit } = useForm({
      form: {},
      submitFn
    })

    await expect(submit()).rejects.toEqual(error)

    expect(mockShowError).toHaveBeenCalledWith('Network error')
  })
})
