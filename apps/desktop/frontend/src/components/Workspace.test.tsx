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
    const separator = screen.getByRole('separator', { name: '调整两个资源面板的宽度' })
    expect(separator).toHaveAttribute('aria-valuetext', '左侧 6 列，右侧 6 列')
    fireEvent.keyDown(separator, { key: 'ArrowRight' })
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      widgets: [
        expect.objectContaining({ id: 'catalog', x: 0, w: 7 }),
        expect.objectContaining({ id: 'health', x: 7, w: 5 }),
      ],
    }))
  })
})
