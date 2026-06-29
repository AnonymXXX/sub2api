package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	httppool "github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

const (
	relayQuotaRequestTimeout = 12 * time.Second
	relayQuotaMaxBodyBytes   = 512 * 1024
)

type RelayQuotaSummary struct {
	Provider                string   `json:"provider,omitempty"`
	Pool                    string   `json:"pool,omitempty"`
	PlanName                string   `json:"plan_name,omitempty"`
	SubscriptionStatus      string   `json:"subscription_status,omitempty"`
	SubscriptionSource      string   `json:"subscription_source,omitempty"`
	BalanceRemaining        *float64 `json:"balance_remaining,omitempty"`
	BalanceUnit             string   `json:"balance_unit,omitempty"`
	BalanceSource           string   `json:"balance_source,omitempty"`
	BalanceSyncedAt         string   `json:"balance_synced_at,omitempty"`
	BalanceValid            *bool    `json:"balance_valid,omitempty"`
	DailyLimitUSD           *float64 `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD          *float64 `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD         *float64 `json:"monthly_limit_usd,omitempty"`
	RateMultiplier          *float64 `json:"rate_multiplier,omitempty"`
	PlanPrice               *float64 `json:"plan_price,omitempty"`
	PlanCurrency            string   `json:"plan_currency,omitempty"`
	ValidityDays            *int     `json:"validity_days,omitempty"`
	LastSyncedAt            string   `json:"last_synced_at,omitempty"`
	Error                   string   `json:"error,omitempty"`
	Refreshable             bool     `json:"refreshable"`
	SubscriptionRefreshable bool     `json:"subscription_refreshable"`
}

type RelayQuotaRefreshResult struct {
	Summary *RelayQuotaSummary `json:"summary"`
}

func BuildRelayQuotaSummary(account *Account) *RelayQuotaSummary {
	if account == nil || !isRelayQuotaAccount(account) {
		return nil
	}
	extra := account.Extra
	if extra == nil {
		extra = map[string]any{}
	}

	summary := &RelayQuotaSummary{
		Provider:                relayQuotaProvider(account),
		Pool:                    relayFirstNonEmptyString(extraString(extra, "routing_pool"), extraString(extra, "pool")),
		PlanName:                relayFirstNonEmptyString(extraString(extra, "relay_subscription_plan_name"), extraString(extra, "quota_plan")),
		SubscriptionStatus:      relayFirstNonEmptyString(extraString(extra, "relay_subscription_status"), "configured"),
		SubscriptionSource:      relayFirstNonEmptyString(extraString(extra, "relay_subscription_source"), "configured"),
		BalanceSource:           extraString(extra, "relay_balance_source"),
		BalanceSyncedAt:         extraString(extra, "relay_balance_synced_at"),
		LastSyncedAt:            relayFirstNonEmptyString(extraString(extra, "relay_subscription_synced_at"), extraString(extra, "relay_balance_synced_at")),
		Error:                   relayFirstNonEmptyString(extraString(extra, "relay_balance_error"), extraString(extra, "relay_subscription_error")),
		Refreshable:             relayQuotaCanRefreshBalance(account),
		SubscriptionRefreshable: relayQuotaCanRefreshSubscription(account),
		PlanCurrency:            relayFirstNonEmptyString(extraString(extra, "relay_subscription_price_currency"), "CNY"),
	}

	if summary.Pool == "" && strings.Contains(strings.ToLower(account.Name), "apipod") {
		summary.Pool = "relay-apipod"
	}
	if summary.PlanName == "" && strings.Contains(strings.ToLower(account.Name), "apipod") {
		summary.PlanName = "Codex Basic"
	}
	if v, ok := optionalExtraFloat(extra, "relay_balance_remaining_usd", "relay_balance_remaining", "balance_remaining"); ok {
		summary.BalanceRemaining = &v
	}
	summary.BalanceUnit = relayFirstNonEmptyString(extraString(extra, "relay_balance_unit"), "USD")
	if v, ok := optionalExtraBool(extra, "relay_balance_valid"); ok {
		summary.BalanceValid = &v
	}
	if v, ok := optionalExtraFloat(extra, "relay_subscription_daily_limit_usd", "quota_daily_limit_usd", "quota_daily_limit"); ok {
		summary.DailyLimitUSD = &v
	}
	if v, ok := optionalExtraFloat(extra, "relay_subscription_weekly_limit_usd", "quota_weekly_limit_usd", "quota_weekly_limit"); ok {
		summary.WeeklyLimitUSD = &v
	}
	if v, ok := optionalExtraFloat(extra, "relay_subscription_monthly_limit_usd", "quota_monthly_limit_usd", "quota_limit"); ok {
		summary.MonthlyLimitUSD = &v
	}
	if v, ok := optionalExtraFloat(extra, "relay_subscription_rate_multiplier", "quota_billing_multiplier"); ok {
		summary.RateMultiplier = &v
	}
	if v, ok := optionalExtraFloat(extra, "relay_subscription_plan_price", "quota_plan_price", "quota_monthly_price_rmb"); ok {
		summary.PlanPrice = &v
	}
	if v, ok := optionalExtraInt(extra, "relay_subscription_validity_days", "quota_validity_days"); ok {
		summary.ValidityDays = &v
	}
	return summary
}

