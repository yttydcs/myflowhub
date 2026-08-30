import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Definitions, Status as GetStatus } from '../wailsjs/go/main/App'
import { definitions, status } from './api'

vi.mock('../wailsjs/go/main/App', () => ({
  Configuration: vi.fn(),
  Definitions: vi.fn(),
  Identity: vi.fn(),
  Start: vi.fn(),
  Status: vi.fn(),
  Stop: vi.fn(),
  UpdateConfiguration: vi.fn(),
}))

describe('Wails Metrics API adapter', () => {
  beforeEach(() => vi.clearAllMocks())

  it('normalizes Go definition fields at the frontend boundary', async () => {
    vi.mocked(Definitions).mockResolvedValue(JSON.stringify([{Name: 'cpu_percent', Unit: 'percent', Controllable: false, IntervalMS: 2000, Platforms: {windows: true}}]))
    await expect(definitions()).resolves.toEqual([{name: 'cpu_percent', unit: 'percent', controllable: false, interval_ms: 2000}])
  })

  it('fails explicitly for malformed definition and status payloads', async () => {
    vi.mocked(Definitions).mockResolvedValue(JSON.stringify([{Name: 'cpu_percent'}]))
    await expect(definitions()).rejects.toThrow('Definitions[0] 字段无效')
    vi.mocked(GetStatus).mockResolvedValue('{')
    await expect(status()).rejects.toThrow('Status 返回了无效 JSON')
  })
})
