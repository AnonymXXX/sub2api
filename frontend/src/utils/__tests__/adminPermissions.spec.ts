import { describe, expect, it } from 'vitest'

import {
  canAccessAdminPanel,
  defaultHomePath,
  hasAdminPermission
} from '@/utils/adminPermissions'

describe('adminPermissions', () => {
  it('grants the operator exactly the three read permissions', () => {
    expect(hasAdminPermission('operator', 'admin.dashboard.read')).toBe(true)
    expect(hasAdminPermission('operator', 'admin.ops.read')).toBe(true)
    expect(hasAdminPermission('operator', 'admin.usage.read')).toBe(true)
    expect(hasAdminPermission('user', 'admin.dashboard.read')).toBe(false)
  })

  it('treats admin and operator as panel roles with the admin dashboard home', () => {
    expect(canAccessAdminPanel('admin')).toBe(true)
    expect(canAccessAdminPanel('operator')).toBe(true)
    expect(canAccessAdminPanel('user')).toBe(false)
    expect(defaultHomePath('operator')).toBe('/admin/dashboard')
    expect(defaultHomePath('user')).toBe('/dashboard')
  })
})
