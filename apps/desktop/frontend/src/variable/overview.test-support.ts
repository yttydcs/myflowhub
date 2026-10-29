import { vi } from 'vitest'
import type { DesktopAPI, PollResult } from '../api'
import type { ResourceEvent } from '../types'
import { snapshot, variableResource } from './fixtures.test-support'

export function overviewFixture(count = 2000) {
  const resources = Array.from({ length: count }, (_, index) => ({ ...variableResource(), id: { owner_node_id: '719', name: `load/${index.toString().padStart(4, '0')}` } }))
  const owners = new Map<number, string>()
  const initialized = new Set<number>()
  const pending = new Map<number, (value: PollResult) => void>()
  let nextID = 0
  const packet = (id: number, value = false): PollResult => ({ kind: 'event', payload: { kind: 'snapshot', owner_node_id: '719', resource_name: owners.get(id)!, capability: 'subscribe', schema: 'mfh.variable-value.v1', revision: value ? '2' : '1', value: snapshot(value, value ? '2' : '1') } satisfies ResourceEvent })
  const api = {
    snapshot: vi.fn().mockResolvedValue(snapshot(false, '1')),
    subscribe: vi.fn(async (_owner: string, name: string) => { const id = ++nextID; owners.set(id, name); return id }),
    poll: vi.fn((id: number) => {
      if (!initialized.has(id)) { initialized.add(id); return Promise.resolve(packet(id)) }
      return new Promise<PollResult>((resolve) => { pending.set(id, resolve) })
    }),
    cancel: vi.fn(async (id: number) => { pending.get(id)?.({ kind: 'closed' }); pending.delete(id) }),
    operate: vi.fn(),
  } as unknown as DesktopAPI
  const push = (name: string, value: boolean) => {
    const id = [...owners].find(([, item]) => item === name)?.[0]
    if (!id || !pending.has(id)) throw new Error('resource has no active poll')
    const receive = pending.get(id)!
    pending.delete(id)
    receive(packet(id, value))
  }
  const fail = (name: string) => {
    const id = [...owners].find(([, item]) => item === name)?.[0]
    if (!id || !pending.has(id)) throw new Error('resource has no active poll')
    const receive = pending.get(id)!
    pending.delete(id)
    receive({ kind: 'error', payload: { code: 'forbidden', message: 'subscription revoked' } })
  }
  return { api, resources, push, fail }
}
