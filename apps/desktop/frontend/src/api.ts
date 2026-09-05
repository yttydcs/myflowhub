import * as App from '../wailsjs/go/main/App'
import type {
  ConnectionStatus,
  Profile,
  ProfileState,
  ResourceCatalog,
  Settings,
  TopologyQuery,
  ViewDefinition,
  ViewDocument,
} from './types'
import { parseTopology, TOPOLOGY_QUERY_SCHEMA, validateTopologyRequest } from './discovery/topology'

export type OperationResult = { schema?: string; payload: unknown }
export type PollResult = { kind: 'event' | 'error' | 'closed' | 'timeout'; payload?: unknown }
export type PublicIdentity = { node_id?: string; public_key: string }
export type PreparedProfile = { profile: Profile; identity: PublicIdentity }

export interface DesktopAPI {
  settings(): Promise<Settings>
  profileStates(): Promise<ProfileState[]>
  prepareProfile(profile: Profile): Promise<PreparedProfile>
  saveProfile(profile: Profile): Promise<Profile>
  login(profile: Profile, permitJSON: string, allowTOFU: boolean): Promise<Profile>
  switchProfile(profileID: string): Promise<void>
  deactivateProfile(): Promise<void>
  deleteProfile(profileID: string, confirmation: string): Promise<void>
  identity(): Promise<PublicIdentity>
  connect(): Promise<void>
  disconnect(): Promise<void>
  status(): Promise<ConnectionStatus>
  catalog(ownerNodeID: string): Promise<ResourceCatalog>
  topology(ownerNodeID: string, depth?: number): Promise<TopologyQuery>
  snapshot(ownerNodeID: string, name: string): Promise<unknown>
  operate(ownerNodeID: string, name: string, capability: string, schema: string, input: unknown): Promise<OperationResult>
  subscribe(ownerNodeID: string, name: string, capability: string, leaseMS?: number): Promise<number>
  poll(subscriptionID: number, timeoutMS: number): Promise<PollResult>
  cancel(subscriptionID: number): Promise<void>
  pickFile(): Promise<string>
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
  profileStates: async () => JSON.parse(await App.ProfileStatesJSON()),
  prepareProfile: async (profile) => JSON.parse(await App.PrepareProfileJSON(JSON.stringify(profile))),
  saveProfile: async (profile) => JSON.parse(await App.SaveProfileJSON(JSON.stringify(profile))),
  login: async (profile, permitJSON, allowTOFU) => JSON.parse(await App.LoginJSON(JSON.stringify({ profile, permit_json: permitJSON, allow_tofu: allowTOFU }))),
  switchProfile: App.SwitchProfile,
  deactivateProfile: App.DeactivateProfile,
  deleteProfile: App.DeleteProfile,
  identity: async () => JSON.parse(await App.IdentityJSON()),
  connect: App.Connect,
  disconnect: App.Disconnect,
  status: async () => JSON.parse(await App.StatusJSON()),
  catalog: async (ownerNodeID) => JSON.parse(await App.CatalogJSON(ownerNodeID)),
  topology: async (ownerNodeID, depth = 1) => {
    validateTopologyRequest(ownerNodeID, depth)
    const capability = depth === 1 ? 'children' : 'subtree'
    const schema = depth === 1 ? 'mfh.management.topology-children-request.v1' : 'mfh.management.topology-query-request.v1'
    const result = parseOperation(await App.OperateJSON(ownerNodeID, 'system/topology', capability, schema, JSON.stringify({ version: 1, depth })))
    if (result.schema !== TOPOLOGY_QUERY_SCHEMA) throw new Error('topology 响应 schema 不匹配')
    return parseTopology(result.payload, ownerNodeID, depth)
  },
  snapshot: async (ownerNodeID, name) => JSON.parse(await App.SnapshotJSON(ownerNodeID, name)),
  operate: async (ownerNodeID, name, capability, schema, input) =>
    parseOperation(await App.OperateJSON(ownerNodeID, name, capability, schema, JSON.stringify(input))),
  subscribe: (ownerNodeID, name, capability, leaseMS = 60_000) =>
    App.SubscribeCapability(ownerNodeID, name, capability, leaseMS),
  poll: async (subscriptionID, timeoutMS) => JSON.parse(await App.PollSubscription(subscriptionID, timeoutMS)),
  cancel: async (subscriptionID) => App.CancelSubscription(subscriptionID),
  pickFile: App.SelectUploadFile,
  uploadFile: async (ownerNodeID, source, destination, contentType) =>
    JSON.parse(await App.UploadFile(ownerNodeID, source, destination, contentType)),
  views: async () => JSON.parse(await App.ViewsJSON()),
  saveView: async (view) => JSON.parse(await App.SaveViewJSON(JSON.stringify(view))),
  deleteView: App.DeleteView,
}
