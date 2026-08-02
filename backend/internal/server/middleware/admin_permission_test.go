package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestRequireAdminPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		role       string
		setRole    bool
		permission service.AdminPermission
		want       int
	}{
		{name: "admin", role: service.RoleAdmin, setRole: true, permission: service.AdminPermissionDashboardRead, want: http.StatusOK},
		{name: "operator allowed", role: service.RoleOperator, setRole: true, permission: service.AdminPermissionOpsRead, want: http.StatusOK},
		{name: "operator denied unknown", role: service.RoleOperator, setRole: true, permission: service.AdminPermission("admin.settings.write"), want: http.StatusForbidden},
		{name: "user denied", role: service.RoleUser, setRole: true, permission: service.AdminPermissionUsageRead, want: http.StatusForbidden},
		{name: "missing role", permission: service.AdminPermissionUsageRead, want: http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			if tc.setRole {
				router.Use(func(c *gin.Context) {
					c.Set(string(ContextKeyUserRole), tc.role)
					c.Next()
				})
			}
			router.Use(RequireAdminPermission(tc.permission))
			router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}
}
