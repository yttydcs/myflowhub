import { useDroppable } from '@dnd-kit/core'
import { useMemo } from 'react'
import { ArrowLeft, ArrowRight, Grid2X2, Minus, Plus, Save, Trash2 } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { resourceKey } from '../lib/utils'
import { removeWidget, updateWidget } from '../store'
import type { ResourceDescriptor, ViewDefinition } from '../types'
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

export function Workspace({ id, api, resources, view, dirty, saving, onChange, onSave }: Props) {
  const drop = useDroppable({ id: 'workspace-drop' })
  const resourceIndex = useMemo(() => new Map(resources.map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource])), [resources])
  const changeWidgets = (widgets: ViewDefinition['widgets']) => onChange({ ...view, widgets })
  return <section id={id} ref={drop.setNodeRef} className={`workspace ${drop.isOver ? 'is-drop-target' : ''}`} aria-label="资源工作区" tabIndex={-1}>
    <header className="workspace-toolbar">
      <div className="view-name"><Grid2X2 aria-hidden="true" size={16} /><Input aria-label="视图名称" name="view-name" autoComplete="off" value={view.name} onChange={(event) => onChange({ ...view, name: event.target.value })} /></div>
      <div className="toolbar-meta"><Badge>{view.widgets.length} widgets</Badge>{dirty && <span className="dirty-dot" role="status">未保存</span>}<Button size="sm" onClick={onSave} disabled={saving || !dirty}><Save aria-hidden="true" size={14} />{saving ? '保存中…' : '保存视图'}</Button></div>
    </header>

    <div className={`widget-grid ${view.widgets.length === 0 ? 'is-empty' : ''}`}>
      {view.widgets.length === 0 && <div className="workspace-empty"><span className="drop-glyph">＋</span><h3>把资源放到这里</h3><p>从左侧拖入，或使用资源行末尾的添加按钮。布局会保存在当前 Profile。</p></div>}
      {view.widgets.map((widget) => {
        const resource = resourceIndex.get(resourceKey(widget.owner_node_id, widget.resource_name))
        return <article key={widget.id} className="widget" aria-label={`组件 ${resource?.presentation?.label || widget.resource_name}`} style={{ gridColumn: `${widget.x + 1} / span ${widget.w}`, gridRow: `${widget.y + 1} / span ${widget.h}` }}>
          <header className="widget-header"><div><strong>{resource?.presentation?.label || widget.resource_name.split('/').at(-1)}</strong><small>{widget.owner_node_id} / {widget.resource_name}</small></div>
            <div className="widget-controls">
              <button aria-label="向左移动" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { x: widget.x - 1 }))}><ArrowLeft aria-hidden="true" size={13} /></button>
              <button aria-label="向右移动" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { x: widget.x + 1 }))}><ArrowRight aria-hidden="true" size={13} /></button>
              <button aria-label="缩小" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { w: widget.w - 1 }))}><Minus aria-hidden="true" size={13} /></button>
              <button aria-label="放大" onClick={() => changeWidgets(updateWidget(view.widgets, widget.id, { w: widget.w + 1 }))}><Plus aria-hidden="true" size={13} /></button>
              <button aria-label="移除组件" onClick={() => changeWidgets(removeWidget(view.widgets, widget.id))}><Trash2 aria-hidden="true" size={13} /></button>
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
    <Button variant="secondary" className="new-view" onClick={onCreate}><Plus aria-hidden="true" size={14} />新建视图</Button>
    <div className="view-list">{views.length === 0 && <p className="empty-copy">尚未保存任何视图</p>}{views.map((view) => <div key={view.id} className={`view-row ${view.id === activeID ? 'is-selected' : ''}`}>
      <button onClick={() => onOpen(view)}><Grid2X2 aria-hidden="true" size={15} /><span><strong>{view.name}</strong><small>{view.widgets.length} widgets · r{view.revision}</small></span></button>
      <button aria-label={`删除视图 ${view.name}`} onClick={() => onDelete(view)}><Trash2 aria-hidden="true" size={14} /></button>
    </div>)}</div>
  </div>
}
