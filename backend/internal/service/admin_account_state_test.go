//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type accountRepoStubForStateUpdate struct {
	accountRepoStub
	account        *Account
	updatedAccount *Account
	setSchedulable *bool
}

func (s *accountRepoStubForStateUpdate) GetByID(context.Context, int64) (*Account, error) {
	if s.updatedAccount != nil {
		return s.updatedAccount, nil
	}
	return s.account, nil
}

func (s *accountRepoStubForStateUpdate) Update(_ context.Context, account *Account) error {
	copy := *account
	s.updatedAccount = &copy
	return nil
}

func (s *accountRepoStubForStateUpdate) SetSchedulable(_ context.Context, _ int64, schedulable bool) error {
	s.setSchedulable = &schedulable
	return nil
}

func (s *accountRepoStubForStateUpdate) BulkUpdate(_ context.Context, _ []int64, updates AccountBulkUpdate) (int64, error) {
	copy := *s.account
	if updates.Status != nil {
		copy.Status = *updates.Status
	}
	if updates.Schedulable != nil {
		copy.Schedulable = *updates.Schedulable
	}
	s.updatedAccount = &copy
	return 1, nil
}

func TestAdminServiceUpdateAccount_NormalizesManualStatus(t *testing.T) {
	tests := []struct {
		name            string
		status          string
		wantStatus      string
		wantSchedulable bool
	}{
		{name: "legacy inactive", status: "inactive", wantStatus: StatusDisabled, wantSchedulable: false},
		{name: "disabled", status: StatusDisabled, wantStatus: StatusDisabled, wantSchedulable: false},
		{name: "error", status: StatusError, wantStatus: StatusError, wantSchedulable: false},
		{name: "active", status: StatusActive, wantStatus: StatusActive, wantSchedulable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &accountRepoStubForStateUpdate{account: &Account{
				ID: 1, Status: StatusActive, Schedulable: true,
			}}
			svc := &adminServiceImpl{accountRepo: repo}

			updated, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{Status: tt.status})

			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, updated.Status)
			require.Equal(t, tt.wantSchedulable, updated.Schedulable)
		})
	}
}

func TestAdminServiceSetAccountSchedulable_UpdatesStatusAndSwitchTogether(t *testing.T) {
	tests := []struct {
		name        string
		schedulable bool
		wantStatus  string
	}{
		{name: "turn off", schedulable: false, wantStatus: StatusDisabled},
		{name: "turn on", schedulable: true, wantStatus: StatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &accountRepoStubForStateUpdate{account: &Account{
				ID: 1, Status: StatusActive, Schedulable: true,
			}}
			svc := &adminServiceImpl{accountRepo: repo}

			updated, err := svc.SetAccountSchedulable(context.Background(), 1, tt.schedulable)

			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, updated.Status)
			require.Equal(t, tt.schedulable, updated.Schedulable)
			require.NotNil(t, repo.updatedAccount)
		})
	}
}
