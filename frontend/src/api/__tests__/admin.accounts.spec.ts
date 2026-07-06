import { describe, it, expect, vi, beforeEach } from 'vitest'

const post = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    post
  }
}))

describe('admin accounts API', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('导入数据时透传账号默认容量、优先级和分组', async () => {
    post.mockResolvedValue({
      data: {
        proxy_created: 0,
        proxy_reused: 0,
        proxy_failed: 0,
        account_created: 0,
        account_updated: 1,
        account_failed: 0
      }
    })

    const accountsAPI = (await import('@/api/admin/accounts')).default
    const payload = {
      data: {
        exported_at: '2026-07-06T00:00:00Z',
        proxies: [],
        accounts: []
      },
      skip_default_group_bind: true,
      concurrency: 5,
      priority: 1,
      group_ids: [22]
    }

    await accountsAPI.importData(payload)

    expect(post).toHaveBeenCalledWith('/admin/accounts/data', payload)
  })
})
