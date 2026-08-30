import type { ResourceDescriptor } from '../types'

export interface ResourcePathNode {
  key: string
  ownerNodeID: string
  path: string
  segment: string
  resource?: ResourceDescriptor
  children: string[]
}

export interface ResourceTreeIndex {
  roots: string[]
  nodesByKey: Map<string, ResourcePathNode>
  parentByKey: Map<string, string | undefined>
}

export interface ResourceTreeRow {
  key: string
  parentKey?: string
  depth: number
  posInSet: number
  setSize: number
  node: ResourcePathNode
}

export function resourcePathKey(ownerNodeID: string, path: string): string {
  return `resource-path:${JSON.stringify([ownerNodeID, path])}`
}

export function buildResourceTree(resources: ResourceDescriptor[]): ResourceTreeIndex {
  const roots: string[] = []
  const nodesByKey = new Map<string, ResourcePathNode>()
  const parentByKey = new Map<string, string | undefined>()

  for (const resource of resources) {
    const { owner_node_id: ownerNodeID, name } = resource.id
    const segments = name.split('/')
    if (!ownerNodeID || !name || segments.some((segment) => segment.length === 0)) {
      throw new Error(`invalid Resource path: ${ownerNodeID}/${name}`)
    }

    let parentKey: string | undefined
    let path = ''
    for (let index = 0; index < segments.length; index += 1) {
      const segment = segments[index]!
      path = path ? `${path}/${segment}` : segment
      const key = resourcePathKey(ownerNodeID, path)
      let node = nodesByKey.get(key)
      if (!node) {
        node = { key, ownerNodeID, path, segment, children: [] }
        nodesByKey.set(key, node)
        parentByKey.set(key, parentKey)
        if (parentKey) nodesByKey.get(parentKey)?.children.push(key)
        else roots.push(key)
      }
      if (index === segments.length - 1) {
        if (node.resource) throw new Error(`duplicate Resource identity: ${ownerNodeID}/${name}`)
        node.resource = resource
      }
      parentKey = key
    }
  }

  const compareKeys = (leftKey: string, rightKey: string) => {
    const left = nodesByKey.get(leftKey)!
    const right = nodesByKey.get(rightKey)!
    return left.ownerNodeID.localeCompare(right.ownerNodeID, undefined, { numeric: true })
      || left.segment.localeCompare(right.segment, undefined, { numeric: true })
      || left.path.localeCompare(right.path, undefined, { numeric: true })
  }
  roots.sort(compareKeys)
  for (const node of nodesByKey.values()) node.children.sort(compareKeys)

  return { roots, nodesByKey, parentByKey }
}

export function defaultExpandedResourcePaths(index: ResourceTreeIndex): string[] {
  return index.roots.filter((key) => (index.nodesByKey.get(key)?.children.length || 0) > 0)
}

export function flattenResourceRows(
  index: ResourceTreeIndex,
  expandedPaths: ReadonlySet<string>,
  rawQuery = '',
): ResourceTreeRow[] {
  const query = rawQuery.trim().toLocaleLowerCase()
  const visibleKeys = query ? buildVisibleResourceKeys(index, query) : undefined
  const roots = index.roots.filter((key) => !visibleKeys || visibleKeys.has(key))
  const rows: ResourceTreeRow[] = []
  const stack: Array<{
    key: string
    parentKey?: string
    depth: number
    posInSet: number
    setSize: number
  }> = roots.slice().reverse().map((key, reverseIndex) => ({
    key,
    parentKey: undefined,
    depth: 1,
    posInSet: roots.length - reverseIndex,
    setSize: roots.length,
  }))

  while (stack.length > 0) {
    const current = stack.pop()!
    const node = index.nodesByKey.get(current.key)
    if (!node) continue
    rows.push({ ...current, node })

    if (!query && !expandedPaths.has(current.key)) continue
    const children = node.children.filter((key) => !visibleKeys || visibleKeys.has(key))
    for (let childIndex = children.length - 1; childIndex >= 0; childIndex -= 1) {
      const key = children[childIndex]
      if (!key) continue
      stack.push({
        key,
        parentKey: current.key,
        depth: current.depth + 1,
        posInSet: childIndex + 1,
        setSize: children.length,
      })
    }
  }
  return rows
}

function buildVisibleResourceKeys(index: ResourceTreeIndex, query: string): Set<string> {
  const visibleKeys = new Set<string>()
  for (const [key, node] of index.nodesByKey) {
    if (resourceSearchValues(node).some((value) => value.includes(query))) visibleKeys.add(key)
  }
  for (const key of [...visibleKeys]) {
    let parentKey = index.parentByKey.get(key)
    while (parentKey && !visibleKeys.has(parentKey)) {
      visibleKeys.add(parentKey)
      parentKey = index.parentByKey.get(parentKey)
    }
  }
  return visibleKeys
}

function resourceSearchValues(node: ResourcePathNode): string[] {
  return [
    node.segment,
    node.path,
    node.resource?.presentation?.label || '',
    node.resource?.type || '',
  ].map((value) => value.toLocaleLowerCase())
}
