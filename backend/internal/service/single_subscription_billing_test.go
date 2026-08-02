//go:build unit

package service

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type singleSubscriptionRepoStub struct {
	*subscriptionUserSubRepoStub
	active            *UserSubscription
	lockCalls         int
	bonusValue        float64
	monthlyResetCalls int
}

func newSingleSubscriptionRepoStub() *singleSubscriptionRepoStub {
	return &singleSubscriptionRepoStub{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
}

func (s *singleSubscriptionRepoStub) LockUser(context.Context, int64) error {
	s.lockCalls++
	return nil
}

func (s *singleSubscriptionRepoStub) GetActiveByUserID(_ context.Context, userID int64) (*UserSubscription, error) {
	if s.active == nil || s.active.UserID != userID {
		return nil, ErrSubscriptionNotFound
	}
	cp := *s.active
	return &cp, nil
}

func (s *singleSubscriptionRepoStub) SetMonthlyBonus(_ context.Context, id int64, amountUSD float64) error {
	sub := s.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.MonthlyBonusUSD = amountUSD
	s.bonusValue = amountUSD
	return nil
}

func (s *singleSubscriptionRepoStub) ActivateWindows(_ context.Context, id int64, start time.Time) error {
	sub := s.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.DailyWindowStart = &start
	sub.WeeklyWindowStart = &start
	sub.MonthlyWindowStart = &start
	return nil
}

func (s *singleSubscriptionRepoStub) ResetMonthlyUsage(_ context.Context, id int64, _ *time.Time, start time.Time) error {
	sub := s.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.MonthlyUsageUSD = 0
	sub.MonthlyBonusUSD = 0
	sub.MonthlyWindowStart = &start
	s.monthlyResetCalls++
	return nil
}

func (s *singleSubscriptionRepoStub) GetByIDIncludeDeleted(ctx context.Context, id int64) (*UserSubscription, error) {
	return s.GetByID(ctx, id)
}

func (s *singleSubscriptionRepoStub) Restore(ctx context.Context, id int64, status string) (*UserSubscription, error) {
	sub, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	sub.DeletedAt = nil
	sub.Status = status
	if err := s.Update(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *singleSubscriptionRepoStub) ExtendExpiry(_ context.Context, id int64, expiresAt time.Time) error {
	sub := s.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.ExpiresAt = expiresAt
	return nil
}

func (s *singleSubscriptionRepoStub) UpdateStatus(_ context.Context, id int64, status string) error {
	sub := s.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.Status = status
	return nil
}

func TestAssignSubscriptionRejectsDifferentActivePlan(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	repo.active = &UserSubscription{
		ID: 9, UserID: 42, GroupID: 100, Status: SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	groups := &subscriptionGroupRepoStub{group: &Group{ID: 200, SubscriptionType: SubscriptionTypeSubscription}}
	svc := NewSubscriptionService(groups, repo, nil, nil, nil)

	_, err := svc.AssignSubscription(context.Background(), &AssignSubscriptionInput{UserID: 42, GroupID: 200})

	require.ErrorIs(t, err, ErrActiveSubscriptionExists)
	require.Equal(t, 1, repo.lockCalls)
	require.Zero(t, repo.createCalls)
}

func TestAssignOrExtendKeepsLiveSamePlanTerm(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	expiresAt := time.Now().Add(10 * 24 * time.Hour).Round(time.Second)
	sub := &UserSubscription{
		ID: 3, UserID: 42, GroupID: 200, Status: SubscriptionStatusActive,
		StartsAt: expiresAt.Add(-20 * 24 * time.Hour), ExpiresAt: expiresAt,
	}
	repo.seed(sub)
	repo.active = sub
	groups := &subscriptionGroupRepoStub{group: &Group{ID: 200, SubscriptionType: SubscriptionTypeSubscription}}
	svc := NewSubscriptionService(groups, repo, nil, nil, nil)

	got, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{UserID: 42, GroupID: 200, ValidityDays: 30})

	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, expiresAt, got.ExpiresAt)
}

func TestMonthlyBonusRaisesEffectiveLimitAndRenewalThreshold(t *testing.T) {
	limit := 100.0
	group := &Group{MonthlyLimitUSD: &limit}
	sub := &UserSubscription{
		Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour),
		MonthlyUsageUSD: 119, MonthlyBonusUSD: 20,
	}

	require.Equal(t, 120.0, sub.EffectiveMonthlyLimitUSD(group))
	require.True(t, sub.CheckMonthlyLimit(group, 1))
	require.False(t, sub.CheckMonthlyLimit(group, 1.01))
	require.False(t, sub.IsRenewalEligible(group))
	sub.MonthlyUsageUSD = 120
	require.True(t, sub.IsRenewalEligible(group))
}

