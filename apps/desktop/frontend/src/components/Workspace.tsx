import { useDraggable, useDroppable } from '@dnd-kit/core'
import { useMemo, useRef, type CSSProperties, type KeyboardEvent, type PointerEvent } from 'react'
import { ArrowLeft, ArrowRight, GripVertical, Grid2X2, Plus, Save, Trash2 } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { resourceKey } from '../lib/utils'
import { moveWidgetByOffset, removeWidget, resizeWorkspacePanels } from '../store'
import type { ResourceDescriptor, ViewDefinition, ViewWidget } from '../types'
import { ResourceRenderer } from './Renderer'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { Input } from './ui/input'

type Props = {
  id?: string
  api: DesktopAPI
  resources: ResourceDescriptor[]
  view: ViewDefinition
  dirty: boolean
  saving: boolean
  onChange(view: ViewDefinition): void
  onSave(): void
}

type WorkspaceWidgetProps = {
  api: DesktopAPI
  resource?: ResourceDescriptor
  widget: ViewWidget
  index: number
  count: number
  rows: number
  onMove(offset: -1 | 1): void
  onRemove(): void
}

function WorkspaceWidget({ api, resource, widget, index, count, rows, onMove, onRemove }: WorkspaceWidgetProps) {
  const draggable = useDraggable({
    id: `workspace-drag:${widget.id}`,
    data: { kind: 'workspace-widget', widgetID: widget.id },
  })
  const target = useDroppable({
    id: `workspace-widget:${widget.id}`,
    data: { kind: 'workspace-widget-target', widgetID: widget.id },
  })
  const setNodeRef = (node: HTMLElement | null) => {
    draggable.setNodeRef(node)
    target.setNodeRef(node)
  }
  const label = resource?.presentation?.label || widget.resource_name.split('/').at(-1) || widget.resource_name
  const style: CSSProperties = {
    gridColumn: `${widget.x + 1} / span ${widget.w}`,
    gridRow: `${widget.y + 1} / span ${Math.min(widget.h, rows - widget.y)}`,
    transform: draggable.transform
      ? `translate3d(${draggable.transform.x}px, ${draggable.transform.y}px, 0)`
      : undefined,
  }

  return (
    <article
      ref={setNodeRef}
      className={`widget ${draggable.isDragging ? 'is-dragging' : ''} ${target.isOver ? 'is-drop-target' : ''}`}
      aria-label={`组件 ${label}`}
      style={style}
    >
      <header className="widget-header">
        <button
          ref={draggable.setActivatorNodeRef}
          className="widget-drag-handle"
          {...draggable.listeners}
          {...draggable.attributes}
          aria-label={`拖动 ${label} 调整面板位置`}
        >
          <GripVertical aria-hidden="true" size={14} />
        </button>
        <div className="widget-title"><strong>{label}</strong><small>{widget.owner_node_id} / {widget.resource_name}</small></div>
        <div className="widget-controls">
          <button aria-label="移到左侧" disabled={index === 0} onClick={() => onMove(-1)}><ArrowLeft aria-hidden="true" size={13} /></button>
          <button aria-label="移到右侧" disabled={index === count - 1} onClick={() => onMove(1)}><ArrowRight aria-hidden="true" size={13} /></button>
          <button aria-label="移除组件" onClick={onRemove}><Trash2 aria-hidden="true" size={13} /></button>
        </div>
      </header>
      <div className="widget-body">{resource ? <ResourceRenderer api={api} resource={resource} /> : <div className="missing-resource"><strong>资源暂不可用</strong><p>保留布局，等待 {widget.owner_node_id}/{widget.resource_name} 恢复。</p></div>}</div>
    </article>
  )
}

