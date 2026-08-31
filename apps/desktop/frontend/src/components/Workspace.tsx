import { useDraggable, useDroppable } from '@dnd-kit/core'
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
  type ReactNode,
} from 'react'
import { GripVertical, Grid2X2, Plus, Save, Trash2, X } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { deriveResourceActions } from '../lib/resource-actions'
import { resourceKey } from '../lib/utils'
import type { ResourceDescriptor, ViewDefinition, ViewLayoutNode, ViewWidget } from '../types'
import type { PaneDensity } from '../rendering/registry'
import {
  WORKSPACE_SEPARATOR_SIZE,
  firstWorkspaceLeafID,
  lastWorkspaceLeafID,
  removeWorkspaceWidget,
  resizeWorkspaceSplitPair,
  workspaceLayoutMinimumSize,
  type WorkspaceDockSide,
} from '../workspace-layout'
import { ResourceRenderer, ResourceRendererSelector } from './Renderer'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { Input } from './ui/input'

export type WorkspaceDockPreview = {
  view: ViewDefinition
  widgetID: string
  description: string
}

type Props = {
  id?: string
  api: DesktopAPI
  resources: ResourceDescriptor[]
  view: ViewDefinition
  dirty: boolean
  saving: boolean
  dockPreview?: WorkspaceDockPreview | null
  onChange(view: ViewDefinition): void
  onSave(): void
  onError(message: string): void
}

type LayoutRenderProps = {
  api: DesktopAPI
  node: ViewLayoutNode
  path: number[]
  view: ViewDefinition
  resourceIndex: Map<string, ResourceDescriptor>
  onRemove(widgetID: string): void
  onRendererChange(widgetID: string, rendererID: string): void
  onResize(path: number[], dividerIndex: number, ratio: number, minimum: number, maximum: number): void
}

type WorkspaceWidgetProps = {
  api: DesktopAPI
  resource?: ResourceDescriptor
  widget: ViewWidget
  onRemove(): void
  onRendererChange(rendererID: string): void
}

function paneDOMID(widgetID: string): string {
  return `workspace-pane-${widgetID.replace(/[^a-zA-Z0-9_-]+/g, '-')}`
}

function widgetLabel(widget: ViewWidget, resource?: ResourceDescriptor): string {
  return resource?.presentation?.label || widget.resource_name.split('/').at(-1) || widget.resource_name
}

