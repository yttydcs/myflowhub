import { createContext, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useDraggable, useDroppable } from '@dnd-kit/core'
import { GripVertical, Plus, Trash2 } from 'lucide-react'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor, ViewDefinition, ViewVariable, ViewWidget } from '../types'
import { resourceKey } from '../lib/utils'
import { widgetVariables, cardTitle, defaultVariableName, removeVariable, updateVariable, transferVariable, asSingleVariableCard, uniqueReferenceID, sameVariable } from '../workspace-variables'
import { VariablePanel } from '../variable/VariablePanel'
import { CatalogContext, DiscoveryStatus, MissingResource } from '../discovery/DiscoveryStatus'
import { WidgetSettings } from './WidgetSettings'
import './variable-cards.css'
import { Select } from './ui/select'
import { Button } from './ui/button'
import { defaultRenderer } from '../store'

export type VariableDropPreview = { widgetID: string; beforeID?: string; description: string }
export const ViewEditorContext = createContext({ editing: false, saving: false })

type Hosts = { header: HTMLDivElement; body: HTMLDivElement }
type Entry = { variable: ViewVariable; list: boolean; resource?: ResourceDescriptor; hosts: Hosts }
const VariableHostsContext = createContext<Map<string, Hosts>>(new Map())

// Flat keyed portals own editors; layout slots only move their stable DOM hosts.
// Reparenting/reordering/converting cards therefore cannot remount an editor or
// flush an invalid local input. No editor state is serialized into a View.
export function VariableCardsProvider({ api, view, resources, children }: { api: DesktopAPI; view: ViewDefinition; resources: Map<string, ResourceDescriptor>; children: ReactNode }) {
  const hosts = useRef(new Map<string, Hosts>())
  const entries = useMemo(() => view.widgets.flatMap(widget => widgetVariables(widget).flatMap(variable => {
    const resource = resources.get(resourceKey(variable.owner_node_id, variable.resource_name))
    if (widget.kind !== 'variable-list' && resource?.type !== 'mfh.variable' && !variable.renderer.startsWith('mfh.variable')) return []
    let containers = hosts.current.get(variable.id)
    if (!containers) {
      containers = { header: document.createElement('div'), body: document.createElement('div') }
      containers.header.className = 'variable-header-host'
      containers.body.className = 'variable-body-host'
      hosts.current.set(variable.id, containers)
    }
    return [{ variable, resource, hosts: containers, list: widget.kind === 'variable-list' }]
  })), [view.widgets, resources])
  useLayoutEffect(() => {
    const ids = new Set(entries.map(entry => entry.variable.id))
    for (const id of hosts.current.keys()) if (!ids.has(id)) hosts.current.delete(id)
  }, [entries])
  return <VariableHostsContext.Provider value={hosts.current}>
    {children}
    {entries.map(entry => <VariableContent key={entry.variable.id} api={api} {...entry} />)}
  </VariableHostsContext.Provider>
}

function VariableContent({ api, variable, resource: currentResource, hosts, list }: Entry & { api: DesktopAPI }) {
  const discovery = useContext(CatalogContext)
  const catalog = discovery?.states.get(variable.owner_node_id)
  const retained = useRef(currentResource)
  if (currentResource) retained.current = currentResource
  const resource = currentResource || retained.current
  const unavailable = !!discovery && (!currentResource || catalog?.status !== 'loaded')
  return <>{!resource && createPortal(<strong>{variable.title || defaultVariableName(variable)}</strong>, hosts.header)}{createPortal(<>
    {resource && unavailable && <DiscoveryStatus state={catalog?.status === 'loaded' ? { status: 'error', stale: true, error: '目录中已找不到该变量；保留未保存输入' } : catalog} label={`Node ${variable.owner_node_id} 资源目录`} onRetry={() => discovery?.retry(variable.owner_node_id)} />}
    {resource ? <fieldset className="discovery-renderer" disabled={unavailable}>
      <VariablePanel api={api} resource={resource} rendererID={variable.renderer} titleOverride={variable.title} inline={list} disabledReason={unavailable ? '资源目录不可用，请刷新后重试' : undefined} showFooter={false} autoApply headerTarget={hosts.header} showAdvancedEditor={false} />
    </fieldset> : <MissingResource owner={variable.owner_node_id} name={variable.resource_name} />}
  </>, hosts.body)}</>
}

