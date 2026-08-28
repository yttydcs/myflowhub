import { useDroppable } from '@dnd-kit/core'
import { ArrowLeft, ArrowRight, Grid2X2, Minus, Plus, Save, Trash2 } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { resourceKey } from '../lib/utils'
import { removeWidget, updateWidget } from '../store'
import type { ResourceDescriptor, ViewDefinition, WorkspaceSelection } from '../types'
import { NodeRenderer, ResourceRenderer } from './Renderer'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { Input } from './ui/input'

type Props = {
  api: DesktopAPI
  selection: WorkspaceSelection
  resources: ResourceDescriptor[]
  view: ViewDefinition
  dirty: boolean
  saving: boolean
  onChange(view: ViewDefinition): void
  onSave(): void
}

export function Workspace({ api, selection, resources, view, dirty, saving, onChange, onSave }: Props) {
  const drop = useDroppable({ id: 'workspace-drop' })
  const resourceIndex = new Map(resources.map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource]))
  const changeWidgets = (widgets: ViewDefinition['widgets']) => onChange({ ...view, widgets })
  return <section ref={drop.setNodeRef} className={`workspace ${drop.isOver ? 'is-drop-target' : ''}`} aria-label="资源工作区">
    <header className="workspace-toolbar">
      <div className="view-name"><Grid2X2 size={16} /><Input aria-label="视图名称" value={view.name} onChange={(event) => onChange({ ...view, name: event.target.value })} /></div>
      <div className="toolbar-meta"><Badge>{view.widgets.length} widgets</Badge>{dirty && <span className="dirty-dot">未保存</span>}<Button size="sm" onClick={onSave} disabled={saving || !dirty}><Save size={14} />{saving ? '保存中' : '保存视图'}</Button></div>
    </header>

    {selection && <div className="preview-card">
      <div className="preview-heading">
        <div><p className="eyebrow">LIVE PREVIEW</p><h2>{selection.kind === 'node' ? selection.node.display_name || `Node ${selection.node.node_id}` : selection.resource.presentation?.label || selection.resource.id.name}</h2></div>
        {selection.kind === 'resource' && <Badge>{selection.resource.type}</Badge>}
      </div>
      {selection.kind === 'node'
        ? <NodeRenderer node={selection.node} resources={resources.filter((resource) => resource.id.owner_node_id === selection.node.node_id)} />
        : <ResourceRenderer api={api} resource={selection.resource} />}
    </div>}

    <div className={`widget-grid ${view.widgets.length === 0 ? 'is-empty' : ''}`}>
      {view.widgets.length === 0 && <div className="workspace-empty"><span className="drop-glyph">＋</span><h3>把资源放到这里</h3><p>从左侧拖入，或使用资源行末尾的添加按钮。布局会保存在当前 Profile。</p></div>}
      {view.widgets.map((widget) => {
        const resource = resourceIndex.get(resourceKey(widget.owner_node_id, widget.resource_name))
        return <article key={widget.id} className="widget" style={{ gridColumn: `${widget.x + 1} / span ${widget.w}`, gridRow: `${widget.y + 1} / span ${widget.h}` }}>
          <header className="widget-header"><div><strong>{resource?.presentation?.label || widget.resource_name.split('/').at(-1)}</strong><small>{widget.owner_node_id} / {widget.resource_name}</small></div>
            <div className="widget-controls">
              <button aria-label="向左移动" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { x: widget.x - 1 }))}><ArrowLeft size={13} /></button>
              <button aria-label="向右移动" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { x: widget.x + 1 }))}><ArrowRight size={13} /></button>
              <button aria-label="缩小" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { w: widget.w - 1 }))}><Minus size={13} /></button>
              <button aria-label="放大" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { w: widget.w + 1 }))}><Plus size={13} /></button>
              <button aria-label="移除组件" onClick={() => changeWidgets(removeWidget(view.widgets, widget.id))}><Trash2 size={13} /></button>
            </div>
          </header>
          <div className="widget-body">{resource ? <ResourceRenderer api={api} resource={resource} /> : <div className="missing-resource"><strong>资源暂不可用</strong><p>保留布局，等待 {widget.owner_node_id}/{widget.resource_name} 恢复。</p></div>}</div>
        </article>
      })}
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
    <Button variant="secondary" className="new-view" onClick={onCreate}><Plus size={14} />新建视图</Button>
    <div className="view-list">{views.map((view) => <div key={view.id} className={`view-row ${view.id === activeID ? 'is-selected' : ''}`}>
      <button onClick={() => onOpen(view)}><Grid2X2 size={15} /><span><strong>{view.name}</strong><small>{view.widgets.length} widgets · r{view.revision}</small></span></button>
      <button aria-label={`删除视图 ${view.name}`} onClick={() => onDelete(view)}><Trash2 size={14} /></button>
    </div>)}</div>
  </div>
}
