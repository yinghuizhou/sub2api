import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SubscriptionBadge from '../SubscriptionBadge.vue'

describe('SubscriptionBadge', () => {
  it('renders normal status correctly', () => {
    const wrapper = mount(SubscriptionBadge, {
      props: {
        status: 'normal'
      }
    })

    expect(wrapper.text()).toBe('正常')
    expect(wrapper.classes()).toContain('bg-green-100')
    expect(wrapper.classes()).toContain('text-green-800')
  })

  it('renders warning status correctly', () => {
    const wrapper = mount(SubscriptionBadge, {
      props: {
        status: 'warning'
      }
    })

    expect(wrapper.text()).toBe('警告')
    expect(wrapper.classes()).toContain('bg-yellow-100')
    expect(wrapper.classes()).toContain('text-yellow-800')
  })

  it('renders exceeded status correctly', () => {
    const wrapper = mount(SubscriptionBadge, {
      props: {
        status: 'exceeded'
      }
    })

    expect(wrapper.text()).toBe('超限')
    expect(wrapper.classes()).toContain('bg-red-100')
    expect(wrapper.classes()).toContain('text-red-800')
  })

  it('renders expired status correctly', () => {
    const wrapper = mount(SubscriptionBadge, {
      props: {
        status: 'expired'
      }
    })

    expect(wrapper.text()).toBe('已过期')
    expect(wrapper.classes()).toContain('bg-gray-100')
    expect(wrapper.classes()).toContain('text-gray-600')
  })

  it('renders disabled status correctly', () => {
    const wrapper = mount(SubscriptionBadge, {
      props: {
        status: 'disabled'
      }
    })

    expect(wrapper.text()).toBe('未配置')
    expect(wrapper.classes()).toContain('bg-gray-50')
    expect(wrapper.classes()).toContain('text-gray-400')
  })
})
