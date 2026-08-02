<template>
  <BaseDialog
    :show="show"
    :title="t('admin.subscriptions.switchSubscription')"
    width="narrow"
    @close="close"
  >
    <div v-if="subscription" class="space-y-4">
      <div class="flex min-w-0 items-center gap-3 border-b border-gray-200 pb-4 dark:border-dark-600">
        <div class="min-w-0 flex-1">
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.subscriptions.currentPlan') }}
          </p>
          <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
            {{ subscription.group?.name || `#${subscription.group_id}` }}
          </p>
        </div>
        <Icon name="arrowRight" size="sm" class="shrink-0 text-gray-400" />
        <div class="min-w-0 flex-1 text-right">
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.subscriptions.targetPlan') }}
          </p>
          <p class="truncate text-sm font-medium text-primary-600 dark:text-primary-400">
            {{ selectedTarget?.name || t('admin.subscriptions.selectTargetPlan') }}
          </p>
        </div>
      </div>

      <div class="space-y-1 text-sm text-gray-600 dark:text-gray-300">
        <p>
          {{ t('admin.subscriptions.expirationUnchanged') }}:
          <span class="font-medium text-gray-900 dark:text-white">
            {{ subscription.expires_at ? formatDateOnly(subscription.expires_at) : t('admin.subscriptions.noExpiration') }}
          </span>
        </p>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.subscriptions.switchCarryRule') }}
        </p>
      </div>

      <div v-if="availableTargets.length" class="max-h-64 space-y-2 overflow-y-auto pr-1">
        <label
          v-for="group in availableTargets"
          :key="group.id"
          data-switch-target
          class="flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors"
          :class="selectedGroupId === group.id
            ? 'border-primary-400 bg-primary-50 dark:border-primary-500 dark:bg-primary-900/20'
            : 'border-gray-200 hover:border-gray-300 dark:border-dark-600 dark:hover:border-dark-500'"
          @click="selectTarget(group.id)"
        >
          <input
            type="radio"
            name="target-subscription-group"
            :value="group.id"
            :checked="selectedGroupId === group.id"
            class="mt-0.5 h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500"
            @change="selectTarget(group.id)"
          />
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">
              {{ group.name }}
            </span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
              {{ quotaSummary(group) }}
            </span>
          </span>
        </label>
      </div>
      <p v-else class="py-5 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.subscriptions.noSwitchTargets') }}
      </p>

      <div
        v-if="selectedWarnings.length"
        data-quota-warnings
        class="rounded-lg border border-amber-300 bg-amber-50 p-3 text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
      >
        <div class="flex items-start gap-2">
          <Icon name="exclamationTriangle" size="sm" class="mt-0.5 shrink-0" />
          <div class="min-w-0 text-xs leading-5">
            <p class="font-medium">{{ t('admin.subscriptions.switchQuotaWarning.title') }}</p>
            <ul class="mt-1 list-disc pl-4">
              <li v-for="warning in selectedWarnings" :key="warning">
                {{ t(`admin.subscriptions.switchQuotaWarning.${warning}`) }}
              </li>
            </ul>
          </div>
        </div>
      </div>

      <p
        v-if="errorMessage"
        data-switch-error
        class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300"
      >
        {{ errorMessage }}
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="close">
          {{ t('common.cancel') }}
        </button>
        <button
          data-switch-submit
          type="button"
          class="btn btn-primary"
          :disabled="!selectedGroupId || submitting"
          @click="submit"
        >
          <Icon name="swap" size="sm" class="mr-2" />
          {{ submitting ? t('admin.subscriptions.switching') : t('admin.subscriptions.switch') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { Group, SubscriptionQuotaWarning, SwitchSubscriptionResult, UserSubscription } from '@/types'
import { formatDateOnly } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
  subscription: UserSubscription | null
  groups: Group[]
}>()

const emit = defineEmits<{
  close: []
  success: [result: SwitchSubscriptionResult]
}>()

const { t } = useI18n()
const appStore = useAppStore()
const selectedGroupId = ref<number | null>(null)
const submitting = ref(false)
const errorMessage = ref('')
const idempotencyKey = ref('')

const availableTargets = computed(() => {
  const current = props.subscription
  if (!current?.group) return []
  return props.groups.filter(
    (group) =>
      group.id !== current.group_id &&
      group.platform === current.group?.platform &&
      group.status === 'active' &&
      group.subscription_type === 'subscription'
  )
})

const selectedTarget = computed(() =>
  availableTargets.value.find((group) => group.id === selectedGroupId.value)
)

const selectedWarnings = computed<SubscriptionQuotaWarning[]>(() => {
  const subscription = props.subscription
  const target = selectedTarget.value
  if (!subscription || !target) return []
  const warnings: SubscriptionQuotaWarning[] = []
  if (target.daily_limit_usd && subscription.daily_usage_usd >= target.daily_limit_usd) {
    warnings.push('daily')
  }
  if (target.weekly_limit_usd && subscription.weekly_usage_usd >= target.weekly_limit_usd) {
    warnings.push('weekly')
  }
  const effectiveMonthlyLimit = (target.monthly_limit_usd || 0) + subscription.monthly_bonus_usd
  if (target.monthly_limit_usd && subscription.monthly_usage_usd >= effectiveMonthlyLimit) {
    warnings.push('monthly')
  }
  return warnings
})

watch(
  () => [props.show, props.subscription?.id],
  ([show]) => {
    if (show) {
      selectedGroupId.value = null
      errorMessage.value = ''
      idempotencyKey.value = createIdempotencyKey()
    }
  },
  { immediate: true }
)

function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `subscription-switch-${crypto.randomUUID()}`
  }
  return `subscription-switch-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const selectTarget = (groupId: number) => {
  if (selectedGroupId.value !== groupId) {
    selectedGroupId.value = groupId
    errorMessage.value = ''
    idempotencyKey.value = createIdempotencyKey()
  }
}

const quotaSummary = (group: Group): string => {
  const limits = [
    group.daily_limit_usd ? t('admin.subscriptions.switchQuotaDaily', { value: group.daily_limit_usd }) : '',
    group.weekly_limit_usd ? t('admin.subscriptions.switchQuotaWeekly', { value: group.weekly_limit_usd }) : '',
    group.monthly_limit_usd ? t('admin.subscriptions.switchQuotaMonthly', { value: group.monthly_limit_usd }) : ''
  ].filter(Boolean)
  return limits.length ? limits.join(' · ') : t('admin.subscriptions.unlimited')
}

const close = () => {
  if (!submitting.value) emit('close')
}

const submit = async () => {
  if (!props.subscription || !selectedGroupId.value || submitting.value) return
  submitting.value = true
  errorMessage.value = ''
  try {
    const result = await adminAPI.subscriptions.switchSubscription(
      props.subscription.id,
      { target_group_id: selectedGroupId.value },
      idempotencyKey.value
    )
    appStore.showSuccess(
      t('admin.subscriptions.subscriptionSwitched', {
        group: selectedTarget.value?.name || `#${selectedGroupId.value}`,
        count: result.migrated_keys
      })
    )
    emit('success', result)
  } catch (error: any) {
    errorMessage.value =
      error?.response?.data?.detail ||
      error?.response?.data?.message ||
      error?.message ||
      t('admin.subscriptions.failedToSwitch')
    appStore.showError(errorMessage.value)
  } finally {
    submitting.value = false
  }
}
</script>
