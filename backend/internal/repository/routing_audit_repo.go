package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type routingAuditRepository struct {
	db *sql.DB
}

func NewRoutingAuditRepository(db *sql.DB) service.RoutingAuditRepository {
	return &routingAuditRepository{db: db}
}

func (r *routingAuditRepository) Create(ctx context.Context, log *service.RoutingAuditLog) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil routing audit repository")
	}
	if log == nil {
		return fmt.Errorf("nil routing audit log")
	}
	if len(log.CandidatePools) == 0 {
		log.CandidatePools = json.RawMessage("[]")
	}
	if len(log.PlusPoolSnapshot) == 0 {
		log.PlusPoolSnapshot = json.RawMessage("{}")
	}
	if len(log.RelayQuotaSnapshot) == 0 {
		log.RelayQuotaSnapshot = json.RawMessage("{}")
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO routing_audit_logs (
    request_id, user_id, api_key_id, group_id, account_id, model,
    inbound_endpoint, upstream_endpoint, request_type, stream,
    routing_policy_version, precheck_rules_version, candidate_pools,
    selected_pool, selected_account_id, decision_reason, fallback_reason,
    privacy_decision, privacy_redirect, quota_redirect, error_redirect,
    schedule_layer, sticky_previous_hit, sticky_session_hit, candidate_count,
    top_k, scheduler_latency_ms, load_skew, plus_pool_snapshot,
    relay_quota_snapshot, final_route, success, error_type, upstream_status,
    input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
    image_output_tokens, total_cost, actual_cost, duration_ms, created_at
) VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,
    $18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,
    $33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43
)`,
		nullableString(log.RequestID),
		nullableInt64(log.UserID),
		nullableInt64(log.APIKeyID),
		log.GroupID,
		log.AccountID,
		nullableString(log.Model),
		nullableString(log.InboundEndpoint),
		nullableString(log.UpstreamEndpoint),
		nullableString(log.RequestType),
		log.Stream,
		defaultString(log.RoutingPolicyVersion, service.RoutingAuditPolicyVersion),
		defaultString(log.PrecheckRulesVersion, service.RoutingAuditPrecheckRulesVersion),
		[]byte(log.CandidatePools),
		nullableString(log.SelectedPool),
		log.SelectedAccountID,
		nullableString(log.DecisionReason),
		nullableString(log.FallbackReason),
		nullableString(log.PrivacyDecision),
		log.PrivacyRedirect,
		log.QuotaRedirect,
		log.ErrorRedirect,
		nullableString(log.ScheduleLayer),
		log.StickyPreviousHit,
		log.StickySessionHit,
		log.CandidateCount,
		log.TopK,
		log.SchedulerLatencyMs,
		log.LoadSkew,
		[]byte(log.PlusPoolSnapshot),
		[]byte(log.RelayQuotaSnapshot),
		nullableString(log.FinalRoute),
		log.Success,
		nullableString(log.ErrorType),
		log.UpstreamStatus,
		log.InputTokens,
		log.OutputTokens,
		log.CacheCreationTokens,
		log.CacheReadTokens,
		log.ImageOutputTokens,
		log.TotalCost,
		log.ActualCost,
		log.DurationMs,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert routing audit log: %w", err)
	}
	return nil
}

func (r *routingAuditRepository) List(ctx context.Context, params pagination.PaginationParams, filter service.RoutingAuditFilter) ([]service.RoutingAuditLog, *pagination.PaginationResult, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("nil routing audit repository")
	}
	where, args := buildRoutingAuditWhere(filter)
	sortBy := normalizeRoutingAuditSortBy(params.SortBy)
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	limit := params.Limit()
	page := params.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int64
	countQuery := "SELECT COUNT(*) FROM routing_audit_logs " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count routing audit logs: %w", err)
	}
	query := routingAuditSelectSQL + " " + where + fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortBy, strings.ToUpper(sortOrder), len(args)+1, len(args)+2)
	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("list routing audit logs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	logs := make([]service.RoutingAuditLog, 0, limit)
	for rows.Next() {
		item, err := scanRoutingAuditLog(rows)
		if err != nil {
			return nil, nil, err
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	pages := int((total + int64(limit) - 1) / int64(limit))
	if pages < 1 {
		pages = 1
	}
	return logs, &pagination.PaginationResult{Total: total, Page: page, PageSize: limit, Pages: pages}, nil
}

func (r *routingAuditRepository) Summary(ctx context.Context, filter service.RoutingAuditFilter) (*service.RoutingAuditSummary, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil routing audit repository")
	}
	where, args := buildRoutingAuditWhere(filter)
	summary := &service.RoutingAuditSummary{}
	err := r.db.QueryRowContext(ctx, `
