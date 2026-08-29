import { describe, expect, it } from 'vitest'
import {
  addWidget,
  buildExplorerIndex,
  explorerBreadcrumb,
  flattenNodeRows,
  groupResources,
  moveWidget,
  nodeRowKey,
  removeWidget,
  resizeWorkspacePanels,
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
  it('indexes authority and resource ownership without deriving either from resource paths', () => {
    const topology: Topology = {
      version: 1,
      epoch: 2,
      nodes: [
        { node_id: '1', role: 'root', generation: 1 },
        { node_id: '3', parent_id: '1', role: 'child', generation: 1 },
      ],
    }
    const index = buildExplorerIndex(topology, [resource])
    expect(index.childrenByID.get('1')).toEqual(['3'])
    expect(index.resourcesByNodeID.get('3')?.[0]?.id.name).toBe('metrics/cpu')
  })

  it('fills the workspace with the first widget and splits the second widget to the right', () => {
    const widgets = addWidget([], resource, 'cpu')
    expect(widgets[0]).toMatchObject({ renderer: 'mfh.variable', x: 0, y: 0, w: 12, h: 24 })
    const memory = { ...resource, id: { ...resource.id, name: 'metrics/memory' } }
    expect(addWidget(widgets, memory, 'memory')).toMatchObject([
      { id: 'cpu', x: 0, y: 0, w: 6, h: 24 },
      { id: 'memory', x: 6, y: 0, w: 6, h: 24 },
    ])
  })

  it('inserts, reorders, resizes, and removes workspace panels without changing the View schema', () => {
    const memory = { ...resource, id: { ...resource.id, name: 'metrics/memory' } }
    const split = addWidget(addWidget([], resource, 'cpu'), memory, 'memory', { targetID: 'cpu', side: 'left' })
    expect(split.map((widget) => widget.id)).toEqual(['memory', 'cpu'])

    const reordered = moveWidget(split, 'memory', 'cpu', 'right')
    expect(reordered.map((widget) => widget.id)).toEqual(['cpu', 'memory'])
    expect(resizeWorkspacePanels(reordered, 8)).toMatchObject([
      { id: 'cpu', x: 0, w: 8 },
      { id: 'memory', x: 8, w: 4 },
    ])
    expect(removeWidget(reordered, 'cpu')).toMatchObject([{ id: 'memory', x: 0, y: 0, w: 12, h: 24 }])
  })

  it('keeps the generic widget bounds helper compatible with persisted layouts', () => {
    const widgets = addWidget([], resource, 'cpu')
    const resized = updateWidget(widgets, 'cpu', { x: 11, w: 8, h: 30 })
    expect(resized[0]).toMatchObject({ x: 4, w: 8, h: 24 })
  })

  it('flattens arbitrary-depth Node trees with Node-only ARIA hierarchy metadata', () => {
    const nodes = Array.from({ length: 3_000 }, (_, index) => ({
      node_id: String(index + 1),
      parent_id: index === 0 ? undefined : String(index),
      role: index === 0 ? 'root' : 'child',
      generation: 1,
    }))
    const index = buildExplorerIndex({ version: 1, epoch: 1, nodes }, [resource])
    const rows = flattenNodeRows(index, new Set(nodes.map((node) => node.node_id)), '')
    expect(rows).toHaveLength(nodes.length)
    expect(rows.at(-1)).toMatchObject({ key: nodeRowKey('3000'), depth: 3000, posInSet: 1, setSize: 1 })
    expect(rows.some((row) => row.key.includes('metrics/cpu'))).toBe(false)
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
    expect(flattenNodeRows(index, new Set(), '')).toHaveLength(3)
  })

  it('keeps matching deep Nodes and their ancestors without mixing Resources into search results', () => {
    const topology: Topology = {
      version: 1,
      epoch: 1,
      nodes: [
        { node_id: '1', display_name: 'Root', role: 'root', generation: 1 },
        { node_id: '2', parent_id: '1', display_name: 'Branch', role: 'branch', generation: 1 },
        { node_id: '3', parent_id: '2', display_name: 'Leaf', role: 'leaf', generation: 1 },
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

  it('groups direct Resources by presentation prefix and filters without changing descriptors', () => {
    const resources: ResourceDescriptor[] = [
      resource,
      { ...resource, id: { owner_node_id: '3', name: 'system/health' }, presentation: { label: 'Health' } },
      { ...resource, id: { owner_node_id: '3', name: 'metrics/memory' } },
      { ...resource, id: { owner_node_id: '3', name: 'other/value' } },
      { ...resource, id: { owner_node_id: '3', name: 'ungrouped' } },
    ]
    const groups = groupResources(resources)
    expect(groups.map((group) => [group.label, group.resources.length])).toEqual([
      ['metrics', 2],
      ['other', 1],
      ['system', 1],
      ['其他', 1],
    ])
    expect(groupResources(resources, 'health')[0]?.resources[0]).toBe(resources[1])
  })

  it('indexes, flattens, and filters a representative large catalog within the desktop budget', () => {
    const nodes = Array.from({ length: 2_000 }, (_, index) => ({
      node_id: String(index + 1),
      parent_id: index === 0 ? undefined : '1',
      display_name: index === 1_999 ? 'Target leaf' : undefined,
      role: index === 0 ? 'root' : 'child',
      generation: 1,
    }))
    const resources = Array.from({ length: 10_000 }, (_, index): ResourceDescriptor => ({
      ...resource,
      id: { owner_node_id: '1', name: `metrics/value-${index}` },
    }))
    const started = performance.now()
    const explorerIndex = buildExplorerIndex({ version: 1, epoch: 1, nodes }, resources)
    const rows = flattenNodeRows(explorerIndex, new Set(), 'target leaf')
    const groups = groupResources(explorerIndex.resourcesByNodeID.get('1') || [], 'value-9999')
    const elapsed = performance.now() - started
    expect(rows.map((row) => row.node.node_id)).toEqual(['1', '2000'])
    expect(groups[0]?.resources[0]?.id.name).toBe('metrics/value-9999')
    expect(elapsed).toBeLessThan(750)
  })
})
