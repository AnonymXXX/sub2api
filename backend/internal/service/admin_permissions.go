package service

type AdminPermission string

const (
	AdminPermissionDashboardRead AdminPermission = "admin.dashboard.read"
	AdminPermissionOpsRead       AdminPermission = "admin.ops.read"
	AdminPermissionUsageRead     AdminPermission = "admin.usage.read"
)

func CanAccessAdminPanel(role string) bool {
	return role == RoleAdmin || role == RoleOperator
}

func HasAdminPermission(role string, permission AdminPermission) bool {
	if role == RoleAdmin {
		return true
	}
	if role != RoleOperator {
		return false
	}
	switch permission {
	case AdminPermissionDashboardRead, AdminPermissionOpsRead, AdminPermissionUsageRead:
		return true
	default:
		return false
	}
}
