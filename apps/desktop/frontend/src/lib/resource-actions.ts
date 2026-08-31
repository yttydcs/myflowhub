import type { CapabilityDescriptor, ResourceDescriptor } from '../types'

export type ResourceActionMode = 'operation' | 'observe' | 'session'

export type AuthoritativeCapabilityState = Readonly<Record<string, {
  status: 'allowed' | 'forbidden'
  reason?: string
}>>

export type ResourceAction = {
  capability: string
  descriptor: CapabilityDescriptor
  mode: ResourceActionMode
  label: string
  mutating: boolean
  unsafe: boolean
  disabled: boolean
  disabledReason?: string
}

export type FocusedResourceAction = {
  ownerNodeID: string
  resourceName: string
  capability: string
}

const readOnlyCapabilities = new Set(['get', 'list', 'read'])
const unsafeCapabilities = new Set(['archive', 'cancel', 'delete', 'remove', 'revoke'])

export function deriveResourceActions(
  capabilities: readonly CapabilityDescriptor[],
  authority: AuthoritativeCapabilityState = {},
): ResourceAction[] {
  return [...capabilities]
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((descriptor) => {
      const mode = actionMode(descriptor)
      const mutating = mode === 'session' || (mode === 'operation' && !readOnlyCapabilities.has(descriptor.name))
      const knownAuthority = authority[descriptor.name]
      const disabled = knownAuthority?.status === 'forbidden'
      return {
        capability: descriptor.name,
        descriptor,
        mode,
        label: actionLabel(descriptor.name, mode),
        mutating,
        unsafe: unsafeCapabilities.has(descriptor.name),
        disabled,
        disabledReason: disabled ? knownAuthority.reason || '权威权限状态为 Forbidden' : undefined,
      }
    })
}

export function focusResourceAction(resource: ResourceDescriptor, capability: string): FocusedResourceAction {
  return {
    ownerNodeID: resource.id.owner_node_id,
    resourceName: resource.id.name,
    capability,
  }
}

export function resolveFocusedResourceAction(
  resource: ResourceDescriptor,
  focused: FocusedResourceAction,
  authority?: AuthoritativeCapabilityState,
): ResourceAction | undefined {
  if (resource.id.owner_node_id !== focused.ownerNodeID || resource.id.name !== focused.resourceName) return undefined
  return deriveResourceActions(resource.capabilities, authority).find((action) => action.capability === focused.capability)
}

function actionMode(capability: CapabilityDescriptor): ResourceActionMode {
  if (capability.name === 'open') return 'session'
  if (capability.event_schema && !capability.input_schema && !capability.output_schema) return 'observe'
  return 'operation'
}

function actionLabel(capability: string, mode: ResourceActionMode): string {
  if (mode === 'observe') return `观察 ${capability}`
  if (mode === 'session') return `打开会话 ${capability}`
  return `操作 ${capability}`
}
