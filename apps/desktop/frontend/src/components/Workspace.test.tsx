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
})