SELECT
    COUNT(*),
    COUNT(*) FILTER (WHERE success),
    COUNT(*) FILTER (WHERE NOT success),
    COUNT(*) FILTER (WHERE privacy_redirect),
    COUNT(*) FILTER (WHERE quota_redirect),
    COUNT(*) FILTER (WHERE error_redirect),
    COALESCE(AVG(scheduler_latency_ms), 0),
    COALESCE(SUM(input_tokens), 0),
    COALESCE(SUM(output_tokens), 0),
    COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0),
    COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens + image_output_tokens), 0),
    COALESCE(SUM(total_cost), 0),
    COALESCE(SUM(actual_cost), 0)
FROM routing_audit_logs `+where, args...).Scan(
		&summary.TotalRequests,
		&summary.SuccessRequests,
		&summary.FailedRequests,
		&summary.PrivacyRedirectCount,
		&summary.QuotaRedirectCount,
		&summary.ErrorRedirectCount,
		&summary.AvgSchedulerLatencyMs,
		&summary.TotalInputTokens,
		&summary.TotalOutputTokens,
		&summary.TotalCacheTokens,
		&summary.TotalTokens,
		&summary.TotalCost,
		&summary.TotalActualCost,
	)
	if err != nil {
		return nil, fmt.Errorf("routing audit summary: %w", err)
	}
	byPool, err := r.dimensionStats(ctx, where, args, "COALESCE(NULLIF(selected_pool, ''), 'unknown')")
	if err != nil {
		return nil, err
	}
	byReason, err := r.dimensionStats(ctx, where, args, "COALESCE(NULLIF(decision_reason, ''), 'unknown')")
	if err != nil {
		return nil, err
	}
	summary.ByPool = byPool
	summary.ByReason = byReason
	_ = r.db.QueryRowContext(ctx, `
SELECT plus_pool_snapshot, relay_quota_snapshot
FROM routing_audit_logs `+where+`
ORDER BY created_at DESC
LIMIT 1`, args...).Scan(&summary.LatestPlusPoolSnapshot, &summary.LatestRelaySnapshot)
	return summary, nil
}

func (r *routingAuditRepository) dimensionStats(ctx context.Context, where string, args []any, expr string) ([]service.RoutingAuditDimensionStat, error) {
	query := fmt.Sprintf(`
SELECT
    %s AS key,
    COUNT(*) AS requests,
    COUNT(*) FILTER (WHERE success) AS success_requests,
    COUNT(*) FILTER (WHERE NOT success) AS failed_requests,
    COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens + image_output_tokens), 0) AS total_tokens,
    COALESCE(SUM(actual_cost), 0) AS total_actual_cost,
    COALESCE(AVG(duration_ms), 0) AS avg_duration_ms,
    COALESCE(AVG(scheduler_latency_ms), 0) AS avg_scheduler_latency_ms,
    COUNT(*) FILTER (WHERE privacy_redirect) AS privacy_redirects,
    COUNT(*) FILTER (WHERE quota_redirect) AS quota_redirects,
    COUNT(*) FILTER (WHERE error_redirect) AS error_redirects
FROM routing_audit_logs %s
GROUP BY key
ORDER BY requests DESC, key ASC
LIMIT 20`, expr, where)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("routing audit dimension stats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.RoutingAuditDimensionStat, 0, 16)
	for rows.Next() {
		var item service.RoutingAuditDimensionStat
		if err := rows.Scan(
			&item.Key,
			&item.Requests,
			&item.SuccessRequests,
			&item.FailedRequests,
			&item.TotalTokens,
			&item.TotalActualCost,
			&item.AvgDurationMs,
			&item.AvgSchedulerMs,
			&item.PrivacyRedirects,
			&item.QuotaRedirects,
			&item.ErrorRedirects,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

const routingAuditSelectSQL = `
SELECT
    id, request_id, user_id, api_key_id, group_id, account_id, model,
    inbound_endpoint, upstream_endpoint, request_type, stream,
    routing_policy_version, precheck_rules_version, candidate_pools,
    selected_pool, selected_account_id, decision_reason, fallback_reason,
    privacy_decision, privacy_redirect, quota_redirect, error_redirect,
    schedule_layer, sticky_previous_hit, sticky_session_hit, candidate_count,
    top_k, scheduler_latency_ms, load_skew, plus_pool_snapshot,
    relay_quota_snapshot, final_route, success, error_type, upstream_status,
    input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
    image_output_tokens, total_cost, actual_cost, duration_ms, created_at
