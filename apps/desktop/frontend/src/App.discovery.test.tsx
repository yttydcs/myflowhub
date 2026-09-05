import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { App } from './App'
import type { DesktopAPI } from './api'
import { deferred, node, query } from './discovery/fixtures.test-support'
import type { ConnectionStatus, Profile, ResourceCatalog, ResourceDescriptor, TopologyQuery, ViewDefinition } from './types'

const profile: Profile = { id: 'a', name: 'Profile A', node_id: '7', endpoint: 'localhost:9540', parent_node_id: '1', parent_public_key: '', auto_connect: false }
const command: ResourceDescriptor = { id: { owner_node_id: '90', name: 'actions/draft' }, type: 'mfh.command', type_version: 1, capabilities: [{ name: 'invoke', permission: 'invoke', max_payload_bytes: 4096 }], limits: { max_payload_bytes: 4096 }, presentation: { label: 'Draft command' } }
const savedView: ViewDefinition = { id: 'saved', name: 'Saved layout', revision: 1, widgets: [{ id: 'draft', owner_node_id: '90', resource_name: 'actions/draft', renderer: 'mfh.command' }], layout_root: { kind: 'leaf', widget_id: 'draft' } }
const emptyCatalog: ResourceCatalog = { version: 2, revision: 1, resources: [] }

function apiFixture(view?: ViewDefinition): DesktopAPI {
  return {
    settings: vi.fn().mockResolvedValue({ version: 2, profiles: [profile], active_profile_id: profile.id, updated_at_unix_ms: 1, credential_mode: 'session-only' }),
    profileStates: vi.fn().mockResolvedValue([{ profile_id: profile.id, state: 'legacy' }]),
    prepareProfile: vi.fn(), saveProfile: vi.fn(), login: vi.fn(), switchProfile: vi.fn(), deactivateProfile: vi.fn(), deleteProfile: vi.fn(), identity: vi.fn(), connect: vi.fn(), disconnect: vi.fn(),
    status: vi.fn().mockResolvedValue({ state: 'connected', parent_node_id: '10', generation: 1, link_generation: 1 }),
    topology: vi.fn().mockImplementation(async (owner: string, depth = 1) => query(owner, [node(owner)], depth)),
    catalog: vi.fn().mockImplementation(async (owner: string) => ({ ...emptyCatalog, resources: owner === '90' ? [command] : [] })),
    snapshot: vi.fn(), operate: vi.fn(), subscribe: vi.fn(), poll: vi.fn(), cancel: vi.fn(), pickFile: vi.fn(), uploadFile: vi.fn(),
    views: vi.fn().mockResolvedValue({ version: 3, views: view ? [view] : [] }),
    saveView: vi.fn().mockImplementation(async (view: ViewDefinition) => ({ ...view, revision: view.revision + 1 })), deleteView: vi.fn(),
  }
}

afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); window.localStorage.clear() })

