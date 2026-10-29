import { createRoot } from 'react-dom/client'
import { App } from '../../src/App'
import type { DesktopAPI, PollResult } from '../../src/api'
import type { ResourceDescriptor } from '../../src/types'
import type { DataSchemaDefinition, VariableValueV1 } from '../../src/generated/variable-contract.generated'
import '../../src/shadcn.css'
import '../../src/style.css'

// Isolated visual/interaction fixture. TCP, policy and CAS evidence comes from
// Go integration tests; this preview never claims a real device connection.
const kinds = ['bool', 'number', 'enum', 'text', 'object', 'array']
const definitions: DataSchemaDefinition[] = [
  { type: 'boolean' }, { type: 'number', minimum: 0, maximum: 100 },
  { type: 'string', enum: ['busy', 'normal', 'quiet'], read_only: true },
  { type: 'string', max_length: 128, read_only: true },
  { type: 'object', read_only: true, properties: [{ name: 'sample', required: true, schema: { type: 'integer' } }] },
  { type: 'array', read_only: true, items: { type: 'number' }, max_items: 8 },
]
const resources: ResourceDescriptor[] = Array.from({ length: 2000 }, (_, i) => ({
  id: { owner_node_id: '3', name: `example/foundation/${kinds[i % 6]}/${String(i).padStart(4, '0')}` }, type: 'mfh.variable', type_version: 2,
  capabilities: [
    { name: 'read', permission: 'example.read', output_schema: 'mfh.variable-value.v1', max_payload_bytes: 4096 },
    { name: 'subscribe', permission: 'example.read', event_schema: 'mfh.variable-value.v1', max_payload_bytes: 4096 },
    ...(i % 6 < 2 ? [{ name: 'write', permission: 'example.write', input_schema: 'mfh.variable-write.v3', output_schema: 'mfh.variable-value.v1', max_payload_bytes: 4096 }] : []),
  ], limits: { max_payload_bytes: 4096 }, presentation: { label: `模拟负载 ${kinds[i % 6]} ${String(i).padStart(4, '0')}` },
}))
const states = new Map(resources.map((resource, i) => [resource.id.name, {
  version: 1, instance_id: (i + 1).toString(16).padStart(32, '0'), revision: '1', definition: definitions[i % 6],
  status: 'ready', observed_at_unix_ms: Date.now(), value: [false, 20, 'normal', '模拟文本', { sample: i }, [i % 100, 20]][i % 6],
} as VariableValueV1]))
let sequence = 0
type Handle = { name: string; queue: PollResult[]; waiting?: (value: PollResult) => void }
const handles = new Map<number, Handle>()
const push = (handle: Handle, kind: 'snapshot' | 'data') => {
  const state = { ...states.get(handle.name)! }
  if (kind === 'data') delete state.definition
  const result: PollResult = { kind: 'event', payload: { kind, owner_node_id: '3', resource_name: handle.name, capability: 'subscribe', schema: 'mfh.variable-value.v1', revision: state.revision, value: state } }
  if (handle.waiting) { const deliver = handle.waiting; handle.waiting = undefined; deliver(result) }
  else { handle.queue.length = 0; handle.queue.push(result) }
}
setInterval(() => {
  for (const [name, state] of states) states.set(name, { ...state, revision: String(BigInt(state.revision) + 1n), observed_at_unix_ms: Date.now() })
  for (const handle of handles.values()) push(handle, 'data')
}, 1000)
const profile = { id: 'phase-a-preview', name: '界面测试 · 模拟数据', node_id: '2', parent_node_id: '1', parent_public_key: 'preview', endpoint: 'preview-only', auto_connect: true }
const api: DesktopAPI = {
  settings: async () => ({ version: 3, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }),
  profileStates: async () => [{ profile_id: profile.id, state: 'legacy', node_id: '2' }],
  identity: async () => ({ node_id: '2', public_key: 'preview' }),
  status: async () => ({ state: 'connected', node_id: '2', parent_node_id: '1', generation: 1, link_generation: 1 }),
  topology: async () => ({ version: 1, root_node_id: '1', depth: 1, instance_id: 'f'.repeat(32), revision: 1, nodes: [
    { node_id: '1', role: 'root', generation: 1, has_children: true }, { node_id: '3', parent_id: '1', role: 'child', generation: 1, has_children: false },
  ] }),
  catalog: async owner => ({ version: 2, revision: 1, resources: owner === '3' ? resources : [] }),
  snapshot: async (_, name) => { const state = states.get(name); if (!state) throw new Error('not_found'); return { ...state } },
  subscribe: async (_, name) => { const id = ++sequence; const handle: Handle = { name, queue: [] }; handles.set(id, handle); push(handle, 'snapshot'); return id },
  poll: async id => { const handle = handles.get(id); if (!handle) return { kind: 'closed' }; return handle.queue.shift() || new Promise<PollResult>(resolve => { handle.waiting = resolve }) },
  cancel: async id => { const handle = handles.get(id); handle?.waiting?.({ kind: 'closed' }); handles.delete(id) },
  operate: async (_, name, capability, schema, raw) => {
    const request = raw as { expected_instance_id: string; expected_revision: string; value: VariableValueV1['value'] }
    const current = states.get(name)!
    if (capability !== 'write' || schema !== 'mfh.variable-write.v3') throw new Error('unsupported')
    if (request.expected_instance_id !== current.instance_id || request.expected_revision !== current.revision) throw new Error('conflict')
    const next = { ...current, value: request.value, revision: String(BigInt(current.revision) + 1n), observed_at_unix_ms: Date.now() }
    states.set(name, next)
    return { schema: 'mfh.variable-value.v1', payload: next }
  },
  views: async () => ({ version: 3, views: [] }), saveView: async view => ({ ...view, revision: view.revision + 1 }), deleteView: async () => {},
  prepareProfile: async () => { throw new Error('preview only') }, saveProfile: async () => profile, login: async () => profile,
  switchProfile: async () => {}, deactivateProfile: async () => {}, deleteProfile: async () => {}, connect: async () => {}, disconnect: async () => {},
  pickFile: async () => '', uploadFile: async () => { throw new Error('preview only') },
}
Object.assign(window, { phaseAPreview: { handles, states } })
createRoot(document.getElementById('app')!).render(<App api={api} />)
