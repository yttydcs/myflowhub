import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor, ResourceEvent } from '../types'
import { MAX_VISIBLE_EVENTS, prependBoundedEvents, ResourceRenderer, SchemaEditor } from './Renderer'

function mockAPI(overrides: Partial<DesktopAPI> = {}): DesktopAPI {
  return {
    settings: vi.fn(),
    profileStates: vi.fn(),
    prepareProfile: vi.fn(),
    saveProfile: vi.fn(),
    login: vi.fn(),
    switchProfile: vi.fn(),
    deactivateProfile: vi.fn(),
    deleteProfile: vi.fn(),
    identity: vi.fn(),
    connect: vi.fn(),
    disconnect: vi.fn(),
    status: vi.fn(),
    catalog: vi.fn(),
    topology: vi.fn(),
    snapshot: vi.fn(),
    operate: vi.fn(),
    subscribe: vi.fn(),
    poll: vi.fn(),
    cancel: vi.fn(),
    pickFile: vi.fn(),
    uploadFile: vi.fn(),
    views: vi.fn(),
    saveView: vi.fn(),
    deleteView: vi.fn(),
    ...overrides,
  }
}

function resource(type: string, capabilities: ResourceDescriptor['capabilities']): ResourceDescriptor {
  return {
    id: { owner_node_id: '1', name: 'system/example' },
    type,
    type_version: 1,
    capabilities,
    limits: { max_payload_bytes: 4096 },
  }
}

const health = {
  version: 1,
  state: 'running',
  started_at_unix_ms: 1_700_000_000_000,
  topology_epoch: 3,
  active_links: 2,
  active_subscriptions: 4,
}

function encodedEvent(value: unknown, revision = 7): ResourceEvent {
  return {
    kind: 'snapshot',
    owner_node_id: '1',
    resource_name: 'system/example',
    capability: 'subscribe',
    schema: 'mfh.management.health.v1',
    revision,
    value: btoa(JSON.stringify(value)),
  }
}

