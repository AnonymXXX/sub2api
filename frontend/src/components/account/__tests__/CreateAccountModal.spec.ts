import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const openAIOAuthMock = vi.hoisted(() => ({
  authUrl: { value: '' },
  generateAuthUrl: vi.fn()
}))

const clipboardMock = vi.hoisted(() => ({
  copyToClipboard: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: false
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      generateAuthUrl: vi.fn(),
      exchangeCode: vi.fn(),
      refreshOpenAIToken: vi.fn()
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }),
      getSettings: vi.fn().mockResolvedValue({})
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([])
    },
    gemini: {
      getCapabilities: vi.fn().mockResolvedValue({ ai_studio_oauth_enabled: false }),
      generateAuthUrl: vi.fn(),
      exchangeCode: vi.fn()
    },
    antigravity: {
      generateAuthUrl: vi.fn(),
      exchangeCode: vi.fn(),
      refreshAntigravityToken: vi.fn()
    }
  }
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn().mockResolvedValue({}),
  accountsAPI: {
    syncUpstreamModelsPreview: vi.fn(),
    syncUpstreamModels: vi.fn()
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.count !== undefined ? `${key}:${params.count}` : key
    })
  }
})

vi.mock('@/composables/useAccountOAuth', () => ({
  useAccountOAuth: () => ({
    authUrl: ref(''),
    authCode: ref(''),
    sessionId: ref(''),
    sessionKey: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    exchangeAuthCode: vi.fn(),
    cookieAuth: vi.fn(),
    parseSessionKeys: vi.fn((input: string) => input.split('\n').filter(Boolean)),
    buildExtraInfo: vi.fn(() => undefined)
  })
}))

vi.mock('@/composables/useOpenAIOAuth', () => ({
  useOpenAIOAuth: () => ({
    authUrl: openAIOAuthMock.authUrl,
    sessionId: ref(''),
    oauthState: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn(),
    generateAuthUrl: openAIOAuthMock.generateAuthUrl,
    exchangeAuthCode: vi.fn(),
    validateRefreshToken: vi.fn(),
    buildCredentials: vi.fn(() => ({})),
    buildExtraInfo: vi.fn(() => undefined)
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copied: ref(false),
    copyToClipboard: clipboardMock.copyToClipboard
  })
}))

vi.mock('@/composables/useGeminiOAuth', () => ({
  useGeminiOAuth: () => ({
    authUrl: ref(''),
    sessionId: ref(''),
    state: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    exchangeAuthCode: vi.fn(),
    buildCredentials: vi.fn(() => ({})),
    buildExtraInfo: vi.fn(() => undefined),
    getCapabilities: vi.fn().mockResolvedValue({ ai_studio_oauth_enabled: false })
  })
}))

vi.mock('@/composables/useAntigravityOAuth', () => ({
  useAntigravityOAuth: () => ({
    authUrl: ref(''),
    sessionId: ref(''),
    state: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    exchangeAuthCode: vi.fn(),
    validateRefreshToken: vi.fn(),
    buildCredentials: vi.fn(() => ({}))
  })
}))

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const ModelWhitelistSelectorStub = defineComponent({
  name: 'ModelWhitelistSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <div data-testid="model-whitelist-value">
      {{ Array.isArray(modelValue) ? modelValue.join(',') : '' }}
    </div>
  `
})

const GroupSelectorStub = defineComponent({
  name: 'GroupSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <div data-testid="group-selector-value">
      {{ Array.isArray(modelValue) ? modelValue.join(',') : '' }}
    </div>
  `
})

