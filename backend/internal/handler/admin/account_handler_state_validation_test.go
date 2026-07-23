//go:build unit

package admin

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/require"
)

func TestAccountStateRequestsAcceptCanonicalDisabledStatus(t *testing.T) {
	tests := []struct {
		name    string
		request any
	}{
		{name: "single update", request: &UpdateAccountRequest{Status: "disabled"}},
		{name: "bulk update", request: &BulkUpdateAccountsRequest{Status: "disabled"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, binding.Validator.ValidateStruct(tt.request))
		})
	}
}
