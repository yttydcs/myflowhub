import { useDeferredValue, useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent } from 'react'
import { useDraggable } from '@dnd-kit/core'
import * as ContextMenu from '@radix-ui/react-context-menu'
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  CircleDot,
  Command,
  FileUp,
  FolderTree,
  Gauge,
  GripVertical,
  Eye,
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
  type NodeExplorerRow,
} from '../store'
import {
  buildResourceTree,
  defaultExpandedResourcePaths,
  flattenResourceRows,
  type ResourceTreeRow,
} from '../lib/resource-tree'
import type { ExplorerCollapsedPane } from '../preferences'
import { deriveResourceActions, type ResourceAction } from '../lib/resource-actions'
import type { ResourceDescriptor, Topology, TopologyNode, WorkspaceSelection } from '../types'
import type { DiscoverySnapshot, LoadState } from '../discovery/controller'
import { DiscoveryStatus } from '../discovery/DiscoveryStatus'
import { ExplorerSplitPane } from './ExplorerSplitPane'
import { Input } from './ui/input'
import { ScrollArea } from './ui/scroll-area'

type Props = {
  topology: Topology
  discovery?: DiscoverySnapshot
  scopeRoot?: string
  defaultRoot?: string
  onScopeRootChange?(root: string, followParent?: boolean): void
  onLoadSubtree?(): void
  onExpandNode?(owner: string): void
  onRetryNode?(owner: string): void
  onCatalogOwnerChange?(owner: string): void
  onRetryCatalog?(owner: string): void
  resources: ResourceDescriptor[]
  selection: WorkspaceSelection
  expandedNodeIDs?: string[]
  expandedResourcePaths?: string[]
  focusedNodeID?: string
  splitRatio?: number
  collapsedPane?: ExplorerCollapsedPane
  onExpandedNodeIDsChange(nodeIDs: string[]): void
  onExpandedResourcePathsChange(paths: string[]): void
  onFocusedNodeIDChange(nodeID?: string): void
  onSplitRatioChange(ratio: number): void
  onCollapsedPaneChange(pane?: ExplorerCollapsedPane): void
  onSelect(selection: WorkspaceSelection): void
  onAdd(resource: ResourceDescriptor): void
  onAction(resource: ResourceDescriptor, action: ResourceAction): void
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
  resourceCount?: number
  state?: LoadState
  onRetry(): void
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
        aria-busy={props.state?.status === 'loading'}
        tabIndex={props.active ? 0 : -1}
        onFocus={() => props.onFocus(props.row.key)}
        onClick={() => props.onSelect(node)}
        onDoubleClick={() => props.expandable && props.onToggle(node.node_id)}
        onKeyDown={(event) => props.onKeyDown(event, props.row)}
      >
        <CircleDot aria-hidden="true" size={14} />
        <span className="tree-label">
          <strong>{node.display_name || `Node ${node.node_id}`}</strong>
          <small>{node.role}{props.state?.status === 'loading' ? ' · 加载中…' : props.state?.stale ? ' · 缓存待刷新' : ''}</small>
        </span>
      </button>
      <span className="tree-count" aria-label={props.resourceCount === undefined ? '资源目录尚未加载' : `${props.resourceCount} 个直接资源`}>{props.resourceCount ?? '—'}</span>
      {props.state?.status === 'error' && <div className="node-discovery-error"><DiscoveryStatus state={props.state} label={`节点 ${node.display_name || node.node_id}`} onRetry={props.onRetry} /></div>}
    </div>
  )
}

type ResourceRowProps = {
  row: ResourceTreeRow
  active: boolean
  selected: boolean
  expanded: boolean
  expandable: boolean
  setRef(key: string, element: HTMLButtonElement | null): void
  onFocus(key: string): void
  onKeyDown(event: KeyboardEvent<HTMLButtonElement>, row: ResourceTreeRow): void
  onToggle(key: string): void
  onSelect(resource: ResourceDescriptor): void
  onAdd(resource: ResourceDescriptor): void
  onAction(resource: ResourceDescriptor, action: ResourceAction): void
}

