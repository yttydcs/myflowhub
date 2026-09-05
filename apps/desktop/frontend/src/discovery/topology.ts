import type { TopologyNode, TopologyQuery } from '../types'

export const MAX_TOPOLOGY_NODES = 4096
export const TOPOLOGY_QUERY_SCHEMA = 'mfh.management.topology-query.v1'

export function validateTopologyRequest(owner: string, depth: number): void {
  if (!validNodeID(owner)) throw new Error('topology owner 必须是正整数 Node ID 字符串')
  if (!Number.isInteger(depth) || depth < 0 || depth > MAX_TOPOLOGY_NODES) throw new Error('topology depth 必须是 0..4096 的整数')
}

function validNodeID(value: unknown): value is string {
  return typeof value === 'string' && /^[1-9][0-9]{0,19}$/.test(value) && BigInt(value) <= 18446744073709551615n
}

function record(value: unknown, fields: string[], label: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${label} 必须是对象`)
  const result = value as Record<string, unknown>
  if (Object.keys(result).some((key) => !fields.includes(key))) throw new Error(`${label} 包含未知字段`)
  return result
}

export function topologyCoverage(query: TopologyQuery): Map<string, string[]> {
  const children = new Map(query.nodes.map((node) => [node.node_id, [] as string[]]))
  for (const node of query.nodes) if (node.parent_id) children.get(node.parent_id)!.push(node.node_id)
  const covered = new Map<string, string[]>()
  const queue = [{ id: query.root_node_id, distance: 0 }]
  for (let cursor = 0; cursor < queue.length; cursor += 1) {
    const { id, distance } = queue[cursor]!
    const ids = children.get(id)!
    if (query.depth === 0 || distance < query.depth) covered.set(id, ids)
    for (const child of ids) queue.push({ id: child, distance: distance + 1 })
  }
  return covered
}

export function parseTopology(value: unknown, owner: string, depth: number): TopologyQuery {
  validateTopologyRequest(owner, depth)
  if (new TextEncoder().encode(JSON.stringify(value)).byteLength > 1 << 20) throw new Error('topology payload 超过 1 MiB 上限')
  const result = record(value, ['version', 'root_node_id', 'depth', 'instance_id', 'revision', 'nodes'], 'topology')
  if (result.version !== 1 || result.root_node_id !== owner || result.depth !== depth) throw new Error('topology version/root/depth 与请求不一致')
  if (typeof result.instance_id !== 'string' || !/^[0-9a-f]{32}$/.test(result.instance_id)) throw new Error('topology instance_id 必须是 32 位小写十六进制')
  if (!Number.isSafeInteger(result.revision) || (result.revision as number) < 1) throw new Error('topology revision 必须是安全正整数')
  if (!Array.isArray(result.nodes) || result.nodes.length === 0 || result.nodes.length > MAX_TOPOLOGY_NODES) throw new Error('topology nodes 必须有 1..4096 项')
  const nodes = result.nodes.map((value): TopologyNode => {
    const node = record(value, ['node_id', 'parent_id', 'display_name', 'role', 'generation', 'has_children'], 'topology node')
    if (!validNodeID(node.node_id) || (node.parent_id !== undefined && !validNodeID(node.parent_id))) throw new Error('topology Node ID 无效')
    if (typeof node.role !== 'string' || !node.role || new TextEncoder().encode(node.role).length > 128) throw new Error('topology role 无效')
    if (node.display_name !== undefined && (typeof node.display_name !== 'string' || new TextEncoder().encode(node.display_name).length > 255)) throw new Error('topology display_name 无效')
    if (!Number.isInteger(node.generation) || (node.generation as number) <= 0 || (node.generation as number) > Number(18446744073709551615n)) throw new Error('topology generation 无效')
    if (typeof node.has_children !== 'boolean') throw new Error('topology has_children 必须是布尔值')
    return { ...node } as TopologyNode
  })
  const byID = new Map(nodes.map((node) => [node.node_id, node]))
  if (byID.size !== nodes.length || !byID.has(owner)) throw new Error('topology 重复 Node ID 或缺少根')
  const children = new Map(nodes.map((node) => [node.node_id, [] as string[]]))
  for (const node of nodes) {
    if (node.node_id === owner) {
      if (node.parent_id !== undefined) throw new Error('topology 查询根不能带 parent_id')
    } else {
      if (!node.parent_id || !byID.has(node.parent_id)) throw new Error('topology parent 必须在响应内')
      children.get(node.parent_id)!.push(node.node_id)
    }
  }
  const queue = [{ id: owner, distance: 0 }]
  const visited = new Set<string>()
  for (let cursor = 0; cursor < queue.length; cursor += 1) {
    const { id, distance } = queue[cursor]!
    if (visited.has(id)) throw new Error('topology 存在环')
    visited.add(id)
    if (depth > 0 && distance > depth) throw new Error('topology 超过请求深度')
    const ids = children.get(id)!
    const node = byID.get(id)!
    if ((ids.length > 0 && !node.has_children) || ((depth === 0 || distance < depth) && node.has_children !== (ids.length > 0))) throw new Error('topology has_children 与关系不一致')
    for (const child of ids) queue.push({ id: child, distance: distance + 1 })
  }
  if (visited.size !== nodes.length) throw new Error('topology 存在不可达节点或环')
  return { version: 1, root_node_id: owner, depth, instance_id: result.instance_id, revision: result.revision as number, nodes }
}
