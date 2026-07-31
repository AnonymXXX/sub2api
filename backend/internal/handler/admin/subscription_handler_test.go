//go:build unit

package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionHandlerResetQuotaUsesAdminIdempotency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(storeUnavailableRepoStub{}, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	handler := NewSubscriptionHandler(nil)
	router := gin.New()
	router.POST("/api/v1/admin/subscriptions/:id/reset-quota", handler.ResetQuota)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/subscriptions/42/reset-quota", bytes.NewBufferString(`{"daily":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "reset-subscription-42-daily")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
