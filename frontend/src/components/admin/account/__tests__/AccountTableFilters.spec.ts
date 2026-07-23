import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const SelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue', 'change'],
  template: `
    <div>
      <button
        v-for="option in options"
        :key="String(option.value)"
        type="button"
        @click="$emit('update:modelValue', option.value); $emit('change', option.value, option)"
      >
        {{ option.label }}
      </button>
    </div>
  `
})

describe('AccountTableFilters', () => {
  it('uses the backend disabled status when selecting the inactive label', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          type: '',
          status: '',
          privacy_mode: '',
          group: ''
        },
        groups: []
      },
      global: {
        stubs: {
          SearchInput: true,
          Select: SelectStub
        }
      }
    })

    const inactiveOption = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.accounts.status.inactive')

    expect(inactiveOption).toBeDefined()
    await inactiveOption!.trigger('click')

    expect(wrapper.emitted('update:filters')).toContainEqual([
      expect.objectContaining({ status: 'disabled' })
    ])
    expect(wrapper.emitted('change')).toHaveLength(1)
  })
})