func TestFutureSubscriptionIsNotActiveOrRenewable(t *testing.T) {
	limit := 100.0
	sub := &UserSubscription{
		Status:          SubscriptionStatusActive,
		StartsAt:        time.Now().Add(time.Hour),
		ExpiresAt:       time.Now().Add(31 * 24 * time.Hour),
		MonthlyUsageUSD: limit,
	}

	require.False(t, sub.IsActive())
	require.False(t, sub.IsRenewalEligible(&Group{MonthlyLimitUSD: &limit}))
}

func TestAdminSetMonthlyBonusReplacesValueAndRejectsInvalidNumbers(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	now := time.Now()
	repo.seed(&UserSubscription{
		ID: 5, UserID: 42, GroupID: 200, Status: SubscriptionStatusActive,
		ExpiresAt: now.Add(24 * time.Hour), MonthlyWindowStart: &now,
	})
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	got, err := svc.AdminSetMonthlyBonus(context.Background(), 5, 25)
	require.NoError(t, err)
	require.Equal(t, 25.0, got.MonthlyBonusUSD)
	require.Equal(t, 25.0, repo.bonusValue)

	_, err = svc.AdminSetMonthlyBonus(context.Background(), 5, -1)
	require.ErrorIs(t, err, ErrInvalidMonthlyBonus)
	_, err = svc.AdminSetMonthlyBonus(context.Background(), 5, math.NaN())
	require.ErrorIs(t, err, ErrInvalidMonthlyBonus)
}

func TestAdminSetMonthlyBonusAdvancesExpiredMonthlyWindowBeforeSettingValue(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	staleWindow := time.Now().Add(-31 * 24 * time.Hour)
	repo.seed(&UserSubscription{
		ID: 6, UserID: 42, GroupID: 200, Status: SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(24 * time.Hour), MonthlyWindowStart: &staleWindow,
		MonthlyUsageUSD: 80, MonthlyBonusUSD: 10,
	})
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	got, err := svc.AdminSetMonthlyBonus(context.Background(), 6, 25)

	require.NoError(t, err)
	require.Equal(t, 1, repo.monthlyResetCalls)
	require.Zero(t, got.MonthlyUsageUSD)
	require.Equal(t, 25.0, got.MonthlyBonusUSD)
}

func TestAdminSetMonthlyBonusRejectsExpiredSubscription(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	repo.seed(&UserSubscription{
		ID: 7, UserID: 42, GroupID: 200, Status: SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(-time.Minute),
	})
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	_, err := svc.AdminSetMonthlyBonus(context.Background(), 7, 25)

	require.ErrorIs(t, err, ErrSubscriptionExpired)
	require.Zero(t, repo.bonusValue)
}

func TestRenewSubscriptionRejectsOlderPaidPeriod(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	currentStart := time.Now().Add(-time.Hour).Truncate(time.Second)
	repo.seed(&UserSubscription{
		ID: 5, UserID: 42, GroupID: 200,
		Status: SubscriptionStatusActive, StartsAt: currentStart,
		ExpiresAt: currentStart.Add(30 * 24 * time.Hour),
	})
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	_, err := svc.RenewSubscription(context.Background(), 5, 42, 200, currentStart.Add(-time.Minute), 30, "old order")

	require.ErrorIs(t, err, ErrStaleSubscriptionRenewal)
	stored, getErr := repo.GetByID(context.Background(), 5)
	require.NoError(t, getErr)
	require.Equal(t, currentStart, stored.StartsAt)
}

