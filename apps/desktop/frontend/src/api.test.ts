import { beforeEach, describe, expect, it, vi } from 'vitest'
import * as App from '../wailsjs/go/main/App'
import { api } from './api'
import { node, query } from './discovery/fixtures.test-support'
import { TOPOLOGY_QUERY_SCHEMA } from './discovery/topology'
import { snapshot } from './variable/fixtures.test-support'

vi.mock('../wailsjs/go/main/App', async (original) => ({ ...await original<typeof App>(), OperateJSON: vi.fn(), TopologyJSON: vi.fn(), SnapshotJSON: vi.fn(), PollSubscription: vi.fn() }))

describe('Variable Wails JSON boundary', () => {
  it('returns an envelope with false and an exact string revision', async () => {
    vi.mocked(App.SnapshotJSON).mockResolvedValue(JSON.stringify(snapshot()))
    expect(await api.snapshot('719', 'anything')).toEqual(snapshot())
  })
  it.each(['{"value":false}', '"ZmFsc2U="', JSON.stringify(snapshot()).replace('"observed_at_unix_ms":100', '"observed_at_unix_ms":100.0')])('rejects unsupported or malformed read JSON', async (raw) => {
    vi.mocked(App.SnapshotJSON).mockResolvedValue(raw)
    await expect(api.snapshot('719', 'anything')).rejects.toThrow()
  })
  it('preserves the operation wire envelope and confirmed value', async () => {
    vi.mocked(App.OperateJSON).mockResolvedValue(JSON.stringify({ schema: 'mfh.variable-value.v1', payload: snapshot(false) }))
    const input = { version: 3, expected_instance_id: 'a'.repeat(32), expected_revision: '9007199254740993', value: true }
    expect(await api.operate('719', 'anything', 'write', 'mfh.variable-write.v3', input)).toEqual({ schema: 'mfh.variable-value.v1', payload: snapshot(false) })
    expect(App.OperateJSON).toHaveBeenLastCalledWith('719', 'anything', 'write', 'mfh.variable-write.v3', JSON.stringify(input))
  })
  it('receives a JSON subscription envelope without base64 conversion', async () => {
    const event = { kind: 'data', revision: '9007199254740993', value: snapshot(false) }
    vi.mocked(App.PollSubscription).mockResolvedValue(JSON.stringify({ kind: 'event', payload: event }))
    expect(await api.poll(1, 1000)).toEqual({ kind: 'event', payload: event })
  })
})

describe('typed topology API boundary', () => {
  beforeEach(() => vi.resetAllMocks())

  it.each([undefined, 0, 1, 2, 4096])('uses existing generic operation at depth %s', async (depth) => {
    const expected = depth ?? 1
    vi.mocked(App.OperateJSON).mockResolvedValue(JSON.stringify({ schema: TOPOLOGY_QUERY_SCHEMA, payload: query('41', [node('41')], expected) }))
    expect(await api.topology('41', depth)).toMatchObject({ root_node_id: '41', depth: expected })
    expect(App.OperateJSON).toHaveBeenCalledExactlyOnceWith('41', 'system/topology', expected === 1 ? 'children' : 'subtree', expected === 1 ? 'mfh.management.topology-children-request.v1' : 'mfh.management.topology-query-request.v1', JSON.stringify({ version: 1, depth: expected }))
    expect(App.TopologyJSON).not.toHaveBeenCalled()
  })

  it.each([-1, 0.5, 4097, NaN, Infinity, null, '1', true])('rejects invalid depth %s before sending', async (depth) => {
    await expect(api.topology('41', depth as number)).rejects.toThrow('depth')
    expect(App.OperateJSON).not.toHaveBeenCalled()
  })

  it.each(['0', '01', '-1', 'x', '18446744073709551616'])('rejects invalid owner %s', async (owner) => {
    await expect(api.topology(owner)).rejects.toThrow('owner')
    expect(App.OperateJSON).not.toHaveBeenCalled()
  })

  it.each(['Forbidden', 'Unsupported', 'NotFound', 'Unreachable'])('preserves %s without a full-snapshot fallback', async (message) => {
    vi.mocked(App.OperateJSON).mockRejectedValue(new Error(message))
    await expect(api.topology('1')).rejects.toThrow(message)
    expect(App.TopologyJSON).not.toHaveBeenCalled()
  })

  it('rejects mismatched output schemas and query roots', async () => {
    vi.mocked(App.OperateJSON).mockResolvedValue(JSON.stringify({ schema: 'mfh.management.topology.v1', payload: query('1', [node('1')]) }))
    await expect(api.topology('1')).rejects.toThrow('schema')
    vi.mocked(App.OperateJSON).mockResolvedValue(JSON.stringify({ schema: TOPOLOGY_QUERY_SCHEMA, payload: query('2', [node('2')]) }))
    await expect(api.topology('1')).rejects.toThrow('root')
  })
})
