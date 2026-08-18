import axios from 'axios'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  clearBootstrapToken,
  getSetupStatus,
  install,
  setBootstrapToken,
  testDatabase,
  testRedis
} from '@/api/setup'

const { client } = vi.hoisted(() => ({
  client: {
    get: vi.fn(),
    post: vi.fn()
  }
}))

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => client)
  }
}))

describe('setup bootstrap token handling', () => {
  beforeEach(() => {
    clearBootstrapToken()
    client.get.mockResolvedValue({ data: { data: { needs_setup: true, step: 'database' } } })
    client.post.mockResolvedValue({ data: { data: { message: 'ok', restart: true } } })
    localStorage.clear()
    sessionStorage.clear()
    vi.clearAllMocks()
  })

  it('adds the in-memory token only to setup POSTs', async () => {
    setBootstrapToken('  bootstrap-secret  ')

    await getSetupStatus()
    await testDatabase({ host: 'localhost', port: 5432, user: 'postgres', password: '', dbname: 'sub2api', sslmode: 'disable' })
    await testRedis({ host: 'localhost', port: 6379, username: '', password: '', db: 0, enable_tls: false })
    await install({
      database: { host: 'localhost', port: 5432, user: 'postgres', password: '', dbname: 'sub2api', sslmode: 'disable' },
      redis: { host: 'localhost', port: 6379, username: '', password: '', db: 0, enable_tls: false },
      admin: { email: 'admin@example.com', password: 'password' },
      server: { host: '127.0.0.1', port: 3000, mode: 'release' }
    })

    expect(client.get).toHaveBeenCalledWith('/setup/status')
    expect(client.post).toHaveBeenNthCalledWith(1, '/setup/test-db', expect.anything(), {
      headers: { 'X-Bootstrap-Token': 'bootstrap-secret' }
    })
    expect(client.post).toHaveBeenNthCalledWith(2, '/setup/test-redis', expect.anything(), {
      headers: { 'X-Bootstrap-Token': 'bootstrap-secret' }
    })
    expect(client.post).toHaveBeenNthCalledWith(3, '/setup/install', expect.anything(), {
      headers: { 'X-Bootstrap-Token': 'bootstrap-secret' }
    })
    expect(JSON.stringify(localStorage)).not.toContain('bootstrap-secret')
    expect(JSON.stringify(sessionStorage)).not.toContain('bootstrap-secret')
  })
})
