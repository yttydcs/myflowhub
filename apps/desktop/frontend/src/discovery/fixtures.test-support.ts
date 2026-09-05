import type { TopologyNode, TopologyQuery } from '../types'

export function node(id: string, parent?: string, hasChildren = false): TopologyNode {
  return { node_id: id, parent_id: parent, has_children: hasChildren, display_name: `Node ${id}`, role: parent ? 'child' : 'root', generation: 1 }
}

export function query(owner: string, nodes: TopologyNode[], depth = 1, revision = 1, instance = 'a'.repeat(32)): TopologyQuery {
  return { version: 1, root_node_id: owner, depth, instance_id: instance, revision, nodes }
}

export function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}
