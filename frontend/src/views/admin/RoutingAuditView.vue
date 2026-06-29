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
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
            <span class="h-2 w-2 rounded-full" :class="item.dotClass"></span>
          </div>
          <p class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
          <p v-if="item.hint" class="mt-1 truncate text-[11px] text-gray-400 dark:text-gray-500">{{ item.hint }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-12">
        <ChartPanel class="xl:col-span-4" title="请求结果" subtitle="成功 / 失败占比">
          <Doughnut v-if="outcomeChartData" :data="outcomeChartData" :options="doughnutOptions" />
          <EmptyChart v-else />
        </ChartPanel>

        <ChartPanel class="xl:col-span-4" title="改道构成" subtitle="隐私、额度和错误改道">
          <Doughnut v-if="redirectChartData" :data="redirectChartData" :options="doughnutOptions" />
          <EmptyChart v-else />
        </ChartPanel>

        <ChartPanel class="xl:col-span-4" title="Token 构成" subtitle="输入、输出、缓存占比">
          <Doughnut v-if="tokenMixChartData" :data="tokenMixChartData" :options="doughnutOptions" />
          <EmptyChart v-else />
        </ChartPanel>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <ChartPanel title="分流池请求分布" subtitle="看 APIPod / Plus 分流是否符合预期">
          <Bar v-if="poolRequestChartData" :data="poolRequestChartData" :options="horizontalCountBarOptions" />
          <EmptyChart v-else />
        </ChartPanel>

        <ChartPanel title="决策原因费用分布" subtitle="看成本集中在哪类调度原因">
          <Bar v-if="reasonCostChartData" :data="reasonCostChartData" :options="horizontalCurrencyBarOptions" />
          <EmptyChart v-else />
        </ChartPanel>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <DistributionPanel
          title="分流池明细"
          subtitle="请求、成功率、Token 和费用"
          :rows="poolDistributionRows"
          value-label="请求"
        />
        <DistributionPanel
          title="决策原因明细"
          subtitle="请求、成功率、Token 和费用"
          :rows="reasonDistributionRows"
          value-label="请求"
        />
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <SnapshotPanel title="Plus 号池压力" :items="summary?.latest_plus_pool_snapshot ?? []" />
        <SnapshotPanel title="中转额度压力" :items="summary?.latest_relay_quota_snapshot ?? []" />
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <ChartPanel title="当前页模型 Token" subtitle="基于下方明细当前页聚合">
          <Bar v-if="modelTokenChartData" :data="modelTokenChartData" :options="horizontalCountBarOptions" />
          <EmptyChart v-else />
        </ChartPanel>

        <ChartPanel title="当前页耗时分布" subtitle="基于下方明细当前页聚合">
          <Bar v-if="latencyChartData" :data="latencyChartData" :options="latencyBarOptions" />
          <EmptyChart v-else />
        </ChartPanel>
      </div>

      <div class="card overflow-hidden">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <div>
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">请求明细</h2>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">图表判断异常后，在这里查看具体路由记录</p>
          </div>
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
import {
  ArcElement,
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Tooltip
} from 'chart.js'
import type { ChartData, ChartOptions } from 'chart.js'
import { Bar, Doughnut } from 'vue-chartjs'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import { adminAPI } from '@/api/admin'
import type { PoolSnapshot, RoutingAuditDimensionStat, RoutingAuditLog, RoutingAuditQueryParams, RoutingAuditSummary } from '@/api/admin'
import { formatCurrency, formatDateTime, formatNumber } from '@/utils/format'

ChartJS.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip, Legend)

type DistributionRow = RoutingAuditDimensionStat & {
  color: string
  requestPercent: number
  successRate: number
}

type SummaryCard = {
  label: string
  value: string
  hint?: string
  dotClass: string
}

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

const chartColors = ['#14b8a6', '#2563eb', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4', '#84cc16', '#f97316']
const outcomeColors = ['#10b981', '#ef4444']
const redirectColors = ['#f59e0b', '#8b5cf6', '#ef4444']
const tokenColors = ['#2563eb', '#14b8a6', '#f59e0b', '#8b5cf6', '#06b6d4']

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))
const chartTheme = computed(() => ({
  grid: isDarkMode.value ? '#374151' : '#eef2f7',
  text: isDarkMode.value ? '#9ca3af' : '#64748b',
  tooltipBg: isDarkMode.value ? '#111827' : '#ffffff',
  tooltipTitle: isDarkMode.value ? '#f9fafb' : '#111827',
  tooltipBody: isDarkMode.value ? '#e5e7eb' : '#475569'
}))

