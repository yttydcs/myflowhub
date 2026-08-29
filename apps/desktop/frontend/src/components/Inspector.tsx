import { Focus, Plus, X } from 'lucide-react'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor, WorkspaceSelection } from '../types'
import { NodeRenderer, ResourceRenderer } from './Renderer'
import { Button } from './ui/button'

export function Inspector({ api, selection, resources, onClose, onAdd, onFocusNode }: {
  api: DesktopAPI
  selection: Exclude<WorkspaceSelection, null>
  resources: ResourceDescriptor[]
  onClose(): void
  onAdd(resource: ResourceDescriptor): void
  onFocusNode(nodeID: string): void
}) {
  const nodeResources = selection.kind === 'node'
    ? resources.filter((resource) => resource.id.owner_node_id === selection.node.node_id)
    : []
  const title = selection.kind === 'node'
    ? selection.node.display_name || `Node ${selection.node.node_id}`
    : selection.resource.presentation?.label || selection.resource.id.name
  return (
    <aside className="inspector" aria-label={`${title} 预览`}>
      <header className="inspector-header">
        <div><h2>{title}</h2>{selection.kind === 'resource' && <code>{selection.resource.type}</code>}</div>
        <button className="icon-button" onClick={onClose} aria-label="关闭预览"><X aria-hidden="true" size={15} /></button>
      </header>
      <div className="inspector-body">
        {selection.kind === 'node'
          ? <NodeRenderer node={selection.node} resources={nodeResources} />
          : (
            <>
              <dl className="inspector-facts">
                <div><dt>所有者</dt><dd>{selection.resource.id.owner_node_id}</dd></div>
                <div><dt>资源名</dt><dd><code>{selection.resource.id.name}</code></dd></div>
                <div><dt>版本</dt><dd>{selection.resource.type_version}</dd></div>
              </dl>
              {selection.resource.presentation?.description && <p className="resource-description">{selection.resource.presentation.description}</p>}
              <div className="capability-list" aria-label="资源能力">
                {selection.resource.capabilities.map((capability) => <span key={capability.name}>{capability.name}</span>)}
              </div>
              <ResourceRenderer api={api} resource={selection.resource} />
            </>
          )}
      </div>
      <footer className="inspector-actions">
        {selection.kind === 'node'
          ? <Button variant="secondary" onClick={() => onFocusNode(selection.node.node_id)}><Focus aria-hidden="true" size={14} />聚焦此节点</Button>
          : <Button onClick={() => onAdd(selection.resource)}><Plus aria-hidden="true" size={14} />添加到工作区</Button>}
      </footer>
    </aside>
  )
}
