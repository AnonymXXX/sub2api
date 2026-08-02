//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type failingSwitchAPIKeyRepository struct {
	service.APIKeyRepository
}

func (f failingSwitchAPIKeyRepository) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	return 0, errors.New("forced key migration failure")
}

func TestSwitchSubscriptionRollsBackEveryDatabaseWriteWhenKeyMigrationFails(t *testing.T) {
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	user, err := integrationEntClient.User.Create().
		SetEmail(fmt.Sprintf("subscription-switch-rollback-%d@example.com", stamp)).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)

	createGroup := func(name string) int64 {
		group, createErr := integrationEntClient.Group.Create().
			SetName(fmt.Sprintf("%s-%d", name, stamp)).
			SetPlatform(service.PlatformOpenAI).
			SetStatus(service.StatusActive).
			SetSubscriptionType(service.SubscriptionTypeSubscription).
			Save(ctx)
		require.NoError(t, createErr)
		return group.ID
	}
	sourceGroupID := createGroup("switch-source")
	targetGroupID := createGroup("switch-target")
	now := time.Now()
	targetHistory, err := integrationEntClient.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(targetGroupID).
		SetStartsAt(now.Add(-60 * 24 * time.Hour)).
		SetExpiresAt(now.Add(-30 * 24 * time.Hour)).
		SetStatus(service.SubscriptionStatusExpired).
		SetAssignedAt(now.Add(-60 * 24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)
	source, err := integrationEntClient.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(sourceGroupID).
		SetStartsAt(now.Add(-24 * time.Hour)).
		SetExpiresAt(now.Add(29 * 24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		SetDailyUsageUsd(12.5).
		SetAssignedAt(now.Add(-24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	subRepo := NewUserSubscriptionRepository(integrationEntClient)
	groupRepo := NewGroupRepository(integrationEntClient, integrationDB)
	svc := service.NewSubscriptionServiceWithSwitchDependencies(
		groupRepo,
		subRepo,
		failingSwitchAPIKeyRepository{},
		nil,
		nil,
		integrationEntClient,
		nil,
	)
	t.Cleanup(svc.Stop)

	result, err := svc.SwitchSubscription(ctx, source.ID, targetGroupID, user.ID)
	require.ErrorContains(t, err, "forced key migration failure")
	require.Nil(t, result)

	sourceAfter, err := subRepo.GetByID(ctx, source.ID)
	require.NoError(t, err, "source soft delete must be rolled back")
	require.Equal(t, 12.5, sourceAfter.DailyUsageUSD)
	targetAfter, err := subRepo.GetByID(ctx, targetHistory.ID)
	require.NoError(t, err, "target history soft delete must be rolled back")
	require.Equal(t, service.SubscriptionStatusExpired, targetAfter.Status)
	targetRows, err := integrationEntClient.UserSubscription.Query().
		Where(usersubscription.UserIDEQ(user.ID), usersubscription.GroupIDEQ(targetGroupID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, targetRows, "rolled back create must not leave an extra target subscription")
}