function ResourceRow(props: ResourceRowProps) {
  const { node } = props.row
  const resource = node.resource
  const mainRef = useRef<HTMLButtonElement | null>(null)
  const draggable = useDraggable({
    id: node.key,
    data: { kind: 'resource', resource },
    disabled: !resource,
  })
  const Icon = resource ? resourceIcons[resource.type] ?? Layers3 : FolderTree
  const actions = resource ? deriveResourceActions(resource.capabilities) : []
  const style = {
    ...rowIndent(props.row.depth),
    ...(draggable.transform
      ? { transform: `translate3d(${draggable.transform.x}px, ${draggable.transform.y}px, 0)` }
      : {}),
  }
  const row = (
    <div
      ref={draggable.setNodeRef}
      className={`tree-row-shell resource-row ${props.selected ? 'is-selected' : ''} ${draggable.isDragging ? 'is-dragging' : ''}`}
      role="none"
      style={style}
    >
      <button
        className="disclosure"
        onClick={() => props.onToggle(node.key)}
        disabled={!props.expandable}
        aria-label={`${props.expanded ? '折叠' : '展开'}资源路径 ${node.path}`}
        tabIndex={-1}
      >
        {props.expandable
          ? props.expanded ? <ChevronDown aria-hidden="true" size={13} /> : <ChevronRight aria-hidden="true" size={13} />
          : <span className="disclosure-placeholder" />}
      </button>
      {resource
        ? (
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
          )
        : <span className="drag-placeholder" aria-hidden="true" />}
      <button
        ref={(element) => {
          mainRef.current = element
          props.setRef(node.key, element)
        }}
        className="tree-main"
        role="treeitem"
        aria-level={props.row.depth}
        aria-posinset={props.row.posInSet}
        aria-setsize={props.row.setSize}
        aria-expanded={props.expandable ? props.expanded : undefined}
        aria-selected={resource ? props.selected : undefined}
        tabIndex={props.active ? 0 : -1}
        onFocus={() => props.onFocus(node.key)}
        onClick={() => resource ? props.onSelect(resource) : props.onToggle(node.key)}
        onDoubleClick={() => props.expandable && props.onToggle(node.key)}
        onKeyDown={(event) => {
          if (resource && (event.key === 'ContextMenu' || (event.key === 'F10' && event.shiftKey))) {
            event.preventDefault()
            event.stopPropagation()
            const bounds = mainRef.current?.getBoundingClientRect()
            mainRef.current?.dispatchEvent(new MouseEvent('contextmenu', {
              bubbles: true,
              cancelable: true,
              button: 2,
              clientX: bounds ? bounds.left + Math.min(bounds.width / 2, 24) : 0,
              clientY: bounds ? bounds.top + bounds.height / 2 : 0,
            }))
            return
          }
          props.onKeyDown(event, props.row)
        }}
      >
        <Icon aria-hidden="true" size={14} />
        <span className="tree-label">
          <strong>{resource?.presentation?.label || node.segment}</strong>
          <small>{node.path}</small>
        </span>
      </button>
      {resource
        ? (
            <button className="row-action" onClick={() => props.onAdd(resource)} aria-label={`添加 ${resource.id.name} 到工作区`}>
              <Plus aria-hidden="true" size={13} />
            </button>
          )
        : <span aria-hidden="true" />}
    </div>
  )
  if (!resource) return row
  const label = resource.presentation?.label || resource.id.name
  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger asChild>{row}</ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content
          className="resource-context-menu"
          aria-label={`${label} 操作菜单`}
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            requestAnimationFrame(() => mainRef.current?.focus())
          }}
        >
          <ContextMenu.Label className="resource-context-label">{label}</ContextMenu.Label>
          <ContextMenu.Item className="resource-context-item" onSelect={() => props.onSelect(resource)}><Eye aria-hidden="true" size={13} />查看和选择</ContextMenu.Item>
          <ContextMenu.Item className="resource-context-item" onSelect={() => props.onAdd(resource)}><Plus aria-hidden="true" size={13} />添加到当前 View</ContextMenu.Item>
          {actions.length > 0 && <ContextMenu.Separator className="resource-context-separator" />}
          {actions.map((action) => (
            <ContextMenu.Item
              className="resource-context-item"
              key={action.capability}
              disabled={action.disabled}
              title={action.disabledReason}
              onSelect={() => props.onAction(resource, action)}
            >
              {action.mode === 'observe' ? <Radio aria-hidden="true" size={13} /> : action.mode === 'session' ? <FileUp aria-hidden="true" size={13} /> : <Command aria-hidden="true" size={13} />}
              <span>{action.label}</span>
              {(action.mutating || action.unsafe) && <small>{action.unsafe ? '谨慎' : '显式'}</small>}
            </ContextMenu.Item>
          ))}
        </ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  )
}

