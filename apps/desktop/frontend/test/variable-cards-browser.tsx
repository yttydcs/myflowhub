// Browser-only QA entry. Uses an in-memory API; never connects or writes to a device.
import { createRoot } from 'react-dom/client'
import { App } from '../src/App'
import type { DesktopAPI } from '../src/api'
import type { ResourceDescriptor, ViewDefinition } from '../src/types'
import { addWorkspaceWidget } from '../src/workspace-layout'
import '../src/shadcn.css'
import '../src/style.css'

const profile = { id: 'cards-qa', name: '卡片交互验收 · 模拟数据', node_id: '2', endpoint: 'localhost:9540', parent_node_id: '1', parent_public_key: 'test', auto_connect: false }
const names = ['cpu_percent', 'memory_percent', 'battery_percent', 'battery_charging', 'battery_on_ac', 'brightness_percent', 'volume_percent', 'volume_muted']
const resources: ResourceDescriptor[] = names.map(name => ({
  id: { owner_node_id: '1', name: `metrics/${name}` }, type: 'mfh.variable', type_version: 2,
  capabilities: [{ name: 'read', permission: 'metrics.read', output_schema: 'mfh.variable-value.v1', max_payload_bytes: 4096 }, ...(name.startsWith('volume') || name.startsWith('brightness') ? [{ name: 'write', permission: 'metrics.write', input_schema: 'mfh.variable-write.v3', output_schema: 'mfh.variable-value.v1', max_payload_bytes: 4096 }] : [])],
  limits: { max_payload_bytes: 4096 }, presentation: { label: name, renderer: 'mfh.variable.auto.v2' },
}))
let initial: ViewDefinition = { id: 'notebook', name: '笔记本', revision: 1, layout_mode: 'compact', widgets: [] }
for (const name of names.slice(0, 2)) initial = addWorkspaceWidget(initial, { id: name, owner_node_id: '1', resource_name: `metrics/${name}`, renderer: 'mfh.variable.auto.v2' })
let saved = [initial]
const values: Record<string, number | boolean> = { cpu_percent: 21, memory_percent: 65, battery_percent: 100, battery_charging: false, battery_on_ac: true, brightness_percent: 100, volume_percent: 50, volume_muted: false }
const revisions: Record<string, number> = {}
const writes: unknown[] = []
const qa = { writes, snapshots: 0, failSave: false, saved: () => saved }
Object.assign(window, { __cardsQA: qa })
const snapshot = (name: string) => {
  const key = name.split('/').at(-1)!
  const value = values[key]!
  return { version: 1, instance_id: 'a'.repeat(32), revision: String(revisions[key] || 1), definition: typeof value === 'boolean' ? { type: 'boolean' } : { type: 'number', minimum: 0, maximum: 100, unit: '%' }, value, status: 'ready', observed_at_unix_ms: (revisions[key] || 1) * 100 }
}
const api = {
  settings: async () => ({ version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }),
  profileStates: async () => [{ profile_id: profile.id, state: 'legacy' }],
  status: async () => ({ state: 'connected', parent_node_id: '1', generation: 1 }),
  topology: async () => ({ version: 1, root_node_id: '1', depth: 1, instance_id: 'a'.repeat(32), revision: 1, nodes: [{ node_id: '1', role: 'root', display_name: '笔记本', generation: 1, has_children: false }] }),
  catalog: async () => ({ version: 2, revision: 1, resources }),
  snapshot: async (_owner: string, name: string) => { qa.snapshots++; return snapshot(name) },
  operate: async (_owner: string, name: string, capability: string, _schema: string, input: { value: number | boolean }) => { writes.push({ name, capability, input }); values[name.split('/').at(-1)!] = input.value; revisions[name.split('/').at(-1)!] = (revisions[name.split('/').at(-1)!] || 1) + 1; return { schema: 'mfh.variable-value.v1', payload: snapshot(name) } },
  views: async () => ({ version: 3, views: saved }),
  saveView: async (view: ViewDefinition) => {
    if (qa.failSave) throw new Error('模拟保存失败')
    const next = { ...view, revision: view.revision + 1 }
    saved = [...saved.filter(item => item.id !== view.id), next]
    return next
  },
  deleteView: async (id: string) => { saved = saved.filter(item => item.id !== id) },
} as unknown as DesktopAPI
createRoot(document.getElementById('app')!).render(<App api={api} />)
