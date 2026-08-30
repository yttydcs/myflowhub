import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ExplorerSplitPane } from './ExplorerSplitPane'

describe('ExplorerSplitPane', () => {
  it('previews pointer movement and commits once on release', () => {
    class TestPointerEvent extends MouseEvent {
      readonly pointerId: number

      constructor(type: string, init: PointerEventInit = {}) {
        super(type, init)
        this.pointerId = init.pointerId ?? 1
      }
    }

    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      value: TestPointerEvent,
    })

    const onRatioChange = vi.fn()
    render(
      <ExplorerSplitPane
        ratio={0.35}
        onRatioChange={onRatioChange}
        topID="top-pane"
        bottomID="bottom-pane"
        top={<span>Nodes</span>}
        bottom={<span>Resources</span>}
      />,
    )
    const separator = screen.getByRole('separator')
    const root = separator.parentElement!
    vi.spyOn(root, 'getBoundingClientRect').mockReturnValue({
      x: 0, y: 0, top: 0, left: 0, right: 272, bottom: 507, width: 272, height: 507, toJSON: () => ({}),
    })
    Object.defineProperties(separator, {
      setPointerCapture: { configurable: true, value: vi.fn() },
      hasPointerCapture: { configurable: true, value: vi.fn().mockReturnValue(true) },
      releasePointerCapture: { configurable: true, value: vi.fn() },
    })

    fireEvent.pointerDown(separator, { pointerId: 7, button: 0, clientY: 200 })
    fireEvent.pointerMove(separator, { pointerId: 7, clientY: 300 })
    expect(separator).toHaveAttribute('aria-valuenow', '60')
    expect(onRatioChange).not.toHaveBeenCalled()
    fireEvent.pointerUp(separator, { pointerId: 7, clientY: 300 })
    expect(onRatioChange).toHaveBeenCalledTimes(1)
    expect(onRatioChange).toHaveBeenCalledWith(0.6)

    fireEvent.pointerDown(separator, { pointerId: 8, button: 0, clientY: 260 })
    fireEvent.pointerMove(separator, { pointerId: 8, clientY: 340 })
    fireEvent.pointerCancel(separator, { pointerId: 8 })
    expect(separator).toHaveAttribute('aria-valuenow', '35')
    expect(onRatioChange).toHaveBeenCalledTimes(1)
    expect(separator.releasePointerCapture).toHaveBeenCalledTimes(2)
  })

  it('removes the separator while one pane is collapsed without changing the stored ratio', () => {
    const { container, rerender } = render(
      <ExplorerSplitPane
        ratio={0.62}
        onRatioChange={vi.fn()}
        topID="top-pane"
        bottomID="bottom-pane"
        top={<span>Nodes</span>}
        bottom={<span>Resources</span>}
        collapsedPane="node"
      />,
    )
    expect(screen.queryByRole('separator')).not.toBeInTheDocument()
    expect(container.firstElementChild).toHaveStyle({ gridTemplateRows: 'auto 0 minmax(0, 1fr)' })

    rerender(
      <ExplorerSplitPane
        ratio={0.62}
        onRatioChange={vi.fn()}
        topID="top-pane"
        bottomID="bottom-pane"
        top={<span>Nodes</span>}
        bottom={<span>Resources</span>}
      />,
    )
    expect(screen.getByRole('separator')).toHaveAttribute('aria-valuenow', '62')
  })
})
