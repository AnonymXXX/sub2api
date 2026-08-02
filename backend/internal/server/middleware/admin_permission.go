package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func RequireAdminPermission(permission service.AdminPermission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok {
			AbortWithError(c, 401, "UNAUTHORIZED", "User not found in context")
			return
		}
		if !service.HasAdminPermission(role, permission) {
			AbortWithError(c, 403, "FORBIDDEN", "Admin permission required")
			return
		}
		c.Next()
	}
}
