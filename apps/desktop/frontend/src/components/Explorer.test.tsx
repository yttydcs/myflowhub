import { DndContext } from '@dnd-kit/core'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import type { ResourceDescriptor, Topology, WorkspaceSelection } from '../types'
import { Explorer } from './Explorer'

const topology: Topology = {
  version: 1,
  epoch: 1,
  nodes: [
    { node_id: '1', display_name: 'Root', role: 'root', generation: 1 },
    { node_id: '2', parent_id: '1', display_name: 'Branch', role: 'branch', generation: 1 },
    { node_id: '3', parent_id: '2', display_name: 'Leaf', role: 'leaf', generation: 1 },
  ],
}
const resources: ResourceDescriptor[] = [
  {
    id: { owner_node_id: '3', name: 'sensors/temperature' },
    type: 'mfh.variable',
    type_version: 1,
    capabilities: [],
    limits: { max_payload_bytes: 1024 },
    presentation: { label: 'Temperature' },
  },
  {
    id: { owner_node_id: '3', name: 'system/health' },
    type: 'mfh.variable',
    type_version: 1,
    capabilities: [],
    limits: { max_payload_bytes: 1024 },
    presentation: { label: 'Health' },
  },
]

function Harness({ onAdd = vi.fn() }: { onAdd?: (resource: ResourceDescriptor) => void }) {
  const [selection, setSelection] = useState<WorkspaceSelection>(null)
  const [expanded, setExpanded] = useState<string[] | undefined>()
  const [focused, setFocused] = useState<string>()
  const [ratio, setRatio] = useState(0.35)
  return (
    <DndContext>
      <button onClick={() => { setFocused('3'); setExpanded(['3']) }}>聚焦叶节点</button>
      <output aria-label="当前分隔比例">{ratio}</output>
      <Explorer
        topology={topology}
        resources={resources}
        selection={selection}
        expandedNodeIDs={expanded}
        focusedNodeID={focused}
        splitRatio={ratio}
        onExpandedNodeIDsChange={setExpanded}
        onFocusedNodeIDChange={setFocused}
        onSplitRatioChange={setRatio}
        onSelect={setSelection}
        onAdd={onAdd}
      />
    </DndContext>
  )
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

  it('shows only the current Node Resources in presentation groups and preserves add actions', async () => {
    const onAdd = vi.fn()
    render(<Harness onAdd={onAdd} />)
    expect(screen.getByText('此节点没有资源')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('treeitem', { name: /Leaf/ }))
    expect(await screen.findByRole('region', { name: 'sensors 资源' })).toHaveTextContent('Temperature')
    expect(screen.getByRole('region', { name: 'system 资源' })).toHaveTextContent('Health')
    fireEvent.change(screen.getByLabelText('搜索当前节点资源'), { target: { value: 'health' } })
    expect(await screen.findByRole('button', { name: /Health/, pressed: false })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Temperature/, pressed: false })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '添加 system/health 到工作区' }))
    expect(onAdd).toHaveBeenCalledWith(resources[1])
  })

  it('searches deep Nodes with ancestors and retains focused-subtree breadcrumbs', async () => {
    render(<Harness />)
    fireEvent.change(screen.getByLabelText('搜索节点'), { target: { value: 'leaf' } })
    const tree = screen.getByRole('tree', { name: '节点' })
    expect(within(tree).getByRole('treeitem', { name: /Root/ })).toBeInTheDocument()
    expect(within(tree).getByRole('treeitem', { name: /Branch/ })).toBeInTheDocument()
    expect(within(tree).getByRole('treeitem', { name: /Leaf/ })).toHaveAttribute('aria-level', '3')
    fireEvent.click(screen.getByRole('button', { name: '聚焦叶节点' }))
    expect(await screen.findByLabelText('当前节点路径')).toHaveTextContent('RootBranchLeaf')
    expect(screen.getByRole('button', { name: '返回完整节点树' })).toBeInTheDocument()
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
