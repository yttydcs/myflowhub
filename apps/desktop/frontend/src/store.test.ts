import { describe, expect, it } from 'vitest'
import {
  addWidget,
  buildExplorerIndex,
  buildExplorerTree,
  explorerBreadcrumb,
  filterExplorerTree,
  flattenExplorerRows,
  nodeRowKey,
  updateWidget,
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

  it('keeps matching resources from deep descendant nodes', () => {
    const topology: Topology = {
      version: 1,
      epoch: 2,
      nodes: [
        { node_id: '1', role: 'root', generation: 1 },
        { node_id: '2', parent_id: '1', role: 'branch', generation: 1 },
        { node_id: '3', parent_id: '2', role: 'leaf', generation: 1 },
      ],
    }
    const filtered = filterExplorerTree(buildExplorerTree(topology, [resource]), 'metrics/cpu')
    expect(filtered[0]?.children[0]?.children[0]?.resources[0]?.id.name).toBe('metrics/cpu')
  })

  it('flattens arbitrary-depth trees and exposes ARIA hierarchy metadata without recursion', () => {
    const nodes = Array.from({ length: 3_000 }, (_, index) => ({
      node_id: String(index + 1),
      parent_id: index === 0 ? undefined : String(index),
      role: index === 0 ? 'root' : 'child',
      generation: 1,
    }))
    const index = buildExplorerIndex({ version: 1, epoch: 1, nodes }, [])
    const rows = flattenExplorerRows(index, new Set(nodes.map((node) => node.node_id)), '')
    expect(rows).toHaveLength(nodes.length)
    expect(rows.at(-1)).toMatchObject({ key: nodeRowKey('3000'), depth: 3000, posInSet: 1, setSize: 1 })
    expect(explorerBreadcrumb(index, '3000')).toHaveLength(3000)
  })

  it('turns malformed parent cycles and missing parents into stable roots', () => {
    const index = buildExplorerIndex({
      version: 1,
      epoch: 1,
      nodes: [
        { node_id: '1', parent_id: '2', role: 'child', generation: 1 },
        { node_id: '2', parent_id: '1', role: 'child', generation: 1 },
        { node_id: '3', parent_id: 'missing', role: 'child', generation: 1 },
      ],
    }, [])
    expect(index.roots).toEqual(['1', '2', '3'])
    expect(flattenExplorerRows(index, new Set(), '')).toHaveLength(3)
  })

  it('keeps matched deep resources and all ancestors in visible row order', () => {
    const topology: Topology = {
      version: 1,
      epoch: 1,
      nodes: [
        { node_id: '1', role: 'root', generation: 1 },
        { node_id: '2', parent_id: '1', role: 'branch', generation: 1 },
        { node_id: '3', parent_id: '2', role: 'leaf', generation: 1 },
      ],
    }
    const unrelatedResource: ResourceDescriptor = {
      ...resource,
      id: { owner_node_id: '3', name: 'metrics/memory' },
    }
    const rows = flattenExplorerRows(buildExplorerIndex(topology, [resource, unrelatedResource]), new Set(), 'metrics/cpu')
    expect(rows.map((row) => row.key)).toEqual([
      nodeRowKey('1'),
      nodeRowKey('2'),
      nodeRowKey('3'),
      'resource:3:metrics/cpu',
    ])
  })

  it('builds and filters a representative large catalog within the desktop budget', () => {
    const nodes = Array.from({ length: 2_000 }, (_, index) => ({
      node_id: String(index + 1),
      parent_id: index === 0 ? undefined : '1',
      role: index === 0 ? 'root' : 'child',
      generation: 1,
    }))
    const resources = Array.from({ length: 10_000 }, (_, index): ResourceDescriptor => ({
      ...resource,
      id: { owner_node_id: String((index % nodes.length) + 1), name: `metrics/value-${index}` },
    }))
    const started = performance.now()
    const tree = buildExplorerTree({ version: 1, epoch: 1, nodes }, resources)
    const filtered = filterExplorerTree(tree, 'value-9999')
    const elapsed = performance.now() - started
    expect(filtered).toHaveLength(1)
    expect(elapsed).toBeLessThan(750)
  })
})
