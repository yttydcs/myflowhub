import { describe, expect, it } from 'vitest'
import type { CapabilityDescriptor } from '../types'
import { deriveResourceActions } from './resource-actions'

function capability(name: string, patch: Partial<CapabilityDescriptor> = {}): CapabilityDescriptor {
  return { name, permission: `ignored.${name}`, max_payload_bytes: 4096, ...patch }
}

describe('descriptor-driven Resource actions', () => {
  it('derives deterministic modes and conservative mutation flags from capability semantics', () => {
    const actions = deriveResourceActions([
      capability('subscribe', { event_schema: 'event.v1' }),
      capability('open', { input_schema: 'open.v1' }),
      capability('archive', { input_schema: 'archive.v1' }),
      capability('get', { input_schema: 'member.v1', output_schema: 'value.v1' }),
      capability('mixed', { input_schema: 'mixed-input.v1', output_schema: 'mixed-output.v1', event_schema: 'mixed-event.v1' }),
      capability('custom'),
    ])

    expect(actions.map((action) => [action.capability, action.mode])).toEqual([
      ['archive', 'operation'],
      ['custom', 'operation'],
      ['get', 'operation'],
      ['mixed', 'operation'],
      ['open', 'session'],
      ['subscribe', 'observe'],
    ])
    expect(actions.find((action) => action.capability === 'get')).toMatchObject({ mutating: false, unsafe: false })
    expect(actions.find((action) => action.capability === 'custom')).toMatchObject({ mutating: true, unsafe: false })
    expect(actions.find((action) => action.capability === 'archive')).toMatchObject({ mutating: true, unsafe: true })
  })

  it('does not infer authority from permission strings and only disables authoritative Forbidden', () => {
    const capabilities = [capability('invoke', { permission: 'looks.forbidden' })]
    expect(deriveResourceActions(capabilities)[0]).toMatchObject({ disabled: false })
    expect(deriveResourceActions(capabilities, {
      invoke: { status: 'forbidden', reason: '管理员策略已明确拒绝' },
    })[0]).toMatchObject({ disabled: true, disabledReason: '管理员策略已明确拒绝' })
  })
})
