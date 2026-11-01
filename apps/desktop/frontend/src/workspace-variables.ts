import type { ViewDefinition, ViewVariable, ViewWidget } from './types'
import { addWorkspaceWidget, dockWorkspaceView, validateWorkspaceView, type WorkspaceDockIntent } from './workspace-layout'

export type VariableSource = { kind: 'catalog'; variable: ViewVariable } | { kind: 'variable'; widgetID: string; variableID: string }
export type VariableTarget = { kind: 'list'; widgetID: string; beforeID?: string } | { kind: 'layout'; intent: WorkspaceDockIntent }
export const MAX_CARD_VARIABLES = 64
export const MAX_VIEW_VARIABLES = 512

export function widgetVariables(widget: ViewWidget): ViewVariable[] {
  return widget.kind === 'variable-list' ? widget.variables || [] : [{
    id: widget.variable_id || widget.id, owner_node_id: widget.owner_node_id,
    resource_name: widget.resource_name, renderer: widget.renderer, title: widget.title, settings: widget.settings,
  }]
}

export function defaultVariableName(variable: Pick<ViewVariable, 'resource_name'>): string {
  return variable.resource_name.split('/').at(-1) || variable.resource_name
}

export function cardTitle(widget: ViewWidget): string {
  return widget.title || (widget.kind === 'variable-list' ? '变量列表' : defaultVariableName(widget))
}

export function sameVariable(a: Pick<ViewVariable, 'owner_node_id' | 'resource_name'>, b: Pick<ViewVariable, 'owner_node_id' | 'resource_name'>): boolean {
  return a.owner_node_id === b.owner_node_id && a.resource_name === b.resource_name
}

export function uniqueReferenceID(view: ViewDefinition, base = 'variable'): string {
  const used = new Set(view.widgets.flatMap(widget => [widget.id, ...widgetVariables(widget).map(variable => variable.id)]))
  const stem = base.toLowerCase().replace(/[^a-z0-9._-]+/g, '-').replace(/^[^a-z0-9]+/, '').slice(0, 48) || 'variable'
  let id = stem
  for (let suffix = 2; used.has(id); suffix++) id = `${stem}-${suffix}`
  return id
}

export function asVariableList(widget: ViewWidget): ViewWidget {
  if (widget.kind === 'variable-list') return widget
  return { id: widget.id, kind: 'variable-list', title: cardTitle(widget), compact_size: widget.compact_size,
    owner_node_id: '', resource_name: '', renderer: '', variables: widgetVariables(widget) }
}

export function asSingleVariableCard(widget: ViewWidget): ViewWidget {
  const variables = widgetVariables(widget)
  if (variables.length !== 1) throw new Error('只有一个变量时才能转为单变量卡片')
  const variable = variables[0]!
  return { ...variable, id: widget.id, variable_id: variable.id, title: widget.title || variable.title, compact_size: widget.compact_size }
}

export function updateVariable(view: ViewDefinition, widgetID: string, variableID: string, update: Partial<Pick<ViewVariable, 'title' | 'renderer'>>): ViewDefinition {
  return { ...view, widgets: view.widgets.map(widget => {
    if (widget.id !== widgetID) return widget
    if (widget.kind !== 'variable-list') return { ...widget, ...update }
    return { ...widget, variables: widgetVariables(widget).map(variable => variable.id === variableID ? { ...variable, ...update } : variable) }
  }) }
}

export function removeVariable(view: ViewDefinition, widgetID: string, variableID: string): ViewDefinition {
  return { ...view, widgets: view.widgets.map(widget => widget.id !== widgetID ? widget : {
    ...asVariableList(widget), variables: widgetVariables(widget).filter(variable => variable.id !== variableID),
  }) }
}

// Atomic immutable operation: duplicate checks precede source removal, so failed
// cross-card drops never lose the source. Stable reference IDs follow the editor.
export function transferVariable(view: ViewDefinition, source: VariableSource, target: VariableTarget): ViewDefinition {
  validateWorkspaceView(view)
  const from = source.kind === 'variable' ? view.widgets.find(widget => widget.id === source.widgetID) : undefined
  const variable = source.kind === 'catalog' ? source.variable : from && widgetVariables(from).find(item => item.id === source.variableID)
  if (!variable) throw new Error('拖动的变量已不存在')
  if (target.kind === 'list') {
    const to = view.widgets.find(widget => widget.id === target.widgetID)
    if (!to) throw new Error('目标卡片已不存在')
    const items = widgetVariables(to)
    const sameCard = from?.id === to.id
    if (sameCard && to.kind !== 'variable-list') return view
    if (!sameCard && items.some(item => sameVariable(item, variable))) throw new Error('此卡片已包含该变量')
    if (!sameCard && items.length >= MAX_CARD_VARIABLES) throw new Error(`每张卡片最多 ${MAX_CARD_VARIABLES} 个变量`)
    if (target.beforeID === variable.id && sameCard) return view
    const remaining = sameCard ? items.filter(item => item.id !== variable.id) : [...items]
    const index = target.beforeID === undefined ? remaining.length : remaining.findIndex(item => item.id === target.beforeID)
    if (index < 0) throw new Error('变量插入位置已失效')
    remaining.splice(index, 0, variable)
    if (sameCard && remaining.every((item, at) => item === items[at])) return view
    const base = from && !sameCard ? removeVariable(view, from.id, variable.id) : view
    const next = { ...base, widgets: base.widgets.map(widget => widget.id === to.id ? { ...asVariableList(widget), variables: remaining } : widget) }
    validateWorkspaceView(next)
    return next
  }
  // Dropping a single variable outside its card moves that card, not an empty
  // copy plus a replacement. List rows split out into a new independent card.
  if (from && from.kind !== 'variable-list') return dockWorkspaceView(view, { kind: 'existing', widgetID: from.id }, target.intent)
  const base = from ? removeVariable(view, from.id, variable.id) : view
  const id = uniqueReferenceID(view, `${variable.id}-card`)
  const widget: ViewWidget = { ...variable, id, variable_id: variable.id }
  return dockWorkspaceView(base, { kind: 'new', widget }, target.intent)
}

export function addEmptyVariableCard(view: ViewDefinition): ViewDefinition {
  return addWorkspaceWidget(view, { id: uniqueReferenceID(view, 'variables'), kind: 'variable-list', variables: [], owner_node_id: '', resource_name: '', renderer: '' })
}
