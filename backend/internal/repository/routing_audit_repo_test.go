package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildRoutingAuditWhere(t *testing.T) {
	start := time.Date(2026, 6, 28, 8, 30, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	where, args := buildRoutingAuditWhere(service.RoutingAuditFilter{
		UserID:               11,
		APIKeyID:             22,
		AccountID:            33,
		GroupID:              44,
		Model:                "gpt-5.5",
		SelectedPool:         "relay-apipod",
		DecisionReason:       "scheduler_load_balance",
		RoutingPolicyVersion: "routing-audit-v1",
		StartTime:            &start,
		EndTime:              &end,
	})

	for _, want := range []string{
		"user_id = $1",
		"api_key_id = $2",
		"account_id = $3",
		"group_id = $4",
		"model = $5",
		"selected_pool = $6",
		"decision_reason = $7",
		"routing_policy_version = $8",
		"created_at >= $9",
		"created_at < $10",
	} {
		if !strings.Contains(where, want) {
			t.Fatalf("where missing %q\nfull: %s", want, where)
		}
	}
	if len(args) != 10 {
		t.Fatalf("args len = %d, want 10", len(args))
	}
}

func TestNormalizeRoutingAuditSortByWhitelist(t *testing.T) {
	if got := normalizeRoutingAuditSortBy("created_at; DROP TABLE accounts"); got != "created_at" {
		t.Fatalf("unexpected unsafe sort fallback: %s", got)
	}
	if got := normalizeRoutingAuditSortBy("actual_cost"); got != "actual_cost" {
		t.Fatalf("sort = %s, want actual_cost", got)
	}
}
