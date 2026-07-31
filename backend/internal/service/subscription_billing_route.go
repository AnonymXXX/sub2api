package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

func balanceFallbackRoute(ctx context.Context, selectedGroupID *int64) (*APIKey, *int64, bool) {
	if ctx == nil {
		return nil, nil, false
	}
	apiKey, ok := ctx.Value(ctxkey.RuntimeAPIKey).(*APIKey)
	if !ok || apiKey == nil || apiKey.BalanceGroup == nil || apiKey.BalanceGroupID == nil || apiKey.GroupID == nil {
		return nil, nil, false
	}
	if selectedGroupID == nil || *selectedGroupID != *apiKey.GroupID || *apiKey.BalanceGroupID == *apiKey.GroupID {
		return nil, nil, false
	}
	if apiKey.User == nil || apiKey.User.Balance <= 0 || !apiKey.BalanceGroup.IsActive() {
		return nil, nil, false
	}
	return apiKey, apiKey.BalanceGroupID, true
}

func activateBalanceFallbackRoute(apiKey *APIKey) {
	if apiKey == nil || apiKey.BalanceGroup == nil {
		return
	}
	apiKey.GroupID = apiKey.BalanceGroupID
	apiKey.Group = apiKey.BalanceGroup
}

func validateBalanceFallbackRoute(ctx context.Context, apiKey *APIKey) error {
	if apiKey == nil {
		return nil
	}
	return apiKey.validateBalanceFallback(ctx)
}