function WorkspaceWidgetPane({ api, resource, widget, onRemove, onRendererChange }: WorkspaceWidgetProps) {
  const draggable = useDraggable({
    id: `workspace-drag:${widget.id}`,
    data: { kind: 'workspace-widget', widgetID: widget.id, label: widgetLabel(widget, resource) },
  })
  const droppable = useDroppable({
    id: `workspace-panel:${widget.id}`,
    data: { kind: 'workspace-panel', widgetID: widget.id },
  })
  const paneRef = useRef<HTMLElement | null>(null)
  const [density, setDensity] = useState<PaneDensity>('normal')
  const [focusedCapability, setFocusedCapability] = useState<string>()
  const setNodeRef = (node: HTMLElement | null) => {
    paneRef.current = node
    draggable.setNodeRef(node)
    droppable.setNodeRef(node)
  }
  useEffect(() => {
    const pane = paneRef.current
    if (!pane || typeof ResizeObserver === 'undefined') return
    let frame: number | undefined
    const observer = new ResizeObserver(([entry]) => {
      if (!entry) return
      if (frame !== undefined) window.cancelAnimationFrame(frame)
      frame = window.requestAnimationFrame(() => {
        frame = undefined
        const { width, height } = entry.contentRect
        const next: PaneDensity = width < 360 || height < 260 ? 'compact' : width >= 680 && height >= 440 ? 'expanded' : 'normal'
        setDensity((current) => current === next ? current : next)
      })
    })
    observer.observe(pane)
    return () => {
      observer.disconnect()
      if (frame !== undefined) window.cancelAnimationFrame(frame)
    }
  }, [])
  useEffect(() => setFocusedCapability(undefined), [resource?.id.name, resource?.id.owner_node_id, widget.id])
  const label = widgetLabel(widget, resource)
  const actions = resource ? deriveResourceActions(resource.capabilities).sort((left, right) => Number(left.mutating) - Number(right.mutating) || left.capability.localeCompare(right.capability)) : []
  const visibleActionCount = density === 'compact' ? 2 : density === 'expanded' ? 5 : 3
  const visibleActions = actions.slice(0, visibleActionCount)
  const overflowActions = actions.slice(visibleActionCount)
  return (
    <article
      id={paneDOMID(widget.id)}
      ref={setNodeRef}
      className={`widget density-${density} ${draggable.isDragging ? 'is-dragging' : ''} ${droppable.isOver ? 'is-drop-target' : ''}`}
      aria-label={`组件 ${label}`}
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
        <div className="widget-title">
          <strong>{label}</strong>
          <small>{widget.owner_node_id} / {widget.resource_name}</small>
        </div>
        {resource && <ResourceRendererSelector resource={resource} rendererID={widget.renderer} onRendererChange={onRendererChange} />}
        {resource && actions.length > 0 && (
          <div className="widget-actions" aria-label="Resource actions">
            {visibleActions.map((action) => <button type="button" className={focusedCapability === action.capability ? 'is-active' : ''} disabled={action.disabled} title={action.label} aria-label={action.label} key={action.capability} onClick={() => setFocusedCapability(action.capability)}>{action.capability}</button>)}
            {overflowActions.length > 0 && <details><summary aria-label="更多 Resource actions">+{overflowActions.length}</summary><div>{overflowActions.map((action) => <button type="button" disabled={action.disabled} title={action.label} key={action.capability} onClick={() => setFocusedCapability(action.capability)}>{action.capability}</button>)}</div></details>}
          </div>
        )}
        <div className="widget-controls">
          <button aria-label="移除组件" onClick={onRemove}><Trash2 aria-hidden="true" size={13} /></button>
        </div>
      </header>
      <div className="widget-body">
        {resource && focusedCapability && <div className="widget-action-focusbar"><strong>{focusedCapability}</strong><button type="button" aria-label="关闭操作面板" onClick={() => setFocusedCapability(undefined)}><X aria-hidden="true" size={13} /></button></div>}
        {resource
          ? <ResourceRenderer api={api} resource={resource} rendererID={widget.renderer} density={density} focusedCapability={focusedCapability} />
          : (
            <div className="missing-resource">
              <strong>资源暂不可用</strong>
              <p>保留布局，等待 {widget.owner_node_id}/{widget.resource_name} 恢复。</p>
            </div>
          )}
      </div>
    </article>
  )
}

function LayoutLeaf({ api, widget, resource, onRemove, onRendererChange }: {
  api: DesktopAPI
  widget: ViewWidget
  resource?: ResourceDescriptor
  onRemove(): void
  onRendererChange(rendererID: string): void
}) {
  return (
    <div className="workspace-layout-leaf">
      <WorkspaceWidgetPane api={api} widget={widget} resource={resource} onRemove={onRemove} onRendererChange={onRendererChange} />
    </div>
  )
}

function splitGridStyle(node: Extract<ViewLayoutNode, { kind: 'split' }>, weights: number[]): CSSProperties {
  const tracks: string[] = []
  node.children.forEach((child, index) => {
    const minimum = workspaceLayoutMinimumSize(child)
    const pixels = node.axis === 'horizontal' ? minimum.width : minimum.height
    tracks.push(`minmax(${pixels}px, ${weights[index]}fr)`)
    if (index < node.children.length - 1) tracks.push(`${WORKSPACE_SEPARATOR_SIZE}px`)
  })
  return node.axis === 'horizontal'
    ? { gridTemplateColumns: tracks.join(' '), gridTemplateRows: 'minmax(0, 1fr)' }
    : { gridTemplateColumns: 'minmax(0, 1fr)', gridTemplateRows: tracks.join(' ') }
}

