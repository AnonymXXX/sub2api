//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type switchGroupRepoStub struct {
	GroupRepository
	groups map[int64]*Group
}

func (r *switchGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	group := r.groups[id]
	if group == nil {
		return nil, ErrGroupNotFound
	}
	copy := *group
	return &copy, nil
}

type switchSubscriptionRepoStub struct {
	UserSubscriptionRepository
	source        *UserSubscription
	targetHistory *UserSubscription
	created       *UserSubscription
	deletedIDs    []int64
	lockedUserID  int64
	active        []UserSubscription
}

func (r *switchSubscriptionRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.source == nil || r.source.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	copy := *r.source
	return &copy, nil
}

func (r *switchSubscriptionRepoStub) LockUser(_ context.Context, userID int64) error {
	r.lockedUserID = userID
	return nil
}

func (r *switchSubscriptionRepoStub) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	if r.active != nil {
		return append([]UserSubscription(nil), r.active...), nil
	}
	if r.source == nil || r.source.UserID != userID {
		return nil, nil
	}
	return []UserSubscription{*r.source}, nil
}

func (r *switchSubscriptionRepoStub) GetByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	if r.targetHistory == nil || r.targetHistory.UserID != userID || r.targetHistory.GroupID != groupID {
		return nil, ErrSubscriptionNotFound
	}
	copy := *r.targetHistory
	return &copy, nil
}

func (r *switchSubscriptionRepoStub) Delete(_ context.Context, id int64) error {
	r.deletedIDs = append(r.deletedIDs, id)
	return nil
}

func (r *switchSubscriptionRepoStub) Create(_ context.Context, sub *UserSubscription) error {
	copy := *sub
	copy.ID = 501
	sub.ID = copy.ID
	r.created = &copy
	return nil
}

type switchAPIKeyRepoStub struct {
	APIKeyRepository
	userID     int64
	oldGroupID int64
	newGroupID int64
	migrated   int64
}

func (r *switchAPIKeyRepoStub) UpdateGroupIDByUserAndGroup(_ context.Context, userID, oldGroupID, newGroupID int64) (int64, error) {
	r.userID = userID
	r.oldGroupID = oldGroupID
	r.newGroupID = newGroupID
	return r.migrated, nil
}

type switchAuthCacheInvalidatorStub struct {
	APIKeyAuthCacheInvalidator
	userIDs []int64
}

func (s *switchAuthCacheInvalidatorStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func TestSwitchSubscriptionCarriesStateAndMigratesOnlySourceGroupKeys(t *testing.T) {
	startsAt := time.Date(2026, time.July, 10, 8, 0, 0, 0, time.UTC)
	expiresAt := startsAt.AddDate(0, 0, 30)
	dailyWindow := startsAt.Add(2 * time.Hour)
	weeklyWindow := startsAt.Add(3 * time.Hour)
	monthlyWindow := startsAt.Add(4 * time.Hour)
	dailyLimit := 50.0
	weeklyLimit := 400.0
	monthlyLimit := 700.0

	groups := &switchGroupRepoStub{groups: map[int64]*Group{
		10: {
			ID: 10, Name: "OpenAI Pro", Platform: PlatformOpenAI,
			Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription,
		},
		20: {
			ID: 20, Name: "OpenAI Lite", Platform: PlatformOpenAI,
			Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription,
			DailyLimitUSD: &dailyLimit, WeeklyLimitUSD: &weeklyLimit, MonthlyLimitUSD: &monthlyLimit,
		},
	}}
	subscriptions := &switchSubscriptionRepoStub{
		source: &UserSubscription{
			ID: 100, UserID: 15, GroupID: 10,
			StartsAt: startsAt, ExpiresAt: expiresAt, Status: SubscriptionStatusActive,
			DailyWindowStart: &dailyWindow, WeeklyWindowStart: &weeklyWindow, MonthlyWindowStart: &monthlyWindow,
			DailyUsageUSD: 75, WeeklyUsageUSD: 350, MonthlyUsageUSD: 725, MonthlyBonusUSD: 20,
			Notes: "original subscription",
		},
		targetHistory: &UserSubscription{ID: 99, UserID: 15, GroupID: 20, Status: SubscriptionStatusExpired},
	}
	keys := &switchAPIKeyRepoStub{migrated: 2}
	authCache := &switchAuthCacheInvalidatorStub{}
	svc := NewSubscriptionServiceWithSwitchDependencies(groups, subscriptions, keys, authCache, nil, nil, nil)
	t.Cleanup(svc.Stop)

	result, err := svc.SwitchSubscription(context.Background(), 100, 20, 9)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(100), result.PreviousSubscriptionID)
	require.Equal(t, int64(2), result.MigratedKeys)
	require.Equal(t, []string{QuotaWarningDaily, QuotaWarningMonthly}, result.QuotaWarnings)
	require.Equal(t, int64(15), subscriptions.lockedUserID)
	require.Equal(t, []int64{99, 100}, subscriptions.deletedIDs)
	require.Equal(t, int64(15), keys.userID)
	require.Equal(t, int64(10), keys.oldGroupID)
	require.Equal(t, int64(20), keys.newGroupID)
	require.Equal(t, []int64{15}, authCache.userIDs)

	created := subscriptions.created
	require.NotNil(t, created)
	require.Equal(t, int64(20), created.GroupID)
	require.Equal(t, startsAt, created.StartsAt)
	require.Equal(t, expiresAt, created.ExpiresAt)
	require.Equal(t, dailyWindow, *created.DailyWindowStart)
	require.Equal(t, weeklyWindow, *created.WeeklyWindowStart)
	require.Equal(t, monthlyWindow, *created.MonthlyWindowStart)
	require.Equal(t, 75.0, created.DailyUsageUSD)
	require.Equal(t, 350.0, created.WeeklyUsageUSD)
	require.Equal(t, 725.0, created.MonthlyUsageUSD)
	require.Equal(t, 20.0, created.MonthlyBonusUSD)
	require.Equal(t, int64(9), *created.AssignedBy)
	require.Equal(t, int64(501), result.Subscription.ID)
}

