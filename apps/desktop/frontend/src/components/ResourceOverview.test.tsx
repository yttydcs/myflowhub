import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ResourceOverview } from './ResourceOverview'
import { overviewFixture } from '../variable/overview.test-support'
import { acquireVariable } from '../variable/controller'

describe('virtualized resource overview', () => {
  it('retains watched rows across discovery resets and releases them when explicitly stopped', async () => {
    const { api, resources, push, fail } = overviewFixture(2)
    const onSelect = vi.fn()
    const { rerender } = render(<ResourceOverview api={api} resources={resources} hidden={false} onSelect={onSelect} />)
    fireEvent.click(screen.getByRole('button', { name: '订阅筛选结果' }))
    await waitFor(() => expect(api.subscribe).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(screen.getAllByRole('cell', { name: '实时' })).toHaveLength(2))
    rerender(<ResourceOverview api={api} resources={[]} hidden={false} onSelect={onSelect} />)
    expect(screen.getByRole('button', { name: /^load\/0000/ })).toBeInTheDocument()
    await act(async () => { push('load/0000', true); fail('load/0001') })
    expect(screen.getByRole('cell', { name: 'true' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: '不可用' })).toBeInTheDocument()
    expect(api.cancel).toHaveBeenCalledTimes(1)
    const updated = { ...resources[0]!, presentation: { label: 'Current catalog label' } }
    rerender(<ResourceOverview api={api} resources={[updated]} hidden={false} onSelect={onSelect} />)
    fireEvent.click(screen.getByRole('button', { name: /Current catalog label/ }))
    expect(onSelect).toHaveBeenLastCalledWith(updated)
    expect(api.subscribe).toHaveBeenCalledTimes(2)
    fireEvent.click(screen.getByRole('button', { name: '停止订阅 load/0001' }))
    expect(screen.queryByRole('button', { name: /^load\/0001/ })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '停止全部' }))
    rerender(<ResourceOverview api={api} resources={[]} hidden={false} onSelect={onSelect} />)
    expect(screen.getByText(/当前没有匹配变量/)).toBeInTheDocument()
  })
  it('keeps authorized live values visible after a rejected write, then marks revoked subscriptions unavailable', async () => {
    const { api, resources, push, fail } = overviewFixture(1)
    vi.mocked(api.operate).mockRejectedValue(new Error('forbidden: write permission revoked'))
    render(<ResourceOverview api={api} resources={resources} hidden={false} onSelect={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: '订阅筛选结果' }))
    await waitFor(() => expect(screen.getByRole('cell', { name: '实时' })).toBeInTheDocument())
    const acquired = acquireVariable(api, 'standalone', resources[0]!)
    try {
      await act(async () => { acquired.controller.edit(true); await acquired.controller.apply() })
      expect(screen.getByRole('cell', { name: '实时 · 操作异常' })).toBeInTheDocument()
      expect(screen.getByRole('cell', { name: 'false' })).toBeInTheDocument()
      await act(async () => { push(resources[0]!.id.name, true) })
      expect(screen.getByRole('cell', { name: 'true' })).toBeInTheDocument()
      expect(acquired.controller.getSnapshot().writeOutcome).toBe('rejected')
      expect(api.operate).toHaveBeenCalledTimes(1)
      await act(async () => { fail(resources[0]!.id.name) })
      expect(screen.getByRole('cell', { name: '不可用' })).toBeInTheDocument()
      expect(screen.getByRole('cell', { name: 'true' })).toBeInTheDocument()
    } finally { acquired.release() }
  })
  it('browses 2000 resources with bounded DOM while filtering and scrolling leave subscriptions alive', async () => {
    const { api, resources } = overviewFixture()
    render(<ResourceOverview api={api} resources={resources} hidden={false} onSelect={vi.fn()} />)
    expect(screen.getAllByRole('row').length).toBeLessThan(60)
    fireEvent.click(screen.getByRole('button', { name: '订阅筛选结果' }))
    await waitFor(() => expect(api.subscribe).toHaveBeenCalledTimes(2000), { timeout: 10_000 })
    fireEvent.scroll(screen.getByRole('rowgroup'), { target: { scrollTop: 44_000 } })
    expect(screen.getByRole('button', { name: /^load\/1000/ })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('筛选总览变量'), { target: { value: 'load/1999' } })
    expect(screen.getByRole('button', { name: /^load\/1999/ })).toBeInTheDocument()
    expect(screen.getAllByRole('row')).toHaveLength(2)
    expect(api.cancel).not.toHaveBeenCalled()
    expect(api.subscribe).toHaveBeenCalledTimes(2000)
  })
})
