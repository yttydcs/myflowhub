import { describe, expect, it } from 'vitest'
import { metricLabel, normalizeChannels, sampleValue } from './model'

describe('metrics presentation model', () => {
  it('normalizes notification channels deterministically', () => {
    expect(normalizeChannels(' system,alerts\nsystem,  ')).toEqual(['alerts', 'system'])
  })

  it('keeps unavailable and missing values explicit', () => {
    const base = {version: 1, metric: 'cpu_usage', unit: '%', observed_at_unix_ms: 1}
    expect(metricLabel(base.metric)).toBe('cpu usage')
    expect(sampleValue({...base, status: 'unavailable', value: '91'})).toBe('—')
    expect(sampleValue({...base, status: 'stale'})).toBe('—')
    expect(sampleValue({...base, status: 'fresh', value: '17'})).toBe('17')
  })
})
