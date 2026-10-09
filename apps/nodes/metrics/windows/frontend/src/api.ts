import type { Configuration, Definition, IdentityResult, StartRequest, Status } from './contracts'
import { Configuration as GetConfiguration, Definitions, Identity, Start, Status as GetStatus, Stop, UpdateConfiguration } from '../wailsjs/go/main/App'

export interface MetricsAPI {
  identity(stateDirectory: string, nodeID: string): Promise<IdentityResult>
  start(request: StartRequest): Promise<Status>
  stop(): Promise<void>
  status(): Promise<Status>
  configuration(): Promise<Configuration>
  definitions(): Promise<Definition[]>
  updateConfiguration(configuration: Configuration): Promise<Configuration>
}

function decodeJSON<T>(payload: string, label: string): T {
  try {
    return JSON.parse(payload) as T
  } catch {
    throw new Error(`${label} 返回了无效 JSON`)
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function decodeStatus(payload: string, label: string): Status {
  const result = decodeJSON<unknown>(payload, label)
  if (!isRecord(result) || result.version !== 1 || typeof result.running !== 'boolean' || !Array.isArray(result.metrics)) {
    throw new Error(`${label} 返回了不支持的 Metrics 状态`)
  }
  for (const metric of result.metrics) {
    const state = isRecord(metric) ? metric.state : undefined
    if (!isRecord(metric) || typeof metric.metric !== 'string' || !metric.metric || !isRecord(state) ||
      state.version !== 1 || typeof state.instance_id !== 'string' || !/^[0-9a-f]{32}$/.test(state.instance_id) ||
      typeof state.revision !== 'string' || !/^[1-9][0-9]{0,19}$/.test(state.revision) || BigInt(state.revision) > 18_446_744_073_709_551_615n ||
      !isRecord(state.definition) || typeof state.definition.type !== 'string' ||
      !['ready', 'stale', 'unavailable'].includes(String(state.status)) ||
      (state.status === 'unavailable' ? 'value' in state : !('value' in state)) ||
      !Number.isSafeInteger(state.observed_at_unix_ms)) {
      throw new Error(`${label} 返回了不支持的标准变量状态`)
    }
  }
  // The local facade returns snapshots already validated by the shared runtime.
  return result as unknown as Status
}

function normalizeDefinition(value: unknown, index: number): Definition {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Definitions[${index}] 不是对象`)
  }
  const record = value as Record<string, unknown>
  const name = record.Name
  const unit = record.Unit
  const controllable = record.Controllable
  const interval = record.IntervalMS
  if (typeof name !== 'string' || !name || typeof unit !== 'string' || !unit || typeof controllable !== 'boolean' || typeof interval !== 'number' || !Number.isInteger(interval) || interval <= 0) {
    throw new Error(`Definitions[${index}] 字段无效`)
  }
  return {name, unit, controllable, interval_ms: interval}
}

export async function identity(stateDirectory: string, nodeID: string): Promise<IdentityResult> {
  return decodeJSON(await Identity(JSON.stringify({version: 1, state_directory: stateDirectory, node_id: nodeID})), 'Identity')
}

export async function start(request: StartRequest): Promise<Status> {
  return decodeStatus(await Start(JSON.stringify({version: 1, ...request})), 'Start')
}

export async function stop(): Promise<void> { await Stop() }
export async function status(): Promise<Status> { return decodeStatus(await GetStatus(), 'Status') }
export async function configuration(): Promise<Configuration> { return decodeJSON(await GetConfiguration(), 'Configuration') }
export async function definitions(): Promise<Definition[]> {
  const values = decodeJSON<unknown>(await Definitions(), 'Definitions')
  if (!Array.isArray(values)) throw new Error('Definitions 返回值不是数组')
  return values.map(normalizeDefinition)
}

export async function updateConfiguration(configuration: Configuration): Promise<Configuration> {
  return decodeJSON(await UpdateConfiguration(JSON.stringify({
    version: 1,
    expected_revision: configuration.revision,
    settings: configuration.settings,
    notification_channels: configuration.notification_channels,
  })), 'UpdateConfiguration')
}

export const wailsMetricsAPI: MetricsAPI = {
  identity,
  start,
  stop,
  status,
  configuration,
  definitions,
  updateConfiguration,
}
