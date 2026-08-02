package subscriptionkey

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrExpectedCountRequired = errors.New("--expected-count is required with --apply")
	ErrExpectedCountMismatch = errors.New("candidate count does not match --expected-count")
)

type Observation struct {
	UserID                    int64
	KeyID                     int64
	Key                       string
	OldGroupID                int64
	OldGroupName              string
	NewGroupID                int64
	NewGroupName              string
	OldPlatform               string
	NewPlatform               string
	OldSubscriptionType       string
	NewSubscriptionType       string
	OldGroupStatus            string
	NewGroupStatus            string
	ActiveSubscriptionCount   int
	HasHistoricalSubscription bool
	KeyDeleted                bool
	OldGroupDeleted           bool
	NewGroupDeleted           bool
}

type Candidate struct {
	UserID       int64  `json:"user_id"`
	KeyID        int64  `json:"key_id"`
	Key          string `json:"-"`
	OldGroupID   int64  `json:"old_group_id"`
	OldGroupName string `json:"old_group_name"`
	NewGroupID   int64  `json:"new_group_id"`
	NewGroupName string `json:"new_group_name"`
}

type Entry struct {
	Candidate
	Result       string `json:"result"`
	Reason       string `json:"reason,omitempty"`
	Error        string `json:"error,omitempty"`
	CacheWarning string `json:"cache_warning,omitempty"`
}

type Report struct {
	Mode           string  `json:"mode"`
	CandidateCount int     `json:"candidate_count"`
	SkippedCount   int     `json:"skipped_count"`
	AppliedCount   int     `json:"applied_count"`
	FailedCount    int     `json:"failed_count"`
	Entries        []Entry `json:"entries"`
}

type Options struct {
	Apply         bool
	ExpectedCount *int
}

type Store interface {
	Scan(ctx context.Context) ([]Observation, error)
	ApplyUser(ctx context.Context, userID int64, candidates []Candidate) error
}

type CacheInvalidator interface {
	InvalidateKey(ctx context.Context, key string) error
}

type Runner struct {
	store Store
	cache CacheInvalidator
}

func NewRunner(store Store, cache CacheInvalidator) *Runner {
	return &Runner{store: store, cache: cache}
}

func (r *Runner) Run(ctx context.Context, options Options) (*Report, error) {
	observations, err := r.store.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan subscription key candidates: %w", err)
	}
	report := &Report{Mode: "dry-run", Entries: make([]Entry, 0, len(observations))}
	candidates := make([]Candidate, 0, len(observations))
	for _, observation := range observations {
		candidate := candidateFromObservation(observation)
		reason := skipReason(observation)
		if reason != "" {
			report.SkippedCount++
			report.Entries = append(report.Entries, Entry{Candidate: candidate, Result: "skipped", Reason: reason})
			continue
		}
		report.CandidateCount++
		candidates = append(candidates, candidate)
		report.Entries = append(report.Entries, Entry{Candidate: candidate, Result: "candidate"})
	}

	if !options.Apply {
		return report, nil
	}
	if options.ExpectedCount == nil {
		return report, ErrExpectedCountRequired
	}
	if *options.ExpectedCount != report.CandidateCount {
		return report, fmt.Errorf("%w: expected=%d actual=%d", ErrExpectedCountMismatch, *options.ExpectedCount, report.CandidateCount)
	}
	report.Mode = "apply"

	byUser := make(map[int64][]Candidate)
	userOrder := make([]int64, 0)
	for _, candidate := range candidates {
		if _, exists := byUser[candidate.UserID]; !exists {
			userOrder = append(userOrder, candidate.UserID)
		}
		byUser[candidate.UserID] = append(byUser[candidate.UserID], candidate)
	}

	entryByKeyID := make(map[int64]*Entry, len(report.Entries))
	for i := range report.Entries {
		entryByKeyID[report.Entries[i].KeyID] = &report.Entries[i]
	}
	var applyErrors []error
	for _, userID := range userOrder {
		userCandidates := byUser[userID]
		if applyErr := r.store.ApplyUser(ctx, userID, userCandidates); applyErr != nil {
			report.FailedCount += len(userCandidates)
			applyErrors = append(applyErrors, fmt.Errorf("repair user %d: %w", userID, applyErr))
			for _, candidate := range userCandidates {
				entry := entryByKeyID[candidate.KeyID]
				entry.Result = "failed"
				entry.Error = applyErr.Error()
			}
			continue
		}
		for _, candidate := range userCandidates {
			entry := entryByKeyID[candidate.KeyID]
			entry.Result = "repaired"
			report.AppliedCount++
			if r.cache != nil {
				if cacheErr := r.cache.InvalidateKey(ctx, candidate.Key); cacheErr != nil {
					entry.CacheWarning = cacheErr.Error()
				}
			}
		}
	}
	return report, errors.Join(applyErrors...)
}

func candidateFromObservation(observation Observation) Candidate {
	return Candidate{
		UserID: observation.UserID, KeyID: observation.KeyID, Key: observation.Key,
		OldGroupID: observation.OldGroupID, OldGroupName: observation.OldGroupName,
		NewGroupID: observation.NewGroupID, NewGroupName: observation.NewGroupName,
	}
}

func skipReason(observation Observation) string {
	switch {
	case observation.KeyDeleted:
		return "soft_deleted_key"
	case observation.ActiveSubscriptionCount != 1:
		return "active_subscription_count"
	case observation.OldGroupDeleted:
		return "source_group_deleted"
	case observation.NewGroupDeleted || observation.NewGroupStatus != "active":
		return "target_group_inactive"
	case observation.OldSubscriptionType != "subscription":
		return "standard_group"
	case observation.NewSubscriptionType != "subscription":
		return "target_group_not_subscription"
	case observation.OldPlatform == "" || observation.NewPlatform == "" || observation.OldPlatform != observation.NewPlatform:
		return "cross_platform"
	case !observation.HasHistoricalSubscription:
		return "ambiguous_source"
	default:
		return ""
	}
}