function SplitSeparator({ node, path, index, containerRef, draftWeights, onDraft, onCommit, onCancel }: {
  node: Extract<ViewLayoutNode, { kind: 'split' }>
  path: number[]
  index: number
  containerRef: React.RefObject<HTMLDivElement>
  draftWeights: number[]
  onDraft(weights: number[]): void
  onCommit(ratio: number, minimum: number, maximum: number): void
  onCancel(): void
}) {
  const beforeWidgetID = lastWorkspaceLeafID(node.children[index]!)
  const afterWidgetID = firstWorkspaceLeafID(node.children[index + 1]!)
  const droppable = useDroppable({
    id: `workspace-divider:${path.join('.') || 'root'}:${index}`,
    data: {
      kind: 'workspace-divider',
      parentPath: path,
      insertionIndex: index + 1,
      beforeWidgetID,
      afterWidgetID,
    },
  })
  const pendingRef = useRef<{ ratio: number; minimum: number; maximum: number }>()
  const animationRef = useRef<number>()

  const pair = () => {
    const parent = containerRef.current
    const leading = parent?.querySelector<HTMLElement>(`:scope > [data-layout-child="${index}"]`)
    const trailing = parent?.querySelector<HTMLElement>(`:scope > [data-layout-child="${index + 1}"]`)
    if (!leading || !trailing) return undefined
    const leadingRect = leading.getBoundingClientRect()
    const trailingRect = trailing.getBoundingClientRect()
    const horizontal = node.axis === 'horizontal'
    const leadingSize = horizontal ? leadingRect.width : leadingRect.height
    const trailingSize = horizontal ? trailingRect.width : trailingRect.height
    const pairSize = leadingSize + trailingSize
    if (pairSize <= 0) return undefined
    const leadingMinimum = workspaceLayoutMinimumSize(node.children[index]!)
    const trailingMinimum = workspaceLayoutMinimumSize(node.children[index + 1]!)
    const minimum = Math.min(0.49, (horizontal ? leadingMinimum.width : leadingMinimum.height) / pairSize)
    const maximum = Math.max(0.51, 1 - (horizontal ? trailingMinimum.width : trailingMinimum.height) / pairSize)
    return {
      horizontal,
      start: horizontal ? leadingRect.left : leadingRect.top,
      pairSize,
      minimum: Math.min(minimum, maximum - 0.01),
      maximum: Math.max(maximum, minimum + 0.01),
    }
  }

  const ratioAt = (clientX: number, clientY: number) => {
    const geometry = pair()
    if (!geometry) return undefined
    const coordinate = geometry.horizontal ? clientX : clientY
    const raw = (coordinate - geometry.start - WORKSPACE_SEPARATOR_SIZE / 2) / geometry.pairSize
    return {
      ratio: Math.min(geometry.maximum, Math.max(geometry.minimum, raw)),
      minimum: geometry.minimum,
      maximum: geometry.maximum,
    }
  }

  const previewWeights = (ratio: number) => {
    const pairTotal = node.weights[index]! + node.weights[index + 1]!
    const weights = [...node.weights]
    weights[index] = pairTotal * ratio
    weights[index + 1] = pairTotal * (1 - ratio)
    onDraft(weights)
  }

  const schedulePreview = (next: { ratio: number; minimum: number; maximum: number }) => {
    pendingRef.current = next
    if (animationRef.current !== undefined) return
    animationRef.current = window.requestAnimationFrame(() => {
      animationRef.current = undefined
      if (pendingRef.current) previewWeights(pendingRef.current.ratio)
    })
  }

  const cancelAnimation = () => {
    if (animationRef.current !== undefined) window.cancelAnimationFrame(animationRef.current)
    animationRef.current = undefined
    pendingRef.current = undefined
  }

  useEffect(() => () => cancelAnimation(), [])

  const pointerDown = (event: PointerEvent<HTMLDivElement>) => {
    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    const next = ratioAt(event.clientX, event.clientY)
    if (next) schedulePreview(next)
  }
  const pointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (!event.currentTarget.hasPointerCapture(event.pointerId)) return
    const next = ratioAt(event.clientX, event.clientY)
    if (next) schedulePreview(next)
  }
  const pointerUp = (event: PointerEvent<HTMLDivElement>) => {
    if (!event.currentTarget.hasPointerCapture(event.pointerId)) return
    const next = ratioAt(event.clientX, event.clientY) || pendingRef.current
    event.currentTarget.releasePointerCapture(event.pointerId)
    cancelAnimation()
    if (next) onCommit(next.ratio, next.minimum, next.maximum)
    else onCancel()
  }
  const pointerCancel = (event: PointerEvent<HTMLDivElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
    cancelAnimation()
    onCancel()
  }
  const keyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const geometry = pair()
    const minimum = geometry?.minimum ?? 0.05
    const maximum = geometry?.maximum ?? 0.95
    const total = draftWeights[index]! + draftWeights[index + 1]!
    const current = draftWeights[index]! / total
    const step = event.shiftKey ? 0.1 : 0.02
    let next = current
    if ((node.axis === 'horizontal' && event.key === 'ArrowLeft') || (node.axis === 'vertical' && event.key === 'ArrowUp')) next -= step
    else if ((node.axis === 'horizontal' && event.key === 'ArrowRight') || (node.axis === 'vertical' && event.key === 'ArrowDown')) next += step
    else if (event.key === 'Home') next = minimum
    else if (event.key === 'End') next = maximum
    else if (event.key === 'Enter') next = 0.5
    else return
    event.preventDefault()
    onCommit(Math.min(maximum, Math.max(minimum, next)), minimum, maximum)
  }

  const pairTotal = draftWeights[index]! + draftWeights[index + 1]!
  const pairRatio = draftWeights[index]! / pairTotal
  const percent = Math.round(pairRatio * 100)
  const orientation = node.axis === 'horizontal' ? 'vertical' : 'horizontal'
  return (
    <div
      ref={droppable.setNodeRef}
      className={`workspace-splitter is-${orientation} ${droppable.isOver ? 'is-drop-target' : ''}`}
      role="separator"
      aria-label={node.axis === 'horizontal' ? `调整 ${beforeWidgetID} 与 ${afterWidgetID} 的左右比例` : `调整 ${beforeWidgetID} 与 ${afterWidgetID} 的上下比例`}
      aria-orientation={orientation}
      aria-controls={`${paneDOMID(beforeWidgetID)} ${paneDOMID(afterWidgetID)}`}
      aria-valuemin={5}
      aria-valuemax={95}
      aria-valuenow={percent}
      aria-valuetext={node.axis === 'horizontal' ? `前一面板 ${percent}%，后一面板 ${100 - percent}%` : `上方面板 ${percent}%，下方面板 ${100 - percent}%`}
      tabIndex={0}
      onPointerDown={pointerDown}
      onPointerMove={pointerMove}
      onPointerUp={pointerUp}
      onPointerCancel={pointerCancel}
      onKeyDown={keyDown}
      onDoubleClick={() => onCommit(0.5, 0.05, 0.95)}
    >
      <span aria-hidden="true" />
    </div>
  )
}