export function Explorer(props: Props) {
  const [nodeQuery, setNodeQuery] = useState('')
  const [scopeInput, setScopeInput] = useState(props.scopeRoot || '')
  const [scopeOpen, setScopeOpen] = useState(false)
  useEffect(() => setScopeInput(props.scopeRoot || ''), [props.scopeRoot])
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
  const catalogState = props.discovery?.catalogs.get(resolvedNodeID)
  const resourceTree = useMemo(() => buildResourceTree(currentResources), [currentResources])
  const defaultResourceExpansion = useMemo(() => defaultExpandedResourcePaths(resourceTree), [resourceTree])
  const expandedResourcePaths = props.expandedResourcePaths ?? defaultResourceExpansion
  const expandedResources = useMemo(() => new Set(expandedResourcePaths), [expandedResourcePaths])
  const resourceRows = useMemo(
    () => flattenResourceRows(resourceTree, expandedResources, deferredResourceQuery),
    [deferredResourceQuery, expandedResources, resourceTree],
  )

  useEffect(() => {
    if (resolvedNodeID) props.onCatalogOwnerChange?.(resolvedNodeID)
  }, [resolvedNodeID, props.onCatalogOwnerChange])

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
    const firstKey = resourceRows[0]!.key
    if (!resourceRows.some((row) => row.key === activeResourceKey)) setActiveResourceKey(firstKey)
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
    const row = resourceRows[rowIndex]
    if (!row) return
    setActiveResourceKey(row.key)
    requestAnimationFrame(() => resourceRowRefs.current.get(row.key)?.focus())
  }

  function setExpanded(nodeID: string, shouldExpand: boolean) {
    const next = new Set(expanded)
    if (shouldExpand) { next.add(nodeID); props.onExpandNode?.(nodeID) }
    else next.delete(nodeID)
    props.onExpandedNodeIDsChange([...next])
  }

  function toggleNode(nodeID: string) {
    setExpanded(nodeID, !expanded.has(nodeID))
  }

  function setResourceExpanded(key: string, shouldExpand: boolean) {
    const next = new Set(expandedResources)
    if (shouldExpand) next.add(key)
    else next.delete(key)
    props.onExpandedResourcePathsChange([...next])
  }

  function toggleResource(key: string) {
    setResourceExpanded(key, !expandedResources.has(key))
  }

  function togglePane(pane: ExplorerCollapsedPane) {
    props.onCollapsedPaneChange(props.collapsedPane === pane ? undefined : pane)
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
      if (expandable && (!expanded.has(nodeID) || props.discovery?.children.get(nodeID)?.status === 'error')) setExpanded(nodeID, true)
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
        if (sibling.parentKey === row.parentKey && hasChildNodes(index, sibling.node.node_id)) {
          next.add(sibling.node.node_id)
          props.onExpandNode?.(sibling.node.node_id)
        }
      }
      props.onExpandedNodeIDsChange([...next])
    } else return
    event.preventDefault()
  }

  function handleResourceKeyDown(event: KeyboardEvent<HTMLButtonElement>, row: ResourceTreeRow) {
    const rowIndex = resourceRows.findIndex((candidate) => candidate.key === row.key)
    if (rowIndex < 0) return
    if (event.key === 'ArrowDown') focusResourceRow(Math.min(resourceRows.length - 1, rowIndex + 1))
    else if (event.key === 'ArrowUp') focusResourceRow(Math.max(0, rowIndex - 1))
    else if (event.key === 'Home') focusResourceRow(0)
    else if (event.key === 'End') focusResourceRow(resourceRows.length - 1)
    else if (event.key === 'ArrowRight') {
      if (row.node.children.length > 0 && !expandedResources.has(row.key)) setResourceExpanded(row.key, true)
      else {
        const childIndex = resourceRows.findIndex((candidate, indexInRows) => indexInRows > rowIndex && candidate.parentKey === row.key)
        if (childIndex >= 0) focusResourceRow(childIndex)
      }
    } else if (event.key === 'ArrowLeft') {
      if (row.node.children.length > 0 && expandedResources.has(row.key)) setResourceExpanded(row.key, false)
      else if (row.parentKey) {
        const parentIndex = resourceRows.findIndex((candidate) => candidate.key === row.parentKey)
        if (parentIndex >= 0) focusResourceRow(parentIndex)
      }
    } else if (event.key === 'Enter' || event.key === ' ') {
      if (row.node.resource) props.onSelect({ kind: 'resource', resource: row.node.resource })
      else if (row.node.children.length > 0) toggleResource(row.key)
    } else if (event.key === '*') {
      const next = new Set(expandedResources)
      for (const sibling of resourceRows) {
        if (sibling.parentKey === row.parentKey && sibling.node.children.length > 0) next.add(sibling.key)
      }
      props.onExpandedResourcePathsChange([...next])
    }
    else return
    event.preventDefault()
  }

  const nodePaneExpanded = props.collapsedPane !== 'node'
  const resourcePaneExpanded = props.collapsedPane !== 'resource'
  const nodePane = (
    <section className={`explorer-pane node-explorer-pane ${nodePaneExpanded ? '' : 'is-collapsed'}`} aria-label="节点列表">
      <header className="explorer-pane-heading">
        <button
          className="explorer-heading-disclosure"
          aria-expanded={nodePaneExpanded}
          aria-controls="explorer-node-content"
          onClick={() => togglePane('node')}
        >
          {nodePaneExpanded ? <ChevronDown aria-hidden="true" size={13} /> : <ChevronRight aria-hidden="true" size={13} />}
          <Network aria-hidden="true" size={13} />
          <strong>节点</strong>
          <small>{index.nodesByID.size}</small>
        </button>
      </header>
      <div id="explorer-node-content" className="explorer-pane-content" hidden={!nodePaneExpanded}>
        {props.onScopeRootChange && <div className="discovery-scope">
          <div className="discovery-scope-toolbar">
            <button type="button" className="discovery-scope-toggle" aria-label={`修改浏览起点，当前 Node ${props.scopeRoot}`} aria-expanded={scopeOpen} aria-controls="discovery-scope-options" onClick={() => setScopeOpen((open) => !open)}>
              {scopeOpen ? <ChevronDown aria-hidden="true" size={12} /> : <ChevronRight aria-hidden="true" size={12} />}<span>起点：{props.scopeRoot}</span>
            </button>
            <button type="button" onClick={props.onLoadSubtree}>加载完整子树</button>
          </div>
          {scopeOpen && <div id="discovery-scope-options" className="discovery-scope-options" role="region" aria-label="编辑浏览起点">
            <form onSubmit={(event) => { event.preventDefault(); props.onScopeRootChange?.(scopeInput.trim()) }}>
              <Input aria-label="浏览起点 Node ID" value={scopeInput} onChange={(event) => setScopeInput(event.target.value)} />
              <button type="submit">设为起点</button>
            </form>
            <button type="button" onClick={() => props.defaultRoot && props.onScopeRootChange?.(props.defaultRoot, true)}>直接父节点</button>
            <small>仅向下查询</small>
          </div>}
        </div>}
        {props.discovery?.limitError && <p className="discovery-status" role="alert">{props.discovery.limitError}</p>}
        {props.discovery && props.scopeRoot && (nodeRows.length === 0 || props.discovery.children.get(props.scopeRoot)?.status === 'loading') && <DiscoveryStatus state={props.discovery.children.get(props.scopeRoot)} label="节点树" onRetry={() => props.onRetryNode?.(props.scopeRoot!)} />}
        <div className="search-box">
          <Search aria-hidden="true" size={14} />
          <Input
            aria-label="搜索已加载节点"
            name="node-search"
            autoComplete="off"
            spellCheck={false}
            value={nodeQuery}
            onChange={(event) => setNodeQuery(event.target.value)}
            placeholder="搜索已加载节点"
          />
        </div>
        {breadcrumb.length > 0 && (
          <div className="tree-focus-bar">
            <button onClick={() => props.onFocusedNodeIDChange(undefined)} aria-label="返回已加载节点树">
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
            {nodeRows.length === 0 && (!props.discovery || props.discovery.children.get(props.scopeRoot || '')?.status === 'loaded') && <p className="empty-copy">没有匹配的已加载节点</p>}
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
                  resourceCount={!props.discovery || props.discovery.catalogs.get(nodeID)?.status === 'loaded' ? (index.resourcesByNodeID.get(nodeID) || []).length : undefined}
                  state={props.discovery?.children.get(nodeID)}
                  onRetry={() => props.onRetryNode?.(nodeID)}
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
      </div>
    </section>
  )

  const resourcePane = (
    <section className={`explorer-pane resource-explorer-pane ${resourcePaneExpanded ? '' : 'is-collapsed'}`} aria-label="资源列表">
      <header className="explorer-pane-heading">
        <button
          className="explorer-heading-disclosure"
          aria-expanded={resourcePaneExpanded}
          aria-controls="explorer-resource-content"
          onClick={() => togglePane('resource')}
        >
          {resourcePaneExpanded ? <ChevronDown aria-hidden="true" size={13} /> : <ChevronRight aria-hidden="true" size={13} />}
          <Layers3 aria-hidden="true" size={13} />
          <strong>资源</strong>
          <small title={currentNode?.display_name || currentNode?.node_id}>{currentNode ? currentNode.display_name || `Node ${currentNode.node_id}` : '未选择节点'} · {currentResources.length}</small>
        </button>
      </header>
      <div id="explorer-resource-content" className="explorer-pane-content" hidden={!resourcePaneExpanded}>
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
          <div className="tree resource-tree" role="tree" aria-label="资源" aria-busy={resourceQuery !== deferredResourceQuery || catalogState?.status === 'loading'}>
            {!currentNode && <p className="empty-copy">选择一个节点以查看资源</p>}
            {currentNode && props.discovery && <DiscoveryStatus state={catalogState} label="资源目录" onRetry={() => props.onRetryCatalog?.(resolvedNodeID)} />}
            {currentNode && (!props.discovery || catalogState?.status === 'loaded') && currentResources.length === 0 && <p className="empty-copy">此节点没有资源</p>}
            {currentNode && currentResources.length > 0 && resourceRows.length === 0 && <p className="empty-copy">没有匹配的资源</p>}
            {resourceRows.map((row) => (
              <ResourceRow
                key={row.key}
                row={row}
                active={activeResourceKey === row.key}
                selected={row.node.resource ? isResourceSelected(row.node.resource, props.selection) : false}
                expanded={expandedResources.has(row.key) || deferredResourceQuery.trim().length > 0}
                expandable={row.node.children.length > 0}
                setRef={setResourceRowRef}
                onFocus={setActiveResourceKey}
                onKeyDown={handleResourceKeyDown}
                onToggle={toggleResource}
                onSelect={(selected) => props.onSelect({ kind: 'resource', resource: selected })}
                onAdd={props.onAdd}
                onAction={props.onAction}
              />
            ))}
          </div>
        </ScrollArea>
      </div>
    </section>
  )

  return (
    <div className="explorer">
      <ExplorerSplitPane
        ratio={props.splitRatio}
        onRatioChange={props.onSplitRatioChange}
        collapsedPane={props.collapsedPane}
        topID="explorer-node-pane"
        bottomID="explorer-resource-pane"
        top={nodePane}
        bottom={resourcePane}
      />
    </div>
  )
}

function hasChildNodes(index: ReturnType<typeof buildExplorerIndex>, nodeID: string): boolean {
  return index.nodesByID.get(nodeID)?.has_children === true || (index.childrenByID.get(nodeID)?.length || 0) > 0
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
