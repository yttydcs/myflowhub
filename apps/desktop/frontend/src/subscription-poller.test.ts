import { describe, expect, it, vi } from 'vitest'
import { SubscriptionPoller } from './subscription-poller'
import { deferred } from './variable/fixtures.test-support'

describe('shared subscription IPC poller', () => {
  it.each([
    '[]', null, {}, Array.from({ length: 257 }, (_, index) => ({ subscription_id: index + 1, kind: 'event' })),
    [{ subscription_id: 1, kind: 'event' }, { subscription_id: 1, kind: 'event' }],
    [{ subscription_id: 1, kind: 'unknown' }],
    [{ subscription_id: 1, kind: 'event', extra: true }],
  ].map((batch) => [batch]))('rejects malformed structured batches: %j', async (batch) => {
    const poller = new SubscriptionPoller(async () => batch)
    await expect(poller.poll(1, 1000)).rejects.toThrow('订阅批次')
  })
  it('continues ready batches without waiting for a browser timer', async () => {
    vi.useFakeTimers()
    const call = vi.fn(async () => [{ subscription_id: 1, kind: 'event', payload: true }])
    const poller = new SubscriptionPoller(call)
    const received: string[] = []
    const consumer = (async () => {
      for (let i = 0; i < 3; i++) received.push((await poller.poll(1, 1000)).kind)
    })()
    try {
      for (let i = 0; i < 20; i++) await Promise.resolve()
      expect(received).toEqual(['event', 'event', 'event'])
      await consumer
    } finally {
      poller.cancel(1)
      await vi.runOnlyPendingTimersAsync()
      vi.useRealTimers()
    }
  })
  it('batches 2000 logical consumers without reducing their requested subscription set', async () => {
    const delivered = new Set<number>()
    const call = vi.fn(async (raw: string) => {
      const ids = JSON.parse(raw) as number[]
      const items = ids.filter((id) => !delivered.has(id)).slice(0, 256)
      items.forEach((id) => delivered.add(id))
      return items.map((subscription_id) => ({ subscription_id, kind: 'event', payload: { value: subscription_id } }))
    })
    const poller = new SubscriptionPoller(call)
    const results = await Promise.all(Array.from({ length: 2000 }, (_, index) => poller.poll(index + 1, 1000)))
    expect(JSON.parse(call.mock.calls[0]![0])).toHaveLength(2000)
    expect(results).toHaveLength(2000)
    expect(delivered.size).toBe(2000)
    expect(call).toHaveBeenCalledTimes(8)
    expect(results[1999]?.payload).toEqual({ value: 2000 })
  })
  it('isolates cancellation and rejects duplicate or unknown batch IDs', async () => {
    const batch = deferred<unknown>()
    const poller = new SubscriptionPoller(() => batch.promise)
    const first = poller.poll(1, 1000)
    const second = poller.poll(2, 1000)
    await expect(poller.poll(2, 1000)).rejects.toThrow('已有')
    poller.cancel(1)
    batch.resolve([{ subscription_id: 2, kind: 'event', payload: false }])
    expect(await first).toEqual({ kind: 'closed' })
    expect(await second).toEqual({ kind: 'event', payload: false })
    const invalid = new SubscriptionPoller(async () => [{ subscription_id: 99, kind: 'event' }])
    await expect(invalid.poll(3, 1000)).rejects.toThrow('未知')
  })
})