function LayoutSplit({
  api,
  node,
  path,
  view,
  resourceIndex,
  onRemove,
  onRendererChange,
  onResize,
}: LayoutRenderProps & { node: Extract<ViewLayoutNode, { kind: 'split' }> }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [draftWeights, setDraftWeights] = useState(node.weights)
  useEffect(() => setDraftWeights(node.weights), [node.weights])
  return (
    <div
      ref={containerRef}
      className={`workspace-layout-split is-${node.axis}`}
      style={splitGridStyle(node, draftWeights)}
      data-layout-path={path.join('.')}
    >
      {node.children.flatMap((child, index): ReactNode[] => {
        const elements: ReactNode[] = [
          <div className="workspace-layout-child" data-layout-child={index} key={`child-${index}`}>
            <LayoutNode
              api={api}
              node={child}
              path={[...path, index]}
              view={view}
              resourceIndex={resourceIndex}
              onRemove={onRemove}
              onRendererChange={onRendererChange}
              onResize={onResize}
            />
          </div>,
        ]
        if (index < node.children.length - 1) {
          elements.push(
            <SplitSeparator
              key={`separator-${index}`}
              node={node}
              path={path}
              index={index}
              containerRef={containerRef}
              draftWeights={draftWeights}
              onDraft={setDraftWeights}
              onCommit={(ratio, minimum, maximum) => {
                setDraftWeights(node.weights)
                onResize(path, index, ratio, minimum, maximum)
              }}
              onCancel={() => setDraftWeights(node.weights)}
            />,
          )
        }
        return elements
      })}
    </div>
  )
}