func TestRenewSubscriptionRejectsDifferentActiveSubscription(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	target := &UserSubscription{
		ID: 5, UserID: 42, GroupID: 200,
		Status: SubscriptionStatusExpired, StartsAt: time.Now().Add(-31 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}
	repo.seed(target)
	repo.active = &UserSubscription{
		ID: 6, UserID: 42, GroupID: 300,
		Status: SubscriptionStatusActive, StartsAt: time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(29 * 24 * time.Hour),
	}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	_, err := svc.RenewSubscription(context.Background(), 5, 42, 200, time.Now(), 30, "old order")

	require.ErrorIs(t, err, ErrActiveSubscriptionExists)
	require.Equal(t, 1, repo.lockCalls)
}

func TestCheckBillingEligibilityFallsBackToBalanceWhenSubscriptionQuotaIsExhausted(t *testing.T) {
	limit := 10.0
	cache := &subscriptionFallbackCacheStub{
		balanceEligibilityCacheStub: balanceEligibilityCacheStub{balance: 20},
		subscription: &SubscriptionCacheData{
			Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour), DailyUsage: limit,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	balanceGroup := &Group{ID: 1, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard}
	subscriptionGroup := &Group{ID: 2, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeSubscription, DailyLimitUSD: &limit}
	apiKey := &APIKey{
		User: &User{ID: 42}, GroupID: &subscriptionGroup.ID, Group: subscriptionGroup,
		BalanceGroupID: &balanceGroup.ID, BalanceGroup: balanceGroup,
	}
	subscription := &UserSubscription{ID: 9, UserID: 42, GroupID: subscriptionGroup.ID}

	err := svc.CheckBillingEligibility(context.Background(), apiKey.User, apiKey, subscriptionGroup, subscription, PlatformOpenAI)

	require.NoError(t, err)
	require.Equal(t, balanceGroup.ID, *apiKey.GroupID)
	require.Same(t, balanceGroup, apiKey.Group)
}

func TestCheckBillingEligibilityRejectsExhaustedSubscriptionWithNegativeBalance(t *testing.T) {
	limit := 75.0
	cache := &subscriptionFallbackCacheStub{
		balanceEligibilityCacheStub: balanceEligibilityCacheStub{balance: -3.39},
		subscription: &SubscriptionCacheData{
			Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour), DailyUsage: limit,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	balanceGroup := &Group{ID: 1, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard}
	subscriptionGroup := &Group{ID: 2, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeSubscription, DailyLimitUSD: &limit}
	apiKey := &APIKey{
		User: &User{ID: 42}, GroupID: &subscriptionGroup.ID, Group: subscriptionGroup,
		BalanceGroupID: &balanceGroup.ID, BalanceGroup: balanceGroup,
	}
	subscription := &UserSubscription{ID: 9, UserID: 42, GroupID: subscriptionGroup.ID}

	err := svc.CheckBillingEligibility(context.Background(), apiKey.User, apiKey, subscriptionGroup, subscription, PlatformOpenAI)

	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.Equal(t, subscriptionGroup.ID, *apiKey.GroupID)
	require.Same(t, subscriptionGroup, apiKey.Group)
}

type subscriptionFallbackCacheStub struct {
	balanceEligibilityCacheStub
	subscription *SubscriptionCacheData
}

func (s *subscriptionFallbackCacheStub) GetSubscriptionCache(context.Context, int64, int64) (*SubscriptionCacheData, error) {
	return s.subscription, nil
}

type fallbackQuotaCacheStub struct {
	*subscriptionFallbackCacheStub
	mu         sync.Mutex
	quotaCalls int
}

func (s *fallbackQuotaCacheStub) GetUserPlatformQuotaCache(context.Context, int64, string) (*UserPlatformQuotaCacheEntry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quotaCalls++
	zero := 0.0
	now := time.Now()
	return &UserPlatformQuotaCacheEntry{
		DailyLimitUSD:    &zero,
		DailyWindowStart: &now,
		SchemaVersion:    UserPlatformQuotaCacheSchemaV1,
	}, true, nil
}

func (s *fallbackQuotaCacheStub) quotaCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quotaCalls
}

type fallbackRPMCacheStub struct {
	UserRPMCache
	mu       sync.Mutex
	groupIDs []int64
}

func (s *fallbackRPMCacheStub) IncrementUserGroupRPM(_ context.Context, _ int64, groupID int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groupIDs = append(s.groupIDs, groupID)
	if len(s.groupIDs) == 2 {
		return 2, nil
	}
	return 1, nil
}

func (s *fallbackRPMCacheStub) IncrementUserRPM(context.Context, int64) (int, error) {
	return 1, nil
}

func (s *fallbackRPMCacheStub) calls() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.groupIDs...)
}