const totalRedirects = computed(() => {
  const s = summary.value
  return (s?.privacy_redirect_count ?? 0) + (s?.quota_redirect_count ?? 0) + (s?.error_redirect_count ?? 0)
})

const summaryCards = computed<SummaryCard[]>(() => {
  const s = summary.value
  const total = s?.total_requests ?? 0
  const successRate = total > 0 ? Math.round(((s?.success_requests ?? 0) / total) * 100) : 0
  const redirectRate = total > 0 ? Math.round((totalRedirects.value / total) * 100) : 0

  return [
    { label: '总请求', value: formatNumber(total), hint: currentRangeLabel.value, dotClass: 'bg-slate-400' },
    { label: '成功率', value: `${successRate}%`, hint: `${formatNumber(s?.success_requests ?? 0)} 成功`, dotClass: 'bg-emerald-500' },
    { label: '失败', value: formatNumber(s?.failed_requests ?? 0), hint: '上游或调度失败', dotClass: 'bg-rose-500' },
    { label: '改道率', value: `${redirectRate}%`, hint: `${formatNumber(totalRedirects.value)} 次改道`, dotClass: 'bg-amber-500' },
    { label: '总 Token', value: formatNumber(s?.total_tokens ?? 0), hint: tokenDominantLabel.value, dotClass: 'bg-blue-500' },
    { label: '实际费用', value: formatCurrency(s?.total_actual_cost ?? 0), hint: costPerRequestLabel.value, dotClass: 'bg-teal-500' }
  ]
})

const currentRangeLabel = computed(() => {
  if (!filters.start_date && !filters.end_date) return '全部时间'
  return `${filters.start_date || '最早'} 至 ${filters.end_date || '今天'}`
})

const costPerRequestLabel = computed(() => {
  const s = summary.value
  if (!s?.total_requests) return '单次成本 $0.00'
  return `单次成本 ${formatCurrency(s.total_actual_cost / s.total_requests)}`
})

const tokenDominantLabel = computed(() => {
  const s = summary.value
  if (!s) return '暂无 Token'
  const items = [
    { label: '输入', value: s.total_input_tokens },
    { label: '输出', value: s.total_output_tokens },
    { label: '缓存', value: s.total_cache_tokens }
  ].sort((a, b) => b.value - a.value)
  return items[0]?.value > 0 ? `${items[0].label}占比最高` : '暂无 Token'
})

const outcomeChartData = computed<ChartData<'doughnut'> | null>(() => {
  const s = summary.value
  if (!s || s.total_requests <= 0) return null
  return doughnutData(['成功', '失败'], [s.success_requests, s.failed_requests], outcomeColors)
})

const redirectChartData = computed<ChartData<'doughnut'> | null>(() => {
  const s = summary.value
  if (!s || totalRedirects.value <= 0) return null
  return doughnutData(
    ['隐私改道', '额度改道', '错误改道'],
    [s.privacy_redirect_count, s.quota_redirect_count, s.error_redirect_count],
    redirectColors
  )
})

const tokenMixChartData = computed<ChartData<'doughnut'> | null>(() => {
  const s = summary.value
  if (!s || s.total_tokens <= 0) return null
  const knownTokens = s.total_input_tokens + s.total_output_tokens + s.total_cache_tokens
  const otherTokens = Math.max(0, s.total_tokens - knownTokens)
  return doughnutData(
    ['输入', '输出', '缓存', '其他'],
    [s.total_input_tokens, s.total_output_tokens, s.total_cache_tokens, otherTokens],
    tokenColors
  )
})

const poolDistributionRows = computed<DistributionRow[]>(() => makeDistributionRows(summary.value?.by_pool ?? []))
const reasonDistributionRows = computed<DistributionRow[]>(() => makeDistributionRows(summary.value?.by_reason ?? []))

