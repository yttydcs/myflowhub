import type { ViewDefinition, ViewLayoutAxis, ViewLayoutNode, ViewWidget } from './types'

export const MAX_WORKSPACE_WIDGETS = 64
export const MAX_WORKSPACE_LAYOUT_DEPTH = 32
export const MAX_WORKSPACE_LAYOUT_NODES = 127
export const DEFAULT_WORKSPACE_LEAF_WIDTH = 200
export const DEFAULT_WORKSPACE_LEAF_HEIGHT = 120
export const WORKSPACE_SEPARATOR_SIZE = 8

const WEIGHT_EPSILON = 1e-6

export type WorkspaceDockSide = 'left' | 'right' | 'top' | 'bottom'

export type WorkspaceDockIntent =
  | { kind: 'panel-edge'; targetWidgetID: string; side: WorkspaceDockSide }
  | {
      kind: 'split-gap'
      parentPath: number[]
      insertionIndex: number
      beforeWidgetID: string
      afterWidgetID: string
    }
  | { kind: 'root-edge'; side: WorkspaceDockSide }
  | { kind: 'empty-workspace' }

export type WorkspaceDockSource =
  | { kind: 'existing'; widgetID: string }
  | { kind: 'new'; widget: ViewWidget }

export type WorkspaceLayoutSize = { width: number; height: number }

export class WorkspaceLayoutError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'WorkspaceLayoutError'
  }
}

export function workspaceDockAxis(side: WorkspaceDockSide): ViewLayoutAxis {
  return side === 'top' || side === 'bottom' ? 'vertical' : 'horizontal'
}

export function workspaceDockBefore(side: WorkspaceDockSide): boolean {
  return side === 'left' || side === 'top'
}

export function workspacePanelDockSide(
  normalizedX: number,
  normalizedY: number,
  edgeThreshold = 0.28,
): WorkspaceDockSide | undefined {
  if (![normalizedX, normalizedY, edgeThreshold].every(Number.isFinite) || edgeThreshold <= 0 || edgeThreshold >= 0.5) {
    throw new WorkspaceLayoutError('panel docking coordinates or threshold are invalid')
  }
  const distances: Array<[WorkspaceDockSide, number]> = [
    ['left', Math.abs(normalizedX)],
    ['right', Math.abs(1 - normalizedX)],
    ['top', Math.abs(normalizedY)],
    ['bottom', Math.abs(1 - normalizedY)],
  ]
  const nearest = distances.reduce((current, candidate) => candidate[1] < current[1] ? candidate : current)
  return nearest[1] <= edgeThreshold ? nearest[0] : undefined
}

export function workspaceDropTargetPriority(id: string): number {
  if (id.startsWith('workspace-divider:')) return 0
  if (id.startsWith('workspace-root-edge:')) return 1
  if (id.startsWith('workspace-panel:')) return 2
  if (id === 'workspace-drop') return 3
  return 4
}

export function sameWorkspaceDockIntent(left: WorkspaceDockIntent | null, right: WorkspaceDockIntent | null): boolean {
  if (left === right) return true
  if (!left || !right || left.kind !== right.kind) return false
  switch (left.kind) {
    case 'empty-workspace':
      return true
    case 'root-edge':
      return left.side === (right as typeof left).side
    case 'panel-edge': {
      const candidate = right as typeof left
      return left.targetWidgetID === candidate.targetWidgetID && left.side === candidate.side
    }
    case 'split-gap': {
      const candidate = right as typeof left
      return left.insertionIndex === candidate.insertionIndex
        && left.beforeWidgetID === candidate.beforeWidgetID
        && left.afterWidgetID === candidate.afterWidgetID
        && left.parentPath.length === candidate.parentPath.length
        && left.parentPath.every((segment, index) => segment === candidate.parentPath[index])
    }
  }
}

export function createWorkspaceLeaf(widgetID: string): ViewLayoutNode {
  if (!widgetID) throw new WorkspaceLayoutError('widget ID is required')
  return { kind: 'leaf', widget_id: widgetID }
}

