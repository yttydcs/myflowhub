import { useDeferredValue, useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent } from 'react'
import { useDraggable } from '@dnd-kit/core'
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  CircleDot,
  Command,
  FileUp,
  Gauge,
  GripVertical,
  Layers3,
  Plus,
  Radio,
  Search,
  Variable,
} from 'lucide-react'
import { Input } from './ui/input'
import { ScrollArea } from './ui/scroll-area'
import {
  buildExplorerIndex,
  defaultExpandedNodeIDs,
  explorerBreadcrumb,
  flattenExplorerRows,
  nodeRowKey,
  type ExplorerRow,
} from '../store'
import type { ResourceDescriptor, Topology, TopologyNode, WorkspaceSelection } from '../types'

type Props = {
  topology: Topology
  resources: ResourceDescriptor[]
  selection: WorkspaceSelection
  expandedNodeIDs?: string[]
  focusedNodeID?: string
  onExpandedNodeIDsChange(nodeIDs: string[]): void
  onFocusedNodeIDChange(nodeID?: string): void
  onSelect(selection: WorkspaceSelection): void
  onAdd(resource: ResourceDescriptor): void
}

const resourceIcons: Record<string, typeof Variable> = {
  'mfh.variable': Variable,
  'mfh.stream': Gauge,
  'mfh.topic': Radio,
  'mfh.command': Command,
  'mfh.file': FileUp,
}

type RowCommonProps = {
  row: ExplorerRow
  active: boolean
  selected: boolean
  setRef(key: string, element: HTMLButtonElement | null): void
  onFocus(key: string): void
  onKeyDown(event: KeyboardEvent<HTMLButtonElement>, row: ExplorerRow): void
}

function NodeRow(props: RowCommonProps & {
  expanded: boolean
  expandable: boolean
  resourceCount: number
  onSelect(node: TopologyNode): void
  onToggle(nodeID: string): void
}) {
  const node = props.row.node!
  return (
    <div className={`tree-row-shell node-row ${props.selected ? 'is-selected' : ''}`} role="none" style={rowIndent(props.row.depth)}>
      <button
        className="disclosure"
        onClick={() => props.onToggle(node.node_id)}
        disabled={!props.expandable}
        aria-label={`${props.expanded ? '折叠' : '展开'}节点 ${node.display_name || node.node_id}`}
        tabIndex={-1}
      >
        {props.expandable
          ? props.expanded ? <ChevronDown aria-hidden="true" size={13} /> : <ChevronRight aria-hidden="true" size={13} />
          : <span className="disclosure-placeholder" />}
      </button>
      <button
        ref={(element) => props.setRef(props.row.key, element)}
        className="tree-main"
        role="treeitem"
        aria-level={props.row.depth}
        aria-posinset={props.row.posInSet}
        aria-setsize={props.row.setSize}
        aria-expanded={props.expandable ? props.expanded : undefined}
        aria-selected={props.selected}
        tabIndex={props.active ? 0 : -1}
        onFocus={() => props.onFocus(props.row.key)}
        onClick={() => props.onSelect(node)}
        onDoubleClick={() => props.expandable && props.onToggle(node.node_id)}
        onKeyDown={(event) => props.onKeyDown(event, props.row)}
      >
        <CircleDot aria-hidden="true" size={14} />
        <span className="tree-label">
          <strong>{node.display_name || `Node ${node.node_id}`}</strong>
          <small>{node.role}</small>
        </span>
      </button>
      <span className="tree-count" aria-label={`${props.resourceCount} 个直接资源`}>{props.resourceCount}</span>
    </div>
  )
}