describe('schema-driven resource renderer', () => {
  it('keeps the live event buffer newest-first and bounded', () => {
    const events = Array.from({ length: MAX_VISIBLE_EVENTS + 5 }, (_, index) => encodedEvent({ index }, index + 1))
    const buffered = prependBoundedEvents([], events)
    expect(buffered).toHaveLength(MAX_VISIBLE_EVENTS)
    expect(buffered[0]!.revision).toBe(MAX_VISIBLE_EVENTS + 5)
    expect(buffered.at(-1)!.revision).toBe(6)
  })

  it('binds a numeric slider to provider bounds and step', () => {
    const onChange = vi.fn()
    render(<SchemaEditor schema={{ type: 'integer', title: 'Level', minimum: 0, maximum: 100, multiple_of: 1 }} value={20} rendererID="mfh.variable.slider.v1" onChange={onChange} />)
    const slider = screen.getByRole('slider', { name: 'Level' })
    expect(slider).toHaveAttribute('min', '0')
    expect(slider).toHaveAttribute('max', '100')
    expect(slider).toHaveAttribute('step', '1')
    fireEvent.change(slider, { target: { value: '42' } })
    expect(onChange).toHaveBeenCalledWith(42)
  })

  it('never renders a sensitive write-only field as plain text', () => {
    render(<SchemaEditor schema={{ type: 'string', title: 'Permit', write_only: true, sensitive: true, format: 'json' }} value="secret" onChange={vi.fn()} />)
    expect(screen.getByLabelText('Permit')).toHaveAttribute('type', 'password')
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  })

  it('switches compatible display components without calling the resource API again', async () => {
    const api = mockAPI({ snapshot: vi.fn().mockResolvedValue(health) })
    const onRendererChange = vi.fn()
    const descriptor = resource('mfh.variable', [
      { name: 'read', permission: 'system.health.read', output_schema: 'mfh.management.health.v1', max_payload_bytes: 4096 },
    ])
    render(<ResourceRenderer api={api} resource={descriptor} onRendererChange={onRendererChange} />)

    expect(await screen.findByText('running')).toBeInTheDocument()
    expect(api.snapshot).toHaveBeenCalledTimes(1)
    fireEvent.change(screen.getByRole('combobox', { name: /显示方式/ }), { target: { value: 'mfh.variable.raw.v1' } })
    expect(onRendererChange).toHaveBeenCalledWith('mfh.variable.raw.v1')
    expect(api.snapshot).toHaveBeenCalledTimes(1)
    expect(api.operate).not.toHaveBeenCalled()
  })

  it('stages a writable value and only mutates it after Apply', async () => {
    const api = mockAPI({
      subscribe: vi.fn().mockResolvedValue(19),
      poll: vi.fn().mockResolvedValue({ kind: 'event', payload: encodedEvent(health) }),
      cancel: vi.fn().mockResolvedValue(undefined),
      operate: vi.fn().mockResolvedValue({ schema: 'mfh.variable-write-result.v2', payload: { revision: 8 } }),
    })
    const descriptor = resource('mfh.variable', [
      { name: 'subscribe', permission: 'system.health.read', event_schema: 'mfh.management.health.v1', max_payload_bytes: 4096 },
      { name: 'write', permission: 'system.health.write', input_schema: 'mfh.variable-write.v2', max_payload_bytes: 4096 },
    ])
    render(<ResourceRenderer api={api} resource={descriptor} />)

    const state = await screen.findByRole('combobox', { name: 'State' })
    fireEvent.change(state, { target: { value: 'degraded' } })
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '应用' }))

    await waitFor(() => expect(api.operate).toHaveBeenCalledTimes(1))
    const request = vi.mocked(api.operate).mock.calls[0]![4] as { version: number; expected_revision: number; value: string }
    expect(request.version).toBe(2)
    expect(request.expected_revision).toBe(7)
    expect(JSON.parse(atob(request.value))).toEqual({ ...health, state: 'degraded' })
  })

  it('preserves a staged draft after a revision conflict', async () => {
    const api = mockAPI({
      subscribe: vi.fn().mockResolvedValue(19),
      poll: vi.fn().mockResolvedValue({ kind: 'event', payload: encodedEvent(health) }),
      cancel: vi.fn().mockResolvedValue(undefined),
      operate: vi.fn().mockRejectedValue(new Error('revision conflict')),
    })
    const descriptor = resource('mfh.variable', [
      { name: 'subscribe', permission: 'system.health.read', event_schema: 'mfh.management.health.v1', max_payload_bytes: 4096 },
      { name: 'write', permission: 'system.health.write', input_schema: 'mfh.variable-write.v2', max_payload_bytes: 4096 },
    ])
    render(<ResourceRenderer api={api} resource={descriptor} />)

    const state = await screen.findByRole('combobox', { name: 'State' })
    fireEvent.change(state, { target: { value: 'degraded' } })
    fireEvent.click(screen.getByRole('button', { name: '应用' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('revision conflict')
    expect(screen.getByRole('combobox', { name: 'State' })).toHaveValue('degraded')
  })

  it('generates a command form but requires an explicit execute action', async () => {
    const api = mockAPI({ operate: vi.fn().mockResolvedValue({ schema: 'mfh.provisioning.permit.v1', payload: { version: 1 } }) })
    const descriptor = resource('mfh.command', [
      { name: 'invoke', permission: 'system.admission.issue', input_schema: 'mfh.management.issue-permit.v1', output_schema: 'mfh.provisioning.permit.v1', max_payload_bytes: 4096 },
    ])
    render(<ResourceRenderer api={api} resource={descriptor} />)

    expect(screen.getByLabelText('Child Node Id')).toBeInTheDocument()
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText('Child Node Id'), { target: { value: '9' } })
    fireEvent.click(screen.getByRole('button', { name: '执行 invoke' }))
    await waitFor(() => expect(api.operate).toHaveBeenCalledTimes(1))
    expect(api.operate).toHaveBeenCalledWith('1', 'system/example', 'invoke', 'mfh.management.issue-permit.v1', expect.objectContaining({ child_node_id: '9' }))
  })

  it('uses the native file picker and keeps upload explicit', async () => {
    const api = mockAPI({
      pickFile: vi.fn().mockResolvedValue('C:\\tmp\\sample.bin'),
      uploadFile: vi.fn().mockResolvedValue({ transfer_id: 't-1' }),
    })
    const descriptor = resource('mfh.file', [
      { name: 'upload', permission: 'file.upload', max_payload_bytes: 4096 },
    ])
    render(<ResourceRenderer api={api} resource={descriptor} />)

    fireEvent.click(screen.getByRole('button', { name: '选择文件' }))
    await waitFor(() => expect(screen.getByDisplayValue('C:\\tmp\\sample.bin')).toBeInTheDocument())
    expect(api.uploadFile).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '开始上传' }))
    await waitFor(() => expect(api.uploadFile).toHaveBeenCalledWith('1', 'C:\\tmp\\sample.bin', 'sample.bin', 'application/octet-stream'))
  })

  it('shows bounded live controls and an explicit gap state', async () => {
    const gap = { ...encodedEvent({ reason: 'backpressure' }, 12), kind: 'gap' as const, gap_from: 8, gap_to: 11 }
    const api = mockAPI({
      subscribe: vi.fn().mockResolvedValue(31),
      poll: vi.fn()
        .mockResolvedValueOnce({ kind: 'event', payload: gap })
        .mockImplementation(() => new Promise(() => {})),
      cancel: vi.fn().mockResolvedValue(undefined),
    })
    const descriptor = resource('mfh.stream', [
      { name: 'subscribe', permission: 'file.progress', event_schema: 'mfh.file.progress.v1', max_payload_bytes: 4096 },
    ])
    render(<ResourceRenderer api={api} resource={descriptor} />)

    expect(await screen.findByText(/当前列表可能不连续/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '暂停' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '清空' })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: '筛选事件' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox')).toBeChecked()
  })
})