function HostSlot({ id, part }: { id: string; part: keyof Hosts }) {
  const hosts = useContext(VariableHostsContext)
  const ref = useRef<HTMLDivElement>(null)
  const host = hosts.get(id)?.[part]
  useLayoutEffect(() => {
    const slot = ref.current
    if (!slot || !host) return
    slot.appendChild(host)
    return () => { if (host.parentNode === slot) slot.removeChild(host) }
  }, [host])
  return <div ref={ref} className={`variable-${part}-slot`} />
}

function VariableHandle({ widget, variable }: { widget: ViewWidget; variable: ViewVariable }) {
  const { editing } = useContext(ViewEditorContext)
  const label = variable.title || defaultVariableName(variable)
  const drag = useDraggable({ id: `workspace-variable:${variable.id}`, disabled: !editing, data: { kind: 'workspace-variable', widgetID: widget.id, variableID: variable.id, label } })
  return editing ? <button type="button" ref={node => { drag.setNodeRef(node); drag.setActivatorNodeRef(node) }} {...drag.attributes} {...drag.listeners} className={`widget-drag-handle variable-drag-handle${drag.isDragging ? ' is-dragging' : ''}`} aria-label={`拖动变量 ${label}`}><GripVertical size={13} aria-hidden="true" /></button> : null
}

function VariableRow({ api, view, widget, variable, resources, preview, onChange }: {
  api: DesktopAPI; view: ViewDefinition; widget: ViewWidget; variable: ViewVariable; resources: Map<string, ResourceDescriptor>; preview?: VariableDropPreview | null; onChange(view: ViewDefinition): void
}) {
  const { editing } = useContext(ViewEditorContext)
  const [error, setError] = useState('')
  const drop = useDroppable({ id: `workspace-variable-row:${variable.id}`, data: { kind: 'workspace-variable-row', widgetID: widget.id, variableID: variable.id } })
  const resource = resources.get(resourceKey(variable.owner_node_id, variable.resource_name))
  const label = variable.title || defaultVariableName(variable)
  return <div ref={drop.setNodeRef} className={`variable-list-row${preview?.widgetID === widget.id && preview.beforeID === variable.id ? ' insert-variable-before' : ''}`}>
    <VariableHandle widget={widget} variable={variable} />
    <HostSlot id={variable.id} part="header" />
    <HostSlot id={variable.id} part="body" />
    {editing && <div className="variable-row-controls">
      <WidgetSettings api={api} resource={resource} label={label} title={variable.title} onTitleChange={title => onChange(updateVariable(view, widget.id, variable.id, { title }))} rendererID={variable.renderer} onRendererChange={renderer => onChange(updateVariable(view, widget.id, variable.id, { renderer }))}>
        <label className="widget-settings-field">移至卡片<Select aria-label="移至卡片" value="" options={[{ value: '__split', label: '独立为卡片' }, ...view.widgets.filter(item => item.id !== widget.id && (item.kind === 'variable-list' || resources.get(resourceKey(item.owner_node_id, item.resource_name))?.type === 'mfh.variable' || item.renderer.startsWith('mfh.variable'))).map(item => ({ value: `card:${item.id}`, label: cardTitle(item) }))]} onValueChange={value => { try { onChange(transferVariable(view, { kind: 'variable', widgetID: widget.id, variableID: variable.id }, value === '__split' ? { kind: 'layout', intent: view.layout_mode === 'compact' ? { kind: 'compact-end' } : { kind: 'root-edge', side: 'right' } } : { kind: 'list', widgetID: value.slice(5) })); setError('') } catch (failure) { setError(String(failure)) } }} /></label>
        {error && <p role="alert" className="form-error">{error}</p>}
      </WidgetSettings>
      <button type="button" aria-label={`移除变量 ${label}`} onClick={() => onChange(removeVariable(view, widget.id, variable.id))}><Trash2 size={12} aria-hidden="true" /></button>
    </div>}
  </div>
}

