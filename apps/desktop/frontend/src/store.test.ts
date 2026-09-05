import { describe, expect, it } from 'vitest'
import {
  buildExplorerIndex,
  explorerBreadcrumb,
  flattenNodeRows,
  nodeRowKey,
} from './store'
import type { ResourceDescriptor, Topology } from './types'

const resource: ResourceDescriptor = {
  id: { owner_node_id: '3', name: 'metrics/cpu' },
  type: 'mfh.variable',
  type_version: 1,
  capabilities: [{ name: 'subscribe', permission: 'metrics.read', event_schema: 'metrics.v1', max_payload_bytes: 1024 }],
  limits: { max_payload_bytes: 1024 },
}

describe('workspace domain', () => {
  it('indexes authority and resource ownership without deriving either from resource paths', () => {
    const topology: Topology = {
      nodes: [
        { node_id: '1', role: 'root', has_children: true, generation: 1 },
        { node_id: '3', parent_id: '1', role: 'child', has_children: false, generation: 1 },
      ],
    }
    const index = buildExplorerIndex(topology, [resource])
    expect(index.childrenByID.get('1')).toEqual(['3'])
    expect(index.resourcesByNodeID.get('3')?.[0]?.id.name).toBe('metrics/cpu')
  })

  it('flattens arbitrary-depth Node trees with Node-only ARIA hierarchy metadata', () => {
    const nodes = Array.from({ length: 3_000 }, (_, index) => ({
      node_id: String(index + 1),
      parent_id: index === 0 ? undefined : String(index),
      role: index === 0 ? 'root' : 'child',
      has_children: index < 2999,
      generation: 1,
    }))
    const index = buildExplorerIndex({ nodes }, [resource])
    const rows = flattenNodeRows(index, new Set(nodes.map((node) => node.node_id)), '')
    expect(rows).toHaveLength(nodes.length)
    expect(rows.at(-1)).toMatchObject({ key: nodeRowKey('3000'), depth: 3000, posInSet: 1, setSize: 1 })
    expect(rows.some((row) => row.key.includes('metrics/cpu'))).toBe(false)
    expect(explorerBreadcrumb(index, '3000')).toHaveLength(3000)
  })

  it('turns malformed parent cycles and missing parents into stable roots', () => {
    const index = buildExplorerIndex({
      nodes: [
        { node_id: '1', parent_id: '2', role: 'child', has_children: false, generation: 1 },
        { node_id: '2', parent_id: '1', role: 'child', has_children: false, generation: 1 },
        { node_id: '3', parent_id: 'missing', role: 'child', has_children: false, generation: 1 },
      ],
    }, [])
    expect(index.roots).toEqual(['1', '2', '3'])
    expect(flattenNodeRows(index, new Set(), '')).toHaveLength(3)
  })

  it('keeps matching deep Nodes and their ancestors without mixing Resources into search results', () => {
    const topology: Topology = {
      nodes: [
        { node_id: '1', display_name: 'Root', role: 'root', has_children: true, generation: 1 },
        { node_id: '2', parent_id: '1', display_name: 'Branch', role: 'branch', has_children: true, generation: 1 },
        { node_id: '3', parent_id: '2', display_name: 'Leaf', role: 'leaf', has_children: false, generation: 1 },
      ],
    }
    const index = buildExplorerIndex(topology, [resource])
    expect(flattenNodeRows(index, new Set(), 'leaf').map((row) => row.key)).toEqual([
      nodeRowKey('1'),
      nodeRowKey('2'),
      nodeRowKey('3'),
    ])
    expect(flattenNodeRows(index, new Set(), 'metrics/cpu')).toHaveLength(0)
  })

  it('indexes, flattens, and filters a representative large catalog within the desktop budget', () => {
    const nodes = Array.from({ length: 2_000 }, (_, index) => ({
      node_id: String(index + 1),
      parent_id: index === 0 ? undefined : '1',
      display_name: index === 1_999 ? 'Target leaf' : undefined,
      role: index === 0 ? 'root' : 'child',
      has_children: index === 0,
      generation: 1,
    }))
    const resources = Array.from({ length: 10_000 }, (_, index): ResourceDescriptor => ({
      ...resource,
      id: { owner_node_id: '1', name: `metrics/value-${index}` },
    }))
    const started = performance.now()
    const explorerIndex = buildExplorerIndex({ nodes }, resources)
    const rows = flattenNodeRows(explorerIndex, new Set(), 'target leaf')
    const elapsed = performance.now() - started
    expect(rows.map((row) => row.node.node_id)).toEqual(['1', '2000'])
    expect(elapsed).toBeLessThan(750)
  })
})
