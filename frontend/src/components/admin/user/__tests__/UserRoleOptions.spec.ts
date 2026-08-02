import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

describe('admin user role options', () => {
  it.each(['UserCreateModal.vue', 'UserEditModal.vue'])(
    '%s includes the operator role',
    (file) => {
      const source = readFileSync(resolve(here, '..', file), 'utf8')
      expect(source).toContain('<option value="operator">')
      expect(source).toContain("admin.users.roles.operator")
    }
  )
})
