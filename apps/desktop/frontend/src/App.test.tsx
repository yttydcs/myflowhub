import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from './api'
import { App } from './App'
import type { Profile, ProfileState, ResourceDescriptor, Settings, ViewDefinition } from './types'

const profile: Profile = {
  id: 'personal', name: 'Personal', node_id: '2', endpoint: 'localhost:9540', parent_node_id: '1', parent_public_key: 'key', auto_connect: true,
}
const secondProfile: Profile = {
  ...profile, id: 'work', name: 'Work', node_id: '3',
}
const resource: ResourceDescriptor = {
  id: { owner_node_id: '1', name: 'metrics/cpu' }, type: 'mfh.variable', type_version: 1,
  capabilities: [{ name: 'subscribe', permission: 'metrics.read', event_schema: 'metrics.v1', max_payload_bytes: 1024 }],
  limits: { max_payload_bytes: 1024 }, presentation: { label: 'CPU', renderer: 'mfh.variable' },
}

const allowedFiles: ResourceDescriptor = {
  id: { owner_node_id: '1', name: 'storage/allowed-files' },
  type: 'mfh.collection',
  type_version: 1,
  capabilities: [
    { name: 'get', permission: 'filesystem.get', input_schema: 'mfh.collection.member-request.v1', output_schema: 'mfh.collection.member.v1', max_payload_bytes: 4096 },
    { name: 'list', permission: 'filesystem.list', input_schema: 'mfh.collection.list-request.v1', output_schema: 'mfh.collection.page.v1', max_payload_bytes: 4096 },
  ],
  limits: { max_payload_bytes: 4096 },
  presentation: { label: 'Allowed files' },
}

function emptyCollectionPage() {
  return { schema: 'mfh.collection.page.v1', payload: { version: 1, revision: 1, parent: '', members: [], next_cursor: '' } }
}

function mockAPI(settings: Settings): DesktopAPI {
  const profileStates: ProfileState[] = settings.profiles.map((item) => item.enrollment_mode === 'authority'
    ? { profile_id: item.id, state: item.node_id ? 'enrolled' : 'missing', node_id: item.node_id || undefined }
    : { profile_id: item.id, state: 'legacy', node_id: item.node_id })
  return {
    settings: vi.fn().mockResolvedValue(settings),
    profileStates: vi.fn().mockResolvedValue(profileStates),
    prepareProfile: vi.fn().mockImplementation(async (next: Profile) => ({ profile: next, identity: { node_id: next.node_id, public_key: 'desktop-public-key' } })),
    saveProfile: vi.fn().mockImplementation(async (next: Profile) => next),
    login: vi.fn().mockImplementation(async (next: Profile) => next),
    switchProfile: vi.fn(),
    deactivateProfile: vi.fn(),
    deleteProfile: vi.fn(),
    identity: vi.fn().mockResolvedValue({ node_id: '2', public_key: 'key' }),
    connect: vi.fn(),
    disconnect: vi.fn(),
    status: vi.fn().mockResolvedValue({ state: 'connected', endpoint: 'localhost:9540' }),
    catalog: vi.fn().mockResolvedValue({ version: 2, revision: 1, resources: [resource] }),
    topology: vi.fn().mockResolvedValue({ version: 1, epoch: 1, nodes: [{ node_id: '1', role: 'root', generation: 1 }] }),
    snapshot: vi.fn().mockResolvedValue({ value: 42 }),
    operate: vi.fn(),
    subscribe: vi.fn(),
    poll: vi.fn(),
    cancel: vi.fn(),
    pickFile: vi.fn().mockResolvedValue(''),
    uploadFile: vi.fn(),
    views: vi.fn().mockResolvedValue({ version: 3, views: [] }),
    saveView: vi.fn().mockImplementation(async (next: ViewDefinition) => ({ ...next, revision: 1 })),
    deleteView: vi.fn(),
  }
}