func TestBalanceFallbackEligibilityRunsOnceAndEnforcesPlatformQuota(t *testing.T) {
	cache := &fallbackQuotaCacheStub{subscriptionFallbackCacheStub: &subscriptionFallbackCacheStub{
		balanceEligibilityCacheStub: balanceEligibilityCacheStub{balance: 20},
		subscription:                &SubscriptionCacheData{Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour)},
	}}
	cfg := &config.Config{}
	cfg.Billing.UserPlatformQuotaCacheTTLSeconds = 60
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, &fakeQuotaRepo{})
	t.Cleanup(svc.Stop)

	balanceGroup := &Group{ID: 1, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard}
	subscriptionGroup := &Group{ID: 2, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeSubscription}
	apiKey := &APIKey{
		User: &User{ID: 42}, GroupID: &subscriptionGroup.ID, Group: subscriptionGroup,
		BalanceGroupID: &balanceGroup.ID, BalanceGroup: balanceGroup,
	}
	subscription := &UserSubscription{ID: 9, UserID: 42, GroupID: subscriptionGroup.ID}

	require.NoError(t, svc.CheckBillingEligibility(context.Background(), apiKey.User, apiKey, subscriptionGroup, subscription, PlatformOpenAI))
	firstErr := apiKey.validateBalanceFallback(context.Background())
	secondErr := apiKey.validateBalanceFallback(context.Background())

	require.ErrorIs(t, firstErr, ErrUserPlatformDailyQuotaExhausted)
	require.ErrorIs(t, secondErr, ErrUserPlatformDailyQuotaExhausted)
	require.Equal(t, 1, cache.quotaCallCount(), "deferred fallback validation must be idempotent")
	require.Equal(t, subscriptionGroup.ID, *apiKey.GroupID, "failed fallback validation must not switch billing groups")
}

func TestBalanceFallbackEnforcesBalanceGroupRPMWithoutPreconsumingIt(t *testing.T) {
	cache := &subscriptionFallbackCacheStub{
		balanceEligibilityCacheStub: balanceEligibilityCacheStub{balance: 20},
		subscription:                &SubscriptionCacheData{Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour)},
	}
	rpmCache := &fallbackRPMCacheStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, rpmCache, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	balanceGroup := &Group{ID: 1, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RPMLimit: 1}
	subscriptionGroup := &Group{ID: 2, Status: StatusActive, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeSubscription, RPMLimit: 1}
	apiKey := &APIKey{
		User: &User{ID: 42}, GroupID: &subscriptionGroup.ID, Group: subscriptionGroup,
		BalanceGroupID: &balanceGroup.ID, BalanceGroup: balanceGroup,
	}
	subscription := &UserSubscription{ID: 9, UserID: 42, GroupID: subscriptionGroup.ID}

	require.NoError(t, svc.CheckBillingEligibility(context.Background(), apiKey.User, apiKey, subscriptionGroup, subscription, ""))
	require.Equal(t, []int64{subscriptionGroup.ID}, rpmCache.calls(), "subscription routing must not consume balance-group RPM")

	err := apiKey.validateBalanceFallback(context.Background())
	require.True(t, errors.Is(err, ErrGroupRPMExceeded))
	require.Equal(t, []int64{subscriptionGroup.ID, balanceGroup.ID}, rpmCache.calls())
	require.Equal(t, subscriptionGroup.ID, *apiKey.GroupID)
}

func TestRestoreSubscriptionLocksUserAndRejectsAnotherActiveSubscription(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	deletedAt := time.Now().Add(-time.Hour)
	revoked := &UserSubscription{ID: 5, UserID: 42, GroupID: 200, DeletedAt: &deletedAt, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour)}
	repo.seed(revoked)
	repo.active = &UserSubscription{ID: 6, UserID: 42, GroupID: 300, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour)}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	_, err := svc.RestoreSubscription(context.Background(), revoked.ID)

	require.ErrorIs(t, err, ErrSubscriptionRestoreConflict)
	require.Equal(t, 1, repo.lockCalls)
}

func TestExtendExpiredSubscriptionLocksUserAndRejectsAnotherActiveSubscription(t *testing.T) {
	repo := newSingleSubscriptionRepoStub()
	expired := &UserSubscription{ID: 7, UserID: 42, GroupID: 200, Status: SubscriptionStatusExpired, ExpiresAt: time.Now().Add(-24 * time.Hour)}
	repo.seed(expired)
	repo.active = &UserSubscription{ID: 8, UserID: 42, GroupID: 300, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour)}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	_, err := svc.ExtendSubscription(context.Background(), expired.ID, 30)

	require.ErrorIs(t, err, ErrActiveSubscriptionExists)
	require.Equal(t, 1, repo.lockCalls)
}
