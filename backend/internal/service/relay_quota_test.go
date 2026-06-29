package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRelayQuotaSummary_FromConfiguredAndSyncedExtra(t *testing.T) {
	account := &Account{
		ID:       1,
		Name:     "APIPod Code",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":           "https://code.apipod.ai/v1",
			"api_key":            "sk-test",
			"web_session_token":  "dashboard-token",
			"subscription_token": "subscription-token",
		},
		Extra: map[string]any{
			"routing_pool":                         "relay-apipod",
			"relay_balance_remaining_usd":          123.45,
			"relay_balance_unit":                   "USD",
			"relay_balance_source":                 "upstream:/v1/usage",
			"relay_balance_synced_at":              "2026-06-29T10:00:00Z",
			"relay_balance_valid":                  true,
			"relay_subscription_plan_name":         "Codex Basic",
			"relay_subscription_status":            "catalog",
			"relay_subscription_source":            "upstream:/api/v1/subscriptions",
			"relay_subscription_daily_limit_usd":   150,
			"relay_subscription_weekly_limit_usd":  550,
			"relay_subscription_monthly_limit_usd": 1500,
			"relay_subscription_rate_multiplier":   0.8,
			"relay_subscription_plan_price":        249,
			"relay_subscription_price_currency":    "CNY",
			"relay_subscription_validity_days":     30,
			"relay_subscription_synced_at":         "2026-06-29T10:01:00Z",
		},
	}

	summary := BuildRelayQuotaSummary(account)
	require.NotNil(t, summary)
	require.Equal(t, "APIPod Code", summary.Provider)
	require.Equal(t, "relay-apipod", summary.Pool)
	require.Equal(t, "Codex Basic", summary.PlanName)
	require.Equal(t, "catalog", summary.SubscriptionStatus)
	require.Equal(t, "upstream:/api/v1/subscriptions", summary.SubscriptionSource)
	require.True(t, summary.Refreshable)
	require.True(t, summary.SubscriptionRefreshable)
	require.Equal(t, "USD", summary.BalanceUnit)
	require.Equal(t, "upstream:/v1/usage", summary.BalanceSource)
	require.Equal(t, "2026-06-29T10:00:00Z", summary.BalanceSyncedAt)
	require.Equal(t, "2026-06-29T10:01:00Z", summary.LastSyncedAt)
	require.NotNil(t, summary.BalanceValid)
	require.True(t, *summary.BalanceValid)
	require.NotNil(t, summary.BalanceRemaining)
	require.Equal(t, 123.45, *summary.BalanceRemaining)
	require.NotNil(t, summary.DailyLimitUSD)
	require.Equal(t, float64(150), *summary.DailyLimitUSD)
	require.NotNil(t, summary.WeeklyLimitUSD)
	require.Equal(t, float64(550), *summary.WeeklyLimitUSD)
	require.NotNil(t, summary.MonthlyLimitUSD)
	require.Equal(t, float64(1500), *summary.MonthlyLimitUSD)
	require.NotNil(t, summary.RateMultiplier)
	require.Equal(t, 0.8, *summary.RateMultiplier)
	require.NotNil(t, summary.PlanPrice)
	require.Equal(t, float64(249), *summary.PlanPrice)
	require.Equal(t, "CNY", summary.PlanCurrency)
	require.NotNil(t, summary.ValidityDays)
	require.Equal(t, 30, *summary.ValidityDays)
}

func TestRelayQuotaParseRemainingSupportedShapes(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    float64
	}{
		{name: "top level remaining", payload: `{"remaining": 42.5}`, want: 42.5},
		{name: "nested quota remaining", payload: `{"quota":{"remaining":"88.25"}}`, want: 88.25},
		{name: "top level balance", payload: `{"balance": 12}`, want: 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload map[string]any
			decoder := json.NewDecoder(strings.NewReader(tt.payload))
			decoder.UseNumber()
			require.NoError(t, decoder.Decode(&payload))

			got, ok := relayQuotaParseRemaining(payload)
			require.True(t, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSelectRelayQuotaPlanPrefersConfiguredOrOpenAIPlan(t *testing.T) {
	plans := []relayQuotaSubscriptionPlan{
		{Name: "Other", GroupPlatform: PlatformAnthropic, MonthlyLimitUSD: 10},
		{Name: "Codex Basic", GroupName: "Codex Basic", GroupPlatform: PlatformOpenAI, MonthlyLimitUSD: 1500},
	}

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra:    map[string]any{"quota_plan": "Codex Basic"},
	}
	plan := selectRelayQuotaPlan(plans, account)
	require.NotNil(t, plan)
	require.Equal(t, "Codex Basic", plan.Name)

	account.Extra = nil
	plan = selectRelayQuotaPlan(plans, account)
	require.NotNil(t, plan)
	require.Equal(t, PlatformOpenAI, plan.GroupPlatform)
}
