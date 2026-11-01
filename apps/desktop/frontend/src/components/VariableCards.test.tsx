import { DndContext } from '@dnd-kit/core'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import type { ViewDefinition } from '../types'
import { addWorkspaceWidget } from '../workspace-layout'
import { transferVariable } from '../workspace-variables'
import { snapshot, variableAPI, variableResource } from '../variable/fixtures.test-support'
import { Workspace } from './Workspace'

function fixture() {
  const descriptor = variableResource(false)
  const { api } = variableAPI(snapshot(37, '1', { type: 'number', minimum: 0, maximum: 100 }))
  const second = { ...descriptor, id: { ...descriptor.id, name: 'metrics/memory' } }
  let view: ViewDefinition = { id: 'view', name: '笔记本', revision: 1, layout_mode: 'compact', widgets: [] }
  view = addWorkspaceWidget(view, { id: 'cpu', owner_node_id: descriptor.id.owner_node_id, resource_name: descriptor.id.name, renderer: 'mfh.variable' })
  view = addWorkspaceWidget(view, { id: 'memory', owner_node_id: second.id.owner_node_id, resource_name: second.id.name, renderer: 'mfh.variable' })
  return { api, descriptor, second, view }
}

describe('variable cards and editor modes', () => {
  it('keeps actual DOM editors and invalid input across merge, move, split, layout and mode changes without writes', async () => {
    const { api, descriptor, second, view } = fixture()
    function Harness() {
      const [current, setCurrent] = useState(view)
      const [editing, setEditing] = useState(true)
      return <DndContext>
        <button onClick={() => setCurrent(transferVariable(current, { kind: 'variable', widgetID: 'memory', variableID: 'memory' }, { kind: 'list', widgetID: 'cpu' }))}>合并</button>
        <button onClick={() => setCurrent(transferVariable(current, { kind: 'variable', widgetID: 'cpu', variableID: 'memory' }, { kind: 'layout', intent: { kind: 'compact-end' } }))}>拆分</button>
        <button onClick={() => setEditing(value => !value)}>切换编辑状态</button>
        <button onClick={() => setCurrent({ ...current, layout_mode: current.layout_mode === 'compact' ? 'split' : 'compact' })}>切换布局</button>
        <Workspace api={api} view={current} resources={[descriptor, second]} editing={editing} dirty saving={false} onChange={setCurrent} onSave={vi.fn()} onError={vi.fn()} />
      </DndContext>
    }
    render(<Harness />)
    await waitFor(() => expect(screen.getAllByRole('spinbutton')).toHaveLength(2))
    const input = screen.getAllByRole('spinbutton')[1]!
    await waitFor(() => expect(input).toHaveValue(37))
    fireEvent.change(input, { target: { value: '' } })
    await waitFor(() => expect(input).toHaveAttribute('aria-invalid', 'true'))
    fireEvent.click(screen.getByText('合并'))
    expect(screen.getAllByRole('spinbutton')).toContain(input)
    expect(input).toHaveValue(null)
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input.closest('.variable-list-row')).not.toBeNull()
    fireEvent.click(screen.getByText('拆分'))
    expect(screen.getAllByRole('spinbutton')).toContain(input)
    expect(input.closest('.variable-list-row')).toBeNull()
    fireEvent.click(screen.getByText('切换布局'))
    fireEvent.click(screen.getByText('切换编辑状态'))
    expect(screen.getAllByRole('spinbutton')).toContain(input)
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(api.operate).not.toHaveBeenCalled()
    expect(api.snapshot).toHaveBeenCalledTimes(2)
  })

  it('hides all structural controls in viewing mode while keeping variable inputs enabled', async () => {
    const { api, descriptor, second, view } = fixture()
    const onChange = vi.fn()
    const onEdit = vi.fn()
    render(<DndContext><Workspace api={api} view={view} resources={[descriptor, second]} dirty={false} saving={false} onChange={onChange} onSave={vi.fn()} onEdit={onEdit} onError={vi.fn()} /></DndContext>)
    expect((await screen.findAllByRole('spinbutton'))[0]).toBeEnabled()
    expect(screen.getByRole('textbox', { name: '视图名称' })).toHaveAttribute('readonly')
    fireEvent.change(screen.getByRole('textbox', { name: '视图名称' }), { target: { value: 'Blocked edit' } })
    expect(screen.queryByRole('button', { name: /拖动|移除组件|设置 |卡片尺寸|保存视图/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: '视图布局' })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '编辑视图' }))
    expect(onEdit).toHaveBeenCalledOnce()
    expect(onChange).not.toHaveBeenCalled()
  })

  it('renames cards with a local override and can clear it back to the variable name', async () => {
    const { api, descriptor, second, view } = fixture()
    function Harness() {
      const [current, setCurrent] = useState(view)
      return <DndContext><Workspace api={api} view={current} resources={[descriptor, second]} editing dirty saving={false} onChange={setCurrent} onSave={vi.fn()} onError={vi.fn()} /></DndContext>
    }
    render(<Harness />)
    await screen.findAllByRole('spinbutton')
    const first = screen.getAllByRole('article')[0]!
    fireEvent.click(within(first).getByRole('button', { name: /^设置 / }))
    const name = within(screen.getByRole('dialog')).getByRole('textbox', { name: '显示名称' })
    fireEvent.change(name, { target: { value: '处理器' } })
    expect(within(first).getByRole('heading', { name: '处理器', hidden: true })).toBeInTheDocument()
    fireEvent.change(name, { target: { value: '' } })
    await waitFor(() => expect(within(first).queryByRole('heading', { name: '处理器', hidden: true })).not.toBeInTheDocument())
    expect(api.operate).not.toHaveBeenCalled()
  })

  it('supports adding to an empty group and explicitly converting it back without dragging', async () => {
    const { api, descriptor } = fixture()
    function Harness() {
      const [current, setCurrent] = useState<ViewDefinition>({ id: 'view', name: 'Test', revision: 0, layout_mode: 'compact', widgets: [] })
      return <DndContext><Workspace api={api} view={current} resources={[descriptor]} editing dirty saving={false} onChange={setCurrent} onSave={vi.fn()} onError={vi.fn()} /></DndContext>
    }
    render(<Harness />)
    fireEvent.click(screen.getByRole('button', { name: '变量列表卡片' }))
    fireEvent.click(screen.getByRole('button', { name: '设置 变量列表' }))
    fireEvent.keyDown(screen.getByRole('combobox', { name: '添加变量' }), { key: 'ArrowDown' })
    fireEvent.click(await screen.findByRole('option', { name: `${descriptor.id.name} · Node ${descriptor.id.owner_node_id}` }))
    const input = await screen.findByRole('spinbutton', { hidden: true })
    fireEvent.click(screen.getByRole('button', { name: '转为单变量卡片' }))
    expect(document.querySelector('.variable-list-card')).toBeNull()
    expect(screen.getByRole('spinbutton', { hidden: true })).toBe(input)
    expect(api.operate).not.toHaveBeenCalled()
  })

  it('does not submit a numeric draft when a structural pointer action blurs the input', async () => {
    const { api, descriptor, second, view } = fixture()
    render(<DndContext><Workspace api={api} view={view} resources={[descriptor, second]} editing dirty saving={false} onChange={vi.fn()} onSave={vi.fn()} onError={vi.fn()} /></DndContext>)
    const input = (await screen.findAllByRole('spinbutton'))[0]!
    fireEvent.change(input, { target: { value: '45' } })
    fireEvent.pointerDown(screen.getAllByRole('button', { name: /拖动变量/ })[0]!)
    fireEvent.blur(input)
    await new Promise(resolve => setTimeout(resolve, 15))
    expect(api.operate).not.toHaveBeenCalled()
    fireEvent.pointerUp(document)
  })
})
