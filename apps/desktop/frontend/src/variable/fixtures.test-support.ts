import { vi } from 'vitest'
import type { DesktopAPI, PollResult } from '../api'
import type { DataSchemaDefinition, JSONValue, VariableValueV1 } from '../generated/variable-contract.generated'
import type { ResourceDescriptor, ResourceEvent } from '../types'
import { VARIABLE_VALUE_SCHEMA, VARIABLE_WRITE_SCHEMA } from './protocol'

export function variableResource(subscribe = true, write = true): ResourceDescriptor {
  return {
    id: { owner_node_id: '719', name: 'unregistered/8f7ad/value' }, type: 'mfh.variable', type_version: 2,
    capabilities: [
      { name: 'read', permission: 'arbitrary.read', output_schema: VARIABLE_VALUE_SCHEMA, max_payload_bytes: 65536 },
      ...(subscribe ? [{ name: 'subscribe', permission: 'arbitrary.observe', event_schema: VARIABLE_VALUE_SCHEMA, max_payload_bytes: 65536 }] : []),
      ...(write ? [{ name: 'write', permission: 'arbitrary.write', input_schema: VARIABLE_WRITE_SCHEMA, output_schema: VARIABLE_VALUE_SCHEMA, max_payload_bytes: 65536 }] : []),
    ], limits: { max_payload_bytes: 65536 },
  }
}
export function snapshot(value: JSONValue = false, revision = '9007199254740993', definition: DataSchemaDefinition = { type: 'boolean' }, instance = 'a'.repeat(32)): VariableValueV1 {
  return { version: 1, instance_id: instance, revision, definition, value, status: 'ready', observed_at_unix_ms: 100 }
}
export function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: Error) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}
export function variableAPI(initial = snapshot()) {
  const events: PollResult[] = []
  let waiting: ((value: PollResult) => void) | undefined
  const push = (value: PollResult) => { if (waiting) { const receive = waiting; waiting = undefined; receive(value) } else events.push(value) }
  const api = {
    snapshot: vi.fn().mockResolvedValue(initial),
    subscribe: vi.fn().mockResolvedValue(17),
    cancel: vi.fn().mockImplementation(async () => push({ kind: 'closed' })),
    operate: vi.fn().mockResolvedValue({ schema: VARIABLE_VALUE_SCHEMA, payload: { ...initial, revision: String(BigInt(initial.revision) + 1n) } }),
    poll: vi.fn().mockImplementation(() => events.length ? Promise.resolve(events.shift()) : new Promise<PollResult>((resolve) => { waiting = resolve })),
  } as unknown as DesktopAPI
  const event = (value: VariableValueV1, kind: 'snapshot' | 'data' = 'data', resource = variableResource()) => {
    const envelope = { ...value }
    if (kind === 'data') delete envelope.definition
    push({ kind: 'event', payload: { kind, owner_node_id: resource.id.owner_node_id, resource_name: resource.id.name, capability: 'subscribe', schema: VARIABLE_VALUE_SCHEMA, revision: value.revision, value: envelope } satisfies ResourceEvent })
  }
  return { api, push, event }
}
