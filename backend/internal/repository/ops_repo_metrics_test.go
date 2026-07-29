package repository

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryInsertSystemMetrics_PreservesQueueDepthZero(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	createdAt := time.Date(2026, 7, 29, 1, 0, 0, 0, time.UTC)
	queueDepth := 0

	args := make([]driver.Value, 40)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	args[39] = int64(0)

	mock.ExpectExec(`INSERT INTO ops_system_metrics`).
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.InsertSystemMetrics(context.Background(), &service.OpsInsertSystemMetricsInput{
		CreatedAt:             createdAt,
		WindowMinutes:         1,
		ConcurrencyQueueDepth: &queueDepth,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryInsertSystemMetrics_KeepsMissingQueueDepthNull(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	createdAt := time.Date(2026, 7, 29, 1, 1, 0, 0, time.UTC)

	args := make([]driver.Value, 40)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	args[39] = nil

	mock.ExpectExec(`INSERT INTO ops_system_metrics`).
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.InsertSystemMetrics(context.Background(), &service.OpsInsertSystemMetricsInput{
		CreatedAt:     createdAt,
		WindowMinutes: 1,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
