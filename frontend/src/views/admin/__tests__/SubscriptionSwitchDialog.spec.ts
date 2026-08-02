import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Group, UserSubscription } from '@/types'
import SubscriptionSwitchDialog from '../components/SubscriptionSwitchDialog.vue'

const { switchSubscription, showError, showSuccess } = vi.hoisted(() => ({
  switchSubscription: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    subscriptions: { switchSubscription }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const group = (overrides: Partial<Group>): Group =>
  ({
    id: 1,
    name: 'Group',
    platform: 'openai',
    status: 'active',
    subscription_type: 'subscription',
    rate_multiplier: 1,
    description: '',
    ...overrides
  }) as Group

const currentGroup = group({ id: 10, name: 'OpenAI Pro' })
const targetGroup = group({
  id: 20,
  name: 'OpenAI Lite',
  daily_limit_usd: 50,
  weekly_limit_usd: 400,
  monthly_limit_usd: 700
})
const subscription = {
  id: 100,
  user_id: 15,
  group_id: currentGroup.id,
  group: currentGroup,
  status: 'active',
  starts_at: '2026-07-10T08:00:00Z',
  expires_at: '2026-08-09T08:00:00Z',
  daily_usage_usd: 75,
  weekly_usage_usd: 350,
  monthly_usage_usd: 725,
  monthly_bonus_usd: 20,
  effective_monthly_limit_usd: 0,
  renewal_eligible: false,
  renewal_price: null,
  daily_window_start: '2026-07-10T08:00:00Z',
  weekly_window_start: '2026-07-10T08:00:00Z',
  monthly_window_start: '2026-07-10T08:00:00Z',
  created_at: '2026-07-10T08:00:00Z',
  updated_at: '2026-07-10T08:00:00Z'
} as UserSubscription

const mountDialog = () =>
  mount(SubscriptionSwitchDialog, {
    props: {
      show: true,
      subscription,
      groups: [
        currentGroup,
        targetGroup,
        group({ id: 30, name: 'Anthropic Lite', platform: 'anthropic' }),
        group({ id: 40, name: 'Disabled', status: 'inactive' }),
        group({ id: 50, name: 'Standard', subscription_type: 'standard' })
      ]
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>'
        },
        Icon: true
      }
    }
  })

describe('SubscriptionSwitchDialog', () => {
  beforeEach(() => {
    switchSubscription.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('only offers active subscription groups on the same platform and shows immediate fallback warnings', async () => {
    const wrapper = mountDialog()
    const targets = wrapper.findAll('[data-switch-target]')
    expect(targets).toHaveLength(1)
    expect(targets[0].text()).toContain('OpenAI Lite')

    await targets[0].trigger('click')
    expect(wrapper.get('[data-quota-warnings]').text()).toContain('admin.subscriptions.switchQuotaWarning.daily')
    expect(wrapper.get('[data-quota-warnings]').text()).toContain('admin.subscriptions.switchQuotaWarning.monthly')
    expect(wrapper.get('[data-quota-warnings]').text()).not.toContain('admin.subscriptions.switchQuotaWarning.weekly')
  })

  it('submits the target with one idempotency key, reports migrated keys, and emits success', async () => {
    switchSubscription.mockResolvedValue({
      subscription: { ...subscription, id: 501, group_id: 20, group: targetGroup },
      previous_subscription_id: 100,
      migrated_keys: 2,
      quota_warnings: ['daily', 'monthly']
    })
    const wrapper = mountDialog()
    await wrapper.get('[data-switch-target]').trigger('click')
    await wrapper.get('[data-switch-submit]').trigger('click')
    await flushPromises()

    expect(switchSubscription).toHaveBeenCalledWith(100, { target_group_id: 20 }, expect.any(String))
    expect(showSuccess).toHaveBeenCalledWith('admin.subscriptions.subscriptionSwitched')
    expect(wrapper.emitted('success')).toHaveLength(1)
  })

  it('keeps the selected target and displays the backend error after failure', async () => {
    switchSubscription.mockRejectedValue({
      message: 'Request failed with status code 409',
      response: { data: { detail: 'source subscription changed' } }
    })
    const wrapper = mountDialog()
    await wrapper.get('[data-switch-target]').trigger('click')
    await wrapper.get('[data-switch-submit]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-switch-error]').text()).toBe('source subscription changed')
    expect(wrapper.get<HTMLInputElement>('[data-switch-target] input').element.checked).toBe(true)
    expect(showError).toHaveBeenCalledWith('source subscription changed')
  })
})
