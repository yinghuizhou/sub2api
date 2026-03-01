import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SubscriptionProgressBar from '../SubscriptionProgressBar.vue'

describe('SubscriptionProgressBar', () => {
  it('renders usage and limit correctly', () => {
    const wrapper = mount(SubscriptionProgressBar, {
      props: {
        usage: 5.50,
        limit: 10.00,
        percentage: 55
      }
    })

    expect(wrapper.text()).toContain('$5.50')
    expect(wrapper.text()).toContain('$10.00')
    expect(wrapper.text()).toContain('55.0%')
  })

  it('shows green color for normal usage (<80%)', () => {
    const wrapper = mount(SubscriptionProgressBar, {
      props: {
        usage: 7.00,
        limit: 10.00,
        percentage: 70
      }
    })

    const progressBar = wrapper.find('.bg-green-500')
    expect(progressBar.exists()).toBe(true)
  })

  it('shows yellow color for warning usage (80-99%)', () => {
    const wrapper = mount(SubscriptionProgressBar, {
      props: {
        usage: 8.50,
        limit: 10.00,
        percentage: 85
      }
    })

    const progressBar = wrapper.find('.bg-yellow-500')
    expect(progressBar.exists()).toBe(true)
  })

  it('shows red color for exceeded usage (>=100%)', () => {
    const wrapper = mount(SubscriptionProgressBar, {
      props: {
        usage: 12.00,
        limit: 10.00,
        percentage: 120
      }
    })

    const progressBar = wrapper.find('.bg-red-500')
    expect(progressBar.exists()).toBe(true)
  })

  it('limits progress bar width to 100%', () => {
    const wrapper = mount(SubscriptionProgressBar, {
      props: {
        usage: 15.00,
        limit: 10.00,
        percentage: 150
      }
    })

    const progressBar = wrapper.find('.bg-red-500')
    expect(progressBar.attributes('style')).toContain('width: 100%')
  })

  it('formats currency with 2 decimal places', () => {
    const wrapper = mount(SubscriptionProgressBar, {
      props: {
        usage: 5.5,
        limit: 10,
        percentage: 55
      }
    })

    expect(wrapper.text()).toContain('$5.50')
    expect(wrapper.text()).toContain('$10.00')
  })

  it('handles zero usage', () => {
    const wrapper = mount(SubscriptionProgressBar, {
      props: {
        usage: 0,
        limit: 10.00,
        percentage: 0
      }
    })

    expect(wrapper.text()).toContain('$0.00')
    expect(wrapper.text()).toContain('0.0%')
  })
})
