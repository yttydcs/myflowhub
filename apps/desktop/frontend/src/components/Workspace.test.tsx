import { DndContext } from '@dnd-kit/core'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from '../api'
import type { ViewDefinition, ViewWidget } from '../types'
import { Workspace } from './Workspace'

const widget = (id: string, x: number, w: number): ViewWidget => ({
  id,
  owner_node_id: '2',
  resource_name: `system/${id}`,
  renderer: 'mfh.variable',
  x,
  y: 0,
  w,
  h: 24,
})

function renderWorkspace(view: ViewDefinition, onChange = vi.fn()) {
  render(
    <DndContext>
      <Workspace
        api={{} as DesktopAPI}
        resources={[]}
        view={view}
        dirty
        saving={false}
        onChange={onChange}
        onSave={vi.fn()}
      />
    </DndContext>,
  )
  return onChange
}

describe('workspace panels', () => {
  it('renders a single widget across the full persisted workspace grid', () => {
    renderWorkspace({ id: 'view', name: 'View', revision: 0, widgets: [widget('catalog', 0, 12)] })
    expect(screen.getByRole('article', { name: '组件 catalog' })).toHaveStyle({
      gridColumn: '1 / span 12',
      gridRow: '1 / span 24',
    })
  })

  it('resizes a two-panel View through an accessible separator', () => {
    const onChange = renderWorkspace({
      id: 'view',
      name: 'View',
      revision: 0,
      widgets: [widget('catalog', 0, 6), widget('health', 6, 6)],
    })
    const separator = screen.getByRole('separator', { name: '调整左右面板比例' })
    expect(separator).toHaveAttribute('aria-valuetext', '左侧 50%，右侧 50%')
    fireEvent.keyDown(separator, { key: 'ArrowRight' })
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      layout: { direction: 'horizontal', split_ratio: 0.52 },
    }))
  })

  it('switches to a top-bottom layout and exposes direction-aware panel controls', () => {
    const onChange = renderWorkspace({
      id: 'view',
      name: 'View',
      revision: 0,
      widgets: [widget('catalog', 0, 6), widget('health', 6, 6)],
    })
    fireEvent.click(screen.getByRole('button', { name: '上下' }))
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      layout: { direction: 'vertical', split_ratio: 0.5 },
    }))
  })

  it('renders and swaps vertical panels without changing their physical split ratio', () => {
    const onChange = renderWorkspace({
      id: 'view',
      name: 'View',
      revision: 0,
      layout: { direction: 'vertical', split_ratio: 0.625 },
      widgets: [widget('catalog', 0, 8), widget('health', 8, 4)],
    })
    expect(screen.getByRole('article', { name: '组件 catalog' })).toHaveStyle({ gridColumn: '1', gridRow: '1' })
    expect(screen.getByRole('separator', { name: '调整上下区域比例' })).toHaveAttribute('aria-valuetext', '上方 63%，下方 37%')
    fireEvent.click(screen.getByRole('button', { name: '交换' }))
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      widgets: [expect.objectContaining({ id: 'health' }), expect.objectContaining({ id: 'catalog' })],
    }))
  })

  it('previews pointer resizing continuously and persists only when the drag ends', () => {
    class TestPointerEvent extends MouseEvent {
      readonly pointerId: number

      constructor(type: string, init: PointerEventInit = {}) {
        super(type, init)
        this.pointerId = init.pointerId ?? 1
      }
    }
    Object.defineProperty(window, 'PointerEvent', { configurable: true, value: TestPointerEvent })

    const onChange = renderWorkspace({
      id: 'view',
      name: 'View',
      revision: 0,
      widgets: [widget('catalog', 0, 6), widget('health', 6, 6)],
    })
    const separator = screen.getByRole('separator', { name: '调整左右面板比例' })
    const grid = separator.parentElement as HTMLDivElement
    vi.spyOn(grid, 'getBoundingClientRect').mockReturnValue({
      x: 0, y: 0, left: 0, top: 0, right: 1000, bottom: 700, width: 1000, height: 700, toJSON: () => ({}),
    })
    let captured = false
    separator.setPointerCapture = vi.fn(() => { captured = true })
    separator.hasPointerCapture = vi.fn(() => captured)
    separator.releasePointerCapture = vi.fn(() => { captured = false })

    fireEvent.pointerDown(separator, { pointerId: 1, clientX: 300, clientY: 350 })
    fireEvent.pointerMove(separator, { pointerId: 1, clientX: 400, clientY: 350 })
    expect(onChange).not.toHaveBeenCalled()
    expect(grid.style.gridTemplateColumns).not.toContain('0.5fr')
    fireEvent.pointerUp(separator, { pointerId: 1, clientX: 400, clientY: 350 })
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      layout: expect.objectContaining({ direction: 'horizontal', split_ratio: expect.closeTo(384 / 968, 5) }),
    }))
  })
})
