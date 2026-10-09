import { describe, expect, it } from 'vitest'
import {metricState} from './fixtures.test-support'
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
    expect(metricLabel('cpu_percent')).toBe('处理器')
    expect(sampleValue(metricState('cpu_percent', undefined, 'unavailable'))).toBe('—')
    expect(sampleValue(metricState('cpu_percent', 0, 'stale'))).toBe('0')
    expect(sampleValue(metricState('volume_muted', false))).toBe('false')
    expect(sampleValue(metricState('network_type', ''))).toBe('')
  })

  it('formats real booleans and percentages without inventing unavailable values', () => {
    expect(sampleReading(metricState('network_online', true))).toEqual({value: '在线', suffix: ''})
    expect(sampleReading(metricState('network_online', false))).toEqual({value: '离线', suffix: ''})
    expect(sampleReading(metricState('cpu_percent', 0))).toEqual({value: '0', suffix: '%', progress: 0})
    expect(sampleReading(metricState('cpu_percent', undefined, 'unavailable'))).toEqual({value: '—', suffix: ''})
  })

  it('reconciles selection and presents timing and connection state', () => {
    const states = [metricState('cpu_percent', 17), metricState('memory_percent', 52)]
    expect(reconcileSelectedMetric(states, 'memory_percent')).toBe('memory_percent')
    expect(reconcileSelectedMetric(states, 'missing')).toBe('cpu_percent')
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
