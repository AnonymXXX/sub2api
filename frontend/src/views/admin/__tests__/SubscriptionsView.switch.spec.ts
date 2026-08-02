import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { UserSubscription } from '@/types'
import SubscriptionsView from '../SubscriptionsView.vue'

const { listSubscriptions, getAllGroups } = vi.hoisted(() => ({
  listSubscriptions: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    subscriptions: {
      list: listSubscriptions,
      assign: vi.fn(),
      extend: vi.fn(),
      revoke: vi.fn(),
      restore: vi.fn(),
      resetQuota: vi.fn(),
      setMonthlyBonus: vi.fn()
    },
    groups: { getAll: getAllGroups },
    usage: { searchUsers: vi.fn() }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const createSubscription = (id: number, status: UserSubscription['status']): UserSubscription =>
  ({
    id,
    user_id: 15,
    group_id: 10,
    status,
    starts_at: '2026-07-10T08:00:00Z',
    expires_at: '2026-08-09T08:00:00Z',
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    monthly_bonus_usd: 0,
    effective_monthly_limit_usd: 100,
    renewal_eligible: false,
    renewal_price: null,
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-07-10T08:00:00Z',
    updated_at: '2026-07-10T08:00:00Z',
    user: { id: 15, email: 'user@example.com', username: 'user' },
    group: {
      id: 10,
      name: 'OpenAI Pro',
      platform: 'openai',
      status: 'active',
      subscription_type: 'subscription',
      rate_multiplier: 1,
      description: ''
    }
  }) as UserSubscription

const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" :data-row-id="row.id">
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
}

const SwitchDialogStub = {
  props: ['show', 'subscription'],
  emits: ['close', 'success'],
  template: `
    <div v-if="show" data-switch-dialog :data-subscription-id="subscription?.id">
      <button data-switch-success @click="$emit('success', {})">success</button>
    </div>
  `
}

describe('SubscriptionsView switch action', () => {
  beforeEach(() => {
    listSubscriptions.mockReset()
    getAllGroups.mockReset()
    listSubscriptions.mockResolvedValue({
      items: [createSubscription(100, 'active'), createSubscription(101, 'expired')],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getAllGroups.mockResolvedValue([])
  })

  it('shows switch only for active subscriptions and refreshes after success', async () => {
    const wrapper = mount(SubscriptionsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: true,
          ConfirmDialog: true,
          EmptyState: true,
          Select: true,
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true,
          SubscriptionSwitchDialog: SwitchDialogStub,
          RouterLink: true,
          Teleport: true
        }
      }
    })
    await flushPromises()

    const activeRow = wrapper.get('[data-row-id="100"]')
    const expiredRow = wrapper.get('[data-row-id="101"]')
    expect(activeRow.text()).toContain('admin.subscriptions.switch')
    expect(expiredRow.text()).not.toContain('admin.subscriptions.switch')

    const switchButton = activeRow.findAll('button').find((button) =>
      button.text().includes('admin.subscriptions.switch')
    )
    expect(switchButton).toBeTruthy()
    await switchButton!.trigger('click')
    expect(wrapper.get('[data-switch-dialog]').attributes('data-subscription-id')).toBe('100')

    await wrapper.get('[data-switch-success]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-switch-dialog]').exists()).toBe(false)
    expect(listSubscriptions).toHaveBeenCalledTimes(2)
  })
})
