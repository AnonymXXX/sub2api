import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'

const oauthMocks = vi.hoisted(() => {
  const createOAuthMock = () => ({
    authUrl: { value: '' },
    sessionId: { value: '' },
    oauthState: { value: '' },
    state: { value: '' },
    loading: { value: false },
    error: { value: '' },
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    exchangeAuthCode: vi.fn(),
    buildCredentials: vi.fn(() => ({})),
    buildExtraInfo: vi.fn(() => ({}))
  })

  return {
    account: createOAuthMock(),
    openai: createOAuthMock(),
    gemini: createOAuthMock(),
    antigravity: createOAuthMock(),
    grok: createOAuthMock(),
    copyToClipboard: vi.fn()
  }
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { accounts: {} }
}))

vi.mock('@/composables/useAccountOAuth', () => ({
  useAccountOAuth: () => oauthMocks.account
}))

vi.mock('@/composables/useOpenAIOAuth', () => ({
  useOpenAIOAuth: () => oauthMocks.openai
}))

vi.mock('@/composables/useGeminiOAuth', () => ({
  useGeminiOAuth: () => oauthMocks.gemini
}))

vi.mock('@/composables/useAntigravityOAuth', () => ({
  useAntigravityOAuth: () => oauthMocks.antigravity
}))

vi.mock('@/composables/useGrokOAuth', () => ({
  useGrokOAuth: () => oauthMocks.grok
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: oauthMocks.copyToClipboard })
}))

import ReAuthAccountModal from '../ReAuthAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: ['show', 'title', 'width'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const OAuthAuthorizationFlowStub = defineComponent({
  name: 'OAuthAuthorizationFlow',
  setup(_, { expose }) {
    expose({
      authCode: '',
      oauthState: '',
      projectId: '',
      sessionKey: '',
      inputMethod: 'manual',
      reset: vi.fn()
    })
  },
  template: '<div />'
})

const openAIAccount = {
  id: 42,
  name: 'GPT-Plus',
  platform: 'openai',
  type: 'oauth',
  status: 'active',
  proxy_id: 7,
  credentials: {}
}

const mountModal = (show = false) =>
  mount(ReAuthAccountModal, {
    props: {
      show,
      account: openAIAccount
    } as any,
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        OAuthAuthorizationFlow: OAuthAuthorizationFlowStub,
        Icon: true
      }
    }
  })

describe('ReAuthAccountModal', () => {
  beforeEach(() => {
    oauthMocks.openai.authUrl.value = ''
    oauthMocks.openai.generateAuthUrl.mockReset()
    oauthMocks.copyToClipboard.mockReset()
  })

  it('使用适合授权流程的宽弹窗', () => {
    const wrapper = mountModal()

    expect(wrapper.findComponent({ name: 'BaseDialog' }).props('width')).toBe('wide')
  })

  it('OpenAI 弹窗打开后自动生成并复制授权链接', async () => {
    const wrapper = mountModal()
    oauthMocks.openai.generateAuthUrl.mockImplementation(async () => {
      oauthMocks.openai.authUrl.value = 'https://auth.openai.example/authorize?state=test'
      return true
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(oauthMocks.openai.generateAuthUrl).toHaveBeenCalledOnce()
    expect(oauthMocks.openai.generateAuthUrl).toHaveBeenCalledWith(7)
    expect(oauthMocks.copyToClipboard).toHaveBeenCalledWith(
      'https://auth.openai.example/authorize?state=test'
    )
  })

  it('初始已打开时也会自动生成并复制授权链接', async () => {
    oauthMocks.openai.generateAuthUrl.mockImplementation(async () => {
      oauthMocks.openai.authUrl.value = 'https://auth.openai.example/authorize?state=initial'
      return true
    })

    mountModal(true)
    await flushPromises()

    expect(oauthMocks.openai.generateAuthUrl).toHaveBeenCalledOnce()
    expect(oauthMocks.copyToClipboard).toHaveBeenCalledWith(
      'https://auth.openai.example/authorize?state=initial'
    )
  })

  it('生成授权链接期间关闭弹窗时不再复制', async () => {
    const wrapper = mountModal()
    let resolveGeneration!: (generated: boolean) => void
    oauthMocks.openai.generateAuthUrl.mockImplementation(
      () =>
        new Promise<boolean>((resolve) => {
          resolveGeneration = resolve
        })
    )

    await wrapper.setProps({ show: true })
    await wrapper.setProps({ show: false })
    oauthMocks.openai.authUrl.value = 'https://auth.openai.example/authorize?state=stale'
    resolveGeneration(true)
    await flushPromises()

    expect(oauthMocks.copyToClipboard).not.toHaveBeenCalled()
  })
})
