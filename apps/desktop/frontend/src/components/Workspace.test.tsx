import { DndContext } from '@dnd-kit/core'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor, ViewDefinition, ViewLayoutNode, ViewWidget } from '../types'
import { Workspace } from './Workspace'

const widget = (id: string): ViewWidget => ({
  id,
  owner_node_id: '2',
  resource_name: `system/${id}`,
  renderer: 'mfh.variable',
})

const leaf = (widgetID: string): ViewLayoutNode => ({ kind: 'leaf', widget_id: widgetID })

function view(layoutRoot: ViewLayoutNode, ids = ['a', 'b']): ViewDefinition {
  return {
    id: 'view',
    name: 'View',
    revision: 0,
    widgets: ids.map(widget),
    layout_root: layoutRoot,
  }
}

function renderWorkspace(candidate: ViewDefinition, onChange = vi.fn(), dockPreview?: ViewDefinition) {
  render(
    <DndContext>
      <Workspace
        api={{} as DesktopAPI}
        resources={[]}
        view={candidate}
        dirty
        saving={false}
        dockPreview={dockPreview ? { view: dockPreview, widgetID: 'c', description: '停靠预览' } : undefined}
        onChange={onChange}
        onSave={vi.fn()}
        onError={vi.fn()}
      />
    </DndContext>,
  )
  return onChange
}

function renderResourceWorkspace(
  descriptor: ResourceDescriptor,
  api: DesktopAPI,
  onChange = vi.fn(),
) {
  const candidate = view(leaf('action'), ['action'])
  candidate.widgets[0] = {
    ...candidate.widgets[0]!,
    owner_node_id: descriptor.id.owner_node_id,
    resource_name: descriptor.id.name,
    renderer: descriptor.type,
  }
  const rendered = render(
    <DndContext>
      <Workspace
        api={api}
        resources={[descriptor]}
        view={candidate}
        dirty
        saving={false}
        onChange={onChange}
        onSave={vi.fn()}
        onError={vi.fn()}
      />
    </DndContext>,
  )
  return { ...rendered, candidate, onChange }
}

