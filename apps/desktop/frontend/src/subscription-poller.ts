import type { PollResult } from './api'

type Waiter = { resolve(value: PollResult): void; reject(error: unknown): void; deadline: number }
type BatchItem = { subscription_id: number; kind: 'event' | 'error' | 'closed'; payload?: unknown }

// One IPC poll serves all logical subscription waiters. Controllers still
// decode every event; visibility and presentation coalescing do not drop data.
export class SubscriptionPoller {
  private waiters = new Map<number, Waiter>()
  private pumping = false
  constructor(private call: (ids: string, timeout: number) => Promise<unknown>, private now = () => performance.now()) {}
  poll(id: number, timeout: number): Promise<PollResult> {
    if (!Number.isSafeInteger(id) || id <= 0 || !Number.isSafeInteger(timeout) || timeout <= 0 || timeout > 600_000) return Promise.reject(new Error('订阅轮询参数无效'))
    if (this.waiters.has(id)) return Promise.reject(new Error('同一订阅已有轮询请求'))
    if (this.waiters.size >= 4096) return Promise.reject(new Error('订阅轮询超过 4096 上限'))
    return new Promise<PollResult>((resolve, reject) => {
      this.waiters.set(id, { resolve, reject, deadline: this.now() + timeout })
      if (!this.pumping) { this.pumping = true; queueMicrotask(() => void this.pump()) }
    })
  }
  cancel(id: number) {
    const waiter = this.waiters.get(id)
    this.waiters.delete(id)
    waiter?.resolve({ kind: 'closed' })
  }
  private async pump() {
    try {
      while (this.waiters.size) {
        const ids = [...this.waiters.keys()]
        const requested = new Set(ids)
        const batch = await this.call(JSON.stringify(ids), 100)
        if (!Array.isArray(batch) || batch.length > 256) throw new Error('订阅批次格式或数量无效')
        const seen = new Set<number>()
        for (const item of batch as BatchItem[]) {
          if (!item || typeof item !== 'object' || !requested.has(item.subscription_id) || seen.has(item.subscription_id)
            || !['event', 'error', 'closed'].includes(item.kind) || Object.keys(item).some((key) => !['subscription_id', 'kind', 'payload'].includes(key))) throw new Error('订阅批次包含重复、未知句柄或无效事件')
          seen.add(item.subscription_id)
        }
        for (const item of batch as BatchItem[]) {
          const waiter = this.waiters.get(item.subscription_id)
          if (!waiter) continue // Explicitly cancelled during the in-flight IPC.
          this.waiters.delete(item.subscription_id)
          waiter.resolve({ kind: item.kind, payload: item.payload })
        }
        for (const [id, waiter] of this.waiters) {
          if (waiter.deadline <= this.now()) { this.waiters.delete(id); waiter.resolve({ kind: 'timeout' }) }
        }
        // The backend coalesces polls. A browser timer here can be throttled
        // when the native window is inactive and leave live events queued.
      }
    } catch (error) {
      const waiting = [...this.waiters.values()]
      this.waiters.clear()
      waiting.forEach((waiter) => waiter.reject(error))
    } finally {
      this.pumping = false
      if (this.waiters.size) { this.pumping = true; queueMicrotask(() => void this.pump()) }
    }
  }
}
