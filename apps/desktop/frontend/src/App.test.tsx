import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from './api'
import { App } from './App'
import type { Profile, ResourceDescriptor, Settings, ViewDefinition } from './types'

const profile: Profile = {
  id: 'personal', name: 'Personal', node_id: '2', endpoint: 'localhost:9540', parent_node_id: '1', parent_public_key: 'key', auto_connect: true,
}
const resource: ResourceDescriptor = {
  id: { owner_node_id: '1', name: 'metrics/cpu' }, type: 'mfh.variable', type_version: 1,
  capabilities: [{ name: 'subscribe', permission: 'metrics.read', event_schema: 'metrics.v1', max_payload_bytes: 1024 }],
  limits: { max_payload_bytes: 1024 }, presentation: { label: 'CPU', renderer: 'mfh.variable' },
}

function mockAPI(settings: Settings): DesktopAPI {
  return {
    settings: vi.fn().mockResolvedValue(settings),
    saveProfile: vi.fn(), login: vi.fn(), switchProfile: vi.fn(), deleteProfile: vi.fn(),
    identity: vi.fn().mockResolvedValue({ node_id: '2', public_key: 'key' }), connect: vi.fn(), disconnect: vi.fn(),
    status: vi.fn().mockResolvedValue({ state: 'connected', endpoint: 'localhost:9540' }),
    catalog: vi.fn().mockResolvedValue({ version: 2, revision: 1, resources: [resource] }),
    topology: vi.fn().mockResolvedValue({ version: 1, epoch: 1, nodes: [{ node_id: '1', role: 'root', generation: 1 }] }),
    snapshot: vi.fn().mockResolvedValue({ value: 42 }), operate: vi.fn(), subscribe: vi.fn(), poll: vi.fn(), cancel: vi.fn(), uploadFile: vi.fn(),
    views: vi.fn().mockResolvedValue({ version: 1, views: [] }),
    saveView: vi.fn().mockImplementation(async (view: ViewDefinition) => ({ ...view, revision: 1 })), deleteView: vi.fn(),
  }
}

describe('desktop resource workspace', () => {
  it('shows a persistent-profile login when no profile is active', async () => {
    const settings: Settings = { version: 2, profiles: [], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    render(<App api={mockAPI(settings)} />)
    expect(await screen.findByRole('heading', { name: '登录 Profile' })).toBeInTheDocument()
    expect(screen.getByText(/permit 仅用于本次登录/)).toBeInTheDocument()
  })

  it('discovers resources, previews them, and adds them through a keyboard-equivalent action', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'session-only' }
    const api = mockAPI(settings)
    render(<App api={api} />)
    const resourceLabel = await screen.findByText('CPU')
    fireEvent.click(resourceLabel.closest('button')!)
    await waitFor(() => expect(api.snapshot).toHaveBeenCalledWith('1', 'metrics/cpu'))
    fireEvent.click(screen.getByRole('button', { name: '添加 metrics/cpu 到工作区' }))
    expect(screen.getByText('未保存')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /保存视图/ }))
    await waitFor(() => expect(api.saveView).toHaveBeenCalled())
  })

  it('creates, edits, and explicitly deletes profiles from the login shell', async () => {
    const settings: Settings = { version: 2, active_profile_id: profile.id, profiles: [profile], updated_at_unix_ms: 1, credential_mode: 'windows-dpapi-user' }
    const api = mockAPI(settings)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<App api={api} />)
    fireEvent.click(await screen.findByRole('button', { name: '新建 Profile' }))
    fireEvent.click(screen.getByRole('button', { name: `编辑 Profile ${profile.name}` }))
    expect(screen.getByLabelText('Profile 名称')).toHaveValue(profile.name)
    fireEvent.click(screen.getByRole('button', { name: `删除 Profile ${profile.name}` }))
    await waitFor(() => expect(api.deleteProfile).toHaveBeenCalledWith(profile.id, `DELETE ${profile.id}`))
  })
})
