package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetAllGroupUsageSummarySeparatesGroupCostFromSubscriptionQuota(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	todayStart := time.Date(2026, 8, 2, 0, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	mock.ExpectQuery(`(?s)WITH usage_summary AS .*subscription_summary AS .*FROM user_subscriptions`).
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{
			"group_id", "total_cost", "today_cost", "active_subscription_count",
			"max_daily_usage", "daily_limit_reached_count",
		}).AddRow(int64(10), 289.80, 76.76, int64(9), 44.83, int64(0)))

	got, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)

	require.NoError(t, err)
	require.Equal(t, int64(10), got[0].GroupID)
	require.InDelta(t, 76.76, got[0].TodayCost, 0.000001)
	require.InDelta(t, 289.80, got[0].TotalCost, 0.000001)
	require.Equal(t, int64(9), got[0].ActiveSubscriptionCount)
	require.InDelta(t, 44.83, got[0].MaxDailyUsage, 0.000001)
	require.Equal(t, int64(0), got[0].DailyLimitReachedCount)
	require.NoError(t, mock.ExpectationsWereMet())
}
