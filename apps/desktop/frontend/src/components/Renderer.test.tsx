import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor, ResourceEvent } from '../types'
import { MAX_VISIBLE_EVENTS, prependBoundedEvents, ResourceOperationPanel, ResourceRenderer, ResourceRendererSelector, SchemaEditor } from './Renderer'

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

function collectionResource(
  name = 'system/example',
  getOutput = 'mfh.collection.member.v1',
  readable = false,
): ResourceDescriptor {
  return {
    ...resource('mfh.collection', [
      { name: 'list', permission: 'ignored.list', input_schema: 'mfh.collection.list-request.v1', output_schema: 'mfh.collection.page.v1', max_payload_bytes: 4096 },
      { name: 'get', permission: 'ignored.get', input_schema: 'mfh.collection.member-request.v1', output_schema: getOutput, max_payload_bytes: 4096 },
      ...(readable ? [{ name: 'read', permission: 'ignored.read', input_schema: 'mfh.filesystem.read-request.v1', output_schema: 'mfh.filesystem.content.v1', max_payload_bytes: 180_000 }] : []),
    ]),
    id: { owner_node_id: '1', name },
  }
}

function collectionMember(key: string, overrides: Record<string, unknown> = {}) {
  return { key, kind: 'file', label: key, capabilities: ['get'], ...overrides }
}

