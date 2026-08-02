package subscriptionkey

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type repairStoreStub struct {
	observations []Observation
	applyCalls   int
}

func (s *repairStoreStub) Scan(context.Context) ([]Observation, error) {
	return append([]Observation(nil), s.observations...), nil
}

func (s *repairStoreStub) ApplyUser(_ context.Context, _ int64, candidates []Candidate) error {
	s.applyCalls++
	ids := make(map[int64]struct{}, len(candidates))
	for _, candidate := range candidates {
		ids[candidate.KeyID] = struct{}{}
	}
	remaining := s.observations[:0]
	for _, observation := range s.observations {
		if _, repaired := ids[observation.KeyID]; !repaired {
			remaining = append(remaining, observation)
		}
	}
	s.observations = remaining
	return nil
}

type repairCacheStub struct {
	keys []string
}

func (s *repairCacheStub) InvalidateKey(_ context.Context, key string) error {
	s.keys = append(s.keys, key)
	return nil
}

func repairObservations() []Observation {
	base := Observation{
		UserID: 15, KeyID: 12, Key: "sk-candidate", OldGroupID: 108, OldGroupName: "OpenAI Pro",
		NewGroupID: 10, NewGroupName: "OpenAI Lite", OldPlatform: "openai", NewPlatform: "openai",
		OldSubscriptionType: "subscription", NewSubscriptionType: "subscription",
		OldGroupStatus: "active", NewGroupStatus: "active",
		ActiveSubscriptionCount: 1, HasHistoricalSubscription: true,
	}
	standard := base
	standard.KeyID, standard.Key, standard.OldSubscriptionType = 13, "sk-standard", "standard"
	crossPlatform := base
	crossPlatform.KeyID, crossPlatform.Key, crossPlatform.OldPlatform = 14, "sk-cross", "anthropic"
	ambiguous := base
	ambiguous.KeyID, ambiguous.Key, ambiguous.HasHistoricalSubscription = 15, "sk-ambiguous", false
	deleted := base
	deleted.KeyID, deleted.Key, deleted.KeyDeleted = 16, "sk-deleted", true
	invalidTarget := base
	invalidTarget.KeyID, invalidTarget.Key, invalidTarget.NewSubscriptionType = 17, "sk-invalid-target", "standard"
	return []Observation{base, standard, crossPlatform, ambiguous, deleted, invalidTarget}
}

func TestRunnerDryRunFiltersCandidatesWithoutWriting(t *testing.T) {
	store := &repairStoreStub{observations: repairObservations()}
	runner := NewRunner(store, &repairCacheStub{})

	report, err := runner.Run(context.Background(), Options{})
	require.NoError(t, err)
	require.Equal(t, 1, report.CandidateCount)
	require.Equal(t, 5, report.SkippedCount)
	require.Zero(t, report.AppliedCount)
	require.Zero(t, store.applyCalls)
	require.Equal(t, "candidate", report.Entries[0].Result)
	require.Equal(t, "standard_group", report.Entries[1].Reason)
	require.Equal(t, "cross_platform", report.Entries[2].Reason)
	require.Equal(t, "ambiguous_source", report.Entries[3].Reason)
	require.Equal(t, "soft_deleted_key", report.Entries[4].Reason)
	require.Equal(t, "target_group_not_subscription", report.Entries[5].Reason)
}

func TestRunnerApplyRequiresMatchingExpectedCount(t *testing.T) {
	store := &repairStoreStub{observations: repairObservations()}
	runner := NewRunner(store, &repairCacheStub{})

	_, err := runner.Run(context.Background(), Options{Apply: true})
	require.ErrorIs(t, err, ErrExpectedCountRequired)
	require.Zero(t, store.applyCalls)

	wrong := 2
	_, err = runner.Run(context.Background(), Options{Apply: true, ExpectedCount: &wrong})
	require.ErrorIs(t, err, ErrExpectedCountMismatch)
	require.Zero(t, store.applyCalls)
}

func TestRunnerApplyIsIdempotentAcrossRepeatedScans(t *testing.T) {
	store := &repairStoreStub{observations: repairObservations()[:1]}
	cache := &repairCacheStub{}
	runner := NewRunner(store, cache)
	expected := 1

	report, err := runner.Run(context.Background(), Options{Apply: true, ExpectedCount: &expected})
	require.NoError(t, err)
	require.Equal(t, 1, report.AppliedCount)
	require.Equal(t, 1, store.applyCalls)
	require.Equal(t, []string{"sk-candidate"}, cache.keys)

	report, err = runner.Run(context.Background(), Options{})
	require.NoError(t, err)
	require.Zero(t, report.CandidateCount)
	require.Equal(t, 1, store.applyCalls, "repeated dry-run must not perform another update")
}