export function Workspace({ id, api, resources, view, dirty, saving, onChange, onSave }: Props) {
  const drop = useDroppable({ id: 'workspace-drop', data: { kind: 'workspace' } })
  const gridRef = useRef<HTMLDivElement>(null)
  const resourceIndex = useMemo(() => new Map(resources.map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource])), [resources])
  const rows = Math.max(24, ...view.widgets.map((widget) => widget.y + widget.h))
  const changeWidgets = (widgets: ViewDefinition['widgets']) => onChange({ ...view, widgets })
  const splitWidth = view.widgets.length === 2 ? view.widgets[0]?.w : undefined

  function resizeAt(clientX: number) {
    const rect = gridRef.current?.getBoundingClientRect()
    if (!rect || rect.width <= 24) return
    const ratio = (clientX - rect.left - 12) / (rect.width - 24)
    changeWidgets(resizeWorkspacePanels(view.widgets, ratio * 12))
  }

  function handleSplitterPointerDown(event: PointerEvent<HTMLDivElement>) {
    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    resizeAt(event.clientX)
  }

  function handleSplitterPointerMove(event: PointerEvent<HTMLDivElement>) {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) resizeAt(event.clientX)
  }

  function handleSplitterKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (splitWidth === undefined) return
    let next = splitWidth
    if (event.key === 'ArrowLeft') next -= 1
    else if (event.key === 'ArrowRight') next += 1
    else if (event.key === 'Home') next = 2
    else if (event.key === 'End') next = 10
    else if (event.key === 'Enter') next = 6
    else return
    event.preventDefault()
    changeWidgets(resizeWorkspacePanels(view.widgets, next))
  }

  return <section id={id} ref={drop.setNodeRef} className={`workspace ${drop.isOver ? 'is-drop-target' : ''}`} aria-label="资源工作区" tabIndex={-1}>
    <header className="workspace-toolbar">
      <div className="view-name"><Grid2X2 aria-hidden="true" size={16} /><Input aria-label="视图名称" name="view-name" autoComplete="off" value={view.name} onChange={(event) => onChange({ ...view, name: event.target.value })} /></div>
      <div className="toolbar-meta"><Badge>{view.widgets.length} widgets</Badge>{dirty && <span className="dirty-dot" role="status">未保存</span>}<Button size="sm" onClick={onSave} disabled={saving || !dirty}><Save aria-hidden="true" size={14} />{saving ? '保存中…' : '保存视图'}</Button></div>
    </header>

    <div ref={gridRef} className={`widget-grid ${view.widgets.length === 0 ? 'is-empty' : ''}`} style={{ '--workspace-rows': rows } as CSSProperties}>
      {view.widgets.length === 0 && <div className="workspace-empty"><span className="drop-glyph">＋</span><h3>把资源放到这里</h3><p>从左侧拖入，或使用资源行末尾的添加按钮。布局会保存在当前 Profile。</p></div>}
      {view.widgets.map((widget, index) => (
        <WorkspaceWidget
          key={widget.id}
          api={api}
          resource={resourceIndex.get(resourceKey(widget.owner_node_id, widget.resource_name))}
          widget={widget}
          index={index}
          count={view.widgets.length}
          rows={rows}
          onMove={(offset) => changeWidgets(moveWidgetByOffset(view.widgets, widget.id, offset))}
          onRemove={() => changeWidgets(removeWidget(view.widgets, widget.id))}
        />
      ))}
      {splitWidth !== undefined && (
        <div
          className="workspace-splitter"
          role="separator"
          aria-label="调整两个资源面板的宽度"
          aria-orientation="vertical"
          aria-valuemin={2}
          aria-valuemax={10}
          aria-valuenow={splitWidth}
          aria-valuetext={`左侧 ${splitWidth} 列，右侧 ${12 - splitWidth} 列`}
          tabIndex={0}
          style={{ left: `calc(12px + ${(splitWidth / 12) * 100}% - ${(splitWidth / 12) * 24}px)` }}
          onPointerDown={handleSplitterPointerDown}
          onPointerMove={handleSplitterPointerMove}
          onKeyDown={handleSplitterKeyDown}
          onDoubleClick={() => changeWidgets(resizeWorkspacePanels(view.widgets, 6))}
        ><span aria-hidden="true" /></div>
      )}
    </div>
  </section>
}

export function ViewManager({ views, activeID, onOpen, onCreate, onDelete }: {
  views: ViewDefinition[]
  activeID: string
  onOpen(view: ViewDefinition): void
  onCreate(): void
  onDelete(view: ViewDefinition): void
}) {
  return <div className="view-manager">
    <Button variant="secondary" className="new-view" onClick={onCreate}><Plus aria-hidden="true" size={14} />新建视图</Button>
    <div className="view-list">{views.length === 0 && <p className="empty-copy">尚未保存任何视图</p>}{views.map((view) => <div key={view.id} className={`view-row ${view.id === activeID ? 'is-selected' : ''}`}>
      <button onClick={() => onOpen(view)}><Grid2X2 aria-hidden="true" size={15} /><span><strong>{view.name}</strong><small>{view.widgets.length} widgets · r{view.revision}</small></span></button>
      <button aria-label={`删除视图 ${view.name}`} onClick={() => onDelete(view)}><Trash2 aria-hidden="true" size={14} /></button>
    </div>)}</div>
  </div>
}
