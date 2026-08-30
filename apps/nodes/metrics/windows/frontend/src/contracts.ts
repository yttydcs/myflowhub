export interface Sample {
  version: number
  metric: string
  value?: string
  unit: string
  status: 'fresh' | 'stale' | 'unavailable'
  sampled_at_unix_ms?: number
  observed_at_unix_ms: number
  error?: string
}

export interface Definition {
  name: string
  unit: string
  controllable: boolean
  interval_ms: number
}

export interface IdentityResult {
  node_id: string
  public_key: string
}

export interface StartRequest {
  state_directory: string
  node_id: string
  parent_node_id: string
  endpoint: string
  parent_public_key: string
  permit?: Record<string, unknown>
}

export interface Setting {
  metric: string
  enabled: boolean
  writable: boolean
  interval_ms: number
}

export interface Configuration {
  version: number
  revision: number
  platform: 'windows'
  settings: Setting[]
  notification_channels: string[]
}

export interface Connection {
  state: string
  parent_node_id: string
  endpoint: string
  attempt: number
  link_generation: number
  last_error?: string
  next_retry?: string
}

export interface Status {
  version: number
  running: boolean
  connection?: Connection
  configuration?: Configuration
  samples: Sample[]
  notifications: {queue_depth: number; dropped: number; last_error?: string}
  presenter_error?: string
}
