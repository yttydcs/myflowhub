import type { DesktopAPI } from '../api'
import { errorText } from '../lib/utils'
import type { ResourceCatalog, ResourceDescriptor, Topology, TopologyNode, TopologyQuery } from '../types'
import { MAX_TOPOLOGY_NODES, parseTopology, topologyCoverage, validateTopologyRequest } from './topology'

export const DISCOVERY_TTL = 30_000
export const MAX_CATALOG_OWNERS = 128
export type LoadState = {
  status: 'unloaded' | 'loading' | 'loaded' | 'error'
  stale: boolean
  updatedAt?: number
  error?: string
  errorCode?: string
  errorAt?: number
}
export type ChildrenState = LoadState & {
  children: string[]
  instance?: string
  revision?: number
  requestSequence: number
}
export type CatalogState = LoadState & { catalog?: ResourceCatalog; used: number }
export type DiscoverySnapshot = {
  session: number
  scope: string
  topology: Topology
  children: ReadonlyMap<string, ChildrenState>
  catalogs: ReadonlyMap<string, CatalogState>
  resources: ResourceDescriptor[]
  limitError?: string
}

const unloaded = (): LoadState => ({ status: 'unloaded', stale: false })
const emptyChildren = (): ChildrenState => ({ ...unloaded(), children: [], requestSequence: 0 })

export function discoveryFailure(error: unknown): Pick<LoadState, 'error' | 'errorCode'> {
  const text = errorText(error)
  const code = /forbidden|denied|permission/i.test(text) ? 'Forbidden'
    : /unsupported|not.?found|不支持/i.test(text) ? 'Unsupported'
      : /unreachable|timeout|deadline|disconnected|closed|no.route/i.test(text) ? 'Unreachable'
        : /overflow|上限|超过.*4096/i.test(text) ? 'Overflow' : 'Error'
  const label = { Forbidden: '无权访问', Unsupported: '不支持此查询，请确认节点版本与能力', Unreachable: '节点不可达或请求超时', Overflow: '超过缓存或查询上限', Error: '加载失败' }[code]
  return { errorCode: code, error: `${label}：${text}` }
}

type Job = { valid(): boolean; run(): Promise<void>; finish(): void }

// One scheduler survives scope resets: even obsolete, uncancellable Wails calls count
// towards the shared limit until they settle.
export class DiscoveryController {
  private epoch = 0
  private key = ''
  private root = ''
  private serial = 0
  private clock = 0
  private running = 0
  private queue: Job[] = []
  private listeners = new Set<() => void>()
  private nodes = new Map<string, TopologyNode>()
  private publishedNodes?: Map<string, TopologyNode>
  private children = new Map<string, ChildrenState>()
  private catalogs = new Map<string, CatalogState>()
  private pinned = new Set<string>()
  private applied = new Map<string, number>()
  private inFlight = new Map<string, { sequence: number; promise: Promise<void> }>()
  private catalogSequences = new Map<string, number>()
  private limitError?: string
  private snapshot: DiscoverySnapshot = { session: 0, scope: '', topology: { nodes: [] }, children: new Map(), catalogs: new Map(), resources: [] }

