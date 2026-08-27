import { Sample } from './contracts'

export function normalizeChannels(value: string): string[] {
  return [...new Set(value.split(/[,\n]/).map(item => item.trim()).filter(Boolean))].sort()
}

export function metricLabel(metric: string): string {
  return metric.replaceAll('_', ' ')
}

export function sampleValue(sample: Sample): string {
  return sample.status === 'unavailable' ? '—' : (sample.value ?? '—')
}
