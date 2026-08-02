package service

import "testing"

func TestAdminPermissionMatrix(t *testing.T) {
	permissions := []AdminPermission{
		AdminPermissionDashboardRead,
		AdminPermissionOpsRead,
		AdminPermissionUsageRead,
	}

	for _, permission := range permissions {
		if !HasAdminPermission(RoleAdmin, permission) {
			t.Fatalf("admin must have %q", permission)
		}
		if !HasAdminPermission(RoleOperator, permission) {
			t.Fatalf("operator must have %q", permission)
		}
		if HasAdminPermission(RoleUser, permission) {
			t.Fatalf("user must not have %q", permission)
		}
	}

	unknown := AdminPermission("admin.settings.write")
	if !HasAdminPermission(RoleAdmin, unknown) {
		t.Fatal("admin must retain full admin permissions")
	}
	if HasAdminPermission(RoleOperator, unknown) {
		t.Fatal("operator must not receive permissions outside the read-only allowlist")
	}
}

func TestCanAccessAdminPanel(t *testing.T) {
	tests := map[string]bool{
		RoleAdmin:    true,
		RoleOperator: true,
		RoleUser:     false,
		"":           false,
	}
	for role, want := range tests {
		if got := CanAccessAdminPanel(role); got != want {
			t.Fatalf("CanAccessAdminPanel(%q) = %v, want %v", role, got, want)
		}
	}
}