export function validateWorkspaceView(view: ViewDefinition): void {
  if (view.widgets.length > MAX_WORKSPACE_WIDGETS) {
    throw new WorkspaceLayoutError(`view supports at most ${MAX_WORKSPACE_WIDGETS} widgets`)
  }
  const widgetIDs = new Set<string>()
  for (const widget of view.widgets) {
    if (!widget.id) throw new WorkspaceLayoutError('widget ID is required')
    if (widgetIDs.has(widget.id)) throw new WorkspaceLayoutError(`duplicate widget "${widget.id}"`)
    widgetIDs.add(widget.id)
  }
  if (widgetIDs.size === 0) {
    if (view.layout_root) throw new WorkspaceLayoutError('empty view must not have a layout root')
    return
  }
  if (!view.layout_root) throw new WorkspaceLayoutError('non-empty view requires a layout root')

  const leaves = new Set<string>()
  let nodes = 0
  const visit = (node: ViewLayoutNode, depth: number, parentAxis?: ViewLayoutAxis): void => {
    nodes += 1
    if (nodes > MAX_WORKSPACE_LAYOUT_NODES) throw new WorkspaceLayoutError(`layout exceeds ${MAX_WORKSPACE_LAYOUT_NODES} nodes`)
    if (depth > MAX_WORKSPACE_LAYOUT_DEPTH) throw new WorkspaceLayoutError(`layout exceeds depth ${MAX_WORKSPACE_LAYOUT_DEPTH}`)
    if (node.kind === 'leaf') {
      if (!node.widget_id) throw new WorkspaceLayoutError('layout leaf widget ID is required')
      if (leaves.has(node.widget_id)) throw new WorkspaceLayoutError(`duplicate layout leaf "${node.widget_id}"`)
      leaves.add(node.widget_id)
      return
    }
    if (node.kind !== 'split') throw new WorkspaceLayoutError('unknown layout node kind')
    if (node.axis !== 'horizontal' && node.axis !== 'vertical') throw new WorkspaceLayoutError('layout split axis is invalid')
    if (parentAxis === node.axis) throw new WorkspaceLayoutError(`nested ${node.axis} splits must be flattened`)
    if (node.children.length < 2 || node.children.length > MAX_WORKSPACE_WIDGETS) {
      throw new WorkspaceLayoutError('layout split must have between 2 and 64 children')
    }
    if (node.children.length !== node.weights.length) throw new WorkspaceLayoutError('layout weights must match children')
    let weightSum = 0
    for (const weight of node.weights) {
      if (!Number.isFinite(weight) || weight <= 0) throw new WorkspaceLayoutError('layout weights must be finite and positive')
      weightSum += weight
    }
    if (Math.abs(weightSum - 1) > WEIGHT_EPSILON) throw new WorkspaceLayoutError('layout weights must sum to 1')
    for (const child of node.children) visit(child, depth + 1, node.axis)
  }
  visit(view.layout_root, 1)
  if (leaves.size !== widgetIDs.size) throw new WorkspaceLayoutError('layout leaves must match widgets exactly')
  for (const widgetID of widgetIDs) {
    if (!leaves.has(widgetID)) throw new WorkspaceLayoutError(`layout is missing widget "${widgetID}"`)
  }
}

