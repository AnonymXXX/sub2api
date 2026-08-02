import { describe, expect, it } from 'vitest'

import { routes } from '@/router'

function routeMeta(path: string) {
  const route = routes.find((item) => item.path === path)
  expect(route, `missing route ${path}`).toBeDefined()
  return route!.meta
}

describe('operator admin routes', () => {
  it('uses permission meta only for dashboard, ops, and usage', () => {
    expect(routeMeta('/admin/dashboard')?.requiresAdminPermission).toBe('admin.dashboard.read')
    expect(routeMeta('/admin/ops')?.requiresAdminPermission).toBe('admin.ops.read')
    expect(routeMeta('/admin/usage')?.requiresAdminPermission).toBe('admin.usage.read')
  })

  it('keeps user management admin-only', () => {
    expect(routeMeta('/admin/users')?.requiresAdmin).toBe(true)
    expect(routeMeta('/admin/users')?.requiresAdminPermission).toBeUndefined()
  })
})
