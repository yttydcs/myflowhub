import type { DesktopAPI } from '../api'
import type { ResourceDescriptor } from '../types'
import { resourceKey } from '../lib/utils'
import { acquireVariable, MAX_VARIABLE_SESSIONS, type VariableController } from './controller'

type Entry = { resource: ResourceDescriptor; controller: VariableController; release(): void; unsubscribe(): void }
export type OverviewSnapshot = { count: number; live: number; failed: number; revision: number; resources: readonly ResourceDescriptor[] }
export class VariableOverviewStore {
  private entries = new Map<string, Entry>()
  private listeners = new Set<() => void>()
  private timer?: ReturnType<typeof setTimeout>
  private membershipChanged = false
  private snapshot: OverviewSnapshot = { count: 0, live: 0, failed: 0, revision: 0, resources: [] }
  constructor(private api: DesktopAPI, private profile: string) {}
  getSnapshot = () => this.snapshot
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener) } }
  controller(resource: ResourceDescriptor) { return this.entries.get(resourceKey(resource.id.owner_node_id, resource.id.name))?.controller }
  get hasDrafts() { return [...this.entries.values()].some((entry) => entry.controller.getSnapshot().dirty) }
  watch(resources: ResourceDescriptor[]) {
    const fresh = new Map(resources.filter((resource) => !this.controller(resource)).map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource]))
    if (this.entries.size + fresh.size > MAX_VARIABLE_SESSIONS) throw new Error(`总览最多保留 ${MAX_VARIABLE_SESSIONS} 个变量订阅`)
    const added: string[] = []
    try {
      for (const [key, resource] of fresh) {
        const acquired = acquireVariable(this.api, this.profile, resource)
        const unsubscribe = acquired.controller.subscribe(this.schedule)
        this.entries.set(key, { ...acquired, resource, unsubscribe })
        this.membershipChanged = true
        added.push(key)
      }
    } catch (error) { added.forEach((key) => this.remove(key)); throw error }
    finally { this.publish() }
  }
  unwatch(resource: ResourceDescriptor) { this.remove(resourceKey(resource.id.owner_node_id, resource.id.name)); this.publish() }
  stop() { for (const key of this.entries.keys()) this.remove(key); if (this.timer !== undefined) clearTimeout(this.timer); this.timer = undefined; this.publish() }
  private remove(key: string) { const entry = this.entries.get(key); if (this.entries.delete(key)) this.membershipChanged = true; entry?.unsubscribe(); entry?.release() }
  private schedule = () => {
    if (this.timer !== undefined) return
    this.timer = setTimeout(() => { this.timer = undefined; this.publish() }, 16)
  }
  private publish() {
    let live = 0, failed = 0
    for (const entry of this.entries.values()) {
      const state = entry.controller.getSnapshot()
      if (state.live) live++
      if (state.error || state.blocked) failed++
    }
    // Subscription ownership outlives discovery caches. Keep the descriptor list
    // stable on value-only updates so filtering 2000 rows is not repeated at 1Hz.
    const resources = this.membershipChanged ? [...this.entries.values()].map((entry) => entry.resource) : this.snapshot.resources
    this.membershipChanged = false
    this.snapshot = { count: this.entries.size, live, failed, revision: this.snapshot.revision + 1, resources }
    this.listeners.forEach((listener) => listener())
  }
}