const poolRequestChartData = computed<ChartData<'bar'> | null>(() => {
  return barDataFromRows(poolDistributionRows.value, 'requests', '请求数')
})

const reasonCostChartData = computed<ChartData<'bar'> | null>(() => {
  return barDataFromRows(reasonDistributionRows.value, 'total_actual_cost', '实际费用')
})

const modelTokenChartData = computed<ChartData<'bar'> | null>(() => {
  const rows = aggregateCurrentPageModels()
  if (rows.length === 0) return null
  return {
    labels: rows.map((row) => row.label),
    datasets: [
      {
        label: 'Token',
        data: rows.map((row) => row.value),
        backgroundColor: rows.map((_, index) => chartColors[index % chartColors.length]),
        borderRadius: 6,
        barPercentage: 0.7
      }
    ]
  }
})

const latencyChartData = computed<ChartData<'bar'> | null>(() => {
  if (logs.value.length === 0) return null
  const buckets = [
    { label: '< 3s', min: 0, max: 3000, count: 0 },
    { label: '3-10s', min: 3000, max: 10000, count: 0 },
    { label: '10-30s', min: 10000, max: 30000, count: 0 },
    { label: '30-60s', min: 30000, max: 60000, count: 0 },
    { label: '> 60s', min: 60000, max: Number.POSITIVE_INFINITY, count: 0 }
  ]
  for (const row of logs.value) {
    const match = buckets.find((bucket) => row.duration_ms >= bucket.min && row.duration_ms < bucket.max)
    if (match) match.count += 1
  }
  return {
    labels: buckets.map((bucket) => bucket.label),
    datasets: [
      {
        label: '请求数',
        data: buckets.map((bucket) => bucket.count),
        backgroundColor: '#14b8a6',
        borderRadius: 6,
        barPercentage: 0.62
      }
    ]
  }
})

const doughnutOptions = computed<ChartOptions<'doughnut'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '68%',
  plugins: {
    legend: {
      position: 'bottom',
      labels: {
        color: chartTheme.value.text,
        boxWidth: 10,
        boxHeight: 10,
        usePointStyle: true
      }
    },
    tooltip: tooltipOptions()
  }
}))

const horizontalCountBarOptions = computed<ChartOptions<'bar'>>(() => ({
  indexAxis: 'y',
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: tooltipOptions()
  },
  scales: {
    x: countAxis(),
    y: categoryAxis()
  }
}))

const horizontalCurrencyBarOptions = computed<ChartOptions<'bar'>>(() => ({
  indexAxis: 'y',
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      ...tooltipOptions(),
      callbacks: {
        label: (context) => `${context.dataset.label}: ${formatCurrency(Number(context.raw ?? 0))}`
      }
    }
  },
  scales: {
    x: {
      ...countAxis(),
      ticks: {
        color: chartTheme.value.text,
        callback: (value) => formatCurrency(Number(value))
      }
    },
    y: categoryAxis()
  }
}))

const latencyBarOptions = computed<ChartOptions<'bar'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: tooltipOptions()
  },
  scales: {
    x: categoryAxis(),
    y: countAxis()
  }
}))

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
  return formatNumber(totalTokens(row))
}

function totalTokens(row: RoutingAuditLog): number {
  return row.input_tokens + row.output_tokens + row.cache_creation_tokens + row.cache_read_tokens + row.image_output_tokens
}

function makeDistributionRows(rows: RoutingAuditDimensionStat[]): DistributionRow[] {
  const totalRequests = rows.reduce((sum, row) => sum + row.requests, 0)
  return [...rows]
    .sort((a, b) => b.requests - a.requests)
    .map((row, index) => ({
      ...row,
      color: chartColors[index % chartColors.length],
      requestPercent: totalRequests > 0 ? (row.requests / totalRequests) * 100 : 0,
      successRate: row.requests > 0 ? (row.success_requests / row.requests) * 100 : 0
    }))
}

function barDataFromRows(
  rows: DistributionRow[],
  metric: 'requests' | 'total_actual_cost',
  label: string
): ChartData<'bar'> | null {
  const displayRows = rows.slice(0, 8)
  if (displayRows.length === 0) return null
  return {
    labels: displayRows.map((row) => row.key || 'unknown'),
    datasets: [
      {
        label,
        data: displayRows.map((row) => row[metric]),
        backgroundColor: displayRows.map((row) => row.color),
        borderRadius: 6,
        barPercentage: 0.72
      }
    ]
  }
}

