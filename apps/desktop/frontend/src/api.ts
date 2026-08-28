import * as App from '../wailsjs/go/main/App'
import type {
  ConnectionStatus,
  Profile,
  ResourceCatalog,
  Settings,
  Topology,
  ViewDefinition,
  ViewDocument,
} from './types'

export type OperationResult = { schema?: string; payload: unknown }
export type PollResult = { kind: 'event' | 'error' | 'closed' | 'timeout'; payload?: unknown }

export interface DesktopAPI {
  settings(): Promise<Settings>
  saveProfile(profile: Profile): Promise<Profile>
  login(profile: Profile, permitJSON: string): Promise<Profile>
  switchProfile(profileID: string): Promise<void>
  deleteProfile(profileID: string, confirmation: string): Promise<void>
  identity(): Promise<{ node_id: string; public_key: string }>
  connect(): Promise<void>
  disconnect(): Promise<void>
  status(): Promise<ConnectionStatus>
  catalog(ownerNodeID: string): Promise<ResourceCatalog>
  topology(ownerNodeID: string): Promise<Topology>
  snapshot(ownerNodeID: string, name: string): Promise<unknown>
  operate(ownerNodeID: string, name: string, capability: string, schema: string, input: unknown): Promise<OperationResult>
  subscribe(ownerNodeID: string, name: string, capability: string, leaseMS?: number): Promise<number>
  poll(subscriptionID: number, timeoutMS: number): Promise<PollResult>
  cancel(subscriptionID: number): Promise<void>
  uploadFile(ownerNodeID: string, source: string, destination: string, contentType: string): Promise<unknown>
  views(): Promise<ViewDocument>
  saveView(view: ViewDefinition): Promise<ViewDefinition>
  deleteView(viewID: string, revision: number): Promise<void>
}

function parseOperation(raw: string): OperationResult {
  const value = JSON.parse(raw) as { schema?: string; payload: unknown }
  return { schema: value.schema, payload: value.payload }
}

export const api: DesktopAPI = {
  settings: async () => JSON.parse(await App.SettingsJSON()),
  saveProfile: async (profile) => JSON.parse(await App.SaveProfileJSON(JSON.stringify(profile))),
  login: async (profile, permitJSON) => JSON.parse(await App.LoginJSON(JSON.stringify({ profile, permit_json: permitJSON }))),
  switchProfile: App.SwitchProfile,
  deleteProfile: App.DeleteProfile,
  identity: async () => JSON.parse(await App.IdentityJSON()),
  connect: App.Connect,
  disconnect: App.Disconnect,
  status: async () => JSON.parse(await App.StatusJSON()),
  catalog: async (ownerNodeID) => JSON.parse(await App.CatalogJSON(ownerNodeID)),
  topology: async (ownerNodeID) => JSON.parse(await App.TopologyJSON(ownerNodeID)),
  snapshot: async (ownerNodeID, name) => JSON.parse(await App.SnapshotJSON(ownerNodeID, name)),
  operate: async (ownerNodeID, name, capability, schema, input) =>
    parseOperation(await App.OperateJSON(ownerNodeID, name, capability, schema, JSON.stringify(input))),
  subscribe: (ownerNodeID, name, capability, leaseMS = 60_000) =>
    App.SubscribeCapability(ownerNodeID, name, capability, leaseMS),
  poll: async (subscriptionID, timeoutMS) => JSON.parse(await App.PollSubscription(subscriptionID, timeoutMS)),
  cancel: async (subscriptionID) => App.CancelSubscription(subscriptionID),
  uploadFile: async (ownerNodeID, source, destination, contentType) =>
    JSON.parse(await App.UploadFile(ownerNodeID, source, destination, contentType)),
  views: async () => JSON.parse(await App.ViewsJSON()),
  saveView: async (view) => JSON.parse(await App.SaveViewJSON(JSON.stringify(view))),
  deleteView: App.DeleteView,
}
