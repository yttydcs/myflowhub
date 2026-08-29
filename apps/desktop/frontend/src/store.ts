import type { ResourceDescriptor, Topology, TopologyNode, ViewWidget } from './types'
import { resourceKey } from './lib/utils'

export type ExplorerNode = TopologyNode & { children: ExplorerNode[]; resources: ResourceDescriptor[] }

export interface ExplorerIndex {
  roots: string[]
  nodesByID: Map<string, TopologyNode>
  parentByID: Map<string, string | undefined>
  childrenByID: Map<string, string[]>
  resourcesByNodeID: Map<string, ResourceDescriptor[]>
}

export interface ExplorerRow {
  key: string
  kind: 'node' | 'resource'
  depth: number
  parentKey?: string
  posInSet: number
  setSize: number
  node?: TopologyNode
  resource?: ResourceDescriptor
}

export function nodeRowKey(nodeID: string): string {
  return `node:${nodeID}`
}

export function resourceRowKey(resource: ResourceDescriptor): string {
  return `resource:${resourceKey(resource.id.owner_node_id, resource.id.name)}`
}

export function buildExplorerIndex(topology: Topology, resources: ResourceDescriptor[]): ExplorerIndex {
  const nodesByID = new Map<string, TopologyNode>()
  for (const node of topology.nodes) {
    if (node.node_id && !nodesByID.has(node.node_id)) nodesByID.set(node.node_id, node)
  }

  const parentByID = new Map<string, string | undefined>()
  for (const node of nodesByID.values()) {
    const parentID = node.parent_id
    parentByID.set(node.node_id, parentID && parentID !== node.node_id && nodesByID.has(parentID) ? parentID : undefined)
  }
  breakParentCycles(parentByID)

  const childrenByID = new Map<string, string[]>()
  const resourcesByNodeID = new Map<string, ResourceDescriptor[]>()
  for (const nodeID of nodesByID.keys()) {
    childrenByID.set(nodeID, [])
    resourcesByNodeID.set(nodeID, [])
  }

  const roots: string[] = []
  for (const nodeID of nodesByID.keys()) {
    const parentID = parentByID.get(nodeID)
    if (parentID) childrenByID.get(parentID)?.push(nodeID)
    else roots.push(nodeID)
  }
  for (const resource of resources) resourcesByNodeID.get(resource.id.owner_node_id)?.push(resource)

  const compareNodeIDs = (left: string, right: string) => left.localeCompare(right, undefined, { numeric: true })
  roots.sort(compareNodeIDs)
  for (const childIDs of childrenByID.values()) childIDs.sort(compareNodeIDs)
  for (const nodeResources of resourcesByNodeID.values()) {
    nodeResources.sort((left, right) => left.id.name.localeCompare(right.id.name, undefined, { numeric: true }))
  }

  return { roots, nodesByID, parentByID, childrenByID, resourcesByNodeID }
}

function breakParentCycles(parentByID: Map<string, string | undefined>): void {
  const complete = new Set<string>()
  for (const start of parentByID.keys()) {
    if (complete.has(start)) continue
    const path: string[] = []
    const pathIndex = new Map<string, number>()
    let current: string | undefined = start
    while (current && !complete.has(current)) {
      const cycleStart = pathIndex.get(current)
      if (cycleStart !== undefined) {
        for (const cycleNode of path.slice(cycleStart)) parentByID.set(cycleNode, undefined)
        break
      }
      pathIndex.set(current, path.length)
      path.push(current)
      current = parentByID.get(current)
    }
    for (const nodeID of path) complete.add(nodeID)
  }
}

export function buildExplorerTree(topology: Topology, resources: ResourceDescriptor[]): ExplorerNode[] {
  const index = buildExplorerIndex(topology, resources)
  const built = new Map<string, ExplorerNode>()
  const pending = index.roots.map((nodeID) => ({ nodeID, visited: false }))
  while (pending.length > 0) {
    const current = pending.pop()!
    if (!current.visited) {
      pending.push({ ...current, visited: true })
      const children = index.childrenByID.get(current.nodeID) || []
      for (let childIndex = children.length - 1; childIndex >= 0; childIndex -= 1) {
        const childID = children[childIndex]
        if (childID) pending.push({ nodeID: childID, visited: false })
      }
      continue
    }
    const node = index.nodesByID.get(current.nodeID)
    if (!node) continue
    built.set(current.nodeID, {
      ...node,
      children: (index.childrenByID.get(current.nodeID) || []).flatMap((childID) => {
        const child = built.get(childID)
        return child ? [child] : []
      }),
      resources: index.resourcesByNodeID.get(current.nodeID) || [],
    })
  }
  return index.roots.flatMap((nodeID) => {
    const node = built.get(nodeID)
    return node ? [node] : []
  })
}

export function filterExplorerTree(nodes: ExplorerNode[], rawQuery: string): ExplorerNode[] {
  const query = rawQuery.trim().toLocaleLowerCase()
  if (!query) return nodes
  const filtered = new Map<ExplorerNode, ExplorerNode | null>()
  const pending = nodes.map((node) => ({ node, visited: false }))
  while (pending.length > 0) {
    const current = pending.pop()!
    if (!current.visited) {
      pending.push({ node: current.node, visited: true })
      for (const child of current.node.children) pending.push({ node: child, visited: false })
      continue
    }
    const children = current.node.children.flatMap((child) => {
      const match = filtered.get(child)
      return match ? [match] : []
    })
    const nodeMatches = nodeSearchValues(current.node).some((value) => value.includes(query))
    const matchingResources = nodeMatches
      ? current.node.resources
      : current.node.resources.filter((resource) => resourceSearchValues(resource).some((value) => value.includes(query)))
    filtered.set(current.node, nodeMatches || matchingResources.length > 0 || children.length > 0
      ? { ...current.node, resources: matchingResources, children }
      : null)
  }
  return nodes.flatMap((node) => {
    const match = filtered.get(node)
    return match ? [match] : []
  })
}