func (s *adminServiceImpl) RefreshRelayQuotaSummary(ctx context.Context, id int64) (*RelayQuotaSummary, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isRelayQuotaAccount(account) {
		return nil, infraerrors.BadRequest("RELAY_QUOTA_UNSUPPORTED_ACCOUNT", "relay quota is only supported for OpenAI API key relay accounts")
	}

	now := time.Now().UTC()
	updates := make(map[string]any)
	balanceErr := s.refreshRelayBalance(ctx, account, now, updates)
	subscriptionErr := s.refreshRelaySubscription(ctx, account, now, updates)
	if balanceErr != nil {
		updates["relay_balance_error"] = sanitizeRelayQuotaError(balanceErr)
	}
	if subscriptionErr != nil {
		updates["relay_subscription_error"] = sanitizeRelayQuotaError(subscriptionErr)
	}
	if len(updates) > 0 {
		if err := s.accountRepo.UpdateExtra(ctx, id, updates); err != nil {
			return nil, err
		}
		account, err = s.accountRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	if balanceErr != nil || subscriptionErr != nil {
		slog.Warn("relay_quota_refresh_partial_failure", "account_id", id, "balance_error", sanitizeRelayQuotaError(balanceErr), "subscription_error", sanitizeRelayQuotaError(subscriptionErr))
	}
	return BuildRelayQuotaSummary(account), nil
}

func (s *adminServiceImpl) refreshRelayBalance(ctx context.Context, account *Account, now time.Time, updates map[string]any) error {
	if !relayQuotaCanRefreshBalance(account) {
		return fmt.Errorf("missing relay api key or base url")
	}
	endpoint := strings.TrimRight(relayQuotaAPIBase(account), "/") + "/v1/usage"
	body, status, err := relayQuotaGET(ctx, endpoint, account.GetOpenAIApiKey())
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("balance endpoint returned HTTP %d", status)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("parse balance response: %w", err)
	}
	remaining, ok := relayQuotaParseRemaining(payload)
	if !ok {
		return fmt.Errorf("balance response missing remaining")
	}
	unit := relayFirstNonEmptyString(jsonString(payload["unit"]), jsonString(mapValue(payload, "quota", "unit")), "USD")
	updates["relay_balance_remaining"] = remaining
	if strings.EqualFold(unit, "USD") {
		updates["relay_balance_remaining_usd"] = remaining
	}
	updates["relay_balance_unit"] = unit
	updates["relay_balance_valid"] = remaining > 0
	updates["relay_balance_source"] = "upstream:/v1/usage"
	updates["relay_balance_synced_at"] = now.Format(time.RFC3339)
	updates["relay_balance_error"] = ""
	return nil
}

