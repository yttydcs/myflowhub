import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from './api'
import { App } from './App'
import type { Profile, ResourceDescriptor, Settings, ViewDefinition } from './types'

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

function mockAPI(settings: Settings): DesktopAPI {
  return {
    settings: vi.fn().mockResolvedValue(settings),
    prepareProfile: vi.fn().mockImplementation(async (next: Profile) => ({ profile: next, identity: { node_id: next.node_id, public_key: 'desktop-public-key' } })),
    saveProfile: vi.fn().mockImplementation(async (next: Profile) => next),
    login: vi.fn().mockImplementation(async (next: Profile) => next),
    switchProfile: vi.fn(),
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

  it('shows connection, device identity, and explicit admission choices when no profile is active', async () => {
    const settings: Settings = { version: 2, profiles: [], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    render(<App api={mockAPI(settings)} />)
    expect(await screen.findByRole('heading', { name: '登录 MyFlowHub' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '连接到父节点' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '准备这台设备' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '选择准入方式' })).toBeInTheDocument()
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
    fireEvent.change(screen.getByLabelText('Profile ID'), { target: { value: 'personal' } })
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
    fireEvent.change(screen.getByLabelText('Profile ID'), { target: { value: profile.id } })
    fireEvent.change(screen.getByLabelText('父节点地址'), { target: { value: profile.endpoint } })
    fireEvent.click(screen.getByRole('button', { name: '提前生成公钥' }))

    expect(await screen.findByDisplayValue('desktop-public-key')).toBeInTheDocument()
    expect(screen.getByText(/Profile 已保存，但尚未登录或连接/)).toBeInTheDocument()
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
    fireEvent.change(screen.getByLabelText('Profile ID'), { target: { value: 'device' } })
    fireEvent.change(screen.getByLabelText('父节点地址'), { target: { value: '127.0.0.1:7441' } })

    expect(screen.queryByLabelText('本机 Node ID')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('父 Node ID')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '提交注册申请' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('确认信任当前父节点端点')
    expect(api.login).not.toHaveBeenCalled()
    fireEvent.click(screen.getByLabelText(/首次连接时信任该端点返回/))
    fireEvent.click(screen.getByRole('button', { name: '提交注册申请' }))

    await waitFor(() => expect(api.login).toHaveBeenCalledWith(
      expect.objectContaining({ enrollment_mode: 'authority', node_id: '' }),
      '',
      true,
    ))
  })

  it('reduces an enrolled authority profile to a direct reconnect action', async () => {
    const enrolledProfile: Profile = { ...profile, enrollment_mode: 'authority' }
    const settings: Settings = { version: 2, profiles: [enrolledProfile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    render(<App api={mockAPI(settings)} />)

    expect(await screen.findByText(`已注册为 Node ${enrolledProfile.node_id}`)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: '选择准入方式' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Enrollment Permit')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '连接父节点' })).toBeInTheDocument()
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