describe('desktop discovery integration', () => {
  it('uses the current direct parent, renders a saved reference before topology, and loads catalogs only on usage', async () => {
    const api = apiFixture(savedView)
    const tree = deferred<TopologyQuery>()
    const catalog = deferred<ResourceCatalog>()
    vi.mocked(api.topology).mockReturnValue(tree.promise)
    vi.mocked(api.catalog).mockImplementation(async (owner) => owner === '90' ? catalog.promise : emptyCatalog)
    render(<App api={api} />)
    expect(await screen.findByText(/正在加载Node 90 资源目录/)).toBeInTheDocument()
    expect(screen.queryByText('资源暂不可用')).not.toBeInTheDocument()
    expect(api.topology).toHaveBeenCalledExactlyOnceWith('10', 1)
    expect(vi.mocked(api.catalog).mock.calls.map(([owner]) => owner).sort()).toEqual(['10', '90'])
    await act(async () => catalog.resolve({ ...emptyCatalog, resources: [command] }))
    expect(await screen.findByRole('textbox', { name: /输入/ })).toBeInTheDocument()
    expect(screen.queryByRole('treeitem', { name: /Node 10/ })).not.toBeInTheDocument()
    await act(async () => tree.resolve(query('10', [node('10', undefined, true), node('11', '10', true), node('12', '10')])) )
    expect(await screen.findByRole('treeitem', { name: /Node 11/ })).toHaveAttribute('aria-expanded', 'false')
    expect(api.catalog).toHaveBeenCalledTimes(2)
    expect(screen.getByLabelText('搜索已加载节点')).toBeInTheDocument()
    expect(screen.queryByLabelText('浏览起点 Node ID')).not.toBeInTheDocument()
    const scope = screen.getByRole('button', { name: '修改浏览起点，当前 Node 10' })
    expect(scope).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(scope)
    expect(screen.getByLabelText('浏览起点 Node ID')).toHaveValue('10')
    expect(scope).toHaveAttribute('aria-expanded', 'true')
    fireEvent.click(scope)
    expect(screen.queryByRole('region', { name: '编辑浏览起点' })).not.toBeInTheDocument()
  })

  it('loads a six-level path through mouse and keyboard, reuses collapse caches, and never searches the network', async () => {
    const api = apiFixture()
    vi.mocked(api.topology).mockImplementation(async (owner, depth = 1) => query(owner, [node(owner, undefined, true), node(String(Number(owner) + 1), owner, Number(owner) < 14)], depth))
    render(<App api={api} />)
    await screen.findByRole('treeitem', { name: /Node 11/ })
    for (let id = 11; id <= 14; id += 1) {
      const item = screen.getByRole('treeitem', { name: new RegExp(`^Node ${id} `) })
      if (id % 2) fireEvent.keyDown(item, { key: 'ArrowRight' })
      else fireEvent.doubleClick(item)
      await screen.findByRole('treeitem', { name: new RegExp(`^Node ${id + 1} `) })
    }
    expect(vi.mocked(api.topology).mock.calls).toEqual([['10', 1], ['11', 1], ['12', 1], ['13', 1], ['14', 1]])
    const branch = screen.getByRole('treeitem', { name: /^Node 11 / })
    fireEvent.keyDown(branch, { key: 'ArrowLeft' })
    expect(screen.queryByRole('treeitem', { name: /^Node 15 / })).not.toBeInTheDocument()
    fireEvent.keyDown(branch, { key: 'ArrowRight' })
    expect(await screen.findByRole('treeitem', { name: /^Node 15 / })).toBeInTheDocument()
    expect(api.topology).toHaveBeenCalledTimes(5)
    fireEvent.change(screen.getByLabelText('搜索已加载节点'), { target: { value: 'Node 15' } })
    expect(await screen.findByRole('treeitem', { name: /^Node 10 / })).toBeInTheDocument()
    expect(api.topology).toHaveBeenCalledTimes(5)
    const preferences = JSON.parse(window.localStorage.getItem('mfh.desktop.ui.v1:a')!)
    expect(Object.keys(preferences).sort()).toEqual(['expanded_node_ids', 'explorer_split_ratio', 'theme', 'version'])
  })

  it('deduplicates repeated star expansion and displays busy and retry states instead of leaves', async () => {
    const api = apiFixture()
    const left = deferred<TopologyQuery>()
    const right = deferred<TopologyQuery>()
    vi.mocked(api.topology).mockImplementation(async (owner) => owner === '10' ? query('10', [node('10', undefined, true), node('11', '10', true), node('12', '10', true)]) : owner === '11' ? left.promise : right.promise)
    render(<App api={api} />)
    const branch = await screen.findByRole('treeitem', { name: /^Node 11 / })
    fireEvent.keyDown(branch, { key: '*' })
    fireEvent.keyDown(branch, { key: '*' })
    await waitFor(() => expect(api.topology).toHaveBeenCalledTimes(3))
    expect(branch).toHaveAttribute('aria-busy', 'true')
    await act(async () => { left.reject(new Error('Forbidden')); right.resolve(query('12', [node('12')])) })
    expect(screen.getByRole('treeitem', { name: /^Node 11 / })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('treeitem', { name: /^Node 12 / })).not.toHaveAttribute('aria-expanded')
    expect(screen.getByRole('button', { name: '重试节点 Node 11' })).toBeInTheDocument()
    vi.mocked(api.topology).mockResolvedValue(query('11', [node('11')]))
    fireEvent.keyDown(branch, { key: 'ArrowRight' })
    await waitFor(() => expect(screen.queryByRole('button', { name: '重试节点 Node 11' })).not.toBeInTheDocument())
  })

  it('expresses unsupported and catalog errors, then shows empty only after successful retries', async () => {
    const api = apiFixture()
    vi.mocked(api.topology).mockRejectedValue(new Error('Unsupported'))
    vi.mocked(api.catalog).mockRejectedValue(new Error('Forbidden'))
    render(<App api={api} />)
    expect(await screen.findByText(/不支持此查询/)).toBeInTheDocument()
    expect(screen.queryByText('没有匹配的已加载节点')).not.toBeInTheDocument()
    expect(screen.queryByText('此节点没有资源')).not.toBeInTheDocument()
    vi.mocked(api.topology).mockResolvedValue(query('10', [node('10')]))
    fireEvent.click(screen.getByRole('button', { name: '重试节点树' }))
    expect(await screen.findByText(/资源目录：无权访问/)).toBeInTheDocument()
    vi.mocked(api.catalog).mockResolvedValue(emptyCatalog)
    fireEvent.click(screen.getByRole('button', { name: '重试资源目录' }))
    expect(await screen.findByText('此节点没有资源')).toBeInTheDocument()
    expect(api.topology).toHaveBeenCalledTimes(2)
  })

  it('uses explicit roots and depth zero only for the full-subtree action', async () => {
    const api = apiFixture()
    render(<App api={api} />)
    await screen.findByRole('treeitem', { name: /Node 10/ })
    fireEvent.click(screen.getByRole('button', { name: '修改浏览起点，当前 Node 10' }))
    fireEvent.change(screen.getByLabelText('浏览起点 Node ID'), { target: { value: '20' } })
    fireEvent.click(screen.getByRole('button', { name: '设为起点' }))
    await screen.findByRole('treeitem', { name: /Node 20/ })
    fireEvent.click(screen.getByRole('button', { name: '加载完整子树' }))
    await waitFor(() => expect(api.topology).toHaveBeenCalledWith('20', 0))
    fireEvent.click(screen.getByRole('button', { name: '直接父节点' }))
    await screen.findByRole('treeitem', { name: /Node 10/ })
    expect(vi.mocked(api.topology).mock.calls).toEqual([['10', 1], ['20', 1], ['20', 0], ['10', 1]])
  })

  it('retains layout and command input across catalog failure, automatic reconnect and successful recovery', async () => {
    vi.useFakeTimers()
    const api = apiFixture(savedView)
    let status: ConnectionStatus = { state: 'connected', parent_node_id: '10', generation: 1, link_generation: 1 }
    vi.mocked(api.status).mockImplementation(async () => status)
    render(<App api={api} />)
    await act(async () => {})
    const input = screen.getByRole('textbox', { name: /输入/ })
    fireEvent.change(input, { target: { value: '{"draft":"keep me"}' } })
    fireEvent.change(screen.getByLabelText('视图名称'), { target: { value: 'Unsaved layout' } })
    vi.mocked(api.catalog).mockRejectedValue(new Error('Forbidden'))
    status = { ...status, generation: 2, link_generation: 2 }
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    expect(screen.getByRole('textbox', { name: /输入/ })).toHaveValue('{"draft":"keep me"}')
    expect(screen.getByRole('button', { name: '执行 invoke' })).toBeDisabled()
    expect(screen.getByLabelText('视图名称')).toHaveValue('Unsaved layout')
    expect(screen.queryByText('资源暂不可用')).not.toBeInTheDocument()
    expect(api.topology).toHaveBeenCalledTimes(2)
    await act(async () => { await vi.advanceTimersByTimeAsync(2000) })
    expect(api.topology).toHaveBeenCalledTimes(2)
    vi.mocked(api.catalog).mockImplementation(async (owner) => ({ ...emptyCatalog, resources: owner === '90' ? [command] : [] }))
    fireEvent.click(screen.getByRole('button', { name: '重试Node 90 资源目录' }))
    await act(async () => {})
    expect(screen.getByRole('textbox', { name: /输入/ })).toHaveValue('{"draft":"keep me"}')
    expect(screen.getByRole('button', { name: '执行 invoke' })).toBeEnabled()
    expect(api.operate).not.toHaveBeenCalled()
  })

  it('follows current parent changes by default but retains explicitly chosen scopes', async () => {
    vi.useFakeTimers()
    const api = apiFixture()
    let status: ConnectionStatus = { state: 'connected', parent_node_id: '10', generation: 1 }
    vi.mocked(api.status).mockImplementation(async () => status)
    render(<App api={api} />)
    await act(async () => {})
    status = { ...status, parent_node_id: '20', generation: 2 }
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    fireEvent.click(screen.getByRole('button', { name: '修改浏览起点，当前 Node 20' }))
    expect(screen.getByLabelText('浏览起点 Node ID')).toHaveValue('20')
    fireEvent.change(screen.getByLabelText('浏览起点 Node ID'), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: '设为起点' }))
    await act(async () => {})
    status = { ...status, parent_node_id: '40', generation: 3 }
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    expect(screen.getByLabelText('浏览起点 Node ID')).toHaveValue('30')
    expect(api.topology).toHaveBeenLastCalledWith('30', 1)
    fireEvent.click(screen.getByRole('button', { name: '直接父节点' }))
    await act(async () => {})
    expect(api.topology).toHaveBeenLastCalledWith('40', 1)
    status = { ...status, parent_node_id: '50', generation: 4 }
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    expect(api.topology).toHaveBeenLastCalledWith('50', 1)
  })

  it('does not overlap background status checks with initial auto-connect waiting', async () => {
    vi.useFakeTimers()
    const api = apiFixture()
    vi.mocked(api.settings).mockResolvedValue({ version: 2, profiles: [{ ...profile, auto_connect: true }], active_profile_id: profile.id, updated_at_unix_ms: 1, credential_mode: 'session-only' })
    let connected = false
    vi.mocked(api.status).mockImplementation(async () => ({ state: connected ? 'connected' : 'connecting', parent_node_id: '10', generation: connected ? 2 : 1 }))
    render(<App api={api} />)
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    expect(api.status).toHaveBeenCalledTimes(5)
    expect(api.topology).not.toHaveBeenCalled()
    connected = true
    await act(async () => { await vi.advanceTimersByTimeAsync(250) })
    expect(api.topology).toHaveBeenCalledExactlyOnceWith('10', 1)
  })

  it('keeps the tree visible when a selected node catalog is slow', async () => {
    const api = apiFixture()
    const pending = deferred<ResourceCatalog>()
    vi.mocked(api.catalog).mockReturnValue(pending.promise)
    render(<App api={api} />)
    expect(await screen.findByRole('treeitem', { name: /Node 10/ })).toBeInTheDocument()
    expect(within(screen.getByRole('tree', { name: '资源' })).getByText(/正在加载资源目录/)).toBeInTheDocument()
    await act(async () => pending.resolve(emptyCatalog))
  })

  it('switches Profiles while old topology, catalog and View responses are delayed', async () => {
    const api = apiFixture()
    const second: Profile = { ...profile, id: 'b', name: 'Profile B', parent_node_id: '20' }
    let active = profile.id
    vi.mocked(api.settings).mockImplementation(async () => ({ version: 2, profiles: [profile, second], active_profile_id: active, updated_at_unix_ms: 1, credential_mode: 'session-only' }))
    vi.mocked(api.switchProfile).mockImplementation(async (id) => { active = id })
    vi.mocked(api.status).mockImplementation(async () => ({ state: 'connected', parent_node_id: active === 'a' ? '10' : '20', generation: 1 }))
    const oldTree = deferred<TopologyQuery>()
    const oldCatalog = deferred<ResourceCatalog>()
    const oldViews = deferred<{ version: number; views: ViewDefinition[] }>()
    vi.mocked(api.topology).mockImplementation(async (owner) => owner === '10' ? oldTree.promise : query(owner, [node(owner)]))
    vi.mocked(api.catalog).mockImplementation(async (owner) => owner === '10' ? oldCatalog.promise : emptyCatalog)
    vi.mocked(api.views).mockImplementation(async () => active === 'a' ? oldViews.promise : { version: 3, views: [{ id: 'b-view', name: 'Profile B layout', revision: 1, widgets: [] }] })
    render(<App api={api} />)
    fireEvent.click(await screen.findByRole('button', { name: '打开 Profile A 的设置' }))
    fireEvent.click(screen.getByRole('button', { name: 'Profile' }))
    fireEvent.click(screen.getByRole('button', { name: '启用' }))
    await screen.findByRole('treeitem', { name: /Node 20/ })
    await act(async () => {
      oldTree.resolve(query('10', [node('10', undefined, true), node('99', '10')]))
      oldCatalog.reject(new Error('Forbidden OLD PROFILE'))
      oldViews.resolve({ version: 3, views: [savedView] })
    })
    expect(screen.queryByRole('treeitem', { name: /Node 99/ })).not.toBeInTheDocument()
    expect(screen.queryByText(/OLD PROFILE/)).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Profile B layout' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Saved layout' })).not.toBeInTheDocument()
    expect(api.catalog).not.toHaveBeenCalledWith('90')
  })

  it('invalidates pending topology and catalogs across a detected disconnect and reconnect', async () => {
    vi.useFakeTimers()
    const api = apiFixture()
    let status: ConnectionStatus = { state: 'connected', parent_node_id: '10', generation: 1 }
    vi.mocked(api.status).mockImplementation(async () => status)
    const oldTree = deferred<TopologyQuery>()
    const oldCatalog = deferred<ResourceCatalog>()
    vi.mocked(api.topology).mockReturnValueOnce(oldTree.promise)
    vi.mocked(api.catalog).mockReturnValueOnce(oldCatalog.promise)
    render(<App api={api} />)
    await act(async () => {})
    status = { ...status, state: 'disconnected', generation: 2 }
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    await act(async () => {
      oldTree.resolve(query('10', [node('10', undefined, true), node('99', '10')]))
      oldCatalog.reject(new Error('Old disconnected catalog'))
    })
    expect(screen.queryByRole('treeitem', { name: /Node 99/ })).not.toBeInTheDocument()
    status = { ...status, state: 'connected', generation: 3 }
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    expect(api.topology).toHaveBeenCalledTimes(2)
    expect(screen.getByRole('treeitem', { name: /Node 10/ })).toBeInTheDocument()
    expect(screen.queryByText(/Old disconnected/)).not.toBeInTheDocument()
  })

  it('ignores a pending local status response after unmounting', async () => {
    vi.useFakeTimers()
    const api = apiFixture()
    const { unmount } = render(<App api={api} />)
    await act(async () => {})
    const pending = deferred<ConnectionStatus>()
    vi.mocked(api.status).mockReturnValueOnce(pending.promise)
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    unmount()
    await act(async () => pending.resolve({ state: 'connected', parent_node_id: '20', generation: 2 }))
    expect(api.topology).toHaveBeenCalledExactlyOnceWith('10', 1)
  })

  it('keeps monitoring the local connection while a manual remote refresh is slow', async () => {
    vi.useFakeTimers()
    const api = apiFixture()
    let status: ConnectionStatus = { state: 'connected', parent_node_id: '10', generation: 1 }
    vi.mocked(api.status).mockImplementation(async () => status)
    render(<App api={api} />)
    await act(async () => {})
    const old = deferred<TopologyQuery>()
    vi.mocked(api.topology).mockReturnValueOnce(old.promise)
    fireEvent.click(screen.getByRole('button', { name: '刷新节点树' }))
    await act(async () => {})
    status = { ...status, parent_node_id: '20', generation: 2 }
    await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
    expect(screen.getByRole('treeitem', { name: /Node 20/ })).toBeInTheDocument()
    await act(async () => old.resolve(query('10', [node('10', undefined, true), node('99', '10')], 1, 9)))
    expect(screen.queryByRole('treeitem', { name: /Node 99/ })).not.toBeInTheDocument()
  })

  it('updates the selected node preview when a local query supplies its name', async () => {
    const api = apiFixture()
    vi.mocked(api.topology).mockImplementation(async (owner) => owner === '10'
      ? query('10', [node('10', undefined, true), { ...node('11', '10', true), display_name: undefined }])
      : query('11', [{ ...node('11'), display_name: 'Named relay' }]))
    render(<App api={api} />)
    const item = await screen.findByRole('treeitem', { name: /^Node 11 / })
    fireEvent.click(item)
    expect(screen.getByRole('complementary', { name: 'Node 11 预览' })).toBeInTheDocument()
    fireEvent.keyDown(item, { key: 'ArrowRight' })
    expect(await screen.findByRole('complementary', { name: 'Named relay 预览' })).toBeInTheDocument()
  })

  it('never loads Profile A View owners on Profile B while B views are delayed', async () => {
    const viewA: ViewDefinition = { ...savedView, widgets: [{ ...savedView.widgets[0]!, owner_node_id: '99' }] }
    const api = apiFixture(viewA)
    const second: Profile = { ...profile, id: 'b', name: 'Profile B', parent_node_id: '20' }
    let active = profile.id
    const calls: Array<[string, string]> = []
    vi.mocked(api.settings).mockImplementation(async () => ({ version: 2, profiles: [profile, second], active_profile_id: active, updated_at_unix_ms: 1, credential_mode: 'session-only' }))
    vi.mocked(api.switchProfile).mockImplementation(async (id) => { active = id })
    vi.mocked(api.status).mockImplementation(async () => ({ state: 'connected', parent_node_id: active === 'a' ? '10' : '20', generation: 1 }))
    vi.mocked(api.catalog).mockImplementation(async (owner) => {
      calls.push([active, owner])
      return { ...emptyCatalog, resources: owner === '99' ? [{ ...command, id: { ...command.id, owner_node_id: '99' } }] : [] }
    })
    const pending = deferred<{ version: number; views: ViewDefinition[] }>()
    vi.mocked(api.views).mockImplementation(async () => active === 'a' ? { version: 3, views: [viewA] } : pending.promise)
    render(<App api={api} />)
    await screen.findByRole('textbox', { name: /输入/ })
    expect(calls).toContainEqual(['a', '99'])
    fireEvent.click(screen.getByRole('button', { name: '打开 Profile A 的设置' }))
    fireEvent.click(screen.getByRole('button', { name: 'Profile' }))
    fireEvent.click(screen.getByRole('button', { name: '启用' }))
    await screen.findByRole('treeitem', { name: /Node 20/ })
    expect(calls).toContainEqual(['b', '20'])
    expect(calls).not.toContainEqual(['b', '99'])
    fireEvent.click(screen.getByRole('tab', { name: '视图加载中…' }))
    expect(screen.getByText('正在加载当前 Profile 的视图…')).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: /输入/ })).not.toBeInTheDocument()
    await act(async () => pending.resolve({ version: 3, views: [] }))
    expect(calls).not.toContainEqual(['b', '99'])
    expect(screen.getByLabelText('视图名称')).toHaveValue('工作视图 1')
  })

  it.each(['create', 'open'])('preserves a new V2 draft when a slow V1 save finishes after %s', async (action) => {
    const first: ViewDefinition = { id: 'view-1', name: 'V1', revision: 1, widgets: [] }
    const second: ViewDefinition = { id: 'view-2', name: 'V2', revision: 1, widgets: [] }
    const api = apiFixture(first)
    if (action === 'open') vi.mocked(api.views).mockResolvedValue({ version: 3, views: [first, second] })
    const pending = deferred<ViewDefinition>()
    vi.mocked(api.saveView).mockReturnValueOnce(pending.promise)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<App api={api} />)
    fireEvent.change(await screen.findByLabelText('视图名称'), { target: { value: 'V1 saved edit' } })
    fireEvent.click(screen.getByRole('button', { name: '保存视图' }))
    await waitFor(() => expect(api.saveView).toHaveBeenCalledTimes(1))
    if (action === 'create') fireEvent.click(screen.getByRole('button', { name: '新建视图' }))
    else {
      fireEvent.keyDown(screen.getByRole('tab', { name: '视图' }), { key: 'Enter' })
      fireEvent.click(screen.getByRole('button', { name: /V2 0 widgets/ }))
    }
    fireEvent.change(screen.getByLabelText('视图名称'), { target: { value: 'V2 unsaved edit' } })
    await act(async () => pending.resolve({ ...first, name: 'V1 saved edit', revision: 2 }))
    expect(screen.getByLabelText('视图名称')).toHaveValue('V2 unsaved edit')
    expect(screen.getByText('未保存')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '保存视图' }))
    await waitFor(() => expect(api.saveView).toHaveBeenLastCalledWith(expect.objectContaining({ id: 'view-2', name: 'V2 unsaved edit' })))
  })

  it('preserves edits to V1 made during its own save and advances only its acknowledged revision', async () => {
    const first: ViewDefinition = { id: 'view-1', name: 'V1', revision: 1, widgets: [] }
    const api = apiFixture(first)
    const pending = deferred<ViewDefinition>()
    vi.mocked(api.saveView).mockReturnValueOnce(pending.promise)
    render(<App api={api} />)
    fireEvent.change(await screen.findByLabelText('视图名称'), { target: { value: 'Submitted edit' } })
    fireEvent.click(screen.getByRole('button', { name: '保存视图' }))
    fireEvent.change(screen.getByLabelText('视图名称'), { target: { value: 'Newer unsaved edit' } })
    await act(async () => pending.resolve({ ...first, name: 'Submitted edit', revision: 2 }))
    expect(screen.getByLabelText('视图名称')).toHaveValue('Newer unsaved edit')
    expect(screen.getByText('未保存')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '保存视图' }))
    await waitFor(() => expect(api.saveView).toHaveBeenLastCalledWith(expect.objectContaining({ revision: 2, name: 'Newer unsaved edit' })))
  })
})