const SelectStub = defineComponent({
  name: 'AppSelect',
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `
})

const openAIGroup = {
  id: 6,
  name: 'openai',
  description: null,
  platform: 'openai',
  rate_multiplier: 1,
  rpm_limit: 0,
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
  allow_messages_dispatch: false,
  default_mapped_model: '',
  messages_dispatch_model_config: {},
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '2026-06-28T00:00:00Z',
  updated_at: '2026-06-28T00:00:00Z',
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: true,
  supported_model_scopes: [],
  sort_order: 0,
  account_count: 9
}

function mountModal(show = false) {
  return mount(CreateAccountModal, {
    props: {
      show,
      proxies: [],
      groups: [openAIGroup]
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        Select: SelectStub,
        Icon: true,
        ProxySelector: true,
        ProxyAdBanner: true,
        GroupSelector: GroupSelectorStub,
        QuotaLimitCard: true,
        OAuthAuthorizationFlow: true,
        ModelWhitelistSelector: ModelWhitelistSelectorStub
      }
    }
  })
}

describe('CreateAccountModal', () => {
  beforeEach(() => {
    openAIOAuthMock.authUrl.value = ''
    openAIOAuthMock.generateAuthUrl.mockReset().mockImplementation(async () => {
      openAIOAuthMock.authUrl.value = 'https://auth.openai.example/authorize?state=test'
      return true
    })
    clipboardMock.copyToClipboard.mockReset().mockResolvedValue(true)
  })

  it('打开添加账号弹窗时默认使用 GPT-Plus OpenAI 配置', async () => {
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await nextTick()

    expect((wrapper.get('[data-tour="account-form-name"]').element as HTMLInputElement).value).toBe('GPT-Plus')
    expect(wrapper.get('[data-testid="account-platform-openai"]').classes()).toContain('text-green-600')
    expect((wrapper.get('[data-testid="account-concurrency"]').element as HTMLInputElement).value).toBe('3')
    expect((wrapper.get('[data-testid="account-load-factor"]').element as HTMLInputElement).placeholder).toBe('3')
    expect(wrapper.get('[data-testid="group-selector-value"]').text()).toBe('6')
    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe(
      'gpt-5.6-sol,gpt-5.6-terra,gpt-5.6-luna,gpt-5.5,codex-auto-review,gpt-5.4,gpt-5.4-mini'
    )
    expect(wrapper.get('[data-testid="openai-codex-cli-only-toggle"]').classes()).toContain('bg-primary-600')
  })

  it('切换到 OpenAI 时默认选择 Codex 相关模型', async () => {
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await nextTick()

    await wrapper.get('[data-testid="account-platform-openai"]').trigger('click')
    await nextTick()

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe(
      'gpt-5.6-sol,gpt-5.6-terra,gpt-5.6-luna,gpt-5.5,codex-auto-review,gpt-5.4,gpt-5.4-mini'
    )
  })

  it('切换到 OpenAI 时默认选中 openai 分组并使用保守并发', async () => {
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await nextTick()

    expect((wrapper.get('[data-testid="account-concurrency"]').element as HTMLInputElement).value).toBe('3')

    await wrapper.get('[data-testid="account-platform-openai"]').trigger('click')
    await nextTick()

    expect(wrapper.get('[data-testid="group-selector-value"]').text()).toBe('6')
    expect((wrapper.get('[data-testid="account-concurrency"]').element as HTMLInputElement).value).toBe('3')
    expect(wrapper.get('[data-testid="openai-codex-cli-only-toggle"]').classes()).toContain('bg-primary-600')
  })

  it('进入 OpenAI OAuth 授权步骤时自动生成并复制授权链接', async () => {
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await nextTick()
    await wrapper.get('#create-account-form').trigger('submit')
    await flushPromises()

    expect(openAIOAuthMock.generateAuthUrl).toHaveBeenCalledOnce()
    expect(openAIOAuthMock.generateAuthUrl).toHaveBeenCalledWith(null)
    expect(clipboardMock.copyToClipboard).toHaveBeenCalledOnce()
    expect(clipboardMock.copyToClipboard).toHaveBeenCalledWith(
      'https://auth.openai.example/authorize?state=test'
    )
    expect(wrapper.find('#create-account-form').exists()).toBe(false)
  })
})
