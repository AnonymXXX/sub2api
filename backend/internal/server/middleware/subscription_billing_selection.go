package middleware

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// selectSubscriptionBillingGroup returns a request-local API key copy whose
// group points at the user's single usable subscription. The persisted key and
// its original balance group remain unchanged.
func selectSubscriptionBillingGroup(c *gin.Context, apiKey *service.APIKey, subscriptionService *service.SubscriptionService) (*service.APIKey, *service.UserSubscription, error) {
	if c == nil || apiKey == nil || apiKey.User == nil || apiKey.Group == nil || subscriptionService == nil {
		return apiKey, nil, nil
	}

	subscription, err := subscriptionService.GetActiveUserSubscription(c.Request.Context(), apiKey.User.ID)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionNotFound) {
			return apiKey, nil, nil
		}
		return nil, nil, err
	}
	requestPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	if subscription.Group == nil || !subscriptionGroupSupportsRequest(c.Request.URL.Path, requestPlatform, subscription.Group) {
		return apiKey, nil, nil
	}

	needsMaintenance, validationErr := subscriptionService.ValidateAndCheckLimits(subscription, subscription.Group)
	if needsMaintenance {
		originalGroup := subscription.Group
		refreshed, maintenanceErr := subscriptionService.EnsureWindowMaintenance(c.Request.Context(), subscription)
		if maintenanceErr != nil {
			return nil, nil, maintenanceErr
		}
		subscription = refreshed
		if subscription.Group == nil {
			subscription.Group = originalGroup
		}
		if subscription.Group == nil {
			return apiKey, nil, nil
		}
		_, validationErr = subscriptionService.ValidateAndCheckLimits(subscription, subscription.Group)
	}
	if validationErr != nil {
		if errors.Is(validationErr, service.ErrSubscriptionExpired) ||
			errors.Is(validationErr, service.ErrSubscriptionSuspended) ||
			errors.Is(validationErr, service.ErrDailyLimitExceeded) ||
			errors.Is(validationErr, service.ErrWeeklyLimitExceeded) ||
			errors.Is(validationErr, service.ErrMonthlyLimitExceeded) {
			return apiKey, nil, nil
		}
		return nil, nil, validationErr
	}

	runtimeKey := *apiKey
	// The auth snapshot override belongs to the API key's persisted group. A
	// subscription group must resolve its own override instead of inheriting it.
	runtimeUser := *apiKey.User
	runtimeUser.UserGroupRPMOverride = nil
	runtimeKey.User = &runtimeUser
	runtimeKey.BalanceGroupID = cloneGroupID(apiKey.GroupID)
	runtimeKey.BalanceGroup = apiKey.Group
	runtimeKey.GroupID = groupIDPtr(subscription.Group.ID)
	runtimeKey.Group = subscription.Group
	return &runtimeKey, subscription, nil
}

func subscriptionGroupSupportsRequest(path, requestPlatform string, subscriptionGroup *service.Group) bool {
	if subscriptionGroup == nil || !subscriptionGroup.IsActive() || !subscriptionGroup.IsSubscriptionType() {
		return false
	}
	if strings.TrimSpace(requestPlatform) == "" || requestPlatform != subscriptionGroup.Platform {
		return false
	}

	path = strings.ToLower(strings.TrimSpace(path))
	if strings.Contains(path, "/images/batches") || strings.Contains(path, "/batch/images") {
		return subscriptionGroup.AllowBatchImageGeneration
	}
	if strings.Contains(path, "/images/") || strings.HasSuffix(path, "/images") {
		return subscriptionGroup.AllowImageGeneration
	}
	if strings.HasSuffix(path, "/messages") && subscriptionGroup.Platform == service.PlatformOpenAI {
		return subscriptionGroup.AllowMessagesDispatch
	}
	return true
}

func cloneGroupID(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func groupIDPtr(value int64) *int64 {
	return &value
}