func TestSwitchSubscriptionRejectsInvalidTargetsAndStaleSource(t *testing.T) {
	now := time.Now()
	baseSource := &UserSubscription{
		ID: 100, UserID: 15, GroupID: 10,
		StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), Status: SubscriptionStatusActive,
	}
	validSourceGroup := &Group{
		ID: 10, Name: "OpenAI Pro", Platform: PlatformOpenAI,
		Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription,
	}
	validTarget := &Group{
		ID: 20, Name: "OpenAI Lite", Platform: PlatformOpenAI,
		Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription,
	}

	tests := []struct {
		name          string
		targetGroupID int64
		target        *Group
		active        []UserSubscription
		wantErr       error
	}{
		{name: "same group", targetGroupID: 10, target: validSourceGroup, wantErr: ErrSubscriptionSwitchSameGroup},
		{name: "inactive target", targetGroupID: 20, target: func() *Group { copy := *validTarget; copy.Status = StatusDisabled; return &copy }(), wantErr: ErrSubscriptionSwitchInactive},
		{name: "standard target", targetGroupID: 20, target: func() *Group { copy := *validTarget; copy.SubscriptionType = SubscriptionTypeStandard; return &copy }(), wantErr: ErrGroupNotSubscriptionType},
		{name: "cross platform target", targetGroupID: 20, target: func() *Group { copy := *validTarget; copy.Platform = PlatformAnthropic; return &copy }(), wantErr: ErrSubscriptionSwitchPlatform},
		{name: "source no longer active", targetGroupID: 20, target: validTarget, active: []UserSubscription{}, wantErr: ErrSubscriptionSwitchSource},
		{name: "multiple active subscriptions", targetGroupID: 20, target: validTarget, active: []UserSubscription{*baseSource, {ID: 101, UserID: 15, GroupID: 30}}, wantErr: ErrSubscriptionSwitchSource},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			groups := &switchGroupRepoStub{groups: map[int64]*Group{10: validSourceGroup, test.targetGroupID: test.target}}
			subscriptions := &switchSubscriptionRepoStub{source: baseSource, active: test.active}
			svc := NewSubscriptionServiceWithSwitchDependencies(groups, subscriptions, &switchAPIKeyRepoStub{}, nil, nil, nil, nil)
			t.Cleanup(svc.Stop)

			result, err := svc.SwitchSubscription(context.Background(), baseSource.ID, test.targetGroupID, 9)
			require.ErrorIs(t, err, test.wantErr)
			require.Nil(t, result)
			require.Nil(t, subscriptions.created)
			require.Empty(t, subscriptions.deletedIDs)
		})
	}
}