function ResourceRow(props: RowCommonProps & {
  resource: ResourceDescriptor
  onSelect(resource: ResourceDescriptor): void
  onAdd(resource: ResourceDescriptor): void
}) {
  const { resource } = props
  const draggable = useDraggable({
    id: `resource:${resource.id.owner_node_id}:${resource.id.name}`,
    data: { resource },
  })
  const Icon = resourceIcons[resource.type] ?? Layers3
  return (
    <div
      ref={draggable.setNodeRef}
      className={`tree-row-shell resource-row ${props.selected ? 'is-selected' : ''} ${draggable.isDragging ? 'is-dragging' : ''}`}
      role="none"
      style={{
        ...rowIndent(props.row.depth),
        ...(draggable.transform
          ? { transform: `translate3d(${draggable.transform.x}px, ${draggable.transform.y}px, 0)` }
          : undefined),
      }}
    >
      <button
        ref={draggable.setActivatorNodeRef}
        className="drag-handle"
        {...draggable.listeners}
        {...draggable.attributes}
        aria-label={`拖动 ${resource.presentation?.label || resource.id.name} 到工作区`}
        tabIndex={-1}
      >
        <GripVertical aria-hidden="true" size={12} />
      </button>
      <button
        ref={(element) => props.setRef(props.row.key, element)}
        className="tree-main"
        role="treeitem"
        aria-level={props.row.depth}
        aria-posinset={props.row.posInSet}
        aria-setsize={props.row.setSize}
        aria-selected={props.selected}
        tabIndex={props.active ? 0 : -1}
        onFocus={() => props.onFocus(props.row.key)}
        onClick={() => props.onSelect(resource)}
        onKeyDown={(event) => props.onKeyDown(event, props.row)}
      >
        <Icon aria-hidden="true" size={14} />
        <span className="tree-label">
          <strong>{resource.presentation?.label || resource.id.name.split('/').at(-1)}</strong>
          <small>{resource.id.name}</small>
        </span>
      </button>
      <button className="row-action" onClick={() => props.onAdd(resource)} aria-label={`添加 ${resource.id.name} 到工作区`}>
        <Plus aria-hidden="true" size={13} />
      </button>
    </div>
  )
}

