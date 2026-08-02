package subscriptionkey

import (
	"context"
	"database/sql"
	"fmt"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Scan(ctx context.Context) ([]Observation, error) {
	rows, err := s.db.QueryContext(ctx, `
WITH active_subscriptions AS (
    SELECT user_id, COUNT(*) AS active_count, MIN(group_id) AS active_group_id
    FROM user_subscriptions
    WHERE deleted_at IS NULL
      AND status = 'active'
      AND starts_at <= NOW()
      AND expires_at > NOW()
    GROUP BY user_id
)
SELECT
    k.user_id,
    k.id,
    k.key,
    k.group_id,
    COALESCE(old_group.name, ''),
    active.active_group_id,
    COALESCE(new_group.name, ''),
    COALESCE(old_group.platform, ''),
    COALESCE(new_group.platform, ''),
    COALESCE(old_group.subscription_type, ''),
    COALESCE(new_group.subscription_type, ''),
    COALESCE(old_group.status, ''),
    COALESCE(new_group.status, ''),
    active.active_count,
    EXISTS (
        SELECT 1
        FROM user_subscriptions history
        WHERE history.user_id = k.user_id
          AND history.group_id = k.group_id
          AND history.starts_at <= NOW()
    ),
    k.deleted_at IS NOT NULL,
    old_group.deleted_at IS NOT NULL,
    new_group.deleted_at IS NOT NULL
FROM api_keys k
JOIN active_subscriptions active ON active.user_id = k.user_id
LEFT JOIN groups old_group ON old_group.id = k.group_id
LEFT JOIN groups new_group ON new_group.id = active.active_group_id
WHERE k.group_id IS NOT NULL
  AND k.group_id <> active.active_group_id
ORDER BY k.user_id, k.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	observations := make([]Observation, 0)
	for rows.Next() {
		var observation Observation
		if err := rows.Scan(
			&observation.UserID,
			&observation.KeyID,
			&observation.Key,
			&observation.OldGroupID,
			&observation.OldGroupName,
			&observation.NewGroupID,
			&observation.NewGroupName,
			&observation.OldPlatform,
			&observation.NewPlatform,
			&observation.OldSubscriptionType,
			&observation.NewSubscriptionType,
			&observation.OldGroupStatus,
			&observation.NewGroupStatus,
			&observation.ActiveSubscriptionCount,
			&observation.HasHistoricalSubscription,
			&observation.KeyDeleted,
			&observation.OldGroupDeleted,
			&observation.NewGroupDeleted,
		); err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return observations, nil
}

func (s *SQLStore) ApplyUser(ctx context.Context, userID int64, candidates []Candidate) error {
	if len(candidates) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return fmt.Errorf("lock user: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
SELECT
    subscription.group_id,
    target_group.platform,
    target_group.subscription_type,
    target_group.status,
    target_group.deleted_at IS NOT NULL
FROM user_subscriptions subscription
JOIN groups target_group ON target_group.id = subscription.group_id
WHERE subscription.user_id = $1
  AND subscription.deleted_at IS NULL
  AND subscription.status = 'active'
  AND subscription.starts_at <= NOW()
  AND subscription.expires_at > NOW()
FOR UPDATE OF subscription`, userID)
	if err != nil {
		return fmt.Errorf("reload active subscription: %w", err)
	}
	var targetGroupID int64
	var targetPlatform, targetSubscriptionType, targetStatus string
	var targetDeleted bool
	activeCount := 0
	for rows.Next() {
		activeCount++
		if err := rows.Scan(&targetGroupID, &targetPlatform, &targetSubscriptionType, &targetStatus, &targetDeleted); err != nil {
			_ = rows.Close()
			return err
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if activeCount != 1 || targetDeleted || targetStatus != "active" || targetSubscriptionType != "subscription" {
		return fmt.Errorf("active subscription changed during repair")
	}

	for index := range candidates {
		candidate := &candidates[index]
		if candidate.NewGroupID != targetGroupID {
			return fmt.Errorf("target group changed for key %d", candidate.KeyID)
		}
		var oldGroupID int64
		var key string
		var keyDeleted, oldGroupDeleted, hasHistory bool
		var oldPlatform, oldSubscriptionType, oldGroupStatus string
		err := tx.QueryRowContext(ctx, `
SELECT
    key.group_id,
    key.key,
    key.deleted_at IS NOT NULL,
    COALESCE(old_group.platform, ''),
    COALESCE(old_group.subscription_type, ''),
    COALESCE(old_group.status, ''),
    old_group.deleted_at IS NOT NULL,
    EXISTS (
        SELECT 1 FROM user_subscriptions history
        WHERE history.user_id = key.user_id
          AND history.group_id = key.group_id
          AND history.starts_at <= NOW()
    )
FROM api_keys key
LEFT JOIN groups old_group ON old_group.id = key.group_id
WHERE key.id = $1 AND key.user_id = $2
FOR UPDATE OF key`, candidate.KeyID, userID).Scan(
			&oldGroupID,
			&key,
			&keyDeleted,
			&oldPlatform,
			&oldSubscriptionType,
			&oldGroupStatus,
			&oldGroupDeleted,
			&hasHistory,
		)
		if err != nil {
			return fmt.Errorf("reload key %d: %w", candidate.KeyID, err)
		}
		if oldGroupID != candidate.OldGroupID || keyDeleted || oldGroupDeleted || oldSubscriptionType != "subscription" || oldPlatform != targetPlatform || !hasHistory {
			return fmt.Errorf("candidate key %d changed during repair", candidate.KeyID)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE api_keys
SET group_id = $1, updated_at = NOW()
WHERE id = $2 AND user_id = $3 AND group_id = $4 AND deleted_at IS NULL`, targetGroupID, candidate.KeyID, userID, candidate.OldGroupID)
		if err != nil {
			return fmt.Errorf("update key %d: %w", candidate.KeyID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return fmt.Errorf("update key %d affected %d rows: %w", candidate.KeyID, affected, err)
		}
		candidate.Key = key
	}
	return tx.Commit()
}
