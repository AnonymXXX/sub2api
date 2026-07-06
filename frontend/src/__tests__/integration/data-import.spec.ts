import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      importData: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('ImportDataModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    vi.mocked(adminAPI.accounts.importData).mockReset()
  })

  const buildGroup = (overrides: Partial<AdminGroup>): AdminGroup => ({
    id: 1,
    name: 'openai-default',
    description: null,
    platform: 'openai',
    rate_multiplier: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard',
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    allow_image_generation: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    image_price_1k: null,
    image_price_2k: null,
    image_price_4k: null,
    claude_code_only: false,
    fallback_group_id: null,
    fallback_group_id_on_invalid_request: null,
    require_oauth_only: false,
    require_privacy_set: false,
    created_at: '2026-07-06T00:00:00Z',
    updated_at: '2026-07-06T00:00:00Z',
    model_routing: null,
    model_routing_enabled: false,
    mcp_xml_inject: false,
    ...overrides
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时提示解析失败', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const file = new File(['invalid json'], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('invalid json')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailed')
  })

  it('默认使用容量 5、优先级 1，并选中当前 OpenAI 分组提交导入', async () => {
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_updated: 0,
      account_failed: 0
    })

    const wrapper = mount(ImportDataModal, {
      props: {
        show: true,
        groups: [
          buildGroup({ id: 11, name: 'Claude', platform: 'anthropic' }),
          buildGroup({ id: 22, name: 'OpenAI 当前组', platform: 'openai' })
        ],
        currentGroupId: 22
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const concurrencyInput = wrapper.get('[data-test="data-import-concurrency"]')
    const priorityInput = wrapper.get('[data-test="data-import-priority"]')
    const groupSelect = wrapper.get('[data-test="data-import-group"]')

    expect((concurrencyInput.element as HTMLInputElement).value).toBe('5')
    expect((priorityInput.element as HTMLInputElement).value).toBe('1')
    expect((groupSelect.element as HTMLSelectElement).value).toBe('22')

    const input = wrapper.find('input[type="file"]')
    const file = new File([
      JSON.stringify({
        exported_at: '2026-07-06T00:00:00Z',
        proxies: [],
        accounts: [
          {
            name: 'acc',
            platform: 'openai',
            type: 'oauth',
            credentials: { token: 'x' },
            concurrency: 3,
            priority: 50
          }
        ]
      })
    ], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve(JSON.stringify({
        exported_at: '2026-07-06T00:00:00Z',
        proxies: [],
        accounts: [
          {
            name: 'acc',
            platform: 'openai',
            type: 'oauth',
            credentials: { token: 'x' },
            concurrency: 3,
            priority: 50
          }
        ]
      }))
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith(expect.objectContaining({
      concurrency: 5,
      priority: 1,
      group_ids: [22],
      skip_default_group_bind: true
    }))
  })
})