describe('workspace panels', () => {
  it('renders one widget as the only full layout leaf', () => {
    renderWorkspace(view(leaf('a'), ['a']))
    expect(screen.getByRole('article', { name: '组件 a' })).toBeInTheDocument()
    expect(document.querySelectorAll('.workspace-layout-leaf')).toHaveLength(1)
    expect(screen.queryByRole('separator')).not.toBeInTheDocument()
  })

  it('recursively renders mixed horizontal and vertical n-ary splits', () => {
    renderWorkspace(view({
      kind: 'split',
      axis: 'horizontal',
      weights: [0.4, 0.6],
      children: [
        leaf('a'),
        { kind: 'split', axis: 'vertical', weights: [0.5, 0.5], children: [leaf('b'), leaf('c')] },
      ],
    }, ['a', 'b', 'c']))
    expect(screen.getAllByRole('article')).toHaveLength(3)
    expect(screen.getByRole('separator', { name: '调整 a 与 b 的左右比例' })).toHaveAttribute('aria-orientation', 'vertical')
    expect(screen.getByRole('separator', { name: '调整 b 与 c 的上下比例' })).toHaveAttribute('aria-orientation', 'horizontal')
    expect(screen.queryByRole('button', { name: /左右|上下|交换/ })).not.toBeInTheDocument()
  })

  it('resizes an exact divider with the keyboard and persists normalized weights', () => {
    const onChange = renderWorkspace(view({
      kind: 'split', axis: 'horizontal', weights: [0.5, 0.5], children: [leaf('a'), leaf('b')],
    }))
    const separator = screen.getByRole('separator', { name: '调整 a 与 b 的左右比例' })
    expect(separator).toHaveAttribute('aria-valuetext', '前一面板 50%，后一面板 50%')
    fireEvent.keyDown(separator, { key: 'ArrowRight' })
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      layout_root: expect.objectContaining({ weights: [0.52, 0.48] }),
    }))
  })

  it('previews pointer resizing through animation frames and persists only on release', () => {
    class TestPointerEvent extends MouseEvent {
      readonly pointerId: number

      constructor(type: string, init: PointerEventInit = {}) {
        super(type, init)
        this.pointerId = init.pointerId ?? 1
      }
    }
    Object.defineProperty(window, 'PointerEvent', { configurable: true, value: TestPointerEvent })
    const frames: FrameRequestCallback[] = []
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      frames.push(callback)
      return frames.length
    })

    const onChange = renderWorkspace(view({
      kind: 'split', axis: 'horizontal', weights: [0.5, 0.5], children: [leaf('a'), leaf('b')],
    }))
    const separator = screen.getByRole('separator', { name: '调整 a 与 b 的左右比例' })
    const grid = separator.parentElement as HTMLDivElement
    const [leading, , trailing] = Array.from(grid.children) as HTMLElement[]
    vi.spyOn(leading!, 'getBoundingClientRect').mockReturnValue({
      x: 0, y: 0, left: 0, top: 0, right: 496, bottom: 700, width: 496, height: 700, toJSON: () => ({}),
    })
    vi.spyOn(trailing!, 'getBoundingClientRect').mockReturnValue({
      x: 504, y: 0, left: 504, top: 0, right: 1000, bottom: 700, width: 496, height: 700, toJSON: () => ({}),
    })
    let captured = false
    separator.setPointerCapture = vi.fn(() => { captured = true })
    separator.hasPointerCapture = vi.fn(() => captured)
    separator.releasePointerCapture = vi.fn(() => { captured = false })

    fireEvent.pointerDown(separator, { pointerId: 1, clientX: 500, clientY: 350 })
    act(() => frames.shift()?.(0))
    fireEvent.pointerMove(separator, { pointerId: 1, clientX: 400, clientY: 350 })
    act(() => frames.shift()?.(16))
    expect(onChange).not.toHaveBeenCalled()
    expect(grid.style.gridTemplateColumns).toContain('0.39919354838709675fr')
    fireEvent.pointerUp(separator, { pointerId: 1, clientX: 400, clientY: 350 })
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      layout_root: expect.objectContaining({ weights: [expect.closeTo(396 / 992, 5), expect.closeTo(596 / 992, 5)] }),
    }))
  })

  it('renders the hypothetical layout preview without replacing live panels', () => {
    const current = view({ kind: 'split', axis: 'horizontal', weights: [0.5, 0.5], children: [leaf('a'), leaf('b')] })
    const preview = view({
      kind: 'split', axis: 'horizontal', weights: [1 / 3, 1 / 3, 1 / 3], children: [leaf('a'), leaf('c'), leaf('b')],
    }, ['a', 'b', 'c'])
    renderWorkspace(current, vi.fn(), preview)
    expect(screen.getAllByRole('article')).toHaveLength(2)
    expect(document.querySelectorAll('.workspace-preview-leaf')).toHaveLength(3)
    expect(document.querySelector('.workspace-preview-leaf.is-highlighted')).toBeInTheDocument()
  })

  it('persists renderer selection on the matching View widget', async () => {
    const onChange = vi.fn()
    const candidate = view(leaf('health'), ['health'])
    candidate.widgets[0] = { ...candidate.widgets[0]!, resource_name: 'system/health', renderer: 'mfh.structured.health.v1' }
    const descriptor: ResourceDescriptor = {
      id: { owner_node_id: '2', name: 'system/health' },
      type: 'mfh.variable',
      type_version: 1,
      capabilities: [{ name: 'read', permission: 'system.health.read', output_schema: 'mfh.management.health.v1', max_payload_bytes: 4096 }],
      limits: { max_payload_bytes: 4096 },
    }
    render(
      <DndContext>
        <Workspace
          api={{ snapshot: vi.fn().mockResolvedValue({ version: 1, state: 'running' }) } as unknown as DesktopAPI}
          resources={[descriptor]}
          view={candidate}
          dirty
          saving={false}
          onChange={onChange}
          onSave={vi.fn()}
          onError={vi.fn()}
        />
      </DndContext>,
    )

    const selector = await screen.findByRole('combobox', { name: /显示方式/ })
    expect(selector.closest('.widget-header')).not.toBeNull()
    expect(document.querySelector('.widget-body .widget-renderer-selector')).not.toBeInTheDocument()
    expect(screen.queryByText('突出运行状态与组件检查')).not.toBeInTheDocument()
    fireEvent.change(selector, { target: { value: 'mfh.variable.raw.v1' } })
    await waitFor(() => expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      widgets: [expect.objectContaining({ id: 'health', renderer: 'mfh.variable.raw.v1' })],
    })))
  })

  it('derives compact pane density from ResizeObserver without changing the View', async () => {
    const onChange = vi.fn()
    class TestResizeObserver {
      constructor(private readonly callback: ResizeObserverCallback) {}
      observe(target: Element) {
        this.callback([{ target, contentRect: { width: 320, height: 220 } } as ResizeObserverEntry], this as unknown as ResizeObserver)
      }
      disconnect() {}
      unobserve() {}
    }
    vi.stubGlobal('ResizeObserver', TestResizeObserver)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      callback(0)
      return 1
    })
    try {
      renderWorkspace(view(leaf('a'), ['a']), onChange)
      await waitFor(() => expect(screen.getByRole('article', { name: '组件 a' })).toHaveClass('density-compact'))
      expect(onChange).not.toHaveBeenCalled()
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('opens a shared action panel without calling the API until explicit Execute', async () => {
    const operate = vi.fn().mockResolvedValue({ schema: '', payload: { accepted: true } })
    const descriptor: ResourceDescriptor = {
      id: { owner_node_id: '2', name: 'arbitrary/provider' },
      type: 'example.custom',
      type_version: 1,
      capabilities: [{ name: 'custom', permission: 'not-used-for-dispatch', max_payload_bytes: 4096 }],
      limits: { max_payload_bytes: 4096 },
    }
    renderResourceWorkspace(descriptor, { operate } as unknown as DesktopAPI)

    expect(operate).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '操作 custom' }))
    expect(screen.getByLabelText('custom 操作面板')).toBeInTheDocument()
    expect(operate).not.toHaveBeenCalled()
    fireEvent.change(screen.getByRole('textbox', { name: /输入/ }), { target: { value: '{"value":9}' } })
    expect(operate).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '执行 custom' }))
    await waitFor(() => expect(operate).toHaveBeenCalledWith('2', 'arbitrary/provider', 'custom', '', { value: 9 }))
    expect(await screen.findByText(/"accepted": true/)).toBeInTheDocument()
  })

  it('keeps observe and session widget actions off the ordinary operate path', () => {
    const operate = vi.fn()
    const descriptor: ResourceDescriptor = {
      id: { owner_node_id: '2', name: 'channels' },
      type: 'example.custom',
      type_version: 1,
      capabilities: [
        { name: 'open', permission: 'ignored.open', input_schema: 'open.v1', max_payload_bytes: 4096 },
        { name: 'subscribe', permission: 'ignored.subscribe', event_schema: 'event.v1', max_payload_bytes: 4096 },
      ],
      limits: { max_payload_bytes: 4096 },
    }
    renderResourceWorkspace(descriptor, { operate } as unknown as DesktopAPI)

    fireEvent.click(screen.getByRole('button', { name: '观察 subscribe' }))
    expect(screen.getByText(/事件订阅通道/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '打开会话 open' }))
    expect(screen.getByText(/session 通道/)).toBeInTheDocument()
    expect(operate).not.toHaveBeenCalled()
  })

  it('keeps action payload drafts transient across save/reopen-shaped mounts', () => {
    const descriptor: ResourceDescriptor = {
      id: { owner_node_id: '2', name: 'transient' },
      type: 'example.custom',
      type_version: 1,
      capabilities: [{ name: 'custom', permission: 'ignored.custom', max_payload_bytes: 4096 }],
      limits: { max_payload_bytes: 4096 },
    }
    const first = renderResourceWorkspace(descriptor, { operate: vi.fn() } as unknown as DesktopAPI)
    fireEvent.click(screen.getByRole('button', { name: '操作 custom' }))
    fireEvent.change(screen.getByRole('textbox', { name: /输入/ }), { target: { value: '{"draft":"must-not-persist"}' } })
    expect(first.onChange).not.toHaveBeenCalled()
    expect(JSON.stringify(first.candidate)).not.toContain('must-not-persist')
    expect(JSON.stringify(first.candidate)).not.toContain('draft')
    first.unmount()

    renderResourceWorkspace(descriptor, { operate: vi.fn() } as unknown as DesktopAPI)
    fireEvent.click(screen.getByRole('button', { name: '操作 custom' }))
    expect(screen.getByRole('textbox', { name: /输入/ })).toHaveValue('{}')
  })

  it('keeps selected Collection members and file content out of the View document', async () => {
    const member = { key: 'secret-member-key', kind: 'file', label: 'Visible file', content_type: 'text/plain', capabilities: ['get', 'read'] }
    const descriptor: ResourceDescriptor = {
      id: { owner_node_id: '2', name: 'collection' },
      type: 'mfh.collection',
      type_version: 1,
      capabilities: [
        { name: 'list', permission: 'ignored.list', input_schema: 'mfh.collection.list-request.v1', output_schema: 'mfh.collection.page.v1', max_payload_bytes: 4096 },
        { name: 'get', permission: 'ignored.get', input_schema: 'mfh.collection.member-request.v1', output_schema: 'mfh.collection.member.v1', max_payload_bytes: 4096 },
        { name: 'read', permission: 'ignored.read', input_schema: 'mfh.filesystem.read-request.v1', output_schema: 'mfh.filesystem.content.v1', max_payload_bytes: 180_000 },
      ],
      limits: { max_payload_bytes: 180_000 },
    }
    const operate = vi.fn().mockImplementation(async (_owner, _name, capability) => {
      if (capability === 'list') return { schema: 'mfh.collection.page.v1', payload: { version: 1, revision: 1, parent: '', members: [member], next_cursor: '' } }
      if (capability === 'get') return { schema: 'mfh.collection.member.v1', payload: member }
      return { schema: 'mfh.filesystem.content.v1', payload: { version: 1, key: member.key, content_type: 'text/plain', encoding: 'utf-8', data: 'secret-content', size: 14, modified_unix_ms: 1, revision: 'r1' } }
    })
    const rendered = renderResourceWorkspace(descriptor, { operate } as unknown as DesktopAPI)
    fireEvent.click(await screen.findByRole('button', { name: /Visible file/ }))
    fireEvent.click(await screen.findByRole('button', { name: '读取文件内容' }))
    expect(await screen.findByText('secret-content')).toBeInTheDocument()
    expect(rendered.onChange).not.toHaveBeenCalled()
    expect(JSON.stringify(rendered.candidate)).not.toContain('secret-member-key')
    expect(JSON.stringify(rendered.candidate)).not.toContain('secret-content')
    expect(rendered.candidate.widgets[0]?.settings).toBeUndefined()
  })

  it('bounds compact action buttons behind an accessible overflow without network work', async () => {
    class TestResizeObserver {
      constructor(private readonly callback: ResizeObserverCallback) {}
      observe(target: Element) {
        this.callback([{ target, contentRect: { width: 310, height: 210 } } as ResizeObserverEntry], this as unknown as ResizeObserver)
      }
      disconnect() {}
      unobserve() {}
    }
    vi.stubGlobal('ResizeObserver', TestResizeObserver)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => { callback(0); return 1 })
    const operate = vi.fn()
    const descriptor: ResourceDescriptor = {
      id: { owner_node_id: '2', name: 'many-actions' },
      type: 'example.custom',
      type_version: 1,
      capabilities: ['a', 'b', 'c', 'd', 'e'].map((name) => ({ name, permission: `ignored.${name}`, max_payload_bytes: 4096 })),
      limits: { max_payload_bytes: 4096 },
    }
    try {
      renderResourceWorkspace(descriptor, { operate } as unknown as DesktopAPI)
      await waitFor(() => expect(screen.getByRole('article')).toHaveClass('density-compact'))
      expect(screen.getByRole('button', { name: '操作 a' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: '操作 b' })).toBeInTheDocument()
      expect(screen.getByText('+3').closest('summary')).toHaveAttribute('aria-label', '更多 Resource actions')
      expect(operate).not.toHaveBeenCalled()
    } finally {
      vi.unstubAllGlobals()
    }
  })
})
