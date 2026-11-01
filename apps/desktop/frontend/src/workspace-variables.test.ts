import { describe, expect, it } from 'vitest'
import type { ViewDefinition, ViewVariable, ViewWidget } from './types'
import { addWorkspaceWidget, validateWorkspaceView } from './workspace-layout'
import { addEmptyVariableCard, asVariableList, asSingleVariableCard, cardTitle, removeVariable, transferVariable, uniqueReferenceID, updateVariable, widgetVariables } from './workspace-variables'

const variable = (id: string, name = id): ViewVariable => ({ id, owner_node_id: '1', resource_name: `system/${name}`, renderer: 'mfh.variable' })
const empty = (): ViewDefinition => ({ id: 'view', name: 'View', revision: 1, widgets: [], layout_mode: 'compact' })
const initial = () => ['cpu', 'memory'].reduce((view, id) => addWorkspaceWidget(view, variable(id)), empty())
const listTarget = (widgetID: string, beforeID?: string) => ({ kind: 'list' as const, widgetID, beforeID })
const from = (widgetID: string, variableID: string) => ({ kind: 'variable' as const, widgetID, variableID })
const layoutTarget = { kind: 'layout' as const, intent: { kind: 'compact-end' as const } }

describe('variable card operations', () => {
  it('reads legacy singles and restores automatic titles by clearing the override', () => {
    const widget = initial().widgets[0]!
    expect(widgetVariables(widget)).toMatchObject([{ id: 'cpu', resource_name: 'system/cpu' }])
    const renamed = updateVariable(initial(), 'cpu', 'cpu', { title: 'CPU 使用率' })
    expect(cardTitle(renamed.widgets[0]!)).toBe('CPU 使用率')
    expect(cardTitle(updateVariable(renamed, 'cpu', 'cpu', { title: undefined }).widgets[0]!)).toBe('cpu')
    expect(renamed.widgets[0]!.resource_name).toBe('system/cpu')
  })
  it('converts a single on catalog drop, retaining title, geometry, first binding and layout', () => {
    const view = initial()
    view.widgets[0] = { ...view.widgets[0]!, title: '设备', compact_size: { columns: 2, height: 200 } }
    const result = transferVariable(view, { kind: 'catalog', variable: variable('battery') }, listTarget('cpu'))
    expect(result.widgets[0]).toMatchObject({ id: 'cpu', kind: 'variable-list', title: '设备', compact_size: { columns: 2, height: 200 }, owner_node_id: '', renderer: '' })
    expect(widgetVariables(result.widgets[0]!).map(item => item.id)).toEqual(['cpu', 'battery'])
    expect(result.layout_root).toBe(view.layout_root)
    expect(view.widgets[0]!.kind).toBeUndefined()
    validateWorkspaceView(result)
  })
  it('moves a single into another card, retaining the emptied source container', () => {
    const view = transferVariable(initial(), from('memory', 'memory'), listTarget('cpu'))
    expect(view.widgets.map(widget => [widget.id, widgetVariables(widget).length])).toEqual([['cpu', 2], ['memory', 0]])
    expect(view.widgets.every(widget => widget.kind === 'variable-list')).toBe(true)
    validateWorkspaceView(view)
  })
  it('reorders rows and treats self-drops and unchanged append as no-ops', () => {
    const view = transferVariable(initial(), from('memory', 'memory'), listTarget('cpu'))
    expect(transferVariable(view, from('cpu', 'cpu'), listTarget('cpu', 'cpu'))).toBe(view)
    expect(transferVariable(view, from('cpu', 'memory'), listTarget('cpu'))).toBe(view)
    const next = transferVariable(view, from('cpu', 'memory'), listTarget('cpu', 'cpu'))
    expect(widgetVariables(next.widgets[0]!).map(item => item.id)).toEqual(['memory', 'cpu'])
  })
  it('splits a list row out with a stable reference ID and no automatic downgrade', () => {
    const view = transferVariable(initial(), from('memory', 'memory'), listTarget('cpu'))
    const next = transferVariable(view, from('cpu', 'cpu'), layoutTarget)
    expect(next.widgets[0]!.kind).toBe('variable-list')
    expect(widgetVariables(next.widgets[0]!)).toHaveLength(1)
    expect(next.widgets[2]).toMatchObject({ variable_id: 'cpu', owner_node_id: '1', resource_name: 'system/cpu' })
    expect(widgetVariables(next.widgets[2]!)[0]!.id).toBe('cpu')
    validateWorkspaceView(next)
  })
  it('moves single cards at layout targets rather than leaving an empty source', () => {
    const next = transferVariable(initial(), from('cpu', 'cpu'), layoutTarget)
    expect(next.widgets.map(widget => widget.id)).toEqual(['memory', 'cpu'])
    expect(next.widgets.every(widget => !widget.kind)).toBe(true)
  })
  it('rejects duplicates before mutating a source and allows the same resource in different cards', () => {
    const view = addWorkspaceWidget(initial(), variable('cpu-copy', 'cpu'))
    const before = JSON.stringify(view)
    expect(() => transferVariable(view, from('cpu-copy', 'cpu-copy'), listTarget('cpu'))).toThrow('已包含')
    expect(JSON.stringify(view)).toBe(before)
    validateWorkspaceView(view)
  })
  it('creates and round-trips empty list containers, including omitted empty variables from Go', () => {
    const next = addEmptyVariableCard(empty())
    expect(cardTitle(next.widgets[0]!)).toBe('变量列表')
    validateWorkspaceView(next)
    const without = { ...next, widgets: next.widgets.map(widget => ({ ...widget, variables: undefined })) }
    validateWorkspaceView(without)
    expect(removeVariable(initial(), 'cpu', 'cpu').widgets[0]!.variables).toEqual([])
  })
  it('changes row aliases independently of the group title and actual resource', () => {
    const view = { ...initial(), widgets: initial().widgets.map(asVariableList) }
    const next = updateVariable(view, 'cpu', 'cpu', { title: '处理器' })
    expect(next.widgets[0]!.title).toBe('cpu')
    expect(next.widgets[0]!.variables![0]).toMatchObject({ title: '处理器', resource_name: 'system/cpu' })
  })
  it('explicitly converts a one-row list back to a single without losing reference identity or title', () => {
    const group = asVariableList({ ...variable('cpu'), id: 'group', variable_id: 'cpu', title: '设备', compact_size: { columns: 2 } })
    const single = asSingleVariableCard(group)
    expect(single).toMatchObject({ id: 'group', variable_id: 'cpu', title: '设备', resource_name: 'system/cpu', compact_size: { columns: 2 } })
    expect(single.kind).toBeUndefined()
    expect(() => asSingleVariableCard({ ...group, variables: [] })).toThrow()
  })
  it('bounds generated IDs even when splitting a legacy maximum-length reference', () => {
    const id = uniqueReferenceID(empty(), 'a'.repeat(64) + '-card')
    expect(id).toMatch(/^[a-z0-9][a-z0-9._-]{0,63}$/)
  })
  it('does not reuse a variable ID hidden in a group', () => {
    const view = transferVariable(initial(), from('memory', 'memory'), listTarget('cpu'))
    expect(uniqueReferenceID(view, 'memory')).not.toBe('memory')
  })
  it('rejects stale source/target positions and non-variable entries', () => {
    const view = initial()
    expect(() => transferVariable(view, from('missing', 'x'), listTarget('cpu'))).toThrow('不存在')
    expect(() => transferVariable(view, from('cpu', 'cpu'), listTarget('missing'))).toThrow('不存在')
    expect(() => transferVariable(view, from('cpu', 'cpu'), listTarget('memory', 'gone'))).toThrow('失效')
    expect(() => validateWorkspaceView({ ...view, widgets: [{ ...asVariableList(view.widgets[0]!), variables: [{ ...variable('cpu'), renderer: 'mfh.command' }] }, view.widgets[1]!] })).toThrow('仅支持变量')
  })
  it.each([
    (widget: ViewWidget) => ({ ...widget, title: '中'.repeat(54) }),
    (widget: ViewWidget) => ({ ...asVariableList(widget), variables: [variable('cpu'), variable('cpu')] }),
    (widget: ViewWidget) => ({ ...asVariableList(widget), variables: Array.from({ length: 65 }, (_, i) => variable(`v-${i}`)) }),
    (widget: ViewWidget) => ({ ...asVariableList(widget), owner_node_id: '1' }),
  ])('rejects malformed or oversized cards', change => {
    const view = initial()
    expect(() => validateWorkspaceView({ ...view, widgets: [change(view.widgets[0]!), view.widgets[1]!] })).toThrow()
  })
})
