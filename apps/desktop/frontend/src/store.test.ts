import { describe, expect, it } from 'vitest'
import { addWidget, buildExplorerTree, updateWidget } from './store'
import type { ResourceDescriptor, Topology } from './types'

const resource: ResourceDescriptor = {
  id: { owner_node_id: '3', name: 'metrics/cpu' },
  type: 'mfh.variable',
  type_version: 1,
  capabilities: [{ name: 'subscribe', permission: 'metrics.read', event_schema: 'metrics.v1', max_payload_bytes: 1024 }],
  limits: { max_payload_bytes: 1024 },
}

describe('workspace domain', () => {
  it('builds the authority tree without deriving ownership from paths', () => {
    const topology: Topology = {
      version: 1,
      epoch: 2,
      nodes: [
        { node_id: '1', role: 'root', generation: 1 },
        { node_id: '3', parent_id: '1', role: 'child', generation: 1 },
      ],
    }
    const tree = buildExplorerTree(topology, [resource])
    expect(tree[0]?.children[0]?.node_id).toBe('3')
    expect(tree[0]?.children[0]?.resources[0]?.id.name).toBe('metrics/cpu')
  })

  it('adds and bounds responsive widgets', () => {
    const widgets = addWidget([], resource, 'cpu')
    expect(widgets[0]?.renderer).toBe('mfh.variable')
    const resized = updateWidget(widgets, 'cpu', { x: 11, w: 8, h: 30 })
    expect(resized[0]).toMatchObject({ x: 4, w: 8, h: 24 })
  })
})