export function flattenExplorerRows(
  index: ExplorerIndex,
  expandedNodeIDs: ReadonlySet<string>,
  rawQuery: string,
  focusedNodeID?: string,
): ExplorerRow[] {
  const query = rawQuery.trim().toLocaleLowerCase()
  const search = buildSearchIndex(index, query)
  const focused = focusedNodeID && index.nodesByID.has(focusedNodeID) ? focusedNodeID : undefined
  const roots = focused ? [focused] : index.roots
  const rows: ExplorerRow[] = []
  const stack: Array<{
    nodeID: string
    depth: number
    parentKey?: string
    posInSet: number
    setSize: number
  }> = roots.slice().reverse().map((nodeID, reverseIndex) => ({
    nodeID,
    depth: 1,
    parentKey: undefined,
    posInSet: roots.length - reverseIndex,
    setSize: roots.length,
  }))

  while (stack.length > 0) {
    const current = stack.pop()!
    if (query && !search.visibleNodeIDs.has(current.nodeID)) continue
    const node = index.nodesByID.get(current.nodeID)
    if (!node) continue
    const nodeKey = nodeRowKey(current.nodeID)
    rows.push({
      key: nodeKey,
      kind: 'node',
      depth: current.depth,
      parentKey: current.parentKey,
      posInSet: current.posInSet,
      setSize: current.setSize,
      node,
    })

    const expanded = query.length > 0 || expandedNodeIDs.has(current.nodeID)
    if (!expanded) continue
    const resources = (index.resourcesByNodeID.get(current.nodeID) || []).filter((resource) => (
      !query || search.matchedNodeIDs.has(current.nodeID) || search.matchedResourceKeys.has(resourceRowKey(resource))
    ))
    const children = (index.childrenByID.get(current.nodeID) || []).filter((childID) => (
      !query || search.visibleNodeIDs.has(childID)
    ))
    const childSetSize = resources.length + children.length
    for (let childIndex = children.length - 1; childIndex >= 0; childIndex -= 1) {
      const childID = children[childIndex]
      if (!childID) continue
      stack.push({
        nodeID: childID,
        depth: current.depth + 1,
        parentKey: nodeKey,
        posInSet: resources.length + childIndex + 1,
        setSize: childSetSize,
      })
    }
    for (let resourceIndex = 0; resourceIndex < resources.length; resourceIndex += 1) {
      const resource = resources[resourceIndex]
      if (!resource) continue
      rows.push({
        key: resourceRowKey(resource),
        kind: 'resource',
        depth: current.depth + 1,
        parentKey: nodeKey,
        posInSet: resourceIndex + 1,
        setSize: childSetSize,
        resource,
      })
    }
  }
  return rows
}

function buildSearchIndex(index: ExplorerIndex, query: string): {
  matchedNodeIDs: Set<string>
  matchedResourceKeys: Set<string>
  visibleNodeIDs: Set<string>
} {
  const matchedNodeIDs = new Set<string>()
  const matchedResourceKeys = new Set<string>()
  const visibleNodeIDs = new Set<string>()
  if (!query) return { matchedNodeIDs, matchedResourceKeys, visibleNodeIDs }

  for (const [nodeID, node] of index.nodesByID) {
    const nodeMatched = nodeSearchValues(node).some((value) => value.includes(query))
    let resourceMatched = false
    for (const resource of index.resourcesByNodeID.get(nodeID) || []) {
      if (!resourceSearchValues(resource).some((value) => value.includes(query))) continue
      matchedResourceKeys.add(resourceRowKey(resource))
      resourceMatched = true
    }
    if (nodeMatched) matchedNodeIDs.add(nodeID)
    if (nodeMatched || resourceMatched) visibleNodeIDs.add(nodeID)
  }

  for (const nodeID of [...visibleNodeIDs]) {
    let current = index.parentByID.get(nodeID)
    while (current && !visibleNodeIDs.has(current)) {
      visibleNodeIDs.add(current)
      current = index.parentByID.get(current)
    }
  }
  return { matchedNodeIDs, matchedResourceKeys, visibleNodeIDs }
}

function nodeSearchValues(node: TopologyNode): string[] {
  return [node.node_id, node.display_name || '', node.role].map((value) => value.toLocaleLowerCase())
}

function resourceSearchValues(resource: ResourceDescriptor): string[] {
  return [resource.id.name, resource.presentation?.label || '', resource.type].map((value) => value.toLocaleLowerCase())
}

export function defaultExpandedNodeIDs(index: ExplorerIndex): string[] {
  const expanded: string[] = []
  const queue = index.roots.map((nodeID) => ({ nodeID, depth: 1 }))
  for (let cursor = 0; cursor < queue.length; cursor += 1) {
    const current = queue[cursor]
    if (!current) continue
    if (current.depth <= 2) expanded.push(current.nodeID)
    if (current.depth >= 2) continue
    for (const childID of index.childrenByID.get(current.nodeID) || []) {
      queue.push({ nodeID: childID, depth: current.depth + 1 })
    }
  }
  return expanded
}

export function explorerBreadcrumb(index: ExplorerIndex, focusedNodeID?: string): TopologyNode[] {
  if (!focusedNodeID || !index.nodesByID.has(focusedNodeID)) return []
  const path: TopologyNode[] = []
  let current: string | undefined = focusedNodeID
  while (current) {
    const node = index.nodesByID.get(current)
    if (!node) break
    path.push(node)
    current = index.parentByID.get(current)
  }
  return path.reverse()
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
