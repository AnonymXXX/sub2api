<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.routingAudit.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.routingAudit.description') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="refresh">
          {{ loading ? '刷新中' : '刷新' }}
        </button>
      </div>

      <div class="card p-4">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">开始日期</span>
            <input v-model="filters.start_date" type="date" class="input" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">结束日期</span>
            <input v-model="filters.end_date" type="date" class="input" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">用户 ID</span>
            <input v-model.number="filters.user_id" type="number" min="1" class="input" placeholder="全部" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">API Key ID</span>
            <input v-model.number="filters.api_key_id" type="number" min="1" class="input" placeholder="全部" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">账号 ID</span>
            <input v-model.number="filters.account_id" type="number" min="1" class="input" placeholder="全部" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">模型</span>
            <input v-model.trim="filters.model" type="text" class="input" placeholder="全部" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">分流池</span>
            <input v-model.trim="filters.selected_pool" type="text" class="input" placeholder="relay-apipod / trusted-plus" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">决策原因</span>
            <input v-model.trim="filters.decision_reason" type="text" class="input" placeholder="scheduler_load_balance" />
          </label>
          <label class="space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">策略版本</span>
            <input v-model.trim="filters.routing_policy_version" type="text" class="input" placeholder="routing-audit-v1" />
          </label>
          <div class="flex items-end gap-2 xl:col-span-3">
            <button class="btn btn-primary" @click="applyFilters">查询</button>
            <button class="btn btn-ghost" @click="resetFilters">重置</button>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 xl:grid-cols-6">
        <div v-for="item in summaryCards" :key="item.label" class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">分流池分布</h2>
          </div>
          <AuditDimensionTable :rows="summary?.by_pool ?? []" />
        </div>
        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">决策原因分布</h2>
          </div>
          <AuditDimensionTable :rows="summary?.by_reason ?? []" />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <SnapshotPanel title="Plus 号池快照" :items="summary?.latest_plus_pool_snapshot ?? []" />
        <SnapshotPanel title="中转额度快照" :items="summary?.latest_relay_quota_snapshot ?? []" />
      </div>

      <div class="card overflow-hidden">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">请求明细</h2>
          <span class="text-xs text-gray-500 dark:text-gray-400">共 {{ pagination.total.toLocaleString() }} 条</span>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th v-for="col in columns" :key="col" class="whitespace-nowrap px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400">
                  {{ col }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-if="loading">
                <td :colspan="columns.length" class="px-4 py-8 text-center text-sm text-gray-500">加载中...</td>
              </tr>
              <tr v-else-if="logs.length === 0">
                <td :colspan="columns.length" class="px-4 py-8 text-center text-sm text-gray-500">暂无审计日志</td>
              </tr>
              <tr v-for="row in logs" v-else :key="row.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/60">
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(row.created_at) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ row.user_id || '-' }} / {{ row.api_key_id || '-' }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ row.model || '-' }}</td>
                <td class="whitespace-nowrap px-4 py-3">
                  <span class="rounded bg-sky-50 px-2 py-1 text-xs font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-200">{{ row.selected_pool || 'unknown' }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ row.selected_account_id || row.account_id || '-' }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-600 dark:text-gray-300">{{ row.decision_reason || '-' }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-600 dark:text-gray-300">
                  <span v-if="row.privacy_redirect" class="mr-1 rounded bg-amber-50 px-1.5 py-0.5 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">隐私</span>
                  <span v-if="row.quota_redirect" class="mr-1 rounded bg-violet-50 px-1.5 py-0.5 text-violet-700 dark:bg-violet-900/30 dark:text-violet-200">额度</span>
                  <span v-if="row.error_redirect" class="mr-1 rounded bg-rose-50 px-1.5 py-0.5 text-rose-700 dark:bg-rose-900/30 dark:text-rose-200">错误</span>
                  <span v-if="!row.privacy_redirect && !row.quota_redirect && !row.error_redirect">-</span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-xs" :class="row.success ? 'text-emerald-600' : 'text-rose-600'">
                  {{ row.success ? '成功' : row.error_type || '失败' }}
                  <span v-if="row.upstream_status">({{ row.upstream_status }})</span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ formatTokens(row) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ formatCurrency(row.actual_cost) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ row.duration_ms }} ms</td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import { adminAPI } from '@/api/admin'
import type { PoolSnapshot, RoutingAuditDimensionStat, RoutingAuditLog, RoutingAuditQueryParams, RoutingAuditSummary } from '@/api/admin'
import { formatCurrency, formatDateTime, formatNumber } from '@/utils/format'

const { t } = useI18n()

const columns = ['时间', '用户/API Key', '模型', '分流池', '账号', '原因', '改道', '状态', 'Tokens', '费用', '耗时']

const loading = ref(false)
const logs = ref<RoutingAuditLog[]>([])
const summary = ref<RoutingAuditSummary | null>(null)
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const filters = reactive<RoutingAuditQueryParams>({
  start_date: formatDateInput(daysAgo(2)),
  end_date: formatDateInput(new Date()),
  sort_by: 'created_at',
  sort_order: 'desc'
})

const summaryCards = computed(() => {
  const s = summary.value
  return [
    { label: '总请求', value: formatNumber(s?.total_requests ?? 0) },
    { label: '成功', value: formatNumber(s?.success_requests ?? 0) },
    { label: '失败', value: formatNumber(s?.failed_requests ?? 0) },
    { label: '改道', value: `${formatNumber((s?.privacy_redirect_count ?? 0) + (s?.quota_redirect_count ?? 0) + (s?.error_redirect_count ?? 0))}` },
    { label: '总 Token', value: formatNumber(s?.total_tokens ?? 0) },
    { label: '实际费用', value: formatCurrency(s?.total_actual_cost ?? 0) }
  ]
})

