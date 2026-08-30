import { Sample } from './contracts'

export type Theme = 'light' | 'dark'

const metricLabels: Record<string, string> = {
  battery_percent: '电池',
  battery_charging: '充电状态',
  battery_on_ac: '外接电源',
  network_online: '网络连接',
  network_type: '网络类型',
  cpu_percent: '处理器',
  memory_percent: '内存',
  volume_percent: '系统音量',
  volume_muted: '静音',
  brightness_percent: '屏幕亮度',
}

const metricDescriptions: Record<string, string> = {
  battery_percent: '设备当前电池电量。',
  battery_charging: '设备是否正在充电。',
  battery_on_ac: '设备是否连接外部电源。',
  network_online: '系统当前是否具备网络连接。',
  network_type: '系统当前报告的网络类型。',
  cpu_percent: '系统处理器的即时使用率。',
  memory_percent: '系统内存的即时使用率。',
  volume_percent: '系统主音量百分比。',
  volume_muted: '系统主音量是否静音。',
  brightness_percent: '系统当前屏幕亮度百分比。',
}

export function normalizeChannels(value: string): string[] {
  return [...new Set(value.split(/[,\n]/).map(item => item.trim()).filter(Boolean))].sort()
}

export function metricLabel(metric: string): string {
  return metricLabels[metric] ?? metric.replaceAll('_', ' ')
}

export function metricDescription(metric: string): string {
  return metricDescriptions[metric] ?? '由 Metrics 节点报告的设备资源。'
}

export function sampleValue(sample: Sample): string {
  return sample.status === 'unavailable' ? '—' : (sample.value ?? '—')
}

export interface SampleReading {
  value: string
  suffix: string
  progress?: number
}

function booleanReading(metric: string, value: string): string {
  if (value !== 'true' && value !== 'false') return value
  const enabled = value === 'true'
  if (metric === 'network_online') return enabled ? '在线' : '离线'
  if (metric === 'battery_charging') return enabled ? '充电中' : '未充电'
  if (metric === 'battery_on_ac') return enabled ? '是' : '否'
  if (metric === 'volume_muted') return enabled ? '开启' : '关闭'
  return enabled ? '是' : '否'
}

export function sampleReading(sample: Sample): SampleReading {
  const value = sampleValue(sample)
  if (value === '—') return {value, suffix: ''}
  if (sample.unit === 'boolean') return {value: booleanReading(sample.metric, value), suffix: ''}
  if (sample.unit === 'percent') {
    const parsed = Number(value)
    return {
      value,
      suffix: '%',
      ...(Number.isFinite(parsed) ? {progress: Math.min(100, Math.max(0, parsed))} : {}),
    }
  }
  return {value, suffix: ''}
}

export function sampleStatusLabel(status: Sample['status']): string {
  if (status === 'fresh') return '新鲜'
  if (status === 'stale') return '陈旧'
  return '不可用'
}

export function reconcileSelectedMetric(samples: Sample[], selectedMetric?: string): string | undefined {
  if (selectedMetric && samples.some(sample => sample.metric === selectedMetric)) return selectedMetric
  return samples[0]?.metric
}

export function formatInterval(intervalMS?: number): string {
  if (!intervalMS || intervalMS <= 0) return '—'
  if (intervalMS % 1000 === 0) return `${intervalMS / 1000} s`
  return `${intervalMS} ms`
}

export function formatTimestamp(unixMS?: number): string {
  if (!unixMS || unixMS <= 0) return '—'
  const value = new Date(unixMS)
  if (Number.isNaN(value.getTime())) return '—'
  const part = (number: number, length = 2) => String(number).padStart(length, '0')
  return `${part(value.getHours())}:${part(value.getMinutes())}:${part(value.getSeconds())}.${part(value.getMilliseconds(), 3)}`
}

export function connectionStateLabel(state: string | undefined, running: boolean): string {
  if (!running) return '已停止'
  if (!state) return '启动中'
  const labels: Record<string, string> = {
    connected: '已连接',
    connecting: '连接中',
    reconnecting: '重连中',
    backoff: '等待重试',
    disconnected: '未连接',
    stopped: '已停止',
  }
  return labels[state] ?? state
}

export function connectionStateTone(state: string | undefined, running: boolean): 'connected' | 'connecting' | 'stopped' {
  if (!running) return 'stopped'
  if (state === 'connected') return 'connected'
  return 'connecting'
}

export function parseNodeID(value: string, label: string): string {
  const normalized = value.trim()
  if (normalized.length > 20 || !/^[1-9]\d*$/.test(normalized)) throw new Error(`${label}必须是规范的非零十进制整数`)
  try {
    if (BigInt(normalized) > 18_446_744_073_709_551_615n) throw new Error()
  } catch {
    throw new Error(`${label}超出 uint64 范围`)
  }
  return normalized
}

export function requireText(value: string, label: string): string {
  const normalized = value.trim()
  if (!normalized) throw new Error(`请填写${label}`)
  return normalized
}

export function parsePermit(value: string): Record<string, unknown> | undefined {
  const normalized = value.trim()
  if (!normalized) return undefined
  if (new TextEncoder().encode(normalized).byteLength > 240 * 1024) throw new Error('一次性接入凭证过大')
  let parsed: unknown
  try {
    parsed = JSON.parse(normalized)
  } catch {
    throw new Error('一次性接入凭证必须是有效 JSON 对象')
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('一次性接入凭证必须是 JSON 对象')
  }
  return parsed as Record<string, unknown>
}

export function parseInterval(value: string, metric: string): number {
  const interval = Number(value)
  if (!Number.isInteger(interval) || interval < 250 || interval > 86_400_000) {
    throw new Error(`${metricLabel(metric)}的采样间隔必须是 250 到 86400000 毫秒之间的整数`)
  }
  return interval
}
