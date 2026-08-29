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
  Network,
  Plus,
  Radio,
  Search,
  Variable,
} from 'lucide-react'
import {
  buildExplorerIndex,
  defaultExpandedNodeIDs,
  explorerBreadcrumb,
  flattenNodeRows,
  groupResources,
  resourceRowKey,
  type NodeExplorerRow,
} from '../store'
import type { ResourceDescriptor, Topology, TopologyNode, WorkspaceSelection } from '../types'
import { ExplorerSplitPane } from './ExplorerSplitPane'
import { Input } from './ui/input'
import { ScrollArea } from './ui/scroll-area'

type Props = {
  topology: Topology
  resources: ResourceDescriptor[]
  selection: WorkspaceSelection
  expandedNodeIDs?: string[]
  focusedNodeID?: string
  splitRatio?: number
  onExpandedNodeIDsChange(nodeIDs: string[]): void
  onFocusedNodeIDChange(nodeID?: string): void
  onSplitRatioChange(ratio: number): void
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

type NodeRowProps = {
  row: NodeExplorerRow
  active: boolean
  current: boolean
  expanded: boolean
  expandable: boolean
  resourceCount: number
  setRef(key: string, element: HTMLButtonElement | null): void
  onFocus(key: string): void
  onKeyDown(event: KeyboardEvent<HTMLButtonElement>, row: NodeExplorerRow): void
  onSelect(node: TopologyNode): void
  onToggle(nodeID: string): void
}

function NodeRow(props: NodeRowProps) {
  const node = props.row.node
  return (
    <div className={`tree-row-shell node-row ${props.current ? 'is-selected' : ''}`} role="none" style={rowIndent(props.row.depth)}>
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
        aria-selected={props.current}
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

type ResourceRowProps = {
  resource: ResourceDescriptor
  active: boolean
  selected: boolean
  setRef(key: string, element: HTMLButtonElement | null): void
  onFocus(key: string): void
  onKeyDown(event: KeyboardEvent<HTMLButtonElement>, resource: ResourceDescriptor): void
  onSelect(resource: ResourceDescriptor): void
  onAdd(resource: ResourceDescriptor): void
}

function ResourceRow(props: ResourceRowProps) {
  const { resource } = props
  const key = resourceRowKey(resource)
  const draggable = useDraggable({
    id: key,
    data: { kind: 'resource', resource },
  })
  const Icon = resourceIcons[resource.type] ?? Layers3
  return (
    <div
      ref={draggable.setNodeRef}
      className={`tree-row-shell resource-row ${props.selected ? 'is-selected' : ''} ${draggable.isDragging ? 'is-dragging' : ''}`}
      role="listitem"
      style={draggable.transform
        ? { transform: `translate3d(${draggable.transform.x}px, ${draggable.transform.y}px, 0)` }
        : undefined}
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
        ref={(element) => props.setRef(key, element)}
        className="tree-main"
        aria-pressed={props.selected}
        tabIndex={props.active ? 0 : -1}
        onFocus={() => props.onFocus(key)}
        onClick={() => props.onSelect(resource)}
        onKeyDown={(event) => props.onKeyDown(event, resource)}
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
  const [nodeQuery, setNodeQuery] = useState('')
  const [resourceQuery, setResourceQuery] = useState('')
  const deferredNodeQuery = useDeferredValue(nodeQuery)
  const deferredResourceQuery = useDeferredValue(resourceQuery)
  const [activeNodeKey, setActiveNodeKey] = useState('')
  const [activeResourceKey, setActiveResourceKey] = useState('')
  const [currentNodeID, setCurrentNodeID] = useState('')
  const nodeRowRefs = useRef(new Map<string, HTMLButtonElement>())
  const resourceRowRefs = useRef(new Map<string, HTMLButtonElement>())
  const index = useMemo(() => buildExplorerIndex(props.topology, props.resources), [props.topology, props.resources])
  const defaultExpanded = useMemo(() => defaultExpandedNodeIDs(index), [index])
  const expandedNodeIDs = props.expandedNodeIDs ?? defaultExpanded
  const expanded = useMemo(() => new Set(expandedNodeIDs), [expandedNodeIDs])
  const nodeRows = useMemo(
    () => flattenNodeRows(index, expanded, deferredNodeQuery, props.focusedNodeID),
    [deferredNodeQuery, expanded, index, props.focusedNodeID],
  )
  const breadcrumb = useMemo(() => explorerBreadcrumb(index, props.focusedNodeID), [index, props.focusedNodeID])
  const selectedOwnerID = selectionOwnerID(props.selection)
  const resolvedNodeID = index.nodesByID.has(selectedOwnerID)
    ? selectedOwnerID
    : index.nodesByID.has(currentNodeID)
      ? currentNodeID
      : props.focusedNodeID && index.nodesByID.has(props.focusedNodeID)
        ? props.focusedNodeID
        : index.roots[0] || ''
  const currentNode = index.nodesByID.get(resolvedNodeID)
  const currentResources = index.resourcesByNodeID.get(resolvedNodeID) || []
  const resourceGroups = useMemo(
    () => groupResources(currentResources, deferredResourceQuery),
    [currentResources, deferredResourceQuery],
  )
  const resourceRows = useMemo(
    () => resourceGroups.flatMap((group) => group.resources),
    [resourceGroups],
  )

  useEffect(() => {
    if (resolvedNodeID && resolvedNodeID !== currentNodeID) setCurrentNodeID(resolvedNodeID)
  }, [currentNodeID, resolvedNodeID])

  useEffect(() => {
    if (nodeRows.length === 0) {
      if (activeNodeKey) setActiveNodeKey('')
      return
    }
    const firstRow = nodeRows[0]
    if (firstRow && !nodeRows.some((row) => row.key === activeNodeKey)) setActiveNodeKey(firstRow.key)
  }, [activeNodeKey, nodeRows])

  useEffect(() => {
    if (resourceRows.length === 0) {
      if (activeResourceKey) setActiveResourceKey('')
      return
    }
    const firstKey = resourceRowKey(resourceRows[0]!)
    if (!resourceRows.some((resource) => resourceRowKey(resource) === activeResourceKey)) setActiveResourceKey(firstKey)
  }, [activeResourceKey, resourceRows])

  function setNodeRowRef(key: string, element: HTMLButtonElement | null) {
    setMapRef(nodeRowRefs.current, key, element)
  }

  function setResourceRowRef(key: string, element: HTMLButtonElement | null) {
    setMapRef(resourceRowRefs.current, key, element)
  }

  function focusNodeRow(rowIndex: number) {
    const row = nodeRows[rowIndex]
    if (!row) return
    setActiveNodeKey(row.key)
    requestAnimationFrame(() => nodeRowRefs.current.get(row.key)?.focus())
  }

  function focusResourceRow(rowIndex: number) {
    const resource = resourceRows[rowIndex]
    if (!resource) return
    const key = resourceRowKey(resource)
    setActiveResourceKey(key)
    requestAnimationFrame(() => resourceRowRefs.current.get(key)?.focus())
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

  function selectNode(node: TopologyNode) {
    setCurrentNodeID(node.node_id)
    setResourceQuery('')
    props.onSelect({ kind: 'node', node })
  }

  function handleNodeKeyDown(event: KeyboardEvent<HTMLButtonElement>, row: NodeExplorerRow) {
    const rowIndex = nodeRows.findIndex((candidate) => candidate.key === row.key)
    if (rowIndex < 0) return
    if (event.key === 'ArrowDown') focusNodeRow(Math.min(nodeRows.length - 1, rowIndex + 1))
    else if (event.key === 'ArrowUp') focusNodeRow(Math.max(0, rowIndex - 1))
    else if (event.key === 'Home') focusNodeRow(0)
    else if (event.key === 'End') focusNodeRow(nodeRows.length - 1)
    else if (event.key === 'ArrowRight') {
      const nodeID = row.node.node_id
      const expandable = hasChildNodes(index, nodeID)
      if (expandable && !expanded.has(nodeID)) setExpanded(nodeID, true)
      else {
        const childIndex = nodeRows.findIndex((candidate, indexInRows) => indexInRows > rowIndex && candidate.parentKey === row.key)
        if (childIndex >= 0) focusNodeRow(childIndex)
      }
    } else if (event.key === 'ArrowLeft') {
      if (expanded.has(row.node.node_id) && hasChildNodes(index, row.node.node_id)) setExpanded(row.node.node_id, false)
      else if (row.parentKey) {
        const parentIndex = nodeRows.findIndex((candidate) => candidate.key === row.parentKey)
        if (parentIndex >= 0) focusNodeRow(parentIndex)
      }
    } else if (event.key === 'Enter' || event.key === ' ') selectNode(row.node)
    else if (event.key === '*') {
      const next = new Set(expanded)
      for (const sibling of nodeRows) {
        if (sibling.parentKey === row.parentKey && hasChildNodes(index, sibling.node.node_id)) next.add(sibling.node.node_id)
      }
      props.onExpandedNodeIDsChange([...next])
    } else return
    event.preventDefault()
  }

  function handleResourceKeyDown(event: KeyboardEvent<HTMLButtonElement>, resource: ResourceDescriptor) {
    const key = resourceRowKey(resource)
    const rowIndex = resourceRows.findIndex((candidate) => resourceRowKey(candidate) === key)
    if (rowIndex < 0) return
    if (event.key === 'ArrowDown') focusResourceRow(Math.min(resourceRows.length - 1, rowIndex + 1))
    else if (event.key === 'ArrowUp') focusResourceRow(Math.max(0, rowIndex - 1))
    else if (event.key === 'Home') focusResourceRow(0)
    else if (event.key === 'End') focusResourceRow(resourceRows.length - 1)
    else if (event.key === 'Enter' || event.key === ' ') props.onSelect({ kind: 'resource', resource })
    else return
    event.preventDefault()
  }

  const nodePane = (
    <section className="explorer-pane node-explorer-pane" aria-label="节点列表">
      <header className="explorer-pane-heading">
        <span><Network aria-hidden="true" size={13} /><strong>节点</strong></span>
        <small>{index.nodesByID.size}</small>
      </header>
      <div className="search-box">
        <Search aria-hidden="true" size={14} />
        <Input
          aria-label="搜索节点"
          name="node-search"
          autoComplete="off"
          spellCheck={false}
          value={nodeQuery}
          onChange={(event) => setNodeQuery(event.target.value)}
          placeholder="搜索节点"
        />
      </div>
      {breadcrumb.length > 0 && (
        <div className="tree-focus-bar">
          <button onClick={() => props.onFocusedNodeIDChange(undefined)} aria-label="返回完整节点树">
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
        <div className="tree" role="tree" aria-label="节点" aria-busy={nodeQuery !== deferredNodeQuery}>
          {nodeRows.length === 0 && <p className="empty-copy">没有匹配的节点</p>}
          {nodeRows.map((row) => {
            const nodeID = row.node.node_id
            return (
              <NodeRow
                key={row.key}
                row={row}
                active={activeNodeKey === row.key}
                current={resolvedNodeID === nodeID}
                expanded={expanded.has(nodeID) || deferredNodeQuery.trim().length > 0}
                expandable={hasChildNodes(index, nodeID)}
                resourceCount={(index.resourcesByNodeID.get(nodeID) || []).length}
                setRef={setNodeRowRef}
                onFocus={setActiveNodeKey}
                onKeyDown={handleNodeKeyDown}
                onSelect={selectNode}
                onToggle={toggleNode}
              />
            )
          })}
        </div>
      </ScrollArea>
    </section>
  )

  const resourcePane = (
    <section className="explorer-pane resource-explorer-pane" aria-label="资源列表">
      <header className="explorer-pane-heading">
        <span><Layers3 aria-hidden="true" size={13} /><strong>资源</strong></span>
        <small title={currentNode?.display_name || currentNode?.node_id}>{currentNode ? currentNode.display_name || `Node ${currentNode.node_id}` : '未选择节点'} · {currentResources.length}</small>
      </header>
      <div className="search-box">
        <Search aria-hidden="true" size={14} />
        <Input
          aria-label="搜索当前节点资源"
          name="resource-search"
          autoComplete="off"
          spellCheck={false}
          value={resourceQuery}
          onChange={(event) => setResourceQuery(event.target.value)}
          placeholder="搜索当前节点资源"
          disabled={!currentNode}
        />
      </div>
      <ScrollArea className="explorer-scroll resource-scroll">
        <div className="resource-groups" aria-busy={resourceQuery !== deferredResourceQuery}>
          {!currentNode && <p className="empty-copy">选择一个节点以查看资源</p>}
          {currentNode && currentResources.length === 0 && <p className="empty-copy">此节点没有资源</p>}
          {currentNode && currentResources.length > 0 && resourceGroups.length === 0 && <p className="empty-copy">没有匹配的资源</p>}
          {resourceGroups.map((group) => (
            <section className="resource-group" key={group.key} aria-label={`${group.label} 资源`}>
              <header className="resource-group-heading"><strong>{group.label}</strong><span>{group.resources.length}</span></header>
              <div role="list">
                {group.resources.map((resource) => (
                  <ResourceRow
                    key={resourceRowKey(resource)}
                    resource={resource}
                    active={activeResourceKey === resourceRowKey(resource)}
                    selected={isResourceSelected(resource, props.selection)}
                    setRef={setResourceRowRef}
                    onFocus={setActiveResourceKey}
                    onKeyDown={handleResourceKeyDown}
                    onSelect={(selected) => props.onSelect({ kind: 'resource', resource: selected })}
                    onAdd={props.onAdd}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      </ScrollArea>
    </section>
  )

  return (
    <div className="explorer">
      <ExplorerSplitPane
        ratio={props.splitRatio}
        onRatioChange={props.onSplitRatioChange}
        topID="explorer-node-pane"
        bottomID="explorer-resource-pane"
        top={nodePane}
        bottom={resourcePane}
      />
    </div>
  )
}

function hasChildNodes(index: ReturnType<typeof buildExplorerIndex>, nodeID: string): boolean {
  return (index.childrenByID.get(nodeID)?.length || 0) > 0
}

function selectionOwnerID(selection: WorkspaceSelection): string {
  if (!selection) return ''
  return selection.kind === 'node' ? selection.node.node_id : selection.resource.id.owner_node_id
}

function isResourceSelected(resource: ResourceDescriptor, selection: WorkspaceSelection): boolean {
  return selection?.kind === 'resource'
    && selection.resource.id.owner_node_id === resource.id.owner_node_id
    && selection.resource.id.name === resource.id.name
}

function setMapRef(map: Map<string, HTMLButtonElement>, key: string, element: HTMLButtonElement | null) {
  if (element) map.set(key, element)
  else map.delete(key)
}

function rowIndent(depth: number): CSSProperties {
  return { '--tree-depth': Math.max(0, depth - 1) } as CSSProperties
}
