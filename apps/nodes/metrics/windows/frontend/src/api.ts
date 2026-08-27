import { Configuration, Status } from './contracts'
import { Configuration as GetConfiguration, Definitions, Identity, Start, Status as GetStatus, Stop, UpdateConfiguration } from '../wailsjs/go/main/App'

export async function identity(stateDirectory: string, nodeID: string): Promise<{node_id: string; public_key: string}> {
  return JSON.parse(await Identity(JSON.stringify({version: 1, state_directory: stateDirectory, node_id: nodeID})))
}

export async function start(request: Record<string, unknown>): Promise<Status> {
  return JSON.parse(await Start(JSON.stringify({version: 1, ...request})))
}

export async function stop(): Promise<void> { await Stop() }
export async function status(): Promise<Status> { return JSON.parse(await GetStatus()) }
export async function configuration(): Promise<Configuration> { return JSON.parse(await GetConfiguration()) }
export async function definitions(): Promise<unknown[]> { return JSON.parse(await Definitions()) }

export async function updateConfiguration(configuration: Configuration): Promise<Configuration> {
  return JSON.parse(await UpdateConfiguration(JSON.stringify({
    version: 1,
    expected_revision: configuration.revision,
    settings: configuration.settings,
    notification_channels: configuration.notification_channels,
  })))
}