function LayoutNode(props: LayoutRenderProps) {
  if (props.node.kind === 'split') return <LayoutSplit {...props} node={props.node} />
  const widgetID = props.node.widget_id
  const widget = props.view.widgets.find((candidate) => candidate.id === widgetID)
  if (!widget) return <div className="missing-resource" role="alert">布局引用了不存在的组件 {widgetID}</div>
  const resource = props.resourceIndex.get(resourceKey(widget.owner_node_id, widget.resource_name))
  return <LayoutLeaf api={props.api} widget={widget} resource={resource} onRemove={() => props.onRemove(widget.id)} onRendererChange={(rendererID) => props.onRendererChange(widget.id, rendererID)} />
}

function LayoutPreviewNode({ node, highlightWidgetID }: { node: ViewLayoutNode; highlightWidgetID: string }) {
  if (node.kind === 'leaf') {
    return <div className={`workspace-preview-leaf ${node.widget_id === highlightWidgetID ? 'is-highlighted' : ''}`} />
  }
  return (
    <div className={`workspace-preview-split is-${node.axis}`} style={splitGridStyle(node, node.weights)}>
      {node.children.flatMap((child, index): ReactNode[] => {
        const elements: ReactNode[] = [
          <div className="workspace-preview-child" key={`child-${index}`}>
            <LayoutPreviewNode node={child} highlightWidgetID={highlightWidgetID} />
          </div>,
        ]
        if (index < node.children.length - 1) elements.push(<div className="workspace-preview-separator" key={`separator-${index}`} />)
        return elements
      })}
    </div>
  )
}

function RootEdgeTarget({ side }: { side: WorkspaceDockSide }) {
  const droppable = useDroppable({
    id: `workspace-root-edge:${side}`,
    data: { kind: 'workspace-root-edge', side },
  })
  return <div ref={droppable.setNodeRef} className={`workspace-root-edge is-${side} ${droppable.isOver ? 'is-over' : ''}`} aria-hidden="true" />
}