export function Explorer(props: Props) {
  const [query, setQuery] = useState('')
  const deferredQuery = useDeferredValue(query)
  const [activeKey, setActiveKey] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)
  const rowRefs = useRef(new Map<string, HTMLButtonElement>())
  const index = useMemo(() => buildExplorerIndex(props.topology, props.resources), [props.topology, props.resources])
  const defaultExpanded = useMemo(() => defaultExpandedNodeIDs(index), [index])
  const expandedNodeIDs = props.expandedNodeIDs ?? defaultExpanded
  const expanded = useMemo(() => new Set(expandedNodeIDs), [expandedNodeIDs])
  const rows = useMemo(
    () => flattenExplorerRows(index, expanded, deferredQuery, props.focusedNodeID),
    [deferredQuery, expanded, index, props.focusedNodeID],
  )
  const breadcrumb = useMemo(() => explorerBreadcrumb(index, props.focusedNodeID), [index, props.focusedNodeID])

  useEffect(() => {
    if (rows.length === 0) {
      if (activeKey) setActiveKey('')
      return
    }
    const firstRow = rows[0]
    if (firstRow && !rows.some((row) => row.key === activeKey)) setActiveKey(firstRow.key)
  }, [activeKey, rows])

  function setRowRef(key: string, element: HTMLButtonElement | null) {
    if (element) rowRefs.current.set(key, element)
    else rowRefs.current.delete(key)
  }

  function focusRow(rowIndex: number) {
    const row = rows[rowIndex]
    if (!row) return
    setActiveKey(row.key)
    requestAnimationFrame(() => rowRefs.current.get(row.key)?.focus())
  }

  function setExpanded(nodeID: string, shouldExpand: boolean) {
    const next = new Set(expanded)
    if (shouldExpand) next.add(nodeID)
    else next.delete(nodeID)
    props.onExpandedNodeIDsChange([...next])
  }

  function toggleNode(nodeID: string) {
    setExpanded(nodeID, !expanded.has(nodeID))
  }

  function handleTreeKeyDown(event: KeyboardEvent<HTMLButtonElement>, row: ExplorerRow) {
    const rowIndex = rows.findIndex((candidate) => candidate.key === row.key)
    if (rowIndex < 0) return
    if (event.key === 'ArrowDown') focusRow(Math.min(rows.length - 1, rowIndex + 1))
    else if (event.key === 'ArrowUp') focusRow(Math.max(0, rowIndex - 1))
    else if (event.key === 'Home') focusRow(0)
    else if (event.key === 'End') focusRow(rows.length - 1)
    else if (event.key === 'ArrowRight' && row.kind === 'node') {
      const nodeID = row.node!.node_id
      const expandable = hasChildren(index, nodeID)
      if (expandable && !expanded.has(nodeID)) setExpanded(nodeID, true)
      else {
        const childIndex = rows.findIndex((candidate, indexInRows) => indexInRows > rowIndex && candidate.parentKey === row.key)
        if (childIndex >= 0) focusRow(childIndex)
      }
    } else if (event.key === 'ArrowLeft') {
      if (row.kind === 'node' && expanded.has(row.node!.node_id)) setExpanded(row.node!.node_id, false)
      else if (row.parentKey) {
        const parentIndex = rows.findIndex((candidate) => candidate.key === row.parentKey)
        if (parentIndex >= 0) focusRow(parentIndex)
      }
    } else if (event.key === 'Enter' || event.key === ' ') {
      if (row.kind === 'node') props.onSelect({ kind: 'node', node: row.node! })
      else props.onSelect({ kind: 'resource', resource: row.resource! })
    } else if (event.key === '*') {
      const parentKey = row.parentKey
      const next = new Set(expanded)
      for (const sibling of rows) {
        if (sibling.kind === 'node' && sibling.parentKey === parentKey && hasChildren(index, sibling.node!.node_id)) {
          next.add(sibling.node!.node_id)
        }
      }
      props.onExpandedNodeIDsChange([...next])
    } else return
    event.preventDefault()
  }

  function handleExplorerKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === '/' && event.target !== searchRef.current) {
      event.preventDefault()
      searchRef.current?.focus()
    }
  }

  return (
    <div className="explorer" onKeyDown={handleExplorerKeyDown}>
      <div className="search-box">
        <Search aria-hidden="true" size={14} />
        <Input
          ref={searchRef}
          aria-label="搜索节点和资源"
          name="resource-search"
          autoComplete="off"
          spellCheck={false}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="搜索节点或资源"
        />
        <kbd>/</kbd>
      </div>
      {breadcrumb.length > 0 && (
        <div className="tree-focus-bar">
          <button onClick={() => props.onFocusedNodeIDChange(undefined)} aria-label="返回完整资源树">
            <ChevronLeft aria-hidden="true" size={13} /> 返回
          </button>
          <div className="tree-breadcrumb" aria-label="当前节点路径">
            {breadcrumb.map((node, indexInPath) => (
              <span key={node.node_id}>
                {indexInPath > 0 && <ChevronRight aria-hidden="true" size={10} />}
                <button onClick={() => props.onFocusedNodeIDChange(node.node_id)}>{node.display_name || node.node_id}</button>
              </span>
            ))}
          </div>
        </div>
      )}
      <ScrollArea className="explorer-scroll">
        <div className="tree" role="tree" aria-label="节点与资源" aria-busy={query !== deferredQuery}>
          {rows.length === 0 && <p className="empty-copy">没有匹配的节点或资源</p>}
          {rows.map((row) => {
            const selected = isRowSelected(row, props.selection)
            const common: RowCommonProps = {
              row,
              active: activeKey === row.key,
              selected,
              setRef: setRowRef,
              onFocus: setActiveKey,
              onKeyDown: handleTreeKeyDown,
            }
            if (row.kind === 'node') {
              const nodeID = row.node!.node_id
              return (
                <NodeRow
                  key={row.key}
                  {...common}
                  expanded={expanded.has(nodeID) || deferredQuery.trim().length > 0}
                  expandable={hasChildren(index, nodeID)}
                  resourceCount={(index.resourcesByNodeID.get(nodeID) || []).length}
                  onSelect={(node) => props.onSelect({ kind: 'node', node })}
                  onToggle={toggleNode}
                />
              )
            }
            return (
              <ResourceRow
                key={row.key}
                {...common}
                resource={row.resource!}
                onSelect={(resource) => props.onSelect({ kind: 'resource', resource })}
                onAdd={props.onAdd}
              />
            )
          })}
        </div>
      </ScrollArea>
    </div>
  )
}

function hasChildren(index: ReturnType<typeof buildExplorerIndex>, nodeID: string): boolean {
  return (index.childrenByID.get(nodeID)?.length || 0) + (index.resourcesByNodeID.get(nodeID)?.length || 0) > 0
}

function isRowSelected(row: ExplorerRow, selection: WorkspaceSelection): boolean {
  if (!selection) return false
  if (row.kind === 'node') return selection.kind === 'node' && selection.node.node_id === row.node!.node_id
  return selection.kind === 'resource'
    && selection.resource.id.owner_node_id === row.resource!.id.owner_node_id
    && selection.resource.id.name === row.resource!.id.name
}

function rowIndent(depth: number): CSSProperties {
  return { '--tree-depth': Math.max(0, depth - 1) } as CSSProperties
}
