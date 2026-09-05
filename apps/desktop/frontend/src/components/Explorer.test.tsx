import { DndContext } from '@dnd-kit/core'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import type { ExplorerCollapsedPane } from '../preferences'
import type { ResourceDescriptor, Topology, WorkspaceSelection } from '../types'
import { Explorer } from './Explorer'

const topology: Topology = {
  nodes: [
    { node_id: '1', display_name: 'Root', role: 'root', has_children: true, generation: 1 },
    { node_id: '2', parent_id: '1', display_name: 'Branch', role: 'branch', has_children: true, generation: 1 },
    { node_id: '3', parent_id: '2', display_name: 'Leaf', role: 'leaf', has_children: false, generation: 1 },
  ],
}

function resource(name: string, label: string, capabilities: ResourceDescriptor['capabilities'] = []): ResourceDescriptor {
  return {
    id: { owner_node_id: '3', name },
    type: 'mfh.variable',
    type_version: 1,
    capabilities,
    limits: { max_payload_bytes: 1024 },
    presentation: { label },
  }
}

const resources = [
  resource('sensors/temperature', 'Temperature'),
  resource('system/config', 'Config', [{ name: 'invoke', permission: 'ignored.invoke', input_schema: 'example.input.v1', output_schema: 'example.output.v1', max_payload_bytes: 1024 }]),
  resource('system/config/update', 'Update'),
  resource('system/health', 'Health'),
  resource('system/regions/eu/rack/temperature', 'Rack temperature'),
]

function Harness({ onAdd = vi.fn(), onAction = vi.fn() }: {
  onAdd?: (resource: ResourceDescriptor) => void
  onAction?: (resource: ResourceDescriptor, action: { capability: string }) => void
}) {
  const [selection, setSelection] = useState<WorkspaceSelection>(null)
  const [expandedNodes, setExpandedNodes] = useState<string[] | undefined>(['1', '2'])
  const [expandedResources, setExpandedResources] = useState<string[] | undefined>()
  const [focused, setFocused] = useState<string>()
  const [ratio, setRatio] = useState(0.35)
  const [collapsedPane, setCollapsedPane] = useState<ExplorerCollapsedPane>()
  return (
    <DndContext>
      <button onClick={() => { setFocused('3'); setExpandedNodes(['3']) }}>聚焦叶节点</button>
      <output aria-label="当前分隔比例">{ratio}</output>
      <Explorer
        topology={topology}
        resources={resources}
        selection={selection}
        expandedNodeIDs={expandedNodes}
        expandedResourcePaths={expandedResources}
        focusedNodeID={focused}
        splitRatio={ratio}
        collapsedPane={collapsedPane}
        onExpandedNodeIDsChange={setExpandedNodes}
        onExpandedResourcePathsChange={setExpandedResources}
        onFocusedNodeIDChange={setFocused}
        onSplitRatioChange={setRatio}
        onCollapsedPaneChange={setCollapsedPane}
        onSelect={setSelection}
        onAdd={onAdd}
        onAction={onAction}
      />
    </DndContext>
  )
}

async function selectLeaf() {
  fireEvent.click(screen.getByRole('treeitem', { name: /Leaf/ }))
  return screen.findByRole('tree', { name: '资源' })
}

function treeItemByPath(tree: HTMLElement, path: string): HTMLElement {
  const item = within(tree).getAllByRole('treeitem').find((candidate) => candidate.querySelector('small')?.textContent === path)
  if (!item) throw new Error(`missing Resource tree item: ${path}`)
  return item
}