export function dockWorkspaceView(
  view: ViewDefinition,
  source: WorkspaceDockSource,
  intent: WorkspaceDockIntent,
): ViewDefinition {
  validateWorkspaceView(view)
  if (source.kind === 'existing') {
    if (!view.widgets.some((widget) => widget.id === source.widgetID)) {
      throw new WorkspaceLayoutError(`widget "${source.widgetID}" is not in the view`)
    }
    if (intent.kind === 'panel-edge' && intent.targetWidgetID === source.widgetID) return view
    if (intent.kind === 'split-gap' && (intent.beforeWidgetID === source.widgetID || intent.afterWidgetID === source.widgetID)) {
      return view
    }
    const reordered = reorderExistingDirectChild(view, source.widgetID, intent)
    if (reordered) return reordered
  } else if (view.widgets.some((widget) => widget.id === source.widget.id)) {
    throw new WorkspaceLayoutError(`widget "${source.widget.id}" already exists`)
  }

  const sourceWidget = source.kind === 'new'
    ? source.widget
    : view.widgets.find((widget) => widget.id === source.widgetID)!
  let root = view.layout_root
  if (source.kind === 'existing' && root) root = removeWorkspaceLeaf(root, source.widgetID)

  if (!root) {
    if (source.kind === 'existing') return view
    if (intent.kind !== 'empty-workspace') {
      throw new WorkspaceLayoutError('only empty-workspace intent is valid for an empty view')
    }
    const next = {
      ...view,
      widgets: [...view.widgets, sourceWidget],
      layout_root: createWorkspaceLeaf(sourceWidget.id),
    }
    validateWorkspaceView(next)
    return next
  }

  const sourceLeaf = createWorkspaceLeaf(sourceWidget.id)
  switch (intent.kind) {
    case 'empty-workspace':
      throw new WorkspaceLayoutError('empty-workspace intent is invalid for a non-empty view')
    case 'root-edge':
      root = insertAtRootEdge(root, sourceLeaf, intent.side)
      break
    case 'panel-edge':
      root = insertBesideWidget(root, sourceLeaf, intent.targetWidgetID, intent.side)
      break
    case 'split-gap': {
      const gap = findGapByAnchors(root, intent.beforeWidgetID, intent.afterWidgetID)
      if (!gap) throw new WorkspaceLayoutError('target divider no longer exists')
      root = insertAtSplitGap(root, gap.parentPath, gap.insertionIndex, sourceLeaf)
      break
    }
  }
  root = normalizeWorkspaceLayout(root)
  const next = {
    ...view,
    widgets: source.kind === 'new' ? [...view.widgets, sourceWidget] : view.widgets,
    layout_root: root,
  }
  validateWorkspaceView(next)
  return next
}

export function addWorkspaceWidget(view: ViewDefinition, widget: ViewWidget): ViewDefinition {
  return dockWorkspaceView(
    view,
    { kind: 'new', widget },
    view.layout_root ? { kind: 'root-edge', side: 'right' } : { kind: 'empty-workspace' },
  )
}

export function removeWorkspaceWidget(view: ViewDefinition, widgetID: string): ViewDefinition {
  validateWorkspaceView(view)
  if (!view.widgets.some((widget) => widget.id === widgetID)) return view
  const layoutRoot = view.layout_root ? removeWorkspaceLeaf(view.layout_root, widgetID) : undefined
  const next: ViewDefinition = {
    ...view,
    widgets: view.widgets.filter((widget) => widget.id !== widgetID),
    layout_root: layoutRoot,
  }
  validateWorkspaceView(next)
  return next
}

export function resizeWorkspaceSplitPair(
  root: ViewLayoutNode,
  splitPath: number[],
  dividerIndex: number,
  pairRatio: number,
  minimumRatio = 0,
  maximumRatio = 1,
): ViewLayoutNode {
  if (!Number.isFinite(pairRatio)) throw new WorkspaceLayoutError('split ratio must be finite')
  if (!Number.isFinite(minimumRatio) || !Number.isFinite(maximumRatio) || minimumRatio < 0 || maximumRatio > 1 || minimumRatio >= maximumRatio) {
    throw new WorkspaceLayoutError('split ratio bounds are invalid')
  }
  const split = nodeAtPath(root, splitPath)
  if (split.kind !== 'split') throw new WorkspaceLayoutError('resize target is not a split')
  if (dividerIndex < 0 || dividerIndex >= split.children.length - 1) throw new WorkspaceLayoutError('resize divider index is invalid')
  const ratio = Math.min(maximumRatio, Math.max(minimumRatio, pairRatio))
  const pairTotal = split.weights[dividerIndex]! + split.weights[dividerIndex + 1]!
  const weights = [...split.weights]
  weights[dividerIndex] = pairTotal * ratio
  weights[dividerIndex + 1] = pairTotal * (1 - ratio)
  return replaceNodeAtPath(root, splitPath, { ...split, weights: normalizeWeights(weights) })
}

