export type Profile = {
  id: string
  name: string
  node_id: string
  endpoint: string
  parent_node_id: string
  parent_public_key: string
  auto_connect: boolean
  created_at_unix_ms?: number
  updated_at_unix_ms?: number
}

export type Settings = {
  version: number
  active_profile_id?: string
  profiles: Profile[]
  updated_at_unix_ms: number
  credential_mode: string
}

export type ConnectionStatus = {
  state: 'signed_out' | 'disconnected' | 'connecting' | 'connected' | 'failed' | 'stopped'
  parent_node_id?: string
  endpoint?: string
  last_error?: string
}

export type CapabilityDescriptor = {
  name: string
  permission: string
  input_schema?: string
  output_schema?: string
  event_schema?: string
  max_payload_bytes: number
}

export type ResourceDescriptor = {
  id: { owner_node_id: string; name: string }
  type: string
  type_version: number
  capabilities: CapabilityDescriptor[]
  schemas?: Array<{ id: string; content_type: string }>
  limits: {
    max_payload_bytes: number
    max_subscribers?: number
    max_sessions?: number
    max_queue?: number
  }
  presentation?: {
    renderer?: string
    label?: string
    description?: string
    icon?: string
  }
}

export type ResourceCatalog = {
  version: number
  revision: number
  resources: ResourceDescriptor[]
}

export type TopologyNode = {
  node_id: string
  parent_id?: string
  display_name?: string
  role: string
  generation: number
}

export type Topology = { version: number; epoch: number; nodes: TopologyNode[] }

export type WorkspaceSelection =
  | { kind: 'node'; node: TopologyNode }
  | { kind: 'resource'; resource: ResourceDescriptor }
  | null

export type ViewWidget = {
  id: string
  owner_node_id: string
  resource_name: string
  renderer: string
  settings?: unknown
}

export type ViewLayoutAxis = 'horizontal' | 'vertical'

export type ViewLayoutNode =
  | { kind: 'leaf'; widget_id: string }
  | { kind: 'split'; axis: ViewLayoutAxis; children: ViewLayoutNode[]; weights: number[] }

export type ViewDefinition = {
  id: string
  name: string
  revision: number
  widgets: ViewWidget[]
  layout_root?: ViewLayoutNode
  created_at_unix_ms?: number
  updated_at_unix_ms?: number
}

export type ViewDocument = { version: number; views: ViewDefinition[] }

export type ResourceEvent = {
  kind: 'snapshot' | 'data' | 'gap' | 'expired'
  owner_node_id: string
  resource_name: string
  capability: string
  schema?: string
  revision?: number
  sequence?: number
  publisher_node_id?: string
  publisher_sequence?: number
  gap_from?: number
  gap_to?: number
  value?: string
  reason?: string
}