export function VariableCard({ api, view, widget, resources, preview, onChange, onRemove }: {
  api: DesktopAPI; view: ViewDefinition; widget: ViewWidget; resources: Map<string, ResourceDescriptor>; preview?: VariableDropPreview | null; onChange(view: ViewDefinition): void; onRemove(): void
}) {
  const { editing } = useContext(ViewEditorContext)
  const paneRef = useRef<HTMLElement | null>(null)
  const [density, setDensity] = useState('normal')
  const [settingsError, setSettingsError] = useState('')
  useEffect(() => {
    if (!paneRef.current) return
    const observer = new ResizeObserver(([entry]) => {
      if (!entry) return
      const { width, height } = entry.contentRect
      setDensity(width < 360 || height < 260 ? 'compact' : width >= 680 && height >= 440 ? 'expanded' : 'normal')
    })
    observer.observe(paneRef.current)
    return () => observer.disconnect()
  }, [])
  const list = widget.kind === 'variable-list'
  const label = cardTitle(widget)
  const variables = widgetVariables(widget)
  const variable = variables[0]
  const resource = variable && resources.get(resourceKey(variable.owner_node_id, variable.resource_name))
  const available = list ? [...resources.values()].filter(resource => resource.type === 'mfh.variable' && !variables.some(item => sameVariable(item, { owner_node_id: resource.id.owner_node_id, resource_name: resource.id.name }))) : []
  const drag = useDraggable({ id: `workspace-drag:${widget.id}`, disabled: !editing, data: { kind: 'workspace-widget', widgetID: widget.id, label } })
  const panel = useDroppable({ id: `workspace-panel:${widget.id}`, data: { kind: 'workspace-panel', widgetID: widget.id } })
  const body = useDroppable({ id: `workspace-variable-body:${widget.id}`, data: { kind: 'workspace-variable-body', widgetID: widget.id } })
  const activePreview = preview?.widgetID === widget.id ? preview : undefined
  return <article ref={node => { paneRef.current = node; drag.setNodeRef(node); panel.setNodeRef(node) }} id={`workspace-pane-${widget.id.replace(/[^a-zA-Z0-9_-]+/g, '-')}`} className={`widget variable-card density-${density}${list ? ' variable-list-card' : ''}${drag.isDragging ? ' is-dragging' : ''}${activePreview ? ' is-variable-target' : ''}`} aria-label={`组件 ${label}`}>
    <header className="widget-header">
      {editing && <button ref={drag.setActivatorNodeRef} className="widget-drag-handle" {...drag.listeners} {...drag.attributes} aria-label={`拖动 ${label} 调整面板位置`}><GripVertical size={14} aria-hidden="true" /></button>}
      <div className="widget-title">{list ? <strong>{label}</strong> : variable && <HostSlot id={variable.id} part="header" />}</div>
      {editing && <div className="widget-controls">
        <WidgetSettings api={api} resource={list ? undefined : resource} label={label} title={widget.title} onTitleChange={title => onChange({ ...view, widgets: view.widgets.map(item => item.id === widget.id ? { ...item, title } : item) })} rendererID={widget.renderer} onRendererChange={renderer => onChange({ ...view, widgets: view.widgets.map(item => item.id === widget.id ? { ...item, renderer } : item) })}>
          {list && <label className="widget-settings-field">添加变量<Select aria-label="添加变量" value="" placeholder="选择变量…" options={available.map(resource => ({ value: resourceKey(resource.id.owner_node_id, resource.id.name), label: `${resource.presentation?.label || resource.id.name} · Node ${resource.id.owner_node_id}` }))} onValueChange={key => {
            const resource = available.find(item => resourceKey(item.id.owner_node_id, item.id.name) === key)
            if (!resource) return
            try {
              onChange(transferVariable(view, { kind: 'catalog', variable: { id: uniqueReferenceID(view, `${resource.id.owner_node_id}-${resource.id.name}`), owner_node_id: resource.id.owner_node_id, resource_name: resource.id.name, renderer: defaultRenderer(resource) } }, { kind: 'list', widgetID: widget.id }))
              setSettingsError('')
            } catch (failure) { setSettingsError(String(failure)) }
          }} /></label>}
          {list && variables.length === 1 && <Button variant="secondary" size="sm" onClick={() => onChange({ ...view, widgets: view.widgets.map(item => item.id === widget.id ? asSingleVariableCard(item) : item) })}>转为单变量卡片</Button>}
          {settingsError && <p role="alert" className="form-error">{settingsError}</p>}
        </WidgetSettings>
        <button aria-label="移除组件" onClick={onRemove}><Trash2 size={13} aria-hidden="true" /></button>
      </div>}
    </header>
    <div ref={body.setNodeRef} className="widget-body">
      {list ? <div className="variable-list">
        {variables.map(item => <VariableRow key={item.id} api={api} view={view} widget={widget} variable={item} resources={resources} preview={preview} onChange={onChange} />)}
        {variables.length === 0 && <p className="variable-list-empty">拖入变量，组成列表</p>}
        {(editing || activePreview) && <div className={`variable-list-add${activePreview && !activePreview.beforeID ? ' insert-variable-before' : ''}`}><Plus size={12} aria-hidden="true" />拖入变量添加</div>}
      </div> : variable && <><VariableHandle widget={widget} variable={variable} /><HostSlot id={variable.id} part="body" /></>}
      {activePreview && <p className="variable-drop-hint" role="status">{activePreview.description}</p>}
    </div>
  </article>
}