function aggregateCurrentPageModels(): Array<{ label: string; value: number }> {
  const map = new Map<string, number>()
  for (const row of logs.value) {
    const key = row.model || 'unknown'
    map.set(key, (map.get(key) ?? 0) + totalTokens(row))
  }
  return [...map.entries()]
    .map(([label, value]) => ({ label, value }))
    .filter((row) => row.value > 0)
    .sort((a, b) => b.value - a.value)
    .slice(0, 8)
}

function doughnutData(labels: string[], values: number[], colors: string[]): ChartData<'doughnut'> | null {
  const compact = labels
    .map((label, index) => ({ label, value: values[index] ?? 0, color: colors[index % colors.length] }))
    .filter((item) => item.value > 0)
  if (compact.length === 0) return null
  return {
    labels: compact.map((item) => item.label),
    datasets: [
      {
        data: compact.map((item) => item.value),
        backgroundColor: compact.map((item) => item.color),
        borderWidth: 0,
        hoverOffset: 4
      }
    ]
  }
}

function tooltipOptions() {
  return {
    backgroundColor: chartTheme.value.tooltipBg,
    titleColor: chartTheme.value.tooltipTitle,
    bodyColor: chartTheme.value.tooltipBody,
    borderColor: isDarkMode.value ? '#374151' : '#e5e7eb',
    borderWidth: 1
  }
}

function countAxis() {
  return {
    beginAtZero: true,
    grid: { color: chartTheme.value.grid },
    ticks: {
      color: chartTheme.value.text,
      callback: (value: string | number) => formatNumber(Number(value))
    }
  }
}

