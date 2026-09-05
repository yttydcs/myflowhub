import { describe, expect, it } from 'vitest'
import type { ResourceDescriptor } from '../types'
import { selectResourceRenderer } from './registry'

function collection(name: string, permission: string, label: string): ResourceDescriptor {
  return {
    id: { owner_node_id: '7', name },
    type: 'mfh.collection',
    type_version: 1,
    capabilities: [
      { name: 'list', permission, input_schema: 'mfh.collection.list-request.v1', output_schema: 'mfh.collection.page.v1', max_payload_bytes: 4096 },
      { name: 'get', permission, input_schema: 'mfh.collection.member-request.v1', output_schema: 'mfh.collection.member.v1', max_payload_bytes: 4096 },
    ],
    limits: { max_payload_bytes: 4096 },
    presentation: { label, renderer: 'provider-requested-special-case' },
  }
}

describe('Collection renderer matching', () => {
  it('depends on type and capability schemas, not Resource name, permission or label', () => {
    const first = selectResourceRenderer(collection('system/policy/definitions', 'admin.everything', 'Policy Definitions'))
    const second = selectResourceRenderer(collection('totally/arbitrary', 'guest.read', 'Unrelated label'))
    expect(first.selected.id).toBe('mfh.collection.browser.v1')
    expect(second.selected.id).toBe(first.selected.id)
    expect(first.fallbackReason).toContain('provider-requested-special-case')
  })

  it('does not activate the browser when list schemas are incompatible', () => {
    const descriptor = collection('anything', 'ignored', 'Anything')
    descriptor.capabilities[0] = { ...descriptor.capabilities[0]!, output_schema: 'vendor.page.v9' }
    expect(selectResourceRenderer(descriptor).selected.id).toBe('mfh.collection.unsupported.v1')
  })

  it('keeps a saved type-alias Widget readable without treating it as a renderer hint', () => {
    const descriptor = collection('anything', 'ignored', 'Anything')
    descriptor.presentation = { label: 'Anything' }
    const selected = selectResourceRenderer(descriptor, 'mfh.collection')
    expect(selected.selected.id).toBe('mfh.collection.browser.v1')
    expect(selected.fallbackReason).toBeUndefined()
  })

  it('keeps the filesystem envelope schema at the read boundary instead of selecting it as a content renderer', () => {
    const descriptor = collection('storage/files', 'files.read', 'Files')
    descriptor.capabilities.push({
      name: 'read',
      permission: 'files.read',
      input_schema: 'mfh.filesystem.read-request.v1',
      output_schema: 'mfh.filesystem.content.v1',
      max_payload_bytes: 180_000,
    })
    descriptor.presentation = { label: 'Files', renderer: 'mfh.filesystem.content.v1' }

    const selected = selectResourceRenderer(descriptor)
    expect(selected.selected.id).toBe('mfh.collection.browser.v1')
    expect(selected.fallbackReason).toContain('mfh.filesystem.content.v1')
  })
})