export function Workspace({ id, api, resources, view, dirty, saving, dockPreview, onChange, onSave, onError }: Props) {
  const background = useDroppable({ id: 'workspace-drop', data: { kind: 'workspace-background' } })
  const resourceIndex = useMemo(
    () => new Map(resources.map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource])),
    [resources],
  )
  const minimum = view.layout_root ? workspaceLayoutMinimumSize(view.layout_root) : { width: 0, height: 0 }

  const apply = (operation: () => ViewDefinition) => {
    try {
      const next = operation()
      if (next !== view) onChange(next)
    } catch (current) {
      onError(current instanceof Error ? current.message : String(current))
    }
  }
  const onResize = (path: number[], dividerIndex: number, ratio: number, min: number, max: number) => apply(() => ({
    ...view,
    layout_root: resizeWorkspaceSplitPair(view.layout_root!, path, dividerIndex, ratio, min, max),
  }))

  return (
    <section id={id} className="workspace" aria-label="资源工作区" tabIndex={-1}>
      <header className="workspace-toolbar">
        <div className="view-name">
          <Grid2X2 aria-hidden="true" size={16} />
          <Input aria-label="视图名称" name="view-name" autoComplete="off" value={view.name} onChange={(event) => onChange({ ...view, name: event.target.value })} />
        </div>
        <div className="toolbar-meta">
          <Badge>{view.widgets.length} widgets</Badge>
          {dirty && <span className="dirty-dot" role="status">未保存</span>}
          <Button size="sm" onClick={onSave} disabled={saving || !dirty}><Save aria-hidden="true" size={14} />{saving ? '保存中…' : '保存视图'}</Button>
        </div>
      </header>
      <p className="sr-only" role="status" aria-live="polite">{dockPreview?.description || ''}</p>

      <div ref={background.setNodeRef} className={`widget-layout-scroll ${background.isOver ? 'is-drop-target' : ''}`}>
        {!view.layout_root
          ? (
            <div className="workspace-empty">
              <span className="drop-glyph">＋</span>
              <h3>把资源放到这里</h3>
              <p>从左侧拖入，或使用资源行末尾的添加按钮。布局会保存在当前 Profile。</p>
            </div>
          )
          : (
            <div className="workspace-layout-stage" style={{ minWidth: minimum.width, minHeight: minimum.height }}>
              <LayoutNode
                api={api}
                node={view.layout_root}
                path={[]}
                view={view}
                resourceIndex={resourceIndex}
                onRemove={(widgetID) => apply(() => removeWorkspaceWidget(view, widgetID))}
                onRendererChange={(widgetID, rendererID) => apply(() => ({
                  ...view,
                  widgets: view.widgets.map((widget) => widget.id === widgetID ? { ...widget, renderer: rendererID, settings: undefined } : widget),
                }))}
                onResize={onResize}
              />
              <RootEdgeTarget side="left" />
              <RootEdgeTarget side="right" />
              <RootEdgeTarget side="top" />
              <RootEdgeTarget side="bottom" />
              {dockPreview?.view.layout_root && (
                <div className="workspace-layout-preview" aria-hidden="true">
                  <LayoutPreviewNode node={dockPreview.view.layout_root} highlightWidgetID={dockPreview.widgetID} />
                </div>
              )}
            </div>
          )}
      </div>
    </section>
  )
}

export function ViewManager({ views, activeID, onOpen, onCreate, onDelete }: {
  views: ViewDefinition[]
  activeID: string
  onOpen(view: ViewDefinition): void
  onCreate(): void
  onDelete(view: ViewDefinition): void
}) {
  return (
    <div className="view-manager">
      <Button variant="secondary" className="new-view" onClick={onCreate}><Plus aria-hidden="true" size={14} />新建视图</Button>
      <div className="view-list">
        {views.length === 0 && <p className="empty-copy">尚未保存任何视图</p>}
        {views.map((view) => (
          <div key={view.id} className={`view-row ${view.id === activeID ? 'is-selected' : ''}`}>
            <button onClick={() => onOpen(view)}>
              <Grid2X2 aria-hidden="true" size={15} />
              <span><strong>{view.name}</strong><small>{view.widgets.length} widgets · r{view.revision}</small></span>
            </button>
            <button aria-label={`删除视图 ${view.name}`} onClick={() => onDelete(view)}><Trash2 aria-hidden="true" size={14} /></button>
          </div>
        ))}
      </div>
    </div>
  )
}