FROM routing_audit_logs`

type routingAuditScanner interface {
	Scan(dest ...any) error
}

func scanRoutingAuditLog(row routingAuditScanner) (service.RoutingAuditLog, error) {
	var item service.RoutingAuditLog
	var requestID, model, inboundEndpoint, upstreamEndpoint, requestType sql.NullString
	var selectedPool, decisionReason, fallbackReason, privacyDecision, scheduleLayer sql.NullString
	var finalRoute, errorType sql.NullString
	var userID, apiKeyID sql.NullInt64
	var groupID, accountID, selectedAccountID sql.NullInt64
	var upstreamStatus sql.NullInt64
	var candidatePools, plusSnapshot, relaySnapshot []byte
	err := row.Scan(
		&item.ID,
		&requestID,
		&userID,
		&apiKeyID,
		&groupID,
		&accountID,
		&model,
		&inboundEndpoint,
		&upstreamEndpoint,
		&requestType,
		&item.Stream,
		&item.RoutingPolicyVersion,
		&item.PrecheckRulesVersion,
		&candidatePools,
		&selectedPool,
		&selectedAccountID,
		&decisionReason,
		&fallbackReason,
		&privacyDecision,
		&item.PrivacyRedirect,
		&item.QuotaRedirect,
		&item.ErrorRedirect,
		&scheduleLayer,
		&item.StickyPreviousHit,
		&item.StickySessionHit,
		&item.CandidateCount,
		&item.TopK,
		&item.SchedulerLatencyMs,
		&item.LoadSkew,
		&plusSnapshot,
		&relaySnapshot,
		&finalRoute,
		&item.Success,
		&errorType,
		&upstreamStatus,
		&item.InputTokens,
		&item.OutputTokens,
		&item.CacheCreationTokens,
		&item.CacheReadTokens,
		&item.ImageOutputTokens,
		&item.TotalCost,
		&item.ActualCost,
		&item.DurationMs,
		&item.CreatedAt,
	)
	if err != nil {
		return item, err
	}
	item.RequestID = requestID.String
	if userID.Valid {
		item.UserID = userID.Int64
	}
	if apiKeyID.Valid {
		item.APIKeyID = apiKeyID.Int64
	}
	item.GroupID = nullableInt64Ptr(groupID)
	item.AccountID = nullableInt64Ptr(accountID)
	item.Model = model.String
	item.InboundEndpoint = inboundEndpoint.String
	item.UpstreamEndpoint = upstreamEndpoint.String
	item.RequestType = requestType.String
	item.CandidatePools = json.RawMessage(candidatePools)
	item.SelectedPool = selectedPool.String
	item.SelectedAccountID = nullableInt64Ptr(selectedAccountID)
	item.DecisionReason = decisionReason.String
	item.FallbackReason = fallbackReason.String
	item.PrivacyDecision = privacyDecision.String
	item.ScheduleLayer = scheduleLayer.String
	item.PlusPoolSnapshot = json.RawMessage(plusSnapshot)
	item.RelayQuotaSnapshot = json.RawMessage(relaySnapshot)
	item.FinalRoute = finalRoute.String
	item.ErrorType = errorType.String
	if upstreamStatus.Valid {
		status := int(upstreamStatus.Int64)
		item.UpstreamStatus = &status
	}
	return item, nil
}

func buildRoutingAuditWhere(filter service.RoutingAuditFilter) (string, []any) {
	conditions := make([]string, 0, 12)
	args := make([]any, 0, 12)
	addInt := func(column string, value int64) {
		if value <= 0 {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	addText := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	addInt("user_id", filter.UserID)
	addInt("api_key_id", filter.APIKeyID)
	addInt("account_id", filter.AccountID)
	addInt("group_id", filter.GroupID)
	addText("model", filter.Model)
	addText("selected_pool", filter.SelectedPool)
	addText("final_route", filter.FinalRoute)
	addText("decision_reason", filter.DecisionReason)
	addText("routing_policy_version", filter.RoutingPolicyVersion)
	if filter.StartTime != nil {
		args = append(args, *filter.StartTime)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.EndTime != nil {
		args = append(args, *filter.EndTime)
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func normalizeRoutingAuditSortBy(sortBy string) string {
	switch strings.TrimSpace(sortBy) {
	case "id":
		return "id"
	case "user_id":
		return "user_id"
	case "api_key_id":
		return "api_key_id"
	case "account_id":
		return "account_id"
	case "selected_pool":
		return "selected_pool"
	case "duration_ms":
		return "duration_ms"
	case "actual_cost":
		return "actual_cost"
	default:
		return "created_at"
	}
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}

func nullableInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func nullableInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
