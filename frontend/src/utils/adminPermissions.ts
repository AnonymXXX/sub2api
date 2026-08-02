import type { AdminPermission, UserRole } from '@/types'

const operatorPermissions = new Set<AdminPermission>([
  'admin.dashboard.read',
  'admin.ops.read',
  'admin.usage.read'
])

export function canAccessAdminPanel(role?: UserRole | null): boolean {
  return role === 'admin' || role === 'operator'
}

export function hasAdminPermission(role: UserRole | null | undefined, permission: AdminPermission): boolean {
  if (role === 'admin') return true
  return role === 'operator' && operatorPermissions.has(permission)
}

export function defaultHomePath(role?: UserRole | null): string {
  return canAccessAdminPanel(role) ? '/admin/dashboard' : '/dashboard'
}
