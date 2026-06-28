package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	RoutingAuditPolicyVersion        = "routing-audit-v1"
	RoutingAuditPrecheckRulesVersion = "precheck-v1"

	RoutingAuditPoolTrustedPlus = "trusted-plus"
	RoutingAuditPoolRelay       = "relay"
	RoutingAuditPoolUnknown     = "unknown"

	RoutingAuditPrivacyDecisionRulePass = "rule_pass"
	RoutingAuditFinalRouteOpenAI        = "openai-responses"
)

type RoutingAuditRepository interface {
	Create(ctx context.Context, log *RoutingAuditLog) error
	List(ctx context.Context, params pagination.PaginationParams, filter RoutingAuditFilter) ([]RoutingAuditLog, *pagination.PaginationResult, error)
	Summary(ctx context.Context, filter RoutingAuditFilter) (*RoutingAuditSummary, error)
}

type RoutingAuditLog struct {
	ID                   int64           `json:"id"`
	RequestID            string          `json:"request_id,omitempty"`
	UserID               int64           `json:"user_id,omitempty"`
	APIKeyID             int64           `json:"api_key_id,omitempty"`
	GroupID              *int64          `json:"group_id,omitempty"`
	AccountID            *int64          `json:"account_id,omitempty"`
	Model                string          `json:"model,omitempty"`
	InboundEndpoint      string          `json:"inbound_endpoint,omitempty"`
	UpstreamEndpoint     string          `json:"upstream_endpoint,omitempty"`
	RequestType          string          `json:"request_type,omitempty"`
	Stream               bool            `json:"stream"`
	RoutingPolicyVersion string          `json:"routing_policy_version"`
	PrecheckRulesVersion string          `json:"precheck_rules_version"`
	CandidatePools       json.RawMessage `json:"candidate_pools,omitempty"`
	SelectedPool         string          `json:"selected_pool,omitempty"`
	SelectedAccountID    *int64          `json:"selected_account_id,omitempty"`
	DecisionReason       string          `json:"decision_reason,omitempty"`
	FallbackReason       string          `json:"fallback_reason,omitempty"`
	PrivacyDecision      string          `json:"privacy_decision,omitempty"`
	PrivacyRedirect      bool            `json:"privacy_redirect"`
	QuotaRedirect        bool            `json:"quota_redirect"`
	ErrorRedirect        bool            `json:"error_redirect"`
	ScheduleLayer        string          `json:"schedule_layer,omitempty"`
	StickyPreviousHit    bool            `json:"sticky_previous_hit"`
	StickySessionHit     bool            `json:"sticky_session_hit"`
	CandidateCount       int             `json:"candidate_count"`
	TopK                 int             `json:"top_k"`
	SchedulerLatencyMs   int64           `json:"scheduler_latency_ms"`
	LoadSkew             float64         `json:"load_skew"`
	PlusPoolSnapshot     json.RawMessage `json:"plus_pool_snapshot,omitempty"`
	RelayQuotaSnapshot   json.RawMessage `json:"relay_quota_snapshot,omitempty"`
	FinalRoute           string          `json:"final_route,omitempty"`
	Success              bool            `json:"success"`
	ErrorType            string          `json:"error_type,omitempty"`
	UpstreamStatus       *int            `json:"upstream_status,omitempty"`
	InputTokens          int             `json:"input_tokens"`
	OutputTokens         int             `json:"output_tokens"`
	CacheCreationTokens  int             `json:"cache_creation_tokens"`
	CacheReadTokens      int             `json:"cache_read_tokens"`
	ImageOutputTokens    int             `json:"image_output_tokens"`
	TotalCost            float64         `json:"total_cost"`
	ActualCost           float64         `json:"actual_cost"`
	DurationMs           int             `json:"duration_ms"`
	CreatedAt            time.Time       `json:"created_at"`
}

type RoutingAuditFilter struct {
	UserID               int64
	APIKeyID             int64
	AccountID            int64
	GroupID              int64
	Model                string
	SelectedPool         string
	FinalRoute           string
	DecisionReason       string
	RoutingPolicyVersion string
	StartTime            *time.Time
	EndTime              *time.Time
}