async function loadData() {
  loading.value = true
  try {
    const params = buildParams()
    const [listRes, summaryRes] = await Promise.all([
      adminAPI.routingAudit.list(params),
      adminAPI.routingAudit.summary(params)
    ])
    logs.value = listRes.items
    pagination.total = listRes.total
    pagination.page = listRes.page
    pagination.page_size = listRes.page_size
    summary.value = summaryRes
  } finally {
    loading.value = false
  }
}

function buildParams(): RoutingAuditQueryParams {
  const params: RoutingAuditQueryParams = {
    ...filters,
    page: pagination.page,
    page_size: pagination.page_size
  }
  for (const key of Object.keys(params) as Array<keyof RoutingAuditQueryParams>) {
    const value = params[key]
    if (value === '' || value === undefined || value === null || value === 0) {
      delete params[key]
    }
  }
  return params
}

function applyFilters() {
  pagination.page = 1
  void loadData()
}

function refresh() {
  void loadData()
}

function resetFilters() {
  Object.assign(filters, {
    start_date: formatDateInput(daysAgo(2)),
    end_date: formatDateInput(new Date()),
    user_id: undefined,
    api_key_id: undefined,
    account_id: undefined,
    group_id: undefined,
    model: '',
    selected_pool: '',
    decision_reason: '',
    routing_policy_version: '',
    sort_by: 'created_at',
    sort_order: 'desc'
  })
  applyFilters()
}

function onPageChange(page: number) {
  pagination.page = page
  void loadData()
}

function onPageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadData()
}

function formatTokens(row: RoutingAuditLog): string {
  return formatNumber(row.input_tokens + row.output_tokens + row.cache_creation_tokens + row.cache_read_tokens + row.image_output_tokens)
}

function daysAgo(days: number): Date {
  const date = new Date()
  date.setDate(date.getDate() - days)
  return date
}

function formatDateInput(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const AuditDimensionTable = defineComponent({
  name: 'AuditDimensionTable',
  props: {
    rows: { type: Array as () => RoutingAuditDimensionStat[], required: true }
  },
  setup(props) {
    return () =>
      h('div', { class: 'overflow-x-auto' }, [
        h('table', { class: 'min-w-full divide-y divide-gray-100 dark:divide-dark-700' }, [
          h('thead', { class: 'bg-gray-50 dark:bg-dark-800' }, [
            h('tr', [
              tableHead('维度'),
              tableHead('请求'),
              tableHead('成功率'),
              tableHead('Token'),
              tableHead('费用')
            ])
          ]),
          h('tbody', { class: 'divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800' },
            props.rows.length === 0
              ? [h('tr', [h('td', { colspan: 5, class: 'px-4 py-6 text-center text-sm text-gray-500' }, '暂无数据')])]
              : props.rows.map((row) =>
                  h('tr', { key: row.key }, [
                    tableCell(row.key || 'unknown'),
                    tableCell(formatNumber(row.requests)),
                    tableCell(row.requests > 0 ? `${Math.round((row.success_requests / row.requests) * 100)}%` : '-'),
                    tableCell(formatNumber(row.total_tokens)),
                    tableCell(formatCurrency(row.total_actual_cost))
                  ])
                )
          )
        ])
      ])
  }
})

const SnapshotPanel = defineComponent({
  name: 'SnapshotPanel',
  props: {
    title: { type: String, required: true },
    items: { type: Array as () => PoolSnapshot[], required: true }
  },
  setup(props) {
    return () =>
      h('div', { class: 'card overflow-hidden' }, [
        h('div', { class: 'border-b border-gray-100 px-4 py-3 dark:border-dark-700' }, [
          h('h2', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, props.title)
        ]),
        h('div', { class: 'divide-y divide-gray-100 dark:divide-dark-700' },
          props.items.length === 0
            ? [h('div', { class: 'px-4 py-6 text-center text-sm text-gray-500' }, '暂无快照')]
            : props.items.map((item) =>
                h('div', { key: item.pool, class: 'px-4 py-3' }, [
                  h('div', { class: 'flex items-center justify-between gap-3' }, [
                    h('span', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, item.pool || 'unknown'),
                    h('span', { class: 'text-xs text-gray-500 dark:text-gray-400' }, `${item.schedulable_accounts}/${item.total_accounts} 可调度`)
                  ]),
                  h('div', { class: 'mt-2 grid grid-cols-2 gap-2 text-xs text-gray-600 dark:text-gray-300 md:grid-cols-4' }, [
                    h('span', `5h max ${formatPercent(item.codex_5h_max_used_percent)}`),
                    h('span', `7d max ${formatPercent(item.codex_7d_max_used_percent)}`),
                    h('span', `日用量 ${formatMoneyPair(item.quota_daily_used_usd, item.quota_daily_limit_usd)}`),
                    h('span', `周用量 ${formatMoneyPair(item.quota_weekly_used_usd, item.quota_weekly_limit_usd)}`)
                  ])
                ])
              )
        )
      ])
  }
})

function tableHead(text: string) {
  return h('th', { class: 'whitespace-nowrap px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400' }, text)
}

function tableCell(text: string) {
  return h('td', { class: 'whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-200' }, text)
}

function formatPercent(value?: number): string {
  if (value === undefined || value === null) return '-'
  return `${value.toFixed(1)}%`
}

function formatMoneyPair(used?: number, limit?: number): string {
  if (!used && !limit) return '-'
  if (!limit) return formatCurrency(used ?? 0)
  return `${formatCurrency(used ?? 0)} / ${formatCurrency(limit)}`
}

onMounted(loadData)
</script>