  constructor(private api: Pick<DesktopAPI, 'topology' | 'catalog'>, private now = Date.now) {}

  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener) } }
  getSnapshot = () => this.snapshot

  activate(profile: string, connection: number, root: string): void {
    validateTopologyRequest(root, 1)
    const key = JSON.stringify([profile, connection, root])
    if (key === this.key) return
    this.reset()
    this.key = key
    this.root = root
    this.children.set(root, emptyChildren())
    this.publish()
  }

  reset(): void {
    this.epoch += 1
    this.key = ''
    this.root = ''
    this.nodes = new Map()
    this.children.clear()
    this.catalogs.clear()
    this.pinned.clear()
    this.applied.clear()
    this.inFlight.clear()
    this.catalogSequences.clear()
    this.limitError = undefined
    this.queue.splice(0).forEach((job) => job.finish())
    this.publish()
  }

  private publish(): void {
    this.snapshot = {
      session: this.epoch,
      scope: this.root,
      topology: this.publishedNodes === this.nodes ? this.snapshot.topology : { nodes: [...this.nodes.values()] },
      children: new Map(this.children),
      catalogs: new Map(this.catalogs),
      resources: [...this.catalogs.values()].flatMap((entry) => entry.catalog?.resources || []),
      limitError: this.limitError,
    }
    this.publishedNodes = this.nodes
    this.listeners.forEach((listener) => listener())
  }

  private schedule(valid: () => boolean, run: () => Promise<void>): Promise<void> {
    return new Promise((finish) => {
      this.queue.push({ valid, run, finish })
      this.drain()
    })
  }

  private drain(): void {
    while (this.running < 4 && this.queue.length) {
      const job = this.queue.shift()!
      if (!job.valid()) { job.finish(); continue }
      this.running += 1
      void Promise.resolve().then(() => job.valid() ? job.run() : undefined).finally(() => { this.running -= 1; job.finish(); this.drain() })
    }
  }

  query(owner = this.root, depth = 1, force = false): Promise<void> {
    if (!this.root || (owner !== this.root && !this.nodes.has(owner))) return Promise.resolve()
    validateTopologyRequest(owner, depth)
    const requestKey = `tree:${owner}:${depth}`
    const pending = this.inFlight.get(requestKey)
    if (pending) return pending.promise
    const previous = this.children.get(owner) || emptyChildren()
    if (!force && depth === 1 && previous.status === 'loaded' && previous.updatedAt !== undefined && this.now() - previous.updatedAt < DISCOVERY_TTL) return Promise.resolve()
    const epoch = this.epoch
    const sequence = ++this.serial
    this.children.set(owner, { ...previous, requestSequence: sequence, status: 'loading', stale: previous.updatedAt !== undefined, error: undefined, errorCode: undefined })
    const valid = () => epoch === this.epoch && this.children.get(owner)?.requestSequence === sequence
    const promise = this.schedule(valid, async () => {
      try {
        const response = parseTopology(await this.api.topology(owner, depth), owner, depth)
        if (!valid()) return
        const current = this.children.get(owner)!
        if (current.instance === response.instance_id && current.revision !== undefined && response.revision < current.revision) {
          this.children.set(owner, { ...current, status: current.updatedAt === undefined ? 'unloaded' : 'loaded', stale: true })
        } else if (!this.merge(response, sequence)) {
          this.children.set(owner, { ...current, status: current.updatedAt === undefined ? 'unloaded' : 'loaded', stale: true })
        }
      } catch (error) {
        if (valid()) this.children.set(owner, { ...this.children.get(owner)!, status: 'error', ...discoveryFailure(error), errorAt: this.now() })
      }
    }).finally(() => {
      if (this.inFlight.get(requestKey)?.sequence === sequence) this.inFlight.delete(requestKey)
      if (epoch === this.epoch) this.publish()
    })
    this.inFlight.set(requestKey, { sequence, promise })
    this.publish()
    return promise
  }

  private merge(response: TopologyQuery, sequence: number): boolean {
    const coverage = topologyCoverage(response)
    for (const node of response.nodes) if (!node.has_children) coverage.set(node.node_id, [])
    const childSets = new Map([...coverage].map(([id, children]) => [id, new Set(children)]))
    // A newer query may have changed overlapping membership while this call waited.
    // Do not compare revisions from different providers to resolve that race.
    for (const id of coverage.keys()) if ((this.applied.get(id) || 0) > sequence) return false
    const next = new Map(this.nodes)
    const incoming = new Set(response.nodes.map((node) => node.node_id))
    for (const node of response.nodes) {
      const old = next.get(node.node_id)
      const parent = node.node_id === response.root_node_id ? old?.parent_id : node.parent_id
      if (old?.parent_id !== parent && (this.applied.get(node.node_id) || 0) > sequence) return false
      const merged = {
        ...node,
        parent_id: parent,
        // Ancestors may only know membership, while an owner's own query can
        // authoritatively clear its name. Names do not change edge authority.
        display_name: node.display_name ?? (node.node_id === response.root_node_id ? undefined : old?.display_name),
      }
      next.set(node.node_id, old && old.parent_id === merged.parent_id && old.display_name === merged.display_name
        && old.role === merged.role && old.generation === merged.generation && old.has_children === merged.has_children ? old : merged)
    }
    // Detach only complete child lists. Reparented incoming nodes retain their cached
    // descendants; reachability below decides which branches actually left the scope.
    for (const node of next.values()) {
      if (node.parent_id && childSets.has(node.parent_id) && !childSets.get(node.parent_id)!.has(node.node_id)) {
        if ((this.applied.get(node.node_id) || 0) > sequence) return false
        next.delete(node.node_id)
      }
    }
    if (next.get(this.root)?.parent_id) throw new Error('topology 合并将使浏览起点成为后代')
    const edges = new Map<string, string[]>()
    for (const node of next.values()) if (node.parent_id) {
      const ids = edges.get(node.parent_id) || []
      ids.push(node.node_id)
      edges.set(node.parent_id, ids)
    }
    const reachable = new Set<string>()
    const queue = [this.root]
    for (let cursor = 0; cursor < queue.length; cursor += 1) {
      const id = queue[cursor]!
      if (reachable.has(id)) throw new Error('topology 合并产生环')
      if (!next.has(id)) continue
      reachable.add(id)
      queue.push(...(edges.get(id) || []))
    }
    if ([...incoming].some((id) => !reachable.has(id))) throw new Error('topology 合并产生不可达节点或环')
    if (reachable.size > MAX_TOPOLOGY_NODES) throw new Error('导航缓存超过 4096 节点上限；请选择更小的浏览起点')
    const invalidated = [...this.nodes.keys()].filter((id) => !reachable.has(id) || this.nodes.get(id)?.parent_id !== next.get(id)?.parent_id)
    const invalidIDs = new Set(invalidated)
    // A moved ancestor also invalidates requests issued under its former path.
    for (let cursor = 0; cursor < invalidated.length; cursor += 1) {
      for (const id of this.children.get(invalidated[cursor]!)?.children || []) {
        if (!invalidIDs.has(id)) { invalidIDs.add(id); invalidated.push(id) }
      }
    }
    for (const id of invalidIDs) {
      if (this.nodes.has(id)) {
        this.children.delete(id)
        this.applied.delete(id)
        for (const key of this.inFlight.keys()) if (key.startsWith(`tree:${id}:`)) this.inFlight.delete(key)
      }
    }
    this.nodes = new Map([...next].filter(([id]) => reachable.has(id)))
    for (const [id, node] of this.nodes) {
      const entry = this.children.get(id) || emptyChildren()
      const ids = edges.get(id) || []
      if (coverage.has(id)) {
        this.applied.set(id, sequence)
        this.children.set(id, { ...entry, status: 'loaded', children: ids, stale: false, updatedAt: this.now(), error: undefined, errorCode: undefined })
      } else {
        this.children.set(id, { ...entry, children: ids })
      }
      if (incoming.has(id)) this.applied.set(id, Math.max(sequence, this.applied.get(id) || 0))
      if (ids.length && !node.has_children) this.nodes.set(id, { ...node, has_children: true })
    }
    const owner = this.children.get(response.root_node_id)!
    this.children.set(response.root_node_id, { ...owner, instance: response.instance_id, revision: response.revision })
    return true
  }

  private expandedOwners(expanded: readonly string[], focused?: string): string[] {
    const desired = new Set(expanded)
    const queue = [focused && this.nodes.has(focused) ? focused : this.root]
    const owners: string[] = []
    for (let cursor = 0; cursor < queue.length; cursor += 1) {
      const id = queue[cursor]!
      if (!desired.has(id)) continue
      const entry = this.children.get(id)
      if (this.nodes.get(id)?.has_children) owners.push(id)
      queue.push(...(entry?.children || []))
    }
    return owners
  }

  restoreExpanded(expanded: readonly string[], focused?: string): void {
    for (const id of this.expandedOwners(expanded, focused)) {
      if (this.children.get(id)?.status === 'unloaded') void this.query(id)
    }
  }

  setCatalogOwners(owners: Iterable<string>): void {
    this.pinned = new Set(owners)
    const message = this.pinned.size > MAX_CATALOG_OWNERS ? '当前选择与 View 超过 128 个目录 owner 上限；请减少活动 View 引用' : undefined
    if (message !== this.limitError) { this.limitError = message; this.publish() }
  }

  catalog(owner: string, force = false): Promise<void> {
    if (!this.root) return Promise.resolve()
    validateTopologyRequest(owner, 1)
    const key = `catalog:${owner}`
    const pending = this.inFlight.get(key)
    if (pending) return pending.promise
    const previous = this.catalogs.get(owner)
    if (!force && previous?.status === 'loaded' && previous.updatedAt !== undefined && this.now() - previous.updatedAt < DISCOVERY_TTL) {
      this.catalogs.set(owner, { ...previous, used: ++this.clock })
      return Promise.resolve()
    }
    if (!previous && this.catalogs.size >= MAX_CATALOG_OWNERS) {
      const victim = [...this.catalogs].filter(([id, state]) => !this.pinned.has(id) && state.status !== 'loading').sort((a, b) => a[1].used - b[1].used)[0]
      if (!victim) { this.limitError = '目录缓存已达 128 owner 上限；请减少活动 View 引用或稍后重试'; this.publish(); return Promise.resolve() }
      this.catalogs.delete(victim[0])
      this.catalogSequences.delete(victim[0])
    }
    const epoch = this.epoch
    const sequence = ++this.serial
    this.catalogSequences.set(owner, sequence)
    this.catalogs.set(owner, { ...(previous || unloaded()), used: ++this.clock, status: 'loading', stale: previous?.catalog !== undefined, error: undefined, errorCode: undefined })
    const valid = () => epoch === this.epoch && this.catalogSequences.get(owner) === sequence
    const promise = this.schedule(valid, async () => {
      try {
        const catalog = await this.api.catalog(owner)
        if (!valid()) return
        if (!catalog || !Array.isArray(catalog.resources) || catalog.resources.some((resource) => resource.id?.owner_node_id !== owner)) throw new Error('catalog 响应包含无效或其他 owner 的资源')
        this.catalogs.set(owner, { status: 'loaded', stale: false, updatedAt: this.now(), catalog, used: ++this.clock })
      } catch (error) {
        if (valid()) this.catalogs.set(owner, { ...this.catalogs.get(owner)!, status: 'error', ...discoveryFailure(error), errorAt: this.now() })
      }
    }).finally(() => {
      if (this.inFlight.get(key)?.sequence === sequence) this.inFlight.delete(key)
      if (epoch === this.epoch) this.publish()
    })
    this.inFlight.set(key, { sequence, promise })
    this.publish()
    return promise
  }

  async refresh(expanded: readonly string[], focused?: string): Promise<void> {
    const epoch = this.epoch
    await this.query(this.root, 1, true)
    if (epoch !== this.epoch) return
    const known = this.expandedOwners(expanded, focused).filter((id) => id !== this.root)
    await Promise.all([...known.map((id) => this.query(id, 1, true)), ...[...this.pinned].map((id) => this.catalog(id, true))])
  }
}
