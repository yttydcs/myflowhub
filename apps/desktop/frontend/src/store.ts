import type { ResourceDescriptor, Topology, TopologyNode, ViewDefinition, ViewLayout, ViewWidget } from './types'
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

export interface ResourceGroup {
  key: string
  label: string
  resources: ResourceDescriptor[]
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

function resourceSearchValues(resource: ResourceDescriptor): string[] {
  return [resource.id.name, resource.presentation?.label || '', resource.type].map((value) => value.toLocaleLowerCase())
}

export function groupResources(resources: ResourceDescriptor[], rawQuery = ''): ResourceGroup[] {
  const query = rawQuery.trim().toLocaleLowerCase()
  const groups = new Map<string, ResourceGroup>()
  for (const resource of resources) {
    if (query && !resourceSearchValues(resource).some((value) => value.includes(query))) continue
    const separator = resource.id.name.indexOf('/')
    const segment = separator > 0 ? resource.id.name.slice(0, separator) : undefined
    const key = segment ? `segment:${segment}` : 'ungrouped'
    let group = groups.get(key)
    if (!group) {
      group = { key, label: segment || '其他', resources: [] }
      groups.set(key, group)
    }
    group.resources.push(resource)
  }
  return [...groups.values()]
    .sort((left, right) => {
      if (left.key === 'ungrouped') return 1
      if (right.key === 'ungrouped') return -1
      return left.label.localeCompare(right.label, undefined, { numeric: true })
    })
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

export type WidgetPlacement = {
  targetID?: string
  side?: 'left' | 'right'
}

const WORKSPACE_COLUMNS = 12
const WORKSPACE_ROWS = 24
const MAX_WIDGETS_PER_ROW = 4
export const MIN_WORKSPACE_SPLIT_RATIO = 0.2
export const MAX_WORKSPACE_SPLIT_RATIO = 0.8

export function resolveWorkspaceLayout(view: ViewDefinition): ViewLayout {
  const persistedRatio = view.layout?.split_ratio
  const legacyRatio = view.widgets.length === 2 ? (view.widgets[0]?.w || WORKSPACE_COLUMNS / 2) / WORKSPACE_COLUMNS : 0.5
  const ratio = Number.isFinite(persistedRatio) ? persistedRatio! : legacyRatio
  return {
    direction: view.layout?.direction === 'vertical' ? 'vertical' : 'horizontal',
    split_ratio: clampWorkspaceSplitRatio(ratio),
  }
}

export function clampWorkspaceSplitRatio(ratio: number): number {
  if (!Number.isFinite(ratio)) return 0.5
  return Math.min(MAX_WORKSPACE_SPLIT_RATIO, Math.max(MIN_WORKSPACE_SPLIT_RATIO, ratio))
}

export function arrangeWorkspaceWidgets(widgets: ViewWidget[], preserveTwoPanelRatio = true): ViewWidget[] {
  if (widgets.length === 0) return widgets
  if (widgets.length === 1) {
    const widget = widgets[0]!
    return [{ ...widget, x: 0, y: 0, w: WORKSPACE_COLUMNS, h: WORKSPACE_ROWS }]
  }
  if (widgets.length === 2 && preserveTwoPanelRatio && isCompleteTwoPanelLayout(widgets)) {
    return widgets.map((widget) => ({ ...widget, y: 0, h: WORKSPACE_ROWS }))
  }

  const rowCount = Math.ceil(widgets.length / MAX_WIDGETS_PER_ROW)
  const rowHeights = distributeUnits(WORKSPACE_ROWS, rowCount)
  let y = 0
  return widgets.map((widget, index) => {
    const rowIndex = Math.floor(index / MAX_WIDGETS_PER_ROW)
    const rowStart = rowIndex * MAX_WIDGETS_PER_ROW
    const widgetsInRow = Math.min(MAX_WIDGETS_PER_ROW, widgets.length - rowStart)
    const columnWidths = distributeUnits(WORKSPACE_COLUMNS, widgetsInRow)
    const columnIndex = index - rowStart
    const x = columnWidths.slice(0, columnIndex).reduce((total, width) => total + width, 0)
    const next = {
      ...widget,
      x,
      y,
      w: columnWidths[columnIndex]!,
      h: rowHeights[rowIndex]!,
    }
    if (columnIndex === widgetsInRow - 1) y += rowHeights[rowIndex]!
    return next
  })
}

export function addWidget(
  widgets: ViewWidget[],
  resource: ResourceDescriptor,
  id: string,
  placement: WidgetPlacement = {},
): ViewWidget[] {
  if (widgets.some((widget) => widget.id === id)) return widgets
  const widget: ViewWidget = {
    id,
    owner_node_id: resource.id.owner_node_id,
    resource_name: resource.id.name,
    renderer: defaultRenderer(resource),
    x: 0,
    y: 0,
    w: WORKSPACE_COLUMNS,
    h: WORKSPACE_ROWS,
  }
  const next = [...widgets]
  const targetIndex = placement.targetID
    ? next.findIndex((candidate) => candidate.id === placement.targetID)
    : -1
  const insertionIndex = targetIndex < 0
    ? placement.side === 'left' ? 0 : next.length
    : targetIndex + (placement.side === 'right' ? 1 : 0)
  next.splice(insertionIndex, 0, widget)
  return arrangeWorkspaceWidgets(next, false)
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
  return arrangeWorkspaceWidgets(widgets.filter((widget) => widget.id !== id), false)
}

export function moveWidget(
  widgets: ViewWidget[],
  sourceID: string,
  targetID: string,
  side: 'left' | 'right' = 'left',
): ViewWidget[] {
  if (sourceID === targetID) return widgets
  const source = widgets.find((widget) => widget.id === sourceID)
  if (!source || !widgets.some((widget) => widget.id === targetID)) return widgets
  const next = widgets.filter((widget) => widget.id !== sourceID)
  const targetIndex = next.findIndex((widget) => widget.id === targetID)
  next.splice(targetIndex + (side === 'right' ? 1 : 0), 0, source)
  if (widgets.length === 2 && isCompleteTwoPanelLayout(widgets)) {
    return resizeWorkspacePanels(next, widgets[0]!.w)
  }
  return arrangeWorkspaceWidgets(next, false)
}

export function moveWidgetByOffset(widgets: ViewWidget[], id: string, offset: -1 | 1): ViewWidget[] {
  const sourceIndex = widgets.findIndex((widget) => widget.id === id)
  const targetIndex = sourceIndex + offset
  if (sourceIndex < 0 || targetIndex < 0 || targetIndex >= widgets.length) return widgets
  return moveWidget(widgets, id, widgets[targetIndex]!.id, offset < 0 ? 'left' : 'right')
}

export function swapWorkspacePanels(widgets: ViewWidget[]): ViewWidget[] {
  if (widgets.length !== 2) return widgets
  return resizeWorkspacePanels([widgets[1]!, widgets[0]!], widgets[0]!.w)
}

export function resizeWorkspacePanels(widgets: ViewWidget[], leftWidth: number): ViewWidget[] {
  if (widgets.length !== 2) return widgets
  const width = Math.min(10, Math.max(2, Math.round(leftWidth)))
  return [
    { ...widgets[0]!, x: 0, y: 0, w: width, h: WORKSPACE_ROWS },
    { ...widgets[1]!, x: width, y: 0, w: WORKSPACE_COLUMNS - width, h: WORKSPACE_ROWS },
  ]
}

export function nextWidgetID(resource: ResourceDescriptor, widgets: ViewWidget[]): string {
  const stem = `${resource.id.owner_node_id}-${resource.id.name}`.toLowerCase().replace(/[^a-z0-9._-]+/g, '-').slice(0, 48)
  let suffix = 1
  let candidate = stem
  while (widgets.some((widget) => widget.id === candidate)) candidate = `${stem}-${++suffix}`
  return candidate
}

function isCompleteTwoPanelLayout(widgets: ViewWidget[]): boolean {
  const [left, right] = widgets
  return Boolean(
    left
    && right
    && left.x === 0
    && left.y === 0
    && right.y === 0
    && left.w >= 2
    && right.w >= 2
    && left.x + left.w === right.x
    && right.x + right.w === WORKSPACE_COLUMNS,
  )
}

function distributeUnits(total: number, count: number): number[] {
  const base = Math.floor(total / count)
  const remainder = total % count
  return Array.from({ length: count }, (_, index) => base + (index < remainder ? 1 : 0))
}