type RoutingAuditSummary struct {
	TotalRequests          int64                       `json:"total_requests"`
	SuccessRequests        int64                       `json:"success_requests"`
	FailedRequests         int64                       `json:"failed_requests"`
	PrivacyRedirectCount   int64                       `json:"privacy_redirect_count"`
	QuotaRedirectCount     int64                       `json:"quota_redirect_count"`
	ErrorRedirectCount     int64                       `json:"error_redirect_count"`
	AvgSchedulerLatencyMs  float64                     `json:"avg_scheduler_latency_ms"`
	TotalInputTokens       int64                       `json:"total_input_tokens"`
	TotalOutputTokens      int64                       `json:"total_output_tokens"`
	TotalCacheTokens       int64                       `json:"total_cache_tokens"`
	TotalTokens            int64                       `json:"total_tokens"`
	TotalCost              float64                     `json:"total_cost"`
	TotalActualCost        float64                     `json:"total_actual_cost"`
	ByPool                 []RoutingAuditDimensionStat `json:"by_pool"`
	ByReason               []RoutingAuditDimensionStat `json:"by_reason"`
	LatestPlusPoolSnapshot json.RawMessage             `json:"latest_plus_pool_snapshot,omitempty"`
	LatestRelaySnapshot    json.RawMessage             `json:"latest_relay_quota_snapshot,omitempty"`
}

