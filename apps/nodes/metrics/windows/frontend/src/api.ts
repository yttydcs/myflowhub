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
  return decodeJSON(await Start(JSON.stringify({version: 1, ...request})), 'Start')
}

export async function stop(): Promise<void> { await Stop() }
export async function status(): Promise<Status> { return decodeJSON(await GetStatus(), 'Status') }
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
