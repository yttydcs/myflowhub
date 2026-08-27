import * as App from '../wailsjs/go/main/App'

export type Settings = {
  version: number
  profile: string
  node_id: string
  endpoint?: string
  parent_node_id?: string
  parent_public_key?: string
  permit_json?: string
  updated_at_unix_ms: number
}

export const api = {
  settings: async (): Promise<Settings> => JSON.parse(await App.SettingsJSON()),
  saveSettings: async (value: Settings): Promise<Settings> => JSON.parse(await App.SaveSettingsJSON(JSON.stringify(value))),
  identity: async () => JSON.parse(await App.IdentityJSON()),
  connect: App.Connect,
  disconnect: App.Disconnect,
  status: async () => JSON.parse(await App.StatusJSON()),
  catalog: async (owner: number) => JSON.parse(await App.CatalogJSON(owner)),
  snapshot: async (owner: number, name: string) => JSON.parse(await App.SnapshotJSON(owner, name)),
  invoke: async (owner: number, name: string, request: unknown) => JSON.parse(await App.InvokeJSON(owner, name, JSON.stringify(request))),
  subscribe: App.Subscribe,
  poll: async (id: number, timeout: number) => JSON.parse(await App.PollSubscription(id, timeout)),
  cancel: App.CancelSubscription,
  topology: async (owner: number) => JSON.parse(await App.TopologyJSON(owner)),
  health: async (owner: number) => JSON.parse(await App.HealthJSON(owner)),
  managementConfig: async (owner: number) => JSON.parse(await App.ManagementConfigJSON(owner)),
  issuePermit: async (owner: number, request: unknown) => JSON.parse(await App.IssuePermitJSON(owner, JSON.stringify(request))),
  revokeNode: async (owner: number, request: unknown) => JSON.parse(await App.RevokeNodeJSON(owner, JSON.stringify(request))),
  fileTransfers: async (owner: number) => JSON.parse(await App.FileTransfersJSON(owner)),
  flowDefinitions: async (owner: number) => JSON.parse(await App.FlowDefinitionsJSON(owner)),
  flowRuns: async (owner: number) => JSON.parse(await App.FlowRunsJSON(owner)),
  uploadFile: async (owner: number, source: string, destination: string, contentType: string) => JSON.parse(await App.UploadFile(owner, source, destination, contentType)),
  logs: async () => JSON.parse(await App.LogsJSON()),
}