describe('split Node and Resource explorer', () => {
  it('keeps a Node-only tree with a single roving tab stop and standard arrow navigation', async () => {
    render(<Harness />)
    const tree = screen.getByRole('tree', { name: '节点' })
    const root = within(tree).getByRole('treeitem', { name: /Root/ })
    const branch = within(tree).getByRole('treeitem', { name: /Branch/ })
    expect(root).toHaveAttribute('tabindex', '0')
    expect(branch).toHaveAttribute('aria-level', '2')
    expect(within(tree).queryByText('Temperature')).not.toBeInTheDocument()
    root.focus()
    fireEvent.keyDown(root, { key: 'ArrowDown' })
    await waitFor(() => expect(branch).toHaveFocus())
    fireEvent.keyDown(branch, { key: 'ArrowLeft' })
    expect(branch).toHaveAttribute('aria-expanded', 'false')
    fireEvent.keyDown(branch, { key: 'ArrowLeft' })
    await waitFor(() => expect(root).toHaveFocus())
  })

  it('renders path-derived namespace, Resource, and hybrid rows while preserving add actions', async () => {
    const onAdd = vi.fn()
    render(<Harness onAdd={onAdd} />)
    expect(screen.getByText('此节点没有资源')).toBeInTheDocument()
    const tree = await selectLeaf()
    expect(treeItemByPath(tree, 'system')).toHaveAttribute('aria-level', '1')
    const config = within(tree).getByRole('treeitem', { name: /Config/ })
    expect(config).toHaveAttribute('aria-level', '2')
    expect(config).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(screen.getByRole('button', { name: '展开资源路径 system/config' }))
    expect(await within(tree).findByRole('treeitem', { name: /Update/ })).toHaveAttribute('aria-level', '3')
    fireEvent.click(screen.getByRole('button', { name: '添加 system/config 到工作区' }))
    expect(onAdd).toHaveBeenCalledWith(resources[1])
  })

  it('implements WAI-ARIA Resource tree navigation without conflating focus and selection', async () => {
    render(<Harness />)
    const tree = await selectLeaf()
    const system = treeItemByPath(tree, 'system')
    system.focus()
    fireEvent.keyDown(system, { key: 'ArrowRight' })
    const config = within(tree).getByRole('treeitem', { name: /Config/ })
    await waitFor(() => expect(config).toHaveFocus())
    expect(config).toHaveAttribute('aria-selected', 'false')
    fireEvent.keyDown(config, { key: 'ArrowRight' })
    expect(config).toHaveAttribute('aria-expanded', 'true')
    fireEvent.keyDown(config, { key: 'ArrowRight' })
    const update = within(tree).getByRole('treeitem', { name: /Update/ })
    await waitFor(() => expect(update).toHaveFocus())
    fireEvent.keyDown(update, { key: 'ArrowLeft' })
    await waitFor(() => expect(config).toHaveFocus())
    fireEvent.keyDown(config, { key: 'Enter' })
    expect(config).toHaveAttribute('aria-selected', 'true')
  })

  it('opens a descriptor-driven menu only for real Resources without selecting, adding, or operating', async () => {
    const onAdd = vi.fn()
    const onAction = vi.fn()
    render(<Harness onAdd={onAdd} onAction={onAction} />)
    const tree = await selectLeaf()
    const namespace = treeItemByPath(tree, 'system')
    fireEvent.contextMenu(namespace)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()

    const config = within(tree).getByRole('treeitem', { name: /Config/ })
    fireEvent.contextMenu(config)
    const menu = await screen.findByRole('menu', { name: 'Config 操作菜单' })
    expect(within(menu).getByRole('menuitem', { name: '查看和选择' })).toBeInTheDocument()
    expect(within(menu).getByRole('menuitem', { name: '添加到当前 View' })).toBeInTheDocument()
    expect(within(menu).getByRole('menuitem', { name: /操作 invoke/ })).toBeInTheDocument()
    expect(onAdd).not.toHaveBeenCalled()
    expect(onAction).not.toHaveBeenCalled()
    fireEvent.click(within(menu).getByRole('menuitem', { name: /操作 invoke/ }))
    expect(onAction).toHaveBeenCalledWith(resources[1], expect.objectContaining({ capability: 'invoke' }))
    expect(onAdd).not.toHaveBeenCalled()
    fireEvent.contextMenu(config)
    const reopened = await screen.findByRole('menu', { name: 'Config 操作菜单' })
    fireEvent.click(within(reopened).getByRole('menuitem', { name: '添加到当前 View' }))
    expect(onAdd).toHaveBeenCalledWith(resources[1])
    fireEvent.contextMenu(config)
    const selectMenu = await screen.findByRole('menu', { name: 'Config 操作菜单' })
    fireEvent.click(within(selectMenu).getByRole('menuitem', { name: '查看和选择' }))
    expect(config).toHaveAttribute('aria-selected', 'true')
  })

  it.each([
    ['ContextMenu', false],
    ['F10', true],
  ])('opens with %s and returns focus to the treeitem on Escape', async (key, shiftKey) => {
    render(<Harness />)
    const tree = await selectLeaf()
    const config = within(tree).getByRole('treeitem', { name: /Config/ })
    config.focus()
    fireEvent.keyDown(config, { key, shiftKey })
    const menu = await screen.findByRole('menu', { name: 'Config 操作菜单' })
    fireEvent.keyDown(menu, { key: 'Escape' })
    await waitFor(() => expect(menu).not.toBeInTheDocument())
    await waitFor(() => expect(config).toHaveFocus())
  })

  it('shows matching Resource ancestors during search and restores saved expansion when cleared', async () => {
    render(<Harness />)
    const tree = await selectLeaf()
    const system = treeItemByPath(tree, 'system')
    fireEvent.keyDown(system, { key: 'ArrowLeft' })
    expect(within(tree).queryByRole('treeitem', { name: /Config/ })).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('搜索当前节点资源'), { target: { value: 'update' } })
    await within(tree).findByRole('treeitem', { name: /Update/ })
    expect(treeItemByPath(tree, 'system')).toHaveAttribute('aria-expanded', 'true')
    expect(within(tree).getByRole('treeitem', { name: /Config/ })).toHaveAttribute('aria-expanded', 'true')
    expect(within(tree).getByRole('treeitem', { name: /Update/ })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('搜索当前节点资源'), { target: { value: '' } })
    await waitFor(() => expect(within(tree).queryByRole('treeitem', { name: /Config/ })).not.toBeInTheDocument())
  })

  it('collapses either Explorer section, keeps one section available, and restores the split ratio', () => {
    render(<Harness />)
    const nodeHeading = screen.getByRole('button', { name: /^节点 3$/ })
    fireEvent.click(nodeHeading)
    expect(nodeHeading).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('tree', { name: '节点' })).not.toBeInTheDocument()
    expect(screen.getByRole('tree', { name: '资源' })).toBeInTheDocument()
    expect(screen.queryByRole('separator')).not.toBeInTheDocument()

    const resourceHeading = screen.getByRole('button', { name: /^资源 Root · 0$/ })
    fireEvent.click(resourceHeading)
    expect(screen.getByRole('tree', { name: '节点' })).toBeInTheDocument()
    expect(screen.queryByRole('tree', { name: '资源' })).not.toBeInTheDocument()
    fireEvent.click(resourceHeading)
    expect(screen.getByRole('tree', { name: '资源' })).toBeInTheDocument()
    expect(screen.getByRole('separator')).toHaveAttribute('aria-valuenow', '35')
  })

  it('searches deep Nodes with ancestors and retains focused-subtree breadcrumbs', async () => {
    render(<Harness />)
    fireEvent.change(screen.getByLabelText('搜索已加载节点'), { target: { value: 'leaf' } })
    const tree = screen.getByRole('tree', { name: '节点' })
    expect(within(tree).getByRole('treeitem', { name: /Root/ })).toBeInTheDocument()
    expect(within(tree).getByRole('treeitem', { name: /Branch/ })).toBeInTheDocument()
    expect(within(tree).getByRole('treeitem', { name: /Leaf/ })).toHaveAttribute('aria-level', '3')
    fireEvent.click(screen.getByRole('button', { name: '聚焦叶节点' }))
    expect(await screen.findByLabelText('当前节点路径')).toHaveTextContent('RootBranchLeaf')
    expect(screen.getByRole('button', { name: '返回已加载节点树' })).toBeInTheDocument()
    expect(within(tree).getByRole('treeitem', { name: /Leaf/ })).toHaveAttribute('aria-level', '1')
  })

  it('adjusts the bounded split from the keyboard and resets it on double click', async () => {
    render(<Harness />)
    const separator = screen.getByRole('separator', { name: '调整节点列表和资源列表高度' })
    expect(separator).toHaveAttribute('aria-valuenow', '35')
    fireEvent.keyDown(separator, { key: 'ArrowDown' })
    await waitFor(() => expect(separator).toHaveAttribute('aria-valuenow', '38'))
    expect(screen.getByLabelText('当前分隔比例')).toHaveTextContent('0.38')
    fireEvent.doubleClick(separator)
    await waitFor(() => expect(separator).toHaveAttribute('aria-valuenow', '35'))
  })
})