export function workspaceLayoutMinimumSize(
  node: ViewLayoutNode,
  leafWidth = DEFAULT_WORKSPACE_LEAF_WIDTH,
  leafHeight = DEFAULT_WORKSPACE_LEAF_HEIGHT,
  separatorSize = WORKSPACE_SEPARATOR_SIZE,
): WorkspaceLayoutSize {
  if (node.kind === 'leaf') return { width: leafWidth, height: leafHeight }
  const children = node.children.map((child) => workspaceLayoutMinimumSize(child, leafWidth, leafHeight, separatorSize))
  const separators = separatorSize * (children.length - 1)
  if (node.axis === 'horizontal') {
    return {
      width: children.reduce((total, child) => total + child.width, separators),
      height: Math.max(...children.map((child) => child.height)),
    }
  }
  return {
    width: Math.max(...children.map((child) => child.width)),
    height: children.reduce((total, child) => total + child.height, separators),
  }
}

export function firstWorkspaceLeafID(node: ViewLayoutNode): string {
  return node.kind === 'leaf' ? node.widget_id : firstWorkspaceLeafID(node.children[0]!)
}

export function lastWorkspaceLeafID(node: ViewLayoutNode): string {
  return node.kind === 'leaf' ? node.widget_id : lastWorkspaceLeafID(node.children.at(-1)!)
}

export function normalizeWorkspaceLayout(node: ViewLayoutNode): ViewLayoutNode {
  if (node.kind === 'leaf') return { ...node }
  const children: ViewLayoutNode[] = []
  const weights: number[] = []
  for (let index = 0; index < node.children.length; index += 1) {
    const child = normalizeWorkspaceLayout(node.children[index]!)
    const parentWeight = node.weights[index]!
    if (child.kind === 'split' && child.axis === node.axis) {
      child.children.forEach((grandchild, childIndex) => {
        children.push(grandchild)
        weights.push(parentWeight * child.weights[childIndex]!)
      })
    } else {
      children.push(child)
      weights.push(parentWeight)
    }
  }
  if (children.length === 1) return children[0]!
  return { kind: 'split', axis: node.axis, children, weights: normalizeWeights(weights) }
}

export function findWorkspaceLeafPath(root: ViewLayoutNode, widgetID: string): number[] | undefined {
  if (root.kind === 'leaf') return root.widget_id === widgetID ? [] : undefined
  for (let index = 0; index < root.children.length; index += 1) {
    const childPath = findWorkspaceLeafPath(root.children[index]!, widgetID)
    if (childPath) return [index, ...childPath]
  }
  return undefined
}

function reorderExistingDirectChild(
  view: ViewDefinition,
  sourceWidgetID: string,
  intent: WorkspaceDockIntent,
): ViewDefinition | undefined {
  const root = view.layout_root
  if (!root) return undefined
  const sourcePath = findWorkspaceLeafPath(root, sourceWidgetID)
  if (!sourcePath || sourcePath.length === 0) return undefined

  if (intent.kind === 'panel-edge') {
    const targetPath = findWorkspaceLeafPath(root, intent.targetWidgetID)
    if (!targetPath || targetPath.length !== sourcePath.length) return undefined
    const sourceParentPath = sourcePath.slice(0, -1)
    if (!pathsEqual(sourceParentPath, targetPath.slice(0, -1))) return undefined
    const parent = nodeAtPath(root, sourceParentPath)
    if (parent.kind !== 'split' || parent.axis !== workspaceDockAxis(intent.side)) return undefined
    const sourceIndex = sourcePath.at(-1)!
    const targetIndex = targetPath.at(-1)!
    const insertionIndex = targetIndex + (workspaceDockBefore(intent.side) ? 0 : 1)
    return reorderDirectChild(view, sourceParentPath, sourceIndex, insertionIndex)
  }

  if (intent.kind === 'split-gap') {
    if (sourceWidgetID === intent.beforeWidgetID || sourceWidgetID === intent.afterWidgetID) return view
    const sourceParentPath = sourcePath.slice(0, -1)
    if (!pathsEqual(sourceParentPath, intent.parentPath)) return undefined
    return reorderDirectChild(view, sourceParentPath, sourcePath.at(-1)!, intent.insertionIndex)
  }

  if (intent.kind === 'root-edge' && sourcePath.length === 1 && root.kind === 'split' && root.axis === workspaceDockAxis(intent.side)) {
    return reorderDirectChild(view, [], sourcePath[0]!, workspaceDockBefore(intent.side) ? 0 : root.children.length)
  }
  return undefined
}

