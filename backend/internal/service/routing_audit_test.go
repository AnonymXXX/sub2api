package service

import (
	"strings"
	"testing"
)

func TestRoutingAuditClassifyAccountPool_DoesNotLeakBaseURL(t *testing.T) {
	account := &Account{
		ID:       1,
		Name:     "relay account",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://secret-token.example.com/v1",
			"api_key":  "sample-token-value",
		},
	}

	pool := classifyRoutingAuditAccountPool(account)
	if pool != RoutingAuditPoolRelay {
		t.Fatalf("pool = %q, want %q", pool, RoutingAuditPoolRelay)
	}
	if strings.Contains(pool, "secret") || strings.Contains(pool, "sample-token") || strings.Contains(pool, "example.com") {
		t.Fatalf("pool classification leaked credential/base URL data: %q", pool)
	}
}

func TestRoutingAuditBuildPoolSnapshots_UsesAggregateOnly(t *testing.T) {
	accounts := []Account{
		{
			ID:          1,
			Name:        "plus-a",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"access_token": "secret-access-token"},
			Extra: map[string]any{
				"codex_5h_used_percent": 25.0,
				"codex_7d_used_percent": 50.0,
			},
		},
	}

	snapshots := buildRoutingAuditPoolSnapshots(accounts)
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	got := snapshots[0]
	if got.Pool != RoutingAuditPoolTrustedPlus {
		t.Fatalf("pool = %q, want %q", got.Pool, RoutingAuditPoolTrustedPlus)
	}
	if got.TotalAccounts != 1 || got.SchedulableAccounts != 1 {
		t.Fatalf("unexpected aggregate counts: %+v", got)
	}
	if got.Codex5hMaxUsedPercent == nil || *got.Codex5hMaxUsedPercent != 25.0 {
		t.Fatalf("5h max percent = %v, want 25", got.Codex5hMaxUsedPercent)
	}
}