function categoryAxis() {
  return {
    grid: { display: false },
    ticks: { color: chartTheme.value.text }
  }
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

function formatPercent(value?: number): string {
  if (value === undefined || value === null) return '-'
  return `${value.toFixed(1)}%`
}

function formatMoneyPair(used?: number, limit?: number): string {
  if (!used && !limit) return '-'
  if (!limit) return formatCurrency(used ?? 0)
  return `${formatCurrency(used ?? 0)} / ${formatCurrency(limit)}`
}

function boundedPercent(value: number): number {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

function quotaPercent(used?: number, limit?: number): number {
  if (!limit || limit <= 0) return 0
  return boundedPercent(((used ?? 0) / limit) * 100)
}

const ChartPanel = defineComponent({
  name: 'ChartPanel',
  props: {
    title: { type: String, required: true },
    subtitle: { type: String, default: '' }
  },
  setup(props, { slots, attrs }) {
    return () =>
      h('div', { ...attrs, class: ['card p-4', attrs.class] }, [
        h('div', { class: 'mb-3 flex items-start justify-between gap-3' }, [
          h('div', [
            h('h2', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, props.title),
            props.subtitle ? h('p', { class: 'mt-0.5 text-xs text-gray-500 dark:text-gray-400' }, props.subtitle) : null
          ])
        ]),
        h('div', { class: 'h-64 min-h-0' }, slots.default?.())
      ])
  }
})

const EmptyChart = defineComponent({
  name: 'EmptyChart',
  setup() {
    return () =>
      h('div', { class: 'flex h-full items-center justify-center rounded-lg border border-dashed border-gray-200 text-sm text-gray-400 dark:border-dark-700 dark:text-gray-500' }, '暂无可视化数据')
  }
})

const DistributionPanel = defineComponent({
  name: 'DistributionPanel',
  props: {
    title: { type: String, required: true },
    subtitle: { type: String, default: '' },
    rows: { type: Array as () => DistributionRow[], required: true },
    valueLabel: { type: String, required: true }
  },
  setup(props) {
    return () =>
      h('div', { class: 'card overflow-hidden' }, [
        h('div', { class: 'border-b border-gray-100 px-4 py-3 dark:border-dark-700' }, [
          h('h2', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, props.title),
          props.subtitle ? h('p', { class: 'mt-0.5 text-xs text-gray-500 dark:text-gray-400' }, props.subtitle) : null
        ]),
        props.rows.length === 0
          ? h('div', { class: 'px-4 py-8 text-center text-sm text-gray-500' }, '暂无数据')
          : h('div', { class: 'divide-y divide-gray-100 dark:divide-dark-700' },
              props.rows.map((row) =>
                h('div', { key: row.key, class: 'px-4 py-3' }, [
                  h('div', { class: 'flex items-center justify-between gap-3' }, [
                    h('div', { class: 'min-w-0' }, [
                      h('div', { class: 'flex items-center gap-2' }, [
                        h('span', { class: 'h-2.5 w-2.5 shrink-0 rounded-full', style: { backgroundColor: row.color } }),
                        h('span', { class: 'truncate text-sm font-semibold text-gray-900 dark:text-white' }, row.key || 'unknown')
                      ]),
                      h('p', { class: 'mt-1 text-xs text-gray-500 dark:text-gray-400' }, `成功率 ${Math.round(row.successRate)}% · Token ${formatNumber(row.total_tokens)} · ${formatCurrency(row.total_actual_cost)}`)
                    ]),
                    h('div', { class: 'text-right' }, [
                      h('p', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, formatNumber(row.requests)),
                      h('p', { class: 'text-[11px] text-gray-400 dark:text-gray-500' }, props.valueLabel)
                    ])
                  ]),
                  h('div', { class: 'mt-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700' }, [
                    h('div', {
                      class: 'h-full rounded-full',
                      style: { width: `${boundedPercent(row.requestPercent)}%`, backgroundColor: row.color }
                    })
                  ])
                ])
              )
            )
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
          h('h2', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, props.title),
          h('p', { class: 'mt-0.5 text-xs text-gray-500 dark:text-gray-400' }, '可调度账号、Codex 窗口与额度使用')
        ]),
        h('div', { class: 'divide-y divide-gray-100 dark:divide-dark-700' },
          props.items.length === 0
            ? [h('div', { class: 'px-4 py-8 text-center text-sm text-gray-500' }, '暂无快照')]
            : props.items.map((item) =>
                h('div', { key: item.pool, class: 'space-y-3 px-4 py-3' }, [
                  h('div', { class: 'flex items-center justify-between gap-3' }, [
                    h('span', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, item.pool || 'unknown'),
                    h('span', { class: 'text-xs text-gray-500 dark:text-gray-400' }, `${item.schedulable_accounts}/${item.total_accounts} 可调度`)
                  ]),
                  snapshotMeter('可调度', item.total_accounts > 0 ? (item.schedulable_accounts / item.total_accounts) * 100 : 0, `${item.schedulable_accounts}/${item.total_accounts}`),
                  snapshotMeter('5h max', item.codex_5h_max_used_percent ?? 0, formatPercent(item.codex_5h_max_used_percent)),
                  snapshotMeter('7d max', item.codex_7d_max_used_percent ?? 0, formatPercent(item.codex_7d_max_used_percent)),
                  snapshotMeter('日额度', quotaPercent(item.quota_daily_used_usd, item.quota_daily_limit_usd), formatMoneyPair(item.quota_daily_used_usd, item.quota_daily_limit_usd)),
                  snapshotMeter('周额度', quotaPercent(item.quota_weekly_used_usd, item.quota_weekly_limit_usd), formatMoneyPair(item.quota_weekly_used_usd, item.quota_weekly_limit_usd))
                ])
              )
        )
      ])
  }
})

function snapshotMeter(label: string, percent: number, value: string) {
  const bounded = boundedPercent(percent)
  const color = bounded >= 85 ? '#ef4444' : bounded >= 65 ? '#f59e0b' : '#14b8a6'
  return h('div', [
    h('div', { class: 'mb-1 flex items-center justify-between gap-3 text-xs' }, [
      h('span', { class: 'font-medium text-gray-600 dark:text-gray-300' }, label),
      h('span', { class: 'text-gray-500 dark:text-gray-400' }, value)
    ]),
    h('div', { class: 'h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700' }, [
      h('div', { class: 'h-full rounded-full', style: { width: `${bounded}%`, backgroundColor: color } })
    ])
  ])
}

onMounted(loadData)
</script>
