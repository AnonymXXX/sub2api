package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestOperatorCannotReachAdminWriteRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserRole), service.RoleOperator)
		c.Next()
	})

	adminGroup := router.Group("/api/v1/admin")
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{}}
	registerDashboardRoutes(adminGroup, handlers)
	registerOpsRoutes(adminGroup, handlers)
	registerUsageRoutes(adminGroup, handlers)

	adminOnly := adminGroup.Group("")
	adminOnly.Use(middleware.AdminOnly())
	registerUserManagementRoutes(adminOnly, handlers)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/admin/dashboard/aggregation/backfill"},
		{http.MethodPost, "/api/v1/admin/ops/alert-rules"},
		{http.MethodPut, "/api/v1/admin/ops/alert-rules/1"},
		{http.MethodDelete, "/api/v1/admin/ops/alert-rules/1"},
		{http.MethodPut, "/api/v1/admin/ops/alert-events/1/status"},
		{http.MethodPost, "/api/v1/admin/ops/alert-silences"},
		{http.MethodPut, "/api/v1/admin/ops/email-notification/config"},
		{http.MethodPut, "/api/v1/admin/ops/runtime/alert"},
		{http.MethodPut, "/api/v1/admin/ops/runtime/logging"},
		{http.MethodPost, "/api/v1/admin/ops/runtime/logging/reset"},
		{http.MethodPut, "/api/v1/admin/ops/advanced-settings"},
		{http.MethodPut, "/api/v1/admin/ops/settings/metric-thresholds"},
		{http.MethodPut, "/api/v1/admin/ops/errors/1/resolve"},
		{http.MethodPut, "/api/v1/admin/ops/request-errors/1/resolve"},
		{http.MethodPut, "/api/v1/admin/ops/upstream-errors/1/resolve"},
		{http.MethodPost, "/api/v1/admin/ops/system-logs/cleanup"},
		{http.MethodGet, "/api/v1/admin/usage/cleanup-tasks"},
		{http.MethodPost, "/api/v1/admin/usage/cleanup-tasks"},
		{http.MethodPost, "/api/v1/admin/usage/cleanup-tasks/1/cancel"},
		{http.MethodGet, "/api/v1/admin/users"},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403; body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}