function collectionPage(parent: string, members: unknown[], nextCursor = '', revision = 1) {
  return { schema: 'mfh.collection.page.v1', payload: { version: 1, revision, parent, members, next_cursor: nextCursor } }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((success, failure) => { resolve = success; reject = failure })
  return { promise, resolve, reject }
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
    render(<><ResourceRendererSelector resource={descriptor} onRendererChange={onRendererChange} /><ResourceRenderer api={api} resource={descriptor} /></>)

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

    expect(screen.getByLabelText('invoke 操作面板')).toBeInTheDocument()
    expect(screen.getByLabelText('Child Node Id')).toBeInTheDocument()
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText('Child Node Id'), { target: { value: '9' } })
    fireEvent.click(screen.getByRole('button', { name: '执行 invoke' }))
    await waitFor(() => expect(api.operate).toHaveBeenCalledTimes(1))
    expect(api.operate).toHaveBeenCalledWith('1', 'system/example', 'invoke', 'mfh.management.issue-permit.v1', expect.objectContaining({ child_node_id: '9' }))
  })

  it('uses Advanced JSON for a capability without an input schema and sends only after Execute', async () => {
    const api = mockAPI({ operate: vi.fn().mockResolvedValue({ schema: '', payload: { accepted: true } }) })
    const descriptor = resource('mfh.collection', [
      { name: 'custom', permission: 'ignored.custom', max_payload_bytes: 4096 },
    ])
    render(<ResourceOperationPanel api={api} resource={descriptor} capabilityName="custom" />)

    const draft = screen.getByRole('textbox', { name: /输入/ })
    expect(screen.getByText(/Advanced JSON/)).toBeInTheDocument()
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.change(draft, { target: { value: '{"value":7}' } })
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '执行 custom' }))
    await waitFor(() => expect(api.operate).toHaveBeenCalledWith('1', 'system/example', 'custom', '', { value: 7 }))
  })

  it('keeps the draft after authoritative Forbidden and blocks a stale descriptor', async () => {
    const api = mockAPI({ operate: vi.fn().mockRejectedValue(new Error('Forbidden: custom denied')) })
    const descriptor = resource('mfh.collection', [
      { name: 'custom', permission: 'ignored.custom', max_payload_bytes: 4096 },
    ])
    const { rerender } = render(<ResourceOperationPanel api={api} resource={descriptor} capabilityName="custom" />)
    const draft = screen.getByRole('textbox', { name: /输入/ })
    fireEvent.change(draft, { target: { value: '{"keep":"me"}' } })
    fireEvent.click(screen.getByRole('button', { name: '执行 custom' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Forbidden: custom denied')
    expect(draft).toHaveValue('{"keep":"me"}')

    rerender(<ResourceOperationPanel api={api} resource={{ ...descriptor, capabilities: [] }} capabilityName="custom" />)
    expect(screen.getByRole('alert')).toHaveTextContent('当前 descriptor 已不再声明')
    expect(screen.queryByRole('button', { name: '执行 custom' })).not.toBeInTheDocument()
    expect(api.operate).toHaveBeenCalledTimes(1)
  })

  it('keeps observe and session capabilities off the ordinary operate path', () => {
    const api = mockAPI()
    const descriptor = resource('mfh.collection', [
      { name: 'subscribe', permission: 'ignored.subscribe', event_schema: 'event.v1', max_payload_bytes: 4096 },
      { name: 'open', permission: 'ignored.open', input_schema: 'open.v1', max_payload_bytes: 4096 },
    ])
    const { rerender } = render(<ResourceOperationPanel api={api} resource={descriptor} capabilityName="subscribe" />)
    expect(screen.getByText(/事件订阅通道/)).toBeInTheDocument()
    rerender(<ResourceOperationPanel api={api} resource={descriptor} capabilityName="open" />)
    expect(screen.getByText(/session 通道/)).toBeInTheDocument()
    expect(api.operate).not.toHaveBeenCalled()
  })

  it('keeps a mixed operation plus event capability on explicit operate', async () => {
    const api = mockAPI({ operate: vi.fn().mockResolvedValue({ schema: 'mfh.collection.member.v1', payload: collectionMember('done') }) })
    const descriptor = resource('mfh.collection', [
      { name: 'mixed', permission: 'ignored.mixed', input_schema: 'mfh.collection.member-request.v1', output_schema: 'mfh.collection.member.v1', event_schema: 'event.v1', max_payload_bytes: 4096 },
    ])
    render(<ResourceOperationPanel api={api} resource={descriptor} capabilityName="mixed" />)
    expect(screen.queryByText(/事件订阅通道/)).not.toBeInTheDocument()
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText('Key'), { target: { value: 'member-1' } })
    fireEvent.click(screen.getByRole('button', { name: '执行 mixed' }))
    await waitFor(() => expect(api.operate).toHaveBeenCalledWith('1', 'system/example', 'mixed', 'mfh.collection.member-request.v1', { version: 1, key: 'member-1' }))
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

describe('Collection renderer', () => {
  it('shows list metadata without calling get when the member does not declare it', async () => {
    const listed = collectionMember('metadata-only', {
      label: 'Metadata only',
      content_type: 'application/octet-stream',
      capabilities: [],
      attributes: { origin: 'list-page' },
    })
    const api = mockAPI({ operate: vi.fn().mockResolvedValue(collectionPage('', [listed])) })
    render(<ResourceRenderer api={api} resource={collectionResource()} />)

    fireEvent.click(await screen.findByRole('button', { name: /Metadata only/ }))
    expect(screen.getByText('列表 metadata')).toBeInTheDocument()
    expect(screen.getByText('list-page')).toBeInTheDocument()
    expect(screen.getByText(/未声明 get，仅展示列表 metadata/)).toBeInTheDocument()
    expect(api.operate).toHaveBeenCalledTimes(1)
    expect(vi.mocked(api.operate).mock.calls.some((call) => call[2] === 'get')).toBe(false)
  })

  it('shows metadata and a descriptor error without get when Resource get is missing or incompatible', async () => {
    const listed = collectionMember('needs-get', { label: 'Needs get', attributes: { source: 'page' } })
    const api = mockAPI({ operate: vi.fn().mockResolvedValue(collectionPage('', [listed])) })
    const missing = collectionResource('missing-get')
    missing.capabilities = missing.capabilities.filter((capability) => capability.name !== 'get')
    const first = render(<ResourceRenderer api={api} resource={missing} />)
    fireEvent.click(await screen.findByRole('button', { name: /Needs get/ }))
    expect(screen.getByText('page')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('未声明 get capability')
    expect(api.operate).toHaveBeenCalledTimes(1)
    first.unmount()

    vi.mocked(api.operate).mockClear()
    const incompatible = collectionResource('incompatible-get')
    incompatible.capabilities = incompatible.capabilities.map((capability) => capability.name === 'get'
      ? { ...capability, input_schema: 'vendor.member-request.v2' }
      : capability)
    render(<ResourceRenderer api={api} resource={incompatible} />)
    fireEvent.click(await screen.findByRole('button', { name: /Needs get/ }))
    expect(screen.getByText('page')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('与 Collection member request 不兼容')
    expect(api.operate).toHaveBeenCalledTimes(1)
    expect(vi.mocked(api.operate).mock.calls.some((call) => call[2] === 'get')).toBe(false)
  })

  it('uses exact list payloads, opaque cursors and bounded pagination', async () => {
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability, _schema, payload: { cursor?: string }) => {
        if (capability !== 'list') throw new Error('unexpected capability')
        return payload.cursor
          ? collectionPage('', [collectionMember('gamma')], '', 4)
          : collectionPage('', [collectionMember('alpha'), collectionMember('beta')], 'opaque:do-not-parse', 4)
      }),
    })
    render(<ResourceRenderer api={api} resource={collectionResource()} />)

    expect(await screen.findByText('alpha')).toBeInTheDocument()
    expect(api.operate).toHaveBeenNthCalledWith(1, '1', 'system/example', 'list', 'mfh.collection.list-request.v1', {
      version: 1, parent: '', cursor: '', limit: 64,
    })
    fireEvent.click(screen.getByRole('button', { name: '加载下一页' }))
    expect(await screen.findByText('gamma')).toBeInTheDocument()
    expect(api.operate).toHaveBeenNthCalledWith(2, '1', 'system/example', 'list', 'mfh.collection.list-request.v1', {
      version: 1, parent: '', cursor: 'opaque:do-not-parse', limit: 64,
    })
  })

  it('shows empty/error states and retries from the current parent', async () => {
    const api = mockAPI({ operate: vi.fn().mockRejectedValueOnce(new Error('temporary list failure')).mockResolvedValueOnce(collectionPage('', [])) })
    render(<ResourceRenderer api={api} resource={collectionResource()} />)

    expect(await screen.findByRole('alert')).toHaveTextContent('temporary list failure')
    fireEvent.click(screen.getByRole('button', { name: '重试' }))
    expect(await screen.findByText('这个 Collection 位置没有成员')).toBeInTheDocument()
    expect(api.operate).toHaveBeenCalledTimes(2)
  })

  it('enters only a member-declared directory and supports breadcrumb/back navigation', async () => {
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability, _schema, payload: { parent?: string }) => {
        if (capability !== 'list') throw new Error('unexpected capability')
        return payload.parent === 'docs'
          ? collectionPage('docs', [collectionMember('docs/readme', { label: 'README' })])
          : collectionPage('', [collectionMember('docs', { kind: 'directory', label: 'Docs', capabilities: ['get', 'list'] })])
      }),
    })
    render(<ResourceRenderer api={api} resource={collectionResource()} />)

    fireEvent.click(await screen.findByRole('button', { name: '进入 Docs' }))
    expect(await screen.findByText('README')).toBeInTheDocument()
    expect(api.operate).toHaveBeenLastCalledWith('1', 'system/example', 'list', 'mfh.collection.list-request.v1', expect.objectContaining({ parent: 'docs', cursor: '' }))
    fireEvent.click(screen.getByRole('button', { name: '返回' }))
    expect(await screen.findByRole('button', { name: '进入 Docs' })).toBeInTheDocument()
    expect(api.operate).toHaveBeenLastCalledWith('1', 'system/example', 'list', 'mfh.collection.list-request.v1', expect.objectContaining({ parent: '' }))
  })

  it('ignores a stale get result after rapid member selection', async () => {
    const alpha = deferred<{ schema: string; payload: unknown }>()
    const beta = deferred<{ schema: string; payload: unknown }>()
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability, _schema, payload: { key?: string }) => {
        if (capability === 'list') return collectionPage('', [collectionMember('alpha'), collectionMember('beta')])
        if (payload.key === 'alpha') return alpha.promise
        return beta.promise
      }),
    })
    render(<ResourceRenderer api={api} resource={collectionResource()} />)
    fireEvent.click(await screen.findByRole('button', { name: /^alpha/ }))
    fireEvent.click(screen.getByRole('button', { name: /^beta/ }))
    beta.resolve({ schema: 'mfh.collection.member.v1', payload: collectionMember('beta', { label: 'Beta current' }) })
    expect(await screen.findByText('Beta current')).toBeInTheDocument()
    alpha.resolve({ schema: 'mfh.collection.member.v1', payload: collectionMember('alpha', { label: 'Alpha stale' }) })
    await Promise.resolve()
    expect(screen.queryByText('Alpha stale')).not.toBeInTheDocument()
  })

  it('renders a domain get result according to the descriptor output schema', async () => {
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability) => capability === 'list'
        ? collectionPage('', [collectionMember('flow-1', { label: 'Flow one', schema: 'mfh.flow.definition.v1' })])
        : { schema: 'mfh.flow.definition.v1', payload: { version: 1, flow_id: 'flow-1', revision: 3, name: 'Demo Flow', nodes: [], edges: [] } }),
    })
    render(<ResourceRenderer api={api} resource={collectionResource('any/provider-name', 'mfh.flow.definition.v1')} />)
    fireEvent.click(await screen.findByRole('button', { name: /Flow one/ }))
    expect(await screen.findByText('Demo Flow')).toBeInTheDocument()
    expect(api.operate).toHaveBeenLastCalledWith('1', 'any/provider-name', 'get', 'mfh.collection.member-request.v1', { version: 1, key: 'flow-1' })
  })

  it('gets then reads a member, warns about sniffed content type and keeps HTML escaped', async () => {
    const listed = collectionMember('readme', { label: 'README', content_type: 'application/octet-stream', capabilities: ['get', 'read'], attributes: { revision: 'r1' } })
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability) => {
        if (capability === 'list') return collectionPage('', [listed])
        if (capability === 'get') return { schema: 'mfh.collection.member.v1', payload: listed }
        return { schema: 'mfh.filesystem.content.v1', payload: { version: 1, key: 'readme', content_type: 'text/html', encoding: 'utf-8', data: '<b>safe</b>', size: 11, modified_unix_ms: 7, revision: 'r1' } }
      }),
    })
    render(<ResourceRenderer api={api} resource={collectionResource('unprivileged/files', 'mfh.collection.member.v1', true)} />)
    fireEvent.click(await screen.findByRole('button', { name: /README/ }))
    fireEvent.click(await screen.findByRole('button', { name: '读取文件内容' }))
    expect(await screen.findByText('<b>safe</b>')).toBeInTheDocument()
    expect(screen.getByText(/content_type.*不一致/)).toBeInTheDocument()
    expect(screen.getByText(/HTML.*不会执行/)).toBeInTheDocument()
    expect(document.querySelector('.filesystem-content b')).toBeNull()
    expect(api.operate).toHaveBeenLastCalledWith('1', 'unprivileged/files', 'read', 'mfh.filesystem.read-request.v1', {
      version: 1, key: 'readme', max_bytes: 131072, expected_revision: 'r1',
    })
  })

  it('validates the filesystem envelope but renders decoded JSON instead of its wrapper schema', async () => {
    const data = JSON.stringify({ fixture: '<safe>', nested: { enabled: true } })
    const size = new TextEncoder().encode(data).byteLength
    const listed = collectionMember('valid.json', {
      label: 'valid.json',
      content_type: 'application/json',
      schema: 'mfh.filesystem.content.v1',
      capabilities: ['get', 'read'],
      attributes: { revision: 'r-json' },
    })
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability) => {
        if (capability === 'list') return collectionPage('', [listed])
        if (capability === 'get') return { schema: 'mfh.collection.member.v1', payload: listed }
        return {
          schema: 'mfh.filesystem.content.v1',
          payload: {
            version: 1,
            key: 'valid.json',
            content_type: 'application/json',
            encoding: 'utf-8',
            data,
            size,
            modified_unix_ms: 7,
            revision: 'r-json',
          },
        }
      }),
    })
    render(<ResourceRenderer api={api} resource={collectionResource('allowed/files', 'mfh.collection.member.v1', true)} />)

    fireEvent.click(await screen.findByRole('button', { name: /valid\.json/ }))
    fireEvent.click(await screen.findByRole('button', { name: '读取文件内容' }))

    const content = await screen.findByLabelText('文件内容 valid.json')
    expect(within(content).getByText('application/json')).toBeInTheDocument()
    expect(within(content).getByText(`${size} bytes · revision r-json`)).toBeInTheDocument()
    expect(within(content).getByText('fixture')).toBeInTheDocument()
    expect(within(content).getByText('<safe>')).toBeInTheDocument()
    expect(within(content).queryByText('Version')).not.toBeInTheDocument()
    expect(within(content).queryByText('Content Type')).not.toBeInTheDocument()
    expect(content.querySelector('safe')).toBeNull()
  })

  it('creates, replaces and revokes only Blob URLs for raster previews', async () => {
    const createObjectURL = vi.fn().mockReturnValueOnce('blob:first').mockReturnValueOnce('blob:second')
    const revokeObjectURL = vi.fn()
    const oldCreate = URL.createObjectURL
    const oldRevoke = URL.revokeObjectURL
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: createObjectURL })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: revokeObjectURL })
    const listed = collectionMember('pixel', { label: 'Pixel', content_type: 'image/png', capabilities: ['get', 'read'] })
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability) => {
        if (capability === 'list') return collectionPage('', [listed])
        if (capability === 'get') return { schema: 'mfh.collection.member.v1', payload: listed }
        return { schema: 'mfh.filesystem.content.v1', payload: { version: 1, key: 'pixel', content_type: 'image/png', encoding: 'base64', data: 'iVBORw==', size: 4, modified_unix_ms: 7, revision: 'r1' } }
      }),
    })
    try {
      const rendered = render(<ResourceRenderer api={api} resource={collectionResource('images', 'mfh.collection.member.v1', true)} />)
      fireEvent.click(await screen.findByRole('button', { name: /Pixel/ }))
      fireEvent.click(await screen.findByRole('button', { name: '读取文件内容' }))
      expect(await screen.findByRole('img', { name: 'Pixel' })).toHaveAttribute('src', 'blob:first')
      fireEvent.click(screen.getByRole('button', { name: '读取文件内容' }))
      await waitFor(() => expect(screen.getByRole('img', { name: 'Pixel' })).toHaveAttribute('src', 'blob:second'))
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:first')
      rendered.unmount()
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:second')
    } finally {
      if (oldCreate) Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: oldCreate })
      else delete (URL as unknown as { createObjectURL?: unknown }).createObjectURL
      if (oldRevoke) Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: oldRevoke })
      else delete (URL as unknown as { revokeObjectURL?: unknown }).revokeObjectURL
    }
  })

  it('does not expose read unless member and Resource schemas both opt in', async () => {
    const listed = collectionMember('guarded', { label: 'Guarded', capabilities: ['get', 'read'] })
    const descriptor = collectionResource('guarded', 'mfh.collection.member.v1', true)
    descriptor.capabilities = descriptor.capabilities.map((capability) => capability.name === 'read'
      ? { ...capability, output_schema: 'vendor.unsafe-content.v1' }
      : capability)
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, _name, capability) => capability === 'list'
        ? collectionPage('', [listed])
        : { schema: 'mfh.collection.member.v1', payload: listed }),
    })
    render(<ResourceRenderer api={api} resource={descriptor} />)
    fireEvent.click(await screen.findByRole('button', { name: /Guarded/ }))
    await waitFor(() => expect(api.operate).toHaveBeenCalledTimes(2))
    expect(screen.queryByRole('button', { name: '读取文件内容' })).not.toBeInTheDocument()
    expect(vi.mocked(api.operate).mock.calls.some((call) => call[2] === 'read')).toBe(false)
  })

  it('drops stale list results when the Resource identity changes', async () => {
    const stale = deferred<{ schema: string; payload: unknown }>()
    const api = mockAPI({
      operate: vi.fn().mockImplementation(async (_owner, name) => name === 'old' ? stale.promise : collectionPage('', [collectionMember('new-only')]))
    })
    const rendered = render(<ResourceRenderer api={api} resource={collectionResource('old')} />)
    rendered.rerender(<ResourceRenderer api={api} resource={collectionResource('new')} />)
    expect(await screen.findByText('new-only')).toBeInTheDocument()
    stale.resolve(collectionPage('', [collectionMember('stale-only')]))
    await Promise.resolve()
    expect(screen.queryByText('stale-only')).not.toBeInTheDocument()
  })
})