describe('desktop resource workspace', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    window.localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('separates existing Profiles from a minimal first connection form', async () => {
    const settings: Settings = { version: 2, profiles: [], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    render(<App api={mockAPI(settings)} />)
    expect(await screen.findByRole('heading', { name: '登录 MyFlowHub' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '首次连接' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: '使用现有 Profile' })).toHaveAttribute('aria-selected', 'false')
    expect(screen.queryByText('01')).not.toBeInTheDocument()
    expect(screen.queryByText('02')).not.toBeInTheDocument()
    expect(screen.queryByText('03')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Profile ID')).not.toBeVisible()
    expect(screen.queryByLabelText('本机 Node ID')).not.toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /申请管理员审批/ })).toBeChecked()
    expect(screen.queryByLabelText('Enrollment Permit')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('radio', { name: /已有 Permit/ }))
    expect(screen.getByLabelText('Enrollment Permit')).toBeInTheDocument()
    expect(screen.queryByLabelText(/首次连接时信任/)).not.toBeInTheDocument()
  })

  it('discovers resources, previews them in the inspector, and adds them through a keyboard-equivalent action', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    const api = mockAPI(settings)
    render(<App api={api} />)
    const resourceItem = await screen.findByRole('treeitem', { name: /CPU/ })
    fireEvent.click(resourceItem)
    expect(await screen.findByRole('complementary', { name: 'CPU 预览' })).toBeInTheDocument()
    await waitFor(() => expect(api.snapshot).toHaveBeenCalledWith('1', 'metrics/cpu'))
    fireEvent.click(screen.getByRole('button', { name: '添加 metrics/cpu 到工作区' }))
    expect(screen.getByText('未保存')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /保存视图/ }))
    await waitFor(() => expect(api.saveView).toHaveBeenCalled())
    vi.mocked(api.catalog).mockResolvedValue({ version: 2, revision: 2, resources: [] })
    fireEvent.click(screen.getByRole('button', { name: '刷新节点树' }))
    await waitFor(() => expect(screen.queryByRole('complementary', { name: 'CPU 预览' })).not.toBeInTheDocument())
  })

  it('focuses a descriptor capability without API calls, executes explicitly, and fails stale after refresh', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    const actionResource: ResourceDescriptor = {
      id: { owner_node_id: '1', name: 'actions/example' },
      type: 'mfh.collection',
      type_version: 1,
      capabilities: [{ name: 'invoke', permission: 'ignored.invoke', max_payload_bytes: 1024 }],
      limits: { max_payload_bytes: 1024 },
      presentation: { label: 'Example actions' },
    }
    const api = mockAPI(settings)
    vi.mocked(api.catalog).mockResolvedValue({ version: 2, revision: 1, resources: [actionResource] })
    vi.mocked(api.operate).mockResolvedValue({ schema: 'example.result.v1', payload: { accepted: true } })
    render(<App api={api} />)

    const resourceItem = await screen.findByRole('treeitem', { name: /Example actions/ })
    fireEvent.contextMenu(resourceItem)
    fireEvent.click(await screen.findByRole('menuitem', { name: /操作 invoke/ }))
    expect(await screen.findByLabelText('已聚焦 capability invoke')).toBeInTheDocument()
    expect(api.operate).not.toHaveBeenCalled()
    expect(api.snapshot).not.toHaveBeenCalled()
    expect(api.subscribe).not.toHaveBeenCalled()

    const draft = screen.getByRole('textbox', { name: /输入/ })
    fireEvent.change(draft, { target: { value: '{"request":1}' } })
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '执行 invoke' }))
    await waitFor(() => expect(api.operate).toHaveBeenCalledWith('1', 'actions/example', 'invoke', '', { request: 1 }))

    vi.mocked(api.catalog).mockResolvedValue({ version: 2, revision: 2, resources: [{ ...actionResource, capabilities: [] }] })
    fireEvent.click(screen.getByRole('button', { name: '刷新节点树' }))
    expect(await screen.findByText(/当前 descriptor 已移除此 capability/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '执行 invoke' })).not.toBeInTheDocument()
    expect(api.operate).toHaveBeenCalledTimes(1)
  })

  it('returns from a focused Collection action to the Resource preview', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    const api = mockAPI(settings)
    vi.mocked(api.catalog).mockResolvedValue({ version: 2, revision: 1, resources: [allowedFiles] })
    vi.mocked(api.operate).mockResolvedValue(emptyCollectionPage())
    render(<App api={api} />)

    const item = await screen.findByRole('treeitem', { name: /Allowed files/ })
    fireEvent.contextMenu(item)
    fireEvent.click(await screen.findByRole('menuitem', { name: /操作 list/ }))
    expect(await screen.findByLabelText('已聚焦 capability list')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '返回资源预览' }))

    await waitFor(() => expect(screen.queryByLabelText('已聚焦 capability list')).not.toBeInTheDocument())
    expect(await screen.findByText('这个 Collection 位置没有成员')).toBeInTheDocument()
    expect(api.operate).toHaveBeenCalledWith('1', 'storage/allowed-files', 'list', 'mfh.collection.list-request.v1', { version: 1, parent: '', cursor: '', limit: 64 })
  })

  it('does not restore a stale action after closing and reselecting the same Resource', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    const api = mockAPI(settings)
    vi.mocked(api.catalog).mockResolvedValue({ version: 2, revision: 1, resources: [allowedFiles] })
    vi.mocked(api.operate).mockResolvedValue(emptyCollectionPage())
    render(<App api={api} />)

    const item = await screen.findByRole('treeitem', { name: /Allowed files/ })
    fireEvent.contextMenu(item)
    fireEvent.click(await screen.findByRole('menuitem', { name: /操作 list/ }))
    expect(await screen.findByLabelText('已聚焦 capability list')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '关闭预览' }))
    await waitFor(() => expect(screen.queryByRole('complementary', { name: 'Allowed files 预览' })).not.toBeInTheDocument())
    fireEvent.click(item)

    expect(await screen.findByRole('complementary', { name: 'Allowed files 预览' })).toBeInTheDocument()
    expect(screen.queryByLabelText('已聚焦 capability list')).not.toBeInTheDocument()
    expect(await screen.findByText('这个 Collection 位置没有成员')).toBeInTheDocument()
  })

  it('clears a focused action when selecting a different Resource', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    const api = mockAPI(settings)
    vi.mocked(api.catalog).mockResolvedValue({ version: 2, revision: 1, resources: [allowedFiles, resource] })
    render(<App api={api} />)

    const filesItem = await screen.findByRole('treeitem', { name: /Allowed files/ })
    fireEvent.contextMenu(filesItem)
    fireEvent.click(await screen.findByRole('menuitem', { name: /操作 list/ }))
    expect(await screen.findByLabelText('已聚焦 capability list')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('treeitem', { name: /CPU/ }))

    expect(await screen.findByRole('complementary', { name: 'CPU 预览' })).toBeInTheDocument()
    expect(screen.queryByLabelText('已聚焦 capability list')).not.toBeInTheDocument()
    await waitFor(() => expect(api.snapshot).toHaveBeenCalledWith('1', 'metrics/cpu'))
  })

  it('creates and deletes profiles from the full settings tab', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<App api={api} />)
    fireEvent.click(await screen.findByRole('button', { name: `打开 ${profile.name} 的设置` }))
    fireEvent.click(screen.getByRole('button', { name: 'Profile' }))
    fireEvent.click(screen.getByRole('button', { name: '新建 Profile' }))
    fireEvent.click(screen.getByRole('radio', { name: /Legacy 兼容/ }))
    fireEvent.change(screen.getByLabelText('Profile 名称'), { target: { value: 'Lab' } })
    fireEvent.change(screen.getByLabelText('Profile ID'), { target: { value: 'lab' } })
    fireEvent.change(screen.getByLabelText('本机 Node ID'), { target: { value: '4' } })
    fireEvent.change(screen.getByLabelText('父 Node ID'), { target: { value: '1' } })
    fireEvent.change(screen.getByLabelText('父节点地址'), { target: { value: '127.0.0.1:7441' } })
    fireEvent.change(screen.getByLabelText('父节点公钥'), { target: { value: 'public-key' } })
    fireEvent.click(screen.getByRole('button', { name: '保存 Profile' }))
    await waitFor(() => expect(api.saveProfile).toHaveBeenCalledWith(expect.objectContaining({ id: 'lab', name: 'Lab' })))
    fireEvent.click(screen.getByRole('button', { name: `删除 Profile ${profile.name}` }))
    await waitFor(() => expect(api.deleteProfile).toHaveBeenCalledWith(profile.id, `DELETE ${profile.id}`))
  })

  it('keeps a one-time permit available after a failed admission attempt', async () => {
    const settings: Settings = { version: 2, profiles: [], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    vi.mocked(api.login).mockRejectedValue(new Error('首次接入需要一次性准入 Permit'))
    render(<App api={api} />)
    fireEvent.change(await screen.findByLabelText('Profile 名称'), { target: { value: 'Personal' } })
    fireEvent.change(screen.getByLabelText('父节点地址'), { target: { value: '127.0.0.1:7441' } })
    fireEvent.click(screen.getByRole('radio', { name: /已有 Permit/ }))
    const permit = screen.getByLabelText(/Enrollment Permit/)
    fireEvent.change(permit, { target: { value: '{"version":1}' } })
    fireEvent.click(screen.getByRole('button', { name: /使用 Permit 注册并连接/ }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('一次性准入 Permit')
    expect(alert).toHaveFocus()
    expect(permit).toHaveValue('{"version":1}')
  })

  it('prepares a public identity without activating or leaving the login screen', async () => {
    const initial: Settings = { version: 2, profiles: [], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const preparedSettings: Settings = { ...initial, profiles: [profile], updated_at_unix_ms: 2 }
    const api = mockAPI(initial)
    vi.mocked(api.settings).mockResolvedValueOnce(initial).mockResolvedValueOnce(preparedSettings)
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })

    render(<App api={api} />)
    fireEvent.change(await screen.findByLabelText('Profile 名称'), { target: { value: profile.name } })
    fireEvent.change(screen.getByLabelText('父节点地址'), { target: { value: profile.endpoint } })
    fireEvent.click(screen.getByRole('radio', { name: /已有 Permit/ }))
    fireEvent.click(screen.getByRole('button', { name: '生成本机公钥' }))

    expect(await screen.findByDisplayValue('desktop-public-key')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '登录 MyFlowHub' })).toBeInTheDocument()
    expect(api.login).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '复制本机公钥' }))
    await waitFor(() => expect(writeText).toHaveBeenCalledWith('desktop-public-key'))
    expect(screen.getByText('公钥已复制')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('父节点地址'), { target: { value: '127.0.0.1:7331' } })
    expect(screen.queryByDisplayValue('desktop-public-key')).not.toBeInTheDocument()
  })

  it('keeps Node ID and parent identity out of the default enrollment form and requires explicit TOFU', async () => {
    const settings: Settings = { version: 2, profiles: [], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    render(<App api={api} />)
    fireEvent.change(await screen.findByLabelText('Profile 名称'), { target: { value: 'Device' } })
    fireEvent.change(screen.getByLabelText('父节点地址'), { target: { value: '127.0.0.1:7441' } })

    expect(screen.queryByLabelText('本机 Node ID')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('父 Node ID')).not.toBeInTheDocument()
    const generatedID = (screen.getByLabelText('Profile ID') as HTMLInputElement).value
    expect(generatedID).toMatch(/^profile-[0-9a-f]{16}$/)
    fireEvent.change(screen.getByLabelText('Profile 名称'), { target: { value: 'Renamed Device' } })
    expect(screen.getByLabelText('Profile ID')).toHaveValue(generatedID)
    fireEvent.click(screen.getByRole('button', { name: '提交注册申请' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('请确认首次连接时信任')
    expect(api.login).not.toHaveBeenCalled()
    fireEvent.click(screen.getByLabelText(/首次连接时信任该端点返回/))
    fireEvent.click(screen.getByRole('button', { name: '提交注册申请' }))

    await waitFor(() => expect(api.login).toHaveBeenCalledWith(
      expect.objectContaining({ id: generatedID, enrollment_mode: 'authority', node_id: '' }),
      '',
      true,
    ))
  })

  it('reduces an enrolled authority profile to a direct reconnect action', async () => {
    const enrolledProfile: Profile = { ...profile, enrollment_mode: 'authority' }
    const settings: Settings = { version: 2, profiles: [enrolledProfile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    render(<App api={mockAPI(settings)} />)

    expect(await screen.findByText(/Node 2 · localhost:9540/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Enrollment Permit')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '连接父节点' })).toBeInTheDocument()
  })

  it('retries a pending Profile without Permit or TOFU', async () => {
    const pendingProfile: Profile = { ...profile, id: 'pending', name: 'Pending', enrollment_mode: 'authority', node_id: '' }
    const settings: Settings = { version: 2, profiles: [pendingProfile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    vi.mocked(api.profileStates).mockResolvedValue([{ profile_id: pendingProfile.id, state: 'pending', request_id: '00112233445566778899aabbccddeeff' }])
    render(<App api={api} />)
    fireEvent.click(await screen.findByRole('button', { name: '检查审批并连接' }))
    await waitFor(() => expect(api.login).toHaveBeenCalledWith(pendingProfile, '', false))
  })

  it('renders protected credential states without guessing from Profile Node ID', async () => {
    const items: Profile[] = [
      { ...profile, id: 'missing', name: 'Missing', enrollment_mode: 'authority', node_id: '' },
      { ...profile, id: 'device', name: 'Device', enrollment_mode: 'authority', node_id: '' },
      { ...profile, id: 'pending-state', name: 'Pending state', enrollment_mode: 'authority', node_id: '' },
      { ...profile, id: 'enrolled', name: 'Enrolled', enrollment_mode: 'authority', node_id: '' },
      { ...profile, id: 'broken', name: 'Broken', enrollment_mode: 'authority', node_id: '' },
      { ...profile, id: 'legacy', name: 'Legacy' },
    ]
    const settings: Settings = { version: 2, profiles: items, updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    vi.mocked(api.profileStates).mockResolvedValue([
      { profile_id: 'missing', state: 'missing' },
      { profile_id: 'device', state: 'device' },
      { profile_id: 'pending-state', state: 'pending', request_id: '0011' },
      { profile_id: 'enrolled', state: 'enrolled', node_id: '7' },
      { profile_id: 'broken', state: 'error', message: '凭据无法读取' },
      { profile_id: 'legacy', state: 'legacy', node_id: '2' },
    ])
    render(<App api={api} />)
    expect(await screen.findByText(/尚未准备 · localhost:9540/)).toBeInTheDocument()
    expect(screen.getByText(/设备已准备 · localhost:9540/)).toBeInTheDocument()
    expect(screen.getByText(/等待审批 · localhost:9540/)).toBeInTheDocument()
    expect(screen.getByText(/Node 7 · localhost:9540/)).toBeInTheDocument()
    expect(screen.getByText(/凭据异常 · localhost:9540/)).toBeInTheDocument()
    expect(screen.getByText(/Legacy · Node 2 · localhost:9540/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Broken 凭据异常/ }))
    expect(screen.getByText('凭据无法读取')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '需要恢复' })).toBeDisabled()
  })

  it('does not switch profiles while unsaved workspace changes are rejected', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile, secondProfile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<App api={api} />)
    await screen.findByRole('treeitem', { name: /CPU/ })
    fireEvent.click(screen.getByRole('button', { name: '添加 metrics/cpu 到工作区' }))
    fireEvent.click(screen.getByRole('button', { name: `打开 ${profile.name} 的设置` }))
    fireEvent.click(screen.getByRole('button', { name: 'Profile' }))
    fireEvent.click(screen.getByRole('button', { name: '启用' }))
    await waitFor(() => expect(window.confirm).toHaveBeenCalled())
    expect(api.switchProfile).not.toHaveBeenCalled()
  })

  it('returns to Profile selection without deleting the active Profile', async () => {
    const activeSettings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const inactiveSettings: Settings = { ...activeSettings, active_profile_id: undefined, updated_at_unix_ms: 2 }
    const api = mockAPI(activeSettings)
    vi.mocked(api.settings).mockResolvedValueOnce(activeSettings).mockResolvedValueOnce(inactiveSettings)
    render(<App api={api} />)
    fireEvent.click(await screen.findByRole('button', { name: `打开 ${profile.name} 的设置` }))
    fireEvent.click(screen.getByRole('button', { name: '返回 Profile 选择' }))
    await waitFor(() => expect(api.deactivateProfile).toHaveBeenCalled())
    expect(await screen.findByRole('tab', { name: '使用现有 Profile' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getAllByText(profile.name)).not.toHaveLength(0)
    expect(api.deleteProfile).not.toHaveBeenCalled()
  })

  it('requires confirmation before deleting a persisted view', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    vi.mocked(api.views).mockResolvedValue({ version: 3, views: [{ id: 'saved', name: 'Saved view', revision: 1, widgets: [] }] })
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<App api={api} />)
    fireEvent.mouseDown(await screen.findByRole('tab', { name: '视图' }), { button: 0, ctrlKey: false })
    fireEvent.click(await screen.findByRole('button', { name: '删除视图 Saved view' }))
    expect(window.confirm).toHaveBeenCalled()
    expect(api.deleteView).not.toHaveBeenCalled()
  })

  it('persists the theme per active profile without duplicating Profile controls in the top bar', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    render(<App api={mockAPI(settings)} />)
    fireEvent.click(await screen.findByRole('button', { name: '切换到深色主题' }))
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(window.localStorage.getItem('mfh.desktop.ui.v1:personal')).toContain('"theme":"dark"')
    fireEvent.keyDown(screen.getByRole('separator', { name: '调整节点列表和资源列表高度' }), { key: 'ArrowDown' })
    expect(window.localStorage.getItem('mfh.desktop.ui.v1:personal')).toContain('"explorer_split_ratio":0.38')
    fireEvent.click(screen.getByRole('button', { name: /^节点 1$/ }))
    expect(window.localStorage.getItem('mfh.desktop.ui.v1:personal')).toContain('"collapsed_explorer_pane":"node"')
    fireEvent.click(screen.getByRole('button', { name: '折叠资源路径 metrics' }))
    expect(window.localStorage.getItem('mfh.desktop.ui.v1:personal')).toContain('"expanded_resource_paths":[]')
    expect(screen.queryByLabelText('活动 Profile')).not.toBeInTheDocument()
  })
})
