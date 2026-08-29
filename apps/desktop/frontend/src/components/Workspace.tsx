import { useDraggable, useDroppable } from '@dnd-kit/core'
import { useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent, type PointerEvent } from 'react'
import { ArrowDown, ArrowLeft, ArrowLeftRight, ArrowRight, ArrowUp, Columns2, GripVertical, Grid2X2, Plus, Rows2, Save, Trash2 } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { resourceKey } from '../lib/utils'
import {
  MAX_WORKSPACE_SPLIT_RATIO,
  MIN_WORKSPACE_SPLIT_RATIO,
  clampWorkspaceSplitRatio,
  moveWidgetByOffset,
  removeWidget,
  resizeWorkspacePanels,
  resolveWorkspaceLayout,
  swapWorkspacePanels,
} from '../store'
import type { ResourceDescriptor, ViewDefinition, ViewLayoutDirection, ViewWidget } from '../types'
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
  direction: ViewLayoutDirection
  onMove(offset: -1 | 1): void
  onRemove(): void
}

function WorkspaceWidget({ api, resource, widget, index, count, rows, direction, onMove, onRemove }: WorkspaceWidgetProps) {
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
  const style: CSSProperties = count === 2 ? {
    gridColumn: direction === 'horizontal' ? `${index + 1}` : '1',
    gridRow: direction === 'vertical' ? `${index + 1}` : '1',
  } : {
    gridColumn: `${widget.x + 1} / span ${widget.w}`,
    gridRow: `${widget.y + 1} / span ${Math.min(widget.h, rows - widget.y)}`,
  }
  Object.assign(style, {
    transform: draggable.transform
      ? `translate3d(${draggable.transform.x}px, ${draggable.transform.y}px, 0)`
      : undefined,
  })
  const PreviousIcon = direction === 'vertical' ? ArrowUp : ArrowLeft
  const NextIcon = direction === 'vertical' ? ArrowDown : ArrowRight
  const previousLabel = direction === 'vertical' ? '移到上方' : '移到左侧'
  const nextLabel = direction === 'vertical' ? '移到下方' : '移到右侧'

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
          <button aria-label={previousLabel} disabled={index === 0} onClick={() => onMove(-1)}><PreviousIcon aria-hidden="true" size={13} /></button>
          <button aria-label={nextLabel} disabled={index === count - 1} onClick={() => onMove(1)}><NextIcon aria-hidden="true" size={13} /></button>
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
  const previewRatioRef = useRef<number>()
  const [previewRatio, setPreviewRatio] = useState<number>()
  const resourceIndex = useMemo(() => new Map(resources.map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource])), [resources])
  const rows = Math.max(24, ...view.widgets.map((widget) => widget.y + widget.h))
  const changeWidgets = (widgets: ViewDefinition['widgets']) => onChange({ ...view, widgets })
  const layout = resolveWorkspaceLayout(view)
  const isTwoPanel = view.widgets.length === 2
  const splitRatio = previewRatio ?? layout.split_ratio
  const gridStyle: CSSProperties = isTwoPanel
    ? layout.direction === 'horizontal'
      ? { gridTemplateColumns: `minmax(0, ${splitRatio}fr) minmax(0, ${1 - splitRatio}fr)`, gridTemplateRows: 'minmax(0, 1fr)' }
      : { gridTemplateColumns: 'minmax(0, 1fr)', gridTemplateRows: `minmax(0, ${splitRatio}fr) minmax(0, ${1 - splitRatio}fr)` }
    : { '--workspace-rows': rows } as CSSProperties

  useEffect(() => {
    previewRatioRef.current = undefined
    setPreviewRatio(undefined)
  }, [view.id, view.layout?.direction, view.layout?.split_ratio, view.widgets.length])

  function ratioAt(clientX: number, clientY: number): number | undefined {
    const rect = gridRef.current?.getBoundingClientRect()
    if (!rect) return undefined
    const horizontal = layout.direction === 'horizontal'
    const length = horizontal ? rect.width : rect.height
    const trackSpace = length - 32
    if (trackSpace <= 0) return undefined
    const position = (horizontal ? clientX - rect.left : clientY - rect.top) - 16
    return clampWorkspaceSplitRatio(position / trackSpace)
  }

  function previewResize(clientX: number, clientY: number) {
    const ratio = ratioAt(clientX, clientY)
    if (ratio === undefined) return
    previewRatioRef.current = ratio
    setPreviewRatio(ratio)
  }

  function commitResize(ratio: number) {
    const split = clampWorkspaceSplitRatio(ratio)
    onChange({
      ...view,
      layout: { direction: layout.direction, split_ratio: split },
      widgets: resizeWorkspacePanels(view.widgets, split * 12),
    })
  }

  function handleSplitterPointerDown(event: PointerEvent<HTMLDivElement>) {
    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    previewResize(event.clientX, event.clientY)
  }

  function handleSplitterPointerMove(event: PointerEvent<HTMLDivElement>) {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) previewResize(event.clientX, event.clientY)
  }

  function handleSplitterPointerUp(event: PointerEvent<HTMLDivElement>) {
    if (!event.currentTarget.hasPointerCapture(event.pointerId)) return
    const ratio = ratioAt(event.clientX, event.clientY) ?? previewRatioRef.current ?? layout.split_ratio
    event.currentTarget.releasePointerCapture(event.pointerId)
    previewRatioRef.current = undefined
    setPreviewRatio(undefined)
    commitResize(ratio)
  }

  function handleSplitterPointerCancel(event: PointerEvent<HTMLDivElement>) {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
    previewRatioRef.current = undefined
    setPreviewRatio(undefined)
  }

  function handleSplitterKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (!isTwoPanel) return
    const step = event.shiftKey ? 0.1 : 0.02
    let next = layout.split_ratio
    if ((layout.direction === 'horizontal' && event.key === 'ArrowLeft') || (layout.direction === 'vertical' && event.key === 'ArrowUp')) next -= step
    else if ((layout.direction === 'horizontal' && event.key === 'ArrowRight') || (layout.direction === 'vertical' && event.key === 'ArrowDown')) next += step
    else if (event.key === 'Home') next = MIN_WORKSPACE_SPLIT_RATIO
    else if (event.key === 'End') next = MAX_WORKSPACE_SPLIT_RATIO
    else if (event.key === 'Enter') next = 0.5
    else return
    event.preventDefault()
    commitResize(next)
  }

  const splitPercent = Math.round(splitRatio * 100)
  const splitterPosition = `calc(16px + ${splitRatio * 100}% - ${splitRatio * 32}px)`

  return <section id={id} ref={drop.setNodeRef} className={`workspace ${drop.isOver ? 'is-drop-target' : ''}`} aria-label="资源工作区" tabIndex={-1}>
    <header className="workspace-toolbar">
      <div className="view-name"><Grid2X2 aria-hidden="true" size={16} /><Input aria-label="视图名称" name="view-name" autoComplete="off" value={view.name} onChange={(event) => onChange({ ...view, name: event.target.value })} /></div>
      <div className="toolbar-meta">
        {isTwoPanel && <div className="workspace-layout-controls" role="group" aria-label="面板布局">
          <button type="button" className={layout.direction === 'horizontal' ? 'is-active' : ''} aria-pressed={layout.direction === 'horizontal'} onClick={() => onChange({ ...view, layout: { ...layout, direction: 'horizontal' } })}><Columns2 aria-hidden="true" size={13} />左右</button>
          <button type="button" className={layout.direction === 'vertical' ? 'is-active' : ''} aria-pressed={layout.direction === 'vertical'} onClick={() => onChange({ ...view, layout: { ...layout, direction: 'vertical' } })}><Rows2 aria-hidden="true" size={13} />上下</button>
          <button type="button" onClick={() => changeWidgets(swapWorkspacePanels(view.widgets))}><ArrowLeftRight aria-hidden="true" size={13} />交换</button>
        </div>}
        <Badge>{view.widgets.length} widgets</Badge>{dirty && <span className="dirty-dot" role="status">未保存</span>}<Button size="sm" onClick={onSave} disabled={saving || !dirty}><Save aria-hidden="true" size={14} />{saving ? '保存中…' : '保存视图'}</Button>
      </div>
    </header>

    <div ref={gridRef} className={`widget-grid ${view.widgets.length === 0 ? 'is-empty' : ''} ${isTwoPanel ? `is-two-panel is-${layout.direction}` : ''}`} style={gridStyle}>
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
          direction={layout.direction}
          onMove={(offset) => changeWidgets(moveWidgetByOffset(view.widgets, widget.id, offset))}
          onRemove={() => changeWidgets(removeWidget(view.widgets, widget.id))}
        />
      ))}
      {isTwoPanel && (
        <div
          className={`workspace-splitter is-${layout.direction === 'horizontal' ? 'vertical' : 'horizontal'}`}
          role="separator"
          aria-label={layout.direction === 'horizontal' ? '调整左右面板比例' : '调整上下区域比例'}
          aria-orientation={layout.direction === 'horizontal' ? 'vertical' : 'horizontal'}
          aria-valuemin={MIN_WORKSPACE_SPLIT_RATIO * 100}
          aria-valuemax={MAX_WORKSPACE_SPLIT_RATIO * 100}
          aria-valuenow={splitPercent}
          aria-valuetext={layout.direction === 'horizontal' ? `左侧 ${splitPercent}%，右侧 ${100 - splitPercent}%` : `上方 ${splitPercent}%，下方 ${100 - splitPercent}%`}
          tabIndex={0}
          style={layout.direction === 'horizontal' ? { left: splitterPosition } : { top: splitterPosition }}
          onPointerDown={handleSplitterPointerDown}
          onPointerMove={handleSplitterPointerMove}
          onPointerUp={handleSplitterPointerUp}
          onPointerCancel={handleSplitterPointerCancel}
          onKeyDown={handleSplitterKeyDown}
          onDoubleClick={() => commitResize(0.5)}
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