type RoutingAuditDimensionStat struct {
	Key              string  `json:"key"`
	Requests         int64   `json:"requests"`
	SuccessRequests  int64   `json:"success_requests"`
	FailedRequests   int64   `json:"failed_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalActualCost  float64 `json:"total_actual_cost"`
	AvgDurationMs    float64 `json:"avg_duration_ms"`
	AvgSchedulerMs   float64 `json:"avg_scheduler_latency_ms"`
	PrivacyRedirects int64   `json:"privacy_redirects"`
	QuotaRedirects   int64   `json:"quota_redirects"`
	ErrorRedirects   int64   `json:"error_redirects"`
}

type RoutingAuditRecordInput struct {
	RequestID            string
	User                 *User
	APIKey               *APIKey
	Account              *Account
	Model                string
	InboundEndpoint      string
	UpstreamEndpoint     string
	RequestType          string
	Stream               bool
	ScheduleDecision     OpenAIAccountScheduleDecision
	Result               *OpenAIForwardResult
	Err                  error
	UpstreamStatus       *int
	FallbackReason       string
	PrivacyDecision      string
	PrivacyRedirect      bool
	QuotaRedirect        bool
	ErrorRedirect        bool
	FinalRoute           string
	DecisionReason       string
	DurationMs           int
	ChannelUsageFields   ChannelUsageFields
	RoutingPolicyVersion string
	PrecheckRulesVersion string
}

type RoutingAuditService struct {
	repo        RoutingAuditRepository
	accountRepo AccountRepository
}

func NewRoutingAuditService(repo RoutingAuditRepository, accountRepo AccountRepository) *RoutingAuditService {
	return &RoutingAuditService{repo: repo, accountRepo: accountRepo}
}

func (s *RoutingAuditService) RecordOpenAIResponses(ctx context.Context, input RoutingAuditRecordInput) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if input.APIKey == nil || input.User == nil {
		return fmt.Errorf("routing audit input missing api key or user")
	}
	now := time.Now()
	accountID := optionalAccountID(input.Account)
	selectedAccountID := optionalAccountID(input.Account)
	selectedPool := s.ClassifyAccountPool(input.Account)
	if selectedPool == "" {
		selectedPool = RoutingAuditPoolUnknown
	}
	decisionReason := strings.TrimSpace(input.DecisionReason)
	if decisionReason == "" {
		decisionReason = resolveRoutingAuditDecisionReason(input)
	}
	privacyDecision := strings.TrimSpace(input.PrivacyDecision)
	if privacyDecision == "" {
		privacyDecision = RoutingAuditPrivacyDecisionRulePass
	}
	finalRoute := strings.TrimSpace(input.FinalRoute)
	if finalRoute == "" {
		finalRoute = RoutingAuditFinalRouteOpenAI
	}
	policyVersion := strings.TrimSpace(input.RoutingPolicyVersion)
	if policyVersion == "" {
		policyVersion = RoutingAuditPolicyVersion
	}
	precheckVersion := strings.TrimSpace(input.PrecheckRulesVersion)
	if precheckVersion == "" {
		precheckVersion = RoutingAuditPrecheckRulesVersion
	}

	candidatePools, plusSnapshot, relaySnapshot := s.buildResourceSnapshots(ctx, input.APIKey.GroupID)
	log := &RoutingAuditLog{
		RequestID:            strings.TrimSpace(input.RequestID),
		UserID:               input.User.ID,
		APIKeyID:             input.APIKey.ID,
		GroupID:              input.APIKey.GroupID,
		AccountID:            accountID,
		Model:                strings.TrimSpace(input.Model),
		InboundEndpoint:      strings.TrimSpace(input.InboundEndpoint),
		UpstreamEndpoint:     strings.TrimSpace(input.UpstreamEndpoint),
		RequestType:          strings.TrimSpace(input.RequestType),
		Stream:               input.Stream,
		RoutingPolicyVersion: policyVersion,
		PrecheckRulesVersion: precheckVersion,
		CandidatePools:       candidatePools,
		SelectedPool:         selectedPool,
		SelectedAccountID:    selectedAccountID,
		DecisionReason:       decisionReason,
		FallbackReason:       strings.TrimSpace(input.FallbackReason),
		PrivacyDecision:      privacyDecision,
		PrivacyRedirect:      input.PrivacyRedirect,
		QuotaRedirect:        input.QuotaRedirect,
		ErrorRedirect:        input.ErrorRedirect,
		ScheduleLayer:        strings.TrimSpace(input.ScheduleDecision.Layer),
		StickyPreviousHit:    input.ScheduleDecision.StickyPreviousHit,
		StickySessionHit:     input.ScheduleDecision.StickySessionHit,
		CandidateCount:       input.ScheduleDecision.CandidateCount,
		TopK:                 input.ScheduleDecision.TopK,
		SchedulerLatencyMs:   input.ScheduleDecision.LatencyMs,
		LoadSkew:             input.ScheduleDecision.LoadSkew,
		PlusPoolSnapshot:     plusSnapshot,
		RelayQuotaSnapshot:   relaySnapshot,
		FinalRoute:           finalRoute,
		Success:              input.Err == nil,
		UpstreamStatus:       input.UpstreamStatus,
		CreatedAt:            now,
	}
	if input.Err != nil {
		log.ErrorType = classifyRoutingAuditError(input.Err)
	}
	if input.Result != nil {
		log.RequestID = routingAuditFirstNonEmpty(log.RequestID, strings.TrimSpace(input.Result.RequestID))
		log.InputTokens = routingAuditMaxInt(0, input.Result.Usage.InputTokens-input.Result.Usage.CacheReadInputTokens)
		log.OutputTokens = input.Result.Usage.OutputTokens
		log.CacheCreationTokens = input.Result.Usage.CacheCreationInputTokens
		log.CacheReadTokens = input.Result.Usage.CacheReadInputTokens
		log.ImageOutputTokens = input.Result.Usage.ImageOutputTokens
		if input.DurationMs <= 0 {
			input.DurationMs = int(input.Result.Duration.Milliseconds())
		}
	}
	log.DurationMs = routingAuditMaxInt(0, input.DurationMs)
	if input.Result != nil && input.APIKey != nil && input.Account != nil {
		cost := s.estimateOpenAIAuditCost(ctx, input)
		if cost != nil {
			log.TotalCost = cost.TotalCost
			log.ActualCost = cost.ActualCost
		}
	}
	return s.repo.Create(ctx, log)
}

func (s *RoutingAuditService) List(ctx context.Context, params pagination.PaginationParams, filter RoutingAuditFilter) ([]RoutingAuditLog, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("routing audit service is not configured")
	}
	return s.repo.List(ctx, params, filter)
}

func (s *RoutingAuditService) Summary(ctx context.Context, filter RoutingAuditFilter) (*RoutingAuditSummary, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("routing audit service is not configured")
	}
	return s.repo.Summary(ctx, filter)
}

func (s *RoutingAuditService) ClassifyAccountPool(account *Account) string {
	return classifyRoutingAuditAccountPool(account)
}

func (s *RoutingAuditService) buildResourceSnapshots(ctx context.Context, groupID *int64) (json.RawMessage, json.RawMessage, json.RawMessage) {
	accounts := s.loadOpenAIAccountsForSnapshot(ctx, groupID)
	pools := buildRoutingAuditPoolSnapshots(accounts)
	candidatePools := mustJSONRaw(pools)
	plusSnapshot := mustJSONRaw(filterRoutingAuditPoolSnapshots(pools, true))
	relaySnapshot := mustJSONRaw(filterRoutingAuditPoolSnapshots(pools, false))
	return candidatePools, plusSnapshot, relaySnapshot
}

func (s *RoutingAuditService) loadOpenAIAccountsForSnapshot(ctx context.Context, groupID *int64) []Account {
	if s == nil || s.accountRepo == nil {
		return nil
	}
	if groupID != nil && *groupID > 0 {
		accounts, err := s.accountRepo.ListByGroup(ctx, *groupID)
		if err == nil {
			return filterRoutingAuditOpenAIAccounts(accounts)
		}
	}
	accounts, err := s.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil
	}
	return accounts
}

func filterRoutingAuditOpenAIAccounts(accounts []Account) []Account {
	out := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == PlatformOpenAI {
			out = append(out, account)
		}
	}
	return out
}

func (s *RoutingAuditService) estimateOpenAIAuditCost(ctx context.Context, input RoutingAuditRecordInput) *CostBreakdown {
	if input.Result == nil || input.APIKey == nil || input.Account == nil {
		return nil
	}
	actualInputTokens := input.Result.Usage.InputTokens - input.Result.Usage.CacheReadInputTokens
	if actualInputTokens < 0 {
		actualInputTokens = 0
	}
	tokens := UsageTokens{
		InputTokens:         actualInputTokens,
		ImageInputTokens:    input.Result.Usage.ImageInputTokens,
		OutputTokens:        input.Result.Usage.OutputTokens,
		CacheCreationTokens: input.Result.Usage.CacheCreationInputTokens,
		CacheReadTokens:     input.Result.Usage.CacheReadInputTokens,
		ImageOutputTokens:   input.Result.Usage.ImageOutputTokens,
	}
	multiplier := 1.0
	if input.APIKey.GroupID != nil && input.APIKey.Group != nil {
		multiplier = input.APIKey.Group.RateMultiplier
	}
	if multiplier <= 0 {
		multiplier = 1.0
	}
	billingModel := strings.TrimSpace(input.Result.BillingModel)
	if billingModel == "" {
		billingModel = forwardResultBillingModel(input.Result.Model, input.Result.UpstreamModel)
	}
	if input.ChannelUsageFields.BillingModelSource == BillingModelSourceChannelMapped &&
		input.ChannelUsageFields.ChannelMappedModel != "" &&
		input.ChannelUsageFields.ChannelMappedModel != input.ChannelUsageFields.OriginalModel {
		billingModel = input.ChannelUsageFields.ChannelMappedModel
	}
	if input.ChannelUsageFields.BillingModelSource == BillingModelSourceRequested && input.ChannelUsageFields.OriginalModel != "" {
		billingModel = input.ChannelUsageFields.OriginalModel
	}
	if billingModel == "" {
		billingModel = input.Model
	}
	serviceTier := ""
	if input.Result.ServiceTier != nil {
		serviceTier = strings.TrimSpace(*input.Result.ServiceTier)
	}
	if input.Result.ImageCount > 0 {
		return nil
	}
	bs := NewBillingService(nil, nil)
	cost, err := bs.CalculateCostWithServiceTier(billingModel, tokens, multiplier, serviceTier)
	if err != nil {
		return nil
	}
	return cost
}

type routingAuditPoolSnapshot struct {
	Pool                    string   `json:"pool"`
	AccountType             string   `json:"account_type,omitempty"`
	TotalAccounts           int      `json:"total_accounts"`
	SchedulableAccounts     int      `json:"schedulable_accounts"`
	RateLimitedAccounts     int      `json:"rate_limited_accounts"`
	OverloadedAccounts      int      `json:"overloaded_accounts"`
	TempUnschedulable       int      `json:"temp_unschedulable"`
	QuotaLimitedAccounts    int      `json:"quota_limited_accounts"`
	QuotaLimitUSD           float64  `json:"quota_limit_usd,omitempty"`
	QuotaUsedUSD            float64  `json:"quota_used_usd,omitempty"`
	QuotaDailyLimitUSD      float64  `json:"quota_daily_limit_usd,omitempty"`
	QuotaDailyUsedUSD       float64  `json:"quota_daily_used_usd,omitempty"`
	QuotaWeeklyLimitUSD     float64  `json:"quota_weekly_limit_usd,omitempty"`
	QuotaWeeklyUsedUSD      float64  `json:"quota_weekly_used_usd,omitempty"`
	Codex5hMaxUsedPercent   *float64 `json:"codex_5h_max_used_percent,omitempty"`
	Codex5hAvgUsedPercent   *float64 `json:"codex_5h_avg_used_percent,omitempty"`
	Codex7dMaxUsedPercent   *float64 `json:"codex_7d_max_used_percent,omitempty"`
	Codex7dAvgUsedPercent   *float64 `json:"codex_7d_avg_used_percent,omitempty"`
	OldestSnapshotUpdatedAt string   `json:"oldest_snapshot_updated_at,omitempty"`
}

func buildRoutingAuditPoolSnapshots(accounts []Account) []routingAuditPoolSnapshot {
	byPool := make(map[string]*routingAuditPoolSnapshot)
	type codexAgg struct {
		sum5h, sum7d       float64
		count5h, count7d   int
		max5h, max7d       *float64
		oldestSnapshotTime *time.Time
	}
	codexByPool := make(map[string]*codexAgg)
	now := time.Now()
	for i := range accounts {
		account := &accounts[i]
		pool := classifyRoutingAuditAccountPool(account)
		if pool == "" {
			pool = RoutingAuditPoolUnknown
		}
		snap := byPool[pool]
		if snap == nil {
			snap = &routingAuditPoolSnapshot{Pool: pool, AccountType: account.Type}
			byPool[pool] = snap
		}
		snap.TotalAccounts++
		if account.IsSchedulable() {
			snap.SchedulableAccounts++
		}
		if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
			snap.RateLimitedAccounts++
		}
		if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
			snap.OverloadedAccounts++
		}
		if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
			snap.TempUnschedulable++
		}
		if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
			snap.QuotaLimitedAccounts++
		}
		snap.QuotaLimitUSD += account.GetQuotaLimit()
		snap.QuotaUsedUSD += account.GetQuotaUsed()
		snap.QuotaDailyLimitUSD += account.GetQuotaDailyLimit()
		snap.QuotaDailyUsedUSD += account.GetQuotaDailyUsed()
		snap.QuotaWeeklyLimitUSD += account.GetQuotaWeeklyLimit()
		snap.QuotaWeeklyUsedUSD += account.GetQuotaWeeklyUsed()

		agg := codexByPool[pool]
		if agg == nil {
			agg = &codexAgg{}
			codexByPool[pool] = agg
		}
		if account.Extra != nil {
			if progress := buildCodexUsageProgressFromExtra(account.Extra, "5h", now); progress != nil {
				used := progress.Utilization
				agg.sum5h += used
				agg.count5h++
				agg.max5h = maxFloatPtr(agg.max5h, used)
			}
			if progress := buildCodexUsageProgressFromExtra(account.Extra, "7d", now); progress != nil {
				used := progress.Utilization
				agg.sum7d += used
				agg.count7d++
				agg.max7d = maxFloatPtr(agg.max7d, used)
			}
			if updatedRaw, ok := account.Extra["codex_usage_updated_at"]; ok {
				if t, err := time.Parse(time.RFC3339, fmt.Sprint(updatedRaw)); err == nil {
					if agg.oldestSnapshotTime == nil || t.Before(*agg.oldestSnapshotTime) {
						tt := t
						agg.oldestSnapshotTime = &tt
					}
				}
				if t, err := time.Parse(time.RFC3339Nano, fmt.Sprint(updatedRaw)); err == nil {
					if agg.oldestSnapshotTime == nil || t.Before(*agg.oldestSnapshotTime) {
						tt := t
						agg.oldestSnapshotTime = &tt
					}
				}
			}
		}
	}
	out := make([]routingAuditPoolSnapshot, 0, len(byPool))
	for pool, snap := range byPool {
		if agg := codexByPool[pool]; agg != nil {
			if agg.count5h > 0 {
				avg := agg.sum5h / float64(agg.count5h)
				snap.Codex5hAvgUsedPercent = &avg
				snap.Codex5hMaxUsedPercent = agg.max5h
			}
			if agg.count7d > 0 {
				avg := agg.sum7d / float64(agg.count7d)
				snap.Codex7dAvgUsedPercent = &avg
				snap.Codex7dMaxUsedPercent = agg.max7d
			}
			if agg.oldestSnapshotTime != nil {
				snap.OldestSnapshotUpdatedAt = agg.oldestSnapshotTime.Format(time.RFC3339)
			}
		}
		out = append(out, *snap)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pool < out[j].Pool })
	return out
}

func filterRoutingAuditPoolSnapshots(pools []routingAuditPoolSnapshot, plus bool) []routingAuditPoolSnapshot {
	out := make([]routingAuditPoolSnapshot, 0, len(pools))
	for _, pool := range pools {
		isPlus := strings.Contains(pool.Pool, "plus") || pool.AccountType == AccountTypeOAuth || pool.AccountType == AccountTypeSetupToken
		if isPlus == plus {
			out = append(out, pool)
		}
	}
	return out
}

func classifyRoutingAuditAccountPool(account *Account) string {
	if account == nil {
		return RoutingAuditPoolUnknown
	}
	for _, key := range []string{"routing_pool", "pool"} {
		if account.Extra != nil {
			if raw, ok := account.Extra[key]; ok {
				if pool := strings.TrimSpace(fmt.Sprint(raw)); pool != "" {
					return pool
				}
			}
		}
	}
	name := strings.ToLower(account.Name)
	if strings.Contains(name, "apipod") {
		return "relay-apipod"
	}
	if account.IsOpenAIApiKey() {
		baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
		if baseURL != "" && !isOfficialOpenAIBaseURL(baseURL) {
			if strings.Contains(strings.ToLower(baseURL), "apipod") {
				return "relay-apipod"
			}
			return RoutingAuditPoolRelay
		}
	}
	if account.IsOpenAIOAuth() || account.Type == AccountTypeSetupToken {
		return RoutingAuditPoolTrustedPlus
	}
	return RoutingAuditPoolUnknown
}

func isOfficialOpenAIBaseURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "api.openai.com" || host == "openai.com" || strings.HasSuffix(host, ".openai.com")
}

func optionalAccountID(account *Account) *int64 {
	if account == nil || account.ID <= 0 {
		return nil
	}
	id := account.ID
	return &id
}

func resolveRoutingAuditDecisionReason(input RoutingAuditRecordInput) string {
	if input.PrivacyRedirect {
		return "privacy_redirect"
	}
	if input.QuotaRedirect {
		return "quota_redirect"
	}
	if input.ErrorRedirect {
		return "error_redirect"
	}
	if strings.TrimSpace(input.ScheduleDecision.Layer) != "" {
		return "scheduler_" + strings.TrimSpace(input.ScheduleDecision.Layer)
	}
	return "scheduler_selected"
}

func classifyRoutingAuditError(err error) string {
	if err == nil {
		return ""
	}
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "rate") || strings.Contains(lower, "429"):
		return "rate_limited"
	case strings.Contains(lower, "403") || strings.Contains(lower, "forbidden"):
		return "forbidden"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		return "timeout"
	default:
		return "upstream_error"
	}
}

func mustJSONRaw(v any) json.RawMessage {
	if v == nil {
		return json.RawMessage("null")
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return json.RawMessage("null")
	}
	return json.RawMessage(b)
}

func routingAuditFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func routingAuditMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloatPtr(current *float64, next float64) *float64 {
	if current == nil || next > *current {
		v := next
		return &v
	}
	return current
}
