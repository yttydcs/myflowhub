import type { DesktopAPI } from '../api'

type Job = { run(): Promise<void> }
class Scheduler {
  private running = 0
  private queue: Job[] = []
  private head = 0
  run<T>(valid: () => boolean, operation: () => Promise<T>): Promise<T> {
    if (this.queue.length - this.head >= 8192) return Promise.reject(new Error('变量请求队列已满'))
    return new Promise<T>((resolve, reject) => {
      this.queue.push({ run: async () => {
        try {
          if (!valid()) throw new Error('变量会话已释放')
          resolve(await operation())
        } catch (error) { reject(error) }
      } })
      this.drain()
    })
  }
  private drain() {
    while (this.running < 32 && this.head < this.queue.length) {
      const job = this.queue[this.head++]!
      this.running++
      void job.run().finally(() => { this.running--; this.drain() })
    }
    if (this.head === this.queue.length) { this.queue = []; this.head = 0 }
    else if (this.head > 1024) { this.queue = this.queue.slice(this.head); this.head = 0 }
  }
}
const schedulers = new WeakMap<DesktopAPI, Scheduler>()
export function scheduleVariable<T>(api: DesktopAPI, valid: () => boolean, operation: () => Promise<T>): Promise<T> {
  let scheduler = schedulers.get(api)
  if (!scheduler) { scheduler = new Scheduler(); schedulers.set(api, scheduler) }
  return scheduler.run(valid, operation)
}