function reorderDirectChild(view: ViewDefinition, parentPath: number[], sourceIndex: number, rawInsertionIndex: number): ViewDefinition {
  const root = view.layout_root!
  const parent = nodeAtPath(root, parentPath)
  if (parent.kind !== 'split') throw new WorkspaceLayoutError('reorder parent is not a split')
  let insertionIndex = rawInsertionIndex
  if (sourceIndex < insertionIndex) insertionIndex -= 1
  if (insertionIndex === sourceIndex) return view
  if (sourceIndex < 0 || sourceIndex >= parent.children.length || insertionIndex < 0 || insertionIndex >= parent.children.length) {
    throw new WorkspaceLayoutError('reorder index is invalid')
  }
  const children = [...parent.children]
  const weights = [...parent.weights]
  const [child] = children.splice(sourceIndex, 1)
  const [weight] = weights.splice(sourceIndex, 1)
  children.splice(insertionIndex, 0, child!)
  weights.splice(insertionIndex, 0, weight!)
  const next = { ...view, layout_root: replaceNodeAtPath(root, parentPath, { ...parent, children, weights }) }
  validateWorkspaceView(next)
  return next
}

function removeWorkspaceLeaf(root: ViewLayoutNode, widgetID: string): ViewLayoutNode | undefined {
  if (root.kind === 'leaf') return root.widget_id === widgetID ? undefined : root
  let removed = false
  const children: ViewLayoutNode[] = []
  const weights: number[] = []
  for (let index = 0; index < root.children.length; index += 1) {
    const child = root.children[index]!
    const nextChild = removed ? child : removeWorkspaceLeaf(child, widgetID)
    if (nextChild !== child) removed = true
    if (nextChild) {
      children.push(nextChild)
      weights.push(root.weights[index]!)
    }
  }
  if (!removed) return root
  if (children.length === 0) return undefined
  if (children.length === 1) return children[0]
  return normalizeWorkspaceLayout({ ...root, children, weights: normalizeWeights(weights) })
}

function insertAtRootEdge(root: ViewLayoutNode, source: ViewLayoutNode, side: WorkspaceDockSide): ViewLayoutNode {
  const axis = workspaceDockAxis(side)
  const before = workspaceDockBefore(side)
  if (root.kind === 'split' && root.axis === axis) {
    return insertSibling(root, before ? 0 : root.children.length, before ? 0 : root.children.length - 1, source)
  }
  return {
    kind: 'split',
    axis,
    children: before ? [source, root] : [root, source],
    weights: [0.5, 0.5],
  }
}

function insertBesideWidget(
  root: ViewLayoutNode,
  source: ViewLayoutNode,
  targetWidgetID: string,
  side: WorkspaceDockSide,
): ViewLayoutNode {
  const targetPath = findWorkspaceLeafPath(root, targetWidgetID)
  if (!targetPath) throw new WorkspaceLayoutError(`target widget "${targetWidgetID}" is not in the layout`)
  const axis = workspaceDockAxis(side)
  const before = workspaceDockBefore(side)
  if (targetPath.length > 0) {
    const parentPath = targetPath.slice(0, -1)
    const parent = nodeAtPath(root, parentPath)
    if (parent.kind === 'split' && parent.axis === axis) {
      const targetIndex = targetPath.at(-1)!
      return replaceNodeAtPath(root, parentPath, insertSibling(parent, targetIndex + (before ? 0 : 1), targetIndex, source))
    }
  }
  const target = nodeAtPath(root, targetPath)
  const replacement: ViewLayoutNode = {
    kind: 'split',
    axis,
    children: before ? [source, target] : [target, source],
    weights: [0.5, 0.5],
  }
  return replaceNodeAtPath(root, targetPath, replacement)
}

