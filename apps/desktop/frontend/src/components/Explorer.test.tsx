import { DndContext } from '@dnd-kit/core'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
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
const resource: ResourceDescriptor = {
  id: { owner_node_id: '3', name: 'sensors/temperature' },
  type: 'mfh.variable',
  type_version: 1,
  capabilities: [],
  limits: { max_payload_bytes: 1024 },
  presentation: { label: 'Temperature' },
}

function Harness() {
  const [selection, setSelection] = useState<WorkspaceSelection>(null)
  const [expanded, setExpanded] = useState<string[] | undefined>()
  const [focused, setFocused] = useState<string>()
  return (
    <DndContext>
      <button onClick={() => { setFocused('3'); setExpanded(['3']) }}>聚焦叶节点</button>
      <Explorer
        topology={topology}
        resources={[resource]}
        selection={selection}
        expandedNodeIDs={expanded}
        focusedNodeID={focused}
        onExpandedNodeIDsChange={setExpanded}
        onFocusedNodeIDChange={setFocused}
        onSelect={setSelection}
        onAdd={vi.fn()}
      />
    </DndContext>
  )
}

describe('accessible resource tree', () => {
  it('uses a single roving tab stop and standard tree arrow navigation', async () => {
    render(<Harness />)
    const root = screen.getByRole('treeitem', { name: /Root/ })
    const branch = screen.getByRole('treeitem', { name: /Branch/ })
    expect(root).toHaveAttribute('tabindex', '0')
    expect(branch).toHaveAttribute('aria-level', '2')
    root.focus()
    fireEvent.keyDown(root, { key: 'ArrowDown' })
    await waitFor(() => expect(branch).toHaveFocus())
    fireEvent.keyDown(branch, { key: 'ArrowLeft' })
    expect(branch).toHaveAttribute('aria-expanded', 'false')
    fireEvent.keyDown(branch, { key: 'ArrowLeft' })
    await waitFor(() => expect(root).toHaveFocus())
  })

  it('finds deep resources with ancestors and supports focused-subtree breadcrumbs', async () => {
    render(<Harness />)
    expect(screen.queryByRole('treeitem', { name: /Temperature/ })).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('搜索节点和资源'), { target: { value: 'temperature' } })
    expect(await screen.findByRole('treeitem', { name: /Temperature/ })).toHaveAttribute('aria-level', '4')
    fireEvent.change(screen.getByLabelText('搜索节点和资源'), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: '聚焦叶节点' }))
    expect(await screen.findByLabelText('当前节点路径')).toHaveTextContent('RootBranchLeaf')
    expect(screen.getByRole('button', { name: '返回完整资源树' })).toBeInTheDocument()
    expect(screen.getByRole('treeitem', { name: /Leaf/ })).toHaveAttribute('aria-level', '1')
    expect(screen.getByRole('treeitem', { name: /Temperature/ })).toHaveAttribute('aria-level', '2')
  })
})
