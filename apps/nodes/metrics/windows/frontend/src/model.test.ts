import { describe, expect, it } from 'vitest'
import {
  connectionStateLabel,
  formatInterval,
  metricLabel,
  normalizeChannels,
  parseInterval,
  parseNodeID,
  parsePermit,
  reconcileSelectedMetric,
  sampleReading,
  sampleValue,
} from './model'

describe('metrics presentation model', () => {
  it('normalizes notification channels deterministically', () => {
    expect(normalizeChannels(' system,alerts\nsystem,  ')).toEqual(['alerts', 'system'])
  })

  it('keeps unavailable and missing values explicit', () => {
    const base = {version: 1, metric: 'cpu_usage', unit: '%', observed_at_unix_ms: 1}
    expect(metricLabel('cpu_percent')).toBe('处理器')
    expect(metricLabel(base.metric)).toBe('cpu usage')
    expect(sampleValue({...base, status: 'unavailable', value: '91'})).toBe('—')
    expect(sampleValue({...base, status: 'stale'})).toBe('—')
    expect(sampleValue({...base, status: 'fresh', value: '17'})).toBe('17')
  })

  it('formats boolean and percent readings without inventing unavailable values', () => {
    expect(sampleReading({version: 1, metric: 'network_online', unit: 'boolean', status: 'fresh', value: 'true', observed_at_unix_ms: 1})).toEqual({value: '在线', suffix: ''})
    expect(sampleReading({version: 1, metric: 'cpu_percent', unit: 'percent', status: 'fresh', value: '117', observed_at_unix_ms: 1})).toEqual({value: '117', suffix: '%', progress: 100})
    expect(sampleReading({version: 1, metric: 'cpu_percent', unit: 'percent', status: 'unavailable', observed_at_unix_ms: 1})).toEqual({value: '—', suffix: ''})
  })

  it('reconciles selection and presents timing and connection state', () => {
    const samples = [
      {version: 1, metric: 'cpu_percent', unit: 'percent', status: 'fresh' as const, value: '17', observed_at_unix_ms: 1},
      {version: 1, metric: 'memory_percent', unit: 'percent', status: 'fresh' as const, value: '52', observed_at_unix_ms: 1},
    ]
    expect(reconcileSelectedMetric(samples, 'memory_percent')).toBe('memory_percent')
    expect(reconcileSelectedMetric(samples, 'missing')).toBe('cpu_percent')
    expect(reconcileSelectedMetric([], 'memory_percent')).toBeUndefined()
    expect(formatInterval(2000)).toBe('2 s')
    expect(formatInterval(750)).toBe('750 ms')
    expect(connectionStateLabel('backoff', true)).toBe('等待重试')
    expect(connectionStateLabel('connected', false)).toBe('已停止')
  })

  it('validates connection and policy inputs before crossing the API boundary', () => {
    expect(parseNodeID('20', '节点 ID')).toBe('20')
    expect(() => parseNodeID('020', '节点 ID')).toThrow('规范的非零十进制整数')
    expect(() => parseNodeID('100000000000000000000', '节点 ID')).toThrow('规范的非零十进制整数')
    expect(parsePermit('{"version":1}')).toEqual({version: 1})
    expect(parsePermit('')).toBeUndefined()
    expect(() => parsePermit('[]')).toThrow('JSON 对象')
    expect(parseInterval('250', 'cpu_percent')).toBe(250)
    expect(() => parseInterval('249', 'cpu_percent')).toThrow('250 到 86400000')
  })
})
