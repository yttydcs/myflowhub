import type { ResourceDescriptor, Topology, TopologyNode, ViewWidget } from './types'
import { resourceKey } from './lib/utils'

export type ExplorerNode = TopologyNode & { children: ExplorerNode[]; resources: ResourceDescriptor[] }

export function buildExplorerTree(topology: Topology, resources: ResourceDescriptor[]): ExplorerNode[] {
  const byID = new Map<string, ExplorerNode>()
  for (const node of topology.nodes) byID.set(node.node_id, { ...node, children: [], resources: [] })
  for (const resource of resources) byID.get(resource.id.owner_node_id)?.resources.push(resource)
  const roots: ExplorerNode[] = []
  for (const node of byID.values()) {
    const parent = node.parent_id ? byID.get(node.parent_id) : undefined
    if (parent) parent.children.push(node)
    else roots.push(node)
  }
  const sort = (nodes: ExplorerNode[]) => {
    nodes.sort((a, b) => a.node_id.localeCompare(b.node_id, undefined, { numeric: true }))
    for (const node of nodes) {
      node.resources.sort((a, b) => a.id.name.localeCompare(b.id.name))
      sort(node.children)
    }
  }
  sort(roots)
  return roots
}

export function indexResources(resources: ResourceDescriptor[]): Map<string, ResourceDescriptor> {
  return new Map(resources.map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource]))
}

export function defaultRenderer(resource: ResourceDescriptor | undefined): string {
  return resource?.presentation?.renderer || resource?.type || 'mfh.unknown'
}

export function addWidget(widgets: ViewWidget[], resource: ResourceDescriptor, id: string): ViewWidget[] {
  if (widgets.some((widget) => widget.id === id)) return widgets
  const index = widgets.length
  return [
    ...widgets,
    {
      id,
      owner_node_id: resource.id.owner_node_id,
      resource_name: resource.id.name,
      renderer: defaultRenderer(resource),
      x: (index * 4) % 12,
      y: Math.floor(index / 3) * 4,
      w: resource.type === 'mfh.stream' || resource.type === 'mfh.topic' ? 8 : 4,
      h: 4,
    },
  ]
}

export function updateWidget(
  widgets: ViewWidget[],
  id: string,
  update: Partial<Pick<ViewWidget, 'x' | 'y' | 'w' | 'h'>>,
): ViewWidget[] {
  return widgets.map((widget) => {
    if (widget.id !== id) return widget
    const next = { ...widget, ...update }
    next.w = Math.min(12, Math.max(1, next.w))
    next.h = Math.min(24, Math.max(1, next.h))
    next.x = Math.min(12 - next.w, Math.max(0, next.x))
    next.y = Math.max(0, next.y)
    return next
  })
}

export function removeWidget(widgets: ViewWidget[], id: string): ViewWidget[] {
  return widgets.filter((widget) => widget.id !== id)
}

export function nextWidgetID(resource: ResourceDescriptor, widgets: ViewWidget[]): string {
  const stem = `${resource.id.owner_node_id}-${resource.id.name}`.toLowerCase().replace(/[^a-z0-9._-]+/g, '-').slice(0, 48)
  let suffix = 1
  let candidate = stem
  while (widgets.some((widget) => widget.id === candidate)) candidate = `${stem}-${++suffix}`
  return candidate
}
