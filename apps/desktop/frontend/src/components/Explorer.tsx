import { useMemo, useState } from 'react'
import { useDraggable } from '@dnd-kit/core'
import { ChevronDown, ChevronRight, CircleDot, Command, FileUp, Gauge, GripVertical, Layers3, Plus, Radio, Search, Variable } from 'lucide-react'
import { Input } from './ui/input'
import { ScrollArea } from './ui/scroll-area'
import { Badge } from './ui/badge'
import { buildExplorerTree, type ExplorerNode } from '../store'
import type { ResourceDescriptor, Topology, WorkspaceSelection } from '../types'

type Props = {
  topology: Topology
  resources: ResourceDescriptor[]
  selection: WorkspaceSelection
  onSelect(selection: WorkspaceSelection): void
  onAdd(resource: ResourceDescriptor): void
}

const icons: Record<string, typeof Variable> = {
  'mfh.variable': Variable,
  'mfh.stream': Gauge,
  'mfh.topic': Radio,
  'mfh.command': Command,
  'mfh.file': FileUp,
}

function ResourceRow({ resource, selected, onSelect, onAdd }: {
  resource: ResourceDescriptor
  selected: boolean
  onSelect(): void
  onAdd(): void
}) {
  const draggable = useDraggable({ id: `resource:${resource.id.owner_node_id}:${resource.id.name}`, data: { resource } })
  const Icon = icons[resource.type] ?? Layers3
  return (
    <div
      ref={draggable.setNodeRef}
      className={`resource-row ${selected ? 'is-selected' : ''} ${draggable.isDragging ? 'is-dragging' : ''}`}
      style={draggable.transform ? { transform: `translate3d(${draggable.transform.x}px, ${draggable.transform.y}px, 0)` } : undefined}
    >
      <button ref={draggable.setActivatorNodeRef} className="drag-handle" {...draggable.listeners} {...draggable.attributes} aria-label={`拖动 ${resource.id.name} 到工作区`}><GripVertical size={13} /></button>
      <button className="tree-main" onClick={onSelect}>
        <Icon size={15} />
        <span><strong>{resource.presentation?.label || resource.id.name.split('/').at(-1)}</strong><small>{resource.id.name}</small></span>
      </button>
      <button className="row-action" onClick={onAdd} aria-label={`添加 ${resource.id.name} 到工作区`}><Plus size={14} /></button>
    </div>
  )
}

function NodeBranch({ node, query, selection, onSelect, onAdd }: {
  node: ExplorerNode
  query: string
  selection: WorkspaceSelection
  onSelect(selection: WorkspaceSelection): void
  onAdd(resource: ResourceDescriptor): void
}) {
  const [open, setOpen] = useState(true)
  const resources = node.resources.filter((resource) => resource.id.name.toLowerCase().includes(query))
  const childMatches = node.children.filter((child) => !query || child.node_id.includes(query) || child.resources.some((resource) => resource.id.name.toLowerCase().includes(query)))
  if (query && resources.length === 0 && childMatches.length === 0 && !node.node_id.includes(query)) return null
  const nodeSelected = selection?.kind === 'node' && selection.node.node_id === node.node_id
  return (
    <div className="node-branch">
      <div className={`node-row ${nodeSelected ? 'is-selected' : ''}`}>
        <button className="disclosure" onClick={() => setOpen((value) => !value)} aria-label={`${open ? '折叠' : '展开'}节点 ${node.node_id}`}>
          {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </button>
        <button className="tree-main" onClick={() => onSelect({ kind: 'node', node })}>
          <CircleDot size={15} /><span><strong>{node.display_name || `Node ${node.node_id}`}</strong><small>{node.role}</small></span>
        </button>
        <Badge>{node.resources.length}</Badge>
      </div>
      {open && <div className="branch-children">
        {resources.map((resource) => (
          <ResourceRow
            key={resource.id.name}
            resource={resource}
            selected={selection?.kind === 'resource' && selection.resource.id.owner_node_id === resource.id.owner_node_id && selection.resource.id.name === resource.id.name}
            onSelect={() => onSelect({ kind: 'resource', resource })}
            onAdd={() => onAdd(resource)}
          />
        ))}
        {childMatches.map((child) => <NodeBranch key={child.node_id} node={child} query={query} selection={selection} onSelect={onSelect} onAdd={onAdd} />)}
      </div>}
    </div>
  )
}

export function Explorer(props: Props) {
  const [query, setQuery] = useState('')
  const tree = useMemo(() => buildExplorerTree(props.topology, props.resources), [props.topology, props.resources])
  return (
    <div className="explorer">
      <div className="search-box"><Search size={15} /><Input aria-label="搜索节点和资源" value={query} onChange={(event) => setQuery(event.target.value.toLowerCase())} placeholder="搜索资源…" /></div>
      <ScrollArea className="explorer-scroll">
        <div className="tree" role="tree" aria-label="节点与资源">
          {tree.map((node) => <NodeBranch key={node.node_id} node={node} query={query} selection={props.selection} onSelect={props.onSelect} onAdd={props.onAdd} />)}
        </div>
      </ScrollArea>
    </div>
  )
}
