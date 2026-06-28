import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface RoutingAuditQueryParams {
  page?: number
  page_size?: number
  start_date?: string
  end_date?: string
  timezone?: string
  user_id?: number
  api_key_id?: number
  account_id?: number
  group_id?: number
  model?: string
  selected_pool?: string
  final_route?: string
  decision_reason?: string
  routing_policy_version?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface PoolSnapshot {
  pool: string
  account_type?: string
  total_accounts: number
  schedulable_accounts: number
  rate_limited_accounts: number
  overloaded_accounts: number
  temp_unschedulable: number
  quota_limited_accounts: number
  quota_limit_usd?: number
  quota_used_usd?: number
  quota_daily_limit_usd?: number
  quota_daily_used_usd?: number
  quota_weekly_limit_usd?: number
  quota_weekly_used_usd?: number
  codex_5h_max_used_percent?: number
  codex_5h_avg_used_percent?: number
  codex_7d_max_used_percent?: number
  codex_7d_avg_used_percent?: number
  oldest_snapshot_updated_at?: string
}

export interface RoutingAuditLog {
  id: number
  request_id?: string
  user_id?: number
  api_key_id?: number
  group_id?: number
  account_id?: number
  model?: string
  inbound_endpoint?: string
  upstream_endpoint?: string
  request_type?: string
  stream: boolean
  routing_policy_version: string
  precheck_rules_version: string
  candidate_pools?: PoolSnapshot[]
  selected_pool?: string
  selected_account_id?: number
  decision_reason?: string
  fallback_reason?: string
  privacy_decision?: string
  privacy_redirect: boolean
  quota_redirect: boolean
  error_redirect: boolean
  schedule_layer?: string
  sticky_previous_hit: boolean
  sticky_session_hit: boolean
  candidate_count: number
  top_k: number
  scheduler_latency_ms: number
  load_skew: number
  plus_pool_snapshot?: PoolSnapshot[]
  relay_quota_snapshot?: PoolSnapshot[]
  final_route?: string
  success: boolean
  error_type?: string
  upstream_status?: number
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  image_output_tokens: number
  total_cost: number
  actual_cost: number
  duration_ms: number
  created_at: string
}

export interface RoutingAuditDimensionStat {
  key: string
  requests: number
  success_requests: number
  failed_requests: number
  total_tokens: number
  total_actual_cost: number
  avg_duration_ms: number
  avg_scheduler_latency_ms: number
  privacy_redirects: number
  quota_redirects: number
  error_redirects: number
}

export interface RoutingAuditSummary {
  total_requests: number
  success_requests: number
  failed_requests: number
  privacy_redirect_count: number
  quota_redirect_count: number
  error_redirect_count: number
  avg_scheduler_latency_ms: number
  total_input_tokens: number
  total_output_tokens: number
  total_cache_tokens: number
  total_tokens: number
  total_cost: number
  total_actual_cost: number
  by_pool: RoutingAuditDimensionStat[]
  by_reason: RoutingAuditDimensionStat[]
  latest_plus_pool_snapshot?: PoolSnapshot[]
  latest_relay_quota_snapshot?: PoolSnapshot[]
}

export async function list(
  params: RoutingAuditQueryParams,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<RoutingAuditLog>> {
  const { data } = await apiClient.get<PaginatedResponse<RoutingAuditLog>>('/admin/routing-audit/logs', {
    params,
    signal: options?.signal
  })
  return data
}

export async function summary(params: RoutingAuditQueryParams): Promise<RoutingAuditSummary> {
  const { data } = await apiClient.get<RoutingAuditSummary>('/admin/routing-audit/summary', {
    params
  })
  return data
}

export default {
  list,
  summary
}