function insertAtSplitGap(root: ViewLayoutNode, parentPath: number[], insertionIndex: number, source: ViewLayoutNode): ViewLayoutNode {
  const parent = nodeAtPath(root, parentPath)
  if (parent.kind !== 'split') throw new WorkspaceLayoutError('divider parent is not a split')
  if (insertionIndex <= 0 || insertionIndex >= parent.children.length) throw new WorkspaceLayoutError('divider insertion index is invalid')
  return replaceNodeAtPath(root, parentPath, insertSibling(parent, insertionIndex, insertionIndex, source))
}

function insertSibling(
  split: Extract<ViewLayoutNode, { kind: 'split' }>,
  insertionIndex: number,
  splitWeightIndex: number,
  source: ViewLayoutNode,
): ViewLayoutNode {
  if (insertionIndex < 0 || insertionIndex > split.children.length) throw new WorkspaceLayoutError('sibling insertion index is invalid')
  if (splitWeightIndex < 0 || splitWeightIndex >= split.children.length) throw new WorkspaceLayoutError('target weight index is invalid')
  const children = [...split.children]
  children.splice(insertionIndex, 0, source)
  let weights: number[]
  if (weightsAreUniform(split.weights)) {
    weights = Array.from({ length: children.length }, () => 1 / children.length)
  } else {
    weights = [...split.weights]
    const half = weights[splitWeightIndex]! / 2
    weights[splitWeightIndex] = half
    weights.splice(insertionIndex, 0, half)
    weights = normalizeWeights(weights)
  }
  return { ...split, children, weights }
}

function findGapByAnchors(
  root: ViewLayoutNode,
  beforeWidgetID: string,
  afterWidgetID: string,
): { parentPath: number[]; insertionIndex: number } | undefined {
  const beforePath = findWorkspaceLeafPath(root, beforeWidgetID)
  const afterPath = findWorkspaceLeafPath(root, afterWidgetID)
  if (!beforePath || !afterPath) return undefined
  let commonLength = 0
  while (commonLength < beforePath.length && commonLength < afterPath.length && beforePath[commonLength] === afterPath[commonLength]) {
    commonLength += 1
  }
  if (commonLength >= beforePath.length || commonLength >= afterPath.length) return undefined
  const beforeIndex = beforePath[commonLength]!
  const afterIndex = afterPath[commonLength]!
  if (afterIndex !== beforeIndex + 1) return undefined
  const parentPath = beforePath.slice(0, commonLength)
  return { parentPath, insertionIndex: afterIndex }
}

function nodeAtPath(root: ViewLayoutNode, path: number[]): ViewLayoutNode {
  let node = root
  for (const segment of path) {
    if (node.kind !== 'split' || segment < 0 || segment >= node.children.length) {
      throw new WorkspaceLayoutError('layout path is invalid')
    }
    node = node.children[segment]!
  }
  return node
}

function replaceNodeAtPath(root: ViewLayoutNode, path: number[], replacement: ViewLayoutNode): ViewLayoutNode {
  if (path.length === 0) return replacement
  if (root.kind !== 'split') throw new WorkspaceLayoutError('layout path does not target a child')
  const [segment, ...rest] = path
  if (segment === undefined || segment < 0 || segment >= root.children.length) throw new WorkspaceLayoutError('layout path is invalid')
  const children = [...root.children]
  children[segment] = replaceNodeAtPath(children[segment]!, rest, replacement)
  return { ...root, children }
}

function normalizeWeights(weights: number[]): number[] {
  if (weights.length === 0 || weights.some((weight) => !Number.isFinite(weight) || weight <= 0)) {
    throw new WorkspaceLayoutError('weights must be finite and positive')
  }
  const total = weights.reduce((sum, weight) => sum + weight, 0)
  const normalized = weights.map((weight) => weight / total)
  const prefix = normalized.slice(0, -1).reduce((sum, weight) => sum + weight, 0)
  normalized[normalized.length - 1] = 1 - prefix
  return normalized
}

function weightsAreUniform(weights: number[]): boolean {
  const expected = 1 / weights.length
  return weights.every((weight) => Math.abs(weight - expected) <= WEIGHT_EPSILON)
}

function pathsEqual(left: number[], right: number[]): boolean {
  return left.length === right.length && left.every((segment, index) => segment === right[index])
}
