import type { ResourceDescriptor, Topology, TopologyNode, ViewWidget } from './types'
import { resourceKey } from './lib/utils'

export interface ExplorerIndex {
  roots: string[]
  nodesByID: Map<string, TopologyNode>
  parentByID: Map<string, string | undefined>
  childrenByID: Map<string, string[]>
  resourcesByNodeID: Map<string, ResourceDescriptor[]>
}

export interface NodeExplorerRow {
  key: string
  depth: number
  parentKey?: string
  posInSet: number
  setSize: number
  node: TopologyNode
}

export function nodeRowKey(nodeID: string): string {
  return `node:${nodeID}`
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

export function flattenNodeRows(
  index: ExplorerIndex,
  expandedNodeIDs: ReadonlySet<string>,
  rawQuery: string,
  focusedNodeID?: string,
): NodeExplorerRow[] {
  const query = rawQuery.trim().toLocaleLowerCase()
  const visibleNodeIDs = query ? buildVisibleNodeIDs(index, query) : undefined
  const focused = focusedNodeID && index.nodesByID.has(focusedNodeID) ? focusedNodeID : undefined
  const roots = focused ? [focused] : index.roots
  const rows: NodeExplorerRow[] = []
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
    if (visibleNodeIDs && !visibleNodeIDs.has(current.nodeID)) continue
    const node = index.nodesByID.get(current.nodeID)
    if (!node) continue
    const key = nodeRowKey(current.nodeID)
    rows.push({
      key,
      depth: current.depth,
      parentKey: current.parentKey,
      posInSet: current.posInSet,
      setSize: current.setSize,
      node,
    })

    if (!query && !expandedNodeIDs.has(current.nodeID)) continue
    const children = (index.childrenByID.get(current.nodeID) || []).filter((childID) => (
      !visibleNodeIDs || visibleNodeIDs.has(childID)
    ))
    for (let childIndex = children.length - 1; childIndex >= 0; childIndex -= 1) {
      const childID = children[childIndex]
      if (!childID) continue
      stack.push({
        nodeID: childID,
        depth: current.depth + 1,
        parentKey: key,
        posInSet: childIndex + 1,
        setSize: children.length,
      })
    }
  }
  return rows
}

function buildVisibleNodeIDs(index: ExplorerIndex, query: string): Set<string> {
  const visibleNodeIDs = new Set<string>()
  for (const [nodeID, node] of index.nodesByID) {
    if (nodeSearchValues(node).some((value) => value.includes(query))) visibleNodeIDs.add(nodeID)
  }
  for (const nodeID of [...visibleNodeIDs]) {
    let current = index.parentByID.get(nodeID)
    while (current && !visibleNodeIDs.has(current)) {
      visibleNodeIDs.add(current)
      current = index.parentByID.get(current)
    }
  }
  return visibleNodeIDs
}

function nodeSearchValues(node: TopologyNode): string[] {
  return [node.node_id, node.display_name || '', node.role].map((value) => value.toLocaleLowerCase())
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

export function nextWidgetID(resource: ResourceDescriptor, widgets: ViewWidget[]): string {
  const stem = `${resource.id.owner_node_id}-${resource.id.name}`.toLowerCase().replace(/[^a-z0-9._-]+/g, '-').slice(0, 48)
  let suffix = 1
  let candidate = stem
  while (widgets.some((widget) => widget.id === candidate)) candidate = `${stem}-${++suffix}`
  return candidate
}
