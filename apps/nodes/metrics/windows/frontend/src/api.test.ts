import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Definitions, Status as GetStatus } from '../wailsjs/go/main/App'
import { definitions, status } from './api'
import {metricState} from './fixtures.test-support'

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
  it('accepts standard false/zero and preserves the exact revision string', async () => {
    vi.mocked(GetStatus).mockResolvedValue(JSON.stringify({version: 1, running: true, metrics: [metricState('volume_muted', false), metricState('volume_percent', 0)], notifications: {queue_depth: 0, dropped: 0}}))
    const result = await status()
    expect(result.metrics[0].state.value).toBe(false)
    expect(result.metrics[1].state.value).toBe(0)
    expect(result.metrics[0].state.revision).toBe('9007199254740993')
  })

  it('rejects old facade payloads and nonstandard variable states without conversion', async () => {
    vi.mocked(GetStatus).mockResolvedValue(JSON.stringify({version: 1, running: true, samples: []}))
    await expect(status()).rejects.toThrow('不支持的 Metrics 状态')
    const metric = metricState('volume_percent', 30)
    for (const state of [{...metric.state, version: 2}, {...metric.state, revision: 9007199254740992}, {...metric.state, status: 'fresh'}, {...metric.state, definition: undefined}]) {
      vi.mocked(GetStatus).mockResolvedValue(JSON.stringify({version: 1, running: true, metrics: [{...metric, state}]}))
      await expect(status()).rejects.toThrow('不支持的标准变量状态')
    }
  })

})
