import { describe, expect, it, vi } from 'vitest'
import type { ResourceCatalog, ResourceDescriptor, TopologyQuery } from '../types'
import { DiscoveryController, DISCOVERY_TTL } from './controller'
import { deferred, node, query } from './fixtures.test-support'

const emptyCatalog: ResourceCatalog = { version: 2, revision: 1, resources: [] }
const turn = async () => { for (let i = 0; i < 8; i += 1) await Promise.resolve() }

function fixture(now?: () => number) {
  const api = { topology: vi.fn<(owner: string, depth?: number) => Promise<TopologyQuery>>(), catalog: vi.fn<(owner: string) => Promise<ResourceCatalog>>().mockResolvedValue(emptyCatalog) }
  const controller = new DiscoveryController(api, now)
  controller.activate('profile-a', 1, '1')
  const ids = () => controller.getSnapshot().topology.nodes.map((node) => node.node_id)
  return { api, controller, ids }
}

describe('scoped discovery controller', () => {
  it('walks six levels only along known expanded paths and does not load catalogs', async () => {
    const { api, controller, ids } = fixture()
    api.topology.mockImplementation(async (owner) => {
      const id = Number(owner)
      return query(owner, id < 6 ? [node(owner, undefined, true), node(String(id + 1), owner, id < 5)] : [node(owner)])
    })
    await controller.query()
    expect(ids()).toEqual(['1', '2'])
    await controller.query('999')
    controller.restoreExpanded(['1', '5', '999'])
    await turn()
    expect(api.topology).toHaveBeenCalledTimes(1)
    for (let id = 2; id < 6; id += 1) await controller.query(String(id))
    expect(ids()).toEqual(['1', '2', '3', '4', '5', '6'])
    expect(api.topology.mock.calls).toEqual([['1', 1], ['2', 1], ['3', 1], ['4', 1], ['5', 1]])
    expect(api.catalog).not.toHaveBeenCalled()
  })

  it('restores saved expanded paths as they become known without probing unknown IDs', async () => {
    const { api, controller, ids } = fixture()
    api.topology.mockImplementation(async (owner) => query(owner, [node(owner, undefined, true), node(String(Number(owner) + 1), owner, Number(owner) < 5)]))
    const expanded = ['1', '2', '3', '4', '5', '999']
    const stop = controller.subscribe(() => controller.restoreExpanded(expanded))
    await controller.query()
    await turn(); await turn()
    expect(ids()).toEqual(['1', '2', '3', '4', '5', '6'])
    expect(api.topology).toHaveBeenCalledTimes(5)
    stop()
  })

  it('deduplicates double expansion, forced refresh and catalog retry while in flight', async () => {
    const { api, controller } = fixture()
    const tree = deferred<TopologyQuery>()
    const catalog = deferred<ResourceCatalog>()
    api.topology.mockReturnValue(tree.promise)
    api.catalog.mockReturnValue(catalog.promise)
    const first = controller.query()
    expect(controller.query()).toBe(first)
    expect(controller.query('1', 1, true)).toBe(first)
    const second = controller.catalog('1')
    expect(controller.catalog('1', true)).toBe(second)
    await turn()
    expect(api.topology).toHaveBeenCalledTimes(1)
    expect(api.catalog).toHaveBeenCalledTimes(1)
    tree.resolve(query('1', [node('1')]))
    catalog.resolve(emptyCatalog)
    await Promise.all([first, second])
  })

  it('reuses fresh results, refreshes stale results in place after 30 seconds, and retains errors', async () => {
    let now = 0
    const { api, controller, ids } = fixture(() => now)
    api.topology.mockResolvedValue(query('1', [node('1', undefined, true), node('2', '1')]))
    await controller.query()
    now = DISCOVERY_TTL - 1
    await controller.query()
    expect(api.topology).toHaveBeenCalledTimes(1)
    now += 1
    const pending = deferred<TopologyQuery>()
    api.topology.mockReturnValue(pending.promise)
    const refresh = controller.query()
    expect(controller.getSnapshot().children.get('1')).toMatchObject({ status: 'loading', stale: true })
    expect(ids()).toEqual(['1', '2'])
    pending.reject(new Error('Forbidden'))
    await refresh
    expect(controller.getSnapshot().children.get('1')).toMatchObject({ status: 'error', errorCode: 'Forbidden', errorAt: now, children: ['2'] })
    expect(ids()).toEqual(['1', '2'])
    api.topology.mockResolvedValue(query('1', [node('1')], 1, 2))
    await controller.query('1', 1, true)
    expect(controller.getSnapshot().children.get('1')).toMatchObject({ status: 'loaded', children: [] })
    expect(ids()).toEqual(['1'])
  })

  it('shares four slots across topology and catalogs, including stale calls after reconnect', async () => {
    const { api, controller } = fixture()
    const trees: ReturnType<typeof deferred<TopologyQuery>>[] = []
    const catalogs: ReturnType<typeof deferred<ResourceCatalog>>[] = []
    api.topology.mockImplementation(() => { const item = deferred<TopologyQuery>(); trees.push(item); return item.promise })
    api.catalog.mockImplementation(() => { const item = deferred<ResourceCatalog>(); catalogs.push(item); return item.promise })
    const first = controller.query()
    const oldCatalogs = ['10', '11', '12', '13', '14'].map((id) => controller.catalog(id))
    await turn()
    expect(api.topology.mock.calls.length + api.catalog.mock.calls.length).toBe(4)
    controller.activate('profile-b', 2, '2')
    const next = controller.query('2')
    await turn()
    expect(api.topology).toHaveBeenCalledTimes(1)
    trees[0]!.resolve(query('1', [node('1')]))
    await first; await turn()
    expect(api.topology).toHaveBeenCalledTimes(2)
    expect(controller.getSnapshot().topology.nodes).toEqual([])
    trees[1]!.resolve(query('2', [node('2')]))
    catalogs.forEach((item) => item.resolve(emptyCatalog))
    await Promise.all([next, ...oldCatalogs])
    expect(controller.getSnapshot().topology.nodes.map((node) => node.node_id)).toEqual(['2'])
    expect(controller.getSnapshot().catalogs.size).toBe(0)
    expect(api.catalog).toHaveBeenCalledTimes(3)
  })

  it.each(['profile', 'connection', 'root'])('discards stale responses when %s changes', async (kind) => {
    const { api, controller, ids } = fixture()
    const pending = deferred<TopologyQuery>()
    api.topology.mockReturnValueOnce(pending.promise)
    const old = controller.query()
    await turn()
    controller.activate(kind === 'profile' ? 'profile-b' : 'profile-a', kind === 'connection' ? 2 : 1, kind === 'root' ? '2' : '1')
    const root = kind === 'root' ? '2' : '1'
    api.topology.mockResolvedValue(query(root, [node(root)]))
    await controller.query(root)
    pending.resolve(query('1', [node('1', undefined, true), node('99', '1')]))
    await old
    expect(ids()).toEqual([root])
  })

  it('preserves outer parents and uncovered boundary caches, comparing versions only per owner', async () => {
    const { api, controller, ids } = fixture()
    api.topology.mockResolvedValueOnce(query('1', [node('1', undefined, true), node('2', '1', true)], 1, 800))
    await controller.query()
    api.topology.mockResolvedValueOnce(query('2', [node('2', undefined, true), node('3', '2')], 1, 1, 'b'.repeat(32)))
    await controller.query('2')
    expect(controller.getSnapshot().topology.nodes.find((node) => node.node_id === '2')?.parent_id).toBe('1')
    api.topology.mockResolvedValueOnce(query('1', [node('1', undefined, true), node('2', '1', true)], 1, 801))
    await controller.query('1', 1, true)
    expect(ids()).toEqual(['1', '2', '3'])
    expect(controller.getSnapshot().children.get('2')).toMatchObject({ instance: 'b'.repeat(32), revision: 1, children: ['3'] })
  })

  it('rejects lower revisions within an instance and accepts a restarted provider', async () => {
    const { api, controller, ids } = fixture()
    api.topology.mockResolvedValue(query('1', [node('1', undefined, true), node('2', '1')], 1, 100))
    await controller.query()
    api.topology.mockResolvedValue(query('1', [node('1')], 1, 99))
    await controller.query('1', 1, true)
    expect(ids()).toEqual(['1', '2'])
    api.topology.mockResolvedValue(query('1', [node('1')], 1, 1, 'b'.repeat(32)))
    await controller.query('1', 1, true)
    expect(ids()).toEqual(['1'])
    expect(controller.getSnapshot().children.get('1')).toMatchObject({ revision: 1, instance: 'b'.repeat(32) })
  })

  it('invalidates removed branches and their requests without deleting independent View catalogs', async () => {
    const { api, controller, ids } = fixture()
    api.topology.mockResolvedValueOnce(query('1', [node('1', undefined, true), node('2', '1', true)]))
    await controller.query()
    await controller.catalog('2')
    const child = deferred<TopologyQuery>()
    api.topology.mockReturnValueOnce(child.promise)
    const pending = controller.query('2')
    await turn()
    api.topology.mockResolvedValueOnce(query('1', [node('1')], 1, 2))
    await controller.query('1', 1, true)
    child.resolve(query('2', [node('2', undefined, true), node('3', '2')]))
    await pending
    expect(ids()).toEqual(['1'])
    expect(controller.getSnapshot().children.has('2')).toBe(false)
    expect(controller.getSnapshot().catalogs.get('2')?.status).toBe('loaded')
  })

  it('reparents cached descendants and rejects requests issued under the former path', async () => {
    const { api, controller } = fixture()
    api.topology.mockResolvedValueOnce(query('1', [node('1', undefined, true), node('2', '1', true), node('3', '1'), node('4', '2', true), node('5', '4')], 0))
    await controller.query('1', 0)
    const pendingChild = deferred<TopologyQuery>()
    api.topology.mockReturnValueOnce(pendingChild.promise)
    const pending = controller.query('4', 1, true)
    await turn()
    api.topology.mockResolvedValueOnce(query('3', [node('3', undefined, true), node('2', '3', true)], 1, 1, 'c'.repeat(32)))
    await controller.query('3', 1, true)
    pendingChild.resolve(query('4', [node('4', undefined, true), node('99', '4')], 1, 900))
    await pending
    const nodes = controller.getSnapshot().topology.nodes
    expect(nodes.find((node) => node.node_id === '2')?.parent_id).toBe('3')
    expect(nodes.find((node) => node.node_id === '4')?.parent_id).toBe('2')
    expect(nodes.some((node) => node.node_id === '5')).toBe(true)
    expect(nodes.some((node) => node.node_id === '99')).toBe(false)
  })

  it('does not let an older overlapping response undo a newer reparent', async () => {
    const { api, controller } = fixture()
    const initial = query('1', [node('1', undefined, true), node('2', '1'), node('3', '1')], 0)
    api.topology.mockResolvedValueOnce(initial)
    await controller.query('1', 0)
    const old = deferred<TopologyQuery>()
    api.topology.mockReturnValueOnce(old.promise)
    const pending = controller.query('1', 0, true)
    await turn()
    api.topology.mockResolvedValueOnce(query('3', [node('3', undefined, true), node('2', '3')]))
    await controller.query('3', 1, true)
    old.resolve({ ...initial, revision: 2 })
    await pending
    expect(controller.getSnapshot().topology.nodes.find((node) => node.node_id === '2')?.parent_id).toBe('3')
  })

  it('bounds the merged navigation cache atomically without truncating', async () => {
    const { api, controller, ids } = fixture()
    const nodes = [node('1', undefined, true), ...Array.from({ length: 4095 }, (_, i) => node(String(i + 2), '1', i === 0))]
    api.topology.mockResolvedValueOnce(query('1', nodes))
    await controller.query()
    expect(ids()).toHaveLength(4096)
    api.topology.mockResolvedValueOnce(query('2', [node('2', undefined, true), node('4097', '2')]))
    await controller.query('2')
    expect(ids()).toHaveLength(4096)
    expect(ids()).not.toContain('4097')
    expect(controller.getSnapshot().children.get('2')).toMatchObject({ status: 'error', errorCode: 'Overflow' })
  })

  it('loads catalogs for unknown View owners, preserving descriptors on errors and distinguishing successful empty', async () => {
    const { api, controller } = fixture()
    const descriptor: ResourceDescriptor = { id: { owner_node_id: '99', name: 'command' }, type: 'mfh.command', type_version: 1, capabilities: [], limits: { max_payload_bytes: 1024 } }
    api.catalog.mockResolvedValueOnce({ ...emptyCatalog, resources: [descriptor] })
    await controller.catalog('99')
    api.catalog.mockRejectedValueOnce(new Error('Forbidden'))
    await controller.catalog('99', true)
    expect(controller.getSnapshot().resources).toEqual([descriptor])
    expect(controller.getSnapshot().catalogs.get('99')).toMatchObject({ status: 'error', errorCode: 'Forbidden' })
    await controller.catalog('99', true)
    expect(controller.getSnapshot().resources).toEqual([])
    expect(controller.getSnapshot().catalogs.get('99')).toMatchObject({ status: 'loaded', catalog: emptyCatalog })
    expect(api.topology).not.toHaveBeenCalled()
  })

  it('evicts least-recently-used catalogs but pins selection and active View owners', async () => {
    const { controller } = fixture()
    controller.setCatalogOwners(['1', '2'])
    for (let id = 1; id <= 128; id += 1) await controller.catalog(String(id))
    await controller.catalog('3') // touch; owner 4 becomes the oldest evictable entry
    await controller.catalog('129')
    expect(controller.getSnapshot().catalogs.size).toBe(128)
    expect(controller.getSnapshot().catalogs.has('1')).toBe(true)
    expect(controller.getSnapshot().catalogs.has('2')).toBe(true)
    expect(controller.getSnapshot().catalogs.has('3')).toBe(true)
    expect(controller.getSnapshot().catalogs.has('4')).toBe(false)
    controller.setCatalogOwners([...controller.getSnapshot().catalogs.keys(), '130'])
    await controller.catalog('130')
    expect(controller.getSnapshot().catalogs.size).toBe(128)
    expect(controller.getSnapshot().limitError).toContain('128')
  })

  it('handles synchronous provider exceptions without leaving phantom in-flight entries', async () => {
    const { api, controller } = fixture()
    api.topology.mockImplementationOnce(() => { throw new Error('Unsupported') })
    await controller.query()
    api.topology.mockResolvedValue(query('1', [node('1')]))
    await controller.query('1', 1, true)
    expect(api.topology).toHaveBeenCalledTimes(2)
    expect(controller.getSnapshot().children.get('1')?.status).toBe('loaded')
  })

  it('releases superseded queued query keys and allows a later shallow refresh', async () => {
    const { api, controller } = fixture()
    const blockers = Array.from({ length: 4 }, () => deferred<ResourceCatalog>())
    api.catalog.mockImplementation((owner) => blockers[Number(owner) - 10]!.promise)
    const catalogs = [10, 11, 12, 13].map((id) => controller.catalog(String(id)))
    await turn()
    const shallow = controller.query('1', 1)
    const deep = controller.query('1', 0)
    api.topology.mockImplementation(async (_, depth) => query('1', [node('1')], depth))
    blockers.forEach((item) => item.resolve(emptyCatalog))
    await Promise.all([...catalogs, shallow, deep])
    expect(api.topology).toHaveBeenCalledExactlyOnceWith('1', 0)
    await controller.query('1', 1, true)
    expect(api.topology).toHaveBeenLastCalledWith('1', 1)
    expect(api.topology).toHaveBeenCalledTimes(2)
  })

  it('refreshes visible expanded paths and focused known nodes without visiting collapsed branches', async () => {
    const { api, controller } = fixture()
    const nodes = [node('1', undefined, true), node('2', '1', true), node('3', '2', true), node('4', '3')]
    api.topology.mockResolvedValueOnce(query('1', nodes, 0))
    await controller.query('1', 0)
    api.topology.mockImplementation(async (owner) => owner === '1' ? query('1', nodes.slice(0, 2)) : query(owner, [node(owner, undefined, true), node('4', owner)]))
    api.topology.mockClear()
    await controller.refresh(['1', '3'])
    expect(api.topology).toHaveBeenCalledExactlyOnceWith('1', 1)
    api.topology.mockClear()
    await controller.refresh(['1', '3'], '3')
    expect(api.topology.mock.calls).toEqual([['1', 1], ['3', 1]])
  })

  it('retains locally discovered descendant names through ancestor shallow/full refresh and allows an owner to clear its name', async () => {
    const { api, controller } = fixture()
    const root = node('1', undefined, true)
    const child = { ...node('2', '1', true), display_name: undefined }
    const leaf = { ...node('3', '2'), display_name: undefined }
    api.topology.mockResolvedValueOnce(query('1', [root, child]))
    await controller.query()
    api.topology.mockResolvedValueOnce(query('2', [{ ...node('2', undefined, true), display_name: 'Local name 2' }, leaf]))
    await controller.query('2')
    api.topology.mockResolvedValueOnce(query('3', [{ ...node('3'), display_name: 'Local name 3' }]))
    await controller.query('3', 1, true)
    const find = (id: string) => controller.getSnapshot().topology.nodes.find((node) => node.node_id === id)!
    api.topology.mockResolvedValueOnce(query('1', [root, child], 1, 2))
    await controller.query('1', 1, true)
    expect(find('2').display_name).toBe('Local name 2')
    expect(find('3').display_name).toBe('Local name 3')
    api.topology.mockResolvedValueOnce(query('1', [root, child, leaf], 0, 3))
    await controller.query('1', 0, true)
    expect(find('2').display_name).toBe('Local name 2')
    expect(find('3').display_name).toBe('Local name 3')
    const stableChild = find('2')
    api.topology.mockResolvedValueOnce(query('1', [root, child, leaf], 0, 4))
    await controller.query('1', 0, true)
    expect(find('2')).toBe(stableChild)
    api.topology.mockResolvedValueOnce(query('2', [{ ...child, parent_id: undefined }, leaf], 1, 2))
    await controller.query('2', 1, true)
    expect(find('2').display_name).toBeUndefined()
    expect(find('3').display_name).toBe('Local name 3')
  })
})