func (s *adminServiceImpl) refreshRelaySubscription(ctx context.Context, account *Account, now time.Time, updates map[string]any) error {
	if !relayQuotaCanRefreshSubscription(account) {
		return nil
	}
	endpoint := strings.TrimRight(relayQuotaAPIBase(account), "/") + "/api/v1/subscriptions?timezone=" + url.QueryEscape("Asia/Shanghai")
	body, status, err := relayQuotaGET(ctx, endpoint, relayQuotaSubscriptionToken(account))
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("subscription endpoint returned HTTP %d", status)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Plans []relayQuotaSubscriptionPlan `json:"plans"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("parse subscription response: %w", err)
	}
	if envelope.Code != 0 {
		return fmt.Errorf("subscription endpoint code %d", envelope.Code)
	}
	plan := selectRelayQuotaPlan(envelope.Data.Plans, account)
	if plan == nil {
		updates["relay_subscription_status"] = "none"
		updates["relay_subscription_source"] = "upstream:/api/v1/subscriptions"
		updates["relay_subscription_synced_at"] = now.Format(time.RFC3339)
		updates["relay_subscription_error"] = ""
		return nil
	}
	updates["relay_subscription_plan_name"] = relayFirstNonEmptyString(plan.Name, plan.GroupName)
	updates["relay_subscription_status"] = "catalog"
	updates["relay_subscription_source"] = "upstream:/api/v1/subscriptions"
	updates["relay_subscription_daily_limit_usd"] = plan.DailyLimitUSD
	updates["relay_subscription_weekly_limit_usd"] = plan.WeeklyLimitUSD
	updates["relay_subscription_monthly_limit_usd"] = plan.MonthlyLimitUSD
	updates["relay_subscription_rate_multiplier"] = plan.RateMultiplier
	updates["relay_subscription_plan_price"] = plan.Price
	updates["relay_subscription_price_currency"] = "CNY"
	updates["relay_subscription_validity_days"] = plan.ValidityDays
	updates["relay_subscription_synced_at"] = now.Format(time.RFC3339)
	updates["relay_subscription_error"] = ""
	return nil
}

type relayQuotaSubscriptionPlan struct {
	GroupPlatform   string  `json:"group_platform"`
	GroupName       string  `json:"group_name"`
	Name            string  `json:"name"`
	Price           float64 `json:"price"`
	RateMultiplier  float64 `json:"rate_multiplier"`
	DailyLimitUSD   float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD float64 `json:"monthly_limit_usd"`
	ValidityDays    int     `json:"validity_days"`
}

func relayQuotaGET(ctx context.Context, endpoint, token string) ([]byte, int, error) {
	client, err := httppool.GetClient(httppool.Options{
		Timeout:               relayQuotaRequestTimeout,
		ResponseHeaderTimeout: relayQuotaRequestTimeout,
		ValidateResolvedIP:    true,
	})
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, relayQuotaMaxBodyBytes))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func isRelayQuotaAccount(account *Account) bool {
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeAPIKey {
		return false
	}
	if strings.EqualFold(account.GetExtraString("routing_pool"), "relay-apipod") {
		return true
	}
	if strings.Contains(strings.ToLower(account.GetOpenAIBaseURL()), "apipod") {
		return true
	}
	return strings.Contains(strings.ToLower(account.Name), "apipod")
}

func relayQuotaProvider(account *Account) string {
	if account == nil {
		return ""
	}
	if strings.Contains(strings.ToLower(account.GetOpenAIBaseURL()), "apipod") || strings.Contains(strings.ToLower(account.Name), "apipod") {
		return "APIPod Code"
	}
	return "Relay"
}

func relayQuotaAPIBase(account *Account) string {
	baseURL := strings.TrimRight(account.GetOpenAIBaseURL(), "/")
	if strings.HasSuffix(baseURL, "/v1") {
		baseURL = strings.TrimSuffix(baseURL, "/v1")
	}
	return baseURL
}

func relayQuotaCanRefreshBalance(account *Account) bool {
	return account != nil && strings.TrimSpace(account.GetOpenAIBaseURL()) != "" && strings.TrimSpace(account.GetOpenAIApiKey()) != ""
}

func relayQuotaCanRefreshSubscription(account *Account) bool {
	return strings.TrimSpace(relayQuotaSubscriptionToken(account)) != ""
}

func relayQuotaSubscriptionToken(account *Account) string {
	if account == nil {
		return ""
	}
	for _, key := range []string{"web_session_token", "subscription_token", "dashboard_token"} {
		if token := strings.TrimSpace(account.GetCredential(key)); token != "" {
			return token
		}
	}
	return ""
}

func selectRelayQuotaPlan(plans []relayQuotaSubscriptionPlan, account *Account) *relayQuotaSubscriptionPlan {
	if len(plans) == 0 {
		return nil
	}
	configured := strings.ToLower(strings.TrimSpace(relayFirstNonEmptyString(account.GetExtraString("quota_plan"), account.GetExtraString("relay_subscription_plan_name"))))
	for i := range plans {
		name := strings.ToLower(strings.TrimSpace(relayFirstNonEmptyString(plans[i].Name, plans[i].GroupName)))
		if configured != "" && (name == configured || strings.Contains(configured, name) || strings.Contains(name, configured)) {
			return &plans[i]
		}
	}
	for i := range plans {
		if strings.EqualFold(plans[i].GroupPlatform, PlatformOpenAI) {
			return &plans[i]
		}
	}
	return &plans[0]
}

func relayQuotaParseRemaining(payload map[string]any) (float64, bool) {
	if v, ok := numberFromAny(payload["remaining"]); ok {
		return v, true
	}
	if v, ok := numberFromAny(mapValue(payload, "quota", "remaining")); ok {
		return v, true
	}
	if v, ok := numberFromAny(payload["balance"]); ok {
		return v, true
	}
	return 0, false
}

func mapValue(root map[string]any, path ...string) any {
	var cur any = root
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[key]
	}
	return cur
}

func optionalExtraFloat(extra map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if v, ok := numberFromAny(extra[key]); ok {
			return v, true
		}
	}
	return 0, false
}

func optionalExtraInt(extra map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		if v, ok := numberFromAny(extra[key]); ok {
			return int(v), true
		}
	}
	return 0, false
}

func optionalExtraBool(extra map[string]any, key string) (bool, bool) {
	v, ok := extra[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func extraString(extra map[string]any, key string) string {
	if extra == nil {
		return ""
	}
	return jsonString(extra[key])
}

func jsonString(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	default:
		return ""
	}
}

func numberFromAny(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return finiteNumber(value)
	case float32:
		return finiteNumber(float64(value))
	case int:
		return finiteNumber(float64(value))
	case int64:
		return finiteNumber(float64(value))
	case int32:
		return finiteNumber(float64(value))
	case json.Number:
		parsed, err := value.Float64()
		if err != nil {
			return 0, false
		}
		return finiteNumber(parsed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, false
		}
		return finiteNumber(parsed)
	default:
		return 0, false
	}
}

func finiteNumber(v float64) (float64, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func relayFirstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func sanitizeRelayQuotaError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 180 {
		msg = msg[:180]
	}
	return msg
}
