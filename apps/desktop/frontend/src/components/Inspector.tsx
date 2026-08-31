import { ArrowLeft, Focus, Plus, X } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { resolveFocusedResourceAction, type FocusedResourceAction } from '../lib/resource-actions'
import type { ResourceDescriptor, WorkspaceSelection } from '../types'
import { NodeRenderer, ResourceRenderer } from './Renderer'
import { Button } from './ui/button'

export function Inspector({ api, selection, resources, focusedAction, onClose, onClearAction, onAdd, onFocusNode }: {
  api: DesktopAPI
  selection: Exclude<WorkspaceSelection, null>
  resources: ResourceDescriptor[]
  focusedAction: FocusedResourceAction | null
  onClose(): void
  onClearAction(): void
  onAdd(resource: ResourceDescriptor): void
  onFocusNode(nodeID: string): void
}) {
  const nodeResources = selection.kind === 'node'
    ? resources.filter((resource) => resource.id.owner_node_id === selection.node.node_id)
    : []
  const title = selection.kind === 'node'
    ? selection.node.display_name || `Node ${selection.node.node_id}`
    : selection.resource.presentation?.label || selection.resource.id.name
  const resolvedAction = selection.kind === 'resource' && focusedAction
    ? resolveFocusedResourceAction(selection.resource, focusedAction)
    : undefined
  const focusedForResource = selection.kind === 'resource' && focusedAction
    && focusedAction.ownerNodeID === selection.resource.id.owner_node_id
    && focusedAction.resourceName === selection.resource.id.name
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
              {focusedForResource && (
                <section className="focused-resource-action" aria-label={`已聚焦 capability ${focusedAction.capability}`}>
                  <header>
                    <div><strong>{resolvedAction?.label || `旧 capability ${focusedAction.capability}`}</strong><small>{resolvedAction ? actionDescription(resolvedAction.mode, resolvedAction.mutating, resolvedAction.unsafe) : '当前 descriptor 已移除此 capability，不能执行。'}</small></div>
                    <Button variant="ghost" size="sm" onClick={onClearAction}><ArrowLeft aria-hidden="true" size={13} />返回资源预览</Button>
                  </header>
                </section>
              )}
              <ResourceRenderer
                api={api}
                resource={selection.resource}
                focusedCapability={focusedForResource ? focusedAction.capability : undefined}
                focusedDisabledReason={resolvedAction?.disabledReason}
              />
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

function actionDescription(mode: 'operation' | 'observe' | 'session', mutating: boolean, unsafe: boolean): string {
  if (mode === 'observe') return '事件观察能力；不会调用普通 operate。'
  if (mode === 'session') return '有界会话能力；不会调用普通 operate。'
  if (unsafe) return '潜在破坏性操作；仅在明确执行后发送。'
  return mutating ? '可能修改远端状态；仅在明确执行后发送。' : '普通只读操作；仅在明确执行后发送。'
}
