import type { DesktopAPI, OperationResult } from '../api'
import type { ResourceDescriptor, ResourceEvent } from '../types'
import type { VariableValueV1, VariableWriteV3 } from '../generated/variable-contract.generated'
import { scheduleVariable } from './scheduler'
import { setUnsavedChanges } from '../unsaved'
import { descriptorError, parseVariableEnvelope, parseVariableEvent, parseVariableOperationPayload, parseRevision, validateVariableValue, sameValue, variablePayloadLimit, VARIABLE_VALUE_SCHEMA, VARIABLE_WRITE_SCHEMA } from './protocol'

function hasNegativeZero(value: unknown): boolean {
  if (typeof value === 'number') return Object.is(value, -0)
  return Boolean(value && typeof value === 'object' && Object.values(value).some(hasNegativeZero))
}

export type VariableSessionState = {
  snapshot?: VariableValueV1
  draft?: unknown
  base?: VariableValueV1
  dirty: boolean
  conflict: boolean
  blocked: string
  error: string
  observation: string
  live: boolean
  loading: boolean
  writing: boolean
  writeOutcome?: 'pending' | 'confirmed' | 'rejected' | 'uncertain' | 'confirmed-unreadable'
}

// A session owns one resource in one Profile. Definitions live only here and
// are discarded with the last consumer; an instance change retires old CAS bases.
export class VariableController {
  private state: VariableSessionState = { dirty: false, conflict: false, blocked: '', error: '', observation: '', live: false, loading: false, writing: false }
  private listeners = new Set<() => void>()
  private active = false
  private epoch = 0
  private request = 0
  private subscription?: number
  private observing = false
  private retired = new Set<string>()
  private invalidDraft = false
  private readBlocked = ''
  private draftGeneration = 0
  private submission = 0
  private writeConfirmation?: { state: VariableValueV1; draftVersion: number }
  private unsavedOwner = Symbol('variable draft')
  constructor(readonly api: DesktopAPI, readonly profile: string, readonly resource: ResourceDescriptor) {}
  getSnapshot = () => this.state
  get draftVersion() { return this.draftGeneration }
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener) } }
  private update(patch: Partial<VariableSessionState>) {
    this.state = { ...this.state, ...patch }
    setUnsavedChanges(this.unsavedOwner, this.active && (this.state.dirty || this.state.writing))
    this.listeners.forEach((listener) => listener())
  }
  get sessionKey() { return JSON.stringify([this.profile, this.resource.id.owner_node_id, this.resource.id.name, this.state.snapshot?.instance_id]) }
  start() {
    if (this.active) return
    this.active = true
    this.epoch++
    const reason = descriptorError(this.resource)
    if (reason) { this.update({ blocked: reason, error: reason }); return }
    void this.refresh(false)
    if (this.resource.capabilities.some((capability) => capability.name === 'subscribe')) this.ensureObservation()
    else this.update({ observation: '未声明 subscribe；可手动刷新' })
  }
  stop() {
    this.submission++
    this.writeConfirmation = undefined
    this.draftGeneration++
    this.active = false
    this.epoch++
    const id = this.subscription
    this.subscription = undefined
    this.observing = false
    if (id !== undefined) void this.cancel(id)
    this.retired.clear()
    this.invalidDraft = false
    this.readBlocked = ''
    this.update({ snapshot: undefined, base: undefined, draft: undefined, dirty: false, conflict: false, live: false, loading: false, writing: false, blocked: '会话已释放' })
  }
  private async cancel(id: number) {
    try { await this.api.cancel(id) }
    catch (error) { if (this.active) this.update({ observation: `取消订阅失败：${String(error)}`, live: false }) }
  }
  private current(epoch: number) { return this.active && epoch === this.epoch }
  async refresh(restartObservation = true) {
    if (!this.active || descriptorError(this.resource)) return
    if (!this.resource.capabilities.some((capability) => capability.name === 'read')) {
      if (restartObservation) this.ensureObservation()
      this.update({ error: '未声明 read；等待订阅快照' }); return
    }
    const epoch = this.epoch
    const request = ++this.request
    const instance = this.state.snapshot?.instance_id
    this.update({ loading: true })
    try {
      const result = await scheduleVariable(this.api, () => this.current(epoch), () => this.api.snapshot(this.resource.id.owner_node_id, this.resource.id.name))
      if (!this.current(epoch) || request !== this.request) return
      const next = parseVariableEnvelope(result, undefined, variablePayloadLimit(this.resource, 'read'))
      // A read issued for an earlier instance cannot switch a newer instance back.
      if (this.state.snapshot?.instance_id !== instance && this.state.snapshot?.instance_id !== next.instance_id) return
      this.readBlocked = ''
      this.accept(next)
      const confirmed = this.writeConfirmation
      if (confirmed && next.status !== 'unavailable' && next.instance_id === confirmed.state.instance_id && BigInt(next.revision) >= BigInt(confirmed.state.revision) && this.draftGeneration === confirmed.draftVersion) {
        this.writeConfirmation = undefined
        this.invalidDraft = false
        this.update({ dirty: false, conflict: false, draft: this.state.snapshot?.value, base: this.state.snapshot, writeOutcome: 'confirmed' })
      }
      this.update({ error: '' })
    } catch (error) {
      if (this.current(epoch) && request === this.request) {
        this.readBlocked = `读取失败，先刷新确认：${String(error)}`
        if (this.state.dirty) this.invalidDraft = true
        this.update({ error: String(error), blocked: this.readBlocked })
      }
    } finally {
      if (this.current(epoch) && request === this.request) {
        this.update({ loading: false })
        if (restartObservation) this.ensureObservation()
      }
    }
  }
  private accept(next: VariableValueV1) {
    const previous = this.state.snapshot
    if (this.retired.has(next.instance_id)) return
    const changedInstance = previous && previous.instance_id !== next.instance_id
    if (previous && !changedInstance) {
      if (BigInt(next.revision) < BigInt(previous.revision)) return
      if (!sameValue(previous.definition, next.definition)) throw new Error('同一实例的 definition 发生变化')
      if (BigInt(next.revision) === BigInt(previous.revision) && !sameValue(previous, next)) throw new Error('同一 revision 的状态不一致')
    }
    if (changedInstance) {
      this.writeConfirmation = undefined
      if (this.retired.size >= 64) throw new Error('资源实例变更过于频繁；请重新打开资源')
      this.retired.add(previous.instance_id)
    }
    const patch: Partial<VariableSessionState> = { snapshot: next, blocked: this.readBlocked || (next.status === 'unavailable' ? '当前值不可用' : '') }
    if (this.state.dirty) {
      if (changedInstance || this.state.base?.instance_id !== next.instance_id) {
        this.invalidDraft = true
        patch.conflict = true
        patch.blocked = '资源已重建；重置旧草稿后再编辑'
      } else if (next.status === 'unavailable') {
        this.invalidDraft = true
        patch.blocked = '当前值不可用；重置草稿前禁止提交'
      } else if (this.invalidDraft) {
        patch.blocked = '草稿因资源或权限变化已失效；请重置'
      } else if (this.state.conflict) {
        patch.conflict = true
      } else if (sameValue(next.value, this.state.base?.value)) {
        patch.base = next
        patch.conflict = false
      } else patch.conflict = true
    } else {
      patch.base = next
      patch.draft = next.value
      patch.conflict = false
    }
    this.update(patch)
  }
  edit(value: unknown) {
    if (!this.state.snapshot || this.state.writing) return
    this.draftGeneration++
    this.update({ draft: value, dirty: true, base: this.state.dirty ? this.state.base : this.state.snapshot, error: '' })
  }
  reset() {
    const snapshot = this.state.snapshot
    if (this.state.writing) return
    this.draftGeneration++
    this.invalidDraft = false
    this.update({ draft: snapshot?.value, base: snapshot, dirty: false, conflict: false, error: '', blocked: this.readBlocked || (snapshot?.status === 'unavailable' ? '当前值不可用' : '') })
  }
  async apply() {
    const { base, snapshot, draft, dirty, blocked, conflict, writing } = this.state
    if (!this.active || !base || !snapshot || !dirty || blocked || conflict || writing) return
    if (!this.resource.capabilities.some((capability) => capability.name === 'write') || snapshot.definition?.read_only) return
    const epoch = this.epoch
    const submission = ++this.submission
    const draftVersion = this.draftGeneration
    const current = () => this.current(epoch) && this.submission === submission && this.state.snapshot?.instance_id === base.instance_id
    let sent = false
    let acknowledged = false
    let timer: ReturnType<typeof setTimeout> | undefined
    const deadline = new Promise<never>((_, reject) => { timer = setTimeout(() => reject(new Error('已达到 5 秒最大等待时间')), 5000) })
    try {
      validateVariableValue(snapshot.definition!, draft)
      const request: VariableWriteV3 = { version: 3, expected_instance_id: base.instance_id, expected_revision: base.revision, value: draft }
      if (new TextEncoder().encode(JSON.stringify(request)).byteLength > variablePayloadLimit(this.resource, 'write')) throw new Error('写请求超过资源大小限制')
      this.writeConfirmation = undefined
      this.update({ writing: true, writeOutcome: 'pending', error: '' })
      const canRead = this.resource.capabilities.some((capability) => capability.name === 'read')
      let result: OperationResult
      // JSON.stringify erases -0 in a base snapshot. Keep that rare case on
      // the existing path so its Object.is conflict boundary is preserved.
      if (canRead && this.api.writeVariableFromSnapshot && !hasNegativeZero(base)) {
        // Once dispatched, a lost native reply may hide a write. The returned
        // stage can prove a preflight rejection, but a timeout cannot.
        sent = true
        const attempt = await Promise.race([this.api.writeVariableFromSnapshot(this.resource.id.owner_node_id, this.resource.id.name, base, request), deadline])
        if (!current()) return
        if (!attempt || typeof attempt.sent !== 'boolean' || Object.keys(attempt).some((key) => !['sent', 'expected_revision', 'response_json', 'error'].includes(key))
          || attempt.error !== undefined && typeof attempt.error !== 'string') throw new Error('写入阶段响应无效')
        sent = attempt.sent
        if (attempt.error) throw new Error(attempt.error)
        if (!sent || typeof attempt.response_json !== 'string' || parseRevision(attempt.expected_revision) < parseRevision(base.revision)) throw new Error('写入阶段没有确认有效修订')
        request.expected_revision = attempt.expected_revision!
        result = parseVariableOperationPayload(attempt.response_json)
      } else {
        if (canRead) {
          // Sampling can advance metadata while a draft is open. Read once before
          // dispatch, without rebasing over a changed value or retrying a write.
          const read = await Promise.race([this.api.snapshot(this.resource.id.owner_node_id, this.resource.id.name), deadline])
          if (!current()) return
          const observed = parseVariableEnvelope(read, undefined, variablePayloadLimit(this.resource, 'read'))
          this.accept(observed)
          if (!current()) return
          const latest = this.state.snapshot!
          if (latest.status === 'unavailable' || this.state.blocked) throw new Error('写前读取没有确认可用状态')
          if (this.state.conflict || !sameValue(latest.value, base.value)) throw new Error('conflict: 当前值已变化，请确认后重新编辑')
          request.expected_revision = latest.revision
        }
        sent = true
        result = await Promise.race([this.api.operate(this.resource.id.owner_node_id, this.resource.id.name, 'write', VARIABLE_WRITE_SCHEMA, request), deadline])
      }
      if (!current()) return
      if (result.schema !== VARIABLE_VALUE_SCHEMA) throw new Error('写响应 schema 不受支持')
      const confirmed = parseVariableEnvelope(result.payload, undefined, variablePayloadLimit(this.resource, 'write'))
      if (confirmed.instance_id !== base.instance_id || BigInt(confirmed.revision) <= BigInt(request.expected_revision)) throw new Error('写响应没有确认当前实例的新 revision')
      acknowledged = true
      this.accept(confirmed)
      this.writeConfirmation = { state: confirmed, draftVersion }
      const latest = this.state.snapshot!
      this.update({ base: confirmed, conflict: BigInt(latest.revision) > BigInt(confirmed.revision) && !sameValue(latest.value, confirmed.value) })
      // An acknowledgement and a fresh read are distinct evidence. Keep the
      // submitted draft until the read finishes; never replay the write.
      const read = await Promise.race([this.api.snapshot(this.resource.id.owner_node_id, this.resource.id.name), deadline])
      if (!current()) return
      const actual = parseVariableEnvelope(read, undefined, variablePayloadLimit(this.resource, 'read'))
      if (actual.instance_id !== base.instance_id || BigInt(actual.revision) < BigInt(confirmed.revision) || actual.status === 'unavailable') throw new Error('回读没有确认写入后的可用状态')
      if (actual.revision === confirmed.revision && !sameValue(actual, confirmed)) throw new Error('回读与写响应在同一 revision 下不一致')
      this.accept(actual)
      this.writeConfirmation = undefined
      this.invalidDraft = false
      this.update({ dirty: false, conflict: false, draft: this.state.snapshot?.value, base: this.state.snapshot, writeOutcome: 'confirmed' })
    } catch (error) {
      if (current()) {
        const rejected = !sent || /\b(forbidden|conflict|revision_conflict|not_found|unsupported|unsupported_capability|gone)\b/i.test(String(error))
        if (acknowledged) {
          this.readBlocked = '写入已确认，回读失败；请刷新确认当前值'
          this.update({ error: `${this.readBlocked}：${String(error)}`, blocked: this.readBlocked, writeOutcome: 'confirmed-unreadable' })
        } else if (rejected) {
          const denied = /\bforbidden\b/i.test(String(error))
          if (denied) this.readBlocked = '没有写入权限；重新授权后请刷新确认'
          this.update({ error: `写入已拒绝：${String(error)}`, writeOutcome: 'rejected', ...(denied ? { blocked: this.readBlocked } : {}) })
        } else {
          this.readBlocked = '写入结果不确定；请刷新确认后重新决定'
          this.update({ error: `${this.readBlocked}：${String(error)}`, blocked: this.readBlocked, writeOutcome: 'uncertain' })
        }
      }
    } finally {
      if (timer !== undefined) clearTimeout(timer)
      if (this.current(epoch) && this.submission === submission) {
        this.submission++
        this.update({ writing: false, ...(!sent && this.state.writeOutcome === 'pending' ? { writeOutcome: 'rejected' as const } : {}) })
      }
    }
  }
  private ensureObservation() {
    if (!this.active || this.observing || !this.resource.capabilities.some((capability) => capability.name === 'subscribe')) return
    this.observing = true
    void this.observe(this.epoch)
  }
  private async observe(epoch: number) {
    let id: number | undefined
    let registered = false
    try {
      this.update({ observation: '正在订阅…', live: false })
      id = await scheduleVariable(this.api, () => this.current(epoch), () => this.api.subscribe(this.resource.id.owner_node_id, this.resource.id.name, 'subscribe', 60_000))
      if (!this.current(epoch)) return
      this.subscription = id
      registered = true
      let receivedSnapshot = false
      while (this.current(epoch)) {
        const result = await this.api.poll(id, 1000)
        if (!this.current(epoch)) return
        if (result.kind === 'timeout') continue
        if (result.kind !== 'event') throw new Error(result.kind === 'closed' ? '订阅已关闭；可手动刷新' : JSON.stringify(result.payload))
        const event = result.payload as ResourceEvent
        if (event.kind === 'gap' || event.kind === 'expired') {
          if (event.owner_node_id !== this.resource.id.owner_node_id || event.resource_name !== this.resource.id.name) throw new Error('订阅控制事件身份不符')
          // The SDK owns lease renewal and connection recovery. Keep polling
          // its durable handle, but require a complete new snapshot before data.
          receivedSnapshot = false
          this.readBlocked = '订阅恢复中；等待完整快照'
          this.update({ observation: `${event.kind}: ${event.reason || '正在恢复订阅'}`, live: false, blocked: this.readBlocked })
          continue
        }
        if (!receivedSnapshot && event.kind !== 'snapshot') throw new Error('订阅缺少初始快照')
        const next = parseVariableEvent(event, this.resource, event.kind === 'snapshot' ? undefined : this.state.snapshot)
        if (event.kind === 'snapshot' && this.readBlocked === '订阅恢复中；等待完整快照') this.readBlocked = ''
        this.accept(next)
        receivedSnapshot = true
        this.update({ observation: '实时订阅中', live: true })
      }
    } catch (error) {
      if (this.current(epoch)) {
        if (this.state.live) {
          this.readBlocked = '订阅已失效；请刷新确认权限与当前值'
          if (this.state.dirty) this.invalidDraft = true
        }
        this.update({ observation: `未订阅：${String(error)}`, live: false, ...(this.readBlocked ? { blocked: this.readBlocked } : {}) })
      }
    } finally {
      if (this.current(epoch)) this.observing = false
      if (id !== undefined && (!registered || this.subscription === id)) {
        if (this.subscription === id) this.subscription = undefined
        await this.cancel(id)
      }
    }
  }
}

const pools = new WeakMap<DesktopAPI, Map<string, { controller: VariableController; references: number }>>()
export const MAX_VARIABLE_SESSIONS = 4096
export function acquireVariable(api: DesktopAPI, profile: string, resource: ResourceDescriptor) {
  let pool = pools.get(api)
  if (!pool) { pool = new Map(); pools.set(api, pool) }
  const key = JSON.stringify([profile, resource.id, resource.type_version, resource.capabilities, resource.limits])
  let entry = pool.get(key)
  if (!entry) {
    if (pool.size >= MAX_VARIABLE_SESSIONS) throw new Error(`最多同时打开 ${MAX_VARIABLE_SESSIONS} 个变量会话`)
    entry = { controller: new VariableController(api, profile, resource), references: 0 }
    pool.set(key, entry)
  }
  entry.references++
  entry.controller.start()
  let released = false
  return { controller: entry.controller, release() {
    if (released) return
    released = true
    if (--entry.references === 0) { entry.controller.stop(); pool!.delete(key) }
  } }
}
