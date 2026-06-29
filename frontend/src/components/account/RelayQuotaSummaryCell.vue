<template>
  <div v-if="summary" class="space-y-1">
    <div class="flex flex-wrap items-center gap-1 text-[9px] text-gray-500 dark:text-gray-400">
      <span
        v-if="balanceText"
        class="rounded bg-emerald-50 px-1.5 py-0.5 font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
        :title="balanceTitle"
      >
        余额 {{ balanceText }}
      </span>
      <span
        v-if="planText"
        class="rounded bg-blue-50 px-1.5 py-0.5 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
        :title="subscriptionTitle"
      >
        {{ planText }}
      </span>
      <button
        v-if="summary.refreshable"
        type="button"
        class="inline-flex h-5 w-5 items-center justify-center rounded border border-gray-200 text-gray-500 hover:bg-gray-100 disabled:cursor-wait disabled:opacity-60 dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-800"
        :disabled="refreshing"
        title="刷新余额/订阅"
        @click="refresh"
      >
        <Icon name="refresh" size="xs" :class="{ 'animate-spin': refreshing }" />
      </button>
    </div>

    <div class="flex flex-wrap items-center gap-1 text-[9px] text-gray-500 dark:text-gray-400">
      <span v-if="limitText" class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-800">
        {{ limitText }}
      </span>
      <span v-if="priceText" class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-800">
        {{ priceText }}
      </span>
      <span v-if="syncedText" class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-800" :title="syncedTitle">
        {{ syncedText }}
      </span>
    </div>

    <div v-if="errorText" class="text-[9px] leading-tight text-amber-600 dark:text-amber-400" :title="errorText">
      {{ errorText }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { Account, RelayQuotaSummary } from '@/types'
import { formatCurrency, formatRelativeTime } from '@/utils/format'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  account: Account
}>()

const localSummary = ref<RelayQuotaSummary | null>(null)
const refreshing = ref(false)
const refreshError = ref<string | null>(null)

const summary = computed(() => localSummary.value ?? props.account.relay_quota_summary ?? null)

const balanceText = computed(() => {
  const value = summary.value?.balance_remaining
  if (value == null) return ''
  return formatCurrency(value, summary.value?.balance_unit || 'USD')
})

const balanceTitle = computed(() => {
  const parts: string[] = []
  if (summary.value?.balance_source) parts.push(`来源：${summary.value.balance_source}`)
  if (summary.value?.balance_synced_at) parts.push(`同步：${summary.value.balance_synced_at}`)
  return parts.join('\n')
})

const planText = computed(() => {
  const name = summary.value?.plan_name
  if (!name) return ''
  const multiplier = summary.value?.rate_multiplier
  return multiplier != null ? `${name} x${multiplier}` : name
})

const subscriptionTitle = computed(() => {
  const parts: string[] = []
  if (summary.value?.subscription_status) parts.push(`状态：${summary.value.subscription_status}`)
  if (summary.value?.subscription_source) parts.push(`来源：${summary.value.subscription_source}`)
  if (summary.value?.subscription_refreshable === false) parts.push('未配置后台订阅 token，使用本地套餐配置')
  return parts.join('\n')
})

const limitText = computed(() => {
  const parts: string[] = []
  if (summary.value?.daily_limit_usd != null) parts.push(`日 $${formatPlainMoney(summary.value.daily_limit_usd)}`)
  if (summary.value?.weekly_limit_usd != null) parts.push(`周 $${formatPlainMoney(summary.value.weekly_limit_usd)}`)
  if (summary.value?.monthly_limit_usd != null) parts.push(`月 $${formatPlainMoney(summary.value.monthly_limit_usd)}`)
  return parts.join(' / ')
})

const priceText = computed(() => {
  const price = summary.value?.plan_price
  if (price == null) return ''
  const currency = summary.value?.plan_currency || 'CNY'
  const days = summary.value?.validity_days
  return `${currency} ${formatPlainMoney(price)}${days ? ` / ${days}天` : ''}`
})

const syncedText = computed(() => {
  const value = summary.value?.last_synced_at || summary.value?.balance_synced_at
  if (!value) return ''
  return formatRelativeTime(value)
})

const syncedTitle = computed(() => summary.value?.last_synced_at || summary.value?.balance_synced_at || '')

const errorText = computed(() => refreshError.value || summary.value?.error || '')

const formatPlainMoney = (value: number): string => {
  if (!Number.isFinite(value)) return '0'
  return Number.isInteger(value) ? value.toFixed(0) : value.toFixed(2)
}

const refresh = async () => {
  if (refreshing.value) return
  refreshing.value = true
  refreshError.value = null
  try {
    const result = await adminAPI.accounts.refreshRelayQuota(props.account.id)
    localSummary.value = result.summary
  } catch (err: any) {
    refreshError.value = err?.response?.data?.message || err?.message || '刷新失败'
  } finally {
    refreshing.value = false
  }
}
</script>
